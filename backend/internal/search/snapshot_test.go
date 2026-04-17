package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerSnapshot_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "key")
	require.NoError(t, err)

	resp, err := c.TriggerSnapshot(context.Background(), "/tmp/snap")
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.True(t, strings.HasPrefix(gotPath, "/operations/snapshot?snapshot_path="))
}

func TestTriggerSnapshot_EmptyPathRejected(t *testing.T) {
	c, err := NewClient("http://example", "key")
	require.NoError(t, err)
	_, err = c.TriggerSnapshot(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot_path is required")
}

func TestTriggerSnapshot_ServerFailureSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":false,"message":"disk full"}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "key")
	require.NoError(t, err)

	_, err = c.TriggerSnapshot(context.Background(), "/tmp/snap")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}
