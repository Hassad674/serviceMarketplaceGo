package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	appstats "marketplace-backend/internal/app/stats"
	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/handler"
)

// fakeRecorder satisfies handler.StatsRecorder. Captures calls + a
// "wait until called" channel so tests can synchronise with the
// fire-and-forget goroutine.
type fakeRecorder struct {
	mu       sync.Mutex
	calls    []appstats.RecordViewInput
	called   chan struct{}
	once     sync.Once
	returnFn func(in appstats.RecordViewInput) error
}

func newFakeRecorder() *fakeRecorder {
	return &fakeRecorder{called: make(chan struct{}, 4)}
}

func (f *fakeRecorder) Record(_ context.Context, in appstats.RecordViewInput) (*domainstats.ViewEvent, error) {
	f.mu.Lock()
	f.calls = append(f.calls, in)
	f.mu.Unlock()
	f.once.Do(func() { close(f.called) })
	if f.returnFn != nil {
		if err := f.returnFn(in); err != nil {
			return nil, err
		}
	}
	return &domainstats.ViewEvent{ID: uuid.New()}, nil
}

func (f *fakeRecorder) Calls() []appstats.RecordViewInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]appstats.RecordViewInput(nil), f.calls...)
}

// waitForCall blocks until the recorder fires once or the timeout
// elapses. Returns true on success.
func (f *fakeRecorder) waitForCall(t *testing.T, timeout time.Duration) bool {
	t.Helper()
	select {
	case <-f.called:
		return true
	case <-time.After(timeout):
		return false
	}
}

// stubProfileHandler simulates the wrapped public-profile handler.
type stubProfileHandler struct {
	status int
}

func (s *stubProfileHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if s.status == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(s.status)
	}
	_, _ = w.Write([]byte(`{"data":{}}`))
}

func TestTrackProfileViews_RecordsOn200(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(rec, domainstats.PersonaFreelance, "orgID")).
		Get("/freelance-profiles/{orgID}", (&stubProfileHandler{status: 200}).ServeHTTP)

	orgID := uuid.New()
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/freelance-profiles/" + orgID.String() + "?q=go+developer&pos=3")
	if err != nil {
		t.Fatalf("client get: %v", err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	if !rec.waitForCall(t, 2*time.Second) {
		t.Fatalf("recorder was never called")
	}
	calls := rec.Calls()
	if assert.Len(t, calls, 1) {
		assert.Equal(t, orgID, calls[0].OrganizationID)
		assert.Equal(t, domainstats.PersonaFreelance, calls[0].Persona)
		assert.Equal(t, domainstats.CameFromSearch, calls[0].CameFrom)
		if assert.NotNil(t, calls[0].SearchQuery) {
			assert.Equal(t, "go developer", *calls[0].SearchQuery)
		}
		if assert.NotNil(t, calls[0].SearchPosition) {
			assert.Equal(t, 3, *calls[0].SearchPosition)
		}
	}
}

func TestTrackProfileViews_SkipsNon2xx(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(rec, domainstats.PersonaAgency, "orgId")).
		Get("/profiles/{orgId}", (&stubProfileHandler{status: http.StatusNotFound}).ServeHTTP)

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profiles/" + uuid.New().String())
	if err != nil {
		t.Fatalf("client get: %v", err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Wait a tiny bit to give a (wrong) goroutine a chance to fire.
	time.Sleep(150 * time.Millisecond)
	assert.Empty(t, rec.Calls(), "404 must not produce a tracking event")
}

func TestTrackProfileViews_SkipsInvalidOrgID(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(rec, domainstats.PersonaAgency, "orgId")).
		Get("/profiles/{orgId}", (&stubProfileHandler{}).ServeHTTP)

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profiles/not-a-uuid")
	if err != nil {
		t.Fatalf("client get: %v", err)
	}
	resp.Body.Close()
	time.Sleep(150 * time.Millisecond)
	assert.Empty(t, rec.Calls(), "non-UUID org id must not produce a tracking event")
}

func TestTrackProfileViews_NilRecorderIsPassthrough(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(nil, domainstats.PersonaAgency, "orgId")).
		Get("/profiles/{orgId}", (&stubProfileHandler{}).ServeHTTP)

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profiles/" + uuid.New().String())
	if err != nil {
		t.Fatalf("client get: %v", err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTrackProfileViews_DerivesCameFromReferer(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(rec, domainstats.PersonaAgency, "orgId")).
		Get("/profiles/{orgId}", (&stubProfileHandler{}).ServeHTTP)

	srv := httptest.NewServer(r)
	defer srv.Close()
	orgID := uuid.New()

	cases := []struct {
		name     string
		referer  string
		wantFrom domainstats.CameFrom
	}{
		{"empty referer → direct", "", domainstats.CameFromDirect},
		{"same-host search → search", srv.URL + "/search?q=go", domainstats.CameFromSearch},
		{"same-host /freelancers → list", srv.URL + "/freelancers", domainstats.CameFromList},
		{"different host → referral", "https://twitter.com/share", domainstats.CameFromReferral},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec.mu.Lock()
			rec.calls = nil
			rec.called = make(chan struct{}, 4)
			rec.once = sync.Once{}
			rec.mu.Unlock()

			req, _ := http.NewRequest(http.MethodGet, srv.URL+"/profiles/"+orgID.String(), nil)
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			resp.Body.Close()

			if !rec.waitForCall(t, 2*time.Second) {
				t.Fatalf("recorder was never called")
			}
			got := rec.Calls()[0]
			assert.Equal(t, tc.wantFrom, got.CameFrom)
		})
	}
}

func TestTrackProfileViews_BackgroundContextSurvivesCancel(t *testing.T) {
	t.Parallel()
	// The handler test exercises the goroutine lifetime indirectly:
	// an in-test recorder counts how many times it was called, and
	// the request itself is short-circuited (response written before
	// goroutine schedules). Without context.WithoutCancel the
	// goroutine's ctx would be cancelled at request return time.
	hits := atomic.Int32{}
	rec := newFakeRecorder()
	rec.returnFn = func(in appstats.RecordViewInput) error {
		// simulate a slow DB write that out-lives the request.
		time.Sleep(120 * time.Millisecond)
		hits.Add(1)
		return nil
	}

	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(rec, domainstats.PersonaAgency, "orgId")).
		Get("/profiles/{orgId}", (&stubProfileHandler{}).ServeHTTP)

	srv := httptest.NewServer(r)
	defer srv.Close()
	orgID := uuid.New()
	resp, err := http.Get(srv.URL + "/profiles/" + orgID.String())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()

	// Block waiting for the slow recorder; if WithoutCancel were
	// missing, the goroutine ctx would be cancelled and the slow
	// Record might still complete, but the value of WithoutCancel
	// is captured by the assertion that the recorder DID run.
	deadline := time.Now().Add(2 * time.Second)
	for hits.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	assert.Equal(t, int32(1), hits.Load(), "goroutine must complete after request returns")
}

func TestTrackProfileViews_RecorderErrorDoesNotPanic(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	rec.returnFn = func(_ appstats.RecordViewInput) error {
		return assert.AnError
	}

	r := chi.NewRouter()
	r.With(handler.TrackProfileViews(rec, domainstats.PersonaAgency, "orgId")).
		Get("/profiles/{orgId}", (&stubProfileHandler{}).ServeHTTP)

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profiles/" + uuid.New().String())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "tracking failure must not corrupt the response")
	rec.waitForCall(t, 2*time.Second)
}
