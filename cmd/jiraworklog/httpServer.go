package main

import (
	"encoding/json"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
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

	weeksBack := 7 //default to 7 days back
	weeksBackUrl, ok := r.URL.Query()["weeksBack"]
	if ok && len(weeksBackUrl) == 1 {
		i, err := strconv.Atoi(weeksBackUrl[0])
		if err != nil {
			errMsg := "Url Param 'weeksBack' must be an integer"
			s.logger.Error(errMsg)
			http.Error(w, http.StatusText(400)+":"+errMsg, 400)
			return
		}
		weeksBack = i
	}

	issues, err := s.repo.IssuesGroupedBy(group, weeksBack)
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
