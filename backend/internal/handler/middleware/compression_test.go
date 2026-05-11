package middleware

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeLargeJSON returns a JSON-shaped body of approximately n bytes.
// The repeated content compresses well; that is intentional — we want
// to verify the middleware behaviour, not the gzip algorithm.
func makeLargeJSON(n int) []byte {
	chunk := []byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","name":"Acme Agency","description":"Lorem ipsum dolor sit amet consectetur adipiscing elit."},`)
	var b bytes.Buffer
	b.WriteByte('[')
	for b.Len() < n {
		b.Write(chunk)
	}
	// Replace trailing comma with closing bracket so the result is
	// valid JSON.
	if b.Len() > 1 {
		b.Truncate(b.Len() - 1)
	}
	b.WriteByte(']')
	return b.Bytes()
}

func decodeGzip(t *testing.T, src []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(src))
	require.NoError(t, err)
	defer zr.Close()
	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	return out
}

// jsonHandler returns a handler emitting the given body with Content-
// Type: application/json. Used as the "next" in middleware tests.
func jsonHandler(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func TestCompression_LargeJSONIsGzipped(t *testing.T) {
	body := makeLargeJSON(8 * 1024) // 8 KiB JSON
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	assert.Empty(t, res.Header.Get("Content-Length"))
	assert.Contains(t, res.Header.Get("Vary"), "Accept-Encoding")

	// Decode and verify the body round-trips.
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	decoded := decodeGzip(t, raw)
	assert.Equal(t, body, decoded)

	// The compressed payload must be measurably smaller than raw.
	// JSON with the helper's chunk compresses ~10×.
	assert.Less(t, len(raw), len(body)/2,
		"compressed payload should be < 50%% of original (got %d / %d)", len(raw), len(body))
}

func TestCompression_SmallResponseStaysRaw(t *testing.T) {
	body := []byte(`{"status":"ok"}`)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
	raw, _ := io.ReadAll(res.Body)
	assert.Equal(t, body, raw)
}

func TestCompression_NoAcceptEncodingHeaderSkips(t *testing.T) {
	body := makeLargeJSON(8 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	// No Accept-Encoding header at all.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
	raw, _ := io.ReadAll(res.Body)
	assert.Equal(t, body, raw)
}

func TestCompression_AcceptEncodingWithoutGzipSkips(t *testing.T) {
	body := makeLargeJSON(8 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "br, deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
}

func TestCompression_AcceptEncodingMultipleTokensWithGzipMatches(t *testing.T) {
	body := makeLargeJSON(8 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "br, gzip;q=0.8, deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestCompression_AcceptEncodingIsCaseInsensitive(t *testing.T) {
	body := makeLargeJSON(8 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "GZIP")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestCompression_ImageContentTypeIsSkipped(t *testing.T) {
	// A pretend PNG payload (already-compressed binary). The
	// middleware must not re-encode it.
	body := bytes.Repeat([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 2048) // 16 KiB
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/avatar.png", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
	raw, _ := io.ReadAll(res.Body)
	assert.Equal(t, body, raw)
}

func TestCompression_PreEncodedResponseIsNotDoubleEncoded(t *testing.T) {
	// Handler set Content-Encoding itself (e.g. streamed a pre-
	// gzipped blob from R2).
	preGzipped := gzipBytes([]byte(`{"hint":"pre-encoded"}`))
	body := preGzipped // exceeds 1 KiB? no — make it big.
	for len(body) < 4*1024 {
		body = append(body, preGzipped...)
	}
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/something", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	raw, _ := io.ReadAll(res.Body)
	// Verify the body is exactly what the handler wrote, not
	// re-encoded.
	assert.Equal(t, body, raw)
}

func TestCompression_HeadRequestSkipsCompression(t *testing.T) {
	body := makeLargeJSON(8 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodHead, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
}

func TestCompression_204NoContentIsNotCompressed(t *testing.T) {
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/something", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
	assert.Empty(t, res.Header.Get("Content-Encoding"))
}

func TestCompression_PreservesStatusCode(t *testing.T) {
	body := makeLargeJSON(4 * 1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestCompression_DropsStaleContentLength(t *testing.T) {
	body := makeLargeJSON(4 * 1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Handler set a stale Content-Length — middleware must
		// strip it because the gzipped body has a different
		// length the handler did not compute.
		w.Header().Set("Content-Length", "9999")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Length"))
}

func TestCompression_AddsVaryAcceptEncoding(t *testing.T) {
	body := makeLargeJSON(4 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Contains(t, res.Header.Get("Vary"), "Accept-Encoding")
}

func TestCompression_TextHTMLIsCompressed(t *testing.T) {
	body := bytes.Repeat([]byte(`<p>Hello world from a marketplace SSR fallback page.</p>`), 200)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/render", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	raw, _ := io.ReadAll(res.Body)
	assert.Equal(t, body, decodeGzip(t, raw))
}

func TestCompression_NDJSONIsCompressed(t *testing.T) {
	var b bytes.Buffer
	for i := 0; i < 200; i++ {
		b.WriteString(`{"event":"profile_view","ts":"2026-05-11T10:00:00Z"}` + "\n")
	}
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b.Bytes())
	}))

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestCompression_OctetStreamIsSkipped(t *testing.T) {
	body := bytes.Repeat([]byte{0x42}, 8*1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/download.bin", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
}

func TestCompression_HandlerWithImplicit200(t *testing.T) {
	// Handler calls Write without an explicit WriteHeader; net/http
	// implicitly sends 200. The middleware must honour that.
	body := makeLargeJSON(4 * 1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestCompression_EmptyResponseStaysEmpty(t *testing.T) {
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
}

func TestCompression_SmallTextSniffedAsHTMLStaysRaw(t *testing.T) {
	// Handler did not set Content-Type. http.DetectContentType
	// would sniff this as text/html — but the body is under the
	// size threshold, so it must stay raw.
	body := []byte(`<html><body>hi</body></html>`)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Empty(t, res.Header.Get("Content-Encoding"))
}

func TestCompression_StreamingHandlerFlushesGzipFrames(t *testing.T) {
	// A handler that writes a chunk, flushes, and writes another
	// chunk. The middleware must compress both and emit them in
	// proper gzip framing.
	chunk := makeLargeJSON(4 * 1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chunk)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write(chunk)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	raw, _ := io.ReadAll(res.Body)
	decoded := decodeGzip(t, raw)
	assert.Equal(t, append(append([]byte{}, chunk...), chunk...), decoded)
}

func TestCompression_DoubleWriteHeaderIsIgnored(t *testing.T) {
	body := makeLargeJSON(4 * 1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.WriteHeader(http.StatusOK) // double call — must be a no-op
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode)
}

func TestCompression_StripsAcceptRangesOnGzip(t *testing.T) {
	body := makeLargeJSON(4 * 1024)
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	assert.Empty(t, res.Header.Get("Accept-Ranges"))
}

func TestCompression_FlushBeforeWriteSetsImplicit200(t *testing.T) {
	// A handler that flushes before writing any body — the
	// middleware should still emit a valid response.
	h := Compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write(makeLargeJSON(4 * 1024))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

// gzipBytes is a test helper that wraps the bytes in a gzip stream so
// "pre-encoded" responses can be simulated without going through the
// middleware.
func gzipBytes(b []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

// ----- Benchmarks (bench-perf harness) -----

func BenchmarkCompression_LargeJSON(b *testing.B) {
	body := makeLargeJSON(50 * 1024)
	h := Compression(jsonHandler(body))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkCompression_NoGzipAccept(b *testing.B) {
	body := makeLargeJSON(50 * 1024)
	h := Compression(jsonHandler(body))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	// no Accept-Encoding
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkCompression_SmallJSON(b *testing.B) {
	body := []byte(`{"status":"ok","ts":"2026-05-11T10:00:00Z"}`)
	h := Compression(jsonHandler(body))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func TestCompression_ReportSizeReduction_Bench(t *testing.T) {
	// Soft assertion that records the before/after byte counts in
	// the test log so CI surfaces the win. Asserts a >= 3× ratio
	// on a 16 KiB JSON payload — typical for the search endpoint.
	body := makeLargeJSON(16 * 1024)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	compressed, _ := io.ReadAll(res.Body)

	t.Logf("compression: %d → %d bytes (%.1f×)",
		len(body), len(compressed), float64(len(body))/float64(len(compressed)))
	require.NotZero(t, len(compressed))
	assert.Less(t, len(compressed), len(body)/3,
		"expected >= 3× compression on 16KiB JSON")
}

// TestCompression_HijackPassesThrough verifies the wrapped writer
// proxies Hijack to the underlying ResponseWriter when it supports
// it — protocol upgrades (websocket / SSE) need raw access to the
// net.Conn and MUST never be wrapped in gzip framing.
func TestCompression_HijackPassesThrough(t *testing.T) {
	called := false
	rec := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder(), called: &called}
	cw := &compressingResponseWriter{ResponseWriter: rec, buf: getBuffer()}
	_, _, _ = cw.Hijack()
	assert.True(t, called, "Hijack must reach the underlying ResponseWriter")
}

// TestCompression_HijackReturnsErrWhenUnderlyingDoesNotSupport asserts
// the graceful error path when the inner writer is not a Hijacker.
func TestCompression_HijackReturnsErrWhenUnderlyingDoesNotSupport(t *testing.T) {
	rec := httptest.NewRecorder() // does not implement Hijacker
	cw := &compressingResponseWriter{ResponseWriter: rec, buf: getBuffer()}
	_, _, err := cw.Hijack()
	assert.ErrorContains(t, err, "compression")
}

// TestCompression_OversizedBufferIsDroppedFromPool exercises the
// 64 KiB cap in releaseBuffer — buffers larger than the cap are
// discarded so a long-lived process does not pin tens of MiB of
// pooled memory after a single huge response.
func TestCompression_OversizedBufferIsDroppedFromPool(t *testing.T) {
	b := new(bytes.Buffer)
	b.Grow(128 * 1024) // cap >> 64 KiB
	releaseBuffer(b)
	// The pool returned a fresh buffer because the oversized one
	// was discarded. We cannot directly observe pool state, but
	// getBuffer should never panic and should return an empty buf.
	got := getBuffer()
	assert.Equal(t, 0, got.Len())
}

// TestCompression_NilBufferReleaseIsSafe pins the nil-safety contract
// — defer-driven release with a never-allocated buffer must be a
// no-op, not a panic.
func TestCompression_NilBufferReleaseIsSafe(t *testing.T) {
	assert.NotPanics(t, func() { releaseBuffer(nil) })
}

type hijackableRecorder struct {
	*httptest.ResponseRecorder
	called *bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	*h.called = true
	return nil, nil, nil
}

// regression: ensure the "Vary" header is added (multi-valued) even
// when the middleware short-circuits on small bodies, so a shared
// cache that already saw a small response still keys gzip vs
// identity correctly.
func TestCompression_VaryAddedEvenWhenSmallResponse(t *testing.T) {
	body := []byte(`{"ok":true}`)
	h := Compression(jsonHandler(body))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	// We add Vary at request time, before the size decision —
	// keeps cache-key correctness for shared caches.
	vary := strings.ToLower(res.Header.Get("Vary"))
	assert.Contains(t, vary, "accept-encoding")
}
