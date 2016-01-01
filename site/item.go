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
	return s[i].Dir.Path < s[j].Dir.Path
}

func (s Items) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type Item struct {
	lftp.Dir
	LocalDir string
	Transfer bool
	Reason   string
	Media    parser.Media
	*Queue   `json:"-"`
}

func (i *Item) String() string {
	return fmt.Sprintf("Path=%q LocalDir=%q Transfer=%t Reason=%q", i.Path, i.LocalDir, i.Transfer, i.Reason)
}

func (i *Item) DstDir() string {
	// When LocalDir has a trailing slash, the actual destination dir will be a directory inside LocalDir (same
	// behaviour as rsync)
	if strings.HasSuffix(i.LocalDir, string(os.PathSeparator)) {
		return filepath.Join(i.LocalDir, i.Dir.Base())
	}
	return i.LocalDir
}

func (i *Item) IsEmpty(readDir readDir) bool {
	dirs, _ := readDir(i.DstDir())
	return len(dirs) == 0
}

func (i *Item) Weight() int {
	for _i, p := range i.Queue.priorities {
		if i.Dir.Match(p) {
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

func (i *Item) setMetadata() error {
	m, err := i.Queue.parser(i.Dir.Base())
	if err != nil {
		return err
	}
	for _, r := range i.Replacements {
		m.ReplaceName(r.pattern, r.Replacement)
	}
	i.Media = m

	d, err := i.parseLocalDir()
	if err != nil {
		return err
	}
	i.LocalDir = d
	return nil
}

func newItem(q *Queue, d lftp.Dir) (Item, error) {
	item := Item{Queue: q, Dir: d, Reason: "no match"}
	if err := item.setMetadata(); err != nil {
		return item, err
	}
	return item, nil
}
