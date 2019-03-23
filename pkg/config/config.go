package config

import (
	"io/ioutil"
	"os"

	"github.com/RasmusLindroth/indy/pkg/news"
	"gopkg.in/yaml.v2"
)

//Config holds the config for the app
type Config struct {
	Web      `yaml:"web"`
	Database `yaml:"db"`
	Sites    []*news.Site `yaml:"sites"`
}

//Web holds configuration for the web server
type Web struct {
	Port  string `yaml:"port"`
	Files string `yaml:"files"`
}

//Database holds credentials for the db
type Database struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
}

//ParseFile parses a config from disk
func ParseFile(path string) (*Config, error) {
	conf := &Config{}

	f, err := os.Open(path)
	if err != nil {
		return conf, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}
