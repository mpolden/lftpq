package queue

import (
	"bytes"
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

var testTemplate = template.Must(template.New("localDir").Parse("/tmp/{{ .Name }}/S{{ .Season }}/"))

func newTestSite() Site {
	localDir := template.Must(template.New("localDir").Parse("/tmp/{{ .Name }}/S{{ .Season }}/"))
	patterns := []*regexp.Regexp{regexp.MustCompile(".*")}
	return Site{
		Name:     "test",
		localDir: localDir,
		patterns: patterns,
		parser:   parser.Show,
	}
}

func newTestQueue(s Site, files []lftp.File) Queue {
	return newQueue(s, files, readDirStub)
}

func newTestItem(q *Queue, remotePath string) Item {
	item, _ := newItem(q, lftp.File{Path: remotePath})
	return item
}

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
		Dir:          "/remote",
		maxAge:       time.Duration(24) * time.Hour,
		patterns:     []*regexp.Regexp{regexp.MustCompile("dir\\d")},
		filters:      []*regexp.Regexp{regexp.MustCompile("^incomplete-")},
		SkipSymlinks: true,
		SkipExisting: true,
		SkipFiles:    true,
		localDir:     template.Must(template.New("localDir").Parse("/tmp/")),
		parser:       parser.Default,
	}
	files := []lftp.File{
		{
			Path:     "/remote/dir1@",
			Modified: now,
			// Filtered because of symlink
			FileMode: os.ModeSymlink,
		},
		{
			Path: "/remote/dir2/",
			// Filtered because of exceeded MaxAge
			Modified: now.Add(-time.Duration(48) * time.Hour),
			FileMode: os.ModeDir,
		},
		{
			Path: "/remote/dir3/",
			// Included because of equal MaxAge
			Modified: now.Add(-time.Duration(24) * time.Hour),
			FileMode: os.ModeDir,
		},
		{
			Path: "/remote/dir4/",
			// Included because less than MaxAge
			Modified: now,
			FileMode: os.ModeDir,
		},
		{
			Path: "/remote/dir5/",
			// Filtered because it already exists
			Modified: now,
			FileMode: os.ModeDir,
		},
		{
			Path: "/remote/foo/",
			// Filtered because of not matching any Patterns
			Modified: now,
			FileMode: os.ModeDir,
		},
		{
			Path: "/remote/incomplete-dir3/",
			// Filtered because of matching any Filters
			Modified: now,
			FileMode: os.ModeDir,
		},
		{
			Path: "/remote/xfile",
			// Filtered because it is not a directory
			Modified: now,
			FileMode: os.FileMode(0),
		},
	}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		if dirname == "/tmp/dir5" {
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
	s := newTestSite()
	q := newTestQueue(s, []lftp.File{{Path: "/foo/bar"}})
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

func TestDeduplicate(t *testing.T) {
	s := newTestSite()
	s.Deduplicate = true
	s.priorities = []*regexp.Regexp{
		regexp.MustCompile("\\.PROPER\\.REPACK\\."),
		regexp.MustCompile("\\.PROPER\\."),
		regexp.MustCompile("\\.REPACK\\."),
	}
	files := []lftp.File{
		{Path: "/tmp/The.Wire.S01E01.PROPER.foo"}, /* keep */
		{Path: "/tmp/The.Wire.S01E01.REPACK.foo"},
		{Path: "/tmp/The.Wire.S01E01.foo"},
		{Path: "/tmp/The.Wire.S01E02.PROPER.REPACK"}, /* keep */
		{Path: "/tmp/The.Wire.S01E02.bar"},
		{Path: "/tmp/The.Wire.S01E03.PROPER.REPACK"},
		{Path: "/tmp/The.Wire.S01E03.bar"}, /* keep */
	}
	q := newTestQueue(s, files)
	expected := []Item{q.Items[0], q.Items[3], q.Items[5]}
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
	s := newTestSite()
	s.priorities = []*regexp.Regexp{regexp.MustCompile("\\.HDTV\\.")}
	s.maxAge = time.Duration(24) * time.Hour
	s.Deduplicate = true
	files := []lftp.File{
		{Path: "/tmp/The.Wire.S01E01.HDTV.foo", Modified: now.Add(-time.Duration(48) * time.Hour)},
		{Path: "/tmp/The.Wire.S01E01.WEBRip.foo", Modified: now},
	}
	q := newTestQueue(s, files)
	for _, item := range q.Transferable() {
		t.Errorf("Expected empty queue, got %s", item.Remote.Path)
	}
}

func TestDeduplicateIgnoreSelf(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := newTestSite()
	s.Deduplicate = true
	files := []lftp.File{
		{Path: "/tmp/The.Wire.S01E01", Modified: now},
		{Path: "/tmp/The.Wire.S01E01", Modified: now},
	}
	q := newQueue(s, files, readDirStub)
	for _, item := range q.Items {
		if item.Duplicate {
			t.Errorf("Expected Duplicate=false for %+v", item)
		}
	}
}

func TestPostCommand(t *testing.T) {
	s := newTestSite()
	s.PostCommand = "xargs echo"
	q := newTestQueue(s, []lftp.File{{Path: "/tmp/foo"}})
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
	s := newTestSite()
	s.Merge = true
	s.Deduplicate = true
	s.priorities = []*regexp.Regexp{regexp.MustCompile("\\.foo$")}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			fileInfoStub{name: "The.Wire.S01E01.720p.BluRay.bar"},
			fileInfoStub{name: "The.Wire.S01E01.720p.BluRay.baz"},
		}, nil
	}
	q := newQueue(s, []lftp.File{{Path: "/tmp/The.Wire.S01E01.foo"}}, readDir)
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
	s := newTestSite()
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

func TestFprintln(t *testing.T) {
	s := newTestSite()
	s.Client = lftp.Client{GetCmd: "mirror"}
	q := newTestQueue(s, []lftp.File{{Path: "/tmp/The.Wire.S01E01"}})

	var buf bytes.Buffer
	if err := q.Fprintln(&buf, true); err != nil {
		t.Fatal(err)
	}

	json := `[
  {
    "Remote": {
      "Modified": "0001-01-01T00:00:00Z",
      "Path": "/tmp/The.Wire.S01E01",
      "FileMode": 0
    },
    "LocalDir": "/tmp/The.Wire/S1/",
    "Transfer": true,
    "Reason": "Match=.*",
    "Media": {
      "Release": "The.Wire.S01E01",
      "Name": "The.Wire",
      "Year": 0,
      "Season": 1,
      "Episode": 1
    },
    "Duplicate": false,
    "Merged": false
  }
]
`
	if got := buf.String(); got != json {
		t.Errorf("Expected %q, got %q", json, got)
	}

	buf.Reset()
	if err := q.Fprintln(&buf, false); err != nil {
		t.Fatal(err)
	}

	script := `open test
queue mirror /tmp/The.Wire.S01E01 /tmp/The.Wire/S1/
queue start
wait
exit
`
	if got := buf.String(); got != script {
		t.Errorf("Expected %q, got %q", script, got)
	}
}
