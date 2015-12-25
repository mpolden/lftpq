package parser

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	movieExp   = regexp.MustCompile("(.*?)\\.(\\d{4})")
	episodeExp = regexp.MustCompile("(.*)\\.(?:(" +
		"(?:S(\\d{2}))?E(\\d{2})" + // S01E04, E04
		"|(\\d{1,2})x(\\d{2})" + // 1x04, 01x04
		"|Part\\.?(\\d{1,2})" + // Part4, Part11, Part.4, Part.11
		"))")
)

type Parser func(s string) (Media, error)

type Media struct {
	Release string
	Name    string
	Year    int
	Season  int
	Episode int
}

func (m *Media) IsEmpty() bool {
	return m.Name == ""
}

func (m *Media) ReplaceName(re *regexp.Regexp, repl string) {
	m.Name = re.ReplaceAllString(m.Name, repl)
}

func (a *Media) Equal(b Media) bool {
	if a.IsEmpty() {
		return false
	}
	return a.Name == b.Name && a.Season == b.Season && a.Episode == b.Episode && a.Year == b.Year
}

func Default(s string) (Media, error) {
	return Media{}, nil
}

func Movie(s string) (Media, error) {
	m := movieExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 || len(m[0]) < 3 {
		return Media{}, fmt.Errorf("failed to parse: %s", s)
	}
	name := m[0][1]
	year, err := strconv.Atoi(m[0][2])
	if err != nil {
		return Media{}, err
	}
	return Media{
		Release: s,
		Name:    name,
		Year:    year,
	}, nil
}

func Show(s string) (Media, error) {
	m := episodeExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 || len(m[0]) < 8 {
		return Media{}, fmt.Errorf("failed to parse: %s", s)
	}
	name := m[0][1]
	var season string
	var episode string
	if m[0][4] != "" {
		if m[0][3] != "" {
			season = m[0][3]
		} else {
			season = "1"
		}
		episode = m[0][4]
	} else if m[0][5] != "" && m[0][6] != "" {
		season = m[0][5]
		episode = m[0][6]
	} else if m[0][7] != "" {
		season = "1"
		episode = m[0][7]
	}
	ss, err := strconv.Atoi(season)
	if err != nil {
		return Media{}, err
	}
	ep, err := strconv.Atoi(episode)
	if err != nil {
		return Media{}, err
	}
	return Media{
		Release: s,
		Name:    name,
		Season:  ss,
		Episode: ep,
	}, nil
}
