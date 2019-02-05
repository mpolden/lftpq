package parser

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	movieExp    = regexp.MustCompile(`(.*?)\.(\d{4})`)
	episodeExps = [4]*regexp.Regexp{
		regexp.MustCompile(`^(?P<name>.+)\.S(?P<season>\d{2})(?:E(?P<episode>\d{2}))?`), // S01, S01E04
		regexp.MustCompile(`^(?P<name>.+)\.E(?P<episode>\d{2})`),                        // E04
		regexp.MustCompile(`^(?P<name>.+)\.(?P<season>\d{1,2})x(?P<episode>\d{2})`),     // 1x04, 01x04
		regexp.MustCompile(`^(?P<name>.+)\.Part\.?(?P<episode>\d{1,2})`),                // Part4, Part11, Part.4, Part.11
	}
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
	for _, p := range episodeExps {
		groupNames := p.SubexpNames()
		matches := p.FindAllStringSubmatch(s, -1)
		if len(matches) == 0 {
			continue
		}
		match := matches[0]
		var (
			name    string
			season  = 0
			episode = 0
			err     error
		)
		for i, group := range match {
			if group == "" {
				continue
			}
			switch groupNames[i] {
			case "name":
				name = group
			case "season":
				season, err = strconv.Atoi(group)
				if err != nil {
					return Media{}, fmt.Errorf("invalid input: %q: %s", s, err)
				}
			case "episode":
				episode, err = strconv.Atoi(group)
				if err != nil {
					return Media{}, fmt.Errorf("invalid input: %q: %s", s, err)
				}
			}
		}
		if season == 0 {
			season = 1
		}
		if season > 0 || episode > 0 {
			return Media{
				Release: s,
				Name:    name,
				Season:  season,
				Episode: episode,
			}, nil
		}
	}
	return Media{}, fmt.Errorf("invalid input: %q", s)
}
