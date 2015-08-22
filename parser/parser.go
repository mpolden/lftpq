package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	movieExp   = regexp.MustCompile("(.*?)\\.(\\d{4})")
	episodeExp = regexp.MustCompile("(.*)\\.(?:(" +
		"S(\\d{2})E(\\d{2})" + // S01E04
		"|(\\d{1,2})x(\\d{2})" + // 1x04, 01x04
		"|Part\\.(\\d{1,2})" + // Part.4, Part.11
		"))")
)

type Parser func(s string) (Media, error)

type Media struct {
	Release string
	Name    string
	Year    int
	Season  string
	Episode string
}

func (m *Media) IsEmpty() bool {
	return m.Name == ""
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
		return Media{}, fmt.Errorf("failed to parse year for %s", s)
	}
	return Media{
		Release: s,
		Name:    name,
		Year:    year,
	}, nil
}

func Show(s string) (Media, error) {
	m := episodeExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 || len(m[0]) < 7 {
		return Media{}, fmt.Errorf("failed to parse: %s", s)
	}
	name := strings.Replace(m[0][1], "_", ".", -1)
	var season string
	var episode string
	if m[0][3] != "" && m[0][4] != "" {
		season = m[0][3]
		episode = m[0][4]
	} else if m[0][5] != "" && m[0][6] != "" {
		season = m[0][5]
		episode = m[0][6]
	} else if m[0][7] != "" {
		season = "1"
		episode = m[0][7]
	} else {
		return Media{}, fmt.Errorf("failed to parse season and episode for %s", s)
	}
	season = fmt.Sprintf("%02s", season)
	episode = fmt.Sprintf("%02s", episode)
	return Media{
		Release: s,
		Name:    name,
		Season:  season,
		Episode: episode,
	}, nil
}