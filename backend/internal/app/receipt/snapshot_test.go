package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingdomain "marketplace-backend/internal/domain/invoicing"
	referraldomain "marketplace-backend/internal/domain/referral"
	portservice "marketplace-backend/internal/port/service"
)

type fakeBillingProfiles struct {
	byOrg map[uuid.UUID]*invoicingdomain.BillingProfile
	err   error
}

func (f *fakeBillingProfiles) FindByOrganization(ctx context.Context, orgID uuid.UUID) (*invoicingdomain.BillingProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	if p, ok := f.byOrg[orgID]; ok {
		return p, nil
	}
	return nil, invoicingdomain.ErrNotFound
}

type fakeReferrals struct {
	attribution *referraldomain.Attribution
	parent      *referraldomain.Referral
	err         error
}

func (f *fakeReferrals) FindAttributionByProposal(ctx context.Context, proposalID uuid.UUID) (*referraldomain.Attribution, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.attribution, nil
}

func (f *fakeReferrals) GetByID(ctx context.Context, id uuid.UUID) (*referraldomain.Referral, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.parent, nil
}

func makeProfile(orgID uuid.UUID, name, siret string) *invoicingdomain.BillingProfile {
	return &invoicingdomain.BillingProfile{
		OrganizationID: orgID,
		LegalName:      name,
		TaxID:          siret,
		AddressLine1:   "1 rue de Paris",
		PostalCode:     "75001",
		City:           "Paris",
		Country:        "FR",
	}
}

func TestSnapshotResolver_ResolveForPayment_NoReferrer(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	clientUser := uuid.New()
	providerUser := uuid.New()

	users := UserOrgFunc(func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
		switch userID {
		case clientUser:
			return clientOrg, nil
		case providerUser:
			return providerOrg, nil
		}
		return uuid.Nil, nil
	})
	billing := &fakeBillingProfiles{
		byOrg: map[uuid.UUID]*invoicingdomain.BillingProfile{
			clientOrg:   makeProfile(clientOrg, "Client SAS", "12345678900012"),
			providerOrg: makeProfile(providerOrg, "Provider SARL", "12345678900099"),
		},
	}

	r := NewSnapshotResolver(SnapshotResolverDeps{
		Users:     users,
		Billing:   billing,
		Referrals: &fakeReferrals{},
	})

	out, err := r.ResolveForPayment(context.Background(), portservice.ReceiptSnapshotInput{
		ClientUserID:   clientUser,
		ProviderUserID: providerUser,
		ProposalID:     uuid.New(),
	})
	require.NoError(t, err)
	assert.Equal(t, clientOrg, out.Client.OrganizationID)
	assert.Equal(t, "Client SAS", out.Client.Name)
	assert.Equal(t, "12345678900012", out.Client.SIRET)
	assert.Equal(t, providerOrg, out.Provider.OrganizationID)
	assert.Nil(t, out.Referrer)
}

func TestSnapshotResolver_ResolveForPayment_WithReferrer(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	referrerOrg := uuid.New()
	clientUser := uuid.New()
	providerUser := uuid.New()
	referrerUser := uuid.New()
	proposalID := uuid.New()

	users := UserOrgFunc(func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
		switch userID {
		case clientUser:
			return clientOrg, nil
		case providerUser:
			return providerOrg, nil
		case referrerUser:
			return referrerOrg, nil
		}
		return uuid.Nil, nil
	})
	billing := &fakeBillingProfiles{
		byOrg: map[uuid.UUID]*invoicingdomain.BillingProfile{
			clientOrg:   makeProfile(clientOrg, "Client SAS", "11111111111111"),
			providerOrg: makeProfile(providerOrg, "Provider SARL", "22222222222222"),
			referrerOrg: makeProfile(referrerOrg, "Apporteur SAS", "33333333333333"),
		},
	}
	referralID := uuid.New()
	attr := &referraldomain.Attribution{
		ID:         uuid.New(),
		ReferralID: referralID,
		ProposalID: proposalID,
		ProviderID: providerUser,
		ClientID:   clientUser,
	}
	parent := &referraldomain.Referral{
		ID:         referralID,
		ReferrerID: referrerUser,
		ProviderID: providerUser,
		ClientID:   clientUser,
	}
	r := NewSnapshotResolver(SnapshotResolverDeps{
		Users:     users,
		Billing:   billing,
		Referrals: &fakeReferrals{attribution: attr, parent: parent},
	})

	out, err := r.ResolveForPayment(context.Background(), portservice.ReceiptSnapshotInput{
		ClientUserID:   clientUser,
		ProviderUserID: providerUser,
		ProposalID:     proposalID,
	})
	require.NoError(t, err)
	require.NotNil(t, out.Referrer)
	assert.Equal(t, referrerOrg, out.Referrer.OrganizationID)
	assert.Equal(t, "Apporteur SAS", out.Referrer.Name)
}

