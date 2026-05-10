package retention_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/retention"
)

func TestStrategy_IsValid(t *testing.T) {
	tests := []struct {
		name string
		s    retention.Strategy
		want bool
	}{
		{"delete is valid", retention.StrategyDelete, true},
		{"archive is valid", retention.StrategyArchive, true},
		{"anonymize is valid", retention.StrategyAnonymize, true},
		{"empty rejected", retention.Strategy(""), false},
		{"unknown rejected", retention.Strategy("purge"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.IsValid())
		})
	}
}

func TestPolicy_Validate(t *testing.T) {
	base := retention.Policy{
		Name:      "messages_3y",
		Table:     "messages",
		AgeColumn: "created_at",
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyDelete,
	}

	tests := []struct {
		name    string
		mutate  func(p *retention.Policy)
		wantErr error
	}{
		{
			name:   "valid delete policy",
			mutate: func(p *retention.Policy) {},
		},
		{
			name:    "missing name",
			mutate:  func(p *retention.Policy) { p.Name = "" },
			wantErr: retention.ErrPolicyNameRequired,
		},
		{
			name:    "missing table",
			mutate:  func(p *retention.Policy) { p.Table = "" },
			wantErr: retention.ErrPolicyTableRequired,
		},
		{
			name:    "missing age column",
			mutate:  func(p *retention.Policy) { p.AgeColumn = "" },
			wantErr: retention.ErrPolicyAgeColumnRequired,
		},
		{
			name:    "max age zero",
			mutate:  func(p *retention.Policy) { p.MaxAge = 0 },
			wantErr: retention.ErrPolicyMaxAgeInvalid,
		},
		{
			name:    "max age negative",
			mutate:  func(p *retention.Policy) { p.MaxAge = -time.Second },
			wantErr: retention.ErrPolicyMaxAgeInvalid,
		},
		{
			name:    "unknown strategy",
			mutate:  func(p *retention.Policy) { p.Strategy = retention.Strategy("foo") },
			wantErr: retention.ErrPolicyStrategyInvalid,
		},
		{
			name: "archive missing target",
			mutate: func(p *retention.Policy) {
				p.Strategy = retention.StrategyArchive
				p.ArchiveTable = ""
			},
			wantErr: retention.ErrPolicyArchiveMissing,
		},
		{
			name: "archive ok",
			mutate: func(p *retention.Policy) {
				p.Strategy = retention.StrategyArchive
				p.ArchiveTable = "messages_archive"
			},
		},
		{
			name: "anonymize missing columns",
			mutate: func(p *retention.Policy) {
				p.Strategy = retention.StrategyAnonymize
				p.AnonymizeColumns = nil
			},
			wantErr: retention.ErrPolicyAnonymizeMissing,
		},
		{
			name: "anonymize ok",
			mutate: func(p *retention.Policy) {
				p.Strategy = retention.StrategyAnonymize
				p.AnonymizeColumns = []string{"user_id"}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := base
			tt.mutate(&p)
			err := p.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantErr), "got %v, want errors.Is %v", err, tt.wantErr)
		})
	}
}

func TestPolicy_EffectiveBatchSize(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{"zero defaults", 0, retention.DefaultBatchSize},
		{"explicit value wins", 250, 250},
		{"negative defaults", -10, retention.DefaultBatchSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := retention.Policy{BatchSize: tt.size}
			assert.Equal(t, tt.want, p.EffectiveBatchSize())
		})
	}
}

func TestPolicy_Cutoff(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	p := retention.Policy{MaxAge: 24 * time.Hour}
	assert.Equal(t, now.Add(-24*time.Hour), p.Cutoff(now))
}

func TestDefaultPolicies_AllValid(t *testing.T) {
	for _, p := range retention.DefaultPolicies(retention.Overrides{}) {
		t.Run(p.Name, func(t *testing.T) {
			require.NoError(t, p.Validate(), "default policy %q should validate", p.Name)
		})
	}
}

func TestDefaultPolicies_OverridesApply(t *testing.T) {
	o := retention.Overrides{
		MessagesMaxAge:      48 * time.Hour,
		NotificationsMaxAge: 7 * 24 * time.Hour,
	}
	policies := retention.DefaultPolicies(o)
	for _, p := range policies {
		switch p.Name {
		case "messages_3y":
			assert.Equal(t, 48*time.Hour, p.MaxAge)
		case "notifications_90d":
			assert.Equal(t, 7*24*time.Hour, p.MaxAge)
		case "device_tokens_60d_inactive":
			assert.Equal(t, retention.DefaultDeviceTokensMaxAge, p.MaxAge)
		}
	}
}

func TestDefaultPolicies_StrategyMapping(t *testing.T) {
	policies := retention.DefaultPolicies(retention.Overrides{})
	wantStrategies := map[string]retention.Strategy{
		"messages_3y":                   retention.StrategyDelete,
		"notifications_90d":             retention.StrategyDelete,
		"device_tokens_60d_inactive":    retention.StrategyDelete,
		"search_queries_12mo_anonymize": retention.StrategyAnonymize,
		"audit_logs_24mo_archive":       retention.StrategyArchive,
	}
	got := map[string]retention.Strategy{}
	for _, p := range policies {
		got[p.Name] = p.Strategy
	}
	assert.Equal(t, wantStrategies, got)
}
