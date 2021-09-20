package queue

import (
	"path/filepath"
	"time"

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
	localDir   LocalDir
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
		item, err := newItem(path, i.ModTime, i.localDir)
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

func newItem(remotePath string, modTime time.Time, localDir LocalDir) (Item, error) {
	item := Item{RemotePath: remotePath, ModTime: modTime, Reason: "no match", localDir: localDir}
	media, err := localDir.Media(remotePath)
	if err != nil {
		return Item{}, err
	}
	item.Media = media
	item.LocalPath, err = media.PathIn(localDir.Template)
	if err != nil {
		return Item{}, err
	}
	return item, nil
}
