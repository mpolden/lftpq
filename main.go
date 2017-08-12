package main

import (
	"fmt"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"

	"github.com/mpolden/lftpq/queue"
)

type CLI struct {
	Config string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpqrc"`
	Dryrun bool   `short:"n" long:"dryrun" description:"Print queue and exit"`
	Format string `short:"F" long:"format" description:"Format to use in dryrun mode" choice:"lftp" choice:"json" default:"lftp"`
	Test   bool   `short:"t" long:"test" description:"Test and print config"`
	Quiet  bool   `short:"q" long:"quiet" description:"Do not print output from lftp"`
	Import string `short:"i" long:"import" description:"Read remote paths from stdin and build a queue for SITE" value-name:"SITE"`
}

func (c *CLI) logf(format string, v ...interface{}) {
	if !c.Quiet {
		log.Printf(format, v...)
	}
}

func (c *CLI) importQueue(name string, cfg queue.Config) error {
	s, err := cfg.LookupSite(name)
	if err != nil {
		return err
	}
	queue, err := queue.Read(s, os.Stdin)
	if err != nil {
		return err
	}
	return c.process(queue)
}

func (c *CLI) buildQueue(s queue.Site) error {
	if s.Skip {
		c.logf("[%s] Skipping site (Skip=%t)", s.Name, s.Skip)
		return nil
	}
	var files []os.FileInfo
	for _, dir := range s.Dirs {
		f, err := s.Client.List(s.Name, dir)
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
		return q.Fprintln(os.Stdout, c.Format == "json")
	}
	if len(q.Transferable()) == 0 {
		c.logf("[%s] Queue is empty", q.Site.Name)
		return nil
	}
	if err := q.Start(!c.Quiet); err != nil {
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
	cfg, err := queue.ReadConfig(cli.Config)
	if err != nil {
		log.Fatal(err)
	}
	if cli.Test {
		json, err := cfg.JSON()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", json)
		return
	}
	if cli.Import == "" {
		for _, s := range cfg.Sites {
			if err := cli.buildQueue(s); err != nil {
				log.Print(err)
			}
		}
		return
	}
	if err := cli.importQueue(cli.Import, cfg); err != nil {
		log.Print(err)
	}
}
