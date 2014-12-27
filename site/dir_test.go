package site

import (
	"testing"
	"time"
)

func TestParseDir(t *testing.T) {
	s := "2014-12-16 00:04:30 +0100 CET foo/"
	dir, err := ParseDir(s)
	if err != nil {
		t.Fatal(err)
	}
	if name := "foo"; dir.Name != name {
		t.Fatalf("Expected %s, got %s", name, dir.Name)
	}
	d := time.Date(2014, 12, 16, 0, 4, 30, 0, time.FixedZone("CET", 1))
	if created := d; dir.Created.Equal(created) {
		t.Fatalf("Expected %s, got %s", created, dir.Created)
	}
}

func TestCreatedAfter(t *testing.T) {
	age := time.Duration(24) * time.Hour
	d1 := Dir{
		Name:    "foo",
		Created: time.Now(),
	}
	if !d1.CreatedAfter(age) {
		t.Fatal("Expected true")
	}
	d2 := Dir{
		Name:    "bar",
		Created: time.Now().Add(-time.Duration(48) * time.Hour),
	}
	if d2.CreatedAfter(age) {
		t.Fatal("Expected false")
	}
}

func TestMatch(t *testing.T) {
	d := Dir{
		Name:    "foo",
		Created: time.Now(),
	}
	if !d.Match("f") {
		t.Fatal("Expected true")
	}
	if d.Match("o") {
		t.Fatal("Expected false")
	}
}

func TestMatchAny(t *testing.T) {
	d := Dir{
		Name:    "foo",
		Created: time.Now(),
	}
	if !d.MatchAny([]string{"b", "f"}) {
		t.Fatal("Expected true")
	}
	if d.MatchAny([]string{"x", "z"}) {
		t.Fatal("Expected false")
	}
}
