package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/martinp/lftpq/lftp"
)

type readDir func(dirname string) ([]os.FileInfo, error)

type Queue struct {
	Site
	Items
}

func (q *Queue) deduplicate() {
	for i, _ := range q.Items {
		for j, _ := range q.Items {
			if i == j {
				continue
			}
			a := &q.Items[i]
			b := &q.Items[j]
			if a.Transfer && b.Transfer && a.Media.Equal(b.Media) {
				if a.Weight() <= b.Weight() {
					a.Reject(fmt.Sprintf("DuplicateOf=%s Weight=%d", b.Dir.Path, a.Weight()))
				} else {
					b.Reject(fmt.Sprintf("DuplicateOf=%s Weight=%d", a.Dir.Path, b.Weight()))
				}
			}
		}
	}
}

func newQueue(site Site, dirs []lftp.Dir, readDir readDir) Queue {
	q := Queue{Site: site, Items: make(Items, 0, len(dirs))}
	// Initial filtering
	for _, dir := range dirs {
		item, err := newItem(&q, dir)
		if err != nil {
			item.Reject(err.Error())
		} else if q.SkipSymlinks && dir.IsSymlink {
			item.Reject(fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", dir.IsSymlink, q.SkipSymlinks))
		} else if q.SkipFiles && dir.IsFile {
			item.Reject(fmt.Sprintf("IsFile=%t SkipFiles=%t", dir.IsFile, q.SkipFiles))
		} else if p, match := dir.MatchAny(q.filters); match {
			item.Reject(fmt.Sprintf("Filter=%s", p))
		} else if p, match := dir.MatchAny(q.patterns); match {
			item.Accept(fmt.Sprintf("Match=%s", p))
		}
		q.Items = append(q.Items, item)
	}
	sort.Sort(q.Items)
	if q.Deduplicate {
		q.deduplicate()
	}
	// Deduplication must happen before MaxAge and IsDstDir checks. This is because items with a higher weight might
	// have been transferred in past runs.
	now := time.Now()
	for _, item := range q.Transferable() {
		if age := item.Age(now); age > q.maxAge {
			item.Reject(fmt.Sprintf("Age=%s MaxAge=%s", age, q.maxAge))
		} else if q.SkipExisting && !item.IsEmpty(readDir) {
			item.Reject(fmt.Sprintf("IsDstDirEmpty=%t", false))
		}
	}
	return q
}

func NewQueue(site Site, dirs []lftp.Dir) Queue {
	return newQueue(site, dirs, ioutil.ReadDir)
}

func (q *Queue) Transferable() []*Item {
	var items []*Item
	for i, _ := range q.Items {
		if item := &q.Items[i]; item.Transfer {
			items = append(items, item)
		}
	}
	return items
}

func (q *Queue) Script() string {
	var buf bytes.Buffer
	buf.WriteString("open ")
	buf.WriteString(q.Site.Name)
	buf.WriteString("\n")
	for _, item := range q.Transferable() {
		buf.WriteString("queue ")
		buf.WriteString(q.Client.GetCmd)
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
	f, err := ioutil.TempFile("", "lftpq")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(q.Script()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func (q *Queue) Start(inheritIO bool) error {
	name, err := q.Write()
	if err != nil {
		return err
	}
	defer os.Remove(name)
	return q.Client.Run([]string{"-f", name}, inheritIO)
}

func (q *Queue) PostCommand() (*exec.Cmd, error) {
	json, err := json.Marshal(q.Items)
	if err != nil {
		return nil, err
	}
	argv := strings.Split(q.Site.PostCommand, " ")
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = bytes.NewReader(json)
	return cmd, nil
}
