package main

import (
	"fmt"
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
	// Get all available roles for the dropdown
	allRoles, err := h.repo.AllRoles()
	if err != nil {
		h.logger.Error("error fetching all roles", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch roles")
	}

	// Get selected roles from query params, default to all roles if none specified
	selectedRoles := c.QueryParams()["roles"]
	if len(selectedRoles) == 0 {
		selectedRoles = allRoles
	}

	data, err := h.repo.MaitenanceRatio(selectedRoles)
	if err != nil {
		h.logger.Error("error fetching maintenance ratio", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch maintenance ratio")
	}

	if wantsHTML(c) {
		return pages.MaintenanceRatio(data, allRoles, selectedRoles).Render(c.Request().Context(), c.Response().Writer)
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

func (h *Handler) GetIssuesMissingProjectCharge(c echo.Context) error {
	issues, err := h.repo.IssuesMissingProjectCharge()
	if err != nil {
		h.logger.Error("error fetching issues missing project charge", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch issues")
	}

	if wantsHTML(c) {
		return pages.MissingProjectCharge(issues).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, issues)
}

func (h *Handler) GetCustomerBugs(c echo.Context) error {
	// Get list of projects for the dropdown
	projects, err := h.repo.CustomerBugProjects()
	if err != nil {
		h.logger.Error("error fetching customer bug projects", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	// Get selected project from query param
	selectedProject := c.QueryParam("project")

	// Get aggregate option from query param
	aggregate := c.QueryParam("aggregate") == "true"

	// Get bug data
	data, err := h.repo.CustomerBugCounts(selectedProject)
	if err != nil {
		h.logger.Error("error fetching customer bug counts", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch bug data")
	}

	if wantsHTML(c) {
		return pages.CustomerBugs(data, projects, selectedProject, aggregate).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":            data,
		"projects":        projects,
		"selectedProject": selectedProject,
		"aggregate":       aggregate,
	})
}

func (h *Handler) GetProjects(c echo.Context) error {
	projects, err := h.repo.AllProjects()
	if err != nil {
		h.logger.Error("error fetching projects", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch projects")
	}

	projectCharges, err := h.repo.AllProjectCharges()
	if err != nil {
		h.logger.Error("error fetching project charges", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project charges")
	}

	if wantsHTML(c) {
		return pages.Projects(projects, projectCharges).Render(c.Request().Context(), c.Response().Writer)
	}

	// JSON response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"projects":       projects,
		"projectCharges": projectCharges,
	})
}

func (h *Handler) CreateProject(c echo.Context) error {
	var req struct {
		Name           string   `json:"name"`
		ProjectCharges []string `json:"projectCharges"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project name is required")
	}

	if err := h.repo.CreateProject(req.Name, req.ProjectCharges); err != nil {
		h.logger.Error("error creating project", "error", err, "name", req.Name)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	h.logger.Info("created project", "name", req.Name, "charges", len(req.ProjectCharges))

	return c.JSON(http.StatusCreated, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) UpdateProject(c echo.Context) error {
	projectId := c.Param("id")
	if projectId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project id is required")
	}

	id, err := strconv.Atoi(projectId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	var req struct {
		Name           string   `json:"name"`
		Visible        bool     `json:"visible"`
		ProjectCharges []string `json:"projectCharges"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project name is required")
	}

	if err := h.repo.UpdateProject(id, req.Name, req.Visible, req.ProjectCharges); err != nil {
		h.logger.Error("error updating project", "error", err, "id", id, "name", req.Name)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project")
	}

	h.logger.Info("updated project", "id", id, "name", req.Name, "visible", req.Visible)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) DeleteProject(c echo.Context) error {
	projectId := c.Param("id")
	if projectId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project id is required")
	}

	id, err := strconv.Atoi(projectId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	if err := h.repo.DeleteProject(id); err != nil {
		h.logger.Error("error deleting project", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	h.logger.Info("deleted project", "id", id)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) GetProjectChargeHours(c echo.Context) error {
	data, err := h.repo.ProjectChargeHours()
	if err != nil {
		h.logger.Error("error fetching project charge hours", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project charge hours")
	}

	if wantsHTML(c) {
		return pages.ProjectChargeHoursPage(data).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, data)
}

// getWeekBounds returns the Monday (start) and Sunday (end) of the week
// containing the given date
func getWeekBounds(t time.Time) (time.Time, time.Time) {
	// Find Monday of the week
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is 7, not 0
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)

	// Sunday is 6 days after Monday
	sunday := monday.AddDate(0, 0, 6)
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, time.UTC)

	return monday, sunday
}

// generateWeekOptions generates the last N weeks as options
func generateWeekOptions(numWeeks int) []types.WeekOption {
	options := make([]types.WeekOption, numWeeks)
	now := time.Now()

	for i := 0; i < numWeeks; i++ {
		// Go back i weeks
		weekDate := now.AddDate(0, 0, -7*i)
		start, end := getWeekBounds(weekDate)

		label := ""
		if i == 0 {
			label = "This Week"
		} else if i == 1 {
			label = "Last Week"
		} else {
			label = start.Format("Jan 2") + " - " + end.Format("Jan 2")
		}

		options[i] = types.WeekOption{
			Offset: i,
			Label:  label,
			Start:  start,
			End:    end,
		}
	}
	return options
}

func (h *Handler) GetWeeklyHours(c echo.Context) error {
	// Get all available roles
	allRoles, err := h.repo.AllRoles()
	if err != nil {
		h.logger.Error("error fetching roles", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch roles")
	}

	// Get selected roles from query params (default to all roles)
	selectedRoles := c.QueryParams()["roles"]
	if len(selectedRoles) == 0 {
		selectedRoles = allRoles
	}

	// Get week offset from query param (default to 0 = this week)
	weekOffset := 0
	if weekParam := c.QueryParam("week"); weekParam != "" {
		weekOffset, err = strconv.Atoi(weekParam)
		if err != nil || weekOffset < 0 {
			weekOffset = 0
		}
	}

	// Generate week options (last 8 weeks)
	weekOptions := generateWeekOptions(8)

	// Calculate the date range for the selected week
	selectedWeekDate := time.Now().AddDate(0, 0, -7*weekOffset)
	startDate, endDate := getWeekBounds(selectedWeekDate)

	fmt.Printf("start: %s end: %s", startDate.String(), endDate.String())
	data, err := h.repo.DailyHoursByRole(selectedRoles, startDate, endDate)
	if err != nil {
		h.logger.Error("error fetching weekly hours", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch weekly hours")
	}

	if wantsHTML(c) {
		return pages.WeeklyHours(data, allRoles, selectedRoles, weekOptions, weekOffset).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, data)
}
