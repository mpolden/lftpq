package site

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/martinp/lftpfetch/cmd"
	"github.com/martinp/lftpfetch/ftpdir"
	"github.com/martinp/lftpfetch/tv"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type Client struct {
	LftpGetCmd string
	LftpPath   string
	LocalPath  *template.Template
	LocalPath_ string `json:"LocalPath"`
}

type Site struct {
	Client
	Name         string
	Dir          string
	MaxAge       time.Duration
	MaxAge_      string   `json:"MaxAge"`
	Patterns_    []string `json:"Patterns"`
	Patterns     []*regexp.Regexp
	Filters_     []string `json:"Filters"`
	Filters      []*regexp.Regexp
	SkipSymlinks bool
}

func (s *Site) ListCmd() cmd.Lftp {
	args := fmt.Sprintf("cls --date --time-style='%%F %%T %%z %%Z' %s",
		s.Dir)
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
		Site: s.Name,
	}
}

func (s *Site) GetDirs() ([]ftpdir.Dir, error) {
	listCmd := s.ListCmd()
	cmd := listCmd.Cmd()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	dirs := []ftpdir.Dir{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t\r\n")
		if len(line) == 0 {
			continue
		}
		dir, err := ftpdir.Parse(line)
		if err != nil {
			return nil, err
		}

		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func (s *Site) FilterDirs(dirs []ftpdir.Dir) []ftpdir.Dir {
	res := []ftpdir.Dir{}
	for _, dir := range dirs {
		if dir.IsSymlink && s.SkipSymlinks {
			continue
		}
		if !dir.CreatedAfter(s.MaxAge) {
			continue
		}
		if !dir.MatchAny(s.Patterns) {
			continue
		}
		if dir.MatchAny(s.Filters) {
			continue
		}
		res = append(res, dir)
	}
	return res
}

func (s *Site) LocalPath(dir ftpdir.Dir) (string, error) {
	show, err := tv.Parse(dir.Base())
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := s.Client.LocalPath.Execute(&b, show); err != nil {
		return "", err
	}
	localPath := b.String()
	if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
		localPath += string(os.PathSeparator)
	}
	return localPath, nil
}

func (s *Site) GetCmd(dir ftpdir.Dir) (cmd.Lftp, error) {
	localPath, err := s.LocalPath(dir)
	if err != nil {
		return cmd.Lftp{}, err
	}
	dstPath := filepath.Join(localPath, dir.Base())
	if _, err := os.Stat(dstPath); err == nil {
		return cmd.Lftp{}, fmt.Errorf("%s already exists", dstPath)
	}
	args := fmt.Sprintf("%s %s %s", s.LftpGetCmd, dir.Path, localPath)
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
		Site: s.Name,
	}, nil
}

func (s *Site) QueueCmd(dir ftpdir.Dir) (cmd.Lftp, error) {
	getCmd, err := s.GetCmd(dir)
	if err != nil {
		return cmd.Lftp{}, err
	}
	args := "queue " + getCmd.Args
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
		Site: s.Name,
	}, nil
}
