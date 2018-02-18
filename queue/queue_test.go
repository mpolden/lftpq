package queue

import (
	"encoding"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/mpolden/lftpq/parser"
)

func newTestSite() Site {
	localDir := template.Must(template.New("t").Parse("/local/{{ .Name }}/S{{ .Season }}/"))
	patterns := []*regexp.Regexp{regexp.MustCompile(".*")}
	return Site{
		GetCmd:   "mirror",
		Name:     "test",
		patterns: patterns,
		itemParser: itemParser{
			template: localDir,
			parser:   parser.Show,
		},
	}
}

func newTestQueue(s Site, files []os.FileInfo) Queue {
	return newQueue(s, files, func(dirname string) ([]os.FileInfo, error) { return nil, nil })
}

type file struct {
	name    string
	modTime time.Time
	mode    os.FileMode
}

func (f file) Name() string       { return f.name }
func (f file) Size() int64        { return 0 }
func (f file) Mode() os.FileMode  { return f.mode }
func (f file) ModTime() time.Time { return f.modTime }
func (f file) IsDir() bool        { return f.Mode().IsDir() }
func (f file) Sys() interface{}   { return nil }

func TestNewQueue(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := Site{
		Name:         "foo",
		Dirs:         []string{"/remote"},
		maxAge:       time.Duration(24) * time.Hour,
		patterns:     []*regexp.Regexp{regexp.MustCompile(`dir\d`)},
		filters:      []*regexp.Regexp{regexp.MustCompile("^incomplete-")},
		SkipSymlinks: true,
		SkipExisting: true,
		SkipFiles:    true,
		itemParser: itemParser{
			template: template.Must(template.New("localDir").Parse("/local/")),
			parser:   parser.Default,
		},
	}
	files := []os.FileInfo{
		file{
			name:    "/remote/dir1@",
			modTime: now,
			// Filtered because of symlink
			mode: os.ModeSymlink,
		},
		file{
			name: "/remote/dir2/",
			// Filtered because of exceeded MaxAge
			modTime: now.Add(-time.Duration(48) * time.Hour),
			mode:    os.ModeDir,
		},
		file{
			name: "/remote/dir3/",
			// Included because of equal MaxAge
			modTime: now.Add(-time.Duration(24) * time.Hour),
			mode:    os.ModeDir,
		},
		file{
			name: "/remote/dir4/",
			// Included because less than MaxAge
			modTime: now,
			mode:    os.ModeDir,
		},
		file{
			name: "/remote/dir5/",
			// Filtered because it already exists
			modTime: now,
			mode:    os.ModeDir,
		},
		file{
			name: "/remote/foo/",
			// Filtered because of not matching any Patterns
			modTime: now,
			mode:    os.ModeDir,
		},
		file{
			name: "/remote/incomplete-dir3/",
			// Filtered because of matching any Filters
			modTime: now,
			mode:    os.ModeDir,
		},
		file{
			name: "/remote/xfile",
			// Filtered because it is not a directory
			modTime: now,
		},
	}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		if dirname == "/local/dir5" {
			return []os.FileInfo{file{}}, nil
		}
		return nil, nil
	}
	q := newQueue(s, files, readDir)
	expected := []Item{
		Item{itemParser: q.itemParser, RemotePath: files[0].Name(), Transfer: false, Reason: "IsSymlink=true SkipSymlinks=true"},
		Item{itemParser: q.itemParser, RemotePath: files[1].Name(), Transfer: false, Reason: "Age=48h0m0s MaxAge=24h0m0s"},
		Item{itemParser: q.itemParser, RemotePath: files[2].Name(), Transfer: true, Reason: "Match=dir\\d"},
		Item{itemParser: q.itemParser, RemotePath: files[3].Name(), Transfer: true, Reason: "Match=dir\\d"},
		Item{itemParser: q.itemParser, RemotePath: files[4].Name(), Transfer: false, Reason: "IsDstDirEmpty=false"},
		Item{itemParser: q.itemParser, RemotePath: files[5].Name(), Transfer: false, Reason: "no match"},
		Item{itemParser: q.itemParser, RemotePath: files[6].Name(), Transfer: false, Reason: "Filter=^incomplete-"},
		Item{itemParser: q.itemParser, RemotePath: files[7].Name(), Transfer: false, Reason: "IsFile=true SkipFiles=true"},
	}
	actual := q.Items
	if len(expected) != len(actual) {
		t.Fatalf("Expected length=%d, got length=%d", len(expected), len(actual))
	}
	for i := range expected {
		e := expected[i]
		a := actual[i]
		if a.Transfer != e.Transfer {
			t.Errorf("Expected Dir=%s to have Transfer=%t, got Transfer=%t",
				e.RemotePath, e.Transfer, a.Transfer)
		}
		if a.Reason != e.Reason {
			t.Errorf("Expected Dir=%s to have Reason=%s, got Reason=%s", e.RemotePath,
				e.Reason, a.Reason)
		}
	}
}

