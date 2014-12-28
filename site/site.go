package site

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	LftpGetCmd string
	LftpPath   string
	LocalPath  string
}

type Config struct {
	Client Client
	Sites  []Site
}

func ReadConfig(name string) (Config, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	for i, site := range cfg.Sites {
		maxAge, err := time.ParseDuration(site.MaxAge_)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].MaxAge = maxAge
		cfg.Sites[i].Client = cfg.Client
	}
	return cfg, nil
}

type Site struct {
	Client
	Name    string
	Dir     string
	MaxAge  time.Duration
	MaxAge_ string `json:"MaxAge"`
	Shows   []string
}

func (s *Site) lftpCmd(cmd string) *exec.Cmd {
	args := []string{"-e", cmd}
	return exec.Command(s.LftpPath, args...)
}

func (s *Site) ListCmd() *exec.Cmd {
	cmd := fmt.Sprintf(`cd %s &&
cls --date --time-style='%%F %%T %%z %%Z' &&
exit`, s.Dir)
	return s.lftpCmd(cmd)
}

func (s *Site) GetDirs() ([]Dir, error) {
	cmd := s.ListCmd()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	dirs := []Dir{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t\r\n")
		if len(line) == 0 {
			continue
		}
		dir, err := ParseDir(line)
		if err != nil {
			return nil, err
		}

		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func (s *Site) FilterDirs(dirs []Dir) []Dir {
	res := []Dir{}
	for _, dir := range dirs {
		if dir.IsSymlink {
			continue
		}
		if !dir.CreatedAfter(s.MaxAge) {
			continue
		}
		if !dir.MatchAny(s.Shows) {
			continue
		}
		res = append(res, dir)
	}
	return res
}

func (s *Site) LocalPath(dir Dir) (string, error) {
	series, err := ParseSeries(dir.Name)
	if err != nil {
		return "", err
	}
	localPath := filepath.Join(s.Client.LocalPath, series.Name,
		"S"+series.Season)
	if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
		localPath += string(os.PathSeparator)
	}
	return localPath, nil
}

func (s *Site) GetCmd(dir Dir) (*exec.Cmd, error) {
	localPath, err := s.LocalPath(dir)
	if err != nil {
		return nil, err
	}
	options := fmt.Sprintf("cd %s && %s %s %s && exit", s.Dir, s.LftpGetCmd,
		dir.Name, localPath)
	return s.lftpCmd(options), nil
}
