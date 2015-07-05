package parser

import (
	"fmt"
	"regexp"
	"strconv"
)

var movieExp = regexp.MustCompile("(.*?)\\.(\\d{4})")

type Movie struct {
	Release string
	Name    string
	Year    int
}

func (a *Movie) Equal(b Movie) bool {
	return a.Name == b.Name && a.Year == b.Year
}

func ParseMovie(s string) (Movie, error) {
	m := movieExp.FindAllStringSubmatch(s, -1)
	if len(m) == 0 {
		return Movie{}, fmt.Errorf("no matches found for %s", s)
	}
	if len(m[0]) < 3 {
		return Movie{}, fmt.Errorf("only %d submatches found for %s",
			len(m[0]), s)
	}
	name := m[0][1]
	year, err := strconv.Atoi(m[0][2])
	if err != nil {
		return Movie{}, fmt.Errorf("failed to parse year for %s", s)
	}
	return Movie{
		Release: s,
		Name:    name,
		Year:    year,
	}, nil
}
