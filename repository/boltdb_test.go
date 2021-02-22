package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAllIssues(t *testing.T) {
	repo, err := NewBoltDBRepo("../bin/jira.db")
	assert.NoError(t, err)
	issues, err := repo.AllIssues()
	assert.NoError(t, err)
	assert.Greater(t, len(issues), 0)
}

func TestWorklobsPerDevWeek(t *testing.T) {
	repo, err := NewBoltDBRepo("../bin/jira.db")
	assert.NoError(t, err)
	worklogs, err := repo.WorklogsPerDevWeek()
	assert.NoError(t, err)
	assert.Greater(t, len(worklogs), 0)
}

func TestWorklobsPerDev(t *testing.T) {
	repo, err := NewBoltDBRepo("../bin/jira.db")
	assert.NoError(t, err)
	start := time.Date(2021, 2, 8, 0, 0, 0, 0, time.Local)
	stop := time.Date(2021, 2, 16, 0, 0, 0, 0, time.Local)
	worklogs, err := repo.WorklogsPerDev(start, stop)
	assert.NoError(t, err)
	assert.Greater(t, len(worklogs), 0)
}
