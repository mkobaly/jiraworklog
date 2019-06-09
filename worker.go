package jiraworklog

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// Worker will do its Action once every interval, making up for lost time that
// happened during the Action by only waiting the time left in the interval.
type Worker struct {
	Stopped  bool // A flag determining the state of the worker
	Jobs     []Job
	logger   *log.Entry
	stopChan chan struct{}
}

type Job interface {
	Run() error
	GetInterval() time.Duration
	GetName() string
}

// NewWorker creates a new worker and instantiates all the data structures required.
func NewWorker(logger *log.Entry, jobs ...Job) *Worker {
	return &Worker{
		Stopped:  false,
		stopChan: make(chan struct{}),
		//ShutdownChannel: make(chan string),
		logger: logger,
		Jobs:   jobs,
	}
}

// Run starts the worker and listens for a shutdown call.
func (w *Worker) Start() {

	for _, job := range w.Jobs {
		go w.Run(job)
	}
}

func (w *Worker) Run(job Job) {
	for {
		started := time.Now()
		err := job.Run()
		if err != nil {
			w.logger.WithError(err).WithField("job", job.GetName()).Error("job run failed")
			return
		}
		finished := time.Now()
		duration := finished.Sub(started)
		w.logger.WithField("duration", duration).WithField("job", job.GetName()).Info("job run complete")

		select {
		case <-w.stopChan:
			w.logger.WithField("job", job.GetName()).Warn("Shutting down")
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

	//w.ShutdownChannel <- "Down"
	//<-w.ShutdownChannel

	//close(w.ShutdownChannel)
}