package site

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
)

type Items []Item

func (s Items) Len() int {
	return len(s)
}

func (s Items) Less(i, j int) bool {
	return s[i].Remote.Path < s[j].Remote.Path
}

func (s Items) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type Item struct {
	Remote    lftp.File
	LocalDir  string
	Transfer  bool
	Reason    string
	Media     parser.Media
	Duplicate bool
	Merged    bool
	*Queue    `json:"-"`
}

func (i *Item) DstDir() string {
	// When LocalDir has a trailing slash, the actual destination dir will be a directory inside LocalDir (same
	// behaviour as rsync)
	if strings.HasSuffix(i.LocalDir, string(os.PathSeparator)) {
		return filepath.Join(i.LocalDir, i.Remote.Base())
	}
	return i.LocalDir
}

func (i *Item) IsEmpty(readDir readDir) bool {
	dirs, _ := readDir(i.DstDir())
	return len(dirs) == 0
}

func (i *Item) Weight() int {
	for _i, p := range i.Queue.priorities {
		if i.Remote.Match(p) {
			return len(i.Queue.priorities) - _i
		}
	}
	return 0
}

func (i *Item) Accept(reason string) {
	i.Transfer = true
	i.Reason = reason
}

func (i *Item) Reject(reason string) {
	i.Transfer = false
	i.Reason = reason
}

func (i *Item) parseLocalDir() (string, error) {
	if i.Queue.localDir == nil {
		return "", fmt.Errorf("template is not set")
	}
	var b bytes.Buffer
	if err := i.Queue.localDir.Execute(&b, i.Media); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (i *Item) setMedia(dirname string) error {
	m, err := i.Queue.parser(dirname)
	if err != nil {
		return err
	}
	for _, r := range i.Replacements {
		m.ReplaceName(r.pattern, r.Replacement)
	}
	i.Media = m
	return nil
}

func (i *Item) setLocalDir() error {
	d, err := i.parseLocalDir()
	if err != nil {
		return err
	}
	i.LocalDir = d
	return nil
}

func (i *Item) duplicates(readDir readDir) Items {
	var items Items
	parent := filepath.Join(i.DstDir(), "..")
	dirs, _ := readDir(parent)
	for _, fi := range dirs {
		// Ignore self
		if i.Remote.Base() == fi.Name() {
			continue
		}
		path := filepath.Join(parent, fi.Name())
		item := Item{
			Remote:   lftp.File{Path: path}, // Needs to be set as weight is calculated based on Path
			Queue:    i.Queue,
			LocalDir: path,
			Transfer: true, // True to make it considerable for deduplication
			Merged:   true,
		}
		if err := item.setMedia(filepath.Base(path)); err != nil {
			item.Reject(err.Error())
		}
		// Ignore unequal media
		if !i.Media.Equal(item.Media) {
			continue
		}
		items = append(items, item)
	}
	return items
}

func newItem(q *Queue, f lftp.File) (Item, error) {
	item := Item{Queue: q, Remote: f, Reason: "no match"}
	if err := item.setMedia(f.Base()); err != nil {
		return item, err
	}
	if err := item.setLocalDir(); err != nil {
		return item, err
	}
	return item, nil
}
