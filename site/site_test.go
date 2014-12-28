package site

import (
	"github.com/martinp/lftpfetch/ftpdir"
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
			LocalPath:  tmpl,
		},
		Name:   "foo",
		Dir:    "/misc",
		MaxAge: time.Duration(24) * time.Hour,
	}
	d := ftpdir.Dir{
		Path:    "/misc/The.Wire.S02E01.720p.HDTV.x264-BATV",
		Created: time.Now(),
	}
	expected := "mirror /misc/The.Wire.S02E01.720p.HDTV.x264-BATV /tmp/The.Wire/S02/"
	getCmd, err := s.GetCmd(d)
	if err != nil {
		t.Fatal(err)
	}
	if getCmd.Args != expected {
		t.Fatalf("Expected %s, got %s", expected, getCmd.Args)
	}
}

func TestQueueCmd(t *testing.T) {
	tmpl := template.Must(template.New("localPath").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}"))
	s := Site{
		Client: Client{
			LftpPath:   "lftp",
			LftpGetCmd: "mirror",
			LocalPath:  tmpl,
		},
		Name:   "foo",
		Dir:    "/misc",
		MaxAge: time.Duration(24) * time.Hour,
	}
	dir := ftpdir.Dir{
		Path:    "/misc/The.Wire.S02E01",
		Created: time.Now(),
	}

	expected := "queue mirror /misc/The.Wire.S02E01 /tmp/The.Wire/S02/"
	queueCmd, err := s.QueueCmd(dir)
	if err != nil {
		t.Fatal(err)
	}
	if queueCmd.Args != expected {
		t.Fatalf("Expected %s, got %s", expected, queueCmd.Args)
	}
}

func TestFilterDirs(t *testing.T) {
	s := Site{
		Name:         "foo",
		Dir:          "/misc",
		MaxAge:       time.Duration(24) * time.Hour,
		Patterns:     []*regexp.Regexp{regexp.MustCompile("dir\\d")},
		Filters:      []*regexp.Regexp{regexp.MustCompile("^incomplete-")},
		SkipSymlinks: true,
	}
	dirs := []ftpdir.Dir{
		ftpdir.Dir{
			Path:    "/tmp/dir1@",
			Created: time.Now(),
			// Filtered because of symlink
			IsSymlink: true,
		},
		ftpdir.Dir{
			Path: "/tmp/dir2",
			// Filtered because of exceeded MaxAge
			Created: time.Now().Add(-time.Duration(48) * time.Hour),
		},
		ftpdir.Dir{
			Path: "/tmp/foo",
			// Filtered because of not matching any Patterns
			Created: time.Now(),
		},
		ftpdir.Dir{
			Path: "/tmp/incomplete-dir3",
			// Filtered because of matching any Filters
			Created: time.Now(),
		},
		ftpdir.Dir{
			Path:    "/tmp/dir4",
			Created: time.Now(),
		},
	}
	expected := []ftpdir.Dir{dirs[4]}
	filtered := s.FilterDirs(dirs)
	if !reflect.DeepEqual(expected, filtered) {
		t.Fatalf("Expected %+v, got %+v", expected, filtered)
	}
}
