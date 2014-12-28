package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/martinp/lftptv/site"
	"log"
	"os"
	"strings"
)

func main() {
	var opts struct {
		Dryrun bool   `short:"n" long:"dryrun" description:"Print generated command instead of running it"`
		Config string `short:"f" long:"config" description:"Path to config" value-name:"FILE" required:"true"`
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
		dirs, err := s.FilterDirs()
		if err != nil {
			log.Fatal(err)
		}
		cmd, err := s.QueueCmd(dirs)
		if err != nil {
			log.Fatal(err)
		}
		if opts.Dryrun {
			fmt.Println(strings.Join(cmd.Args, " "))
		} else {
			fmt.Printf("running %s\n", strings.Join(cmd.Args, " "))
		}
	}
}
