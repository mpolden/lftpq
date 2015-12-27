package lftp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Dir struct {
	Created   time.Time
	Path      string
	IsSymlink bool
	IsFile    bool
}

func ParseDir(s string) (Dir, error) {
	parts := strings.SplitN(s, " ", 5)
	if len(parts) != 5 {
		return Dir{}, fmt.Errorf("failed to parse dir: %s", s)
	}
	t := strings.Join(parts[:4], " ")
	created, err := time.Parse("2006-01-02 15:04:05 -0700 MST", t)
	if err != nil {
		return Dir{}, err
	}
	path := parts[4]
	isSymlink := strings.HasSuffix(path, "@")
	isDir := strings.HasSuffix(path, "/")
	isFile := !isSymlink && !isDir
	path = strings.TrimRight(path, "@/")
	return Dir{
		Path:      path,
		Created:   created,
		IsSymlink: isSymlink,
		IsFile:    isFile,
	}, nil
}

func (d *Dir) Base() string {
	return filepath.Base(d.Path)
}

func (d *Dir) Age(since time.Time) time.Duration {
	return since.Round(time.Second).Sub(d.Created)
}

func (d *Dir) MatchAny(patterns []*regexp.Regexp) (string, bool) {
	for _, p := range patterns {
		if d.Match(p) {
			return p.String(), true
		}
	}
	return "", false
}

func (d *Dir) Match(pattern *regexp.Regexp) bool {
	return pattern.MatchString(d.Base())
}
