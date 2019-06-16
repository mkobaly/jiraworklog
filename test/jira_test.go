package test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWorklogsUpdated(t *testing.T) {
	fj := &FakeJira{}
	wl, err := fj.WorklogsUpdated(12121212)
	require.NoError(t, err)
	require.Equal(t, 3, len(wl.Values))
}

func TestGetIssueWihoutAggTimesDefaultsToZero(t *testing.T) {
	fj := &FakeJira{}
	issue, err := fj.Issue("6003")
	require.NoError(t, err)
	require.Equal(t, 0, issue.Fields.Aggregatetimespent)
	require.Equal(t, 0, issue.Fields.Aggregatetimeoriginalestimate)
}
