package main

import (
	"fmt"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"

	"github.com/martinp/lftpq/site"
)

type CLI struct {
	Config  string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpqrc"`
	Dryrun  bool   `short:"n" long:"dryrun" description:"Print generated queue and exit without executing lftp"`
	Test    bool   `short:"t" long:"test" description:"Test and print config"`
	Quiet   bool   `short:"q" long:"quiet" description:"Only print errors"`
	Verbose bool   `short:"v" long:"verbose" description:"Verbose output"`
}

func (c *CLI) Log(format string, v ...interface{}) {
	if !c.Quiet {
		log.Printf(format, v...)
	}
}

func (c *CLI) LogVerbose(format string, v ...interface{}) {
	if c.Verbose {
		log.Printf(format, v...)
	}
}

func (c *CLI) Run(s site.Site) error {
	dirs, err := s.DirList()
	if err != nil {
		return err
	}
	queue := site.NewQueue(s, dirs)
	for _, item := range queue.Items {
		c.LogVerbose(item.String())
		if !c.Verbose && item.Transfer {
			c.Log(item.String())
		}
	}
	if len(queue.TransferItems()) == 0 {
		c.LogVerbose("Nothing to transfer")
		return nil
	}
	if c.Dryrun {
		fmt.Print(queue.Script())
	} else if err := queue.Start(); err != nil {
		return err
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
		fmt.Printf("%+v\n", cfg)
		return
	}
	for _, s := range cfg.Sites {
		if err := cli.Run(s); err != nil {
			log.Fatal(err)
		}
	}
}
