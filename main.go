package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/martinp/lftptv/site"
	"log"
	"os"
	"strings"
)

func main() {
	var opts struct {
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
		dirs, err := s.GetDirs()
		if err != nil {
			log.Fatal(err)
		}
		fdirs := s.FilterDirs(dirs)
		for _, d := range fdirs {
			cmd, err := s.GetCmd(d)
			if err != nil {
				log.Printf("failed to get cmd for %s: %s",
					d.Name, err)
				continue
			}
			log.Print(strings.Join(cmd.Args, " "))
		}
	}
}
