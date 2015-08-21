package site

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
	"time"

	"github.com/martinp/lftpq/parser"
)

type Config struct {
	Client Client
	Sites  []Site
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

func parseTemplate(tmpl string) (*template.Template, error) {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func ReadConfig(name string) (Config, error) {
	if name == "~/.lftpqrc" {
		home := os.Getenv("HOME")
		name = filepath.Join(home, ".lftpqrc")
	}
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	for i, site := range cfg.Sites {
		maxAge, err := time.ParseDuration(site.MaxAge)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].maxAge = maxAge
		patterns, err := compilePatterns(site.Patterns)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].patterns = patterns
		filters, err := compilePatterns(site.Filters)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].filters = filters
		priorities, err := compilePatterns(site.Priorities)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].priorities = priorities
		cfg.Sites[i].Client = cfg.Client
		tmpl, err := parseTemplate(site.LocalDir)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].localDir = tmpl
		switch site.Parser {
		case "show":
			site.parser = parser.Show
		case "movie":
			site.parser = parser.Movie
		case "":
			site.parser = parser.Default
		default:
			return Config{}, fmt.Errorf("invalid parser: %q (must be %q, %q or %q)",
				site.Parser, "show", "movie", "")
		}
	}
	return cfg, nil
}
