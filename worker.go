package jiraworklog

// import (
// 	log "github.com/sirupsen/logrus"
// 	"time"
// )

// // Worker will do its Action once every interval, making up for lost time that
// // happened during the Action by only waiting the time left in the interval.
// type Worker struct {
// 	Stopped         bool          // A flag determining the state of the worker
// 	ShutdownChannel chan string   // A channel to communicate to the routine
// 	Interval        time.Duration // The interval with which to run the Action
// 	Action          func() error
// 	logger          *log.Entry
// }

// // NewWorker creates a new worker and instantiates all the data structures required.
// func NewWorker(interval time.Duration, logger *log.Entry, action func() error) *Worker {
// 	return &Worker{
// 		Stopped:         false,
// 		ShutdownChannel: make(chan string),
// 		Interval:        interval,
// 		Action:          action,
// 		logger:          logger,
// 	}
// }

// // Run starts the worker and listens for a shutdown call.
// func (w *Worker) Run() {
// 	// Loop that runs forever
// 	for {

// 		started := time.Now()
// 		err := w.Action()
// 		if err != nil {
// 			w.logger.WithError(err).Error("worker run failed")
// 			return
// 		}
// 		finished := time.Now()
// 		duration := finished.Sub(started)
// 		w.logger.WithField("duration", duration).Info("worker run complete")

// 		select {
// 		case <-w.ShutdownChannel:
// 			w.ShutdownChannel <- "Down"
// 			return
// 		case <-time.After(w.Interval):
// 			// This breaks out of the select, not the for loop.
// 			break
// 		}
// 	}
// }

// // Shutdown is a graceful shutdown mechanism
// func (w *Worker) Shutdown() {
// 	w.Stopped = true

// 	w.ShutdownChannel <- "Down"
// 	<-w.ShutdownChannel

// 	close(w.ShutdownChannel)
// }
