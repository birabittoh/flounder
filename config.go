package main

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	FilesPath  string
	RootDomain string
	SiteTitle  string
	Debug      bool
	SecretKey  string
}

func getConfig(filename string) (Config, error) {
	var config Config
	// Attempt to overwrite defaults from file
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}
