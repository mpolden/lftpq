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
	Import   string `short:"i" long:"import" description:"Read remote paths from stdin and build a queue for SITE" value-name:"SITE"`
	LftpPath string `short:"p" long:"lftp" description:"Path to lftp program" value-name:"NAME" default:"lftp"`
	consumer queue.Consumer
	lister   lister
	wr       io.Writer
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
		fmt.Fprintf(c.wr, "%s\n", json)
		return nil
	}
	if c.Import == "" {
		for _, s := range cfg.Sites {
			if err := c.processQueue(s); err != nil {
				fmt.Fprintf(c.wr, "error while processing queue for %s: %s\n", s.Name, err)
			}
		}
		return nil
	}
	return c.processImportedQueue(c.Import, cfg)
}

func (c *CLI) printf(format string, v ...interface{}) {
	if !c.Quiet {
		fmt.Fprintf(c.wr, format, v...)
	}
}

func (c *CLI) processImportedQueue(name string, cfg queue.Config) error {
	s, err := cfg.LookupSite(name)
	if err != nil {
		return err
	}
	queue, err := queue.Read(s, c.rd)
	if err != nil {
		return err
	}
	return c.process(queue)
}

func (c *CLI) processQueue(s queue.Site) error {
	if s.Skip {
		c.printf("[%s] Skipping site (Skip=%t)\n", s.Name, s.Skip)
		return nil
	}
	var files []os.FileInfo
	for _, dir := range s.Dirs {
		f, err := c.lister.List(s.Name, dir)
		if err != nil {
			return err
		}
		files = append(files, f...)
	}
	queue := queue.New(s, files)
	return c.process(queue)
}

func (c *CLI) process(q queue.Queue) error {
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
			fmt.Fprintf(c.wr, "%s", out)
		}
		return err
	}
	if len(q.Transferable()) == 0 {
		c.printf("[%s] Queue is empty\n", q.Site.Name)
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
	cli.wr = os.Stdout
	cli.rd = os.Stdin
	client := lftp.Client{Path: cli.LftpPath, InheritIO: true}
	cli.lister = &client
	cli.consumer = &client
	if err := cli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
