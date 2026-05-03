package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// IdempotencyHeader is the canonical request header carrying the
// client-generated idempotency key. Mirrors the convention popularised
// by Stripe so SDK-style clients (web, mobile, third-party integrations)
// see a familiar surface.
const IdempotencyHeader = "Idempotency-Key"

// IdempotentReplayedHeader is set on the response when the body was
// served from cache. Lets clients tell apart a fresh execution from a
// safe replay — important for UI affordances that show "this was
// already submitted".
const IdempotentReplayedHeader = "Idempotent-Replayed"

// DefaultIdempotencyTTL controls how long a cached response is honoured.
// 24 hours covers the worst-case "user resumes a half-completed flow
// after sleeping the laptop", which is the primary failure mode that
// motivated SEC-FINAL-02.
const DefaultIdempotencyTTL = 24 * time.Hour

// MaxIdempotencyKeyLength caps the length of an accepted Idempotency-Key
// header. Anything longer is silently treated as no key — a misconfigured
// client should not be able to fill Redis with arbitrary-size keys.
const MaxIdempotencyKeyLength = 200

// IdempotencyCache is the storage interface the middleware needs.
// Two operations are sufficient: probe for an existing record and
// atomically claim a key + persist the eventual response. Splitting
// the contract this way keeps the middleware pure (no Redis dependency
// at compile time) and makes the test harness trivial.
//
// SetNX semantics MUST be respected: only the first concurrent caller
// for a given key persists; every other concurrent attempt receives
// (false, nil) and proceeds as a replay.
type IdempotencyCache interface {
	// Get returns the cached response for key, if any. The returned
	// IdempotentResponse is the verbatim previous reply: status code,
	// headers (a small allow-list), and body bytes. (nil, nil) means
	// "no cache entry"; a non-nil error is a transport-level Redis
	// failure the middleware downgrades to a no-op.
	Get(ctx context.Context, key string) (*IdempotentResponse, error)
	// Set atomically persists a response under key with a TTL. The
	// boolean reports whether the SETNX claim succeeded — false
	// means another concurrent request already won the race and
	// the caller's response is discarded (the client will see a
	// stable replay on the next try).
	Set(ctx context.Context, key string, resp IdempotentResponse, ttl time.Duration) (bool, error)
}

// IdempotentResponse is the cached snapshot of a previous successful
// reply. Only a small set of headers is preserved — Content-Type and
// Location specifically — to avoid leaking Set-Cookie / Authorization
// when a future caller replays under a different identity. Body bytes
// are stored verbatim so the response is byte-identical to the first.
//
// RequestBodyHash is the sha256 of the original request body bytes. On
// replay the middleware recomputes the hash from the incoming request
// and compares: a mismatch is a Stripe-spec body-conflict and yields
// 409 Conflict instead of replaying a stale answer. Empty for cached
// entries written before F.6 B1 — those simply replay without the
// conflict check (safe because the legacy behaviour is what the cache
// previously promised).
type IdempotentResponse struct {
	Status          int               `json:"status"`
	ContentType     string            `json:"content_type"`
	Body            []byte            `json:"body"`
	Headers         map[string]string `json:"headers,omitempty"`
	RequestBodyHash string            `json:"request_body_hash,omitempty"`
}

// captureRecorder is an http.ResponseWriter that buffers everything the
// inner handler writes so the middleware can persist it after the
// handler returns. Status defaults to 200 (matches net/http).
type captureRecorder struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
	wrote  bool
}

func (r *captureRecorder) WriteHeader(code int) {
	if r.wrote {
		return
	}
	r.status = code
	r.wrote = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *captureRecorder) Write(p []byte) (int, error) {
	if !r.wrote {
		r.WriteHeader(http.StatusOK)
	}
	if r.body != nil {
		r.body.Write(p)
	}
	return r.ResponseWriter.Write(p)
}

// Idempotency wraps a handler with idempotency-key support.
//
// Behaviour:
//   - No `Idempotency-Key` header → handler runs normally; response is
//     not cached. This is the non-idempotent fallback so legacy clients
//     still work.
//   - Key present, no cache entry → handler runs, response is captured,
//     and persisted under {user-or-anon}:{key} with the configured TTL.
//   - Key present, cache hit → original status + body are replayed
//     byte-for-byte; the inner handler is NOT invoked. The response
//     adds `Idempotent-Replayed: true` so the caller can tell apart a
//     replay from a fresh execution.
//   - Cache transport failure → middleware logs and falls through to
//     "execute the handler, do not cache" so the platform stays
//     available even when Redis is degraded.
//
// Only success responses (2xx) are cached. A 5xx must NOT be replayed:
// a transient outage mustn't poison subsequent retries. 4xx is also
// skipped because the client typically fixes the request before
// retrying with the same key.
//
// Closes SEC-FINAL-02: the 6 critical POSTs (proposals, jobs,
// disputes, register, team invitations, proposal payment) gain
// retry-safety so a network blip doesn't cause double-creation.
func Idempotency(cache IdempotencyCache) func(http.Handler) http.Handler {
	return IdempotencyWithTTL(cache, DefaultIdempotencyTTL)
}

