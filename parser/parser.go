package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var (
	moviePattern    = regexp.MustCompile(`(.*?)\.(\d{4})`)
	episodePatterns = [4]*regexp.Regexp{
		regexp.MustCompile(`^(?P<name>.+?)\.[Ss](?P<season>\d{2})(?:[Ee](?P<episode>\d{2}))?`), // S01, S01E04
		regexp.MustCompile(`^(?P<name>.+?)\.[Ee](?P<episode>\d{2})`),                           // E04
		regexp.MustCompile(`^(?P<name>.+?)\.(?P<season>\d{1,2})x(?P<episode>\d{2})`),           // 1x04, 01x04
		regexp.MustCompile(`^(?P<name>.+?)\.P(?:ar)?t\.?(?P<episode>([^.]+))`),                 // P(ar)t(.)11, Pt(.)XI
	}
	splitPattern = regexp.MustCompile(`[-_.]`)
)

type Parser func(s string) (Media, error)

type Media struct {
	Release    string
	Name       string
	Year       int
	Season     int
	Episode    int
	Resolution string
	Codec      string
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
	return m.Name == o.Name &&
		m.Season == o.Season &&
		m.Episode == o.Episode &&
		m.Year == o.Year &&
		m.Resolution == o.Resolution &&
		m.Codec == o.Codec
}

func (m *Media) PathIn(dir *template.Template) (string, error) {
	var b bytes.Buffer
	if err := dir.Execute(&b, m); err != nil {
		return "", err
	}
	path := b.String()
	// When path has a trailing slash, the actual destination path will be a directory inside LocalPath (same
	// behaviour as rsync)
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		path = filepath.Join(path, m.Release)
	}
	return path, nil
}

func Default(s string) (Media, error) {
	return Media{Release: s}, nil
}

func Movie(s string) (Media, error) {
	matches := moviePattern.FindStringSubmatch(s)
	if len(matches) < 3 {
		return Media{}, fmt.Errorf("invalid input: %q", s)
	}
	name := matches[1]
	year, err := strconv.Atoi(matches[2])
	if err != nil {
		return Media{}, fmt.Errorf("invalid input: %q: %s", s, err)
	}
	return Media{
		Release:    s,
		Name:       name,
		Year:       year,
		Resolution: resolution(s),
		Codec:      codec(s),
	}, nil
}

func Show(s string) (Media, error) {
	for _, p := range episodePatterns {
		matches := p.FindStringSubmatch(s)
		if len(matches) == 0 {
			continue
		}
		groupNames := p.SubexpNames()
		var (
			name    string
			season  = 1
			episode = 0
			err     error
		)
		for i, group := range matches {
			if group == "" {
				continue
			}
			switch groupNames[i] {
			case "name":
				name = group
			case "season":
				season, err = strconv.Atoi(group)
				if err != nil {
					return Media{}, fmt.Errorf("invalid input: %q: %w", s, err)
				}
			case "episode":
				episode, err = strconv.Atoi(group)
				if err != nil {
					episode, err = rtoi(group)
					if err != nil {
						return Media{}, fmt.Errorf("invalid input: %q: %w", s, err)
					}
				}
			}
		}
		if strings.ToLower(name) == name {
			// Capitalize
			name = strings.ToUpper(string(name[0])) + strings.ToLower(string(name[1:]))
		}
		return Media{
			Release:    s,
			Name:       name,
			Season:     season,
			Episode:    episode,
			Resolution: resolution(s),
			Codec:      codec(s),
		}, nil
	}
	return Media{}, fmt.Errorf("invalid input: %q", s)
}

func findPart(s string, partFunc func(part string) bool) string {
	s = strings.ToLower(s)
	parts := splitPattern.Split(s, -1)
	for _, part := range parts {
		if partFunc(part) {
			return part
		}
	}
	return ""
}

func resolution(s string) string {
	return findPart(s, func(part string) bool {
		switch part {
		case "720p", "1080p", "2160p":
			return true
		}
		return false
	})
}

func codec(s string) string {
	return findPart(s, func(part string) bool {
		switch part {
		case "h264", "h265", "xvid", "x264", "x265":
			return true
		}
		return false
	})
}

var numerals = []numeral{
	// units
	{"IX", 9},
	{"VIII", 8},
	{"VII", 7},
	{"VI", 6},
	{"IV", 4},
	{"III", 3},
	{"II", 2},
	{"V", 5},
	{"I", 1},
	// tens
	{"XC", 90},
	{"LXXX", 80},
	{"LXX", 70},
	{"LX", 60},
	{"XL", 40},
	{"XXX", 30},
	{"XX", 20},
	{"L", 50},
	{"X", 10},
	// hundreds
	{"CM", 900},
	{"DCCC", 800},
	{"DCC", 700},
	{"DC", 600},
	{"CD", 400},
	{"CCC", 300},
	{"CC", 200},
	{"D", 500},
	{"C", 100},
	// thousands
	{"MMM", 3000},
	{"MM", 2000},
	{"M", 1000},
}

type numeral struct {
	s string
	n int
}

func (n numeral) parse(s string, lastIndex int) (string, int, int) {
	i := strings.LastIndex(s, n.s)
	if i > -1 && i < lastIndex {
		rest := s[:i] + s[i+len(n.s):]
		return rest, i, n.n
	}
	return s, lastIndex, 0
}

func rtoi(s string) (int, error) {
	// needlessly complete parsing of roman numerals
	sum := 0
	lastIndex := len(s)
	for _, num := range numerals {
		var n int
		s, lastIndex, n = num.parse(s, lastIndex)
		sum += n
		if s == "" {
			break
		}
	}
	if s != "" {
		return 0, fmt.Errorf("invalid roman numeral: %q", s)
	}
	return sum, nil
}
