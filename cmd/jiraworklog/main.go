package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/internal/config"
	"github.com/mkobaly/jiraworklog/internal/email"
	"github.com/mkobaly/jiraworklog/job"
	"github.com/mkobaly/jiraworklog/repository"
)

var db *sqlx.DB

var Version string

// ErrUnknownRepo is error for unknown repository
var ErrUnknownRepo = errors.New("unkown repo")

func main() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	args := config.NewArgs(Version)
	_ = arg.MustParse(&args)

	//Logger setup
	logLevel := "warn"
	if args.Debug {
		logLevel = "info"
	}
	logger := jiraworklog.NewLogger(jiraworklog.LoggerOptions{Application: "jiraWorklog", Level: logLevel})

	cfg, err := jiraworklog.LoadConfig(args.Config)
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

	// //Repo Settings
	repoType := "POSTGRES"
	// if cmdline.IsOptionSet("r") {
	// 	repoType = cmdline.OptionValue("r")
	//}

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

	emailClient := email.NewSmtpClient(cfg)
	if args.Debug {
		emailClient = email.FakeEmailClient{}
	}
	// Create handler
	handler := NewHandler(repo, jira, logger, cfg, emailClient, args.Debug)

	// Static files (paths relative to where binary is run)
	e.Static("/static", "../static")
	e.Static("/web", "../web") // Legacy web directory support

	e.GET("/login", handler.Login)
	e.POST("/login", handler.LoginPost)
	e.GET("/login/confirm", handler.LoginConfirmGet)
	e.POST("/login/confirm", handler.LoginConfirm)

	// Dashboard routes
	e.GET("/", handler.Dashboard, handler.AuthMiddleware)
	e.GET("/dashboard", handler.Dashboard, handler.AuthMiddleware)
	e.GET("/dashboard/leadership", handler.GetLeadershipDashboard, handler.AuthMiddleware)
	e.GET("/dashboard/manager", handler.GetManagerDashboard, handler.AuthMiddleware)

	// Worklog routes
	// worklogs := e.Group("/worklogs", handler.AuthMiddleware)
	// worklogs.GET("/groupby", handler.GetWorklogsGroupBy)
	// worklogs.GET("/perdev", handler.GetWorklogsPerDev)
	// worklogs.GET("/perdevweek", handler.GetWorklogsPerDevWeek)

	// // Issue routes
	// e.GET("/issues/groupby", handler.GetIssuesGroupedBy)
	// e.GET("/issues/accuracy", handler.GetIssueAccuracy)

	// Reports routes
	reports := e.Group("/reports", handler.AuthMiddleware)
	reports.GET("/maintenance", handler.GetMaintenanceRatio)
	reports.GET("/missing-charge", handler.GetIssuesMissingProjectCharge)
	reports.GET("/mismatched-charge", handler.GetIssuesMismatchedProjectCharge)
	reports.GET("/customer-bugs", handler.GetCustomerBugs)
	reports.GET("/project-hours", handler.GetProjectChargeHours)
	reports.GET("/project-hours/csv", handler.GetProjectChargeHoursCSV)
	reports.GET("/weekly-hours", handler.GetWeeklyHours)
	reports.GET("/time-tracking", handler.GetProjectTimeTracking)
	reports.GET("/project-kpis", handler.GetProjectKPIs)

	// Settings routes
	settings := e.Group("/settings", handler.AuthMiddleware)
	settings.GET("/people", handler.GetPeople)
	settings.PUT("/people/:id", handler.UpdatePerson)
	settings.GET("/project-charges", handler.GetProjectCharges)
	settings.PUT("/project-charges", handler.UpdateProjectCharge)

	// Start server in background
	go func() {
		logger.Info("Starting HTTP server", "port", args.Port)
		if err := e.Start(":" + strconv.Itoa(args.Port)); err != nil && err != http.ErrServerClosed {
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
