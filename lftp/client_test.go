package lftp

import (
	"strings"
	"testing"
	"time"
)

func TestParseDirList(t *testing.T) {
	ls := `2014-06-25 14:15:16 +0200 CEST dir1/
	2015-02-02 23:01:15 +0100 CET dir2/
	2015-03-15 08:28:30 +0100 CET dir3@`
	expected := []Dir{
		Dir{Created: time.Date(2014, 6, 25, 14, 15, 16, 0, time.FixedZone("CEST", 7200)), Path: "dir1"},
		Dir{Created: time.Date(2015, 2, 2, 23, 1, 15, 0, time.FixedZone("CET", 3600)), Path: "dir2"},
		Dir{Created: time.Date(2015, 3, 15, 8, 28, 30, 0, time.FixedZone("CET", 3600)), Path: "dir3",
			IsSymlink: true},
	}
	actual, err := parseDirList(strings.NewReader(ls))
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range expected {
		a := actual[i]
		if !e.Created.Equal(a.Created) || e.Path != a.Path || e.IsSymlink != a.IsSymlink {
			t.Fatalf("Expected %+v, got %+v", e, a)
		}
	}
}
