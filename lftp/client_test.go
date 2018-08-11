package lftp

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseDirList(t *testing.T) {
	ls := `1403705716 dir1/
	1422918075 dir2/
	1426408110 dir3@`
	want := []file{
		{modTime: time.Date(2014, 6, 25, 14, 15, 16, 0, time.UTC), path: "dir1"},
		{modTime: time.Date(2015, 2, 2, 23, 1, 15, 0, time.UTC), path: "dir2"},
		{modTime: time.Date(2015, 3, 15, 8, 28, 30, 0, time.UTC), path: "dir3"},
	}
	got, err := parseDirList(strings.NewReader(ls))
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range want {
		g := got[i]
		if !w.ModTime().Equal(g.ModTime()) || w.Name() != g.Name() {
			t.Fatalf("want %+v, got %+v", w, g)
		}
	}
}

func TestListArgs(t *testing.T) {
	want := []string{"-e", "cls -1 --classify --date --time-style='%s' /foo && exit", "bar"}
	got := listArgs("bar", "/foo")
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %q, got %s", want, got)
	}
}
