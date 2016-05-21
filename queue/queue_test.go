package queue

import (
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
)

func readDirStub(dirname string) ([]os.FileInfo, error) { return nil, nil }

type fileInfoStub struct{ name string }

func (f fileInfoStub) Name() string       { return f.name }
func (f fileInfoStub) Size() int64        { return 0 }
func (f fileInfoStub) Mode() os.FileMode  { return 0 }
func (f fileInfoStub) ModTime() time.Time { return time.Time{} }
func (f fileInfoStub) IsDir() bool        { return false }
func (f fileInfoStub) Sys() interface{}   { return nil }

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
	files := []lftp.File{
		lftp.File{
			Path:    "/tmp/dir1@",
			Created: now,
			// Filtered because of symlink
			FileMode: os.ModeSymlink,
		},
		lftp.File{
			Path: "/tmp/dir2/",
			// Filtered because of exceeded MaxAge
			Created:  now.Add(-time.Duration(48) * time.Hour),
			FileMode: os.ModeDir,
		},
		lftp.File{
			Path: "/tmp/dir3/",
			// Included because of equal MaxAge
			Created:  now.Add(-time.Duration(24) * time.Hour),
			FileMode: os.ModeDir,
		},
		lftp.File{
			Path: "/tmp/dir4/",
			// Included because less than MaxAge
			Created:  now,
			FileMode: os.ModeDir,
		},
		lftp.File{
			Path: "/tmp/dir5/",
			// Filtered because it already exists
			Created:  now,
			FileMode: os.ModeDir,
		},
		lftp.File{
			Path: "/tmp/foo/",
			// Filtered because of not matching any Patterns
			Created:  now,
			FileMode: os.ModeDir,
		},
		lftp.File{
			Path: "/tmp/incomplete-dir3/",
			// Filtered because of matching any Filters
			Created:  now,
			FileMode: os.ModeDir,
		},
		lftp.File{
			Path: "/tmp/xfile",
			// Filtered because it is not a directory
			Created:  now,
			FileMode: os.FileMode(0),
		},
	}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		if dirname == "/data/dir5" {
			return []os.FileInfo{fileInfoStub{}}, nil
		}
		return nil, nil
	}
	q := newQueue(s, files, readDir)
	expected := []Item{
		Item{Queue: &q, Remote: files[0], Transfer: false, Reason: "IsSymlink=true SkipSymlinks=true"},
		Item{Queue: &q, Remote: files[1], Transfer: false, Reason: "Age=48h0m0s MaxAge=24h0m0s"},
		Item{Queue: &q, Remote: files[2], Transfer: true, Reason: "Match=dir\\d"},
		Item{Queue: &q, Remote: files[3], Transfer: true, Reason: "Match=dir\\d"},
		Item{Queue: &q, Remote: files[4], Transfer: false, Reason: "IsDstDirEmpty=false"},
		Item{Queue: &q, Remote: files[5], Transfer: false, Reason: "no match"},
		Item{Queue: &q, Remote: files[6], Transfer: false, Reason: "Filter=^incomplete-"},
		Item{Queue: &q, Remote: files[7], Transfer: false, Reason: "IsFile=true SkipFiles=true"},
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
				e.Remote.Path, e.Transfer, a.Transfer)
		}
		if a.Reason != e.Reason {
			t.Errorf("Expected Dir=%s to have Reason=%s, got Reason=%s", e.Remote.Path,
				e.Reason, a.Reason)
		}
	}
}

