package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// devJWTFallback is the JWT_SECRET shipped in .env.example so a fresh
// `make run` works without manual setup. Because this repo is
// open-source the value is public — accepting it in production is
// equivalent to publishing the signing key. Validate() refuses to boot
// when this exact value is used in production. (SEC-04)
const devJWTFallback = "dev-secret-change-me"

// minJWTSecretBytes is the lower bound enforced in production. 32
// bytes (256 bits) matches the security strength of HS256, the JWT
// algorithm we use. See NIST SP 800-131A for the rationale.
const minJWTSecretBytes = 32

// devStorageFallback is MinIO's default credential pair shipped in
// docker-compose.yml. Same open-source-fallback risk as the JWT
// fallback: anybody with the repo can sign requests against any prod
// deployment that forgot to override it. Validate() refuses to boot
// when these defaults are used in production.
const devStorageFallback = "minioadmin"

type Config struct {
	Port             string
	Env              string
	DatabaseURL      string
	// DatabaseAdminURL is the privileged (BYPASSRLS) connection string
	// used by the system-actor pool — schedulers, webhooks, GDPR purge,
	// admin overrides. Optional. When empty, wireInfrastructure falls
	// back to DatabaseURL with a WARN log so the rollout from the
	// single-pool model can ship without a hard dependency on the new
	// env var. See backend/docs/rls-rollout.md for the production rotation.
	DatabaseAdminURL string
	RedisURL         string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration
	SessionTTL       time.Duration
	CookieSecure     bool
	AllowedOrigins   []string
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool
	StoragePublicURL string
	// AuditColdStorageBucket is the destination bucket for the B.2
	// cold-tier audit log archive (gzipped JSONL bundles uploaded to
	// R2 when an audit_logs_archive row is older than the cold-tier
	// cutoff). Empty = feature disabled, the archive_to_r2 retention
	// policy skips itself. Allowed to equal StorageBucket; the sweep
	// scopes its keys under the `audit-cold/` prefix so collisions
	// with user uploads are impossible.
	AuditColdStorageBucket string
	ResendAPIKey         string
	ResendDevRedirectTo  string // optional: if set, all outgoing emails are routed here (dev/staging sandbox)
	EmailFrom            string // RFC 5322 sender, e.g. "Marketplace <noreply@designedtrust.com>"
	FrontendURL          string
	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret   string
	FCMCredentialsPath   string
	StripeSecretKey        string
	StripePublishableKey   string
	StripeWebhookSecret    string
	RekognitionEnabled              bool
	RekognitionRegion               string
	RekognitionThreshold            float64
	RekognitionAutoRejectThreshold  float64
	RekognitionRoleARN              string
	S3ModerationBucket              string
	SNSTopicARN                     string
	SQSQueueURL                     string
	AnthropicAPIKey                 string

	// TextModerationProvider selects which adapter powers automated
	// text moderation (messages, reviews, future scope). Allowed values:
	//   "openai"     -> adapter/openai.TextModerationService (default, free, FR-native)
	//   "comprehend" -> adapter/comprehend.TextModerationService (legacy, EN only, needs
	//                    a region where DetectToxicContent is available)
	//   "noop"       -> adapter/noop.TextModerationService (disables moderation — dev/CI)
	// Missing or unknown values fall back to "noop" to fail safe.
	TextModerationProvider string

	// Search engine (Typesense) — MANDATORY since phase 4. The
	// legacy SQL path was retired after the 30-day grace period
	// ended in April 2026. TypesenseConfigured must return true for
	// the app to boot; /ready fails 503 when the cluster is
	// unreachable so load balancers rotate misconfigured instances
	// out instead of silently returning empty search results.
	TypesenseHost         string
	TypesenseAPIKey       string
	OpenAIAPIKey          string
	OpenAIEmbeddingsModel string

	// TrustedProxies is a comma-separated CIDR list. The rate limiter
	// (SEC-11) honors X-Forwarded-For ONLY when r.RemoteAddr falls
	// inside one of these CIDRs. In production, set this to your load
	// balancer's source range. In dev with no proxy, leave it empty —
	// the limiter will then ignore spoofed XFF headers and key off
	// r.RemoteAddr directly.
	TrustedProxies string

	// RateLimit* let an operator override every rate-limit class
	// without code changes. Defaults are encoded in
	// middleware.DefaultXxxPolicy and are bumped to safe-for-prod
	// values that absorb a single CGNAT-shared mobile user:
	//   - Global:           600 req/min (per IP /24 or IPv6 /64)
	//   - Mutation:         120 req/min (per user_id, IP fallback)
	//   - Upload:            30 req/min (per user_id)
	//   - AuthLogin:         10 req/min (per IP) — credential stuffing
	//   - Auth2FAVerify:     10 req/min (per IP) — code brute-force
	//   - Auth2FAEnable:      5 req/min (per user_id) — email bombing
	//   - PasswordReset:      3 req/min (per email hash) — inbox abuse
	// A value of 0 (or unset env var) means "use default", so the
	// production deployment does not need to set anything unless
	// the operator explicitly wants to widen or tighten a class.
	// RATE-LIMIT-PROD.
	//
	// Env var mapping (all integers, per-minute):
	//   RATE_LIMIT_GLOBAL_PER_MINUTE          -> RateLimitGlobalPerMinute
	//   RATE_LIMIT_MUTATION_PER_MINUTE        -> RateLimitMutationPerMinute
	//   RATE_LIMIT_UPLOAD_PER_MINUTE          -> RateLimitUploadPerMinute
	//   RATE_LIMIT_AUTH_LOGIN_PER_MINUTE      -> RateLimitAuthLoginPerMinute
	//   RATE_LIMIT_AUTH_2FA_VERIFY_PER_MINUTE -> RateLimitAuth2FAVerifyPerMinute
	//   RATE_LIMIT_AUTH_2FA_ENABLE_PER_MINUTE -> RateLimitAuth2FAEnablePerMinute
	//   RATE_LIMIT_PASSWORD_RESET_PER_MINUTE  -> RateLimitPasswordResetPerMinute
	RateLimitGlobalPerMinute         int
	RateLimitMutationPerMinute       int
	RateLimitUploadPerMinute         int
	RateLimitAuthLoginPerMinute      int
	RateLimitAuth2FAVerifyPerMinute  int
	RateLimitAuth2FAEnablePerMinute  int
	RateLimitPasswordResetPerMinute  int

	// NotificationWorkerConcurrency is the number of parallel
	// processors the notification delivery worker spawns. Defaults
	// to 5 (BUG-16). Override via NOTIFICATION_WORKER_CONCURRENCY
	// when sizing for higher load. Setting it to 1 reproduces the
	// pre-fix single-threaded behaviour for debugging.
	NotificationWorkerConcurrency int

	// GDPRAnonymizationSalt is the secret salt used by the GDPR
	// purge cron when computing sha256(email + salt) for audit-log
	// anonymization. MUST be set in production; the dev fallback is
	// "dev-salt-not-for-prod" and Validate() refuses to boot in
	// production with that value.
	GDPRAnonymizationSalt string

	// RetentionInterval overrides the cadence at which the retention
	// scheduler (Phase B.1 of the GDPR roadmap) sweeps stale rows
	// from messages, notifications, device_tokens, search_queries and
	// audit_logs. Default is 1h in production and 1m in development —
	// see app/retention/scheduler.go for the constants. Setting this
	// to zero falls back to the per-env default.
	RetentionInterval time.Duration

	// GDPRContactEmail is the recipient address for compliance
	// notifications surfaced by the user-facing legal endpoints —
	// currently used by the RGPD art. 22 appeal handler to flag a new
	// human-review request to the admin inbox. Empty silences the
	// notification (the appeal row is still persisted) so dev / CI
	// stacks without an email channel do not break the flow.
	GDPRContactEmail string

	// PostHogProjectKey is the public project token used by the
	// posthog-go SDK to ship server-side events. Same value as the
	// browser-side NEXT_PUBLIC_POSTHOG_KEY — PostHog deliberately
	// shares the public project token between server and client.
	// Empty disables analytics (the noop adapter is wired instead);
	// the wiring fail-opens with a WARN log so a missing key never
	// breaks the boot.
	PostHogProjectKey string
	// PostHogHost is the regional PostHog endpoint — https://eu.posthog.com
	// for the RGPD-friendly EU project, or https://us.posthog.com.
	// Defaults to EU because the marketplace stores data in the EU.
	PostHogHost string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		Env:              getEnv("ENV", "development"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/marketplace?sslmode=disable"),
		DatabaseAdminURL: getEnv("DATABASE_URL_ADMIN", ""),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6380"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTAccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m")),
		JWTRefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h")), // 7 days
		SessionTTL:       parseDuration(getEnv("SESSION_TTL", "336h")),        // 14 days
		CookieSecure:     getEnv("ENV", "development") == "production",
		AllowedOrigins:   strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "marketplace"),
		StorageUseSSL:    getEnv("STORAGE_USE_SSL", "false") == "true",
		StoragePublicURL: getEnv("STORAGE_PUBLIC_URL", "http://localhost:9000/marketplace"),
		// B.2 cold-tier audit archive bucket. Empty by default — the
		// retention sweep skips the archive_to_r2 policy until the
		// operator opts in by setting STORAGE_AUDIT_COLD_BUCKET. In
		// production this is a separate R2 bucket from user uploads
		// so cost reporting splits cleanly between the two tiers.
		AuditColdStorageBucket: getEnv("STORAGE_AUDIT_COLD_BUCKET", ""),
		ResendAPIKey:        getEnv("RESEND_API_KEY", ""),
		ResendDevRedirectTo: getEnv("RESEND_DEV_REDIRECT_TO", ""),
		EmailFrom:           getEnv("EMAIL_FROM", "Marketplace Service <onboarding@resend.dev>"),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:3001"),
		LiveKitURL:       getEnv("LIVEKIT_URL", ""),
		LiveKitAPIKey:    getEnv("LIVEKIT_API_KEY", ""),
		LiveKitAPISecret:   getEnv("LIVEKIT_API_SECRET", ""),
		FCMCredentialsPath:   getEnv("FCM_CREDENTIALS_PATH", ""),
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		RekognitionEnabled:             getEnv("REKOGNITION_ENABLED", "false") == "true",
		RekognitionRegion:              getEnv("REKOGNITION_REGION", getEnv("AWS_REGION", "eu-west-1")),
		RekognitionThreshold:           parseFloat(getEnv("REKOGNITION_THRESHOLD", "60")),
		RekognitionAutoRejectThreshold: parseFloat(getEnv("REKOGNITION_AUTO_REJECT_THRESHOLD", "95")),
		RekognitionRoleARN:             getEnv("REKOGNITION_ROLE_ARN", ""),
		S3ModerationBucket:             getEnv("S3_MODERATION_BUCKET", ""),
		SNSTopicARN:                    getEnv("SNS_TOPIC_ARN", ""),
		SQSQueueURL:                    getEnv("SQS_QUEUE_URL", ""),
		AnthropicAPIKey:                getEnv("ANTHROPIC_API_KEY", ""),
		TextModerationProvider:         getEnv("TEXT_MODERATION_PROVIDER", "openai"),

		// Search engine — Typesense is mandatory since phase 4.
		TypesenseHost:         getEnv("TYPESENSE_HOST", "http://localhost:8108"),
		TypesenseAPIKey:       getEnv("TYPESENSE_API_KEY", ""),
		OpenAIAPIKey:          getEnv("OPENAI_API_KEY", ""),
		OpenAIEmbeddingsModel: getEnv("OPENAI_EMBEDDINGS_MODEL", "text-embedding-3-small"),
		TrustedProxies:        getEnv("TRUSTED_PROXIES", ""),

		// RATE-LIMIT-PROD: env-driven rate-limit caps across every
		// class. 0 (or unset) means "fall back to
		// middleware.DefaultXxxPolicy". Production typically leaves
		// these unset and relies on the safe defaults; local dev can
		// bump them in .env when iterating across multiple tabs.
		// Auth/2FA/password-reset classes are tight on purpose —
		// loosen them only with a clear threat-model justification.
		RateLimitGlobalPerMinute:        parseInt(getEnv("RATE_LIMIT_GLOBAL_PER_MINUTE", "0"), 0),
		RateLimitMutationPerMinute:      parseInt(getEnv("RATE_LIMIT_MUTATION_PER_MINUTE", "0"), 0),
		RateLimitUploadPerMinute:        parseInt(getEnv("RATE_LIMIT_UPLOAD_PER_MINUTE", "0"), 0),
		RateLimitAuthLoginPerMinute:     parseInt(getEnv("RATE_LIMIT_AUTH_LOGIN_PER_MINUTE", "0"), 0),
		RateLimitAuth2FAVerifyPerMinute: parseInt(getEnv("RATE_LIMIT_AUTH_2FA_VERIFY_PER_MINUTE", "0"), 0),
		RateLimitAuth2FAEnablePerMinute: parseInt(getEnv("RATE_LIMIT_AUTH_2FA_ENABLE_PER_MINUTE", "0"), 0),
		RateLimitPasswordResetPerMinute: parseInt(getEnv("RATE_LIMIT_PASSWORD_RESET_PER_MINUTE", "0"), 0),

		// BUG-16: parallel notification worker pool size. Zero means
		// "fall back to the package default" (currently 5).
		NotificationWorkerConcurrency: parseInt(getEnv("NOTIFICATION_WORKER_CONCURRENCY", "5"), 5),

		// P5 (GDPR): anonymization salt for the daily purge cron.
		// Defaults to the dev fallback so unit tests don't fail; in
		// production Validate() refuses to boot if the value is
		// still the fallback.
		GDPRAnonymizationSalt: getEnv("GDPR_ANONYMIZATION_SALT", devGDPRSaltFallback),

		// Phase B.1 retention scheduler. Empty value means "unset" —
		// the wire layer interprets that as the per-env default (1h
		// in prod, 1m in dev). Non-empty is parsed; bogus values fall
		// back to zero so the wire-layer default applies.
		RetentionInterval: retentionIntervalFromEnv(getEnv("RETENTION_INTERVAL", "")),

		// RGPD art. 22 admin notification recipient. Empty silences the
		// email best-effort path; the row is still persisted.
		GDPRContactEmail: getEnv("GDPR_CONTACT_EMAIL", ""),

		// PostHog analytics. Same project token shared with the web
		// SDK (NEXT_PUBLIC_POSTHOG_KEY) — the public project token
		// is designed to be embedded in clients. Empty key disables
		// analytics with a WARN log; never fatal at boot.
		PostHogProjectKey: getEnv("POSTHOG_PROJECT_KEY", ""),
		PostHogHost:       getEnv("POSTHOG_HOST", "https://eu.posthog.com"),
	}
}

