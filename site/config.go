package site

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"time"
)

type Config struct {
	Client Client
	Sites  []Site
}

func CompilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}
	return res, nil
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
	for i, site := range cfg.Sites {
		maxAge, err := time.ParseDuration(site.MaxAge_)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].MaxAge = maxAge
		patterns, err := CompilePatterns(site.Patterns_)
		if err != nil {
			return Config{}, err
		}
		cfg.Sites[i].Patterns = patterns
		cfg.Sites[i].Client = cfg.Client

	}
	return cfg, nil
}
