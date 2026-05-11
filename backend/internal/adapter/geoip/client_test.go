package geoip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestClient_Lookup_Success uses an httptest server to mimic the
// ipapi.co success shape and verifies the adapter parses it correctly.
func TestClient_Lookup_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/8.8.8.8/json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"city":"Paris","country_code":"fr","error":false}`))
	}))
	defer srv.Close()

	c := NewClientWithEndpoint(srv.URL, time.Second)
	loc, err := c.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.City != "Paris" {
		t.Errorf("City = %q, want Paris", loc.City)
	}
	if loc.CountryCode != "FR" {
		t.Errorf("CountryCode = %q, want FR (uppercased)", loc.CountryCode)
	}
}

// TestClient_Lookup_NonRoutable proves loopback / private / unspecified
// IPs short-circuit and never hit the network.
func TestClient_Lookup_NonRoutable(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClientWithEndpoint(srv.URL, time.Second)
	for _, ip := range []string{"", "127.0.0.1", "::1", "10.0.0.1", "192.168.1.1", "169.254.1.1", "0.0.0.0", "not-an-ip"} {
		loc, err := c.Lookup(context.Background(), ip)
		if err != nil {
			t.Errorf("%s: unexpected error %v", ip, err)
		}
		if loc.City != "" || loc.CountryCode != "" {
			t.Errorf("%s: expected empty location, got %+v", ip, loc)
		}
	}
	if hits != 0 {
		t.Errorf("expected 0 network calls for non-routable IPs, got %d", hits)
	}
}

// TestClient_Lookup_ApiError parses the {"error": true} response shape
// and returns an empty GeoLocation without bubbling an error.
func TestClient_Lookup_ApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":true,"reason":"RateLimited"}`))
	}))
	defer srv.Close()

	c := NewClientWithEndpoint(srv.URL, time.Second)
	loc, err := c.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.City != "" || loc.CountryCode != "" {
		t.Errorf("expected empty location on rate-limit, got %+v", loc)
	}
}

// TestClient_Lookup_Non2xx returns empty on a 503/etc. — the auth flow
// must NEVER error out because of a third-party hiccup.
func TestClient_Lookup_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClientWithEndpoint(srv.URL, time.Second)
	loc, err := c.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.City != "" || loc.CountryCode != "" {
		t.Errorf("expected empty location on 503, got %+v", loc)
	}
}

// TestClient_Lookup_Timeout ensures a slow upstream is bounded by the
// configured timeout and resolves to empty.
func TestClient_Lookup_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"city":"Paris","country_code":"FR"}`))
	}))
	defer srv.Close()

	// 50ms timeout < 200ms upstream sleep → request must abort.
	c := NewClientWithEndpoint(srv.URL, 50*time.Millisecond)
	loc, err := c.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.City != "" || loc.CountryCode != "" {
		t.Errorf("expected empty location on timeout, got %+v", loc)
	}
}

// TestClient_Lookup_NilSafe — calling Lookup on a nil receiver must
// not panic; it should silently return empty so consumers can rely on
// the optional-adapter pattern without nil checks.
func TestClient_Lookup_NilSafe(t *testing.T) {
	var c *Client
	loc, err := c.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.City != "" || loc.CountryCode != "" {
		t.Errorf("expected empty location for nil client, got %+v", loc)
	}
}
