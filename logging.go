package jiraworklog

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type LoggerOptions struct {
	Application string
	LogFile     string
	Level       string
}

func NewLogger(options LoggerOptions) *logrus.Entry {

	if options.Level == "" {
		options.Level = "warn"
	}
	level, err := logrus.ParseLevel(options.Level)
	if err != nil {
		panic(err)
	}

	log := logrus.New()
	log.Level = level
	log.Formatter = &logrus.TextFormatter{
		//ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		//DisableColors:   true,
	}

	// &logrus.JSONFormatter{}

	if options.LogFile != "" {
		log.Out = os.Stdout
		file, err := os.OpenFile(options.LogFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			log.Out = io.MultiWriter(file, os.Stdout)
		} else {
			log.Info("Failed to log to file, using default stderr")
		}
	}

	logger := log.WithFields(logrus.Fields{"app": options.Application})
	return logger

}
