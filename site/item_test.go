package site

import (
	"regexp"
	"sort"
	"testing"
	"text/template"

	"github.com/martinp/lftpq/parser"
)

func TestNewItemShow(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir: tmpl,
		Parser:   "show",
	}
	d := Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if expected := "/tmp/The.Wire/S03/"; item.LocalDir != expected {
		t.Fatalf("Expected %s, got %s", expected, item.LocalDir)
	}
}

func TestNewItemMovie(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Year }}/{{ .Name }}/"))
	s := Site{
		localDir: tmpl,
		Parser:   "movie",
	}
	d := Dir{Path: "/foo/Apocalypse.Now.1979"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if expected := "/tmp/1979/Apocalypse.Now/"; item.LocalDir != expected {
		t.Fatalf("Expected %s, got %s", expected, item.LocalDir)
	}
}

func TestNewItemWithoutParser(t *testing.T) {
	s := Site{
		LocalDir: "/tmp/",
	}
	d := Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if expected := "/tmp/"; item.LocalDir != expected {
		t.Fatalf("Expected %s, got %s", expected, item.LocalDir)
	}
}

func TestNewItemUnparsable(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir: tmpl,
		Parser:   "show",
	}
	d := Dir{Path: "/foo/bar"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if item.LocalDir != "" {
		t.Fatal("Expected empty string")
	}
}

func TestWeight(t *testing.T) {
	s := Site{
		priorities: []*regexp.Regexp{regexp.MustCompile("\\.PROPER\\."), regexp.MustCompile("\\.REPACK\\.")},
	}
	q := Queue{Site: s}
	var tests = []struct {
		in  Item
		out int
	}{
		{Item{Queue: &q, Dir: Dir{Path: "/tmp/The.Wire.S01E01.foo"}}, 0},
		{Item{Queue: &q, Dir: Dir{Path: "/tmp/The.Wire.S01E01.PROPER.foo"}}, 2},
		{Item{Queue: &q, Dir: Dir{Path: "/tmp/The.Wire.S01E01.REPACK.foo"}}, 1},
	}
	for _, tt := range tests {
		if in := tt.in.Weight(); in != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, in)
		}
	}
}

func TestMediaEqual(t *testing.T) {
	var tests = []struct {
		a   Item
		b   Item
		out bool
	}{
		{
			Item{Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "01"}},
			Item{Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "01"}},
			true,
		},
		{
			Item{Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "01"}},
			Item{Media: parser.Show{Name: "The.Wire", Season: "02", Episode: "01"}},
			false,
		},
		{
			Item{Media: parser.Movie{Name: "Apocalypse.Now", Year: 1979, Release: "foo"}},
			Item{Media: parser.Movie{Name: "Apocalypse.Now", Year: 1979, Release: "bar"}},
			true,
		},
		{
			Item{Media: parser.Movie{Name: "Apocalypse.Now", Year: 1979}},
			Item{Media: parser.Movie{Name: "The.Shawshank.Redemption", Year: 1994}},
			false,
		},
	}
	for _, tt := range tests {
		if in := tt.a.MediaEqual(tt.b); in != tt.out {
			t.Errorf("Expected %t, got %t", tt.out, in)
		}
	}
}

func TestItemsSort(t *testing.T) {
	items := Items{
		Item{Dir: Dir{Path: "/x/c"}},
		Item{Dir: Dir{Path: "/x/b"}},
		Item{Dir: Dir{Path: "/x/a"}},
	}
	sort.Sort(items)
	var tests = []struct {
		in  int
		out string
	}{
		{0, "/x/a"},
		{1, "/x/b"},
		{2, "/x/c"},
	}
	for _, tt := range tests {
		if got := items[tt.in].Dir.Path; got != tt.out {
			t.Errorf("Expected index %d to be %q, got %q", tt.in, tt.out, got)
		}
	}
}

func TestAccept(t *testing.T) {
	item := Item{}
	item.Accept("foo")
	if !item.Transfer {
		t.Error("Expected true")
	}
	if expected := "foo"; item.Reason != expected {
		t.Errorf("Expected %q, got %q", item.Reason)
	}
}

func TestReject(t *testing.T) {
	item := Item{}
	item.Reject("bar")
	if item.Transfer {
		t.Error("Expected false")
	}
	if expected := "bar"; item.Reason != expected {
		t.Errorf("Expected %q, got %q", item.Reason)
	}
}
