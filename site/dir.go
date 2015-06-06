package site

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/martinp/lftpq/tv"
)

type Dir struct {
	Created   time.Time
	Path      string
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
	path := words[4]
	isSymlink := strings.HasSuffix(path, "@")
	path = strings.TrimRight(path, "@/")
	return Dir{
		Path:      path,
		Created:   created,
		IsSymlink: isSymlink,
	}, nil
}

func (d *Dir) Base() string {
	return filepath.Base(d.Path)
}

func (d *Dir) Age() time.Duration {
	return time.Now().Round(time.Second).Sub(d.Created)
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

func (d *Dir) Show() (tv.Show, error) {
	show, err := tv.Parse(d.Base())
	if err != nil {
		return tv.Show{}, err
	}
	return show, nil
}
