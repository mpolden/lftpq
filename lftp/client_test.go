package lftp

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseDirList(t *testing.T) {
	ls := `2014-06-25 14:15:16 +0200 CEST dir1/
	2015-02-02 23:01:15 +0100 CET dir2/
	2015-03-15 08:28:30 +0100 CET dir3@`
	expected := []File{
		File{modTime: time.Date(2014, 6, 25, 14, 15, 16, 0, time.FixedZone("CEST", 7200)), path: "dir1"},
		File{modTime: time.Date(2015, 2, 2, 23, 1, 15, 0, time.FixedZone("CET", 3600)), path: "dir2"},
		File{modTime: time.Date(2015, 3, 15, 8, 28, 30, 0, time.FixedZone("CET", 3600)), path: "dir3"},
	}
	actual, err := parseDirList(strings.NewReader(ls))
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range expected {
		a := actual[i]
		if !e.ModTime().Equal(a.ModTime()) || e.Name() != a.Name() {
			t.Fatalf("Expected %+v, got %+v", e, a)
		}
	}
}

func TestListArgs(t *testing.T) {
	expected := []string{"-e", "cls -1 --classify --date --time-style='%F %T %z %Z' /foo && exit", "bar"}
	args := listArgs("bar", "/foo")
	if !reflect.DeepEqual(expected, args) {
		t.Fatalf("Expected %q, got %q", expected, args)
	}
}
