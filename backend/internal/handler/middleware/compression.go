// Package middleware — Compression: transparent gzip encoding for
// JSON / text responses larger than CompressionMinBytes.
//
// Why this exists (D7 / Target B): the backend emits 10-50 KiB JSON
// payloads on hot read paths (search results, profile aggregates,
// catalog endpoints). gzip compresses JSON 5-10× — on mobile networks
// that turns into a 30-50% TTFB reduction. Without this middleware,
// every byte travels uncompressed even when the client advertised
// `Accept-Encoding: gzip`.
//
// Behaviour:
//   - Reads request `Accept-Encoding`. If it does not include `gzip`
//     (token-bounded match, case-insensitive), the middleware is a
//     no-op — the response is forwarded uncompressed.
//   - Wraps the ResponseWriter with a buffering layer that captures
//     the first write and inspects the Content-Type set by the
//     handler.
//   - **Skip-by-type**: image/*, video/*, audio/*, application/zip,
//     application/gzip, application/x-gzip, application/octet-stream —
//     these are already compressed; re-compressing wastes CPU and
//     usually *grows* the payload by a few bytes of gzip framing.
//   - **Skip-by-size**: bodies smaller than CompressionMinBytes (1 KiB
//     default) are forwarded raw. Below this threshold the 18-byte
//     gzip header + Adler-32 trailer plus the per-response CPU cost
//     dominate any saving. Bench measurements in
//     compression_test.go confirm the breakeven at ~600-800 B for
//     JSON; 1024 B gives a comfortable margin.
//   - If a downstream handler already set `Content-Encoding`
//     (e.g. the response was pre-gzipped server-side, as the audit
//     archive does), the middleware skips compression to avoid
//     double-encoding.
//   - HEAD requests and 204 / 304 responses are never compressed
//     (no body to compress).
//
// Thread safety: each request gets a fresh wrapper instance. The
// gzip.Writer pool is goroutine-safe.
package middleware

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// CompressionMinBytes is the response-size threshold below which the
// middleware short-circuits. See package comment for the rationale.
//
// Exposed (not const) so tests can shrink it without changing
// production behaviour. Production code never overwrites this; the
// value is a sane default for the typical 10-50 KiB JSON payload.
var CompressionMinBytes = 1024

// gzipWriterPool reuses gzip.Writer instances across requests. Each
// writer carries ~32 KiB of compression-window state — pooling avoids
// allocating that on every response. The pool is safe for concurrent
// use; sync.Pool documentation guarantees New is called when the
// pool is empty.
var gzipWriterPool = sync.Pool{
	New: func() any {
		// io.Discard is overwritten with the real sink in Reset
		// before the writer is used. Default compression level is
		// gzip.DefaultCompression (level 6) — a sweet spot between
		// CPU cost and ratio. Higher levels burn 2-3× the CPU for
		// 5-8% extra compression on JSON; not worth it for hot paths.
		return gzip.NewWriter(io.Discard)
	},
}

// compressibleTypes lists the leading Content-Type tokens (before
// any `;` parameter) we ARE willing to gzip. The match is prefix-
// based: `application/json` and `application/ld+json` both match
// `application/json`-family. Anything not in this allow-list passes
// through raw, which keeps already-compressed payloads (images,
// video, archives) untouched.
var compressibleTypes = []string{
	"text/",
	"application/json",
	"application/xml",
	"application/javascript",
	"application/x-ndjson",
	"application/manifest+json",
	"image/svg+xml", // SVG is XML text — compresses well.
}

// Compression returns a middleware that gzip-encodes responses for
// clients that advertise `Accept-Encoding: gzip` and whose response
// body exceeds CompressionMinBytes after the first buffered write.
func Compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !clientAcceptsGzip(r) || !methodHasBody(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		// Tell shared caches that gzip vs identity is a separate
		// representation. The PublicCache middleware already adds
		// `Vary: Accept-Encoding` for cached routes; for everything
		// else we add it here so the browser HTTP cache keys by
		// encoding too.
		w.Header().Add("Vary", "Accept-Encoding")

		cw := &compressingResponseWriter{
			ResponseWriter: w,
			req:            r,
			buf:            getBuffer(),
		}
		defer func() {
			// Flush whatever we still have buffered, in raw or gzip
			// form depending on size/type. Then return the buffer
			// to the pool — releaseBuffer is nil-safe.
			cw.finalize()
			releaseBuffer(cw.buf)
		}()

		next.ServeHTTP(cw, r)
	})
}

