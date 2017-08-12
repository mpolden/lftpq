package lftp

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type file struct {
	path    string
	modTime time.Time
	mode    os.FileMode
}

func (f file) Name() string       { return f.path }
func (f file) Size() int64        { return 0 }
func (f file) Mode() os.FileMode  { return f.mode }
func (f file) ModTime() time.Time { return f.modTime }
func (f file) IsDir() bool        { return f.Mode().IsDir() }
func (f file) Sys() interface{}   { return nil }

func ParseFile(s string) (file, error) {
	parts := strings.SplitN(s, " ", 5)
	if len(parts) != 5 {
		return file{}, fmt.Errorf("failed to parse file: %s", s)
	}
	t := strings.Join(parts[:4], " ")
	modified, err := time.Parse("2006-01-02 15:04:05 -0700 MST", t)
	if err != nil {
		return file{}, err
	}
	path := parts[4]

	var fileMode os.FileMode
	if strings.HasSuffix(path, "@") {
		fileMode = os.ModeSymlink
	} else if strings.HasSuffix(path, "/") {
		fileMode = os.ModeDir
	}

	path = strings.TrimRight(path, "@/")
	return file{
		path:    path,
		modTime: modified,
		mode:    fileMode,
	}, nil
}
