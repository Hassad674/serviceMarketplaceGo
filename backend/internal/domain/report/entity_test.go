package report

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewReport_ValidMessage(t *testing.T) {
	r, err := NewReport(NewReportInput{
		ReporterID:     uuid.New(),
		TargetType:     TargetMessage,
		TargetID:       uuid.New(),
		ConversationID: uuid.New(),
		Reason:         ReasonSpam,
		Description:    "This message is spam",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, TargetMessage, r.TargetType)
	assert.Equal(t, ReasonSpam, r.Reason)
	assert.Equal(t, StatusPending, r.Status)
	assert.Equal(t, "This message is spam", r.Description)
}

func TestNewReport_ValidUser(t *testing.T) {
	r, err := NewReport(NewReportInput{
		ReporterID: uuid.New(),
		TargetType: TargetUser,
		TargetID:   uuid.New(),
		Reason:     ReasonFakeProfile,
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, TargetUser, r.TargetType)
	assert.Equal(t, ReasonFakeProfile, r.Reason)
}

func TestNewReport_Validation(t *testing.T) {
	validReporterID := uuid.New()
	validTargetID := uuid.New()

	tests := []struct {
		name    string
		input   NewReportInput
		wantErr error
	}{
		{
			name: "self report",
			input: NewReportInput{
				ReporterID: validReporterID,
				TargetType: TargetUser,
				TargetID:   validReporterID,
				Reason:     ReasonSpam,
			},
			wantErr: ErrSelfReport,
		},
		{
			name: "invalid target type",
			input: NewReportInput{
				ReporterID: validReporterID,
				TargetType: TargetType("invalid"),
				TargetID:   validTargetID,
				Reason:     ReasonSpam,
			},
			wantErr: ErrInvalidTargetType,
		},
		{
			name: "inappropriate content not allowed for user",
			input: NewReportInput{
				ReporterID: validReporterID,
				TargetType: TargetUser,
				TargetID:   validTargetID,
				Reason:     ReasonInappropriateContent,
			},
			wantErr: ErrReasonNotAllowedForType,
		},
		{
			name: "fake profile not allowed for message",
			input: NewReportInput{
				ReporterID: validReporterID,
				TargetType: TargetMessage,
				TargetID:   validTargetID,
				Reason:     ReasonFakeProfile,
			},
			wantErr: ErrReasonNotAllowedForType,
		},
		{
			name: "description too long",
			input: NewReportInput{
				ReporterID:  validReporterID,
				TargetType:  TargetMessage,
				TargetID:    validTargetID,
				Reason:      ReasonSpam,
				Description: strings.Repeat("a", MaxDescriptionLength+1),
			},
			wantErr: ErrDescriptionTooLong,
		},
		{
			name: "missing reporter",
			input: NewReportInput{
				TargetType: TargetMessage,
				TargetID:   validTargetID,
				Reason:     ReasonSpam,
			},
			wantErr: ErrMissingReporter,
		},
		{
			name: "missing target",
			input: NewReportInput{
				ReporterID: validReporterID,
				TargetType: TargetMessage,
				Reason:     ReasonSpam,
			},
			wantErr: ErrMissingTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReport(tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, r)
		})
	}
}
