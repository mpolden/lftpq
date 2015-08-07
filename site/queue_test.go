package site

import (
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/martinp/lftpq/parser"
)

func TestNewQueue(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := Site{
		Name:         "foo",
		Dir:          "/misc",
		maxAge:       time.Duration(24) * time.Hour,
		patterns:     []*regexp.Regexp{regexp.MustCompile("dir\\d")},
		filters:      []*regexp.Regexp{regexp.MustCompile("^incomplete-")},
		SkipSymlinks: true,
	}
	dirs := []Dir{
		Dir{
			Path:    "/tmp/dir1@",
			Created: now,
			// Filtered because of symlink
			IsSymlink: true,
		},
		Dir{
			Path: "/tmp/dir2",
			// Filtered because of exceeded MaxAge
			Created: now.Add(-time.Duration(48) * time.Hour),
		},
		Dir{
			Path: "/tmp/dir3",
			// Included because of equal MaxAge
			Created: now.Add(-time.Duration(24) * time.Hour),
		},
		Dir{
			Path: "/tmp/dir4",
			// Included because less than MaxAge
			Created: now,
		},
		Dir{
			Path: "/tmp/foo",
			// Filtered because of not matching any Patterns
			Created: now,
		},
		Dir{
			Path: "/tmp/incomplete-dir3",
			// Filtered because of matching any Filters
			Created: now,
		},
	}
	q := NewQueue(s, dirs)
	expected := []Item{
		Item{Queue: &q, Dir: dirs[0], Transfer: false, Reason: "IsSymlink=true SkipSymlinks=true"},
		Item{Queue: &q, Dir: dirs[1], Transfer: false, Reason: "Age=48h0m0s MaxAge="},
		Item{Queue: &q, Dir: dirs[2], Transfer: true, Reason: "Match=dir\\d"},
		Item{Queue: &q, Dir: dirs[3], Transfer: true, Reason: "Match=dir\\d"},
		Item{Queue: &q, Dir: dirs[4], Transfer: false, Reason: "no match"},
		Item{Queue: &q, Dir: dirs[5], Transfer: false, Reason: "Filter=^incomplete-"},
	}
	actual := q.Items
	if len(expected) != len(actual) {
		t.Fatal("Expected equal length")
	}
	for i, _ := range expected {
		if !reflect.DeepEqual(expected[i], actual[i]) {
			t.Fatalf("Expected %+v, got %+v", expected[i], actual[i])
		}
	}
}

func TestWrite(t *testing.T) {
	s := Site{
		Client: Client{
			LftpPath:   "/bin/lftp",
			LftpGetCmd: "mirror",
		},
		Name: "siteA",
	}
	items := []Item{
		Item{Dir: Dir{Path: "/foo"}, LocalDir: "/tmp", Transfer: true},
		Item{Dir: Dir{Path: "/bar"}, LocalDir: "/tmp", Transfer: true},
	}
	q := Queue{Site: s, Items: items}
	name, err := q.Write()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(name)
	expected := `open siteA
queue mirror /foo /tmp
queue mirror /bar /tmp
queue start
wait
exit
`
	f, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	content := string(f)
	if content != expected {
		t.Fatalf("Expected %q, got %q", expected, content)
	}
}

func TestTransferable(t *testing.T) {
	q := Queue{
		Items: []Item{
			Item{Dir: Dir{Path: "/tmp/d1"}, Transfer: true},
			Item{Dir: Dir{Path: "/tmp/d2"}, Transfer: false},
		},
	}
	actual := q.Transferable()
	expected := "/tmp/d1"
	if len(actual) != 1 {
		t.Fatal("Expected length to be 1")
	}
	if got := actual[0].Path; got != expected {
		t.Fatalf("Expected %s, got %s", expected, got)
	}
}

func TestDeduplicate(t *testing.T) {
	s := Site{
		priorities: []*regexp.Regexp{
			regexp.MustCompile("\\.PROPER\\.REPACK\\."),
			regexp.MustCompile("\\.PROPER\\."),
			regexp.MustCompile("\\.REPACK\\."),
		},
	}
	dirs := []Dir{
		Dir{Path: "/tmp/The.Wire.S01E01.foo"},
		Dir{Path: "/tmp/The.Wire.S01E01.PROPER.foo"},
		Dir{Path: "/tmp/The.Wire.S01E01.REPACK.foo"},
		Dir{Path: "/tmp/The.Wire.S01E02.bar"},
		Dir{Path: "/tmp/The.Wire.S01E02.PROPER.REPACK"},
		Dir{Path: "/tmp/The.Wire.S01E03.bar"},
		Dir{Path: "/tmp/The.Wire.S01E03.PROPER.REPACK"},
	}
	q := Queue{Site: s}
	q.Items = []Item{
		Item{Queue: &q, Dir: dirs[0], Transfer: true, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "01"}},
		Item{Queue: &q, Dir: dirs[1], Transfer: true, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "01"}},
		Item{Queue: &q, Dir: dirs[2], Transfer: true, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "01"}},
		Item{Queue: &q, Dir: dirs[3], Transfer: true, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "02"}},
		Item{Queue: &q, Dir: dirs[4], Transfer: true, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "02"}},
		Item{Queue: &q, Dir: dirs[5], Transfer: true, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "03"}},
		Item{Queue: &q, Dir: dirs[6], Transfer: false, Media: parser.Show{Name: "The.Wire", Season: "01", Episode: "03"}},
	}
	q.deduplicate()

	expected := []Item{q.Items[1], q.Items[4], q.Items[5]}
	actual := q.Transferable()
	if len(expected) != len(actual) {
		t.Fatal("Expected equal length")
	}
	for i, _ := range actual {
		if actual[i].Path != expected[i].Path {
			t.Errorf("Expected %s, got %s", expected[i].Path, actual[i].Path)
		}
	}
}
