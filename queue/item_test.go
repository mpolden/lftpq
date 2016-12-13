package queue

import (
	"os"
	"regexp"
	"sort"
	"testing"
	"text/template"
	"time"

	"github.com/martinp/lftpq/parser"
)

func newTestItem(remotePath string, itemParser itemParser) Item {
	item, _ := newItem(remotePath, time.Time{}, itemParser)
	return item
}

func showItemParser() itemParser {
	return itemParser{
		parser:   parser.Show,
		template: template.Must(template.New("t").Parse("/tmp/{{ .Name }}/S{{ .Season }}/")),
	}
}

func movieItemParser() itemParser {
	return itemParser{
		parser:   parser.Movie,
		template: template.Must(template.New("t").Parse("/tmp/{{ .Year }}/{{ .Name }}/")),
	}
}

func TestNewItemShow(t *testing.T) {
	item := newTestItem("/foo/The.Wire.S03E01", showItemParser())
	if expected := "/tmp/The.Wire/S3/"; item.LocalDir != expected {
		t.Fatalf("Expected %q, got %q", expected, item.LocalDir)
	}
}

func TestNewItemMovie(t *testing.T) {
	item := newTestItem("/foo/Apocalypse.Now.1979", movieItemParser())
	if expected := "/tmp/1979/Apocalypse.Now/"; item.LocalDir != expected {
		t.Fatalf("Expected %q, got %q", expected, item.LocalDir)
	}
}

func TestNewItemDefaultParser(t *testing.T) {
	tmpl := itemParser{parser: parser.Default, template: template.Must(template.New("t").Parse("/tmp/"))}
	item := newTestItem("/foo/The.Wire.S03E01", tmpl)
	if expected := "/tmp/"; item.LocalDir != expected {
		t.Fatalf("Expected %s, got %s", expected, item.LocalDir)
	}
}

func TestNewItemUnparsable(t *testing.T) {
	_, err := newItem("/foo/bar", time.Time{}, showItemParser())
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestNewItemWithReplacements(t *testing.T) {
	tmpl := showItemParser()
	tmpl.replacements = []Replacement{
		{pattern: regexp.MustCompile("_"), Replacement: "."},
		{pattern: regexp.MustCompile(`\.Of\.`), Replacement: ".of."},
		{pattern: regexp.MustCompile(`\.the\.`), Replacement: ".The."},
		{pattern: regexp.MustCompile(`\.And\.`), Replacement: ".and."},
	}
	var tests = []struct {
		in  Item
		out string
	}{
		{newTestItem("/foo/Game.Of.Thrones.S01E01", tmpl), "Game.of.Thrones"},
		{newTestItem("/foo/Fear.the.Walking.Dead.S01E01", tmpl), "Fear.The.Walking.Dead"},
		{newTestItem("/foo/Halt.And.Catch.Fire.S01E01", tmpl), "Halt.and.Catch.Fire"},
		{newTestItem("/foo/Top_Gear.01x01", tmpl), "Top.Gear"},
	}
	for _, tt := range tests {
		if tt.in.Media.Name != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, tt.in.Media.Name)
		}
	}
}

func TestItemsSort(t *testing.T) {
	items := Items{
		Item{RemotePath: "/x/c"},
		Item{RemotePath: "/x/b"},
		Item{RemotePath: "/x/a"},
		Item{RemotePath: "/y/a"},
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
		if got := items[tt.in].RemotePath; got != tt.out {
			t.Errorf("Expected index %d to be %q, got %q", tt.in, tt.out, got)
		}
	}
}

func TestAccept(t *testing.T) {
	item := Item{}
	item.accept("foo")
	if !item.Transfer {
		t.Error("Expected true")
	}
	if expected := "foo"; item.Reason != expected {
		t.Errorf("Expected %q, got %q", expected, item.Reason)
	}
}

func TestReject(t *testing.T) {
	item := Item{}
	item.reject("bar")
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
		{Item{RemotePath: "/foo/bar", LocalDir: "/tmp/"}, "/tmp/bar"},
		{Item{RemotePath: "/foo/bar", LocalDir: "/tmp/foo/bar"}, "/tmp/foo/bar"},
	}
	for _, tt := range tests {
		if got := tt.in.dstDir(); got != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, got)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	readDir := func(dirname string) ([]os.FileInfo, error) {
		if dirname == "/tmp/bar" {
			return []os.FileInfo{file{}}, nil
		}
		return nil, nil
	}
	var tests = []struct {
		in  Item
		out bool
	}{
		{Item{LocalDir: "/tmp/foo"}, true},
		{Item{LocalDir: "/tmp/bar"}, false},
	}
	for _, tt := range tests {
		if got := tt.in.isEmpty(readDir); got != tt.out {
			t.Errorf("Expected %t, got %t", tt.out, got)
		}
	}
}
