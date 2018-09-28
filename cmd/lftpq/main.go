package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	flags "github.com/jessevdk/go-flags"

	"github.com/mpolden/lftpq/lftp"
	"github.com/mpolden/lftpq/queue"
)

type lister interface {
	List(site, path string) ([]os.FileInfo, error)
}

type CLI struct {
	Config   string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpqrc"`
	Dryrun   bool   `short:"n" long:"dryrun" description:"Print queue and exit"`
	Format   string `short:"F" long:"format" description:"Format to use in dryrun mode" choice:"lftp" choice:"json" default:"lftp"`
	Test     bool   `short:"t" long:"test" description:"Test and print config"`
	Quiet    bool   `short:"q" long:"quiet" description:"Do not print output from lftp"`
	Import   bool   `short:"i" long:"import" description:"Build queues from stdin"`
	LocalDir string `short:"l" long:"local-dir" description:"Override local dir for this run" value-name:"NAME"`
	LftpPath string `short:"p" long:"lftp" description:"Path to lftp program" value-name:"NAME" default:"lftp"`
	consumer queue.Consumer
	lister   lister
	stderr   io.Writer
	stdout   io.Writer
	stdin    io.Reader
}

func New() *CLI {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGPIPE)
	cli := CLI{}
	go func() {
		<-sig
		cli.unlock()
		os.Exit(1)
	}()
	return &cli
}

func (c *CLI) Run() error {
	cfg, err := queue.ReadConfig(c.Config)
	if err != nil {
		return err
	}
	if c.LocalDir != "" {
		if err := cfg.SetLocalDir(c.LocalDir); err != nil {
			return err
		}
	}
	if c.Test {
		json, err := cfg.JSON()
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "%s\n", json)
		return nil
	}
	var queues []queue.Queue
	if c.Import {
		if queues, err = queue.Read(cfg.Sites, c.stdin); err != nil {
			return err
		}
	} else {
		if err := c.lock(); err != nil {
			return fmt.Errorf("already running: %s", err)
		}
		defer c.unlock()
		queues = c.queuesFor(cfg.Sites)
	}
	for _, q := range queues {
		if err := c.transfer(q); err != nil {
			c.printf("error while transferring queue for %s: %s\n", q.Site.Name, err)
			continue
		}
	}
	return nil
}

func (c *CLI) lockfile() string { return filepath.Join(os.TempDir(), ".lftpqlock") }

func (c *CLI) lock() error {
	_, err := os.OpenFile(c.lockfile(), os.O_CREATE|os.O_EXCL, 0644)
	return err
}

func (c *CLI) unlock() { os.Remove(c.lockfile()) }

func (c *CLI) printf(format string, vs ...interface{}) {
	alwaysPrint := false
	for _, v := range vs {
		if _, ok := v.(error); ok {
			alwaysPrint = true
			break
		}
	}
	if !c.Quiet || alwaysPrint {
		fmt.Fprint(c.stderr, "lftpq: ")
		fmt.Fprintf(c.stderr, format, vs...)
	}
}

func (c *CLI) queuesFor(sites []queue.Site) []queue.Queue {
	var queues []queue.Queue
	for _, s := range sites {
		if s.Skip {
			c.printf("skipping site %s\n", s.Name)
			continue
		}
		var files []os.FileInfo
		for _, dir := range s.Dirs {
			f, err := c.lister.List(s.Name, dir)
			if err != nil {
				c.printf("error while listing %s on %s: %s\n", dir, s.Name, err)
				continue
			}
			files = append(files, f...)
		}
		queue := queue.New(s, files)
		queues = append(queues, queue)
	}
	return queues
}

func (c *CLI) transfer(q queue.Queue) error {
	if c.Dryrun {
		var (
			out []byte
			err error
		)
		if c.Format == "json" {
			out, err = q.MarshalJSON()
			out = append(out, 0x0a) // Add trailing newline
		} else {
			out, err = q.MarshalText()
		}
		if err == nil {
			fmt.Fprint(c.stdout, string(out))
		}
		return err
	}
	if len(q.Transferable()) == 0 {
		c.printf("%s queue is empty\n", q.Site.Name)
		return nil
	}
	if err := q.Transfer(c.consumer); err != nil {
		return err
	}
	return q.PostProcess(!c.Quiet)
}

func main() {
	cli := New()
	cli.stderr = os.Stderr
	cli.stdout = os.Stdout
	cli.stdin = os.Stdin
	p := flags.NewParser(cli, flags.HelpFlag|flags.PassDoubleDash)
	if _, err := p.Parse(); err != nil {
		cli.printf("%s\n", err)
		os.Exit(1)
	}
	client := lftp.Client{Path: cli.LftpPath, InheritIO: !cli.Quiet}
	cli.lister = &client
	cli.consumer = &client
	if err := cli.Run(); err != nil {
		cli.printf("%s\n", err)
		os.Exit(1)
	}
}
