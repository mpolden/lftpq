package site

import (
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

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
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
	Dir          string
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
	parser       parser.Parser
	LocalDir     string
	localDir     *template.Template
	Priorities   []string
	priorities   []*regexp.Regexp
	Deduplicate  bool
	PostCommand  string
	Replacements []Replacement
	Merge        bool
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
	if _, err := exec.LookPath(args[0]); err != nil {
		return err
	}
	return nil
}

func (c *Config) Load() error {
	for i, _ := range c.Sites {
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
		site.localDir = tmpl

		if err := isExecutable(site.Client.Path); err != nil {
			return err
		}
		if err := isExecutable(site.PostCommand); err != nil {
			return err
		}

		switch site.Parser {
		case "show":
			site.parser = parser.Show
		case "movie":
			site.parser = parser.Movie
		case "":
			site.parser = parser.Default
		default:
			return fmt.Errorf("invalid parser: %q (must be %q, %q or %q)",
				site.Parser, "show", "movie", "")
		}
	}
	return nil
}

func (c *Config) JSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
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
	for i, _ := range defaults.Sites {
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
	f, err := os.Open(name)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	cfg, err := readConfig(f)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Load(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