// PostHogConfigured reports whether enough env is present to ship
// events to PostHog. Used by the wiring to pick the real adapter vs
// the noop fallback. Analytics never blocks the boot — a missing
// key only logs a WARN and turns the noop adapter on.
func (c *Config) PostHogConfigured() bool {
	return c.PostHogProjectKey != "" && c.PostHogHost != ""
}

const devGDPRSaltFallback = "dev-salt-not-for-prod"

// TypesenseConfigured reports whether the backend has enough
// configuration to talk to a Typesense cluster at all. Used by
// the startup wiring to decide whether to instantiate the search
// client + indexer. Since phase 4 Typesense is mandatory — the
// app boots without search ONLY in isolated test contexts that
// omit the env vars; production deployments MUST set them.
func (c *Config) TypesenseConfigured() bool {
	return c.TypesenseHost != "" && c.TypesenseAPIKey != ""
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// Validate enforces the security-critical invariants required to boot
// the API. In production it returns an error when:
//   - JWT_SECRET equals the public dev fallback `dev-secret-change-me`
//   - JWT_SECRET is shorter than 32 bytes
//   - STORAGE_SECRET_KEY or STORAGE_ACCESS_KEY equal the MinIO default
//     `minioadmin`
//
// In development the same conditions log a noisy slog.Warn but DO NOT
// fail the boot — local development typically uses these defaults via
// docker-compose, and a hard fail would break every "fresh checkout"
// flow.
//
// Callers (cmd/api/main.go) MUST treat any returned error as fatal:
//
//	if err := cfg.Validate(); err != nil {
//	    slog.Error("config validation failed", "error", err)
//	    os.Exit(1)
//	}
//
// SEC-04 (audit 2026-04-29): closes the "open-source repo with public
// fallback secrets" attack vector.
func (c *Config) Validate() error {
	var errs []string

	addError := func(msg string) {
		if c.IsProduction() {
			errs = append(errs, msg)
		} else {
			slog.Warn("config: insecure default in non-production env — DO NOT deploy this configuration",
				"detail", msg)
		}
	}

	if c.JWTSecret == devJWTFallback {
		addError(fmt.Sprintf(
			"JWT_SECRET is the public dev fallback (%q) — generate a fresh 32+ byte secret",
			devJWTFallback))
	} else if len(c.JWTSecret) < minJWTSecretBytes {
		addError(fmt.Sprintf(
			"JWT_SECRET is %d bytes; minimum is %d bytes for HS256 (NIST SP 800-131A)",
			len(c.JWTSecret), minJWTSecretBytes))
	}

	if c.StorageSecretKey == devStorageFallback {
		addError(fmt.Sprintf(
			"STORAGE_SECRET_KEY is the public MinIO default (%q) — set a strong secret in S3/R2",
			devStorageFallback))
	}
	if c.StorageAccessKey == devStorageFallback {
		addError(fmt.Sprintf(
			"STORAGE_ACCESS_KEY is the public MinIO default (%q) — set a real access key",
			devStorageFallback))
	}
	// P5 (GDPR): a salt collision across deployments would let an
	// attacker correlate anonymized audit rows back to the cleartext
	// email via a public dictionary. Refuse to boot in production
	// when the value is still the dev default.
	if c.GDPRAnonymizationSalt == devGDPRSaltFallback {
		addError(fmt.Sprintf(
			"GDPR_ANONYMIZATION_SALT is the public dev fallback (%q) — generate a fresh 32+ byte secret",
			devGDPRSaltFallback))
	}

	// Rekognition thresholds — INVARIANT: flag < auto-reject. Inverting
	// the two makes every label end up auto-rejected (the "flagged for
	// human review" bucket becomes unreachable) and the source file is
	// silently deleted. We refuse to boot in production AND in dev
	// because a misconfigured threshold has user-visible side-effects
	// (deleted media) even on a developer's local machine.
	if c.RekognitionEnabled && c.RekognitionThreshold > 0 &&
		c.RekognitionAutoRejectThreshold > 0 &&
		c.RekognitionThreshold >= c.RekognitionAutoRejectThreshold {
		errs = append(errs, fmt.Sprintf(
			"REKOGNITION_THRESHOLD (%.1f) must be strictly less than REKOGNITION_AUTO_REJECT_THRESHOLD (%.1f) — inverting them makes the 'flagged for review' bucket unreachable and auto-rejects every detected label",
			c.RekognitionThreshold, c.RekognitionAutoRejectThreshold))
	}

	if len(errs) > 0 {
		return errors.New("config validation failed: " + strings.Join(errs, "; "))
	}
	return nil
}

func (c *Config) LiveKitConfigured() bool {
	return c.LiveKitURL != "" && c.LiveKitAPIKey != "" && c.LiveKitAPISecret != ""
}

func (c *Config) FCMConfigured() bool {
	return c.FCMCredentialsPath != ""
}

func (c *Config) StripeConfigured() bool {
	return c.StripeSecretKey != "" && c.StripePublishableKey != ""
}

// TextModerationProviderOrDefault returns the provider to use, after
// falling back safely. "openai" without an API key degrades to "noop"
// so we never leak requests against an unauthenticated endpoint.
// Anything not in the allow-list also resolves to "noop".
func (c *Config) TextModerationProviderOrDefault() string {
	switch c.TextModerationProvider {
	case "openai":
		if c.OpenAIAPIKey == "" {
			return "noop"
		}
		return "openai"
	case "comprehend", "noop":
		return c.TextModerationProvider
	default:
		return "noop"
	}
}

func (c *Config) RekognitionConfigured() bool {
	return c.RekognitionEnabled && c.RekognitionRegion != ""
}

// VideoModerationConfigured reports whether all AWS prerequisites for async
// video moderation are set (Rekognition + S3 transit bucket + SNS + SQS + role).
func (c *Config) VideoModerationConfigured() bool {
	return c.RekognitionConfigured() &&
		c.S3ModerationBucket != "" &&
		c.SNSTopicARN != "" &&
		c.SQSQueueURL != "" &&
		c.RekognitionRoleARN != ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

// retentionIntervalFromEnv parses an explicit duration when set, and
// returns zero (= "use the per-env default") when the env var is
// blank. Kept distinct from parseDuration because the latter falls
// back to 15m on any parse failure — fine for JWT expiry, wrong here
// where blank means "wire decides".
func retentionIntervalFromEnv(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 60
	}
	return v
}

func parseInt(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
