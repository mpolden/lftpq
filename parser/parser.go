package parser

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	movieExp   = regexp.MustCompile(`(.*?)\.(\d{4})`)
	episodeExp = regexp.MustCompile(`(.*)\.(?:(` +
		`(?:S(\d{2}))(?:E(\d{2}))?` + // S01, S01E04
		`|(?:E(\d{2}))` + // E04
		`|(\d{1,2})x(\d{2})` + // 1x04, 01x04
		`|Part\.?(\d{1,2})` + // Part4, Part11, Part.4, Part.11
		`))`)
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

func (m *Media) Equal(o Media) bool {
	if m.IsEmpty() {
		return false
	}
	return m.Name == o.Name && m.Season == o.Season && m.Episode == o.Episode && m.Year == o.Year
}

func Default(s string) (Media, error) {
	return Media{Release: s}, nil
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
	if len(m) == 0 || len(m[0]) < 9 {
		return Media{}, fmt.Errorf("failed to parse: %s", s)
	}
	name := m[0][1]
	season := "1"
	episode := "0"
	if m[0][3] != "" { // S01, S01E04
		season = m[0][3]
		if m[0][4] != "" {
			episode = m[0][4]
		}
	} else if m[0][5] != "" { // E04
		episode = m[0][5]
	} else if m[0][6] != "" && m[0][7] != "" { // 1x04, 01x04
		season = m[0][6]
		episode = m[0][7]
	} else if m[0][8] != "" { // Part4, Part11, Part.4, Part.11
		episode = m[0][8]
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
