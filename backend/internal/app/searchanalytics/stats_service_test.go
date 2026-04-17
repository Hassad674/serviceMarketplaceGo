package searchanalytics

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeStatsRepo is an in-memory StatsRepository used by the service
// unit tests. Fields record the last filter + limit received so each
// test can assert the normalisation happened before the repo was
// called.
type fakeStatsRepo struct {
	totals            Totals
	top               []TopQuery
	zero              []ZeroResultQuery
	totalsErr         error
	topErr            error
	zeroErr           error
	lastTotalsFilter  StatsFilter
	lastTopFilter     StatsFilter
	lastTopLimit      int
	lastZeroFilter    StatsFilter
	lastZeroLimit     int
	totalsCallCount   int
	topCallCount      int
	zeroCallCount     int
}

func (f *fakeStatsRepo) Totals(ctx context.Context, filter StatsFilter) (Totals, error) {
	f.totalsCallCount++
	f.lastTotalsFilter = filter
	return f.totals, f.totalsErr
}

func (f *fakeStatsRepo) TopQueries(ctx context.Context, filter StatsFilter, limit int) ([]TopQuery, error) {
	f.topCallCount++
	f.lastTopFilter = filter
	f.lastTopLimit = limit
	return f.top, f.topErr
}

func (f *fakeStatsRepo) ZeroResultQueries(ctx context.Context, filter StatsFilter, limit int) ([]ZeroResultQuery, error) {
	f.zeroCallCount++
	f.lastZeroFilter = filter
	f.lastZeroLimit = limit
	return f.zero, f.zeroErr
}

// staticClock returns a fixed time every call. Used by the service
// tests to pin the default window.
func staticClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// TestNewStatsService_RequiresRepository covers the constructor
// guard rail. Missing repository must return a typed error, not a
// panic.
func TestNewStatsService_RequiresRepository(t *testing.T) {
	_, err := NewStatsService(StatsServiceConfig{})
	if err == nil {
		t.Fatal("expected an error when repository is nil")
	}
}

// TestStatsService_Compute_AppliesDefaults pins the window defaulting
// behaviour: empty input → last 7 days ending at s.clock().
func TestStatsService_Compute_AppliesDefaults(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	repo := &fakeStatsRepo{
		totals: Totals{TotalSearches: 100, ZeroResults: 10, ZeroResultRate: 0.1, AvgLatencyMs: 50, P95LatencyMs: 120},
		top:    []TopQuery{{Query: "react", Count: 5}},
		zero:   []ZeroResultQuery{{Query: "unknown", Count: 1}},
	}
	svc, err := NewStatsService(StatsServiceConfig{Repository: repo, Clock: staticClock(now)})
	if err != nil {
		t.Fatalf("unexpected error building service: %v", err)
	}

	stats, err := svc.Compute(context.Background(), StatsQuery{})
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}
	if !stats.To.Equal(now) {
		t.Errorf("default To = %v, want %v", stats.To, now)
	}
	wantFrom := now.Add(-DefaultStatsWindow)
	if !stats.From.Equal(wantFrom) {
		t.Errorf("default From = %v, want %v", stats.From, wantFrom)
	}
	if stats.TotalSearches != 100 {
		t.Errorf("TotalSearches = %d, want 100", stats.TotalSearches)
	}
	if len(stats.TopQueries) != 1 || stats.TopQueries[0].Query != "react" {
		t.Errorf("top queries not forwarded correctly: %+v", stats.TopQueries)
	}
	if len(stats.ZeroResultQueries) != 1 || stats.ZeroResultQueries[0].Query != "unknown" {
		t.Errorf("zero-result queries not forwarded correctly: %+v", stats.ZeroResultQueries)
	}
	if repo.lastTopLimit != DefaultStatsLimit {
		t.Errorf("default limit forwarded = %d, want %d", repo.lastTopLimit, DefaultStatsLimit)
	}
}

// TestStatsService_Compute_CapsLimit pins the MaxStatsLimit guard.
// A caller asking for 5000 rows must be clamped to MaxStatsLimit.
func TestStatsService_Compute_CapsLimit(t *testing.T) {
	repo := &fakeStatsRepo{}
	svc, _ := NewStatsService(StatsServiceConfig{Repository: repo})

	_, err := svc.Compute(context.Background(), StatsQuery{Limit: 5000})
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}
	if repo.lastTopLimit != MaxStatsLimit {
		t.Errorf("forwarded top limit = %d, want %d", repo.lastTopLimit, MaxStatsLimit)
	}
	if repo.lastZeroLimit != MaxStatsLimit {
		t.Errorf("forwarded zero limit = %d, want %d", repo.lastZeroLimit, MaxStatsLimit)
	}
}

// TestStatsService_Compute_RejectsInvalidRange covers the validation
// path: from ≥ to must surface ErrInvalidRange.
func TestStatsService_Compute_RejectsInvalidRange(t *testing.T) {
	repo := &fakeStatsRepo{}
	svc, _ := NewStatsService(StatsServiceConfig{Repository: repo})

	from := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	to := from.Add(-time.Hour) // to is BEFORE from

	_, err := svc.Compute(context.Background(), StatsQuery{From: from, To: to})
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("expected ErrInvalidRange, got %v", err)
	}
	if repo.totalsCallCount != 0 {
		t.Errorf("repo.Totals called despite invalid range (%d times)", repo.totalsCallCount)
	}
}

// TestStatsService_Compute_PropagatesErrors ensures a repo failure
// bubbles up with the operation name so operators can pin which
// underlying query broke.
func TestStatsService_Compute_PropagatesErrors(t *testing.T) {
	wantErr := errors.New("db hiccup")
	repo := &fakeStatsRepo{totalsErr: wantErr}
	svc, _ := NewStatsService(StatsServiceConfig{Repository: repo})

	_, err := svc.Compute(context.Background(), StatsQuery{})
	if err == nil || !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped wantErr, got %v", err)
	}
}

// TestStatsService_Compute_ForwardsPersona covers the persona
// filter plumbing — it must land unchanged on every repo call.
func TestStatsService_Compute_ForwardsPersona(t *testing.T) {
	repo := &fakeStatsRepo{}
	svc, _ := NewStatsService(StatsServiceConfig{Repository: repo})

	_, err := svc.Compute(context.Background(), StatsQuery{Persona: "freelance"})
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}
	if repo.lastTotalsFilter.Persona != "freelance" {
		t.Errorf("Totals filter persona = %q, want %q", repo.lastTotalsFilter.Persona, "freelance")
	}
	if repo.lastTopFilter.Persona != "freelance" {
		t.Errorf("TopQueries filter persona = %q, want %q", repo.lastTopFilter.Persona, "freelance")
	}
	if repo.lastZeroFilter.Persona != "freelance" {
		t.Errorf("ZeroResultQueries filter persona = %q, want %q", repo.lastZeroFilter.Persona, "freelance")
	}
}
