package queue

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var fieldSplitter = regexp.MustCompile(`\s+`)

type readDir func(dirname string) ([]os.FileInfo, error)

type Consumer interface {
	Consume(path string) error
}

type Queue struct {
	Site
	Items []Item
}

func lookupSite(name string, sites []Site) (Site, error) {
	for _, site := range sites {
		if site.Name == name {
			return site, nil
		}
	}
	return Site{}, fmt.Errorf("no such site: %s", name)
}

func New(site Site, files []os.FileInfo) Queue {
	return newQueue(site, files, ioutil.ReadDir)
}

func Read(sites []Site, r io.Reader) ([]Queue, error) {
	scanner := bufio.NewScanner(r)
	var qs []Queue
	// Store mapping from site name to queue index in qs, as we only want to return a single queue per site
	indices := map[string]int{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := fieldSplitter.Split(line, 2)
		if len(fields) < 2 {
			continue
		}
		site, err := lookupSite(fields[0], sites)
		if err != nil {
			return nil, err
		}
		i, ok := indices[site.Name]
		if !ok {
			qs = append(qs, Queue{Site: site})
			i = len(qs) - 1
			indices[site.Name] = i
		}
		q := &qs[i]
		item, err := newItem(fields[1], time.Time{}, q.itemParser)
		if err != nil {
			item.reject(err.Error())
		} else {
			item.accept("Import=true")
		}
		q.Items = append(q.Items, item)
	}
	return qs, scanner.Err()
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

func (q *Queue) Transfer(consumer Consumer) error {
	name, err := q.tempFile()
	if err != nil {
		return err
	}
	defer os.Remove(name)
	return consumer.Consume(name)
}

func (q *Queue) PostProcess(inheritIO bool) error {
	if q.postCommand == nil {
		return nil
	}
	json, err := json.Marshal(q.Items)
	if err != nil {
		return err
	}
	cmd := q.postCommand
	cmd.Stdin = bytes.NewReader(json)
	if inheritIO {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func (q Queue) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("open ")
	buf.WriteString(q.Site.Name)
	buf.WriteString("\n")
	for _, item := range q.Transferable() {
		buf.WriteString("queue ")
		buf.WriteString(q.Site.GetCmd)
		buf.WriteString(" '")
		buf.WriteString(item.RemotePath)
		buf.WriteString("' '")
		buf.WriteString(item.LocalPath)
		buf.WriteString("'\n")
	}
	buf.WriteString("queue start\nwait\n")
	return buf.Bytes(), nil
}

func (q Queue) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(q.Items, "", "  ")
}

func (q *Queue) tempFile() (string, error) {
	f, err := ioutil.TempFile("", "lftpq")
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := q.MarshalText()
	if err != nil {
		return "", err
	}
	if _, err := f.Write(b); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func (q *Queue) rank(item *Item) int {
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
				if (a.Merged || b.Merged) && q.rank(a) == q.rank(b) {
					continue
				}
				if q.rank(a) <= q.rank(b) {
					a.Duplicate = true
					a.reject(fmt.Sprintf("DuplicateOf=%s Rank=%d", b.RemotePath, q.rank(a)))
				} else {
					b.Duplicate = true
					b.reject(fmt.Sprintf("DuplicateOf=%s Rank=%d", a.RemotePath, q.rank(b)))
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
	now := time.Now().Round(time.Second)
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
		} else if age := now.Sub(item.ModTime); q.maxAge != 0 && age > q.maxAge {
			item.reject(fmt.Sprintf("Age=%s MaxAge=%s", age, q.maxAge))
		} else if p, match := matchAny(q.patterns, f); match {
			item.accept(fmt.Sprintf("Match=%s", p))
		}
		q.Items = append(q.Items, item)
	}
	if q.Merge {
		q.merge(readDir)
	}
	sort.Slice(q.Items, func(i, j int) bool { return q.Items[i].RemotePath < q.Items[j].RemotePath })
	if len(q.priorities) > 0 {
		q.deduplicate()
	}
	// Deduplication must happen before IsDstDir check. This is because items with a higher rank might have been
	// transferred in past runs.
	for _, item := range q.Transferable() {
		if q.SkipExisting && !item.isEmpty(readDir) {
			item.reject(fmt.Sprintf("IsDstDirEmpty=%t", false))
		}
	}
	return q
}
