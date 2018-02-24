package main

import (
	"fmt"
	"io"
	"os"

	flags "github.com/jessevdk/go-flags"

	"github.com/mpolden/lftpq/lftp"
	"github.com/mpolden/lftpq/queue"
)

type lister interface {
	List(name, path string) ([]os.FileInfo, error)
}

type CLI struct {
	Config   string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpqrc"`
	Dryrun   bool   `short:"n" long:"dryrun" description:"Print queue and exit"`
	Format   string `short:"F" long:"format" description:"Format to use in dryrun mode" choice:"lftp" choice:"json" default:"lftp"`
	Test     bool   `short:"t" long:"test" description:"Test and print config"`
	Quiet    bool   `short:"q" long:"quiet" description:"Do not print output from lftp"`
	Import   bool   `short:"i" long:"import" description:"Build queues from stdin"`
	LftpPath string `short:"p" long:"lftp" description:"Path to lftp program" value-name:"NAME" default:"lftp"`
	consumer queue.Consumer
	lister   lister
	stderr   io.Writer
	stdout   io.Writer
	rd       io.Reader
}

func (c *CLI) Run() error {
	cfg, err := queue.ReadConfig(c.Config)
	if err != nil {
		return err
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
		if queues, err = queue.Read(cfg.Sites, c.rd); err != nil {
			return err
		}
	} else {
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
			fmt.Fprintf(c.stdout, "%s", out)
		}
		return err
	}
	if len(q.Transferable()) == 0 {
		c.printf("%s queue is empty\n", q.Site.Name)
		return nil
	}
	if err := q.Start(c.consumer); err != nil {
		return err
	}
	if q.Site.PostCommand == "" {
		return nil
	}
	cmd, err := q.PostCommand(!c.Quiet)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func main() {
	var cli CLI
	_, err := flags.ParseArgs(&cli, os.Args)
	if err != nil {
		os.Exit(1)
	}
	cli.stderr = os.Stderr
	cli.stdout = os.Stdout
	cli.rd = os.Stdin
	client := lftp.Client{Path: cli.LftpPath, InheritIO: !cli.Quiet}
	cli.lister = &client
	cli.consumer = &client
	if err := cli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
