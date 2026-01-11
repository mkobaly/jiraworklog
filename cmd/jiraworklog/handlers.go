package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/templates/pages"
	"github.com/mkobaly/jiraworklog/types"
)

type Handler struct {
	repo   repository.Repo
	logger *slog.Logger
}

func NewHandler(r repository.Repo, l *slog.Logger) *Handler {
	return &Handler{
		repo:   r,
		logger: l,
	}
}

// wantsHTML checks if the client prefers HTML over JSON
func wantsHTML(c echo.Context) bool {
	accept := c.Request().Header.Get("Accept")
	// Check for explicit JSON request
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return false
	}
	// Default to HTML for browsers
	return strings.Contains(accept, "text/html") || accept == "" || accept == "*/*"
}

func (h *Handler) Dashboard(c echo.Context) error {
	return pages.Dashboard().Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) GetWorkLogs(c echo.Context) error {
	wl, err := h.repo.AllWorkLogs()
	if err != nil {
		h.logger.Error("error fetching all worklogs", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch worklogs")
	}

	if wantsHTML(c) {
		return pages.Worklogs(wl).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, wl)
}

func (h *Handler) GetIssues(c echo.Context) error {
	issues, err := h.repo.AllIssues()
	if err != nil {
		h.logger.Error("error fetching all issues", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch issues")
	}

	if wantsHTML(c) {
		return pages.Issues(issues).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, issues)
}

func (h *Handler) GetIssuesGroupedBy(c echo.Context) error {
	group := c.QueryParam("group")
	if group == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter 'group' is required")
	}

	group, err := types.ValidateGroupBy(group)
	if err != nil {
		h.logger.Error("invalid group by value", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Default date range: last 7 days to tomorrow
	y, m, d := time.Now().AddDate(0, 0, -7).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	stop := time.Date(y, m, d+8, 0, 0, 0, 0, time.Local)

	// Parse start date if provided
	if startParam := c.QueryParam("start"); startParam != "" {
		start, err = time.Parse("20060102", startParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'start' must be a date in format YYYYMMDD")
		}
	}

	// Parse stop date if provided
	if stopParam := c.QueryParam("stop"); stopParam != "" {
		stop, err = time.Parse("20060102", stopParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'stop' must be a date in format YYYYMMDD")
		}
	}

	issues, err := h.repo.IssuesGroupedBy(group, start, stop)
	if err != nil {
		h.logger.Error("error fetching records", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch grouped issues")
	}

	if wantsHTML(c) {
		return pages.IssuesGroupBy(group, issues).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, issues)
}

func (h *Handler) GetIssueAccuracy(c echo.Context) error {
	// Default date range: last 7 days to tomorrow
	y, m, d := time.Now().AddDate(0, 0, -7).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	stop := time.Date(y, m, d+8, 0, 0, 0, 0, time.Local)

	var err error
	// Parse start date if provided
	if startParam := c.QueryParam("start"); startParam != "" {
		start, err = time.Parse("20060102", startParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'start' must be a date in format YYYYMMDD")
		}
	}

	// Parse stop date if provided
	if stopParam := c.QueryParam("stop"); stopParam != "" {
		stop, err = time.Parse("20060102", stopParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'stop' must be a date in format YYYYMMDD")
		}
	}

	issues, err := h.repo.IssueAccuracy(start, stop)
	if err != nil {
		h.logger.Error("error fetching records", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch issue accuracy")
	}

	if wantsHTML(c) {
		return pages.IssueAccuracy(issues).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, issues)
}

func (h *Handler) GetWorklogsGroupBy(c echo.Context) error {
	group := c.QueryParam("group")
	if group == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter 'group' is required")
	}

	group, err := types.ValidateWorklogsGroupBy(group)
	if err != nil {
		h.logger.Error("invalid group by value", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Default date range: last 7 days to tomorrow
	y, m, d := time.Now().AddDate(0, 0, -7).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	stop := time.Date(y, m, d+8, 0, 0, 0, 0, time.Local)

	// Parse start date if provided
	if startParam := c.QueryParam("start"); startParam != "" {
		start, err = time.Parse("20060102", startParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'start' must be a date in format YYYYMMDD")
		}
	}

	// Parse stop date if provided
	if stopParam := c.QueryParam("stop"); stopParam != "" {
		stop, err = time.Parse("20060102", stopParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'stop' must be a date in format YYYYMMDD")
		}
	}

	wl, err := h.repo.WorklogsGroupBy(group, start, stop)
	if err != nil {
		h.logger.Error("error fetching worklogs grouped by", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch grouped worklogs")
	}

	if wantsHTML(c) {
		// Map the field name back to a display name
		displayGroup := group
		if group == "ParentIssueType" {
			displayGroup = "Type"
		} else if group == "ParentIssuePriority" {
			displayGroup = "Priority"
		}
		return pages.WorklogsGroupBy(displayGroup, wl).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, wl)
}

func (h *Handler) GetWorklogsPerDev(c echo.Context) error {
	// Default date range: last 7 days to tomorrow
	y, m, d := time.Now().AddDate(0, 0, -7).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	stop := time.Date(y, m, d+8, 0, 0, 0, 0, time.Local)

	var err error
	// Parse start date if provided
	if startParam := c.QueryParam("start"); startParam != "" {
		start, err = time.Parse("20060102", startParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'start' must be a date in format YYYYMMDD")
		}
	}

	// Parse stop date if provided
	if stopParam := c.QueryParam("stop"); stopParam != "" {
		stop, err = time.Parse("20060102", stopParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parameter 'stop' must be a date in format YYYYMMDD")
		}
	}

	wl, err := h.repo.WorklogsPerDev(start, stop)
	if err != nil {
		h.logger.Error("error fetching worklogs per dev", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch worklogs per dev")
	}

	if wantsHTML(c) {
		return pages.WorklogsPerDev(wl).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, wl)
}

func (h *Handler) GetWorklogsPerDevWeek(c echo.Context) error {
	wl, err := h.repo.WorklogsPerDevWeek()
	if err != nil {
		h.logger.Error("error fetching worklogs per dev week", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch worklogs per dev week")
	}

	if wantsHTML(c) {
		return pages.WorklogsPerDevWeek(wl).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, wl)
}

func (h *Handler) GetMaintenanceRatio(c echo.Context) error {
	//hard coded roles
	roles := []string{"qa", "dev"}
	data, err := h.repo.MaitenanceRatio(roles)
	if err != nil {
		h.logger.Error("error fetching maintenance ratio", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch maintenance ratio")
	}

	if wantsHTML(c) {
		return pages.MaintenanceRatio(data).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, data)
}

func (h *Handler) GetPeople(c echo.Context) error {
	people, err := h.repo.People()
	if err != nil {
		h.logger.Error("error fetching people", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch people")
	}

	roles, err := h.repo.AllRoles()
	if err != nil {
		h.logger.Error("error fetching roles", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch roles")
	}

	if wantsHTML(c) {
		return pages.People(people, roles).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"people": people,
		"roles":  roles,
	})
}

func (h *Handler) UpdatePersonRole(c echo.Context) error {
	personId := c.Param("id")
	if personId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "person id is required")
	}

	// Parse person ID
	id, err := strconv.Atoi(personId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid person id")
	}

	// Parse request body
	var req struct {
		Role string `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "role is required")
	}

	// Update the role
	if err := h.repo.UpdatePersonRole(id, req.Role); err != nil {
		h.logger.Error("error updating person role", "error", err, "personId", id, "role", req.Role)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update role")
	}

	h.logger.Info("updated person role", "personId", id, "role", req.Role)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"role":   req.Role,
	})
}
