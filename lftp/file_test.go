package lftp

import (
	"regexp"
	"testing"
	"time"
)

func TestParseFile(t *testing.T) {
	t1 := time.Date(2014, 12, 16, 0, 4, 30, 0, time.FixedZone("CET", 3600))
	t2 := time.Date(2015, 2, 3, 15, 12, 30, 0, time.FixedZone("CET", 3600))
	var tests = []struct {
		in        string
		out       File
		IsSymlink bool
		IsDir     bool
		IsRegular bool
	}{
		{"2014-12-16 00:04:30 +0100 CET /bar/foo/", File{Created: t1, Path: "/bar/foo"},
			false /* IsDir */, true, false},
		{"2015-02-03 15:12:30 +0100 CET /foo/bar@",
			File{Created: t2, Path: "/foo/bar"} /* IsSymlink */, true, false, false},
		{"2014-12-16 00:04:30 +0100 CET /foo/bar baz/",
			File{Created: t1, Path: "/foo/bar baz"}, false /* IsDir */, true, false},
		{"2014-12-16 00:04:30 +0100 CET /foo/baz",
			File{Created: t1, Path: "/foo/baz"}, false, false /* IsRegular */, true},
	}
	for _, tt := range tests {
		f, err := ParseFile(tt.in)
		if err != nil {
			t.Error(err)
			continue
		}
		if f.Path != tt.out.Path {
			t.Errorf("Expected %q, got %q", tt.out.Path, f.Path)
		}
		if !f.Created.Equal(tt.out.Created) {
			t.Errorf("Expected %s, got %s", tt.out.Created, f.Created)
		}
		if f.IsDir() != tt.IsDir {
			t.Errorf("Expected IsDir=%t, got %t", tt.IsDir, f.IsDir())
		}
		if f.IsSymlink() != tt.IsSymlink {
			t.Errorf("Expected IsSymlink=%t, got %t", tt.IsSymlink, f.IsSymlink())
		}
		if f.IsRegular() != tt.IsRegular {
			t.Errorf("Expected IsRegular=%t, got %t", tt.IsRegular, f.IsRegular())
		}
	}
}

func TestBase(t *testing.T) {
	in := File{Path: "/foo/bar"}
	out := "bar"
	if got := in.Base(); got != out {
		t.Fatalf("Expected %q, got %q", out, got)
	}
}

func TestAge(t *testing.T) {
	now := time.Now().Round(time.Second)
	var tests = []struct {
		in  File
		out time.Duration
	}{
		{File{Created: now}, time.Duration(0)},
		{File{Created: now.Add(-time.Duration(48) * time.Hour)}, time.Duration(48) * time.Hour},
	}
	for _, tt := range tests {
		if got := tt.in.Age(now); got != tt.out {
			t.Errorf("Expected %s, got %s", tt.out, got)
		}
	}
}

func TestMatch(t *testing.T) {
	f := File{Path: "/tmp/foo"}
	if !f.Match(regexp.MustCompile("f")) {
		t.Fatal("Expected true")
	}
	if f.Match(regexp.MustCompile("bar")) {
		t.Fatal("Expected false")
	}
}

func TestMatchAny(t *testing.T) {
	f := File{Path: "/tmp/foo"}
	patterns := []*regexp.Regexp{
		regexp.MustCompile("fo"),
		regexp.MustCompile("ba"),
	}
	if _, match := f.MatchAny(patterns); !match {
		t.Fatal("Expected true")
	}
	patterns = []*regexp.Regexp{
		regexp.MustCompile("x"),
		regexp.MustCompile("z"),
	}
	if _, match := f.MatchAny(patterns); match {
		t.Fatal("Expected false")
	}
}
