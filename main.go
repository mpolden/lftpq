package main

import (
	"encoding/json"
	"github.com/jessevdk/go-flags"
	"github.com/martinp/lftptv/site"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	LftpPath string
	Sites    []site.Site
}

func ReadConfig(name string) (Config, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func main() {
	var opts struct {
		Config   string `short:"f" long:"config" description:"Path to config" value-name:"FILE" required:"true"`
		LftpPath string `short:"p" long:"lftp-path" description:"Override path to lftp executable" value-name:"PATH" default:"lftp"`
	}
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		os.Exit(1)
	}

	cfg, err := ReadConfig(opts.Config)
	if err != nil {
		log.Fatal(err)
	}

	cfg.LftpPath = opts.LftpPath
	for _, site := range cfg.Sites {
		site.LftpPath = opts.LftpPath
		dirs, err := site.FilterDirs()
		if err != nil {
			log.Fatal(err)
		}
		for _, d := range dirs {
			log.Printf("%+v", d)
		}
	}
}
