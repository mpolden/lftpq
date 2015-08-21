package site

import (
	"regexp"
	"testing"
	"time"
)

func TestParseDir(t *testing.T) {
	s := "2014-12-16 00:04:30 +0100 CET foo/"
	dir, err := ParseDir(s)
	if err != nil {
		t.Fatal(err)
	}
	if name := "foo"; dir.Base() != name {
		t.Fatalf("Expected %s, got %s", name, dir.Base())
	}
	d := time.Date(2014, 12, 16, 0, 4, 30, 0, time.FixedZone("CET", 3600))
	if !d.Equal(dir.Created) {
		t.Fatalf("Expected %s, got %s", d, dir.Created)
	}
}

func TestAge(t *testing.T) {
	now := time.Now().Round(time.Second)
	d1 := Dir{
		Path:    "/tmp/foo",
		Created: now,
	}
	age := d1.Age()
	if expected := time.Duration(0); age != expected {
		t.Errorf("Expected %q, got %q", expected, age)
	}

	d2 := Dir{
		Path:    "/tmp/bar",
		Created: now.Add(-time.Duration(48) * time.Hour),
	}
	age = d2.Age()
	if expected := time.Duration(48) * time.Hour; age != expected {
		t.Errorf("Expected %q, got %q", expected, age)
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
