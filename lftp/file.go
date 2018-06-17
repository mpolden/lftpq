package lftp

import (
	"fmt"
	"os"
	"strconv"
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
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return file{}, fmt.Errorf("invalid file: %q", s)
	}
	secs, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return file{}, fmt.Errorf("invalid time: %q: %s", parts[0], err)
	}
	modified := time.Unix(secs, 0)
	path := parts[1]

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
