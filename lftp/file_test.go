package lftp

import (
	"os"
	"testing"
	"time"
)

func TestParseFile(t *testing.T) {
	t1 := time.Date(2014, 12, 16, 0, 4, 30, 0, time.UTC)
	t2 := time.Date(2015, 2, 3, 15, 12, 30, 0, time.UTC)
	var tests = []struct {
		in        string
		out       file
		IsSymlink bool
		IsDir     bool
		IsRegular bool
	}{
		{"1418688270 /bar/foo/", file{modTime: t1, path: "/bar/foo"},
			false /* IsDir */, true, false},
		{"1422976350 /foo/bar@",
			file{modTime: t2, path: "/foo/bar"} /* IsSymlink */, true, false, false},
		{"1418688270 /foo/bar baz/",
			file{modTime: t1, path: "/foo/bar baz"}, false /* IsDir */, true, false},
		{"1418688270 /foo/baz",
			file{modTime: t1, path: "/foo/baz"}, false, false /* IsRegular */, true},
	}
	for _, tt := range tests {
		f, err := ParseFile(tt.in)
		if err != nil {
			t.Error(err)
			continue
		}
		if f.Name() != tt.out.Name() {
			t.Errorf("Expected %q, got %q", tt.out.Name(), f.Name())
		}
		if !f.ModTime().Equal(tt.out.ModTime()) {
			t.Errorf("Expected %s, got %s", tt.out.ModTime(), f.ModTime())
		}
		if f.IsDir() != tt.IsDir {
			t.Errorf("Expected IsDir=%t, got %t", tt.IsDir, f.IsDir())
		}
		if isSymlink := f.Mode()&os.ModeSymlink != 0; isSymlink != tt.IsSymlink {
			t.Errorf("Expected IsSymlink=%t, got %t", tt.IsSymlink, isSymlink)
		}
		if f.Mode().IsRegular() != tt.IsRegular {
			t.Errorf("Expected IsRegular=%t, got %t", tt.IsRegular, f.Mode().IsRegular())
		}
	}
}
