package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/martinp/lftpfetch/cmd"
	"github.com/martinp/lftpfetch/site"
	"log"
	"os"
)

func main() {
	var opts struct {
		Dryrun bool   `short:"n" long:"dryrun" description:"Print generated command instead of running it"`
		Config string `short:"f" long:"config" description:"Path to config" value-name:"FILE" default:"~/.lftpfetchrc"`
	}
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		os.Exit(1)
	}
	cfg, err := site.ReadConfig(opts.Config)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range cfg.Sites {
		dirs, err := s.GetDirs()
		if err != nil {
			log.Fatal(err)
		}
		filtered := s.FilterDirs(dirs)
		cmds := make([]cmd.Lftp, 0, len(dirs))
		for _, d := range filtered {
			cmd, err := s.QueueCmd(d)
			if err != nil {
				log.Printf("Skipping cmd for %s: %s", d.Path,
					err)
				continue
			}
			cmds = append(cmds, cmd)
		}
		queueCmd, err := cmd.Join(cmds)
		if err != nil {
			log.Fatal(err)
		}
		if opts.Dryrun {
			fmt.Println(queueCmd.String())
		} else {
			if err := queueCmd.Run(); err != nil {
				log.Fatal(err)
			}
		}
	}
}
