package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	LftpPath string
	GetCmd   string
	Sites    []Site
}

type Site struct {
	Config
	Name    string
	Dir     string
	MaxAge_ string `json:"MaxAge"`
	Shows   []string
}

type Dir struct {
	Created   time.Time
	Name      string
	IsSymlink bool
}

func ParseDir(s string) (Dir, error) {
	words := strings.SplitN(s, " ", 5)
	if len(words) != 5 {
		return Dir{}, fmt.Errorf("expected 5 words, found %d", len(words))
	}
	t := strings.Join(words[:4], " ")
	created, err := time.Parse("2006-01-02 15:04:05 -0700 MST", t)
	if err != nil {
		return Dir{}, err
	}
	name := words[4]
	isSymlink := strings.HasSuffix(name, "@")
	name = strings.TrimRight(name, "@/")
	return Dir{
		Name:      name,
		Created:   created,
		IsSymlink: isSymlink,
	}, nil
}

func (s *Site) MaxAge() (time.Duration, error) {
	return time.ParseDuration(s.MaxAge_)
}

func (s *Site) ListCommand() *exec.Cmd {
	options := fmt.Sprintf(`-e "cd %s &&
cls --date --time-style='%%F %%T %%z %%Z' &&
exit"`, s.Dir)
	args := strings.Split(options, " ")
	return exec.Command(s.LftpPath, args...)
}

func (d *Dir) CreatedAfter(age time.Duration) bool {
	return d.Created.After(time.Now().Add(-age))
}

func (d *Dir) MatchAny(ss []string) bool {
	for _, s := range ss {
		if d.Match(s) {
			return true
		}
	}
	return false
}

func (d *Dir) Match(s string) bool {
	return strings.HasPrefix(d.Name, s)
}

func (s *Site) GetDirs() ([]Dir, error) {
	cmd := s.ListCommand()
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
		age, err := s.MaxAge()
		if err != nil {
			return nil, err
		}
		if !dir.CreatedAfter(age) {
			continue
		}
		if !dir.MatchAny(s.Shows) {
			continue
		}
		res = append(res, dir)
	}
	return res, nil
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
	return cfg, nil
}

func main() {
	var opts struct {
		Config   string `short:"f" long:"config" description:"Path to config" value-name:"FILE" required:"true"`
		LftpPath string `short:"p" long:"lftp-path" description:"Override path to lftp executable" value-name:"PATH" default:"lftp"`
	}
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		os.Exit(1)
	}

	cfg, err := ReadConfig(opts.Config)
	if err != nil {
		log.Fatal(err)
	}

	cfg.LftpPath = opts.LftpPath
	for _, site := range cfg.Sites {
		site.Config = cfg
		dirs, err := site.FilterDirs()
		if err != nil {
			log.Fatal(err)
		}
		for _, d := range dirs {
			log.Printf("%+v", d)
		}
	}
}
