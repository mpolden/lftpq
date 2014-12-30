package site

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/martinp/lftpfetch/cmd"
	"io"
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
}

type Site struct {
	Client
	Name         string
	Dir          string
	MaxAge       string
	maxAge       time.Duration
	Patterns     []string
	patterns     []*regexp.Regexp
	Filters      []string
	filters      []*regexp.Regexp
	SkipSymlinks bool
	ParseTVShow  bool
	LocalDir     string
	localDir     *template.Template
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

func (s *Site) GetDirs() ([]Dir, error) {
	listCmd := s.ListCmd()
	cmd := listCmd.Cmd()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		if _, err := io.Copy(os.Stderr, stderr); err != nil {
			panic(err)
		}
	}()
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
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return dirs, nil
}

func (s *Site) FilterDirs(dirs []Dir) []Dir {
	res := []Dir{}
	for _, dir := range dirs {
		if dir.IsSymlink && s.SkipSymlinks {
			continue
		}
		if !dir.CreatedAfter(s.maxAge) {
			continue
		}
		if !dir.MatchAny(s.patterns) {
			continue
		}
		if dir.MatchAny(s.filters) {
			continue
		}
		res = append(res, dir)
	}
	return res
}

func (s *Site) ParseLocalDir(dir Dir) (string, error) {
	localDir := s.LocalDir
	if s.ParseTVShow {
		show, err := dir.Show()
		if err != nil {
			return "", err
		}
		var b bytes.Buffer
		if err := s.localDir.Execute(&b, show); err != nil {
			return "", err
		}
		localDir = b.String()
	}
	if !strings.HasSuffix(localDir, string(os.PathSeparator)) {
		localDir += string(os.PathSeparator)
	}
	return localDir, nil
}

func (s *Site) GetCmd(dir Dir) (cmd.Lftp, error) {
	localDir, err := s.ParseLocalDir(dir)
	if err != nil {
		return cmd.Lftp{}, err
	}
	dstPath := filepath.Join(localDir, dir.Base())
	if _, err := os.Stat(dstPath); err == nil {
		return cmd.Lftp{}, fmt.Errorf("%s already exists", dstPath)
	}
	args := fmt.Sprintf("%s %s %s", s.LftpGetCmd, dir.Path, localDir)
	return cmd.Lftp{
		Path: s.LftpPath,
		Args: args,
		Site: s.Name,
	}, nil
}

func (s *Site) QueueCmd(dir Dir) (cmd.Lftp, error) {
	getCmd, err := s.GetCmd(dir)
	if err != nil {
		return cmd.Lftp{}, err
	}
	getCmd.Args = "queue " + getCmd.Args
	return getCmd, nil
}
