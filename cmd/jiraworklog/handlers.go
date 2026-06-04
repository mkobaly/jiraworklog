package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/internal/email"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/templates/pages"
	"github.com/mkobaly/jiraworklog/types"
)

type Handler struct {
	repo        repository.Repo
	jira        jiraworklog.JiraReader
	logger      *slog.Logger
	tz          *time.Location
	cfg         *jiraworklog.Config
	emailClient email.EmailClient
	loginCodes  map[string]string
	debug       bool
}

func NewHandler(r repository.Repo, j jiraworklog.JiraReader, l *slog.Logger, cfg *jiraworklog.Config, emailClient email.EmailClient, debug bool) *Handler {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return &Handler{
		repo:        r,
		jira:        j,
		logger:      l,
		tz:          nyc,
		cfg:         cfg,
		emailClient: emailClient,
		loginCodes:  make(map[string]string),
		debug:       debug,
	}
}

func (h *Handler) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.debug {
			return next(c)
		}
		cookie, err := c.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		//Cookie holds email:hash
		parts := strings.Split(cookie.Value, ":")
		hash := h.loginCodes[parts[0]]
		//hash := hashForCookie(parts[0], code)
		if hash != cookie.Value {
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		return next(c)
	}
}

// GET /login
func (h *Handler) Login(c echo.Context) error {
	view := pages.Login("")
	return render(c, http.StatusOK, view)
}

// GET /login/confirm
func (h *Handler) LoginConfirmGet(c echo.Context) error {
	email := c.QueryParam("email")
	view := pages.LoginConfirm(email, "")
	return render(c, http.StatusOK, view)
}

// POST /login
func (h *Handler) LoginPost(c echo.Context) error {
	email := c.FormValue("email")
	if email == "" {
		view := pages.Login("email is required")
		return render(c, http.StatusUnprocessableEntity, view)
	}
	if !h.validEmail(email) {
		view := pages.Login("unauthorized email address")
		return render(c, http.StatusUnprocessableEntity, view)
	}
	code := randomCode(6)
	slog.Warn("login code", slog.String("email", email), slog.String("code", code))
	h.loginCodes[email] = code
	err := h.emailClient.SendEmail("Jira Manager - Login Code", email, code)
	if err != nil {
		slog.Error("unable to send email", slog.Any("err", err))
		view := pages.Login("unable to generate code. Please try again later")
		return render(c, http.StatusUnprocessableEntity, view)
	}
	return c.Redirect(http.StatusSeeOther, "/login/confirm?email="+url.QueryEscape(email))
}

// POST /login/confirm - Will login the user and create cookie
func (h *Handler) LoginConfirm(c echo.Context) error {
	code := c.FormValue("code")
	email := c.FormValue("email")
	if code == "" || email == "" {
		view := pages.LoginConfirm(email, "unknown error")
		return render(c, http.StatusUnprocessableEntity, view)
	}
	val := h.loginCodes[email]
	if val == "" || !strings.EqualFold(val, code) {
		view := pages.LoginConfirm(email, "invalid code")
		return render(c, http.StatusUnprocessableEntity, view)
	}
	//delete(ws.loginCodes, email)
	hash := hashForCookie(email, val)
	h.loginCodes[email] = hash

	cookie := new(http.Cookie)
	cookie.Name = "session_id"
	cookie.Value = hash
	cookie.Expires = time.Now().Add(24 * 30 * time.Hour)
	cookie.HttpOnly = true
	if h.cfg.HTTPSecureCookie {
		cookie.Secure = true
	}
	cookie.Path = "/"
	c.SetCookie(cookie)
	return c.Redirect(http.StatusFound, "/")
}

// ── Project charge hours filter helpers ──────────────────────────────────────

func distinctChargeLabels(data []types.ProjectChargeHours) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range data {
		if !seen[item.ProjectCharge] {
			seen[item.ProjectCharge] = true
			out = append(out, item.ProjectCharge)
		}
	}
	sort.Strings(out)
	return out
}

func distinctChargeRoles(data []types.ProjectChargeHours) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range data {
		r := "Unknown"
		if item.Role.Valid && item.Role.String != "" {
			r = item.Role.String
		}
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	sort.Strings(out)
	return out
}

func distinctChargeLocations(data []types.ProjectChargeHours) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range data {
		l := "Unknown"
		if item.Location.Valid && item.Location.String != "" {
			l = item.Location.String
		}
		if !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	sort.Strings(out)
	return out
}

