package site

import (
	"regexp"
	"testing"
	"time"
)

func TestParseDir(t *testing.T) {
	s := "2014-12-16 00:04:30 +0100 CET foo/"
	dir, err := ParseDir("/tmp", s)
	if err != nil {
		t.Fatal(err)
	}
	if name := "foo"; dir.Base() != name {
		t.Fatalf("Expected %s, got %s", name, dir.Base())
	}
	d := time.Date(2014, 12, 16, 0, 4, 30, 0, time.FixedZone("CET", 1))
	if created := d; dir.Created.Equal(created) {
		t.Fatalf("Expected %s, got %s", created, dir.Created)
	}
}

func TestCreatedAfter(t *testing.T) {
	age := time.Duration(24) * time.Hour
	d1 := Dir{
		Path:    "/tmp/foo",
		Created: time.Now(),
	}
	if !d1.CreatedAfter(age) {
		t.Fatal("Expected true")
	}
	d2 := Dir{
		Path:    "/tmp/bar",
		Created: time.Now().Add(-time.Duration(48) * time.Hour),
	}
	if d2.CreatedAfter(age) {
		t.Fatal("Expected false")
	}
}

func TestMatch(t *testing.T) {
	d := Dir{
		Path:    "/tmp/foo",
		Created: time.Now(),
	}
	if !d.Match(regexp.MustCompile("f")) {
		t.Fatal("Expected true")
	}
	if d.Match(regexp.MustCompile("bar")) {
		t.Fatal("Expected false")
	}
}

func TestMatchAny(t *testing.T) {
	d := Dir{
		Path:    "/tmp/foo",
		Created: time.Now(),
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile("fo"),
		regexp.MustCompile("ba"),
	}
	if !d.MatchAny(patterns) {
		t.Fatal("Expected true")
	}
	patterns = []*regexp.Regexp{
		regexp.MustCompile("x"),
		regexp.MustCompile("z"),
	}
	if d.MatchAny(patterns) {
		t.Fatal("Expected false")
	}
}
