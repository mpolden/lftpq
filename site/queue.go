package site

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Item struct {
	Dir
	LocalDir string
	Transfer bool
	Reason   string
	Queue    *Queue
}

func (i *Item) String() string {
	return fmt.Sprintf("Path=%q LocalDir=%q Transfer=%t Reason=%q", i.Path, i.LocalDir, i.Transfer, i.Reason)
}

func (i *Item) IsDstDirEmpty() bool {
	dstDir := i.LocalDir
	// When LocalDir has a trailing slash, the actual dstDir will be a directory inside LocalDir
	// (same behaviour as rsync)
	if strings.HasSuffix(dstDir, string(os.PathSeparator)) {
		dstDir = filepath.Join(dstDir, i.Dir.Base())
	}
	dirs, _ := ioutil.ReadDir(dstDir)
	return len(dirs) == 0
}

func (i *Item) parseLocalDir() (string, error) {
	var data interface{}
	switch i.Queue.Parser {
	case "show":
		show, err := i.Dir.Show()
		if err != nil {
			return "", err
		}
		data = show
	case "movie":
		movie, err := i.Dir.Movie()
		if err != nil {
			return "", err
		}
		data = movie
	}
	var b bytes.Buffer
	if err := i.Queue.localDir.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (i *Item) setLocalDir() {
	localDir := i.Queue.LocalDir
	if i.Queue.Parser != "" {
		d, err := i.parseLocalDir()
		if err != nil {
			i.Transfer = false
			i.Reason = err.Error()
			return
		}
		localDir = d
	}
	i.LocalDir = localDir
}

func newItem(q *Queue, d Dir) Item {
	item := Item{Queue: q, Dir: d}
	item.setLocalDir()
	return item
}

type Queue struct {
	Site
	Items []Item
}

func NewQueue(site Site, dirs []Dir) Queue {
	items := make([]Item, 0, len(dirs))
	q := Queue{Site: site}
	for _, dir := range dirs {
		item := newItem(&q, dir)
		if dir.IsSymlink && q.SkipSymlinks {
			item.Reason = fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", dir.IsSymlink, q.SkipSymlinks)
		} else if age := dir.Age(); age > q.maxAge {
			item.Reason = fmt.Sprintf("Age=%s MaxAge=%s", age, q.MaxAge)
		} else if p, match := dir.MatchAny(q.filters); match {
			item.Reason = fmt.Sprintf("Filter=%s", p)
		} else if empty := item.IsDstDirEmpty(); !empty {
			item.Reason = fmt.Sprintf("IsDstDirEmpty=%t", empty)
		} else if p, match := dir.MatchAny(q.patterns); match {
			item.Transfer = true
			item.Reason = fmt.Sprintf("Match=%s", p)
		} else if item.Reason == "" {
			item.Reason = "no match"
		}
		items = append(items, item)
	}
	q.Items = items
	return q
}

func (q *Queue) TransferItems() []Item {
	items := []Item{}
	for _, item := range q.Items {
		if !item.Transfer {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (q *Queue) Script() string {
	var buf bytes.Buffer
	buf.WriteString("open " + q.Site.Name + "\n")
	for _, item := range q.TransferItems() {
		buf.WriteString("queue ")
		buf.WriteString(q.LftpGetCmd)
		buf.WriteString(" ")
		buf.WriteString(item.Path)
		buf.WriteString(" ")
		buf.WriteString(item.LocalDir)
		buf.WriteString("\n")
	}
	buf.WriteString("queue start\nwait\nexit\n")
	return buf.String()
}

func (q *Queue) Write() (string, error) {
	if len(q.Items) == 0 {
		return "", fmt.Errorf("queue is empty")
	}
	f, err := ioutil.TempFile("", "lftpq")
	if err != nil {
		return "", err
	}
	defer f.Close()
	f.WriteString(q.Script())
	return f.Name(), nil
}

func (q *Queue) Start() error {
	name, err := q.Write()
	if err != nil {
		return err
	}
	defer os.Remove(name)
	cmd := Lftp{
		Path: q.LftpPath,
		Args: []string{"-f", name},
	}
	return cmd.Run()
}
