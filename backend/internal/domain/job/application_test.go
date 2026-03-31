package job

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func validApplicationInput() NewApplicationInput {
	return NewApplicationInput{
		JobID:       uuid.New(),
		ApplicantID: uuid.New(),
		Message:     "I am very interested in this position and have 5 years of experience.",
	}
}

func TestNewJobApplication_Valid(t *testing.T) {
	app, err := NewJobApplication(validApplicationInput())
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.NotEqual(t, uuid.Nil, app.ID)
	assert.NotEmpty(t, app.Message)
	assert.Nil(t, app.VideoURL)
}

func TestNewJobApplication_ValidNoMessage(t *testing.T) {
	input := validApplicationInput()
	input.Message = ""
	app, err := NewJobApplication(input)
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.Empty(t, app.Message)
}

func TestNewJobApplication_ValidWithVideo(t *testing.T) {
	input := validApplicationInput()
	videoURL := "https://r2.example.com/videos/intro.mp4"
	input.VideoURL = &videoURL

	app, err := NewJobApplication(input)
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, &videoURL, app.VideoURL)
}

func TestNewJobApplication_Validation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*NewApplicationInput)
		wantErr error
	}{
		{
			name: "message too long",
			modify: func(i *NewApplicationInput) {
				i.Message = strings.Repeat("a", applicationMessageMaxLength+1)
			},
			wantErr: ErrApplicationMessageTooLong,
		},
		{
			name:    "nil job ID",
			modify:  func(i *NewApplicationInput) { i.JobID = uuid.Nil },
			wantErr: ErrCannotApplyToClosed,
		},
		{
			name:    "nil applicant ID",
			modify:  func(i *NewApplicationInput) { i.ApplicantID = uuid.Nil },
			wantErr: ErrNotApplicant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validApplicationInput()
			tt.modify(&input)
			app, err := NewJobApplication(input)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, app)
		})
	}
}