func applyChargeFilters(data []types.ProjectChargeHours, charges, roles, locations, empTypes []string) []types.ProjectChargeHours {
	if len(charges) == 0 && len(roles) == 0 && len(locations) == 0 && len(empTypes) == 0 {
		return data
	}
	chargeSet := stringSet(charges)
	roleSet := stringSet(roles)
	locSet := stringSet(locations)
	empSet := stringSet(empTypes)

	var out []types.ProjectChargeHours
	for _, item := range data {
		if len(chargeSet) > 0 && !chargeSet[item.ProjectCharge] {
			continue
		}
		role := "Unknown"
		if item.Role.Valid && item.Role.String != "" {
			role = item.Role.String
		}
		if len(roleSet) > 0 && !roleSet[role] {
			continue
		}
		loc := "Unknown"
		if item.Location.Valid && item.Location.String != "" {
			loc = item.Location.String
		}
		if len(locSet) > 0 && !locSet[loc] {
			continue
		}
		if len(empSet) > 0 {
			et := "contractor"
			if item.IsEmployee {
				et = "employee"
			}
			if !empSet[et] {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func stringSet(vals []string) map[string]bool {
	s := make(map[string]bool, len(vals))
	for _, v := range vals {
		s[v] = true
	}
	return s
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
	// Fetch maintenance ratio data (all roles)
	allRoles, _ := h.repo.AllRoles()
	maintenanceData, err := h.repo.MaitenanceRatio(allRoles)
	if err != nil {
		h.logger.Error("error fetching maintenance ratio for dashboard", "error", err)
		maintenanceData = nil
	}

	// Get the most recent month's data
	var latestMaintenance *types.MaitenanceRatio
	if len(maintenanceData) > 0 {
		latestMaintenance = &maintenanceData[len(maintenanceData)-1]
	}

	// Fetch missing project charges count
	missingCharges, err := h.repo.IssuesMissingProjectCharge(h.cfg.QueryExcludedProjects)
	if err != nil {
		h.logger.Error("error fetching missing charges for dashboard", "error", err)
		missingCharges = nil
	}
	missingChargesCount := len(missingCharges)

	// Fetch mismatched project charges count
	mismatchedCharges, err := h.repo.IssuesMismatchedProjectCharge(h.cfg.QueryExcludedProjects)
	if err != nil {
		h.logger.Error("error fetching mismatched charges for dashboard", "error", err)
		mismatchedCharges = nil
	}
	mismatchedChargesCount := len(mismatchedCharges)

	// Fetch people without roles count
	people, err := h.repo.People()
	if err != nil {
		h.logger.Error("error fetching people for dashboard", "error", err)
		people = nil
	}
	peopleMissingRolesCount := 0
	for _, p := range people {
		role := p.GetRole()
		if role == "" || role == "UNKNOWN" {
			peopleMissingRolesCount++
		}
	}

	// Check Jira connectivity
	jiraConnected := false
	jiraDisplayName := ""
	jiraUser, err := h.jira.GetJiraUser()
	if err != nil {
		h.logger.Error("jira connection check failed", "error", err)
	} else {
		jiraConnected = true
		jiraDisplayName = jiraUser.DisplayName
	}

	return pages.Dashboard(latestMaintenance, missingChargesCount, mismatchedChargesCount, peopleMissingRolesCount, jiraConnected, jiraDisplayName).Render(c.Request().Context(), c.Response().Writer)
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

func (h *Handler) UpdatePerson(c echo.Context) error {
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
		Role       string `json:"role"`
		IsEmployee bool   `json:"isEmployee"`
		Location   string `json:"location"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "role is required")
	}

	// Update the person
	if err := h.repo.UpdatePerson(id, req.Role, req.IsEmployee, req.Location); err != nil {
		h.logger.Error("error updating person", "error", err, "personId", id, "role", req.Role, "isEmployee", req.IsEmployee, "location", req.Location)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update person")
	}

	h.logger.Info("updated person", "personId", id, "role", req.Role, "isEmployee", req.IsEmployee, "location", req.Location)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"role":       req.Role,
		"isEmployee": req.IsEmployee,
		"location":   req.Location,
	})
}

func (h *Handler) GetIssuesMissingProjectCharge(c echo.Context) error {
	issues, err := h.repo.IssuesMissingProjectCharge(h.cfg.QueryExcludedProjects)
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

func (h *Handler) GetIssuesMismatchedProjectCharge(c echo.Context) error {
	issues, err := h.repo.IssuesMismatchedProjectCharge(h.cfg.QueryExcludedProjects)
	if err != nil {
		h.logger.Error("error fetching issues with mismatched project charge", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch issues")
	}

	if wantsHTML(c) {
		return pages.MismatchedProjectCharge(issues).Render(c.Request().Context(), c.Response().Writer)
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
	//aggregate := c.QueryParam("aggregate") == "true"

	// Get bug data
	// data, err := h.repo.CustomerBugCounts(selectedProject)
	// if err != nil {
	// 	h.logger.Error("error fetching customer bug counts", "error", err)
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch bug data")
	// }

	trends, err := h.repo.CustomerBugTrends(selectedProject)
	if err != nil {
		h.logger.Error("error fetching customer bug trends", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch bug trends")
	}

	return pages.CustomerBugs(trends, projects, selectedProject).Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) GetProjectCharges(c echo.Context) error {
	charges, err := h.repo.ProjectCharges()
	if err != nil {
		h.logger.Error("error fetching project charges", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project charges")
	}

	if wantsHTML(c) {
		return pages.ProjectCharges(charges).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, charges)
}

func (h *Handler) UpdateProjectCharge(c echo.Context) error {
	var req struct {
		Name    string `json:"name"`
		Visible bool   `json:"visible"`
		Label   string `json:"label"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	if err := h.repo.UpdateProjectCharge(req.Name, req.Visible, req.Label); err != nil {
		h.logger.Error("error updating project charge", "error", err, "name", req.Name)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project charge")
	}

	h.logger.Info("updated project charge", "name", req.Name, "visible", req.Visible)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetProjectChargeHours(c echo.Context) error {
	fromYearMonth := c.QueryParam("from")
	toYearMonth := c.QueryParam("to")

	data, err := h.repo.ProjectChargeHours(fromYearMonth, toYearMonth, h.cfg.QueryExcludedProjects)
	if err != nil {
		h.logger.Error("error fetching project charge hours", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project charge hours")
	}

	// Derive available options from the full (date-filtered) dataset
	allCharges := distinctChargeLabels(data)
	allRoles := distinctChargeRoles(data)
	allLocations := distinctChargeLocations(data)

	// Parse multi-value filter params
	selectedCharges := c.QueryParams()["charge"]
	selectedRoles := c.QueryParams()["role"]
	selectedLocations := c.QueryParams()["location"]
	selectedEmpTypes := c.QueryParams()["empType"]

	// Apply in-memory filters
	filtered := applyChargeFilters(data, selectedCharges, selectedRoles, selectedLocations, selectedEmpTypes)

	// Get grouping options from query params.
	// groupByCharge and groupByRole default to true on initial load (no _f sentinel).
	formSubmitted := c.QueryParam("_f") == "1"
	groupByDate := c.QueryParam("groupByDate") == "true"
	groupByCharge := !formSubmitted || c.QueryParam("groupByCharge") == "true"
	groupByEmployee := c.QueryParam("groupByEmployee") == "true"
	groupByRole := !formSubmitted || c.QueryParam("groupByRole") == "true"
	groupByLocation := c.QueryParam("groupByLocation") == "true"

	filterState := pages.ChargeFilterState{
		FromMonth:    fromYearMonth,
		ToMonth:      toYearMonth,
		Charges:      selectedCharges,
		Roles:        selectedRoles,
		Locations:    selectedLocations,
		EmpTypes:     selectedEmpTypes,
		AllCharges:   allCharges,
		AllRoles:     allRoles,
		AllLocations: allLocations,
	}

	if wantsHTML(c) {
		return pages.ProjectChargeHoursPage(filtered, filterState, groupByDate, groupByEmployee, groupByRole, groupByCharge, groupByLocation).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, filtered)
}

func (h *Handler) GetProjectChargeHoursCSV(c echo.Context) error {
	fromMonth := c.QueryParam("from")
	toMonth := c.QueryParam("to")
	data, err := h.repo.ProjectChargeHours(fromMonth, toMonth, h.cfg.QueryExcludedProjects)
	if err != nil {
		h.logger.Error("error fetching project charge hours", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project charge hours")
	}

	// Get grouping options from query params
	groupByDate := c.QueryParam("groupByDate") == "true"
	groupByCharge := c.QueryParam("groupByCharge") != "false" // Default to true
	groupByEmployee := c.QueryParam("groupByEmployee") == "true"
	groupByRole := c.QueryParam("groupByRole") != "false" // Default to true
	groupByLocation := c.QueryParam("groupByLocation") == "true"

	// Build CSV content
	var csv strings.Builder

	// Write header
	csv.WriteString("Project")
	if groupByCharge {
		csv.WriteString(",Project Charge,Category,Type")
	}
	if groupByDate {
		csv.WriteString(",Month")
	}
	if groupByEmployee {
		csv.WriteString(",Employee Type")
	}
	if groupByRole {
		csv.WriteString(",Role")
	}
	if groupByLocation {
		csv.WriteString(",Location")
	}
	csv.WriteString(",Hours\n")

	// Group and aggregate data based on options
	type rowKey struct {
		Project       string
		ProjectCharge string
		Category      string
		Type          string
		YearMonth     string
		IsEmployee    bool
		Role          string
		Location      string
	}
	aggregated := make(map[rowKey]float64)

	for _, item := range data {
		role := "Unknown"
		if item.Role.Valid && item.Role.String != "" {
			role = item.Role.String
		}

		location := "Unknown"
		if item.Location.Valid && item.Location.String != "" {
			location = item.Location.String
		}

		key := rowKey{
			Project: item.Label,
		}
		if groupByCharge {
			key.ProjectCharge = item.ProjectCharge
			key.Category = item.Category()
			key.Type = item.Type()
		}
		if groupByDate {
			key.YearMonth = item.YearMonth
		}
		if groupByEmployee {
			key.IsEmployee = item.IsEmployee
		}
		if groupByRole {
			key.Role = role
		}
		if groupByLocation {
			key.Location = location
		}

		aggregated[key] += item.Hours
	}

	// Write rows
	for key, hours := range aggregated {
		csv.WriteString(fmt.Sprintf("%q", key.Project))
		if groupByCharge {
			fmt.Fprintf(&csv, ",%q,%q,%q", key.ProjectCharge, key.Category, key.Type)
		}
		if groupByDate {
			csv.WriteString(fmt.Sprintf(",%q", key.YearMonth))
		}
		if groupByEmployee {
			employeeType := "Contractor"
			if key.IsEmployee {
				employeeType = "Employee"
			}
			csv.WriteString(fmt.Sprintf(",%q", employeeType))
		}
		if groupByRole {
			csv.WriteString(fmt.Sprintf(",%q", key.Role))
		}
		if groupByLocation {
			csv.WriteString(fmt.Sprintf(",%q", key.Location))
		}
		csv.WriteString(fmt.Sprintf(",%.2f\n", hours))
	}

	// Set headers for CSV download
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=project-charge-hours.csv")

	return c.String(http.StatusOK, csv.String())
}

// getWeekBounds returns the Monday (start) and Monday (end) of the week
// containing the given date
func getWeekBounds(t time.Time, tz *time.Location) (time.Time, time.Time) {
	// Find Monday of the week
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is 7, not 0
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, tz)

	// Sunday is 6 days after Monday
	nextMonday := monday.AddDate(0, 0, 7)
	nextMonday = time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, tz)

	sunday := monday.AddDate(0, 0, 6)
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 999_999_999, tz)

	return monday, sunday
}

// generateWeekOptions generates the last N weeks as options
func generateWeekOptions(numWeeks int, tz *time.Location) []types.WeekOption {
	options := make([]types.WeekOption, numWeeks)
	now := time.Now()

	for i := 0; i < numWeeks; i++ {
		// Go back i weeks
		weekDate := now.AddDate(0, 0, -7*i)
		start, end := getWeekBounds(weekDate, tz)
		///end = end.Add(time.Second * -1)

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
	weekOptions := generateWeekOptions(8, h.tz)

	// Calculate the date range for the selected week
	selectedWeekDate := time.Now().AddDate(0, 0, -7*weekOffset)
	startDate, endDate := getWeekBounds(selectedWeekDate, h.tz)

	//fmt.Printf("start: %s end: %s\n", startDate.String(), endDate.String())
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

func (h *Handler) GetProjectTimeTracking(c echo.Context) error {
	// Get fixed version from query param
	fixedVersion := c.QueryParam("version")

	var data []types.ProjectTimeTracking
	var projectName string
	var err error

	if fixedVersion != "" {
		data, err = h.repo.ProjectTimeTracking(fixedVersion)
		if err != nil {
			h.logger.Error("error fetching project time tracking", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch time tracking data")
		}
		projectName, _ = h.repo.ProjectName(fixedVersion)
	}

	if wantsHTML(c) {
		return pages.ProjectTimeTracking(data, fixedVersion, projectName).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, data)
}

func (h *Handler) GetProjectKPIs(c echo.Context) error {
	epicOrVersion := c.QueryParam("project")

	var data types.ProjectKPIData
	var projectName string
	var err error

	if epicOrVersion != "" {
		data, err = h.repo.ProjectKPIs(epicOrVersion)
		if err != nil {
			h.logger.Error("error fetching project KPIs", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch project KPIs")
		}
		projectName, _ = h.repo.ProjectName(epicOrVersion)
	}

	if wantsHTML(c) {
		return pages.ProjectKPIs(data, epicOrVersion, projectName).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, data)
}

// GET /dashboard/leadership
func (h *Handler) GetLeadershipDashboard(c echo.Context) error {
	// v1: hard-coded team list. v2 will pull from a teams table or projectCharge.
	teams := []string{"IDM", "SYM", "ESG"}

	rows, err := h.repo.MonthlyTeamMetrics(c.Request().Context(), teams, "", "")
	if err != nil {
		h.logger.Error("error fetching monthly team metrics", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch monthly team metrics")
	}

	// Build month-options dropdown from the rows we received (sorted unique
	// year_month values, descending so the most recent is on top).
	monthSet := map[string]struct{}{}
	for _, r := range rows {
		monthSet[r.YearMonth] = struct{}{}
	}
	monthOptions := make([]string, 0, len(monthSet))
	for m := range monthSet {
		monthOptions = append(monthOptions, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(monthOptions)))

	// Default to last complete month (highest year_month in the result).
	selectedMonth := c.QueryParam("month")
	if selectedMonth == "" && len(monthOptions) > 0 {
		selectedMonth = monthOptions[0]
	}

	if wantsHTML(c) {
		return pages.LeadershipDashboard(rows, teams, selectedMonth, monthOptions).
			Render(c.Request().Context(), c.Response().Writer)
	}
	return c.JSON(http.StatusOK, rows)
}

// GET /dashboard/manager
func (h *Handler) GetManagerDashboard(c echo.Context) error {
	// v1: hard-coded team list matches the leadership page.
	teams := []string{"IDM", "SYM", "ESG"}

	team := c.QueryParam("team")
	if team == "" {
		team = teams[0] // default to first team alphabetically
	} else {
		// Validate against the allowlist so an unknown team can't trigger 8
		// expensive empty-result queries and silently render a normal-looking
		// page of zeros.
		valid := false
		for _, t := range teams {
			if t == team {
				valid = true
				break
			}
		}
		if !valid {
			return echo.NewHTTPError(http.StatusBadRequest, "unknown team")
		}
	}

	data, err := h.repo.ManagerMetrics(c.Request().Context(), team, "", "")
	if err != nil {
		h.logger.Error("error fetching manager metrics", "error", err, "team", team)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch manager metrics")
	}

	if wantsHTML(c) {
		return pages.ManagerDashboard(data, teams).
			Render(c.Request().Context(), c.Response().Writer)
	}
	return c.JSON(http.StatusOK, data)
}

func (h *Handler) validEmail(email string) bool {
	for _, v := range h.cfg.AuthorizedUsers {
		parts := strings.Split(v, "@")
		//allow wildcard match by domain: *@example.com
		if parts[0] == "*" {
			userParts := strings.Split(email, "@")
			if len(userParts) == 2 && len(parts) == 2 && strings.EqualFold(parts[1], userParts[1]) {
				return true
			}
		}
		if strings.EqualFold(email, v) {
			return true
		}
	}
	return false
}

func render(c echo.Context, status int, component templ.Component) error {
	c.Response().WriteHeader(status)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func hashForCookie(email, code string) string {
	hash := sha256.Sum256([]byte(email + code))
	return fmt.Sprintf("%s:%s", email, base64.URLEncoding.EncodeToString(hash[:]))
}
