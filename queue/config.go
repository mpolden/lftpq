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
	Template     *template.Template `json:"-"`
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

func expandUser(path string) string {
	tilde := strings.Index(path, "~")
	end := strings.IndexRune(path, os.PathSeparator)
	if tilde != 0 {
		return path
	}
	if end == -1 {
		end = len(path)
	}
	home := os.Getenv("HOME")
	if end > 1 {
		home = filepath.Join(filepath.Dir(home), path[1:end])
	}
	return filepath.Join(home, path[end:])
}

func command(cmd string) (*exec.Cmd, error) {
	if cmd == "" {
		return nil, nil
	}
	argv := strings.Split(cmd, " ")
	program := expandUser(argv[0])
	if _, err := exec.LookPath(program); err != nil {
		return nil, err
	}
	return exec.Command(program, argv[1:]...), nil
}

func (c *Config) load() error {
	itemParsers := make(map[string]itemParser)
	for i, d := range c.LocalDirs {
		if d.Name == "" {
			return fmt.Errorf("invalid local dir name: %q", d.Name)
		}
		if d.Dir == "" {
			return fmt.Errorf("invalid local dir path: %q", d.Dir)
		}
		if _, ok := itemParsers[d.Name]; ok {
			return fmt.Errorf("invalid local dir: %q: declared multiple times", d.Name)
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
			return fmt.Errorf("invalid local dir %q: invalid parser: %q (must be %q, %q or %q)",
				d.Name, d.Parser, "show", "movie", "")
		}
		tmpl, err := parseTemplate(d.Dir)
		if err != nil {
			return err
		}
		replacements, err := compileReplacements(d.Replacements)
		if err != nil {
			return err
		}
		itemParsers[d.Name] = itemParser{
			parser:       parserFunc,
			replacements: replacements,
			template:     tmpl,
		}
		c.LocalDirs[i].Template = tmpl
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
	newCfg := *c
	newCfg.Sites = make([]Site, len(c.Sites))
	for i, s := range c.Sites {
		s.LocalDir = name
		newCfg.Sites[i] = s
	}
	if err := newCfg.load(); err != nil {
		return err
	}
	*c = newCfg
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

func ReadConfig(path string) (Config, error) {
	path = expandUser(path)
	var r io.Reader
	if path == "-" {
		r = bufio.NewReader(os.Stdin)
	} else {
		f, err := os.Open(path)
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
