package queue

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"text/template"

	"github.com/martinp/lftpq/parser"
)

type Items []Item

func (s Items) Len() int {
	return len(s)
}

func (s Items) Less(i, j int) bool {
	return s[i].RemotePath < s[j].RemotePath
}

func (s Items) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type Item struct {
	RemotePath string
	ModTime    time.Time
	LocalDir   string
	Transfer   bool
	Reason     string
	Media      parser.Media
	Duplicate  bool
	Merged     bool
	itemParser
}

func (i *Item) dstDir() string {
	// When LocalDir has a trailing slash, the actual destination dir will be a directory inside LocalDir (same
	// behaviour as rsync)
	if strings.HasSuffix(i.LocalDir, string(os.PathSeparator)) {
		return filepath.Join(i.LocalDir, filepath.Base(i.RemotePath))
	}
	return i.LocalDir
}

func (i *Item) isEmpty(readDir readDir) bool {
	dirs, _ := readDir(i.dstDir())
	return len(dirs) == 0
}

func (i *Item) accept(reason string) {
	i.Transfer = true
	i.Reason = reason
}

func (i *Item) reject(reason string) {
	i.Transfer = false
	i.Reason = reason
}

func (i *Item) setMedia(dirname string) error {
	m, err := i.itemParser.parser(dirname)
	if err != nil {
		return err
	}
	for _, r := range i.itemParser.replacements {
		m.ReplaceName(r.pattern, r.Replacement)
	}
	i.Media = m
	return nil
}

func (i *Item) setLocalDir(t *template.Template) error {
	var b bytes.Buffer
	if err := t.Execute(&b, i.Media); err != nil {
		return err
	}
	i.LocalDir = b.String()
	return nil
}

func (i *Item) duplicates(readDir readDir) Items {
	var items Items
	parent := filepath.Join(i.dstDir(), "..")
	dirs, _ := readDir(parent)
	for _, fi := range dirs {
		// Ignore self
		if filepath.Base(i.RemotePath) == fi.Name() {
			continue
		}
		path := filepath.Join(parent, fi.Name())
		item, err := newItem(path, i.ModTime, i.itemParser)
		if err != nil {
			item.reject(err.Error())
		} else {
			item.accept("Merged=true") // Make it considerable for deduplication
			item.Merged = true
		}
		// Ignore unequal media
		if !i.Media.Equal(item.Media) {
			continue
		}
		items = append(items, item)
	}
	return items
}

func newItem(remotePath string, modTime time.Time, itemParser itemParser) (Item, error) {
	item := Item{RemotePath: remotePath, ModTime: modTime, Reason: "no match", itemParser: itemParser}
	if err := item.setMedia(filepath.Base(remotePath)); err != nil {
		return item, err
	}
	if err := item.setLocalDir(itemParser.template); err != nil {
		return item, err
	}
	return item, nil
}
