package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"

	"github.com/martinp/lftpq/site"
)

type CLI struct {
	Config  string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpqrc"`
	Dryrun  bool   `short:"n" long:"dryrun" description:"Print queue and exit"`
	Format  string `short:"F" long:"format" description:"Format to use in dryrun mode" choice:"lftp" choice:"json" default:"lftp"`
	Test    bool   `short:"t" long:"test" description:"Test and print config"`
	Quiet   bool   `short:"q" long:"quiet" description:"Do not print output from lftp"`
	Verbose []bool `short:"v" long:"verbose" description:"Verbose output. Can be specified multiple times"`
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
		if c.Format == "json" {
			json, err := json.Marshal(queue.Items)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", json)
		} else {
			fmt.Print(queue.Script())
		}
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
		if err := cli.Run(s); err != nil {
			log.Print(err)
		}
	}
}
