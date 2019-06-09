package main

import (
	"encoding/json"
	"fmt"
	"github.com/mkobaly/jiraworklog/repository"
	"net/http"
)

type HttpServer struct {
	repo repository.Repo
}

func (s *HttpServer) GetWorkLogs(w http.ResponseWriter, r *http.Request) {
	wl, err := s.repo.AllWorkLogs()
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}

	jsn, err := json.Marshal(wl)
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}

func (s *HttpServer) GetIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := s.repo.AllIssues()
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}

	jsn, err := json.Marshal(issues)
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsn)
}
