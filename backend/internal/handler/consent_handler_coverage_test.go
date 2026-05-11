package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	consentapp "marketplace-backend/internal/app/consent"
	"marketplace-backend/internal/domain/consent"
)

// TestConsentHandler_Log_RejectsInvalidJSONBody asserts that a body
// that doesn't decode is rejected with 400 + invalid_body. Regression:
// an earlier version returned 500 for malformed JSON.
func TestConsentHandler_Log_RejectsInvalidJSONBody(t *testing.T) {
	h, _ := newConsentHandlerForTest(t)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log",
		bytes.NewReader([]byte("{not-json")))
	r.RemoteAddr = "1.2.3.4:5"

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid_body") {
		t.Errorf("expected invalid_body code in body, got %s", w.Body.String())
	}
}

// TestConsentHandler_Log_RejectsBodyTooLarge asserts the DecodeBody
// cap actually rejects oversized payloads. We send a 2 KiB payload
// against the 1 KiB limit.
func TestConsentHandler_Log_RejectsBodyTooLarge(t *testing.T) {
	h, _ := newConsentHandlerForTest(t)
	// Fill an extra-large categories array to push the body past 1 KiB.
	big := strings.Repeat("xxxxxxxx", 256) // 2048 bytes
	body := []byte(`{"action":"accept_all","categories":["` + big + `"]}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "1.2.3.4:5"
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on oversized body, got %d body=%s", w.Code, w.Body.String())
	}
}

// TestConsentHandler_Log_IPv6Truncated asserts the IPv6 path in
// gdpr.TruncateIP via the consent surface — anonymous v6 visitors must
// be tracked just like v4.
func TestConsentHandler_Log_IPv6Truncated(t *testing.T) {
	h, repo := newConsentHandlerForTest(t)
	body := mustJSON(t, map[string]any{
		"action":     "accept_all",
		"categories": []string{"analytics"},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "[2001:db8:abcd:1234::1]:443"
	r.Header.Set("User-Agent", "Mozilla/5.0 (ipv6)")

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	if repo.created == nil {
		t.Fatal("expected entry persisted")
	}
	// TruncateIP shape: IPv6 /32 — first two groups + ":x:x:x:x:x:x"
	// or similar. Just assert non-empty and not the raw address.
	if repo.created.IPAnonymized == "" {
		t.Error("expected non-empty truncated IPv6")
	}
	if strings.Contains(repo.created.IPAnonymized, "abcd") {
		t.Errorf("truncated IPv6 should not echo the full address: %s", repo.created.IPAnonymized)
	}
}

// TestConsentHandler_Log_PersistError500 asserts that a repo failure
// other than a validation error becomes a 500 + internal_error.
func TestConsentHandler_Log_PersistError500(t *testing.T) {
	repo := &stubConsentRepo{err: errors.New("disk full")}
	svc := consentapp.NewService(repo)
	h := NewConsentHandler(svc)

	body := mustJSON(t, map[string]any{
		"action":     "accept_all",
		"categories": []string{"analytics"},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "1.2.3.4:5"
	r.Header.Set("User-Agent", "Mozilla/5.0 (e500)")

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "internal_error") {
		t.Errorf("expected internal_error code, got %s", w.Body.String())
	}
}

// TestRemoteIPFromRequest_FallbackPaths covers the helper's branches:
// XFF malformed, RemoteAddr without port, RemoteAddr empty.
func TestRemoteIPFromRequest_FallbackPaths(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		remoteAddr string
		want       string
	}{
		{
			name:       "xff with private then public picks first parseable",
			xff:        "203.0.113.5, 10.0.0.1",
			remoteAddr: "127.0.0.1:5",
			want:       "203.0.113.5",
		},
		{
			name:       "xff with garbage falls through to RemoteAddr",
			xff:        "not-an-ip",
			remoteAddr: "203.0.113.9:1234",
			want:       "203.0.113.9",
		},
		{
			name:       "RemoteAddr without port returned verbatim",
			xff:        "",
			remoteAddr: "203.0.113.10",
			want:       "203.0.113.10",
		},
		{
			name:       "empty RemoteAddr yields empty result",
			xff:        "",
			remoteAddr: "",
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/x", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			got := remoteIPFromRequest(r)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestConsentHandler_Log_TrimsSessionAndCategories asserts the handler
// + service pipeline trims whitespace and dedups categories — the
// domain normalizer is exposed only via the entity, so this test pins
// the behaviour end-to-end so a future refactor cannot regress it.
func TestConsentHandler_Log_TrimsSessionAndCategories(t *testing.T) {
	h, repo := newConsentHandlerForTest(t)
	body := mustJSON(t, map[string]any{
		"action":     "custom",
		"categories": []string{"analytics", " analytics ", "  ", "marketing"},
		"session_id": "  sess-1  ",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "1.2.3.4:5"
	r.Header.Set("User-Agent", "Mozilla/5.0 (trim)")

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	if got := repo.created.SessionID; got != "sess-1" {
		t.Errorf("session_id not trimmed: %q", got)
	}
	if len(repo.created.Categories) != 2 {
		t.Errorf("expected 2 unique categories, got %v", repo.created.Categories)
	}
	if repo.created.Categories[0] != "analytics" || repo.created.Categories[1] != "marketing" {
		t.Errorf("expected [analytics, marketing], got %v", repo.created.Categories)
	}
}

// TestConsent_Service_HashUserAgent_EmptyString asserts the
// hashUserAgent helper short-circuits on empty input. The domain
// rejects empty UA hashes, so the empty branch must return "" and
// surface ErrUserAgentHashRequired downstream.
func TestConsent_Service_EmptyUserAgent_Returns400(t *testing.T) {
	h, repo := newConsentHandlerForTest(t)
	body := mustJSON(t, map[string]any{
		"action":     "accept_all",
		"categories": []string{"analytics"},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "1.2.3.4:5"
	// No User-Agent header set on purpose.

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo must not be called when UA is empty")
	}
}

// TestConsent_DomainNew_NormalizesAndTrims covers the normalizer
// branch in the domain entity that the service-level test does not
// touch directly.
func TestConsent_DomainNew_NormalizesAndTrims(t *testing.T) {
	entry, err := consent.New(consent.NewInput{
		Action:        consent.ActionAcceptAll,
		Categories:    []string{" analytics ", "analytics", "", "  marketing"},
		IPAnonymized:  "  203.0.x.x  ",
		UserAgentHash: "  abc  ",
		SessionID:     "  sess  ",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(entry.Categories) != 2 {
		t.Errorf("expected dedup to yield 2, got %v", entry.Categories)
	}
	if entry.IPAnonymized != "203.0.x.x" {
		t.Errorf("ip not trimmed: %q", entry.IPAnonymized)
	}
	if entry.SessionID != "sess" {
		t.Errorf("session not trimmed: %q", entry.SessionID)
	}
	if entry.UserAgentHash != "abc" {
		t.Errorf("UA hash not trimmed: %q", entry.UserAgentHash)
	}
}

// TestConsent_Service_Record_TestAcceptsNoUserID asserts that an
// anonymous request (UserID nil) is still recorded — RGPD requires
// proof of consent for anonymous visitors too.
func TestConsent_Service_Record_AcceptsNilUserID(t *testing.T) {
	repo := &stubConsentRepo{}
	svc := consentapp.NewService(repo)
	_, err := svc.Record(context.Background(), consentapp.RecordInput{
		UserID:     nil,
		Action:     consent.ActionRefuseAll,
		Categories: []string{"analytics"},
		RawIP:      "1.2.3.4",
		UserAgent:  "Mozilla/5.0",
	})
	if err != nil {
		t.Fatalf("nil UserID must be accepted: %v", err)
	}
	if repo.created == nil {
		t.Fatal("entry must be persisted")
	}
	if repo.created.UserID != nil {
		t.Errorf("expected nil UserID, got %v", repo.created.UserID)
	}
}
