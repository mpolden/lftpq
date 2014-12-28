package site

import (
	"testing"
	"time"
)

func TestGetCmd(t *testing.T) {
	s := Site{
		Client: Client{
			LftpPath:   "lftp",
			LftpGetCmd: "mirror",
			LocalPath:  "/tmp",
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
	getCmd, err := s.getCmd(d)
	if err != nil {
		t.Fatal(err)
	}
	if getCmd != expected {
		t.Fatalf("Expected %s, got %s", expected, getCmd)
	}
}

func TestQueueCmd(t *testing.T) {
	s := Site{
		Client: Client{
			LftpPath:   "lftp",
			LftpGetCmd: "mirror",
			LocalPath:  "/tmp",
		},
		Name:   "foo",
		Dir:    "/misc",
		MaxAge: time.Duration(24) * time.Hour,
	}
	dirs := []Dir{
		Dir{
			Path:    "/misc/The.Wire.S02E01",
			Created: time.Now(),
		},
		Dir{
			Path:    "/misc/The.Wire.S02E02",
			Created: time.Now(),
		},
	}
	expected := "queue mirror /misc/The.Wire.S02E01 /tmp/The.Wire/S02/" +
		" && queue mirror /misc/The.Wire.S02E02 /tmp/The.Wire/S02/" +
		" && queue start && wait"

	queueCmd, err := s.queueCmd(dirs)
	if err != nil {
		t.Fatal(err)
	}
	if queueCmd != expected {
		t.Fatalf("Expected %s, got %s", expected, queueCmd)
	}
}
