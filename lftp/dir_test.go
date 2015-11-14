package lftp

import (
	"regexp"
	"testing"
	"time"
)

func TestParseDir(t *testing.T) {
	t1 := time.Date(2014, 12, 16, 0, 4, 30, 0, time.FixedZone("CET", 3600))
	t2 := time.Date(2015, 2, 3, 15, 12, 30, 0, time.FixedZone("CET", 3600))
	var tests = []struct {
		in  string
		out Dir
	}{
		{"2014-12-16 00:04:30 +0100 CET /bar/foo/", Dir{Created: t1, Path: "/bar/foo"}},
		{"2015-02-03 15:12:30 +0100 CET /foo/bar@", Dir{Created: t2, Path: "/foo/bar", IsSymlink: true}},
		{"2014-12-16 00:04:30 +0100 CET /foo/bar baz/", Dir{Created: t1, Path: "/foo/bar baz"}},
		{"2014-12-16 00:04:30 +0100 CET /foo/baz", Dir{Created: t1, Path: "/foo/baz", IsFile: true}},
	}
	for _, tt := range tests {
		d, err := ParseDir(tt.in)
		if err != nil {
			t.Error(err)
			continue
		}
		if d.Path != tt.out.Path {
			t.Errorf("Expected %q, got %q", tt.out.Path, d.Path)
		}
		if !d.Created.Equal(tt.out.Created) {
			t.Errorf("Expected %s, got %s", tt.out.Created, d.Created)
		}
		if d.IsSymlink != tt.out.IsSymlink {
			t.Errorf("Expected %t, got %t", tt.out.IsSymlink, d.IsSymlink)
		}
		if d.IsFile != tt.out.IsFile {
			t.Errorf("Expected %t, got %t", tt.out.IsFile, d.IsFile)
		}
	}
}

func TestBase(t *testing.T) {
	in := Dir{Path: "/foo/bar"}
	out := "bar"
	if got := in.Base(); got != out {
		t.Fatalf("Expected %q, got %q", out, got)
	}
}

func TestAge(t *testing.T) {
	now := time.Now().Round(time.Second)
	var tests = []struct {
		in  Dir
		out time.Duration
	}{
		{Dir{Created: now}, time.Duration(0)},
		{Dir{Created: now.Add(-time.Duration(48) * time.Hour)}, time.Duration(48) * time.Hour},
	}
	for _, tt := range tests {
		if got := tt.in.Age(); got != tt.out {
			t.Errorf("Expected %s, got %s", tt.out, got)
		}
	}
}

func TestMatch(t *testing.T) {
	d := Dir{Path: "/tmp/foo"}
	if !d.Match(regexp.MustCompile("f")) {
		t.Fatal("Expected true")
	}
	if d.Match(regexp.MustCompile("bar")) {
		t.Fatal("Expected false")
	}
}

func TestMatchAny(t *testing.T) {
	d := Dir{Path: "/tmp/foo"}
	patterns := []*regexp.Regexp{
		regexp.MustCompile("fo"),
		regexp.MustCompile("ba"),
	}
	if _, match := d.MatchAny(patterns); !match {
		t.Fatal("Expected true")
	}
	patterns = []*regexp.Regexp{
		regexp.MustCompile("x"),
		regexp.MustCompile("z"),
	}
	if _, match := d.MatchAny(patterns); match {
		t.Fatal("Expected false")
	}
}