// clientAcceptsGzip reports whether the request's `Accept-Encoding`
// header contains a `gzip` token. The check is intentionally
// permissive: q-values are ignored (a `gzip;q=0` would technically
// veto, but in practice clients never advertise gzip and then veto
// it). We tokenise on commas and trim whitespace so headers like
// `br, gzip, deflate` match.
func clientAcceptsGzip(r *http.Request) bool {
	for _, token := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		token = strings.TrimSpace(token)
		// Strip any q-value parameter — "gzip;q=1.0" → "gzip".
		if i := strings.Index(token, ";"); i >= 0 {
			token = strings.TrimSpace(token[:i])
		}
		if strings.EqualFold(token, "gzip") {
			return true
		}
	}
	return false
}

// methodHasBody returns false for HTTP methods whose responses
// conventionally lack a body (HEAD). We still allow GET / POST / PUT
// / PATCH / DELETE because all of those routinely return JSON in
// this codebase (DELETE returns the deleted entity, PATCH returns
// the new state, etc.).
func methodHasBody(method string) bool {
	return method != http.MethodHead
}

// compressingResponseWriter buffers the first writes from the
// handler, decides at flush time whether the payload is worth
// compressing, and either writes raw bytes or a gzip stream to the
// underlying ResponseWriter.
type compressingResponseWriter struct {
	http.ResponseWriter
	req         *http.Request
	buf         *bytes.Buffer
	status      int
	wroteHeader bool

	// Once a decision is made (compress vs raw), the writer pins
	// itself: gz != nil means we have committed to gzip and
	// subsequent Writes stream through it; passthrough == true means
	// we have committed to raw and subsequent Writes go straight
	// through to ResponseWriter. Until either is set we keep
	// buffering.
	gz          *gzip.Writer
	passthrough bool
}

// WriteHeader records the status but does NOT flush headers yet —
// we may still need to add `Content-Encoding: gzip` and strip
// `Content-Length` based on what the body looks like.
//
// Idempotent: extra calls are dropped (mirrors net/http stdlib
// semantics — only the first call matters).
func (c *compressingResponseWriter) WriteHeader(status int) {
	if c.wroteHeader {
		return
	}
	c.status = status
	c.wroteHeader = true

	// 204 / 304 carry no body. Forward the status and flip to
	// passthrough so any stray Write is forwarded raw (it would be
	// a protocol violation, but we don't want to gzip-frame zero
	// bytes either).
	if status == http.StatusNoContent || status == http.StatusNotModified {
		c.passthrough = true
		c.ResponseWriter.WriteHeader(status)
		return
	}
}

