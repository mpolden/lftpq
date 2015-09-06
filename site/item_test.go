package site

import (
	"regexp"
	"sort"
	"testing"
	"text/template"

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
)

func TestNewItemShow(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir: tmpl,
		parser:   parser.Show,
	}
	d := lftp.Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if expected := "/tmp/The.Wire/S03/"; item.LocalDir != expected {
		t.Fatalf("Expected %q, got %q", expected, item.LocalDir)
	}
}

func TestNewItemMovie(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Year }}/{{ .Name }}/"))
	s := Site{
		localDir: tmpl,
		parser:   parser.Movie,
	}
	d := lftp.Dir{Path: "/foo/Apocalypse.Now.1979"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if expected := "/tmp/1979/Apocalypse.Now/"; item.LocalDir != expected {
		t.Fatalf("Expected %q, got %q", expected, item.LocalDir)
	}
}

func TestNewItemDefaultParser(t *testing.T) {
	s := Site{
		localDir: template.Must(template.New("").Parse("/tmp/")),
		parser:   parser.Default,
	}
	d := lftp.Dir{Path: "/foo/The.Wire.S03E01"}
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
		parser:   parser.Show,
	}
	d := lftp.Dir{Path: "/foo/bar"}
	q := Queue{Site: s}
	item := newItem(&q, d)
	if item.LocalDir != "" {
		t.Fatal("Expected empty string")
	}
	if item.Transfer {
		t.Fatal("Expected item to be rejected")
	}
	if item.Reason == "" {
		t.Fatal("Expected non-empty reason")
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
		{Item{Queue: &q, Dir: lftp.Dir{Path: "/tmp/The.Wire.S01E01.foo"}}, 0},
		{Item{Queue: &q, Dir: lftp.Dir{Path: "/tmp/The.Wire.S01E01.PROPER.foo"}}, 2},
		{Item{Queue: &q, Dir: lftp.Dir{Path: "/tmp/The.Wire.S01E01.REPACK.foo"}}, 1},
	}
	for _, tt := range tests {
		if in := tt.in.Weight(); in != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, in)
		}
	}
}

func TestItemsSort(t *testing.T) {
	items := Items{
		Item{Dir: lftp.Dir{Path: "/x/c"}},
		Item{Dir: lftp.Dir{Path: "/x/b"}},
		Item{Dir: lftp.Dir{Path: "/x/a"}},
		Item{Dir: lftp.Dir{Path: "/y/a"}},
	}
	sort.Sort(items)
	var tests = []struct {
		in  int
		out string
	}{
		{0, "/x/a"},
		{1, "/x/b"},
		{2, "/x/c"},
		{3, "/y/a"},
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
		t.Errorf("Expected %q, got %q", expected, item.Reason)
	}
}

func TestReject(t *testing.T) {
	item := Item{}
	item.Reject("bar")
	if item.Transfer {
		t.Error("Expected false")
	}
	if expected := "bar"; item.Reason != expected {
		t.Errorf("Expected %q, got %q", expected, item.Reason)
	}
}

func TestDstDir(t *testing.T) {
	var tests = []struct {
		in  Item
		out string
	}{
		{Item{Dir: lftp.Dir{Path: "/foo/bar"}, LocalDir: "/tmp/"}, "/tmp/bar"},
		{Item{Dir: lftp.Dir{Path: "/foo/bar"}, LocalDir: "/tmp/foo/bar"}, "/tmp/foo/bar"},
	}
	for _, tt := range tests {
		if got := tt.in.DstDir(); got != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, got)
		}
	}
}
