package response

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

// TestReferralResponse_DisplayNames_OwnerOnly proves the apporteur
// (referrer) viewer is the only role that receives the human-readable
// provider + client labels in the DTO. Other viewers see the masked
// snapshot — names exchanged through the activated conversation, not
// the DTO.
func TestReferralResponse_DisplayNames_OwnerOnly(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()

	r := &referral.Referral{
		ID:               uuid.New(),
		ReferrerID:       referrerID,
		ProviderID:       providerID,
		ClientID:         clientID,
		RatePct:          10,
		DurationMonths:   6,
		Status:           referral.StatusActive,
		Version:          1,
		IntroMessageProvider: "p",
		IntroMessageClient:   "c",
		LastActionAt:     time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	names := ReferralDisplayNames{
		Provider: "Atelier Lumen",
		Client:   "Banque du Sud",
	}

	cases := []struct {
		name        string
		viewerID    uuid.UUID
		expectNames bool
	}{
		{"referrer viewer sees names", referrerID, true},
		{"provider viewer is masked", providerID, false},
		{"client viewer is masked", clientID, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := NewReferralResponseWithNames(r, tc.viewerID, names)
			raw, err := json.Marshal(out)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(raw, &decoded))
			if tc.expectNames {
				assert.Equal(t, "Atelier Lumen", decoded["provider_display_name"])
				assert.Equal(t, "Banque du Sud", decoded["client_display_name"])
			} else {
				_, hasProvider := decoded["provider_display_name"]
				assert.False(t, hasProvider,
					"non-referrer viewer must NOT see provider_display_name")
				_, hasClient := decoded["client_display_name"]
				assert.False(t, hasClient,
					"non-referrer viewer must NOT see client_display_name")
			}
		})
	}
}

// TestReferralResponse_DisplayNames_EmptyNames covers the resolver-not-
// wired branch: the apporteur asks but the resolver returns empty
// strings (lookup failed, user not found, etc.). The JSON must still
// omit the fields rather than render empty strings — keeps the wire
// shape clean.
func TestReferralResponse_DisplayNames_EmptyNames(t *testing.T) {
	referrerID := uuid.New()
	r := &referral.Referral{
		ID:           uuid.New(),
		ReferrerID:   referrerID,
		ProviderID:   uuid.New(),
		ClientID:     uuid.New(),
		Status:       referral.StatusPendingProvider,
		LastActionAt: time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	out := NewReferralResponseWithNames(r, referrerID, ReferralDisplayNames{})
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	_, hasProvider := decoded["provider_display_name"]
	assert.False(t, hasProvider,
		"empty provider name must be omitted via omitempty")
	_, hasClient := decoded["client_display_name"]
	assert.False(t, hasClient,
		"empty client name must be omitted via omitempty")
}

// TestAttributionResponse_TotalAmountCents proves the gross proposal
// amount is surfaced on every attribution row. Visible to every viewer
// — it is the public mission price, not a commission number, so
// Modèle A confidentiality does not require redaction.
func TestAttributionResponse_TotalAmountCents(t *testing.T) {
	clientID := uuid.New()
	row := attributionWithStats{
		Attribution: &referral.Attribution{
			ID:              uuid.New(),
			ReferralID:      uuid.New(),
			ProposalID:      uuid.New(),
			ProviderID:      uuid.New(),
			ClientID:        clientID,
			RatePctSnapshot: 5,
			AttributedAt:    time.Now().UTC(),
		},
		ProposalTitle:       "Refonte LP",
		ProposalStatus:      "active",
		ProposalAmountCents: 123_000,
		TotalCommissionCents: 1_000,
		MilestonesPaid:      1,
		MilestonesTotal:     3,
	}

	cases := []struct {
		name     string
		viewerID uuid.UUID
	}{
		{"referrer viewer", uuid.New()},
		{"client viewer (Modèle A — still sees public price)", clientID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := NewAttributionListFromStats(
				[]attributionWithStats{row}, tc.viewerID, clientID,
			)
			require.Len(t, out, 1)
			raw, err := json.Marshal(out[0])
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(raw, &decoded))
			gotAmount, ok := decoded["total_amount_cents"].(float64)
			require.True(t, ok, "total_amount_cents must be present: %s", string(raw))
			assert.Equal(t, float64(123_000), gotAmount)
		})
	}
}

// TestAttributionResponse_TotalAmountCents_Zero ensures the field is
// rendered as 0 (not omitted) when the proposal lookup failed — the
// UI degrades to "0 €" rather than crashing on a missing field.
func TestAttributionResponse_TotalAmountCents_Zero(t *testing.T) {
	clientID := uuid.New()
	row := attributionWithStats{
		Attribution: &referral.Attribution{
			ID:              uuid.New(),
			ReferralID:      uuid.New(),
			ProposalID:      uuid.New(),
			ProviderID:      uuid.New(),
			ClientID:        clientID,
			RatePctSnapshot: 5,
			AttributedAt:    time.Now().UTC(),
		},
		ProposalAmountCents: 0,
	}
	out := NewAttributionListFromStats(
		[]attributionWithStats{row}, uuid.New(), clientID,
	)
	require.Len(t, out, 1)
	raw, err := json.Marshal(out[0])
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	gotAmount, ok := decoded["total_amount_cents"].(float64)
	require.True(t, ok, "total_amount_cents must be present even when zero")
	assert.Equal(t, float64(0), gotAmount)
}
