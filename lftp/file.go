package lftp

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type File struct {
	path    string
	modTime time.Time
	mode    os.FileMode
}

func (f File) Name() string       { return f.path }
func (f File) Size() int64        { return 0 }
func (f File) Mode() os.FileMode  { return f.mode }
func (f File) ModTime() time.Time { return f.modTime }
func (f File) IsDir() bool        { return f.Mode().IsDir() }
func (f File) Sys() interface{}   { return nil }

func ParseFile(s string) (File, error) {
	parts := strings.SplitN(s, " ", 5)
	if len(parts) != 5 {
		return File{}, fmt.Errorf("failed to parse file: %s", s)
	}
	t := strings.Join(parts[:4], " ")
	modified, err := time.Parse("2006-01-02 15:04:05 -0700 MST", t)
	if err != nil {
		return File{}, err
	}
	path := parts[4]

	var fileMode os.FileMode
	if strings.HasSuffix(path, "@") {
		fileMode = os.ModeSymlink
	} else if strings.HasSuffix(path, "/") {
		fileMode = os.ModeDir
	}

	path = strings.TrimRight(path, "@/")
	return File{
		path:    path,
		modTime: modified,
		mode:    fileMode,
	}, nil
}
