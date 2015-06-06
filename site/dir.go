package site

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/martinp/lftpq/parser"
)

type Dir struct {
	Created   time.Time
	Path      string
	IsSymlink bool
}

func ParseDir(s string) (Dir, error) {
	words := strings.SplitN(s, " ", 5)
	if len(words) != 5 {
		return Dir{}, fmt.Errorf("failed to parse dir: %s", s)
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

func (d *Dir) Show() (parser.Show, error) {
	show, err := parser.ParseShow(d.Base())
	if err != nil {
		return parser.Show{}, err
	}
	return show, nil
}

func (d *Dir) Movie() (parser.Movie, error) {
	movie, err := parser.ParseMovie(d.Base())
	if err != nil {
		return parser.Movie{}, err
	}
	return movie, nil
}
