package jiraworklog

import (
	"log/slog"
	"time"
)

// Worker will do its Action once every interval, making up for lost time that
// happened during the Action by only waiting the time left in the interval.
type Worker struct {
	Stopped  bool // A flag determining the state of the worker
	Jobs     []Job
	logger   *slog.Logger
	stopChan chan struct{}
}

//Job represents a job that needs to run at the given interval
type Job interface {
	Run() error
	GetInterval() time.Duration
	GetName() string
}

// NewWorker creates a new worker and instantiates all the data structures required.
func NewWorker(logger *slog.Logger, jobs ...Job) *Worker {
	return &Worker{
		Stopped:  false,
		stopChan: make(chan struct{}),
		logger:   logger,
		Jobs:     jobs,
	}
}

//Start will start the worker and listens for a shutdown call.
func (w *Worker) Start() {
	for _, job := range w.Jobs {
		go w.Run(job)
	}
}

//Run will execute the given job
func (w *Worker) Run(job Job) {
	for {
		hasError := false
		started := time.Now()
		err := job.Run()
		if err != nil {
			w.logger.Error("job run failed", "error", err, "job", job.GetName())
			hasError = true
		}
		if !hasError {
			finished := time.Now()
			duration := finished.Sub(started)
			w.logger.Info("job run complete", "duration", duration.String(), "job", job.GetName())
		}

		select {
		case <-w.stopChan:
			w.logger.Warn("Shutting down", "job", job.GetName())
			return
		case <-time.After(job.GetInterval()):
			// This breaks out of the select, not the for loop.
			break
		}
	}
}

// Shutdown is a graceful shutdown mechanism
func (w *Worker) Shutdown() {
	w.Stopped = true
	close(w.stopChan)
}
