package main

import (
	"fmt"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"

	"github.com/martinp/lftpq/site"
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

func (c *CLI) importQueue(name string, cfg site.Config) error {
	s, err := cfg.LookupSite(name)
	if err != nil {
		return err
	}
	queue, err := site.ReadQueue(s, os.Stdin)
	if err != nil {
		return err
	}
	return c.process(queue)
}

func (c *CLI) buildQueue(s site.Site) error {
	if s.Skip {
		c.logf("[%s] Skipping site (Skip=%t)", s.Name, s.Skip)
		return nil
	}
	dirs, err := s.Client.List(s.Name, s.Dir)
	if err != nil {
		return err
	}
	queue := site.NewQueue(s, dirs)
	if err := c.process(queue); err != nil {
		return err
	}
	if c.Dryrun || s.PostCommand == "" {
		return nil
	}
	cmd, err := queue.PostCommand(!c.Quiet)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func (c *CLI) process(queue site.Queue) error {
	if c.Dryrun {
		if c.Format == "json" {
			json, err := queue.JSON()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", json)
		} else {
			fmt.Print(queue.Script())
		}
		return nil
	}
	if len(queue.Transferable()) == 0 {
		c.logf("[%s] Queue is empty", queue.Site.Name)
		return nil
	}
	return queue.Start(!c.Quiet)
}

func main() {
	var cli CLI
	_, err := flags.ParseArgs(&cli, os.Args)
	if err != nil {
		os.Exit(1)
	}
	cfg, err := site.ReadConfig(cli.Config)
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
