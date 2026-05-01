package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidate_ProductionFailsOnDefaultJWTSecret asserts the production
// fail-fast behaviour from SEC-04: a deployment that boots with the
// hardcoded fallback JWT secret must refuse to start. The repo is
// open-source so the fallback is a public string — accepting it in
// prod is equivalent to publishing the signing key.
func TestValidate_ProductionFailsOnDefaultJWTSecret(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.JWTSecret = "dev-secret-change-me"

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

// TestValidate_ProductionFailsOnShortJWTSecret guards against the
// "I generated a secret but it's only 8 chars" mistake. NIST SP 800-131A
// requires HMAC keys >= 112 bits (14 bytes); we enforce 32 bytes to
// match the signing strength of HS256.
func TestValidate_ProductionFailsOnShortJWTSecret(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.JWTSecret = "short" // 5 chars

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
	assert.Contains(t, err.Error(), "32")
}

// TestValidate_ProductionFailsOnExactly31Chars is the boundary check
// for the >= 32 rule (off-by-one regression guard).
func TestValidate_ProductionFailsOnExactly31Chars(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.JWTSecret = strings.Repeat("a", 31)

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

// TestValidate_ProductionAcceptsExactly32Chars confirms 32 chars is
// the inclusive lower bound.
func TestValidate_ProductionAcceptsExactly32Chars(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.JWTSecret = strings.Repeat("a", 32)

	err := cfg.Validate()
	assert.NoError(t, err)
}

// TestValidate_ProductionFailsOnDefaultStorageSecretKey covers the
// other half of SEC-04: MinIO's default credentials shipped in this
// repo's docker-compose. Same fallback-published-secret risk as JWT.
func TestValidate_ProductionFailsOnDefaultStorageSecretKey(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.StorageSecretKey = "minioadmin"

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_SECRET_KEY")
}

// TestValidate_ProductionFailsOnDefaultGDPRSalt is the P5 equivalent
// of the JWT/Storage default-secret check: a salt collision across
// open-source deployments would let an attacker correlate anonymized
// audit rows back to the cleartext email via a public dictionary.
func TestValidate_ProductionFailsOnDefaultGDPRSalt(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.GDPRAnonymizationSalt = "dev-salt-not-for-prod"

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GDPR_ANONYMIZATION_SALT")
}

// TestValidate_ProductionFailsOnDefaultStorageAccessKey ensures the
// access-key half of the default MinIO pair is also caught. Either
// of the two being the public default is enough to compromise the
// bucket, so we validate both.
func TestValidate_ProductionFailsOnDefaultStorageAccessKey(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.StorageAccessKey = "minioadmin"

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_ACCESS_KEY")
}

// TestValidate_ProductionPassesWithStrongSecrets is the happy path.
func TestValidate_ProductionPassesWithStrongSecrets(t *testing.T) {
	cfg := defaultProductionConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

// TestValidate_DevelopmentWarnsButDoesNotFail documents the dev-mode
// behaviour: developers regularly boot with the hardcoded defaults,
// so we log a loud slog.Warn but allow the boot to proceed. The test
// captures the slog output and asserts it mentions JWT_SECRET so the
// developer can't claim they didn't see it.
func TestValidate_DevelopmentWarnsButDoesNotFail(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*Config)
		want string
	}{
		{
			name: "default JWT secret",
			mut:  func(c *Config) { c.JWTSecret = "dev-secret-change-me" },
			want: "JWT_SECRET",
		},
		{
			name: "short JWT secret",
			mut:  func(c *Config) { c.JWTSecret = "short" },
			want: "JWT_SECRET",
		},
		{
			name: "default storage secret",
			mut:  func(c *Config) { c.StorageSecretKey = "minioadmin" },
			want: "STORAGE_SECRET_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture slog output by swapping the default logger.
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
			defer slog.SetDefault(prev)

			cfg := defaultDevelopmentConfig()
			tt.mut(cfg)
			err := cfg.Validate()

			assert.NoError(t, err, "development must NOT fail on default secrets")
			assert.Contains(t, buf.String(), tt.want,
				"slog warning must mention the offending env var")
		})
	}
}

// TestValidate_DevelopmentWithStrongSecretsLogsNothing — make sure a
// developer who already set strong secrets gets no warnings.
func TestValidate_DevelopmentWithStrongSecretsLogsNothing(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	cfg := defaultDevelopmentConfig()
	cfg.JWTSecret = strings.Repeat("a", 64)
	cfg.StorageSecretKey = "a-strong-secret"
	cfg.StorageAccessKey = "a-strong-access-key"

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.NotContains(t, buf.String(), "JWT_SECRET",
		"no warning should fire when secrets are strong")
	assert.NotContains(t, buf.String(), "STORAGE_SECRET_KEY")
}

// --- helpers ---

// defaultProductionConfig returns a Config that would PASS validation
// in production. Tests mutate one field at a time to assert the rule
// in isolation.
func defaultProductionConfig() *Config {
	return &Config{
		Env:                   "production",
		JWTSecret:             strings.Repeat("a", 64),
		StorageSecretKey:      "a-strong-secret",
		StorageAccessKey:      "a-strong-access-key",
		GDPRAnonymizationSalt: "a-strong-gdpr-salt-32-bytes-or-more-12345",
	}
}

func defaultDevelopmentConfig() *Config {
	return &Config{
		Env:              "development",
		JWTSecret:        "dev-secret-change-me",
		StorageSecretKey: "minioadmin",
		StorageAccessKey: "minioadmin",
	}
}
