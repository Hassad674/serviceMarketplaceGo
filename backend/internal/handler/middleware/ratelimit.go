package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/pkg/response"
)

// RateLimitClass identifies a single sliding-window quota.
type RateLimitClass string

const (
	// RateLimitClassGlobal applies to every request, keyed by IP.
	RateLimitClassGlobal RateLimitClass = "global"
	// RateLimitClassMutation applies to authenticated POST/PUT/PATCH/DELETE,
	// keyed by user_id (with IP fallback for unauthenticated callers).
	RateLimitClassMutation RateLimitClass = "mutation"
	// RateLimitClassUpload applies to multipart uploads, keyed by user_id.
	RateLimitClassUpload RateLimitClass = "upload"

	// RateLimitClassAuthLogin throttles POST /auth/login. Keyed by
	// client IP (/24 for IPv4, /64 for IPv6). 10/min default — tight
	// enough to defeat credential stuffing, loose enough to absorb a
	// legitimate user typing the wrong password a few times in a row.
	RateLimitClassAuthLogin RateLimitClass = "auth_login"

	// RateLimitClassAuth2FAVerify throttles POST
	// /auth/login/verify-2fa. Keyed by client IP because the user is
	// not yet authenticated when this endpoint fires (the access token
	// only ships in the response). 10/min default mirrors the login
	// class so a bot brute-forcing 6-digit codes burns its budget
	// fast.
	RateLimitClassAuth2FAVerify RateLimitClass = "auth_2fa_verify"

	// RateLimitClassAuth2FAEnable throttles POST /me/two-factor/enable.
	// Keyed by user_id because the endpoint is auth-required and the
	// concern is email-bombing: an attacker with a stolen session
	// could otherwise call /enable repeatedly to spam the inbox with
	// 2FA setup emails. 5/min default.
	RateLimitClassAuth2FAEnable RateLimitClass = "auth_2fa_enable"

	// RateLimitClassPasswordReset throttles POST /auth/forgot-password.
	// Keyed by sha256(email) extracted from the request body — keying
	// by IP alone would let an attacker iterate emails from a single
	// IP, while keying by user_id is impossible (the user is
	// anonymous). 3/min default — much tighter than the legacy
	// per-handler BruteForceService 3/hour cap because the
	// middleware fires BEFORE any service work.
	RateLimitClassPasswordReset RateLimitClass = "password_reset"
)

// RateLimitPolicy bundles a class label with its window + cap. The
// implementation is parameterised so the same Redis-backed limiter can
// host every quota the platform needs without dedicated adapters.
type RateLimitPolicy struct {
	Class  RateLimitClass
	Limit  int
	Window time.Duration
}

// Default policies — safe-for-production caps that absorb a single
// real user behind a CGNAT-class shared IP (Free Mobile, Orange,
// Bouygues) without throttling them. Before these caps were bumped,
// a mobile user behind CGNAT shared a single /24 IPv4 with hundreds
// of other clients and was randomly 429'd whenever a neighbour
// polled. The new defaults add comfortable headroom while specific
// auth/2FA/password-reset classes retain tight caps that prevent
// abuse on the high-value endpoints. See `route_*` wiring for the
// per-endpoint policies layered on top of these globals.
var (
	DefaultGlobalPolicy   = RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 600, Window: time.Minute}
	DefaultMutationPolicy = RateLimitPolicy{Class: RateLimitClassMutation, Limit: 120, Window: time.Minute}
	DefaultUploadPolicy   = RateLimitPolicy{Class: RateLimitClassUpload, Limit: 30, Window: time.Minute}

	// Specific-endpoint policies. These are intentionally tight: the
	// targeted endpoints are high-value (account takeover, email
	// bombing) and have a known, low legitimate request rate.
	DefaultAuthLoginPolicy     = RateLimitPolicy{Class: RateLimitClassAuthLogin, Limit: 10, Window: time.Minute}
	DefaultAuth2FAVerifyPolicy = RateLimitPolicy{Class: RateLimitClassAuth2FAVerify, Limit: 10, Window: time.Minute}
	DefaultAuth2FAEnablePolicy = RateLimitPolicy{Class: RateLimitClassAuth2FAEnable, Limit: 5, Window: time.Minute}
	DefaultPasswordResetPolicy = RateLimitPolicy{Class: RateLimitClassPasswordReset, Limit: 3, Window: time.Minute}
)

