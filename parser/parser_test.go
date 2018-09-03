package parser

import (
	"reflect"
	"regexp"
	"testing"
)

func TestEqual(t *testing.T) {
	var tests = []struct {
		a   Media
		b   Media
		out bool
	}{
		{
			Media{Name: "The.Wire", Season: 1, Episode: 1},
			Media{Name: "The.Wire", Season: 1, Episode: 1},
			true,
		},
		{
			Media{Name: "The.Wire", Season: 1, Episode: 1},
			Media{Name: "The.Wire", Season: 2, Episode: 1},
			false,
		},
		{
			Media{Name: "Apocalypse.Now", Year: 1979, Release: "foo"},
			Media{Name: "Apocalypse.Now", Year: 1979, Release: "bar"},
			true,
		},
		{
			Media{Name: "Apocalypse.Now", Year: 1979},
			Media{Name: "The.Shawshank.Redemption", Year: 1994},
			false,
		},
		{
			Media{},
			Media{},
			false,
		},
	}
	for _, tt := range tests {
		if in := tt.a.Equal(tt.b); in != tt.out {
			t.Errorf("Expected %t, got %t", tt.out, in)
		}
	}
}

func TestDefault(t *testing.T) {
	m, err := Default("foo")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m.Release, "foo"; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestMovie(t *testing.T) {
	s := "Apocalypse.Now.1979.1080p.BluRay-GRP"
	movie, err := Movie(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Media{
		Release: s,
		Name:    "Apocalypse.Now",
		Year:    1979,
	}
	if !reflect.DeepEqual(movie, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, movie)
	}
}

func TestMovieFail(t *testing.T) {
	_, err := Movie("foo")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestShow(t *testing.T) {
	var tests = []struct {
		in  string
		out Media
	}{
		{"Gotham.S01E01.720p.HDTV.X264-DIMENSION",
			Media{
				Release: "Gotham.S01E01.720p.HDTV.X264-DIMENSION",
				Name:    "Gotham",
				Season:  1,
				Episode: 1,
			}},
		{"Top_Gear.21x02.720p_HDTV_x264-FoV",
			Media{
				Release: "Top_Gear.21x02.720p_HDTV_x264-FoV",
				Name:    "Top_Gear",
				Season:  21,
				Episode: 2,
			}},
		{"Eastbound.and.Down.S02E05.720p.BluRay.X264-REWARD",
			Media{
				Release: "Eastbound.and.Down.S02E05.720p.BluRay.X264-REWARD",
				Name:    "Eastbound.and.Down",
				Season:  2,
				Episode: 5,
			}},
		{"Olive.Kitteridge.Part.4.720p.HDTV.x264-KILLERS",
			Media{
				Release: "Olive.Kitteridge.Part.4.720p.HDTV.x264-KILLERS",
				Name:    "Olive.Kitteridge",
				Season:  1,
				Episode: 4,
			}},
		{"Marilyn.The.Secret.Life.of.Marilyn.Monroe.2015.Part1.720p.HDTV.x264-W4F",
			Media{
				Release: "Marilyn.The.Secret.Life.of.Marilyn.Monroe.2015.Part1.720p.HDTV.x264-W4F",
				Name:    "Marilyn.The.Secret.Life.of.Marilyn.Monroe.2015",
				Season:  1,
				Episode: 1,
			}},
		{"The.Jinx-The.Life.and.Deaths.of.Robert.Durst.E04.1080p.BluRay.x264-ROVERS",
			Media{
				Release: "The.Jinx-The.Life.and.Deaths.of.Robert.Durst.E04.1080p.BluRay.x264-ROVERS",
				Name:    "The.Jinx-The.Life.and.Deaths.of.Robert.Durst",
				Season:  1,
				Episode: 4,
			}},
		{"Adventure.Time.With.Finn.And.Jake.S01.SUBPACK.720p.BluRay.x264-DEiMOS",
			Media{
				Release: "Adventure.Time.With.Finn.And.Jake.S01.SUBPACK.720p.BluRay.x264-DEiMOS",
				Name:    "Adventure.Time.With.Finn.And.Jake",
				Season:  1,
				Episode: 0,
			}},
		{"Orange.Is.The.New.Black.S02.NORDiC.SUBPACK.BluRay-REQ",
			Media{
				Release: "Orange.Is.The.New.Black.S02.NORDiC.SUBPACK.BluRay-REQ",
				Name:    "Orange.Is.The.New.Black",
				Season:  2,
				Episode: 0,
			}},
	}
	for _, tt := range tests {
		got, err := Show(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.out) {
			t.Errorf("Expected %+v, got %+v", tt.out, got)
		}
	}

}

func TestShowFail(t *testing.T) {
	_, err := Show("foo")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestReplaceName(t *testing.T) {
	m := Media{Name: "Youre.The.Worst"}
	re := regexp.MustCompile(`\.The\.`)
	m.ReplaceName(re, ".the.")
	if want := "Youre.the.Worst"; m.Name != want {
		t.Errorf("Expected %q, got %q", want, m.Name)
	}
}
