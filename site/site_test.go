package site

import (
	"reflect"
	"regexp"
	"testing"
	"text/template"
	"time"
)

func TestGetCmd(t *testing.T) {
	tmpl := template.Must(template.New("localPath").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}"))
	s := Site{
		Client: Client{
			LftpPath:   "lftp",
			LftpGetCmd: "mirror",
		},
		Name:        "foo",
		Dir:         "/misc",
		maxAge:      time.Duration(24) * time.Hour,
		localDir:    tmpl,
		ParseTVShow: true,
	}
	d := Dir{
		Path:    "/misc/The.Wire.S02E01.720p.HDTV.x264-BATV",
		Created: time.Now(),
	}
	expected := "mirror /misc/The.Wire.S02E01.720p.HDTV.x264-BATV /tmp/The.Wire/S02/"
	getCmd, err := s.GetCmd(d)
	if err != nil {
		t.Fatal(err)
	}
	if getCmd.Script != expected {
		t.Fatalf("Expected %s, got %s", expected, getCmd.Script)
	}
}

func TestQueueCmd(t *testing.T) {
	tmpl := template.Must(template.New("localPath").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}"))
	s := Site{
		Client: Client{
			LftpPath:   "lftp",
			LftpGetCmd: "mirror",
		},
		Name:        "foo",
		Dir:         "/misc",
		maxAge:      time.Duration(24) * time.Hour,
		localDir:    tmpl,
		ParseTVShow: true,
	}
	dir := Dir{
		Path:    "/misc/The.Wire.S02E01",
		Created: time.Now(),
	}

	expected := "queue mirror /misc/The.Wire.S02E01 /tmp/The.Wire/S02/"
	queueCmd, err := s.QueueCmd(dir)
	if err != nil {
		t.Fatal(err)
	}
	if queueCmd.Script != expected {
		t.Fatalf("Expected %s, got %s", expected, queueCmd.Script)
	}
}

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
	expected := []Dir{dirs[4]}
	filtered := s.FilterDirs(dirs)
	if !reflect.DeepEqual(expected, filtered) {
		t.Fatalf("Expected %+v, got %+v", expected, filtered)
	}
}

func TestParseLocalDir(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}"))
	s := Site{
		localDir:    tmpl,
		ParseTVShow: true,
	}
	d := Dir{
		Path: "/foo/The.Wire.S03E01",
	}
	path, err := s.ParseLocalDir(d)
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
	d := Dir{
		Path: "/foo/The.Wire.S03E01",
	}
	path, err := s.ParseLocalDir(d)
	if err != nil {
		t.Fatal(err)
	}
	if expected := "/tmp/"; path != expected {
		t.Fatalf("Expected %s, got %s", expected, path)
	}
}
