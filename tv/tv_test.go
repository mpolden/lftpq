package tv

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	s := "Gotham.S01E01.720p.HDTV.X264-DIMENSION"
	series, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := TVShow{
		Release: s,
		Name:    "Gotham",
		Season:  "01",
		Episode: "01",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}

func TestParse2(t *testing.T) {
	s := "Top_Gear.21x02.720p_HDTV_x264-FoV"
	series, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := TVShow{
		Release: s,
		Name:    "Top.Gear",
		Season:  "21",
		Episode: "02",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}

func TestParse3(t *testing.T) {
	s := "Eastbound.and.Down.S02E05.720p.BluRay.X264-REWARD"
	series, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := TVShow{
		Release: s,
		Name:    "Eastbound.and.Down",
		Season:  "02",
		Episode: "05",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}

func TestParse4(t *testing.T) {
	_, err := Parse("foo")
	if err == nil {
		t.Fatal("Expected error")
	}
}
