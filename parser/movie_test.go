package parser

import (
	"reflect"
	"testing"
)

func TestParseMovie(t *testing.T) {
	s := "Apocalypse.Now.1979.1080p.BluRay-GRP"
	movie, err := ParseMovie(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Movie{
		Release: s,
		Name:    "Apocalypse.Now",
		Year:    1979,
	}
	if !reflect.DeepEqual(movie, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, movie)
	}
}
