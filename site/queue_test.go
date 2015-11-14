package site

import (
	"io/ioutil"
	"reflect"
	"regexp"
	"testing"
	"text/template"
	"time"

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
)

func isEmptyDirStub(dirname string) bool { return true }

func TestNewQueue(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := Site{
		Name:         "foo",
		Dir:          "/misc",
		maxAge:       time.Duration(24) * time.Hour,
		patterns:     []*regexp.Regexp{regexp.MustCompile("dir\\d")},
		filters:      []*regexp.Regexp{regexp.MustCompile("^incomplete-")},
		SkipSymlinks: true,
		SkipExisting: true,
		SkipFiles:    true,
		localDir:     template.Must(template.New("").Parse("/data/")),
		parser:       parser.Default,
	}
	dirs := []lftp.Dir{
		lftp.Dir{
			Path:    "/tmp/dir1@",
			Created: now,
			// Filtered because of symlink
			IsSymlink: true,
		},
		lftp.Dir{
			Path: "/tmp/dir2/",
			// Filtered because of exceeded MaxAge
			Created: now.Add(-time.Duration(48) * time.Hour),
		},
		lftp.Dir{
			Path: "/tmp/dir3/",
			// Included because of equal MaxAge
			Created: now.Add(-time.Duration(24) * time.Hour),
		},
		lftp.Dir{
			Path: "/tmp/dir4/",
			// Included because less than MaxAge
			Created: now,
		},
		lftp.Dir{
			Path: "/tmp/dir5/",
			// Filtered because it already exists
			Created: now,
		},
		lftp.Dir{
			Path: "/tmp/foo/",
			// Filtered because of not matching any Patterns
			Created: now,
		},
		lftp.Dir{
			Path: "/tmp/incomplete-dir3/",
			// Filtered because of matching any Filters
			Created: now,
		},
		lftp.Dir{
			Path: "/tmp/xfile",
			// Filtered because it is not a directory
			Created: now,
			IsFile:  true,
		},
	}
	q := newQueue(s, dirs, func(dirname string) bool { return dirname != "/data/dir5" })
	expected := []Item{
		Item{Queue: &q, Dir: dirs[0], Transfer: false, Reason: "IsSymlink=true SkipSymlinks=true"},
		Item{Queue: &q, Dir: dirs[1], Transfer: false, Reason: "Age=48h0m0s MaxAge=24h0m0s"},
		Item{Queue: &q, Dir: dirs[2], Transfer: true, Reason: "Match=dir\\d"},
		Item{Queue: &q, Dir: dirs[3], Transfer: true, Reason: "Match=dir\\d"},
		Item{Queue: &q, Dir: dirs[4], Transfer: false, Reason: "IsDstDirEmpty=false"},
		Item{Queue: &q, Dir: dirs[5], Transfer: false, Reason: "no match"},
		Item{Queue: &q, Dir: dirs[6], Transfer: false, Reason: "Filter=^incomplete-"},
		Item{Queue: &q, Dir: dirs[7], Transfer: false, Reason: "IsFile=true SkipFiles=true"},
	}
	actual := q.Items
	if len(expected) != len(actual) {
		t.Fatalf("Expected length=%d, got length=%d", len(expected), len(actual))
	}
	for i, _ := range expected {
		e := expected[i]
		a := actual[i]
		if a.Transfer != e.Transfer {
			t.Errorf("Expected Dir=%s to have Transfer=%t, got Transfer=%t",
				e.Dir.Path, e.Transfer, a.Transfer)
		}
		if a.Reason != e.Reason {
			t.Errorf("Expected Dir=%s to have Reason=%s, got Reason=%s", e.Dir.Path,
				e.Reason, a.Reason)
		}
	}
}

func TestScript(t *testing.T) {
	s := Site{
		Client: lftp.Client{
			Path:   "/bin/lftp",
			GetCmd: "mirror",
		},
		Name: "siteA",
	}
	items := []Item{
		Item{Dir: lftp.Dir{Path: "/foo"}, LocalDir: "/tmp", Transfer: true},
		Item{Dir: lftp.Dir{Path: "/bar"}, LocalDir: "/tmp", Transfer: true},
	}
	q := Queue{Site: s, Items: items}
	script := q.Script()
	expected := `open siteA
queue mirror /foo /tmp
queue mirror /bar /tmp
queue start
wait
exit
`
	if script != expected {
		t.Fatalf("Expected %q, got %q", expected, script)
	}
}

func TestTransferable(t *testing.T) {
	q := Queue{
		Items: []Item{
			Item{Dir: lftp.Dir{Path: "/tmp/d1"}, Transfer: true},
			Item{Dir: lftp.Dir{Path: "/tmp/d2"}, Transfer: false},
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
		parser: parser.Show,
		priorities: []*regexp.Regexp{
			regexp.MustCompile("\\.PROPER\\.REPACK\\."),
			regexp.MustCompile("\\.PROPER\\."),
			regexp.MustCompile("\\.REPACK\\."),
		},
	}
	q := Queue{Site: s}
	q.Items = []Item{
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E01.foo"}),
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E01.PROPER.foo"}), /* keep */
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E01.REPACK.foo"}),
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E02.bar"}),
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E02.PROPER.REPACK"}), /* keep */
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E03.bar"}),           /* keep */
		newItem(&q, lftp.Dir{Path: "/tmp/The.Wire.S01E03.PROPER.REPACK"}),
	}
	// Accept all but the last item
	for i, _ := range q.Items[:len(q.Items)-1] {
		q.Items[i].Accept("")
	}
	q.deduplicate()

	expected := []Item{q.Items[1], q.Items[4], q.Items[5]}
	actual := q.Transferable()
	if len(expected) != len(actual) {
		t.Fatalf("Expected length %d, got %d", len(expected), len(actual))
	}
	for i, _ := range actual {
		if actual[i].Path != expected[i].Path {
			t.Errorf("Expected %s, got %s", expected[i].Path, actual[i].Path)
		}
	}
}

func TestDeduplicateIgnoresAge(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := Site{
		parser:      parser.Show,
		priorities:  []*regexp.Regexp{regexp.MustCompile("\\.HDTV\\.")},
		patterns:    []*regexp.Regexp{regexp.MustCompile(".*")},
		maxAge:      time.Duration(24) * time.Hour,
		localDir:    template.Must(template.New("").Parse("/tmp/")),
		Deduplicate: true,
	}
	dirs := []lftp.Dir{
		lftp.Dir{Path: "/tmp/The.Wire.S01E01.HDTV.foo", Created: now.Add(-time.Duration(48) * time.Hour)},
		lftp.Dir{Path: "/tmp/The.Wire.S01E01.WEBRip.foo", Created: now},
	}
	q := newQueue(s, dirs, isEmptyDirStub)
	for _, item := range q.Transferable() {
		t.Errorf("Expected empty queue, got %s", item.Path)
	}
}

func TestPostCommand(t *testing.T) {
	s := Site{
		parser:      parser.Default,
		PostCommand: "xargs echo",
	}
	q := Queue{Site: s}
	q.Items = []Item{newItem(&q, lftp.Dir{Path: "/tmp/foo"})}
	cmd, err := q.PostCommand()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"xargs", "echo"}; !reflect.DeepEqual(want, cmd.Args) {
		t.Fatalf("Expected %+v, got %+v", want, cmd.Args)
	}
	data, err := ioutil.ReadAll(cmd.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("Expected stdin to contain data")
	}
}
