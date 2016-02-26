package lftp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type File struct {
	Created time.Time
	Path    string
	os.FileMode
}

func ParseFile(s string) (File, error) {
	parts := strings.SplitN(s, " ", 5)
	if len(parts) != 5 {
		return File{}, fmt.Errorf("failed to parse file: %s", s)
	}
	t := strings.Join(parts[:4], " ")
	created, err := time.Parse("2006-01-02 15:04:05 -0700 MST", t)
	if err != nil {
		return File{}, err
	}
	path := parts[4]

	var fileMode os.FileMode
	if strings.HasSuffix(path, "@") {
		fileMode = os.ModeSymlink
	} else if strings.HasSuffix(path, "/") {
		fileMode = os.ModeDir
	} else {
		fileMode = os.FileMode(0) // Regular file
	}

	path = strings.TrimRight(path, "@/")
	return File{
		Path:     path,
		Created:  created,
		FileMode: fileMode,
	}, nil
}

func (f *File) IsSymlink() bool {
	return f.FileMode&os.ModeSymlink != 0
}

func (f *File) Base() string {
	return filepath.Base(f.Path)
}

func (f *File) Age(since time.Time) time.Duration {
	return since.Round(time.Second).Sub(f.Created)
}

func (f *File) MatchAny(patterns []*regexp.Regexp) (string, bool) {
	for _, p := range patterns {
		if f.Match(p) {
			return p.String(), true
		}
	}
	return "", false
}

func (f *File) Match(pattern *regexp.Regexp) bool {
	return pattern.MatchString(f.Base())
}