func TestNewQueueRejectsUnparsableItem(t *testing.T) {
	s := newTestSite()
	q := newTestQueue(s, []os.FileInfo{file{name: "/foo/bar"}})
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
		GetCmd: "mirror",
		Name:   "siteA",
	}
	items := []Item{
		Item{RemotePath: "/remote/foo", LocalPath: "/local", Transfer: true},
		Item{RemotePath: "/remote/bar", LocalPath: "/local", Transfer: true},
	}
	q := Queue{Site: s, Items: items}
	script := q.Script()
	expected := `open siteA
queue mirror /remote/foo /local
queue mirror /remote/bar /local
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
	s.priorities = []*regexp.Regexp{
		regexp.MustCompile(`\.PROPER\.REPACK\.`),
		regexp.MustCompile(`\.PROPER\.`),
		regexp.MustCompile(`\.REPACK\.`),
	}
	files := []os.FileInfo{
		file{name: "/remote/The.Wire.S01E01.PROPER.foo"}, /* keep */
		file{name: "/remote/The.Wire.S01E01.REPACK.foo"},
		file{name: "/remote/The.Wire.S01E01.foo"},
		file{name: "/remote/The.Wire.S01E02.PROPER.REPACK"}, /* keep */
		file{name: "/remote/The.Wire.S01E02.bar"},
		file{name: "/remote/The.Wire.S01E03.PROPER.REPACK"},
		file{name: "/remote/The.Wire.S01E03.bar"}, /* keep */
	}
	q := newTestQueue(s, files)
	expected := []Item{q.Items[0], q.Items[3], q.Items[5]}
	actual := q.Transferable()
	if len(expected) != len(actual) {
		t.Fatalf("Expected length %d, got %d", len(expected), len(actual))
	}
	for i := range actual {
		if actual[i].RemotePath != expected[i].RemotePath {
			t.Errorf("Expected %s, got %s", expected[i].RemotePath, actual[i].RemotePath)
		}
	}
}

func TestDeduplicateIgnoreSelf(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := newTestSite()
	s.priorities = []*regexp.Regexp{regexp.MustCompile(`\.PROPER\.`)}
	files := []os.FileInfo{
		file{name: "/remote/The.Wire.S01E01", modTime: now},
		file{name: "/remote/The.Wire.S01E01", modTime: now},
	}
	q := newTestQueue(s, files)
	for _, item := range q.Items {
		if item.Duplicate {
			t.Errorf("Expected Duplicate=false for %+v", item)
		}
	}
}

func TestPostCommand(t *testing.T) {
	s := newTestSite()
	s.PostCommand = "xargs echo"
	q := newTestQueue(s, []os.FileInfo{file{name: "/remote/foo"}})
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

func TestMergePreferringRemoteCopy(t *testing.T) {
	s := newTestSite()
	s.Merge = true
	s.priorities = []*regexp.Regexp{regexp.MustCompile(`\.foo$`)}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			file{name: "The.Wire.S01E01.720p.BluRay.bar"},
			file{name: "The.Wire.S01E01.720p.BluRay.baz"},
		}, nil
	}
	q := newQueue(s, []os.FileInfo{file{name: "/remote/The.Wire.S01E01.foo"}}, readDir)
	if l := len(q.Items); l != 3 {
		t.Fatalf("Expected length 3, got %d", l)
	}
	if q.Items[2].Duplicate || q.Items[2].Merged {
		t.Errorf("Expected Duplicate=false Merged=false for Path=%q", q.Items[2].RemotePath)
	}
	for _, i := range q.Items[0 : len(q.Items)-1] {
		if !i.Duplicate || !i.Merged {
			t.Errorf("Expected Duplicate=true Merged=true for Path=%q", i.RemotePath)
		}
	}
}

func TestMergePreferringLocalCopy(t *testing.T) {
	s := newTestSite()
	s.Merge = true
	s.priorities = []*regexp.Regexp{regexp.MustCompile(`\.bar$`)}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			file{name: "The.Wire.S01E01.720p.BluRay.bar"},
			file{name: "The.Wire.S01E02.720p.BluRay.baz"},
		}, nil
	}
	q := newQueue(s, []os.FileInfo{file{name: "/remote/The.Wire.S01E01.foo"}}, readDir)
	if l := len(q.Items); l != 2 {
		t.Fatalf("Expected length 2, got %d", l)
	}
	if q.Items[0].Duplicate || !q.Items[0].Merged {
		t.Errorf("Expected Duplicate=false Merged=true for Path=%q", q.Items[0].RemotePath)
	}
	if !q.Items[1].Duplicate || q.Items[1].Merged {
		t.Errorf("Expected Duplicate=true Merged=false for Path=%q", q.Items[1].RemotePath)
	}
}

func TestLocalCopyDoesNotDuplicateRemoteWithEqualRank(t *testing.T) {
	s := newTestSite()
	s.Merge = true
	s.priorities = []*regexp.Regexp{regexp.MustCompile(`\.PROPER\.`)}
	s.SkipExisting = true
	readDir := func(dirname string) ([]os.FileInfo, error) {
		if dirname == "/local/The.Wire/S1/The.Wire.S01E01.720p.BluRay.foo" {
			return []os.FileInfo{file{}}, nil
		}
		if dirname == "/local/The.Wire/S1" {
			return []os.FileInfo{file{name: "The.Wire.S01E01.720p.BluRay.foo"}}, nil
		}
		return nil, nil
	}
	q := newQueue(s, []os.FileInfo{
		file{name: "/remote/The.Wire.S01E01.720p.BluRay.foo"},
		file{name: "/remote/The.Wire.S01E01.720p.BluRay.bar"},
	}, readDir)
	if item := q.Items[0]; item.Transfer || item.Duplicate || !item.Merged {
		t.Errorf("Expected Transfer=false Duplicate=false Merged=true for Path=%q", item.RemotePath)
	}
	for _, item := range q.Items[1:] {
		if item.Transfer {
			t.Errorf("Expected Transfer=false for Path=%q", item.RemotePath)
		}
	}
}

func TestLocalCopyWithTooOldReplacement(t *testing.T) {
	now := time.Now().Round(time.Second)
	s := newTestSite()
	s.priorities = []*regexp.Regexp{regexp.MustCompile(`\.HDTV\.`)}
	s.maxAge = time.Duration(24) * time.Hour
	s.SkipExisting = true
	readDir := func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{file{name: "The.Wire.S01E01.WEBRip.foo"}}, nil
	}
	files := []os.FileInfo{
		file{name: "/remote/The.Wire.S01E01.HDTV.foo", modTime: now.Add(-time.Duration(48) * time.Hour)},
		file{name: "/remote/The.Wire.S01E01.WEBRip.foo", modTime: now},
	}
	q := newQueue(s, files, readDir)
	for _, item := range q.Transferable() {
		t.Errorf("Expected empty queue, got %s", item.RemotePath)
	}
	for _, item := range q.Items {
		if item.Duplicate {
			t.Errorf("want Duplicate=false, got Duplicate=%t for %s", item.Duplicate, item.RemotePath)
		}
	}
}

func TestReadQueue(t *testing.T) {
	lines := `
