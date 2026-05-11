package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags_HappyPath(t *testing.T) {
	rc, err := parseFlags([]string{"-since=2026-01-01"})
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), rc.since)
	assert.False(t, rc.dryRun)
}

func TestParseFlags_DryRun(t *testing.T) {
	rc, err := parseFlags([]string{"-since=2026-04-15", "-dry-run"})
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), rc.since)
	assert.True(t, rc.dryRun)
}

func TestParseFlags_RejectsMissingSince(t *testing.T) {
	_, err := parseFlags([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "since")
}

func TestParseFlags_RejectsInvalidSince(t *testing.T) {
	_, err := parseFlags([]string{"-since=not-a-date"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid -since")
}
