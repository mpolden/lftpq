package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/martinp/lftpfetch/cmd"
	"github.com/martinp/lftpfetch/site"
	"log"
	"os"
)

type CLI struct {
	Config string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpfetchrc"`
	Dryrun bool   `short:"n" long:"dryrun" description:"Print generated command instead of running it"`
	Test   bool   `short:"t" long:"test" description:"Test config and exit"`
	Quiet  bool   `short:"q" long:"quiet" description:"Do not print actions"`
}

func (c *CLI) Log(format string, v ...interface{}) {
	if !c.Quiet {
		log.Printf(format, v...)
	}
}

func (c *CLI) Run(s site.Site) error {
	dirs, err := s.GetDirs()
	if err != nil {
		return err
	}
	filtered := s.FilterDirs(dirs)
	cmds := make([]cmd.Lftp, 0, len(dirs))
	for _, d := range filtered {
		cmd, err := s.QueueCmd(d)
		if err != nil {
			c.Log("Skipping %s: %s", d.Path, err)
			continue
		}
		c.Log("Queuing %s", d.Path)
		cmds = append(cmds, cmd)
	}
	if len(cmds) == 0 {
		c.Log("Nothing to queue")
		return nil
	}
	queueCmd, err := cmd.Join(cmds)
	if err != nil {
		return err
	}
	if c.Dryrun {
		fmt.Println(queueCmd.String())
	} else if err := queueCmd.Run(); err != nil {
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
		log.Print("Read config successfully")
		return
	}
	for _, s := range cfg.Sites {
		if err := cli.Run(s); err != nil {
			log.Fatal(err)
		}
	}
}
