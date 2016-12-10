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
			// Ignore self
			if a.Remote.Path == b.Remote.Path {
				continue
			}
			if a.Transfer && b.Transfer && a.Media.Equal(b.Media) {
				if a.weight() <= b.weight() {
					a.Duplicate = true
					a.reject(fmt.Sprintf("DuplicateOf=%s Weight=%d", b.Remote.Path, a.weight()))
				} else {
					b.Duplicate = true
					b.reject(fmt.Sprintf("DuplicateOf=%s Weight=%d", a.Remote.Path, b.weight()))
				}
			}
		}
	}
}

func (q *Queue) merge(readDir readDir) {
	// Merge local duplicates into the queue so that they can included in deduplication
	for _, i := range q.Transferable() {
		for _, item := range i.duplicates(readDir) {
			q.Items = append(q.Items, item)
		}
	}
}

func newQueue(site Site, files []lftp.File, readDir readDir) Queue {
	q := Queue{Site: site, Items: make(Items, 0, len(files))}
	// Initial filtering
	for _, f := range files {
		item, err := newItem(&q, f)
		if err != nil {
			item.reject(err.Error())
		} else if q.SkipSymlinks && f.IsSymlink() {
			item.reject(fmt.Sprintf("IsSymlink=%t SkipSymlinks=%t", f.IsSymlink(), q.SkipSymlinks))
		} else if q.SkipFiles && f.IsRegular() {
			item.reject(fmt.Sprintf("IsFile=%t SkipFiles=%t", f.IsRegular(), q.SkipFiles))
		} else if p, match := f.MatchAny(q.filters); match {
			item.reject(fmt.Sprintf("Filter=%s", p))
		} else if p, match := f.MatchAny(q.patterns); match {
			item.accept(fmt.Sprintf("Match=%s", p))
		}
		q.Items = append(q.Items, item)
	}
	if q.Merge {
		q.merge(readDir)
	}
	sort.Sort(q.Items)
	if q.Deduplicate {
		q.deduplicate()
	}
	// Deduplication must happen before MaxAge and IsDstDir checks. This is because items with a higher weight might
	// have been transferred in past runs.
	now := time.Now()
	for _, item := range q.Transferable() {
		if age := item.Remote.Age(now); age > q.maxAge {
			item.reject(fmt.Sprintf("Age=%s MaxAge=%s", age, q.maxAge))
		} else if q.SkipExisting && !item.isEmpty(readDir) {
			item.reject(fmt.Sprintf("IsDstDirEmpty=%t", false))
		}
	}
	return q
}

func New(site Site, files []lftp.File) Queue {
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
		item, err := newItem(&q, lftp.File{Path: p})
		if err != nil {
			item.reject(err.Error())
		} else {
			item.accept("Import=true")
		}
		q.Items = append(q.Items, item)
	}
	if err := scanner.Err(); err != nil {
		return Queue{}, err
	}
	return q, nil
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
		buf.WriteString(item.Remote.Path)
		buf.WriteString(" ")
		buf.WriteString(item.LocalDir)
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
