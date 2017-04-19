package queue

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type readDir func(dirname string) ([]os.FileInfo, error)

type Queue struct {
	Site
	Items []Item
}

func New(site Site, files []os.FileInfo) Queue {
	return newQueue(site, files, ioutil.ReadDir)
}

func Read(site Site, r io.Reader) (Queue, error) {
	q := Queue{Site: site}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		p := strings.TrimSpace(scanner.Text())
		if len(p) == 0 {
			continue
		}
		item, err := newItem(p, time.Time{}, q.itemParser)
		if err != nil {
			item.reject(err.Error())
		} else {
			item.accept("Import=true")
		}
		q.Items = append(q.Items, item)
	}
	return q, scanner.Err()
}

func (q *Queue) Transferable() []*Item {
	var items []*Item
	for i := range q.Items {
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
		buf.WriteString(item.RemotePath)
		buf.WriteString(" ")
		buf.WriteString(item.LocalPath)
		buf.WriteString("\n")
	}
	buf.WriteString("queue start\nwait\nexit\n")
	return buf.String()
}

func (q *Queue) Start(inheritIO bool) error {
	name, err := q.write()
	if err != nil {
		return err
	}
	defer os.Remove(name)
	return q.Client.Run([]string{"-f", name}, inheritIO)
}

func (q *Queue) Fprintln(w io.Writer, printJSON bool) error {
	if printJSON {
		b, err := json.MarshalIndent(q.Items, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\n", b)
	} else {
		fmt.Fprint(w, q.Script())
	}
	return nil
}

func (q *Queue) PostCommand(inheritIO bool) (*exec.Cmd, error) {
	json, err := json.Marshal(q.Items)
	if err != nil {
		return nil, err
	}
	argv := strings.Split(q.Site.PostCommand, " ")
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = bytes.NewReader(json)
	if inheritIO {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd, nil
}

func (q *Queue) write() (string, error) {
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

func (q *Queue) weight(item *Item) int {
	for i, p := range q.priorities {
		if p.MatchString(filepath.Base(item.RemotePath)) {
			return len(q.priorities) - i
		}
	}
	return 0
}

func (q *Queue) deduplicate() {
	for i := range q.Items {
		for j := range q.Items {
			if i == j {
				continue
			}
			a := &q.Items[i]
			b := &q.Items[j]
			// Ignore self
			if a.RemotePath == b.RemotePath {
				continue
			}
			if a.Transfer && b.Transfer && a.Media.Equal(b.Media) {
				if (a.Merged || b.Merged) && q.weight(a) == q.weight(b) {
					continue
				}
				if q.weight(a) <= q.weight(b) {
					a.Duplicate = true
					a.reject(fmt.Sprintf("DuplicateOf=%s Weight=%d", b.RemotePath, q.weight(a)))
				} else {
					b.Duplicate = true
					b.reject(fmt.Sprintf("DuplicateOf=%s Weight=%d", a.RemotePath, q.weight(b)))
				}
			}
		}
	}
}

func (q *Queue) merge(readDir readDir) {
	// Merge on-disk duplicates into the queue so that they can be considered for deduplication
	for _, i := range q.Transferable() {
		q.Items = append(q.Items, i.duplicates(readDir)...)
	}
}

func matchAny(patterns []*regexp.Regexp, f os.FileInfo) (string, bool) {
	for _, p := range patterns {
		if p.MatchString(filepath.Base(f.Name())) {
			return p.String(), true
		}
	}
	return "", false
}

func newQueue(site Site, files []os.FileInfo, readDir readDir) Queue {
	q := Queue{Site: site, Items: make([]Item, 0, len(files))}
	// Initial filtering
	for _, f := range files {
		item, err := newItem(f.Name(), f.ModTime(), q.itemParser)
		if err != nil {
			item.reject(err.Error())
		} else if isSymlink := f.Mode()&os.ModeSymlink != 0; q.SkipSymlinks && isSymlink {
			item.reject(fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", isSymlink, q.SkipSymlinks))
		} else if q.SkipFiles && f.Mode().IsRegular() {
			item.reject(fmt.Sprintf("IsFile=%t SkipFiles=%t", f.Mode().IsRegular(), q.SkipFiles))
		} else if p, match := matchAny(q.filters, f); match {
			item.reject(fmt.Sprintf("Filter=%s", p))
		} else if p, match := matchAny(q.patterns, f); match {
			item.accept(fmt.Sprintf("Match=%s", p))
		}
		q.Items = append(q.Items, item)
	}
	if q.Merge {
		q.merge(readDir)
	}
	sort.Slice(q.Items, func(i, j int) bool { return q.Items[i].RemotePath < q.Items[j].RemotePath })
	if q.Deduplicate {
		q.deduplicate()
	}
	// Deduplication must happen before MaxAge and IsDstDir checks. This is because items with a higher weight might
	// have been transferred in past runs.
	now := time.Now().Round(time.Second)
	for _, item := range q.Transferable() {
		if age := now.Sub(item.ModTime); q.maxAge != 0 && age > q.maxAge {
			item.reject(fmt.Sprintf("Age=%s MaxAge=%s", age, q.maxAge))
		} else if q.SkipExisting && !item.isEmpty(readDir) {
			item.reject(fmt.Sprintf("IsDstDirEmpty=%t", false))
		}
	}
	return q
}
