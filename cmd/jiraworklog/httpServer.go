package main

import (
	"encoding/json"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func NewHttpServer(r repository.Repo, l *logrus.Entry) *HttpServer {
	return &HttpServer{
		repo:   r,
		logger: l,
	}
}

type HttpServer struct {
	repo   repository.Repo
	logger *logrus.Entry
}

func (s *HttpServer) GetWorkLogs(w http.ResponseWriter, r *http.Request) {
	wl, err := s.repo.AllWorkLogs()
	if err != nil {
		s.logger.WithError(err).Error("error fetching all worklogs")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(wl)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling all worklogs")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := s.repo.AllIssues()
	if err != nil {
		s.logger.WithError(err).Error("error fetching all issues")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(issues)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling all issues")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetIssuesGroupedBy(w http.ResponseWriter, r *http.Request) {
	groupUrl, ok := r.URL.Query()["group"]
	if !ok || len(groupUrl[0]) < 1 {
		errMsg := "Url Param 'group' is required"
		s.logger.Error(errMsg)
		http.Error(w, http.StatusText(400)+":"+errMsg, 400)
		return
	}
	group := groupUrl[0]
	group, err := types.ValidateGroupBy(group)
	if err != nil {

		s.logger.WithError(err).Error("invalid group by value")
		http.Error(w, http.StatusText(400)+":"+err.Error(), 400)
		return
	}
	y, m, d := time.Now().AddDate(0, 0, -7).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	stop := time.Date(y, m, d+8, 0, 0, 0, 0, time.Local)

	startParam, ok := r.URL.Query()["start"]
	if ok && len(startParam) == 1 {
		start, err = time.Parse("20060102", startParam[0])
		if err != nil {
			errMsg := "Url Param 'start' must be a date in format YYYYMMDD"
			s.logger.Error(errMsg)
			http.Error(w, http.StatusText(400)+":"+errMsg, 400)
			return
		}
	}

	stopParam, ok := r.URL.Query()["stop"]
	if ok && len(stopParam) == 1 {
		stop, err = time.Parse("20060102", stopParam[0])
		if err != nil {
			errMsg := "Url Param 'stop' must be a date in format YYYYMMDD"
			s.logger.Error(errMsg)
			http.Error(w, http.StatusText(400)+":"+errMsg, 400)
			return
		}
	}

	issues, err := s.repo.IssuesGroupedBy(group, start, stop)
	if err != nil {
		s.logger.WithError(err).Error("error fetching records")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(issues)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling results")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetIssueAccuracy(w http.ResponseWriter, r *http.Request) {

	y, m, d := time.Now().AddDate(0, 0, -7).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	stop := time.Date(y, m, d+8, 0, 0, 0, 0, time.Local)

	startParam, ok := r.URL.Query()["start"]
	if ok && len(startParam) == 1 {
		st, err := time.Parse("20060102", startParam[0])
		if err != nil {
			errMsg := "Url Param 'start' must be a date in format YYYYMMDD"
			s.logger.Error(errMsg)
			http.Error(w, http.StatusText(400)+":"+errMsg, 400)
			return
		}
		start = st
	}

	stopParam, ok := r.URL.Query()["stop"]
	if ok && len(stopParam) == 1 {
		st, err := time.Parse("20060102", stopParam[0])
		if err != nil {
			errMsg := "Url Param 'stop' must be a date in format YYYYMMDD"
			s.logger.Error(errMsg)
			http.Error(w, http.StatusText(400)+":"+errMsg, 400)
			return
		}
		stop = st
	}

	issues, err := s.repo.IssueAccuracy(start, stop)
	if err != nil {
		s.logger.WithError(err).Error("error fetching records")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(issues)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling results")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetWorklogsGroupBy(w http.ResponseWriter, r *http.Request) {
	groupUrl, ok := r.URL.Query()["group"]
	if !ok || len(groupUrl[0]) < 1 {
		errMsg := "Url Param 'group' is required"
		s.logger.Error(errMsg)
		http.Error(w, http.StatusText(400)+":"+errMsg, 400)
		return
	}
	group := groupUrl[0]
	group, err := types.ValidateWorklogsGroupBy(group)
	if err != nil {

		s.logger.WithError(err).Error("invalid group by value")
		http.Error(w, http.StatusText(400)+":"+err.Error(), 400)
		return
	}

	wl, err := s.repo.WorklogsGroupBy(group)
	if err != nil {
		s.logger.WithError(err).Error("error fetching worklogs grouped by")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(wl)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling worklogs grouped by")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetWorklogsPerDay(w http.ResponseWriter, r *http.Request) {
	wl, err := s.repo.WorklogsPerDay()
	if err != nil {
		s.logger.WithError(err).Error("error fetching worklogs per day")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(wl)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling worklogs per day")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetWorklogsPerDevDay(w http.ResponseWriter, r *http.Request) {
	wl, err := s.repo.WorklogsPerDevDay()
	if err != nil {
		s.logger.WithError(err).Error("error fetching worklogs per dev day")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(wl)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling worklogs per dev day")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetWorklogsPerDevWeek(w http.ResponseWriter, r *http.Request) {
	wl, err := s.repo.WorklogsPerDevWeek()
	if err != nil {
		s.logger.WithError(err).Error("error fetching worklogs per dev week")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	jsn, err := json.Marshal(wl)
	if err != nil {
		s.logger.WithError(err).Error("error marshalling worklogs per dev week")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

type ChartData struct {
	Issue          string
	NonResolved    int
	Resolved       int
	DaysToComplete int
	TimeSpent      float64
	TimeEstimate   float64
}
