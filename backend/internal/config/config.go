package config

import (
	"os"
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
	AllowedOrigins   []string
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool
	StoragePublicURL string
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
		AllowedOrigins:   strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "marketplace"),
		StorageUseSSL:    getEnv("STORAGE_USE_SSL", "false") == "true",
		StoragePublicURL: getEnv("STORAGE_PUBLIC_URL", "http://localhost:9000/marketplace"),
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
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
