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

	"github.com/mpolden/lftpq/parser"
)

type Config struct {
	Default   Site
	LocalDirs []LocalDir
	Sites     []Site
}

type Replacement struct {
	Pattern     string
	pattern     *regexp.Regexp
	Replacement string
}

type LocalDir struct {
	Name         string
	Parser       string
	Dir          string
	Replacements []Replacement
}

type Site struct {
	GetCmd       string
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
	LocalDir     string
	Priorities   []string
	priorities   []*regexp.Regexp
	PostCommand  string
	postCommand  *exec.Cmd
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

func command(cmd string) (*exec.Cmd, error) {
	if cmd == "" {
		return nil, nil
	}
	argv := strings.Split(cmd, " ")
	if _, err := exec.LookPath(argv[0]); err != nil {
		return nil, err
	}
	return exec.Command(argv[0], argv[1:]...), nil
}

func (c *Config) itemParsers() (map[string]itemParser, error) {
	itemParsers := make(map[string]itemParser)
	for _, d := range c.LocalDirs {
		if d.Name == "" {
			return nil, fmt.Errorf("invalid local dir name: %q", d.Name)
		}
		if d.Dir == "" {
			return nil, fmt.Errorf("invalid local dir path: %q", d.Dir)
		}
		var parserFunc parser.Parser
		switch d.Parser {
		case "show":
			parserFunc = parser.Show
		case "movie":
			parserFunc = parser.Movie
		case "":
			parserFunc = parser.Default
		default:
			return nil, fmt.Errorf("invalid local dir %q: invalid parser: %q (must be %q, %q or %q)",
				d.Name, d.Parser, "show", "movie", "")
		}
		tmpl, err := parseTemplate(d.Dir)
		if err != nil {
			return nil, err
		}
		replacements, err := compileReplacements(d.Replacements)
		if err != nil {
			return nil, err
		}
		itemParsers[d.Name] = itemParser{
			parser:       parserFunc,
			replacements: replacements,
			template:     tmpl,
		}
	}
	return itemParsers, nil
}

func (c *Config) load() error {
	itemParsers, err := c.itemParsers()
	if err != nil {
		return err
	}
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

		cmd, err := command(site.PostCommand)
		if err != nil {
			return err
		}
		site.postCommand = cmd

		itemParser, ok := itemParsers[site.LocalDir]
		if !ok {
			return fmt.Errorf("site: %q: invalid local dir: %q", site.Name, site.LocalDir)
		}
		site.itemParser = itemParser
	}
	return nil
}

func (c *Config) SetLocalDir(name string) error {
	itemParsers, err := c.itemParsers()
	if err != nil {
		return err
	}
	_, ok := itemParsers[name]
	if !ok {
		return fmt.Errorf("invalid local dir: %q", name)
	}
	for i := range c.Sites {
		c.Sites[i].LocalDir = name
	}
	return c.load()
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
