package site

import (
	"fmt"
	"regexp"
	"strings"
)

var seriesExp = regexp.MustCompile("(.*)\\.(?:(S(\\d{2})E(\\d{2})|(\\d{1,2})x(\\d{2})))")

type Series struct {
	ReleaseName string
	Name        string
	Season      string
	Episode     string
}

func ParseSeries(s string) (Series, error) {
	m := seriesExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 {
		return Series{}, fmt.Errorf("no matches found")
	}
	if len(m[0]) < 7 {
		return Series{}, fmt.Errorf("only %d submatches found",
			len(m[0]))
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
	} else {
		return Series{}, fmt.Errorf("failed to parse season and episode")
	}
	return Series{
		ReleaseName: s,
		Name:        name,
		Season:      season,
		Episode:     episode,
	}, nil
}
