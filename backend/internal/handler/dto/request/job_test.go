package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreateJobRequest_Validation(t *testing.T) {
	valid := CreateJobRequest{
		Title:           "A title",
		Description:     "A description",
		ApplicantType:   "freelance",
		BudgetType:      "fixed",
		MinBudget:       1000,
		MaxBudget:       5000,
		DescriptionType: "text",
	}
	assert.NoError(t, validator.Validate(valid))

	t.Run("title required", func(t *testing.T) {
		r := valid
		r.Title = ""
		require.Error(t, validator.Validate(r))
	})

	t.Run("description too long", func(t *testing.T) {
		r := valid
		r.Description = strings.Repeat("a", 10001)
		require.Error(t, validator.Validate(r))
	})

	t.Run("budget overflow", func(t *testing.T) {
		r := valid
		r.MaxBudget = 1_000_000_001
		require.Error(t, validator.Validate(r))
	})

	t.Run("budget negative", func(t *testing.T) {
		r := valid
		r.MinBudget = -1
		require.Error(t, validator.Validate(r))
	})

	t.Run("invalid url", func(t *testing.T) {
		r := valid
		bad := "not-a-url"
		r.VideoURL = &bad
		require.Error(t, validator.Validate(r))
	})

	t.Run("too many skills", func(t *testing.T) {
		r := valid
		skills := make([]string, 31)
		for i := range skills {
			skills[i] = "x"
		}
		r.Skills = skills
		require.Error(t, validator.Validate(r))
	})
}

func TestApplyToJobRequest_Validation(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		assert.NoError(t, validator.Validate(ApplyToJobRequest{Message: "hi"}))
	})
	t.Run("missing message", func(t *testing.T) {
		require.Error(t, validator.Validate(ApplyToJobRequest{}))
	})
	t.Run("message too long", func(t *testing.T) {
		require.Error(t, validator.Validate(ApplyToJobRequest{Message: strings.Repeat("a", 5001)}))
	})
}
