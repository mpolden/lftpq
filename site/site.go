package site

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/martinp/lftpq/parser"
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
	SkipExisting bool
	Parser       string
	parser       parser.Parser
	LocalDir     string
	localDir     *template.Template
	Priorities   []string
	priorities   []*regexp.Regexp
	Deduplicate  bool
}

func (s *Site) listCmd() Lftp {
	script := "cls -1 --classify --date --time-style='%F %T %z %Z' " + s.Dir + " && exit"
	args := []string{"-e", script, s.Name}
	return Lftp{Path: s.LftpPath, Args: args}
}

func (s *Site) parseDirList(r io.Reader) ([]Dir, error) {
	dirs := []Dir{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
	return dirs, nil
}

func (s *Site) DirList() ([]Dir, error) {
	listCmd := s.listCmd()
	cmd := listCmd.Cmd()
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	dirs, err := s.parseDirList(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return dirs, nil
}
