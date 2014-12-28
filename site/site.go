package site

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
	"github.com/martinp/lftptv/cmd"
)

type Client struct {
	LftpGetCmd string
	LftpPath   string
	LocalPath  *template.Template
	LocalPath_ string `json:"LocalPath"`
}

type Site struct {
	Client
	Name      string
	Dir       string
	MaxAge    time.Duration
	MaxAge_   string   `json:"MaxAge"`
	Patterns_ []string `json:"Patterns"`
	Patterns  []*regexp.Regexp
}

func (s *Site) ListCmd() cmd.Lftp {
	args := fmt.Sprintf("cls --date --time-style='%%F %%T %%z %%Z' %s",
		s.Dir)
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
	}
}

func (s *Site) GetDirs() ([]Dir, error) {
	listCmd := s.ListCmd()
	cmd := listCmd.Cmd()
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

func (s *Site) FilterDirs() ([]Dir, error) {
	dirs, err := s.GetDirs()
	if err != nil {
		return nil, err
	}
	res := []Dir{}
	for _, dir := range dirs {
		if dir.IsSymlink {
			continue
		}
		if !dir.CreatedAfter(s.MaxAge) {
			continue
		}
		if !dir.MatchAny(s.Patterns) {
			continue
		}
		res = append(res, dir)
	}
	return res, nil
}

func (s *Site) LocalPath(dir Dir) (string, error) {
	series, err := ParseSeries(dir.Base())
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := s.Client.LocalPath.Execute(&b, series); err != nil {
		return "", err
	}
	localPath := b.String()
	if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
		localPath += string(os.PathSeparator)
	}
	return localPath, nil
}

func (s *Site) GetCmd(dir Dir) (cmd.Lftp, error) {
	localPath, err := s.LocalPath(dir)
	if err != nil {
		return cmd.Lftp{}, err
	}
	args := fmt.Sprintf("%s %s %s", s.LftpGetCmd, dir.Path, localPath)
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
	}, nil
}

func (s *Site) QueueCmd(dir Dir) (cmd.Lftp, error) {
	getCmd, err := s.GetCmd(dir)
	if err != nil {
		return cmd.Lftp{}, err
	}
	args := "queue " + getCmd.Args
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
	}, nil
}