func TestSnapshotResolver_ResolveForPayment_MissingDepsRequired(t *testing.T) {
	r := NewSnapshotResolver(SnapshotResolverDeps{
		Users: UserOrgFunc(func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
			return uuid.Nil, nil
		}),
	})
	_, err := r.ResolveForPayment(context.Background(), portservice.ReceiptSnapshotInput{})
	assert.Error(t, err)
}

func TestSnapshotResolver_ResolveForPayment_BillingProfileMissing_EmptyParty(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	clientUser := uuid.New()
	providerUser := uuid.New()

	users := UserOrgFunc(func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
		if userID == clientUser {
			return clientOrg, nil
		}
		if userID == providerUser {
			return providerOrg, nil
		}
		return uuid.Nil, nil
	})
	// Billing returns ErrNotFound for both orgs.
	billing := &fakeBillingProfiles{err: invoicingdomain.ErrNotFound}

	r := NewSnapshotResolver(SnapshotResolverDeps{
		Users:     users,
		Billing:   billing,
		Referrals: &fakeReferrals{},
	})

	out, err := r.ResolveForPayment(context.Background(), portservice.ReceiptSnapshotInput{
		ClientUserID:   clientUser,
		ProviderUserID: providerUser,
	})
	require.NoError(t, err)
	assert.Equal(t, clientOrg, out.Client.OrganizationID)
	assert.Empty(t, out.Client.Name)
	assert.Empty(t, out.Client.SIRET)
}

func TestSnapshotResolver_MarshalSnapshot_RoundTrip(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()

	r := &SnapshotResolver{}
	snap := portservice.ReceiptSnapshot{
		Client: portservice.ReceiptSnapshotParty{
			OrganizationID: clientOrg,
			Name:           "Client SAS",
			SIRET:          "12345678900012",
			AddressLine1:   "1 rue de Paris",
			City:           "Paris",
			PostalCode:     "75001",
			Country:        "FR",
		},
		Provider: portservice.ReceiptSnapshotParty{
			OrganizationID: providerOrg,
			Name:           "Provider SARL",
		},
	}
	raw, err := r.MarshalSnapshot(snap)
	require.NoError(t, err)
	assert.NotEmpty(t, raw)

	// The resulting JSON must contain the canonical snapshot keys.
	var generic map[string]any
	require.NoError(t, json.Unmarshal(raw, &generic))
	assert.Contains(t, generic, "client")
	assert.Contains(t, generic, "provider")
	assert.Contains(t, generic, "referrer")
	assert.Contains(t, generic, "referrer_commission_amount_cents")

	client := generic["client"].(map[string]any)
	assert.Equal(t, clientOrg.String(), client["organization_id"])
	assert.Equal(t, "Client SAS", client["name"])
	assert.Equal(t, "12345678900012", client["siret"])
}

func TestSnapshotResolver_MarshalSnapshot_EmptyReturnsNil(t *testing.T) {
	r := &SnapshotResolver{}
	raw, err := r.MarshalSnapshot(portservice.ReceiptSnapshot{})
	require.NoError(t, err)
	assert.Nil(t, raw)
}

// TestReceiptSnapshot_IsEmpty_TableDriven exercises the port helper
// that backs the marshal short-circuit. Lives in this package so the
// pair "empty → nil JSON" stays a single test sweep.
func TestReceiptSnapshot_IsEmpty_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		s    portservice.ReceiptSnapshot
		want bool
	}{
		{"all zero", portservice.ReceiptSnapshot{}, true},
		{"client set", portservice.ReceiptSnapshot{Client: portservice.ReceiptSnapshotParty{OrganizationID: uuid.New()}}, false},
		{"provider set", portservice.ReceiptSnapshot{Provider: portservice.ReceiptSnapshotParty{OrganizationID: uuid.New()}}, false},
		{"referrer set", portservice.ReceiptSnapshot{Referrer: &portservice.ReceiptSnapshotParty{OrganizationID: uuid.New()}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.IsEmpty())
		})
	}
}

// Sentinel-trip test: ensure errors from the closure-shaped reader
// don't bubble up — empty-fallback contract.
func TestSnapshotResolver_NilUsersFallsBackToEmpty(t *testing.T) {
	r := NewSnapshotResolver(SnapshotResolverDeps{
		Users: UserOrgFunc(func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
			return uuid.Nil, errors.New("boom")
		}),
		Billing:   &fakeBillingProfiles{},
		Referrals: &fakeReferrals{},
	})

	out, err := r.ResolveForPayment(context.Background(), portservice.ReceiptSnapshotInput{
		ClientUserID:   uuid.New(),
		ProviderUserID: uuid.New(),
	})
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, out.Client.OrganizationID)
	assert.Equal(t, uuid.Nil, out.Provider.OrganizationID)
}
