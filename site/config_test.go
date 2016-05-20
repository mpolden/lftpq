package site

import (
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cfg := Config{
		Sites: []Site{Site{
			Name:         "foo",
			Dir:          "/site",
			MaxAge:       "24h",
			Patterns:     []string{"^match"},
			Filters:      []string{"^skip"},
			SkipSymlinks: true,
			Parser:       "show",
			LocalDir:     "/tmp/{{ .Name }}",
			Priorities:   []string{"important"},
			Deduplicate:  true,
			Replacements: []Replacement{
				Replacement{
					Pattern:     "\\.the\\.",
					Replacement: ".The.",
				}},
		}},
	}
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}

	site := cfg.Sites[0]
	if want := time.Duration(24) * time.Hour; site.maxAge != want {
		t.Errorf("Expected %s, got %s", want, site.maxAge)
	}
	if len(site.patterns) == 0 {
		t.Error("Expected non-empty patterns")
	}
	if len(site.filters) == 0 {
		t.Error("Expected non-empty filters")
	}
	if len(site.priorities) == 0 {
		t.Error("Expected non-empty priorities")
	}
	if site.localDir == nil {
		t.Error("Expected template to be compiled")
	}
	if site.parser == nil {
		t.Error("Expected parser to be set")
	}
	if len(site.Replacements) == 0 {
		t.Error("Expected non-empty replacements")
	}
}

func TestReadConfig(t *testing.T) {
	jsonConfig := `
{
  "Default": {
    "Parser": "show"
  },
  "Sites": [
    {
      "Name": "foo"
    },
    {
      "Name": "bar",
      "Parser": "movie"
    },
    {
      "Name": "baz",
      "Parser": ""
    }
  ]
}
`
	cfg, err := readConfig(strings.NewReader(jsonConfig))
	if err != nil {
		t.Fatal(err)
	}
	// Test that defaults are applied and can be overridden
	var tests = []struct {
		i   int
		out string
	}{
		{0, "show"},
		{1, "movie"},
		{2, ""},
	}
	for _, tt := range tests {
		site := cfg.Sites[tt.i]
		if got := site.Parser; got != tt.out {
			t.Errorf("Expected Parser=%q, got Parser=%q for Name=%q", tt.out, got, site.Name)
		}
	}
}

func TestLookupSite(t *testing.T) {
	s := Site{Name: "foo"}
	cfg := Config{Sites: []Site{s}}
	site, err := cfg.LookupSite("foo")
	if err != nil {
		t.Fatal(err)
	}
	if site.Name != s.Name {
		t.Errorf("Expected %q, got %q", s.Name, site.Name)
	}
	if _, err := cfg.LookupSite("bar"); err == nil {
		t.Error("Expected error")
	}
}
