package tv

import (
	"fmt"
	"regexp"
	"strings"
)

var releaseExp = regexp.MustCompile("(.*)\\.(?:(S(\\d{2})E(\\d{2})|(\\d{1,2})x(\\d{2})))")

type Show struct {
	Release string
	Name    string
	Season  string
	Episode string
}

func Parse(s string) (Show, error) {
	m := releaseExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 {
		return Show{}, fmt.Errorf("no matches found")
	}
	if len(m[0]) < 7 {
		return Show{}, fmt.Errorf("only %d submatches found",
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
		return Show{}, fmt.Errorf("failed to parse season and episode")
	}
	return Show{
		Release: s,
		Name:    name,
		Season:  season,
		Episode: episode,
	}, nil
}