// Write either accumulates into the buffer (while still deciding)
// or streams through the committed writer (gzip or raw).
func (c *compressingResponseWriter) Write(p []byte) (int, error) {
	// Implicit 200 if the handler called Write before WriteHeader.
	if !c.wroteHeader {
		c.status = http.StatusOK
		c.wroteHeader = true
	}

	if c.passthrough {
		return c.ResponseWriter.Write(p)
	}
	if c.gz != nil {
		return c.gz.Write(p)
	}

	// Still buffering. If this write tips us over the threshold
	// (or we already had buffered data), commit to a decision now.
	if _, err := c.buf.Write(p); err != nil {
		return 0, err
	}
	if c.buf.Len() >= CompressionMinBytes {
		if err := c.commit(); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// commit decides compress-or-raw based on the current buffer,
// flushes the buffered bytes through the chosen writer, and pins
// the writer so subsequent Write() calls take the fast path.
func (c *compressingResponseWriter) commit() error {
	if c.gz != nil || c.passthrough {
		return nil // already committed
	}

	if c.shouldCompress() {
		c.prepareGzipHeaders()
		c.ResponseWriter.WriteHeader(c.status)

		gz := acquireGzipWriter(c.ResponseWriter)
		c.gz = gz
		if _, err := gz.Write(c.buf.Bytes()); err != nil {
			return err
		}
	} else {
		c.passthrough = true
		c.ResponseWriter.WriteHeader(c.status)
		if _, err := c.ResponseWriter.Write(c.buf.Bytes()); err != nil {
			return err
		}
	}
	c.buf.Reset()
	return nil
}

// shouldCompress returns true when the buffered payload is worth
// compressing: the size is above threshold AND the Content-Type is
// compressible AND the response is not already encoded.
func (c *compressingResponseWriter) shouldCompress() bool {
	if c.buf.Len() < CompressionMinBytes {
		return false
	}
	// A handler that already set Content-Encoding (e.g. pre-gzipped
	// blob streamed from R2) must NOT be re-encoded. The header
	// being present at all is the signal — we don't try to detect
	// the encoding name.
	if c.ResponseWriter.Header().Get("Content-Encoding") != "" {
		return false
	}
	contentType := c.ResponseWriter.Header().Get("Content-Type")
	if contentType == "" {
		// http.DetectContentType is good enough for the threshold
		// check — it sniffs the first 512 bytes. If the handler
		// forgot to set the header, we still want to compress JSON
		// that we recognise.
		contentType = http.DetectContentType(c.buf.Bytes())
	}
	if i := strings.Index(contentType, ";"); i >= 0 {
		contentType = contentType[:i]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	for _, prefix := range compressibleTypes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	return false
}

// prepareGzipHeaders mutates the response headers to advertise
// gzip encoding and drops Content-Length (it is now stale).
func (c *compressingResponseWriter) prepareGzipHeaders() {
	h := c.ResponseWriter.Header()
	h.Set("Content-Encoding", "gzip")
	h.Del("Content-Length")
	// Accept-Ranges does not apply to a transformed body. Strip
	// it to avoid a misleading hint that the client could resume
	// from a byte offset — the offset is into the encoded stream,
	// which the client cannot reconstruct.
	h.Del("Accept-Ranges")
}

// finalize commits any remaining buffer and closes the gzip writer.
// Safe to call even if the handler wrote zero bytes.
func (c *compressingResponseWriter) finalize() {
	if !c.wroteHeader {
		// Empty response with no explicit status — emit 200.
		c.status = http.StatusOK
		c.wroteHeader = true
		c.ResponseWriter.WriteHeader(http.StatusOK)
		return
	}
	if c.passthrough || c.gz != nil {
		if c.gz != nil {
			// Closing a gzip.Writer flushes the deflate trailer +
			// Adler-32. Must happen before we return the writer
			// to the pool, otherwise pool reuse would discard the
			// tail of a previous stream into the next response.
			_ = c.gz.Close()
			releaseGzipWriter(c.gz)
			c.gz = nil
		}
		return
	}
	// Still buffering — handler wrote less than the threshold.
	// Flush as raw.
	c.passthrough = true
	c.ResponseWriter.WriteHeader(c.status)
	if c.buf.Len() > 0 {
		_, _ = c.ResponseWriter.Write(c.buf.Bytes())
	}
}

// Flush propagates a flush request to the underlying writer.
// Compressed streams must Flush the gzip writer first so the deflate
// state catches up; otherwise a streaming endpoint (SSE, chunked
// progress) would see no bytes until the gzip block fills.
func (c *compressingResponseWriter) Flush() {
	if c.gz != nil {
		_ = c.gz.Flush()
	}
	if !c.wroteHeader {
		// http.ResponseWriter.Flush requires a prior WriteHeader.
		// Force the implicit 200 the handler "intended".
		c.WriteHeader(http.StatusOK)
	}
	if c.buf != nil && c.buf.Len() > 0 && !c.passthrough && c.gz == nil {
		// We were still buffering; flush requires committing now.
		_ = c.commit()
	}
	if flusher, ok := c.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack passes the connection take-over through to the underlying
// writer. WebSocket / SSE upgrade handshakes go through Hijack and
// must NEVER be wrapped in gzip — fortunately Hijack returns the raw
// net.Conn so the gzip layer is naturally bypassed.
func (c *compressingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("compression: underlying ResponseWriter does not support Hijack")
}

// ----- pool helpers -----

func acquireGzipWriter(w io.Writer) *gzip.Writer {
	gz, _ := gzipWriterPool.Get().(*gzip.Writer)
	if gz == nil {
		gz = gzip.NewWriter(w)
	} else {
		gz.Reset(w)
	}
	return gz
}

func releaseGzipWriter(gz *gzip.Writer) {
	// Reset to io.Discard before pooling so a future caller that
	// forgets to Reset before writing doesn't accidentally flush
	// bytes into a stale ResponseWriter (would be a memory-safety
	// issue on the next response).
	gz.Reset(io.Discard)
	gzipWriterPool.Put(gz)
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	b, _ := bufferPool.Get().(*bytes.Buffer)
	if b == nil {
		return new(bytes.Buffer)
	}
	b.Reset()
	return b
}

func releaseBuffer(b *bytes.Buffer) {
	if b == nil {
		return
	}
	// Drop buffers that grew oversized to avoid pinning megabytes
	// of memory per long-lived process. 64 KiB cap matches the
	// typical max JSON payload — bigger than that almost certainly
	// streamed through, not buffered.
	if b.Cap() > 64*1024 {
		return
	}
	b.Reset()
	bufferPool.Put(b)
}
