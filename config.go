package main

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	FilesDirectory     string
	TemplatesDirectory string
	Hostname           string
	Host               string
	SiteTitle          string
	Debug              bool
	SecretKey          string
	DBFile             string
	PasswdFile         string // TODO remove
	CookieStoreKey     string
	OkExtensions       []string
	MaxFileSize        int
	TLSCertFile        string
	TLSKeyFile         string
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
