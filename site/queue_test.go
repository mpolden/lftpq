package site

import (
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"testing"
	"text/template"
	"time"
)

func TestFilterDirs(t *testing.T) {
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
			Path: "/tmp/foo",
			// Filtered because of not matching any Patterns
			Created: now,
		},
		Dir{
			Path: "/tmp/incomplete-dir3",
			// Filtered because of matching any Filters
			Created: now,
		},
		Dir{
			Path: "/tmp/dir4",
			// Included because less than MaxAge
			Created: now,
		},
	}
	expected := []Item{
		Item{Dir: dirs[0], Transfer: false, Reason: "IsSymlink=true SkipSymlinks=true"},
		Item{Dir: dirs[1], Transfer: false, Reason: "Age=48h0m0s MaxAge="},
		Item{Dir: dirs[2], Transfer: true, Reason: "Match=dir\\d"},
		Item{Dir: dirs[3], Transfer: false, Reason: "no match"},
		Item{Dir: dirs[4], Transfer: false, Reason: "Filter=^incomplete-"},
		Item{Dir: dirs[5], Transfer: true, Reason: "Match=dir\\d"},
	}
	q := Queue{Site: s}
	actual := q.filterDirs(dirs)
	if len(expected) != len(actual) {
		t.Fatal("Expected equal length")
	}
	for i, _ := range expected {
		if !reflect.DeepEqual(expected[i], actual[i]) {
			t.Fatalf("Expected %+v, got %+v", expected[i], actual[i])
		}
	}
}

func TestBuildLocalDirShow(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}"))
	s := Site{
		localDir: tmpl,
		Parser:   "show",
	}
	d := Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	path, err := q.buildLocalDir(d)
	if err != nil {
		t.Fatal(err)
	}
	if expected := "/tmp/The.Wire/S03/"; path != expected {
		t.Fatalf("Expected %s, got %s", expected, path)
	}
}

func TestBuildLocalDirMovie(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Year }}/{{ .Name }}"))
	s := Site{
		localDir: tmpl,
		Parser:   "movie",
	}
	d := Dir{Path: "/foo/Apocalypse.Now.1979"}
	q := Queue{Site: s}
	path, err := q.buildLocalDir(d)
	if err != nil {
		t.Fatal(err)
	}
	if expected := "/tmp/1979/Apocalypse.Now/"; path != expected {
		t.Fatalf("Expected %s, got %s", expected, path)
	}
}

func TestBuildLocalDirNoTemplate(t *testing.T) {
	s := Site{
		LocalDir: "/tmp",
	}
	d := Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	path, err := q.buildLocalDir(d)
	if err != nil {
		t.Fatal(err)
	}
	if expected := "/tmp/"; path != expected {
		t.Fatalf("Expected %s, got %s", expected, path)
	}
}

func TestWrite(t *testing.T) {
	site := Site{
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
	q := Queue{Site: site, Items: items}
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

func TestTransferItems(t *testing.T) {
	q := Queue{
		Items: []Item{
			Item{Dir: Dir{Path: "/tmp/d1"}, Transfer: true},
			Item{Dir: Dir{Path: "/tmp/d2"}, Transfer: false},
			Item{Dir: Dir{Path: "/tmp/d3"}, Transfer: true},
		},
	}
	actual := q.TransferItems()
	expected := []Item{q.Items[0], q.Items[2]}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %+v, got %+v", expected, actual)
	}
}
