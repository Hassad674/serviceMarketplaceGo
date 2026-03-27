package job

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func validInput() NewJobInput {
	return NewJobInput{
		CreatorID:     uuid.New(),
		Title:         "Senior Go Developer",
		Description:   "Looking for an experienced Go developer.",
		Skills:        []string{"Go", "PostgreSQL"},
		ApplicantType: ApplicantAll,
		BudgetType:    BudgetOneShot,
		MinBudget:     5000,
		MaxBudget:     10000,
	}
}

func TestNewJob_Valid(t *testing.T) {
	j, err := NewJob(validInput())
	assert.NoError(t, err)
	assert.NotNil(t, j)
	assert.Equal(t, StatusOpen, j.Status)
	assert.Equal(t, "Senior Go Developer", j.Title)
	assert.Len(t, j.Skills, 2)
}

func TestNewJob_Validation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*NewJobInput)
		wantErr error
	}{
		{
			name:    "empty title",
			modify:  func(i *NewJobInput) { i.Title = "" },
			wantErr: ErrEmptyTitle,
		},
		{
			name:    "title too long",
			modify:  func(i *NewJobInput) { i.Title = string(make([]byte, 101)) },
			wantErr: ErrTitleTooLong,
		},
		{
			name:    "empty description",
			modify:  func(i *NewJobInput) { i.Description = "" },
			wantErr: ErrEmptyDescription,
		},
		{
			name:    "too many skills",
			modify:  func(i *NewJobInput) { i.Skills = []string{"a", "b", "c", "d", "e", "f"} },
			wantErr: ErrTooManySkills,
		},
		{
			name:    "invalid applicant type",
			modify:  func(i *NewJobInput) { i.ApplicantType = "invalid" },
			wantErr: ErrInvalidApplicantType,
		},
		{
			name:    "invalid budget type",
			modify:  func(i *NewJobInput) { i.BudgetType = "invalid" },
			wantErr: ErrInvalidBudgetType,
		},
		{
			name:    "zero min budget",
			modify:  func(i *NewJobInput) { i.MinBudget = 0 },
			wantErr: ErrInvalidBudget,
		},
		{
			name:    "negative max budget",
			modify:  func(i *NewJobInput) { i.MaxBudget = -1 },
			wantErr: ErrInvalidBudget,
		},
		{
			name: "min exceeds max",
			modify: func(i *NewJobInput) {
				i.MinBudget = 20000
				i.MaxBudget = 10000
			},
			wantErr: ErrMinExceedsMax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput()
			tt.modify(&input)
			j, err := NewJob(input)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, j)
		})
	}
}

func TestNewJob_NilSkills(t *testing.T) {
	input := validInput()
	input.Skills = nil
	j, err := NewJob(input)
	assert.NoError(t, err)
	assert.NotNil(t, j.Skills)
	assert.Len(t, j.Skills, 0)
}

func TestJob_Close_Success(t *testing.T) {
	j, _ := NewJob(validInput())
	err := j.Close(j.CreatorID)
	assert.NoError(t, err)
	assert.Equal(t, StatusClosed, j.Status)
	assert.NotNil(t, j.ClosedAt)
}

func TestJob_Close_NotOwner(t *testing.T) {
	j, _ := NewJob(validInput())
	err := j.Close(uuid.New())
	assert.ErrorIs(t, err, ErrNotOwner)
	assert.Equal(t, StatusOpen, j.Status)
}

func TestJob_Close_AlreadyClosed(t *testing.T) {
	j, _ := NewJob(validInput())
	_ = j.Close(j.CreatorID)
	err := j.Close(j.CreatorID)
	assert.ErrorIs(t, err, ErrAlreadyClosed)
}
