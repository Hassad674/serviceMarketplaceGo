package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags_HappyPath_AllOrgs(t *testing.T) {
	rc, err := parseFlags([]string{"-year=2026", "-month=4"})
	require.NoError(t, err)
	assert.Equal(t, 2026, rc.year)
	assert.Equal(t, 4, rc.month)
	assert.Equal(t, "all", rc.orgArg)
	assert.False(t, rc.dryRun)
}

func TestParseFlags_HappyPath_SingleOrg_DryRun(t *testing.T) {
	id := uuid.New().String()
	rc, err := parseFlags([]string{"-year=2026", "-month=12", "-org=" + id, "-dry-run"})
	require.NoError(t, err)
	assert.Equal(t, 12, rc.month)
	assert.Equal(t, id, rc.orgArg)
	assert.True(t, rc.dryRun)
}

func TestParseFlags_RejectsMissingYear(t *testing.T) {
	_, err := parseFlags([]string{"-month=4"})
	require.Error(t, err)
}

func TestParseFlags_RejectsOutOfRangeMonth(t *testing.T) {
	for _, m := range []string{"0", "13", "-1"} {
		_, err := parseFlags([]string{"-year=2026", "-month=" + m})
		require.Error(t, err, "month=%s must be rejected", m)
	}
}

func TestParseFlags_RejectsBadOrgUUID(t *testing.T) {
	_, err := parseFlags([]string{"-year=2026", "-month=4", "-org=not-a-uuid"})
	require.Error(t, err)
}

func TestProcessOrgResult_StringFormat(t *testing.T) {
	id := uuid.New()
	r := processOrgResult{
		orgID:    id,
		records:  3,
		feeCents: 12345,
		result:   resultIssued,
	}
	assert.Equal(t,
		"org="+id.String()+" records=3 fee=12345 result=issued",
		r.String(),
	)

	rErr := processOrgResult{
		orgID:    id,
		records:  0,
		feeCents: 0,
		result:   resultError,
		errMsg:   "boom",
	}
	assert.Contains(t, rErr.String(), "result=error error=boom")
}
