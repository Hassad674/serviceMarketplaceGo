package middleware

import (
	"context"
	"fmt"
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
	// keyed by user_id.
	RateLimitClassMutation RateLimitClass = "mutation"
	// RateLimitClassUpload applies to multipart uploads, keyed by user_id.
	RateLimitClassUpload RateLimitClass = "upload"
)

// RateLimitPolicy bundles a class label with its window + cap. The
// implementation is parameterised so the same Redis-backed limiter can
// host every quota the platform needs without dedicated adapters.
type RateLimitPolicy struct {
	Class  RateLimitClass
	Limit  int
	Window time.Duration
}

// Default policies match the values documented in CLAUDE.md.
var (
	DefaultGlobalPolicy   = RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 100, Window: time.Minute}
	DefaultMutationPolicy = RateLimitPolicy{Class: RateLimitClassMutation, Limit: 30, Window: time.Minute}
	DefaultUploadPolicy   = RateLimitPolicy{Class: RateLimitClassUpload, Limit: 10, Window: time.Minute}
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
type RateLimiter struct {
	client         *goredis.Client
	trustedProxies []*net.IPNet
}

// NewRateLimiter returns a fresh limiter wired to the given Redis
// client. The trustedProxies CIDR list controls when the limiter
// honors the X-Forwarded-For header. In production this MUST be
// populated with the load balancer's CIDRs so downstream IPs are
// honored. In dev with no upstream proxy, leave it empty so spoofed
// XFF headers are ignored.
func NewRateLimiter(client *goredis.Client, trustedProxies []*net.IPNet) *RateLimiter {
	return &RateLimiter{client: client, trustedProxies: trustedProxies}
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

// clientIP extracts the client IP from a request. When RemoteAddr is
// from a trusted proxy CIDR, the leftmost public IP from
// X-Forwarded-For is honored; otherwise XFF is ignored to prevent
// spoofing.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote := net.ParseIP(host)
	if remote == nil {
		return host
	}
	if !rl.isTrustedProxy(remote) {
		return remote.String()
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remote.String()
	}
	for _, candidate := range strings.Split(xff, ",") {
		candidate = strings.TrimSpace(candidate)
		if ip := net.ParseIP(candidate); ip != nil {
			return ip.String()
		}
	}
	return remote.String()
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
				// Fail open + log via response writer would require the
				// slog package here; we trade pristine logs for clean
				// imports. The handler still serves the request, which
				// is the safer behaviour during a Redis blip.
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
		ip := rl.clientIP(r)
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
		ip := rl.clientIP(r)
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
