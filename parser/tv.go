package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var episodeExp = regexp.MustCompile("(.*)\\.(?:(" +
	"S(\\d{2})E(\\d{2})" + // S01E04
	"|(\\d{1,2})x(\\d{2})" + // 1x04, 01x04
	"|Part\\.(\\d{1,2})" + // Part.4, Part.11
	"))")

type Show struct {
	Release string
	Name    string
	Season  string
	Episode string
}

func (a *Show) Equal(b Show) bool {
	return a.Name == b.Name && a.Season == b.Season && a.Episode == b.Episode
}

func ParseShow(s string) (Show, error) {
	m := episodeExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 || len(m[0]) < 7 {
		return Show{}, fmt.Errorf("failed to parse: %s", s)
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
		return Show{}, fmt.Errorf("failed to parse season and episode for %s", s)
	}
	season = fmt.Sprintf("%02s", season)
	episode = fmt.Sprintf("%02s", episode)
	return Show{
		Release: s,
		Name:    name,
		Season:  season,
		Episode: episode,
	}, nil
}
