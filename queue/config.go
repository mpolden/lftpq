package queue

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/mpolden/lftpq/lftp"
	"github.com/mpolden/lftpq/parser"
)

type Config struct {
	Default Site
	Sites   []Site
}

type Replacement struct {
	Pattern     string
	pattern     *regexp.Regexp
	Replacement string
}

type Site struct {
	Client       lftp.Client
	Name         string
	Dirs         []string
	MaxAge       string
	maxAge       time.Duration
	Patterns     []string
	patterns     []*regexp.Regexp
	Filters      []string
	filters      []*regexp.Regexp
	SkipSymlinks bool
	SkipExisting bool
	SkipFiles    bool
	Parser       string
	LocalDir     string
	Priorities   []string
	priorities   []*regexp.Regexp
	Deduplicate  bool
	PostCommand  string
	Replacements []Replacement
	Merge        bool
	Skip         bool
	itemParser
}

type itemParser struct {
	parser       parser.Parser
	template     *template.Template
	replacements []Replacement
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}
	return res, nil
}

func compileReplacements(replacements []Replacement) ([]Replacement, error) {
	res := make([]Replacement, 0, len(replacements))
	for _, r := range replacements {
		pattern, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, err
		}
		r.pattern = pattern
		res = append(res, r)
	}
	return res, nil
}

func parseTemplate(tmpl string) (*template.Template, error) {
	funcMap := template.FuncMap{"Sprintf": fmt.Sprintf}
	t, err := template.New("").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func isExecutable(s string) error {
	if s == "" {
		return nil
	}
	args := strings.Split(s, " ")
	_, err := exec.LookPath(args[0])
	return err
}

func (c *Config) load() error {
	for i := range c.Sites {
		site := &c.Sites[i]
		maxAge, err := time.ParseDuration(site.MaxAge)
		if err != nil {
			return err
		}
		site.maxAge = maxAge
		patterns, err := compilePatterns(site.Patterns)
		if err != nil {
			return err
		}
		site.patterns = patterns
		filters, err := compilePatterns(site.Filters)
		if err != nil {
			return err
		}
		site.filters = filters
		priorities, err := compilePatterns(site.Priorities)
		if err != nil {
			return err
		}
		site.priorities = priorities
		replacements, err := compileReplacements(site.Replacements)
		if err != nil {
			return err
		}
		site.Replacements = replacements

		tmpl, err := parseTemplate(site.LocalDir)
		if err != nil {
			return err
		}

		if err := isExecutable(site.Client.Path); err != nil {
			return err
		}
		if err := isExecutable(site.PostCommand); err != nil {
			return err
		}

		var parserFunc parser.Parser
		switch site.Parser {
		case "show":
			parserFunc = parser.Show
		case "movie":
			parserFunc = parser.Movie
		case "":
			parserFunc = parser.Default
		default:
			return fmt.Errorf("invalid parser: %q (must be %q, %q or %q)",
				site.Parser, "show", "movie", "")
		}
		site.itemParser = itemParser{
			parser:       parserFunc,
			replacements: site.Replacements,
			template:     tmpl,
		}
	}
	return nil
}

func (c *Config) JSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

func (c *Config) LookupSite(name string) (Site, error) {
	for _, site := range c.Sites {
		if site.Name == name {
			return site, nil
		}
	}
	return Site{}, fmt.Errorf("site not found in config: %s", name)
}

func readConfig(r io.Reader) (Config, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, err
	}
	// Unmarshal config and replace every site with the default one
	var defaults Config
	if err := json.Unmarshal(data, &defaults); err != nil {
		return Config{}, err
	}
	for i := range defaults.Sites {
		defaults.Sites[i] = defaults.Default
	}
	// Unmarshal config again, letting individual sites override the defaults
	cfg := defaults
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ReadConfig(name string) (Config, error) {
	if name == "~/.lftpqrc" {
		home := os.Getenv("HOME")
		name = filepath.Join(home, ".lftpqrc")
	}
	var r io.Reader
	if name == "-" {
		r = bufio.NewReader(os.Stdin)
	} else {
		f, err := os.Open(name)
		if err != nil {
			return Config{}, err
		}
		defer f.Close()
		r = f
	}
	cfg, err := readConfig(r)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.load(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