// keyFn extracts the throttle key from a request. Returning ("", false)
// short-circuits the limiter — useful for routes that should be
// skipped entirely (e.g. auth-class limiter on a public endpoint).
type keyFn func(r *http.Request) (string, bool)

// rateLimitScript performs a sliding-window check against a Redis
// sorted set. ZREMRANGEBYSCORE drops entries older than the window;
// ZADD inserts the current timestamp; ZCARD returns the new count;
// EXPIRE refreshes the TTL so unused keys evict themselves. Doing all
// four in one round-trip removes the race between read + write.
//
// KEYS[1]: sorted set key
// ARGV[1]: now (unix nanos)
// ARGV[2]: window cutoff (unix nanos)
// ARGV[3]: ttl seconds (window seconds)
// ARGV[4]: limit
//
// Returns: { count, allowed (1/0) }
var slidingWindowScript = goredis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, ARGV[2])
redis.call('ZADD', KEYS[1], ARGV[1], ARGV[1])
redis.call('EXPIRE', KEYS[1], ARGV[3])
local count = redis.call('ZCARD', KEYS[1])
if count > tonumber(ARGV[4]) then
	return {count, 0}
else
	return {count, 1}
end
`)

// RateLimiter holds shared state for the four-class Redis-backed
// sliding-window limiter. The legacy in-memory limiter has been
// retired — any production deployment running multiple instances
// must use the Redis-backed implementation so the quota is shared
// across pods.
//
// F.5 S7 — failure-mode policy. A Redis blip in the rate-limiter
// path used to fail-OPEN unconditionally, which silently disables
// throttling on a partial outage. The `failClosedInProd` flag, set
// from cfg.IsProduction() at wiring time, switches the policy to
// fail-CLOSED in production: a Redis error returns 503 to the
// client. In dev/test the legacy fail-OPEN behaviour is preserved
// so a contributor's broken local Redis never blocks every
// endpoint, but the error is logged at slog.Error level so the
// operator sees it on the next test run.
type RateLimiter struct {
	client           *goredis.Client
	trustedProxies   []*net.IPNet
	failClosedInProd bool
}

// NewRateLimiter returns a fresh limiter wired to the given Redis
// client. The trustedProxies CIDR list controls when the limiter
// honors the X-Forwarded-For header. In production this MUST be
// populated with the load balancer's CIDRs so downstream IPs are
// honored. In dev with no upstream proxy, leave it empty so spoofed
// XFF headers are ignored.
//
// Backwards-compatible signature for callers that don't yet pass a
// production flag — those keep the legacy fail-OPEN behaviour. New
// callers (cmd/api/main.go) should use NewRateLimiterWithPolicy.
func NewRateLimiter(client *goredis.Client, trustedProxies []*net.IPNet) *RateLimiter {
	return &RateLimiter{client: client, trustedProxies: trustedProxies}
}

// NewRateLimiterWithPolicy is the production-ready constructor. When
// failClosedInProd is true, a Redis error in the throttle path is
// surfaced as 503 Service Unavailable. When false, the request goes
// through (with a slog.Error breadcrumb).
func NewRateLimiterWithPolicy(client *goredis.Client, trustedProxies []*net.IPNet, failClosedInProd bool) *RateLimiter {
	return &RateLimiter{
		client:           client,
		trustedProxies:   trustedProxies,
		failClosedInProd: failClosedInProd,
	}
}

// ParseTrustedProxies converts a comma-separated string of CIDRs into
// a slice of *net.IPNet. Empty entries are skipped; an invalid CIDR
// returns an error so misconfiguration surfaces at boot rather than
// silently disabling proxy parsing.
func ParseTrustedProxies(raw string) ([]*net.IPNet, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]*net.IPNet, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Allow bare IPs by promoting them to /32 (v4) or /128 (v6).
		if !strings.Contains(p, "/") {
			ip := net.ParseIP(p)
			if ip == nil {
				return nil, fmt.Errorf("invalid trusted proxy IP %q", p)
			}
			if ip.To4() != nil {
				p = ip.String() + "/32"
			} else {
				p = ip.String() + "/128"
			}
		}
		_, cidr, err := net.ParseCIDR(p)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", p, err)
		}
		out = append(out, cidr)
	}
	return out, nil
}

// ClientIP extracts the client IP from a request. When RemoteAddr is
// from a trusted proxy CIDR, the leftmost public IP from
// X-Forwarded-For is honored; otherwise XFF is ignored to prevent
// spoofing.
//
// Exposed publicly so non-rate-limit middleware (e.g. the auth
// handler's per-IP brute-force gate, N4) can derive the same
// limiter-normalised key without duplicating the trusted-proxy /
// IPv6-/64 logic. The output is suitable for use as a Redis key
// directly.
//
// F.5 S6 — IPv6 /64 normalisation. An IPv6 attacker with a routed /64
// has 2^64 distinct addresses. Without normalisation, the rate
// limiter sees each address as a separate slot and is effectively
// disabled. We mask the IPv6 IP to the network's /64 (the smallest
// allocation block any reasonable ISP hands out) so the throttle
// applies per-network rather than per-address. IPv4 keeps its full
// /32 because abuse from a single IPv4 is the only granular signal
// available — anything coarser would over-throttle shared NATs.
func (rl *RateLimiter) ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote := net.ParseIP(host)
	if remote == nil {
		return host
	}
	if !rl.isTrustedProxy(remote) {
		return normaliseIPForLimiter(remote)
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return normaliseIPForLimiter(remote)
	}
	for _, candidate := range strings.Split(xff, ",") {
		candidate = strings.TrimSpace(candidate)
		if ip := net.ParseIP(candidate); ip != nil {
			return normaliseIPForLimiter(ip)
		}
	}
	return normaliseIPForLimiter(remote)
}

// normaliseIPForLimiter returns a key that buckets IPv6 traffic per
// /64 prefix and IPv4 per /32 (i.e. the address itself). The
// returned string is stable across processes — built from
// net.IP.Mask + textual rendering, not from a private hash — so
// distributed Redis keys collide cross-instance as expected.
func normaliseIPForLimiter(ip net.IP) string {
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		// IPv4: keep per-address granularity. /32 mask.
		return v4.String()
	}
	// Treat anything that is NOT a v4-in-v6 mapping as IPv6 and apply
	// the /64 prefix — high 8 bytes only.
	prefix := ip.Mask(net.CIDRMask(64, 128))
	if prefix == nil {
		return ip.String()
	}
	// Tag with a "/64" suffix so an IPv6 prefix never collides with an
	// IPv4 string, AND so the key is human-grepable in Redis.
	return prefix.String() + "/64"
}

func (rl *RateLimiter) isTrustedProxy(ip net.IP) bool {
	for _, cidr := range rl.trustedProxies {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// allow runs the sliding-window check and returns:
//   - count   total requests in the current window after this one
//   - allowed whether this request is within the cap
//   - retry   how long the caller should wait before the next try
//   - err     Redis-level failure (caller decides fail-open vs fail-closed)
func (rl *RateLimiter) allow(ctx context.Context, policy RateLimitPolicy, key string) (count int, allowed bool, retry time.Duration, err error) {
	if key == "" {
		return 0, true, 0, nil
	}
	now := time.Now()
	cutoff := now.Add(-policy.Window).UnixNano()
	redisKey := fmt.Sprintf("ratelimit:%s:%s", policy.Class, key)
	res, err := slidingWindowScript.Run(
		ctx, rl.client, []string{redisKey},
		now.UnixNano(),
		cutoff,
		int(policy.Window.Seconds()),
		policy.Limit,
	).Slice()
	if err != nil {
		return 0, true, 0, err // fail open
	}
	if len(res) != 2 {
		return 0, true, 0, fmt.Errorf("unexpected rate limit script result: %v", res)
	}
	count = int(res[0].(int64))
	allowed = res[1].(int64) == 1
	if !allowed {
		retry = policy.Window
	}
	return count, allowed, retry, nil
}

// Middleware returns an http.Handler middleware that enforces the
// given policy. The keyFn picks the throttle key per request — the
// stock helpers below cover the common cases (IP-based, user-based,
// authenticated-only).
func (rl *RateLimiter) Middleware(policy RateLimitPolicy, key keyFn) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			throttleKey, ok := key(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			count, allowed, retry, err := rl.allow(r.Context(), policy, throttleKey)
			if err != nil {
				// F.5 S7 — fail policy is environment-aware:
				//   * production : fail-CLOSED → 503. A Redis blip MUST
				//     NOT silently disable throttling on a public API.
				//   * dev/test    : fail-OPEN. A contributor with a broken
				//     local Redis still gets a working app — the slog.Error
				//     line surfaces the issue so they fix it.
				if rl.failClosedInProd {
					slog.Error("ratelimit: Redis error — failing closed",
						"class", policy.Class,
						"key", throttleKey,
						"error", err)
					response.Error(w, http.StatusServiceUnavailable,
						"rate_limit_unavailable",
						"throttling backend is degraded — retry shortly")
					return
				}
				slog.Error("ratelimit: Redis error — failing open in non-prod",
					"class", policy.Class,
					"key", throttleKey,
					"error", err)
				next.ServeHTTP(w, r)
				return
			}
			remaining := policy.Limit - count
			if remaining < 0 {
				remaining = 0
			}
			resetAt := time.Now().Add(policy.Window).Unix()
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(policy.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))
			if !allowed {
				retrySeconds := int(retry.Seconds())
				if retrySeconds < 1 {
					retrySeconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
				// Structured 429 metric (proxy for a real Prometheus
				// counter). The key is anonymised so a grep across
				// production logs never leaks raw emails / IPs / UUIDs:
				// only the namespace prefix + a sha256 short-fingerprint
				// is emitted. Aggregate via:
				//   `grep -c 'ratelimit: 429 served' | jq` (LogQL etc.).
				_, authenticated := GetUserID(r.Context())
				slog.Warn("ratelimit: 429 served",
					"class", policy.Class,
					"key", anonymiseRateLimitKey(throttleKey),
					"user_authenticated", authenticated,
					"limit", policy.Limit,
					"window_seconds", int(policy.Window.Seconds()),
					"retry_after_seconds", retrySeconds,
					"method", r.Method,
					"path", r.URL.Path)
				response.Error(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKey returns a keyFn that throttles by client IP. Requests with an
// unparseable IP get an empty key, which short-circuits the limiter
// (we cannot meaningfully throttle without a key).
func (rl *RateLimiter) IPKey() keyFn {
	return func(r *http.Request) (string, bool) {
		ip := rl.ClientIP(r)
		if ip == "" {
			return "", false
		}
		return ip, true
	}
}

// UserKey returns a keyFn that throttles by authenticated user_id.
// Anonymous requests are skipped (returning false) so the limiter
// does not double up with the global IP-based throttle on public
// routes.
func UserKey() keyFn {
	return func(r *http.Request) (string, bool) {
		userID, ok := GetUserID(r.Context())
		if !ok || userID.String() == "" {
			return "", false
		}
		return userID.String(), true
	}
}

// UserOrIPKey returns a keyFn that throttles by authenticated user_id
// when present, falling back to client IP when the request is
// anonymous. Used by the P10 mutation rate limit: anonymous POST
// /auth/login + /auth/register attempts must still hit the 30/min cap
// to bound the request volume per source — using `UserKey` alone
// would let unauthenticated mutation traffic fall back to the looser
// 100/min global cap.
//
// The "user_id|ip" key namespace prefix prevents accidental collisions
// between a UUID stringification and a synthetic IPv4-shaped UUID.
func UserOrIPKey(rl *RateLimiter) keyFn {
	if rl == nil {
		// Nil rate limiter would be a wiring bug — degrade to
		// user-only throttling so the route still functions.
		return UserKey()
	}
	return func(r *http.Request) (string, bool) {
		if userID, ok := GetUserID(r.Context()); ok && userID.String() != "" {
			return "user:" + userID.String(), true
		}
		ip := rl.ClientIP(r)
		if ip == "" {
			return "", false
		}
		return "ip:" + ip, true
	}
}

// MutationOnly wraps a keyFn so the limiter only fires on mutating
// HTTP methods (POST/PUT/PATCH/DELETE). Reads pass through
// unthrottled which is correct for the mutation class — read traffic
// is covered by the global IP-based limiter.
func MutationOnly(inner keyFn) keyFn {
	return func(r *http.Request) (string, bool) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			return inner(r)
		default:
			return "", false
		}
	}
}

// EmailKey returns a keyFn that throttles by the hashed "email" field
// of the JSON request body. Used by the password-reset class so an
// attacker iterating across emails from a single IP cannot bypass the
// per-email cap.
//
// The body is read once, then re-attached to r.Body so the downstream
// handler can still decode it. The hash is sha256(lowercased + trimmed
// email) so case/whitespace variants share the same bucket.
//
// If the body is unreadable, lacks an "email" field, or the field is
// empty, the function returns false so the limiter short-circuits —
// the request continues unthrottled by THIS limiter (the global IP
// throttle still applies). This is intentional: a malformed body has
// no legitimate password-reset semantics and rejecting it here would
// be the handler's job, not the limiter's.
//
// Body size is capped at 4 KiB to bound the cost of the buffer copy.
func EmailKey() keyFn {
	return func(r *http.Request) (string, bool) {
		if r.Body == nil {
			return "", false
		}
		// Tee the body so the handler can still decode it. 4 KiB is
		// well above any sane password-reset payload but well below
		// the size where a copy becomes a DoS vector.
		const maxBody = 4 << 10
		buf, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if err != nil {
			return "", false
		}
		_ = r.Body.Close()
		// Re-attach the captured bytes so the handler's decoder gets
		// the same payload. A NopCloser is fine — the original body
		// has already been drained.
		r.Body = io.NopCloser(bytes.NewReader(buf))

		var payload struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(buf, &payload); err != nil {
			return "", false
		}
		email := strings.ToLower(strings.TrimSpace(payload.Email))
		if email == "" {
			return "", false
		}
		return "email:" + hashEmail(email), true
	}
}

// hashEmail returns the first 16 hex chars of sha256(email). 16 chars
// is enough to make pre-image inversion impractical while keeping
// Redis keys short and the structured-log fingerprints grep-friendly.
func hashEmail(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])[:16]
}

// anonymiseRateLimitKey returns a privacy-safe label for the
// structured 429 log. Raw UUIDs and IPs are stripped to a short
// sha256 fingerprint so the log never leaks PII, while still letting
// an operator group bursts coming from "the same key".
func anonymiseRateLimitKey(key string) string {
	if key == "" {
		return ""
	}
	// Preserve the namespace prefix ("user:", "ip:", "email:") so the
	// log line tells the operator what kind of key tripped — but
	// fingerprint the value after the colon.
	prefix := ""
	suffix := key
	if idx := strings.IndexByte(key, ':'); idx >= 0 {
		prefix = key[:idx+1]
		suffix = key[idx+1:]
	}
	sum := sha256.Sum256([]byte(suffix))
	return prefix + hex.EncodeToString(sum[:])[:12]
}
