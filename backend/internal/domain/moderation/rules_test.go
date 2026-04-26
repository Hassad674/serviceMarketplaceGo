package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/port/service"
)

func TestDecideStatus(t *testing.T) {
	tests := []struct {
		name       string
		result     *service.TextModerationResult
		wantStatus moderation.Status
		wantReason string
	}{
		{
			name:       "nil result is treated as clean",
			result:     nil,
			wantStatus: moderation.StatusClean,
			wantReason: moderation.ReasonNone,
		},
		{
			name:       "empty labels and zero score is clean",
			result:     &service.TextModerationResult{Labels: nil, MaxScore: 0},
			wantStatus: moderation.StatusClean,
			wantReason: moderation.ReasonNone,
		},
		{
			name: "benign score under flag threshold is clean",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHarassment, Score: 0.20}},
				MaxScore: 0.20,
			},
			wantStatus: moderation.StatusClean,
		},
		{
			name: "simple insult crosses flag threshold",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHarassment, Score: 0.65}},
				MaxScore: 0.65,
			},
			wantStatus: moderation.StatusFlagged,
			wantReason: moderation.ReasonAutoFlagScore,
		},
		{
			name: "exact flag threshold still flags",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHarassment, Score: 0.50}},
				MaxScore: 0.50,
			},
			wantStatus: moderation.StatusFlagged,
			wantReason: moderation.ReasonAutoFlagScore,
		},
		{
			name: "very toxic non-threatening content is hidden",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHarassment, Score: 0.92}},
				MaxScore: 0.92,
			},
			wantStatus: moderation.StatusHidden,
			wantReason: moderation.ReasonAutoHideHighScore,
		},
		{
			name: "exact hide threshold still hides",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHate, Score: 0.90}},
				MaxScore: 0.90,
			},
			wantStatus: moderation.StatusHidden,
			wantReason: moderation.ReasonAutoHideHighScore,
		},
		{
			name: "extreme harassment without threat triggers delete via global threshold",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHarassment, Score: 0.97}},
				MaxScore: 0.97,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteExtremeScore,
		},
		{
			name: "exact extreme threshold still deletes",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHate, Score: 0.95}},
				MaxScore: 0.95,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteExtremeScore,
		},
		{
			name: "just below extreme threshold stays hidden",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHarassment, Score: 0.949}},
				MaxScore: 0.949,
			},
			wantStatus: moderation.StatusHidden,
			wantReason: moderation.ReasonAutoHideHighScore,
		},
		{
			name: "explicit threat triggers delete",
			result: &service.TextModerationResult{
				Labels: []service.TextModerationLabel{
					{Name: moderation.CategoryHarassment, Score: 0.70},
					{Name: moderation.CategoryHarassmentThreaten, Score: 0.82},
				},
				MaxScore: 0.82,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteThreat,
		},
		{
			name: "hate threatening triggers delete",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryHateThreatening, Score: 0.81}},
				MaxScore: 0.81,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteHateThreat,
		},
		{
			name: "sexual minors delete at low score",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategorySexualMinors, Score: 0.32}},
				MaxScore: 0.32,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteMinors,
		},
		{
			name: "sexual minors just below threshold does not delete",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategorySexualMinors, Score: 0.29}},
				MaxScore: 0.29,
			},
			wantStatus: moderation.StatusClean,
		},
		{
			name: "self-harm instructions delete at threshold",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategorySelfHarmInstructions, Score: 0.70}},
				MaxScore: 0.70,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteSelfHarmInstr,
		},
		{
			name: "graphic violence delete",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategoryViolenceGraphic, Score: 0.90}},
				MaxScore: 0.90,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteViolenceGfx,
		},
		{
			name: "delete priority beats global hide score",
			result: &service.TextModerationResult{
				Labels: []service.TextModerationLabel{
					{Name: moderation.CategoryHarassment, Score: 0.95},
					{Name: moderation.CategoryHarassmentThreaten, Score: 0.85},
				},
				MaxScore: 0.95,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteThreat,
		},
		{
			name: "non-zero-tolerance category at high but sub-extreme score is hidden",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategorySexual, Score: 0.93}},
				MaxScore: 0.93,
			},
			wantStatus: moderation.StatusHidden,
			wantReason: moderation.ReasonAutoHideHighScore,
		},
		{
			name: "non-zero-tolerance category above extreme threshold is deleted",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: moderation.CategorySexual, Score: 0.99}},
				MaxScore: 0.99,
			},
			wantStatus: moderation.StatusDeleted,
			wantReason: moderation.ReasonAutoDeleteExtremeScore,
		},
		{
			name: "unknown category label is ignored by zero-tolerance check",
			result: &service.TextModerationResult{
				Labels:   []service.TextModerationLabel{{Name: "unknown/made-up", Score: 0.99}},
				MaxScore: 0.40,
			},
			wantStatus: moderation.StatusClean,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, reason := moderation.DecideStatus(tt.result)
			assert.Equal(t, tt.wantStatus, status, "status mismatch")
			if tt.wantReason != "" {
				assert.Equal(t, tt.wantReason, reason, "reason mismatch")
			}
		})
	}
}

func TestDecideStatus_FirstZeroToleranceMatchWins(t *testing.T) {
	// If a text trips multiple zero-tolerance categories, the first
	// match in the label order decides. This test documents that
	// behavior so it does not change silently.
	result := &service.TextModerationResult{
		Labels: []service.TextModerationLabel{
			{Name: moderation.CategoryHateThreatening, Score: 0.90},
			{Name: moderation.CategorySexualMinors, Score: 0.90},
		},
		MaxScore: 0.90,
	}
	status, reason := moderation.DecideStatus(result)
	assert.Equal(t, moderation.StatusDeleted, status)
	assert.Equal(t, moderation.ReasonAutoDeleteHateThreat, reason)
}
