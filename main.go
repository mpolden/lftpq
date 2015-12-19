package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	flags "github.com/jessevdk/go-flags"

	"github.com/martinp/lftpq/site"
)

type CLI struct {
	Config  string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpqrc"`
	Dryrun  bool   `short:"n" long:"dryrun" description:"Print generated queue and exit without executing lftp"`
	Test    bool   `short:"t" long:"test" description:"Test and print config"`
	Quiet   bool   `short:"q" long:"quiet" description:"Only print errors"`
	Verbose []bool `short:"v" long:"verbose" description:"Verbose output"`
	Pattern string `short:"m" long:"match" description:"Only process sites matching PATTERN" value-name:"PATTERN"`
}

func (c *CLI) Log(format string, v ...interface{}) {
	if !c.Quiet {
		log.Printf(format, v...)
	}
}

func (c *CLI) Run(s site.Site) error {
	dirs, err := s.Client.List(s.Name, s.Dir)
	if err != nil {
		return err
	}
	queue := site.NewQueue(s, dirs)
	for _, item := range queue.Items {
		if (item.Transfer && len(c.Verbose) == 1) || len(c.Verbose) > 1 {
			c.Log(item.String())
		}
	}
	if len(queue.Transferable()) == 0 {
		c.Log("Nothing to transfer")
		return nil
	}
	if c.Dryrun {
		fmt.Print(queue.Script())
		return nil
	}
	if err := queue.Start(!c.Quiet); err != nil {
		return err
	}

	if s.PostCommand != "" {
		if cmd, err := queue.PostCommand(); err != nil {
			return err
		} else if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
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
	for _, s := range cfg.Sites {
		if cli.Pattern != "" {
			match, err := filepath.Match(cli.Pattern, s.Name)
			if err != nil {
				log.Fatal(err)
			}
			if !match {
				fmt.Printf("Skipping site: %s (did not match %s)\n", s.Name, cli.Pattern)
				continue
			}
		}
		if err := cli.Run(s); err != nil {
			log.Print(err)
		}
	}
}
