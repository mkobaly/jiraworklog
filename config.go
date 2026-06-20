package jiraworklog

import (
	"errors"
	"os"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var ErrNoConfigFile = errors.New("No config file. One will be created for you")

// JiraSettings represent connection and credential information to Jira
type JiraSettings struct {
	URL      string
	Username string
	Password string
}

type SmtpSettings struct {
	Host     string
	Port     int
	UserName string
	Password string
}

// Config holds info needed for connecting to Jira and SQL
type Config struct {
	mu            sync.Mutex `yaml:"-"`
	path          string
	Jira          JiraSettings
	Smtp          SmtpSettings
	SQLConnection string
	//MaxWorklogID  int
	WorklogUpdatedLastTimestamp int64
	WorklogDeletedLastTimestamp int64
	IssueLastTimestamp          int64

	UserList              []string
	DoneStatus            []string
	ExcludedProjects      []string
	QueryExcludedProjects []string
	AuthorizedUsers       []string
	HTTPSecureCookie      bool
	UtcOffsetHours        int
}

// Save will persist the configuration information
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	bytes, err := yaml.Marshal(c)
	if err == nil {
		return os.WriteFile(c.path, bytes, 0777)
	}
	return err
}

// Write will persist the configuration information to the given path
func (c *Config) Write(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	bytes, err := yaml.Marshal(c)
	if err == nil {
		return os.WriteFile(path, bytes, 0777)
	}
	return err
}

// LoadConfig will load up a Config object based on configPath
func LoadConfig(path string) (*Config, error) {

	//if one does not exist, lets create it and return with err
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := newConfig()
		bytes, err := yaml.Marshal(cfg)
		if err == nil {
			os.WriteFile(path, bytes, 0644)
		}
		return nil, ErrNoConfigFile
	}

	var config = new(Config)
	config.path = path
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return loadEnv(config), nil
}

// newConfig will create a default config file with placeholder values
func newConfig() *Config {

	lastYear := time.Now().Add(time.Hour * -8760).Unix()
	var config = &Config{
		Jira:                        JiraSettings{URL: "https://your-url.example.com/rest/api/latest", Username: "username", Password: "use_api_token"},
		SQLConnection:               "Server=localhost;Database=Jira;User Id=xxx;Password=yyyyyy",
		WorklogDeletedLastTimestamp: lastYear,
		WorklogUpdatedLastTimestamp: lastYear,
		IssueLastTimestamp:          lastYear,
		UserList:                    []string{"leave.empty", "to.pull", "all.users"},
		DoneStatus:                  []string{"done", "closed"},
	}
	return config
}

// loadEnv injects configuration variables from the ENV
func loadEnv(c *Config) *Config {
	// Some fancy dynamicism would be great here.
	if val, ok := os.LookupEnv("JWL_JIRA_PASSWORD"); ok == true {
		c.Jira.Password = val
	}
	if val, ok := os.LookupEnv("JWL_JIRA_URL"); ok == true {
		c.Jira.URL = val
	}
	if val, ok := os.LookupEnv("JWL_JIRA_USERNAME"); ok == true {
		c.Jira.Username = val
	}
	return c
}
