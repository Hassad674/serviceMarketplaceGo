package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appstats "marketplace-backend/internal/app/stats"
	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/handler/middleware"
)

// StatsRecorder is the narrow contract the tracking middleware needs
// from the stats app service. Defined locally so tests inject a
// closure-driven fake without importing the full *appstats.Service.
type StatsRecorder interface {
	Record(ctx context.Context, in appstats.RecordViewInput) (*domainstats.ViewEvent, error)
}

// trackingResponseWriter is a tiny http.ResponseWriter wrapper that
// captures the final status code so the tracking middleware can
// short-circuit on non-2xx responses (we don't record an event when
// the public profile read returned 404 or 5xx).
type trackingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *trackingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// TrackProfileViews wraps a public profile GET handler so a
// successful response triggers a fire-and-forget background
// goroutine that persists a profile_view_events row.
//
// The goroutine uses context.WithoutCancel so request cancellation
// (browser closes connection right after receiving the response)
// does NOT cancel the recording. A 5s upper bound is applied to
// the background context so a stuck DB connection does not pin a
// goroutine forever.
//
// recorder may be nil — in that case the middleware is a no-op
// passthrough. This keeps the feature fully removable: deleting the
// stats wiring in cmd/api skips the wrapping without touching the
// underlying handler signature.
//
// urlParam is the chi.URLParam key holding the org id (e.g. "orgID"
// for the freelance/referrer routes, "orgId" for the legacy agency
// route).
func TrackProfileViews(recorder StatsRecorder, persona domainstats.Persona, urlParam string) func(http.Handler) http.Handler {
	if recorder == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			tw := &trackingResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(tw, r)

			// Only record successful 2xx responses — 4xx/5xx mean the
			// org was never read.
			if tw.status < 200 || tw.status >= 300 {
				return
			}

			rawOrgID := chi.URLParam(r, urlParam)
			orgID, err := uuid.Parse(rawOrgID)
			if err != nil {
				return
			}

			input := buildRecordViewInput(r, orgID, persona)

			go recordViewBackground(r.Context(), recorder, input)
		})
	}
}

// recordViewBackground runs the actual repository write on a fresh
// goroutine with a detached context. Errors are logged at WARN level
// — a tracking failure must never surface to the user.
func recordViewBackground(parent context.Context, recorder StatsRecorder, in appstats.RecordViewInput) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Warn("stats: tracking goroutine panicked", "recover", rec)
		}
	}()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancel()
	if _, err := recorder.Record(ctx, in); err != nil {
		slog.Warn("stats: failed to record profile view", "error", err)
	}
}

// buildRecordViewInput translates an HTTP request into the service's
// input shape. Pure function — exported so the handler test suite
// can call it directly.
func buildRecordViewInput(r *http.Request, orgID uuid.UUID, persona domainstats.Persona) appstats.RecordViewInput {
	came, refererURL := deriveCameFrom(r)
	q, pos := readSearchHints(r)

	in := appstats.RecordViewInput{
		OrganizationID: orgID,
		Persona:        persona,
		RawIP:          clientIP(r),
		UserAgent:      r.UserAgent(),
		CameFrom:       came,
		ReferrerURL:    refererURL,
	}
	if userID, ok := middleware.GetUserID(r.Context()); ok && userID != uuid.Nil {
		copyID := userID
		in.ViewerUserID = &copyID
	}
	if q != "" {
		copyQ := q
		in.SearchQuery = &copyQ
		// When the frontend says the user came from search but we did
		// not get a position, keep CameFrom=search; otherwise default
		// to derived value.
		in.CameFrom = domainstats.CameFromSearch
	}
	if pos > 0 {
		copyPos := pos
		in.SearchPosition = &copyPos
	}
	return in
}

// readSearchHints reads the optional ?q=<query>&pos=<rank> query
// params the frontend passes when navigating from a search result
// to the profile detail page. Both are optional; missing/invalid
// values resolve to "" and 0.
func readSearchHints(r *http.Request) (string, int) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}
	posRaw := r.URL.Query().Get("pos")
	pos, _ := strconv.Atoi(posRaw)
	if pos < 0 {
		pos = 0
	}
	return q, pos
}

// deriveCameFrom inspects the Referer header to decide where the
// visitor was navigating from. Returns the CameFrom value and a
// pointer to the referer URL when present (so the row stores the
// raw URL for forensic debugging — never longer than 2048 chars).
func deriveCameFrom(r *http.Request) (domainstats.CameFrom, *string) {
	raw := strings.TrimSpace(r.Header.Get("Referer"))
	if raw == "" {
		return domainstats.CameFromDirect, nil
	}
	// Cap the stored URL length so a malicious referer cannot bloat
	// the table.
	if len(raw) > 2048 {
		raw = raw[:2048]
	}
	captured := raw

	parsed, err := url.Parse(raw)
	if err != nil {
		return domainstats.CameFromUnknown, &captured
	}

	host := strings.ToLower(parsed.Host)
	path := strings.ToLower(parsed.Path)

	if isExternalReferer(host, r) {
		return domainstats.CameFromReferral, &captured
	}
	switch {
	case strings.Contains(path, "/search"):
		return domainstats.CameFromSearch, &captured
	case strings.Contains(path, "/freelancers") || strings.Contains(path, "/agencies") || strings.Contains(path, "/referrers"):
		return domainstats.CameFromList, &captured
	default:
		return domainstats.CameFromDirect, &captured
	}
}

// isExternalReferer returns true when the referer host is different
// from the request's Host header. Same-origin requests resolve to
// false. Behind a reverse proxy the Host header is the rewritten
// public host so this still works.
func isExternalReferer(refererHost string, r *http.Request) bool {
	if refererHost == "" {
		return false
	}
	reqHost := strings.ToLower(r.Host)
	if reqHost == "" {
		return false
	}
	// Strip ports for the comparison — a referer pointing at
	// example.com:443 still matches example.com.
	reqHost = stripPort(reqHost)
	refererHost = stripPort(refererHost)
	return refererHost != reqHost
}

func stripPort(host string) string {
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}
