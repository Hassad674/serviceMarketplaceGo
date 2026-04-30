package searchanalytics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	mu          sync.Mutex
	rows        []*SearchRow
	insertErr   error
	clickErr    error
	lastClicked struct {
		SearchID string
		DocID    string
		Position int
		At       time.Time
	}
}

func (f *fakeRepo) InsertSearch(_ context.Context, row *SearchRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.insertErr != nil {
		return f.insertErr
	}
	f.rows = append(f.rows, row)
	return nil
}

func (f *fakeRepo) RecordClick(_ context.Context, searchID, docID string, pos int, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.clickErr != nil {
		return f.clickErr
	}
	f.lastClicked.SearchID = searchID
	f.lastClicked.DocID = docID
	f.lastClicked.Position = pos
	f.lastClicked.At = at
	return nil
}

func (f *fakeRepo) snapshot() []*SearchRow {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*SearchRow, len(f.rows))
	copy(out, f.rows)
	return out
}

func waitForRows(t *testing.T, repo *fakeRepo, want int) {
	t.Helper()
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if len(repo.snapshot()) >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d rows, got %d", want, len(repo.snapshot()))
}

func TestCaptureSearch_InsertsRow(t *testing.T) {
	repo := &fakeRepo{}
	fixedTime := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	svc, err := NewService(Config{
		Repository: repo,
		Clock:      func() time.Time { return fixedTime },
	})
	require.NoError(t, err)

	svc.CaptureSearch(context.Background(), CaptureEvent{
		SearchID:     "abc",
		UserID:       "u1",
		Query:        "react",
		FilterBy:     "persona:freelance",
		SortBy:       "rating_score:desc",
		Persona:      "freelance",
		ResultsCount: 42,
		LatencyMs:    15,
	})

	waitForRows(t, repo, 1)
	rows := repo.snapshot()
	got := rows[0]
	assert.Equal(t, "abc", got.SearchID)
	assert.Equal(t, "freelance", got.Persona)
	assert.Equal(t, 42, got.ResultsCount)
	assert.Equal(t, fixedTime, got.CreatedAt)
}

func TestCaptureSearch_SkipsEmptyID(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := NewService(Config{Repository: repo})

	svc.CaptureSearch(context.Background(), CaptureEvent{Query: "react"})

	// Wait for would-be goroutine to finish; expect no rows.
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, repo.snapshot())
}

func TestCaptureSearch_DefaultsPersona(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := NewService(Config{Repository: repo})

	svc.CaptureSearch(context.Background(), CaptureEvent{SearchID: "x", Query: "react"})

	waitForRows(t, repo, 1)
	assert.Equal(t, "all", repo.snapshot()[0].Persona)
}

func TestCaptureSearch_SwallowsRepoErrors(t *testing.T) {
	repo := &fakeRepo{insertErr: errors.New("database is down")}
	svc, _ := NewService(Config{Repository: repo})

	// No panic / no error surfaced.
	svc.CaptureSearch(context.Background(), CaptureEvent{SearchID: "x"})
	time.Sleep(50 * time.Millisecond)
	// No rows since the insert failed.
	assert.Empty(t, repo.snapshot())
}

// TestCaptureSearch_ContextCancellation_DoesNotPropagateToGoroutine
// is the gosec G118 regression test: even when the caller's request
// context is canceled (HTTP handler returning), the persistence
// goroutine MUST still run to completion. The fix uses
// context.WithoutCancel to detach.
func TestCaptureSearch_ContextCancellation_DoesNotPropagateToGoroutine(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := NewService(Config{Repository: repo})

	ctx, cancel := context.WithCancel(context.Background())
	svc.CaptureSearch(ctx, CaptureEvent{SearchID: "abc", Query: "react"})
	// Cancel the request context IMMEDIATELY — the goroutine must
	// keep running because it derives from WithoutCancel(ctx).
	cancel()

	waitForRows(t, repo, 1)
	rows := repo.snapshot()
	require.Len(t, rows, 1)
	assert.Equal(t, "abc", rows[0].SearchID)
}

// TestPersist_RespectsTimeout proves the detached goroutine still
// honors a 3-second timeout — a stuck DB connection cannot leak the
// goroutine indefinitely.
func TestPersist_RespectsTimeout(t *testing.T) {
	// blockingRepo holds InsertSearch until the test signals.
	repo := &fakeRepo{}
	svc, _ := NewService(Config{Repository: repo})

	// Pre-canceled parent — the persist call should still try to
	// insert because of WithoutCancel; the inner WithTimeout(3s)
	// adds the actual deadline.
	parent := context.Background()
	row := svc.buildRow(CaptureEvent{SearchID: "x", Persona: "all"})
	svc.persist(parent, row)

	rows := repo.snapshot()
	require.Len(t, rows, 1)
	assert.Equal(t, "x", rows[0].SearchID)
}

func TestRecordClick_Success(t *testing.T) {
	repo := &fakeRepo{}
	fixedTime := time.Now()
	svc, _ := NewService(Config{
		Repository: repo,
		Clock:      func() time.Time { return fixedTime },
	})

	err := svc.RecordClick(context.Background(), "s1", "d1", 3)
	require.NoError(t, err)
	assert.Equal(t, "s1", repo.lastClicked.SearchID)
	assert.Equal(t, "d1", repo.lastClicked.DocID)
	assert.Equal(t, 3, repo.lastClicked.Position)
	assert.Equal(t, fixedTime, repo.lastClicked.At)
}

func TestRecordClick_Validates(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := NewService(Config{Repository: repo})

	cases := []struct {
		name     string
		search   string
		docID    string
		position int
	}{
		{"empty search id", "", "d1", 0},
		{"empty doc id", "s1", "", 0},
		{"negative position", "s1", "d1", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.RecordClick(context.Background(), tc.search, tc.docID, tc.position)
			require.Error(t, err)
		})
	}
}

func TestRecordClick_PropagatesNotFound(t *testing.T) {
	repo := &fakeRepo{clickErr: ErrNotFound}
	svc, _ := NewService(Config{Repository: repo})

	err := svc.RecordClick(context.Background(), "s1", "d1", 0)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestNewService_RequiresRepo(t *testing.T) {
	_, err := NewService(Config{})
	require.Error(t, err)
}
