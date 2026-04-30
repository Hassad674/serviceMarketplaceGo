package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreatePortfolioItemRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(CreatePortfolioItemRequest{Title: ""}))
	require.Error(t, validator.Validate(CreatePortfolioItemRequest{Title: "ok", LinkURL: "not a url"}))
	require.NoError(t, validator.Validate(CreatePortfolioItemRequest{Title: "ok"}))
}

func TestPortfolioMediaInput_Validation(t *testing.T) {
	require.Error(t, validator.Validate(CreatePortfolioItemRequest{
		Title: "ok",
		Media: []PortfolioMediaInput{{MediaURL: "not-a-url", MediaType: "image"}},
	}))
}

func TestReorderPortfolioRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(ReorderPortfolioRequest{ItemIDs: []string{"not-uuid"}}))
	require.NoError(t, validator.Validate(ReorderPortfolioRequest{ItemIDs: []string{uuid.NewString()}}))
}
