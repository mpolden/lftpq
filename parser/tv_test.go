package parser

import (
	"reflect"
	"testing"
)

func TestParseShow(t *testing.T) {
	s := "Gotham.S01E01.720p.HDTV.X264-DIMENSION"
	series, err := ParseShow(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Show{
		Release: s,
		Name:    "Gotham",
		Season:  "01",
		Episode: "01",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}

func TestParseShow2(t *testing.T) {
	s := "Top_Gear.21x02.720p_HDTV_x264-FoV"
	series, err := ParseShow(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Show{
		Release: s,
		Name:    "Top.Gear",
		Season:  "21",
		Episode: "02",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}

func TestParseShow3(t *testing.T) {
	s := "Eastbound.and.Down.S02E05.720p.BluRay.X264-REWARD"
	series, err := ParseShow(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Show{
		Release: s,
		Name:    "Eastbound.and.Down",
		Season:  "02",
		Episode: "05",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}

func TestParseShow4(t *testing.T) {
	_, err := ParseShow("foo")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestParseShow5(t *testing.T) {
	s := "Olive.Kitteridge.Part.4.720p.HDTV.x264-KILLERS"
	series, err := ParseShow(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Show{
		Release: s,
		Name:    "Olive.Kitteridge",
		Season:  "01",
		Episode: "04",
	}
	if !reflect.DeepEqual(series, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, series)
	}
}