func TestNewQueueRejectsUnparsableItem(t *testing.T) {
	s := Site{parser: parser.Show}
	q := newQueue(s, []lftp.File{lftp.File{Path: "/foo/bar"}}, readDirStub)
	if got := q.Items[0].Transfer; got {
		t.Errorf("Expected false, got %t", got)
	}
	want := "failed to parse: bar"
	if got := q.Items[0].Reason; got != want {
		t.Errorf("Expected %q, got %q", want, got)
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
		Item{Remote: lftp.File{Path: "/foo"}, LocalDir: "/tmp", Transfer: true},
		Item{Remote: lftp.File{Path: "/bar"}, LocalDir: "/tmp", Transfer: true},
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
			Item{Remote: lftp.File{Path: "/tmp/d1"}, Transfer: true},
			Item{Remote: lftp.File{Path: "/tmp/d2"}, Transfer: false},
			Item{Remote: lftp.File{Path: "/tmp/d2"}, Transfer: false},
		},
	}
	actual := q.Transferable()
	expected := "/tmp/d1"
	if len(actual) != 1 {
		t.Fatal("Expected length to be 1")
	}
	if got := actual[0].Remote.Path; got != expected {
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
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E01.foo"}),
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E01.PROPER.foo"}), /* keep */
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E01.REPACK.foo"}),
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E02.bar"}),
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E02.PROPER.REPACK"}), /* keep */
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E03.bar"}),           /* keep */
		newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E03.PROPER.REPACK"}),
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
		if actual[i].Remote.Path != expected[i].Remote.Path {
			t.Errorf("Expected %s, got %s", expected[i].Remote.Path, actual[i].Remote.Path)
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
	files := []lftp.File{
		lftp.File{Path: "/tmp/The.Wire.S01E01.HDTV.foo", Created: now.Add(-time.Duration(48) * time.Hour)},
		lftp.File{Path: "/tmp/The.Wire.S01E01.WEBRip.foo", Created: now},
	}
	q := newQueue(s, files, readDirStub)
	for _, item := range q.Transferable() {
		t.Errorf("Expected empty queue, got %s", item.Remote.Path)
	}
}

func TestDeduplicateIgnoreSelf(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := Site{
		parser:      parser.Show,
		patterns:    []*regexp.Regexp{regexp.MustCompile(".*")},
		maxAge:      time.Duration(24) * time.Hour,
		localDir:    template.Must(template.New("").Parse("/tmp/")),
		Deduplicate: true,
	}
	files := []lftp.File{
		lftp.File{Path: "/tmp/The.Wire.S01E01", Created: now},
		lftp.File{Path: "/tmp/The.Wire.S01E01", Created: now},
	}
	q := newQueue(s, files, readDirStub)
	for _, item := range q.Items {
		if item.Duplicate {
			t.Errorf("Expected Duplicate=false for %+v", item)
		}
	}
}

func TestPostCommand(t *testing.T) {
	s := Site{
		parser:      parser.Default,
		PostCommand: "xargs echo",
	}
	q := Queue{Site: s}
	q.Items = []Item{newTestItem(&q, lftp.File{Path: "/tmp/foo"})}
	cmd, err := q.PostCommand(false)
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

func TestMerge(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir:   tmpl,
		parser:     parser.Show,
		priorities: []*regexp.Regexp{regexp.MustCompile("\\.foo$")},
	}
	q := Queue{Site: s}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			fileInfoStub{name: "The.Wire.S01E01.720p.BluRay.bar"},
			fileInfoStub{name: "The.Wire.S01E01.720p.BluRay.baz"},
		}, nil
	}
	item := newTestItem(&q, lftp.File{Path: "/tmp/The.Wire.S01E01.foo"})
	item.Transfer = true
	q.Items = []Item{item}
	q.merge(readDir)
	q.deduplicate()
	if l := len(q.Items); l != 3 {
		t.Fatalf("Expected length 3, got %d", l)
	}
	for _, i := range q.Items[1:] {
		if !i.Duplicate {
			t.Errorf("Expected Duplicate=true for Path=%q", i.Remote.Path)
		}
	}
}

func TestReadQueue(t *testing.T) {
	json := `
/tv/The.Wire.S01E01

/tv/The.Wire.S01E02

  /tv/The.Wire.S01E03
`
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{localDir: tmpl, parser: parser.Show}

	q, err := Read(s, strings.NewReader(json))
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Items) != 3 {
		t.Fatal("Expected 3 items")
	}
	for i, _ := range q.Items {
		if want := "The.Wire"; q.Items[i].Media.Name != want {
			t.Errorf("Expected Items[%d].Media.Name=%q, want %q", i, q.Items[0].Media.Name, want)
		}
		if !q.Items[i].Transfer {
			t.Errorf("Expected Items[%d].Transfer=true", i)
		}
	}
}
