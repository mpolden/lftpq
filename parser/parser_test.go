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
	s := "Apocalypse.Now.1979.1080p.x264.BluRay-GRP"
	movie, err := Movie(s)
	if err != nil {
		t.Fatal(err)
	}
	expected := Media{
		Release:    s,
		Name:       "Apocalypse.Now",
		Year:       1979,
		Resolution: "1080p",
		Codec:      "x264",
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
				Release:    "Gotham.S01E01.720p.HDTV.X264-DIMENSION",
				Name:       "Gotham",
				Season:     1,
				Episode:    1,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"gotham.s01e01.720p.hdtv.x264-dimension",
			Media{
				Release:    "gotham.s01e01.720p.hdtv.x264-dimension",
				Name:       "Gotham",
				Season:     1,
				Episode:    1,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"Top_Gear.21x02.720p_HDTV_x264-FoV",
			Media{
				Release:    "Top_Gear.21x02.720p_HDTV_x264-FoV",
				Name:       "Top_Gear",
				Season:     21,
				Episode:    2,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"Eastbound.and.Down.S02E05.720p.BluRay.X264-REWARD",
			Media{
				Release:    "Eastbound.and.Down.S02E05.720p.BluRay.X264-REWARD",
				Name:       "Eastbound.and.Down",
				Season:     2,
				Episode:    5,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"Olive.Kitteridge.Part.4.720p.HDTV.x264-KILLERS",
			Media{
				Release:    "Olive.Kitteridge.Part.4.720p.HDTV.x264-KILLERS",
				Name:       "Olive.Kitteridge",
				Season:     1,
				Episode:    4,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"Marilyn.The.Secret.Life.of.Marilyn.Monroe.2015.Part1.720p.HDTV.x264-W4F",
			Media{
				Release:    "Marilyn.The.Secret.Life.of.Marilyn.Monroe.2015.Part1.720p.HDTV.x264-W4F",
				Name:       "Marilyn.The.Secret.Life.of.Marilyn.Monroe.2015",
				Season:     1,
				Episode:    1,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"The.Jinx-The.Life.and.Deaths.of.Robert.Durst.E04.1080p.BluRay.x264-ROVERS",
			Media{
				Release:    "The.Jinx-The.Life.and.Deaths.of.Robert.Durst.E04.1080p.BluRay.x264-ROVERS",
				Name:       "The.Jinx-The.Life.and.Deaths.of.Robert.Durst",
				Season:     1,
				Episode:    4,
				Resolution: "1080p",
				Codec:      "x264",
			}},
		{"Adventure.Time.With.Finn.And.Jake.S01.SUBPACK.720p.BluRay.x264-DEiMOS",
			Media{
				Release:    "Adventure.Time.With.Finn.And.Jake.S01.SUBPACK.720p.BluRay.x264-DEiMOS",
				Name:       "Adventure.Time.With.Finn.And.Jake",
				Season:     1,
				Episode:    0,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"Orange.Is.The.New.Black.S02.NORDiC.SUBPACK.BluRay-REQ",
			Media{
				Release: "Orange.Is.The.New.Black.S02.NORDiC.SUBPACK.BluRay-REQ",
				Name:    "Orange.Is.The.New.Black",
				Season:  2,
				Episode: 0,
			}},
		{"Lost.S01E24.Exodus.Part.2.720p.BluRay.x264-SiNNERS",
			Media{
				Release:    "Lost.S01E24.Exodus.Part.2.720p.BluRay.x264-SiNNERS",
				Name:       "Lost",
				Season:     1,
				Episode:    24,
				Resolution: "720p",
				Codec:      "x264",
			}},
		{"Friends.S01E16.S01E17.UNCUT.DVDrip.XviD-SAiNTS",
			Media{
				Release: "Friends.S01E16.S01E17.UNCUT.DVDrip.XviD-SAiNTS",
				Name:    "Friends",
				Season:  1,
				Episode: 16,
				Codec:   "xvid",
			}},
		{"Generation.Kill.Pt.VII.Bomb.in.the.Garden.720p.Bluray.X264-DIMENSION",
			Media{
				Release:    "Generation.Kill.Pt.VII.Bomb.in.the.Garden.720p.Bluray.X264-DIMENSION",
				Name:       "Generation.Kill",
				Season:     1,
				Episode:    7,
				Resolution: "720p",
				Codec:      "x264",
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

func TestRtoi(t *testing.T) {
	assertRtoiErr(t, "foo")
	assertRtoiErr(t, "VIM")
	assertRtoiErr(t, "DMVI")

	assertRtoi(t, "XXXIX", 39)
	assertRtoi(t, "CCXLVI", 246)
	assertRtoi(t, "DCCLXXXIX", 789)
	assertRtoi(t, "MMCDXXI", 2421)

	assertRtoi(t, "CLX", 160)
	assertRtoi(t, "CCVII", 207)
	assertRtoi(t, "MIX", 1009)
	assertRtoi(t, "MLXVI", 1066)

	assertRtoi(t, "MDCCLXXVI", 1776)
	assertRtoi(t, "MCMXVIII", 1918)
	assertRtoi(t, "MCMLIV", 1954)
	assertRtoi(t, "MMXIV", 2014)

	assertRtoi(t, "MMMCMXCIX", 3999)
}

func assertRtoiErr(t *testing.T, rnum string) {
	if n, err := rtoi(rnum); err == nil {
		t.Fatalf("rtoi(%q) = (%d, %v), want error", rnum, n, err)
	}
}

func assertRtoi(t *testing.T, rnum string, expected int) {
	n, err := rtoi(rnum)
	if err != nil {
		t.Fatal(err)
	}
	if n != expected {
		t.Errorf("rtoi(%q) = %d, want %d", rnum, n, expected)
	}
}
