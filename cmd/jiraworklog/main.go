package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/fatih/color"
	cmdline "github.com/galdor/go-cmdline"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/job"
	"github.com/mkobaly/jiraworklog/repository"
)

var db *sqlx.DB

// ErrUnknownRepo is error for unknown repository
var ErrUnknownRepo = errors.New("unkown repo")

func main() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	//Define command line params and parse input
	cmdline := cmdline.New()
	cmdline.AddOption("c", "config", "config.yaml", "path to configuration file")
	cmdline.AddOption("r", "repo", "POSTGRES", "specific repo to use (MSSQL, POSTGRES)")
	cmdline.SetOptionDefault("r", "POSTGRES")
	cmdline.AddOption("p", "port", "8380", "default port to serve rest API from")
	cmdline.SetOptionDefault("p", "8380")
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
			logger.Error("failed to load config file", "error", err)
			os.Exit(1)
		}
	}

	//Port
	port := 8380
	if cmdline.IsOptionSet("p") {
		port, err = strconv.Atoi(cmdline.OptionValue("p"))
		if err != nil {
			logger.Error("port must be numeric", "error", err)
			os.Exit(1)
		}
	}

	//Repo Settings
	repoType := "POSTGRES"
	if cmdline.IsOptionSet("r") {
		repoType = cmdline.OptionValue("r")
	}

	//load repo
	repo, err := loadRepo(repoType, cfg)
	if err != nil {
		logger.Error("failed to load repository", "error", err, "type", repoType)
		os.Exit(1)
	}

	jira := jiraworklog.NewJira(cfg)
	//List out all jobs we need here to run
	j1 := job.NewJiraSyncWorklogsJob(cfg, jira, repo)
	j2 := job.NewJJiraSyncIssuesJob(cfg, jira, repo)
	worker := jiraworklog.NewWorker(logger, j1, j2)
	go worker.Start()

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// Create handler
	handler := NewHandler(repo, logger)

	// Static files (paths relative to where binary is run)
	e.Static("/static", "../static")
	e.Static("/web", "../web") // Legacy web directory support

	// Dashboard routes
	e.GET("/", handler.Dashboard)
	e.GET("/dashboard", handler.Dashboard)

	// Worklog routes
	e.GET("/worklogs", handler.GetWorkLogs)
	e.GET("/worklogs/groupby", handler.GetWorklogsGroupBy)
	e.GET("/worklogs/perdev", handler.GetWorklogsPerDev)
	e.GET("/worklogs/perdevweek", handler.GetWorklogsPerDevWeek)

	// Issue routes
	e.GET("/issues", handler.GetIssues)
	e.GET("/issues/groupby", handler.GetIssuesGroupedBy)
	e.GET("/issues/accuracy", handler.GetIssueAccuracy)

	// Reports routes
	e.GET("/reports/maintenance", handler.GetMaintenanceRatio)
	e.GET("/reports/missing-charge", handler.GetIssuesMissingProjectCharge)
	e.GET("/reports/customer-bugs", handler.GetCustomerBugs)

	// Settings routes
	e.GET("/settings/people", handler.GetPeople)
	e.PUT("/settings/people/:id/role", handler.UpdatePersonRole)

	// Start server in background
	go func() {
		logger.Info("Starting HTTP server", "port", port)
		if err := e.Start(":" + strconv.Itoa(port)); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal
	sig := <-c
	logger.Warn("Shutting down due to signal", "signal", sig.String())

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown server gracefully", "error", err)
	}

	worker.Shutdown()
	repo.Close()
	time.Sleep(1 * time.Second)
}

func loadRepo(repoType string, cfg *jiraworklog.Config) (repository.Repo, error) {
	switch repoType {
	case "POSTGRES":
		return repository.NewPostgresRepo(cfg)
	default:
		return nil, ErrUnknownRepo
	}
}
