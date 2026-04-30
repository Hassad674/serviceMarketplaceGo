package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreateSkillRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(CreateSkillRequest{DisplayText: ""}))
	require.Error(t, validator.Validate(CreateSkillRequest{DisplayText: strings.Repeat("a", 200)}))
	require.NoError(t, validator.Validate(CreateSkillRequest{DisplayText: "Go"}))
}

func TestPutProfileSkillsRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(PutProfileSkillsRequest{
		SkillTexts: make([]string, 51),
	}))
	require.NoError(t, validator.Validate(PutProfileSkillsRequest{
		SkillTexts: []string{"go", "rust"},
	}))
}
