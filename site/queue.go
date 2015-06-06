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
}

func (i *Item) String() string {
	return fmt.Sprintf("Path=%q LocalDir=%q Transfer=%t Reason=%q", i.Path, i.LocalDir, i.Transfer, i.Reason)
}

type Queue struct {
	Site
	Items []Item
}

func (q *Queue) filterDirs(dirs []Dir) []Item {
	items := make([]Item, 0, len(dirs))
	for _, dir := range dirs {
		item := Item{Dir: dir}
		if dir.IsSymlink && q.SkipSymlinks {
			item.Reason = fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", dir.IsSymlink, q.SkipSymlinks)
		} else if age := dir.Age(); age > q.maxAge {
			item.Reason = fmt.Sprintf("Age=%s MaxAge=%s", age, q.MaxAge)
		} else if p, match := dir.MatchAny(q.filters); match {
			item.Reason = fmt.Sprintf("Filter=%s", p)
		} else if p, match := dir.MatchAny(q.patterns); match {
			item.Transfer = true
			item.Reason = fmt.Sprintf("Match=%s", p)
		} else {
			item.Reason = "no match"
		}
		items = append(items, item)
	}
	return items
}

func (q *Queue) parseLocalDir(dir Dir, localDir string) (string, error) {
	var data interface{}
	switch q.Parser {
	case "show":
		show, err := dir.Show()
		if err != nil {
			return "", err
		}
		data = show
	case "movie":
		movie, err := dir.Movie()
		if err != nil {
			return "", err
		}
		data = movie
	}
	var b bytes.Buffer
	if err := q.localDir.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (q *Queue) buildLocalDir(dir Dir) (string, error) {
	localDir := q.LocalDir
	if q.Parser != "" {
		d, err := q.parseLocalDir(dir, q.LocalDir)
		if err != nil {
			return "", err
		}
		localDir = d
	}
	if !strings.HasSuffix(localDir, string(os.PathSeparator)) {
		localDir += string(os.PathSeparator)
	}
	dstDir := filepath.Join(localDir, dir.Base())
	if dirs, err := ioutil.ReadDir(dstDir); err == nil && len(dirs) > 0 {
		return "", fmt.Errorf("%s already exists and is not empty", dstDir)
	}
	return localDir, nil
}

func (q *Queue) findLocalDir(items []Item) ([]Item, error) {
	for i, item := range items {
		if !item.Transfer {
			continue
		}
		localDir, err := q.buildLocalDir(item.Dir)
		if err != nil {
			items[i].Transfer = false
			items[i].Reason = err.Error()
			continue
		}
		items[i].LocalDir = localDir
	}
	return items, nil
}

func (q *Queue) Process(dirs []Dir) error {
	items := q.filterDirs(dirs)
	items, err := q.findLocalDir(items)
	if err != nil {
		return err
	}
	q.Items = items
	return nil
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

func (q *Queue) String() string {
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
	f.WriteString(q.String())
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
