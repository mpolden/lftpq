package site

import (
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
	d := Dir{
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
	dir := Dir{
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
