package site

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/martinp/lftpfetch/cmd"
)

type Item struct {
	Dir
	LocalDir string
	Skip     bool
	Reason   string
}

func (i *Item) String() string {
	return fmt.Sprintf("Path=%s LocalDir=%q Skip=%t Reason=%q", i.Path, i.LocalDir, i.Skip, i.Reason)
}

type Queue struct {
	Site
	Items []Item
}

func (q *Queue) filterDirs(dirs []Dir) []Item {
	items := make([]Item, 0, len(dirs))
	for _, dir := range dirs {
		item := Item{Dir: dir, Skip: true}
		if dir.IsSymlink && q.SkipSymlinks {
			item.Reason = fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", dir.IsSymlink, q.SkipSymlinks)
		} else if age, after := dir.CreatedAfter(q.maxAge); !after {
			item.Reason = fmt.Sprintf("Age=%s MaxAge=%s", age, q.MaxAge)
		} else if p, match := dir.MatchAny(q.filters); match {
			item.Reason = fmt.Sprintf("Filter=%s", p)
		} else if p, match := dir.MatchAny(q.patterns); match {
			item.Skip = false
			item.Reason = fmt.Sprintf("Match=%s", p)
		} else {
			item.Reason = fmt.Sprintf("Match=<none>")
		}
		items = append(items, item)
	}
	return items
}

func (q *Queue) getLocalDir(dir Dir) (string, error) {
	localDir := q.LocalDir
	if q.ParseTVShow {
		show, err := dir.Show()
		if err != nil {
			return "", err
		}
		var b bytes.Buffer
		if err := q.localDir.Execute(&b, show); err != nil {
			return "", err
		}
		localDir = b.String()
	}
	if !strings.HasSuffix(localDir, string(os.PathSeparator)) {
		localDir += string(os.PathSeparator)
	}
	dstDir := filepath.Join(localDir, dir.Base())
	if _, err := os.Stat(dstDir); err == nil {
		return "", fmt.Errorf("%s already exists", dstDir)
	}
	return localDir, nil
}

func (q *Queue) findLocalDir(items []Item) ([]Item, error) {
	for i, item := range items {
		if item.Skip {
			continue
		}
		localDir, err := q.getLocalDir(item.Dir)
		if err != nil {
			items[i].Skip = true
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
		if item.Skip {
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
	f, err := ioutil.TempFile("", "lftpfetch")
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
	cmd := cmd.Lftp{
		Path: q.LftpPath,
		Args: []string{"-f", name},
	}
	return cmd.Run()
}
