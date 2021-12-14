package main

import (
	"errors"
	"strconv"

	"github.com/mkobaly/jiraworklog/job"

	//"github.com/mkobaly/jiraworklog/test"
	"net/http"
	"os"
	"os/signal"

	//"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/fatih/color"
	cmdline "github.com/galdor/go-cmdline"
	"github.com/jmoiron/sqlx"

	"github.com/mkobaly/jiraworklog"
	//wl "github.com/mkobaly/jiraworklog"
	//"github.com/mkobaly/jiraworklog/workers"
	"github.com/mkobaly/jiraworklog/repository"
	//log "github.com/sirupsen/logrus"
)

var db *sqlx.DB

//ErrUnknownRepo is error for unknown repository
var ErrUnknownRepo = errors.New("unkown repo")

func main() {

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	//Define command line params and parse input
	cmdline := cmdline.New()
	cmdline.AddOption("c", "config", "config.yaml", "path to configuration file")
	cmdline.AddOption("r", "repo", "BOLTDB", "specific repo to use (MSSQL, BOLTDB)")
	cmdline.SetOptionDefault("r", "BOLTDB")
	cmdline.AddOption("p", "port", "8180", "default port to serve rest API from")
	cmdline.SetOptionDefault("p", "8180")
	cmdline.AddFlag("k", "ask", "Ask for username and password from the STDIN")
	cmdline.AddFlag("v", "verbose", "verbose logging")
	cmdline.Parse(os.Args)

	//Logger setup
	logLevel := "warn"
	if cmdline.IsOptionSet("v") {
		logLevel = "info"
	}
	logger := jiraworklog.NewLogger(jiraworklog.LoggerOptions{Application: "jiraWorklog", Level: logLevel})

	//Load up configuration. This holds Jira and SQL connection information
	cfgPath := "config.yaml"
	if cmdline.IsOptionSet("c") {
		cfgPath = cmdline.OptionValue("c")
	}

	cfg, err := jiraworklog.LoadConfig(cfgPath)
	if err != nil {
		switch err {
		case jiraworklog.ErrNoConfigFile:
			color.Yellow("============================================================================================================")
			color.Yellow("Config file not present. Config.yaml was just created for you but you must edit the credential information")
			color.Yellow("============================================================================================================")
			os.Exit(0)
		default:
			logger.WithError(err).Fatal("failed to load config file")
		}
	}

	//Port
	port := 8180
	if cmdline.IsOptionSet("p") {
		port, err = strconv.Atoi(cmdline.OptionValue("p"))
		if err != nil {
			logger.WithError(err).Fatal("port must be numeric")
		}
	}

	//Repo Settings
	repoType := "BOLTDB"
	if cmdline.IsOptionSet("r") {
		repoType = cmdline.OptionValue("r")
	}

	//load repo
	repo, err := loadRepo(repoType, cfg)
	if err != nil {
		logger.Fatal(err)
	}

	//jira := &test.FakeJira{}
	jira := jiraworklog.NewJira(cfg)
	//List out all jobs we need here to run
	j1 := job.NewJiraDownloadWorklogs(cfg, jira, repo, logger)
	j2 := job.NewJiraCheckResolution(cfg, jira, repo, logger)
	worker := jiraworklog.NewWorker(logger, j1, j2)
	go worker.Start()

	//HTTP server stuff
	fileServer := http.FileServer(FileSystem{http.Dir("./web")})
	server := NewHttpServer(repo, logger)
	mux := http.NewServeMux()
	mux.Handle("/worklogs", http.HandlerFunc(server.GetWorkLogs))
	mux.Handle("/worklogs/groupby", http.HandlerFunc(server.GetWorklogsGroupBy))
	//mux.Handle("/worklogs/perday", http.HandlerFunc(server.GetWorklogsPerDay))
	mux.Handle("/worklogs/perdev", http.HandlerFunc(server.GetWorklogsPerDev))
	//mux.Handle("/worklogs/perdevday", http.HandlerFunc(server.GetWorklogsPerDevDay))
	mux.Handle("/worklogs/perdevweek", http.HandlerFunc(server.GetWorklogsPerDevWeek))

	mux.Handle("/issues", http.HandlerFunc(server.GetIssues))
	mux.Handle("/issues/groupby", http.HandlerFunc(server.GetIssuesGroupedBy))
	mux.Handle("/issues/accuracy", http.HandlerFunc(server.GetIssueAccuracy))
	//mux.Handle("/", http.StripPrefix(strings.TrimRight("/dashboard/", "/"), fileServer))
	mux.Handle("/", fileServer)
	logger.Info("Starting HTTP server at *:" + strconv.Itoa(port))
	go http.ListenAndServe(":"+strconv.Itoa(port), mux)

	select {
	case sig := <-c:
		logger.WithField("signal", sig).Warn("Shutting down due to signal")
		worker.Shutdown()
		repo.Close()
		time.Sleep(1 * time.Second)
	}
}

func loadRepo(repoType string, cfg *jiraworklog.Config) (repository.Repo, error) {
	switch repoType {
	case "MSSQL":
		return repository.NewSQLRepo(cfg)
	case "BOLTDB":
		return repository.NewBoltDBRepo("jira.db")
	case "GOOGLESHEET":
		return nil, ErrUnknownRepo
	default:
		return nil, ErrUnknownRepo
	}
}
