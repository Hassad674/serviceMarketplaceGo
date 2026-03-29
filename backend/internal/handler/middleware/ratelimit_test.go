package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimiter_UnderLimit(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     10,
		burst:    10,
	}
	handler := rl.Middleware(newOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_AtLimit(t *testing.T) {
	burst := float64(3)
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     0,
		burst:    burst,
	}
	handler := rl.Middleware(newOKHandler())

	for i := 0; i < int(burst); i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code,
			"request %d of %d must pass", i+1, int(burst))
	}
}

func TestRateLimiter_OverLimit(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     0,
		burst:    2,
	}
	handler := rl.Middleware(newOKHandler())

	ip := "10.0.0.2:8080"

	// Exhaust the burst allowance.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request must be blocked.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "rate_limit_exceeded", body["error"])
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     0,
		burst:    1,
	}
	handler := rl.Middleware(newOKHandler())

	// IP-A uses its single token.
	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "1.1.1.1:1111"
	recA := httptest.NewRecorder()
	handler.ServeHTTP(recA, reqA)
	assert.Equal(t, http.StatusOK, recA.Code)

	// IP-A is now blocked.
	reqA2 := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA2.RemoteAddr = "1.1.1.1:1111"
	recA2 := httptest.NewRecorder()
	handler.ServeHTTP(recA2, reqA2)
	assert.Equal(t, http.StatusTooManyRequests, recA2.Code)

	// IP-B must still pass -- independent bucket.
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "2.2.2.2:2222"
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)
	assert.Equal(t, http.StatusOK, recB.Code)
}

func TestRateLimiter_AllowMethod(t *testing.T) {
	tests := []struct {
		name  string
		burst float64
		calls int
		want  bool
	}{
		{
			name:  "first request allowed",
			burst: 5,
			calls: 1,
			want:  true,
		},
		{
			name:  "burst+1 request blocked",
			burst: 2,
			calls: 3,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RateLimiter{
				visitors: make(map[string]*visitor),
				rate:     0,
				burst:    tt.burst,
			}
			var allowed bool
			for i := 0; i < tt.calls; i++ {
				allowed = rl.allow("key")
			}
			assert.Equal(t, tt.want, allowed)
		})
	}
}
