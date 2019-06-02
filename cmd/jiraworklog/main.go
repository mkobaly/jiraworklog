package main

import (
	"errors"
	"github.com/mkobaly/jiraworklog/job"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	cmdline "github.com/galdor/go-cmdline"
	"github.com/jmoiron/sqlx"

	"github.com/mkobaly/jiraworklog"
	//wl "github.com/mkobaly/jiraworklog"
	//"github.com/mkobaly/jiraworklog/workers"
	"github.com/mkobaly/jiraworklog/writers"
	//log "github.com/sirupsen/logrus"
)

var db *sqlx.DB
var ErrUnknownWriter = errors.New("unkown Writer")

func main() {

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	//Define command line params and parse input
	cmdline := cmdline.New()
	cmdline.AddOption("c", "config", "config.yaml", "path to configuration file")
	cmdline.AddOption("w", "writer", "MSSQL", "specific writer to use")
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

	//Writer Settings
	writerType := "MSSQL"
	if cmdline.IsOptionSet("w") {
		writerType = cmdline.OptionValue("w")
	}

	//load writer
	writer, err := loadWriter(writerType, cfg)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info(writer)

	jiraWorklogDownloader := job.NewJiraDownloadWorklogs(cfg, writer, logger)

	// job1 := jiraworklog.Job{Name: "job1", Interval: 8 * time.Second, Action: func() error {
	// 	logger.Info("Job1 running")
	// 	return nil
	// }}
	// job2 := jiraworklog.Job{Name: "job2", Interval: 5 * time.Second, Action: func() error {
	// 	logger.Info("Job2 running")
	// 	return nil
	// }}
	worker := jiraworklog.NewWorker2(logger, jiraWorklogDownloader)
	go worker.Start()
	//time.Sleep(30 * time.Second)

	select {
	case sig := <-c:
		logger.WithField("signal", sig).Warn("Shutting down due to signal")
		worker.Shutdown()
		time.Sleep(1 * time.Second)
	}

	// worker := jiraworklog.NewWorker(90*time.Second, logger, func() error {
	// 	err := workers.Run(cfg, writer, logger)
	// 	return err
	// })
	// workerResolution := jiraworklog.NewWorker(5*time.Minute, logger, func() error {
	// 	err := workers.RunResolution(cfg, writer, logger)
	// 	return err
	// })
	// //go worker.Run()
	// go workerResolution.Run()
	//select {} // block forever

}

func loadWriter(writerType string, cfg *jiraworklog.Config) (writers.Writer, error) {
	switch writerType {
	case "MSSQL":
		db, err := sqlx.Connect("sqlserver", cfg.SQLConnection)
		if err != nil {
			return nil, err
		}
		writer := &writers.SQLWriter{DB: db}
		return writer, nil
	case "GOOGLESHEET":
		return nil, ErrUnknownWriter
	default:
		return nil, ErrUnknownWriter
	}
	return nil, ErrUnknownWriter
}
