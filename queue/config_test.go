package queue

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cfg := Config{
		LocalDirs: []LocalDir{
			{
				Name:   "d1",
				Parser: "show",
				Dir:    "/tmp/{{ .Name }}",
				Replacements: []Replacement{
					{
						Pattern:     "\\.the\\.",
						Replacement: ".The.",
					}},
			},
		},
		Sites: []Site{{
			Name:         "foo",
			Dirs:         []string{"/site"},
			MaxAge:       "24h",
			Patterns:     []string{"^match"},
			Filters:      []string{"^skip"},
			SkipSymlinks: true,
			LocalDir:     "d1",
			Priorities:   []string{"important"},

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
	if site.localDir.Template == nil {
		t.Error("Expected template to be compiled")
	}
	if site.localDir.parser == nil {
		t.Error("Expected parser to be set")
	}
	if len(site.localDir.Replacements) == 0 {
		t.Error("Expected non-empty replacements")
	}
	if want := []string{"xargs", "echo"}; !reflect.DeepEqual(want, site.postCommand.Args) {
		t.Fatalf("Expected %+v, got %+v", want, site.postCommand.Args)
	}
}

func TestReadConfig(t *testing.T) {
	jsonConfig := `
{
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "show",
      "Dir": "/tmp/d1/"
    },
    {
      "Name": "d2",
      "Parser": "movie",
      "Dir": "/tmp/d2/"
    }
  ],
  "Default": {
    "LocalDir": "d1"
  },
  "Sites": [
    {
      "Name": "foo"
    },
    {
      "Name": "bar",
      "LocalDir": "d2"
    },
    {
      "Name": "baz",
      "LocalDir": ""
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
		{0, "d1"},
		{1, "d2"},
		{2, ""},
	}
	for _, tt := range tests {
		site := cfg.Sites[tt.i]
		if got := site.LocalDir; got != tt.out {
			t.Errorf("got LocalDir=%q, want %q for Name=%q", got, tt.out, site.Name)
		}
	}
}

func TestSetLocalDir(t *testing.T) {
	jsonConfig := `
{
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "show",
      "Dir": "/tmp/d1/"
    },
    {
      "Name": "d2",
      "Parser": "movie",
      "Dir": "/tmp/d2/"
    }
  ],
  "Default": {
    "LocalDir": "d1",
    "MaxAge": "24h"
  },
  "Sites": [
    {
      "Name": "foo"
    },
    {
      "Name": "bar"
    }
  ]
}
`
	cfg, err := readConfig(strings.NewReader(jsonConfig))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.load(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetLocalDir("invalid"); err == nil {
		t.Fatal("want error")
	}
	// Config remains unchanged after invalid local dir
	for _, s := range cfg.Sites {
		if want := "d1"; s.LocalDir != want {
			t.Errorf("got %q, want %q", s.LocalDir, want)
		}
	}
	if err := cfg.SetLocalDir("d2"); err != nil {
		t.Fatal(err)
	}
	for _, s := range cfg.Sites {
		if want := "d2"; s.LocalDir != want {
			t.Errorf("got %q, want %q", s.LocalDir, want)
		}
	}
}

func TestExpandUser(t *testing.T) {
	home := "/home/foo"
	os.Setenv("HOME", home)
	var tests = []struct {
		in  string
		out string
	}{
		{"", ""},
		{"/d", "/d"},
		{"~", home},
		{"~/", home},
		{"~bar", "/home/bar"},
		{"~bar/baz", "/home/bar/baz"},
		{"~/bar", home + "/bar"},
		{"/foo/~/bar", "/foo/~/bar"},
	}
	for i, tt := range tests {
		out := expandUser(tt.in)
		if out != tt.out {
			t.Errorf("#%d: expandUser(%q) = %q, want %q", i, tt.in, out, tt.out)
		}
	}
}