t1 /tv/The.Wire.S01E01

t2 /tv/The.Wire.S01E04 ignored

ignored

  t1 /tv/The.Wire.S01E02

t1  /tv/The.Wire.S01E03

t2	/tv/The.Wire.S01E05
`
	s1, s2 := newTestSite(), newTestSite()
	s1.Name = "t1"
	s2.Name = "t2"
	queues, err := Read([]Site{s1, s2}, strings.NewReader(lines))
	if err != nil {
		t.Fatal(err)
	}
	if len(queues) != 2 {
		t.Fatal("Expected 2 sites")
	}
	for _, q := range queues {
		if q.Site.Name == s1.Name {
			if len(q.Items) != 3 {
				t.Fatalf("Expected 3 items for site %s", q.Site.Name)
			}
		}
		if q.Site.Name == s2.Name {
			if len(q.Items) != 2 {
				t.Fatalf("Expected 2 items for site %s", q.Site.Name)
			}
		}
		for i := range q.Items {
			if want, got := "The.Wire", q.Items[i].Media.Name; got != want {
				t.Errorf("got Items[%d].Media.Name=%q, want %q", i, got, want)
			}
			if !q.Items[i].Transfer {
				t.Errorf("got Items[%d].Transfer=true, want %t", i, false)
			}
		}
	}
}

func TestReadQueueInvalidSite(t *testing.T) {
	lines := `t1 /tv/The.Wire.S01E01`
	_, err := Read([]Site{}, strings.NewReader(lines))
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestMarshalJSON(t *testing.T) {
	s := newTestSite()
	var q json.Marshaler = newTestQueue(s, []os.FileInfo{file{name: "/remote/The.Wire.S01E01"}})
	out, err := q.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	want := `[
  {
    "RemotePath": "/remote/The.Wire.S01E01",
    "LocalPath": "/local/The.Wire/S1/The.Wire.S01E01",
    "ModTime": "0001-01-01T00:00:00Z",
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
]`
	if got := string(out); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestMarshalText(t *testing.T) {
	s := newTestSite()
	var q encoding.TextMarshaler = newTestQueue(s, []os.FileInfo{file{name: "/remote/The.Wire.S01E01"}})
	out, err := q.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	want := `open test
queue mirror /remote/The.Wire.S01E01 /local/The.Wire/S1/The.Wire.S01E01
queue start
wait
exit
`
	if got := string(out); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
