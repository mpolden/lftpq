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
			Created: time.Now(),
			// Filtered because of symlink
			IsSymlink: true,
		},
		Dir{
			Path: "/tmp/dir2",
			// Filtered because of exceeded MaxAge
			Created: time.Now().Add(-time.Duration(48) * time.Hour),
		},
		Dir{
			Path: "/tmp/foo",
			// Filtered because of not matching any Patterns
			Created: time.Now(),
		},
		Dir{
			Path: "/tmp/incomplete-dir3",
			// Filtered because of matching any Filters
			Created: time.Now(),
		},
		Dir{
			Path:    "/tmp/dir4",
			Created: time.Now(),
		},
	}
	q, err := s.Queue(dirs)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range q.Items {
		if item.Transfer && !reflect.DeepEqual(item.Dir, dirs[4]) {
			t.Fatalf("Expected %q, got %q", dirs[4], item.Dir)
		}
	}
}

func TestParseLocalDir(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}"))
	s := Site{
		localDir:    tmpl,
		ParseTVShow: true,
	}
	d := Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	path, err := q.getLocalDir(d)
	if err != nil {
		t.Fatal(err)
	}
	if expected := "/tmp/The.Wire/S03/"; path != expected {
		t.Fatalf("Expected %s, got %s", expected, path)
	}
}

func TestParseLocalDirNoTemplate(t *testing.T) {
	s := Site{
		LocalDir: "/tmp",
	}
	d := Dir{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	path, err := q.getLocalDir(d)
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
