package main

import (
	"errors"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

var errNoConfigFile = errors.New("No config file. One will be created for you")

//JiraSettings represent connection and credential information to Jira
type JiraSettings struct {
	URL      string
	Username string
	Password string
}

// Config holds info needed for connecting to Jira and SQL
type Config struct {
	Jira          JiraSettings
	SQLConnection string
	MaxWorklogID  int
	LastTimestamp int64
	UserList      []string
	DoneStatus    []string
}

//Save will persist the configuration information to the given path
func (c *Config) Save(path string) error {
	bytes, err := yaml.Marshal(c)
	if err == nil {
		return ioutil.WriteFile(path, bytes, 0777)
	}
	return err
}

//LoadConfig will load up a Config object based on configPath
func LoadConfig(path string) (*Config, error) {

	//if one does not exist, lets create it and return with err
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := newConfig()
		bytes, err := yaml.Marshal(cfg)
		if err == nil {
			ioutil.WriteFile(path, bytes, 0644)
		}
		return nil, errNoConfigFile
	}

	var config = new(Config)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

//newConfig will create a default config file with placeholder values
func newConfig() *Config {

	var config = &Config{
		Jira:          JiraSettings{URL: "https://your-url.atlassian.net", Username: "username", Password: "use_api_token"},
		SQLConnection: "Server=localhost;Database=Jira;User Id=xxx;Password=yyyyyy",
		MaxWorklogID:  0,
		LastTimestamp: 0,
		UserList:      []string{"leave.empty", "to.pull", "all.users"},
		DoneStatus:    []string{"done", "closed"},
	}
	return config
}
