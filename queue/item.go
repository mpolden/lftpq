package queue

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"text/template"

	"github.com/mpolden/lftpq/parser"
)

type Item struct {
	RemotePath string
	LocalPath  string
	ModTime    time.Time
	Transfer   bool
	Reason     string
	Media      parser.Media
	Duplicate  bool
	Merged     bool
	itemParser
}

func (i *Item) isEmpty(readDir readDir) bool {
	dirs, _ := readDir(i.LocalPath)
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

func (i *Item) setLocalPath(t *template.Template) error {
	var b bytes.Buffer
	if err := t.Execute(&b, i.Media); err != nil {
		return err
	}
	path := b.String()
	// When path has a trailing slash, the actual destination path will be a directory inside LocalPath (same
	// behaviour as rsync)
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		path = filepath.Join(path, filepath.Base(i.RemotePath))
	}
	i.LocalPath = path
	return nil
}

func (i *Item) duplicates(readDir readDir) []Item {
	var items []Item
	parent := filepath.Join(i.LocalPath, "..")
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
	if err := item.setLocalPath(itemParser.template); err != nil {
		return item, err
	}
	return item, nil
}
