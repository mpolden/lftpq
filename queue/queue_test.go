package queue

import (
	"encoding"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/mpolden/lftpq/parser"
)

func newTestSite() Site {
	tmpl := template.Must(template.New("t").Parse("/local/{{ .Name }}/S{{ .Season }}/"))
	patterns := []*regexp.Regexp{regexp.MustCompile(".*")}
	return Site{
		GetCmd:   "mirror",
		Name:     "test",
		patterns: patterns,
		localDir: LocalDir{
			Template: tmpl,
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
		localDir: LocalDir{
			Template: template.Must(template.New("localDir").Parse("/local/")),
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
		{localDir: q.localDir, RemotePath: files[0].Name(), Transfer: false, Reason: "IsSymlink=true SkipSymlinks=true"},
		{localDir: q.localDir, RemotePath: files[1].Name(), Transfer: false, Reason: "Age=48h0m0s MaxAge=24h0m0s"},
		{localDir: q.localDir, RemotePath: files[2].Name(), Transfer: true, Reason: "Match=dir\\d"},
		{localDir: q.localDir, RemotePath: files[3].Name(), Transfer: true, Reason: "Match=dir\\d"},
		{localDir: q.localDir, RemotePath: files[4].Name(), Transfer: false, Reason: "IsDstDirEmpty=false"},
		{localDir: q.localDir, RemotePath: files[5].Name(), Transfer: false, Reason: "no match"},
		{localDir: q.localDir, RemotePath: files[6].Name(), Transfer: false, Reason: "Filter=^incomplete-"},
		{localDir: q.localDir, RemotePath: files[7].Name(), Transfer: false, Reason: "IsFile=true SkipFiles=true"},
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
	want := `invalid input: "bar"`
	if got := q.Items[0].Reason; got != want {
		t.Errorf("Expected %q, got %q", want, got)
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
	q := newQueue(s, []os.FileInfo{file{name: "/remote/The.Wire.S01E01.720p.BluRay.foo"}}, readDir)
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
	q := newQueue(s, []os.FileInfo{file{name: "/remote/The.Wire.S01E01.720p.BluRay.foo"}}, readDir)
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

     t1      /tv/The.Wire.S01 E04 

ignored

t2 /tv/The.Wire.S01E02

t1 /tv/The.Wire.S01E03

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
	var tests = []struct {
		site        string
		remotePaths []string
	}{
		{"t1", []string{"/tv/The.Wire.S01E01", "/tv/The.Wire.S01 E04", "/tv/The.Wire.S01E03"}},
		{"t2", []string{"/tv/The.Wire.S01E02", "/tv/The.Wire.S01E05"}},
	}
	for i, tt := range tests {
		q := queues[i]
		if q.Site.Name != tt.site {
			t.Errorf("want Site.Name=%s, got %s for queue #%d", tt.site, q.Site.Name, i)
		}
		for j, rpath := range tt.remotePaths {
			item := q.Items[j]
			if item.RemotePath != rpath {
				t.Errorf("want RemotePath=%s, got %s for item #%d in queue #%d", rpath, item.RemotePath, j, i)
			}
			if !item.Transfer {
				t.Errorf("want Transfer=true, got %t for item #%d in queue #%d", item.Transfer, j, i)
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

func TestMarshalText(t *testing.T) {
	s := Site{
		GetCmd: "mirror",
		Name:   "siteA",
	}
	items := []Item{
		{RemotePath: "/remote/foo", LocalPath: "/local", Transfer: true},
		{RemotePath: "/remote/bar", LocalPath: "/local", Transfer: true},
		{RemotePath: "/remote/bar with ' and \\'", LocalPath: "/local with ' and \\'", Transfer: true},
	}
	var q encoding.TextMarshaler = Queue{Site: s, Items: items}
	out, err := q.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	expected := `open siteA
queue mirror '/remote/foo' '/local'
queue mirror '/remote/bar' '/local'
queue mirror '/remote/bar with \' and \'' '/local with \' and \''
queue start
wait
`
	script := string(out)
	if script != expected {
		t.Fatalf("Expected %q, got %q", expected, script)
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
      "Episode": 1,
      "Resolution": "",
      "Codec": ""
    },
    "Duplicate": false,
    "Merged": false
  }
]`
	if got := string(out); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
