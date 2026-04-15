package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_Health(t *testing.T) {
	h := &HealthHandler{} // db not needed for /health

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

// fakeSearchPinger is a local stand-in for the Typesense client's
// Ping method. The test file lives in the handler package so we
// can assign to unexported fields directly via struct construction.
type fakeSearchPinger struct {
	err error
}

func (f *fakeSearchPinger) Ping(_ context.Context) error { return f.err }

// newReadyTestDB returns a *sql.DB backed by sqlmock with ping
// monitoring enabled so /ready's db.PingContext call hits a real
// ExpectedPing expectation.
func newReadyTestDB(t *testing.T, pingErr error) *sql.DB {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expect := mock.ExpectPing()
	if pingErr != nil {
		expect.WillReturnError(pingErr)
	}
	return db
}

func TestHealthHandler_Ready_NoSearchPinger(t *testing.T) {
	db := newReadyTestDB(t, nil)
	h := NewHealthHandler(db)

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthHandler_Ready_DatabaseDown(t *testing.T) {
	db := newReadyTestDB(t, errors.New("db dead"))
	h := NewHealthHandler(db)

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealthHandler_Ready_SearchNotRequired(t *testing.T) {
	// SEARCH_ENGINE=sql: Typesense failure must not take /ready red.
	db := newReadyTestDB(t, nil)
	h := NewHealthHandler(db).WithSearchPinger(&fakeSearchPinger{err: errors.New("ts down")}, false)

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusOK, rec.Code, "typesense optional: ts failure is non-fatal")
}

func TestHealthHandler_Ready_SearchRequired(t *testing.T) {
	// SEARCH_ENGINE=typesense: Typesense failure = 503.
	db := newReadyTestDB(t, nil)
	h := NewHealthHandler(db).WithSearchPinger(&fakeSearchPinger{err: errors.New("ts down")}, true)

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"typesense required: ts failure must take /ready red")
}

func TestHealthHandler_Ready_SearchRequiredAndHealthy(t *testing.T) {
	db := newReadyTestDB(t, nil)
	h := NewHealthHandler(db).WithSearchPinger(&fakeSearchPinger{}, true)

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}