// IdempotencyWithTTL is the variant exposing the cache TTL. Tests use
// short TTLs to exercise expiry without sleeping; production wires
// DefaultIdempotencyTTL via the bare Idempotency constructor.
func IdempotencyWithTTL(cache IdempotencyCache, ttl time.Duration) func(http.Handler) http.Handler {
	if ttl <= 0 {
		ttl = DefaultIdempotencyTTL
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := strings.TrimSpace(r.Header.Get(IdempotencyHeader))
			if rawKey == "" || len(rawKey) > MaxIdempotencyKeyLength {
				// No header or too long → bypass without caching.
				// Logging the over-length case helps spot misbehaving
				// clients without enabling them to fill Redis.
				if len(rawKey) > MaxIdempotencyKeyLength {
					slog.Warn("idempotency: oversized key, ignored",
						"len", len(rawKey),
						"path", r.URL.Path)
				}
				next.ServeHTTP(w, r)
				return
			}

			// Read + restore the request body so the downstream
			// handler still sees it. We bound the read at 1 MiB
			// (idempotencyBodyReadCap) — anything larger trips the
			// pkg/decode size guard and would have failed anyway, so
			// we cap here to keep memory pressure deterministic.
			bodyBytes, readErr := readAndRestoreBody(r)
			if readErr != nil {
				// Body read failure is a transport-layer issue. The
				// handler will see the same problem and surface its
				// own error — fall through without caching.
				slog.Warn("idempotency: body read failed, executing handler",
					"error", readErr, "path", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}
			bodyHash := hashBody(bodyBytes)

			// Cache key now binds (scope, method, path, rawKey). Two
			// requests under the same Idempotency-Key but different
			// methods or paths get distinct cache entries — the body
			// hash is checked on replay (below) to honour the Stripe
			// spec contract: same key + different body = 409 Conflict.
			fullKey := buildCacheKey(r.Context(), r.Method, r.URL.Path, rawKey)

			// Cache lookup. A transport failure must NEVER block the
			// request — log and fall through to a normal execution.
			cached, err := cache.Get(r.Context(), fullKey)
			if err != nil {
				slog.Warn("idempotency: cache get failed, executing handler",
					"error", err, "key", rawKey)
			} else if cached != nil {
				// Stripe-spec body-conflict check: same key, different
				// body → 409 Conflict. The cached entry's RequestBodyHash
				// is empty for legacy entries written before F.6 B1; we
				// skip the check there to avoid spurious 409s during a
				// rolling deploy.
				if cached.RequestBodyHash != "" && cached.RequestBodyHash != bodyHash {
					writeIdempotencyConflict(w)
					return
				}
				replayCachedResponse(w, cached)
				return
			}

			// First execution path: capture handler output, decide
			// whether to persist, then write through.
			rec := &captureRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
				body:           &bytes.Buffer{},
			}
			next.ServeHTTP(rec, r)

			// Only 2xx is replay-safe. We deliberately skip 4xx (the
			// client retries with the same key after fixing the bug,
			// which we want to re-validate) and 5xx (transient errors
			// must not poison the cache).
			if rec.status < 200 || rec.status >= 300 {
				return
			}

			snap := IdempotentResponse{
				Status:          rec.status,
				ContentType:     rec.Header().Get("Content-Type"),
				Body:            bytes.Clone(rec.body.Bytes()),
				Headers:         captureSafeHeaders(rec.Header()),
				RequestBodyHash: bodyHash,
			}
			// SetNX-style: a concurrent first executor may have raced
			// us — discard the loser's response silently so subsequent
			// replays land on a stable cached body.
			won, setErr := cache.Set(r.Context(), fullKey, snap, ttl)
			if setErr != nil {
				slog.Warn("idempotency: cache set failed",
					"error", setErr, "key", rawKey)
			}
			if !won {
				slog.Debug("idempotency: SETNX lost race, response not cached",
					"key", rawKey)
			}
		})
	}
}

// idempotencyBodyReadCap bounds how many bytes we will buffer for the
// hash. 1 MiB matches pkg/decode.DefaultMaxBodyBytes — handlers that
// legitimately accept larger payloads (file uploads) do not currently
// pass through this middleware.
const idempotencyBodyReadCap = 1 << 20

// readAndRestoreBody reads the entire request body up to the cap and
// restores r.Body so the downstream handler still sees the bytes.
// Returns (nil, nil) for an absent body — that is normal for endpoints
// like /auth/refresh-token where the body is empty.
func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, nil
	}
	limited := io.LimitReader(r.Body, idempotencyBodyReadCap+1)
	buf, err := io.ReadAll(limited)
	if cerr := r.Body.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > idempotencyBodyReadCap {
		// We read past the cap — the request is too big for our
		// idempotency comparison to be meaningful. Restore the
		// truncated body anyway so the handler can decide what to
		// do (typically pkg/decode will return 413).
		r.Body = io.NopCloser(bytes.NewReader(buf))
		return buf, nil
	}
	r.Body = io.NopCloser(bytes.NewReader(buf))
	return buf, nil
}

