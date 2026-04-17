package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	Env              string
	DatabaseURL      string
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
	ResendAPIKey         string
	ResendDevRedirectTo  string // optional: if set, all outgoing emails are routed here (dev/staging sandbox)
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
	ComprehendEnabled               bool
	AnthropicAPIKey                 string

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
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		Env:              getEnv("ENV", "development"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/marketplace?sslmode=disable"),
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
		ResendAPIKey:        getEnv("RESEND_API_KEY", ""),
		ResendDevRedirectTo: getEnv("RESEND_DEV_REDIRECT_TO", ""),
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
		ComprehendEnabled:              getEnv("COMPREHEND_ENABLED", "false") == "true",
		AnthropicAPIKey:                getEnv("ANTHROPIC_API_KEY", ""),

		// Search engine — Typesense is mandatory since phase 4.
		TypesenseHost:         getEnv("TYPESENSE_HOST", "http://localhost:8108"),
		TypesenseAPIKey:       getEnv("TYPESENSE_API_KEY", ""),
		OpenAIAPIKey:          getEnv("OPENAI_API_KEY", ""),
		OpenAIEmbeddingsModel: getEnv("OPENAI_EMBEDDINGS_MODEL", "text-embedding-3-small"),
	}
}

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

func (c *Config) LiveKitConfigured() bool {
	return c.LiveKitURL != "" && c.LiveKitAPIKey != "" && c.LiveKitAPISecret != ""
}

func (c *Config) FCMConfigured() bool {
	return c.FCMCredentialsPath != ""
}

func (c *Config) StripeConfigured() bool {
	return c.StripeSecretKey != "" && c.StripePublishableKey != ""
}

func (c *Config) ComprehendConfigured() bool {
	return c.ComprehendEnabled
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
