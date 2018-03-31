package queue

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cfg := Config{
		Sites: []Site{Site{
			Name:         "foo",
			Dirs:         []string{"/site"},
			MaxAge:       "24h",
			Patterns:     []string{"^match"},
			Filters:      []string{"^skip"},
			SkipSymlinks: true,
			Parser:       "show",
			LocalDir:     "/tmp/{{ .Name }}",
			Priorities:   []string{"important"},
			Replacements: []Replacement{
				Replacement{
					Pattern:     "\\.the\\.",
					Replacement: ".The.",
				}},
			PostCommand: "xargs echo",
		}},
	}
	if err := cfg.load(); err != nil {
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
	if site.itemParser.template == nil {
		t.Error("Expected template to be compiled")
	}
	if site.itemParser.parser == nil {
		t.Error("Expected parser to be set")
	}
	if len(site.itemParser.replacements) == 0 {
		t.Error("Expected non-empty replacements")
	}
	if want := []string{"xargs", "echo"}; !reflect.DeepEqual(want, site.postCommand.Args) {
		t.Fatalf("Expected %+v, got %+v", want, site.postCommand.Args)
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