// hashBody returns the hex-encoded sha256 of the request body bytes.
// Empty body hashes to the sha256 of the empty string — that is the
// canonical anchor so a body-less request that retries with another
// body-less request still matches.
func hashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// writeIdempotencyConflict emits the structured 409 response for the
// Stripe-spec body-conflict case (same key, different body). The error
// code matches the convention used by other middleware errors so the
// frontend can branch on a stable string.
func writeIdempotencyConflict(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_, _ = w.Write([]byte(`{"error":"idempotency_key_conflict","message":"Same Idempotency-Key was used with a different body"}`))
}

// buildCacheKey scopes the key under the authenticated user (or "anon"
// for unauth flows like /auth/register) so two unrelated clients
// reusing the same client-side UUID don't collide. Method and path are
// included so the same key reused on a different endpoint produces a
// distinct cache entry — F.6 B1 closes the bug where a same-key call
// to a different verb / path was silently colliding with a previous
// 2xx response and replaying the wrong answer.
//
// The key is hashed to keep it bounded regardless of the user-supplied
// length and to avoid leaking raw key material into Redis logs.
func buildCacheKey(ctx context.Context, method, path, rawKey string) string {
	scope := "anon"
	if uid, ok := GetUserID(ctx); ok {
		scope = uid.String()
	}
	combined := scope + ":" + method + ":" + path + ":" + rawKey
	sum := sha256.Sum256([]byte(combined))
	return "idempotency:" + hex.EncodeToString(sum[:])
}

// replayCachedResponse writes the cached snapshot to w. The
// Idempotent-Replayed header lets clients distinguish replay from a
// fresh execution.
func replayCachedResponse(w http.ResponseWriter, cached *IdempotentResponse) {
	for k, v := range cached.Headers {
		w.Header().Set(k, v)
	}
	if cached.ContentType != "" {
		w.Header().Set("Content-Type", cached.ContentType)
	}
	w.Header().Set(IdempotentReplayedHeader, "true")
	status := cached.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(cached.Body) > 0 {
		_, _ = w.Write(cached.Body)
	}
}

// safeReplayedHeaders is the allow-list of response headers that are
// safe to re-emit on a replay. Anything else (Set-Cookie, Authorization
// echoes, custom auth headers) is intentionally dropped — replaying
// them under a possibly-different requester is a privilege-escalation
// risk we never accept.
var safeReplayedHeaders = map[string]struct{}{
	"Location":              {},
	"Etag":                  {},
	"X-Request-Id":          {},
	"Cache-Control":         {},
	"Vary":                  {},
}

// captureSafeHeaders extracts the safe-to-replay subset from h.
// Returns nil rather than an empty map when nothing is captured so
// the JSON snapshot stays compact.
func captureSafeHeaders(h http.Header) map[string]string {
	out := make(map[string]string, 4)
	for name := range h {
		if _, ok := safeReplayedHeaders[http.CanonicalHeaderKey(name)]; !ok {
			continue
		}
		out[name] = h.Get(name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// Redis-backed adapter
// ---------------------------------------------------------------------------

// RedisIdempotencyCache implements IdempotencyCache against a go-redis
// client. JSON-encodes the snapshot; uses SETNX so a concurrent racer
// cannot overwrite the winner.
type RedisIdempotencyCache struct {
	client *goredis.Client
}

// NewRedisIdempotencyCache wires the cache. The client is required —
// passing nil panics at boot, which is the correct behaviour because
// the middleware is wired only when Redis is available.
func NewRedisIdempotencyCache(client *goredis.Client) *RedisIdempotencyCache {
	if client == nil {
		panic("middleware.NewRedisIdempotencyCache: nil redis client")
	}
	return &RedisIdempotencyCache{client: client}
}

// Get fetches the cached snapshot. Treats redis.Nil as "no record"
// (returns nil, nil) so the caller sees a clean miss path.
func (c *RedisIdempotencyCache) Get(ctx context.Context, key string) (*IdempotentResponse, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	var resp IdempotentResponse
	if jsonErr := json.Unmarshal(raw, &resp); jsonErr != nil {
		// Corrupted entry — treat as a miss so the next caller
		// re-executes and re-caches with a valid snapshot.
		return nil, nil
	}
	return &resp, nil
}

// Set atomically claims and persists the snapshot. Returns the SETNX
// boolean so callers can skip persistence on a lost race.
func (c *RedisIdempotencyCache) Set(ctx context.Context, key string, resp IdempotentResponse, ttl time.Duration) (bool, error) {
	payload, err := json.Marshal(resp)
	if err != nil {
		return false, err
	}
	return c.client.SetNX(ctx, key, payload, ttl).Result()
}

// ---------------------------------------------------------------------------
// Helpers exposed for tests
// ---------------------------------------------------------------------------

// CopyBody is a small helper used by handler tests that want to
// inspect what the middleware persisted. Not part of the public API.
func CopyBody(r io.Reader) []byte {
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, r)
	return buf.Bytes()
}
