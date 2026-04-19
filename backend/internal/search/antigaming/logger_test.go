package antigaming

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// NoopLogger never records anything.
func TestNoopLogger(t *testing.T) {
	l := NoopLogger{}
	l.LogPenalty(Penalty{Rule: RuleKeywordStuffing}) // must not panic
}

// RecordingLogger accumulates entries in order.
func TestRecordingLogger(t *testing.T) {
	l := &RecordingLogger{}
	l.LogPenalty(Penalty{Rule: RuleKeywordStuffing})
	l.LogPenalty(Penalty{Rule: RuleReviewVelocity})
	assert.Len(t, l.Penalties, 2)
	assert.Equal(t, RuleKeywordStuffing, l.Penalties[0].Rule)
	assert.Equal(t, RuleReviewVelocity, l.Penalties[1].Rule)
}

// SlogLogger emits the exact structured JSON from §7.6. Shape locked by a
// parse-then-inspect test so a field rename fails the build.
func TestSlogLogger_EmitsStructuredLine(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	SlogLogger{}.LogPenalty(Penalty{
		Rule:           RuleKeywordStuffing,
		ProfileID:      "org-123",
		Persona:        features.PersonaFreelance,
		DetectionValue: 0.18,
		Threshold:      0.30,
		PenaltyFactor:  0.5,
	})

	var out map[string]any
	err := json.Unmarshal(buf.Bytes(), &out)
	assert.NoError(t, err)
	assert.Equal(t, "ranking.penalty_applied", out["msg"])
	assert.Equal(t, "keyword_stuffing", out["rule"])
	assert.Equal(t, "org-123", out["profile_id"])
	assert.Equal(t, "freelance", out["persona"])
	assert.InDelta(t, 0.18, out["detection_value"].(float64), 1e-9)
	assert.InDelta(t, 0.30, out["threshold"].(float64), 1e-9)
	assert.InDelta(t, 0.5, out["penalty_factor"].(float64), 1e-9)
}

// Rule constants don't collide, exported as strings for log tagging.
func TestRuleConstants_Unique(t *testing.T) {
	rules := []Rule{
		RuleKeywordStuffing,
		RuleReviewVelocity,
		RuleLinkedAccounts,
		RuleReviewerFloor,
		RuleNewAccount,
	}
	seen := make(map[Rule]struct{})
	for _, r := range rules {
		assert.NotEmpty(t, string(r))
		_, dup := seen[r]
		assert.False(t, dup, "rule %s duplicated", r)
		seen[r] = struct{}{}
	}
}
