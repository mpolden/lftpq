package site

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

type Queue struct {
	*Site
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
			if a.Transfer && b.Transfer && a.MediaEqual(*b) {
				if a.Weight() <= b.Weight() {
					a.Transfer = false
					a.Reason = fmt.Sprintf("DuplicateOf=%s Weight=%d", b.Dir.Path, a.Weight())
				} else {
					b.Transfer = false
					b.Reason = fmt.Sprintf("DuplicateOf=%s Weight=%d", a.Dir.Path, b.Weight())
				}
			}
		}
	}
}

func (q *Queue) skipNonEmptyDstDir() {
	for _, item := range q.Transferable() {
		if empty := item.IsDstDirEmpty(); !empty {
			item.Transfer = false
			item.Reason = fmt.Sprintf("IsDstDirEmpty=%t", empty)
		}
	}
}

func NewQueue(site *Site, dirs []Dir) Queue {
	items := make(Items, 0, len(dirs))
	q := Queue{Site: site}
	for _, dir := range dirs {
		item := newItem(&q, dir)
		if dir.IsSymlink && q.SkipSymlinks {
			item.Reason = fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", dir.IsSymlink, q.SkipSymlinks)
		} else if age := dir.Age(); age > q.maxAge {
			item.Reason = fmt.Sprintf("Age=%s MaxAge=%s", age, q.MaxAge)
		} else if p, match := dir.MatchAny(q.filters); match {
			item.Reason = fmt.Sprintf("Filter=%s", p)
		} else if p, match := dir.MatchAny(q.patterns); match {
			item.Transfer = true
			item.Reason = fmt.Sprintf("Match=%s", p)
		}
		items = append(items, item)
	}
	sort.Sort(items)
	q.Items = items
	if q.Deduplicate {
		q.deduplicate()
	}
	// Skipping of existing directories must be done after deduplication. This is because items with a higher weight
	// might have been transferred in a previous run, but should still be respected during deduplication.
	q.skipNonEmptyDstDir()
	return q
}

func (q *Queue) Transferable() []*Item {
	items := []*Item{}
	for i, _ := range q.Items {
		item := &q.Items[i]
		if !item.Transfer {
			continue
		}
		items = append(items, item)
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
	if _, err := f.WriteString(q.Script()); err != nil {
		return "", err
	}
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
