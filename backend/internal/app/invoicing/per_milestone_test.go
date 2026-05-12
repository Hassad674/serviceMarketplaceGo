package invoicing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/domain/payment"
)

// validPaymentRecord builds a payment record fixture suitable for the
// per-milestone emission tests. Carries the platform fee, milestone,
// and the canonical EUR currency.
func validPaymentRecord(platformFee int64) *payment.PaymentRecord {
	rec := payment.NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10_000, 175, platformFee)
	return rec
}

// validProviderProfile builds a BillingProfile fixture that satisfies
// the recipient snapshot invariants (non-empty country, valid org id).
func validProviderProfile(orgID uuid.UUID) *invoicing.BillingProfile {
	return &invoicing.BillingProfile{
		OrganizationID: orgID,
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "Provider SAS",
		AddressLine1:   "1 rue de la République",
		PostalCode:     "75001",
		City:           "Paris",
		Country:        "FR",
		InvoicingEmail: "provider@example.com",
	}
}

// newTestServiceFor wires a Service with default mocks suitable for the
// per-milestone tests. Returns the service AND the underlying mocks so
// tests can stub behavior.
func newTestServiceFor(t *testing.T) (*invoicingapp.Service, *mockInvoiceRepo, *mockProfileRepo) {
	t.Helper()
	repo := &mockInvoiceRepo{}
	profiles := &mockProfileRepo{}
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    profiles,
		PDF:         &mockPDF{},
		Storage:     &mockStorage{},
		Deliverer:   &mockDeliverer{},
		Issuer:      validIssuer(),
		Idempotency: &mockIdempotency{},
	})
	return svc, repo, profiles
}

// validIssuer returns a deterministic IssuerInfo so the test does not
// depend on env vars.
func validIssuer() invoicing.IssuerInfo {
	return invoicing.IssuerInfo{
		LegalName:    "Hassad Smara",
		LegalForm:    "EI",
		SIRET:        "87891296300010",
		APECode:      "6202A",
		AddressLine1: "1 rue du Test",
		PostalCode:   "75001",
		City:         "Paris",
		Country:      "FR",
		Email:        "contact@example.com",
	}
}

func TestIssueFromMilestone_HappyPath(t *testing.T) {
	svc, repo, profiles := newTestServiceFor(t)

	providerOrg := uuid.New()
	rec := validPaymentRecord(500) // 5 EUR platform fee
	approved := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)

	profiles.findByOrgFn = func(ctx context.Context, orgID uuid.UUID) (*invoicing.BillingProfile, error) {
		assert.Equal(t, providerOrg, orgID)
		return validProviderProfile(orgID), nil
	}

	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: providerOrg,
		ApprovedAt:             approved,
	})
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.True(t, inv.IsPlatformFee())
	assert.Equal(t, int64(500), inv.AmountExclTaxCents)
	assert.Equal(t, int64(500), inv.AmountInclTaxCents)
	require.NotNil(t, inv.MilestoneID)
	assert.Equal(t, rec.MilestoneID, *inv.MilestoneID)
	assert.Equal(t, providerOrg, inv.RecipientOrganizationID)
	assert.Len(t, repo.persistedInvoices, 1)
	persisted := repo.persistedInvoices[0]
	require.NotNil(t, persisted.MilestoneID)
	assert.Equal(t, rec.MilestoneID, *persisted.MilestoneID)
}

func TestIssueFromMilestone_AmountExclStripeFees(t *testing.T) {
	// Regression: the invoice MUST bill ONLY the platform fee, NEVER
	// include the Stripe processing fee — Stripe fees are charged to
	// the client and are not legally refacturable.
	svc, _, profiles := newTestServiceFor(t)

	providerOrg := uuid.New()
	rec := validPaymentRecord(500) // 5 EUR platform
	rec.StripeFeeAmount = 175      // 1.75 EUR Stripe processing
	rec.ClientTotalAmount = rec.ProposalAmount + rec.StripeFeeAmount

	profiles.findByOrgFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return validProviderProfile(providerOrg), nil
	}

	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: providerOrg,
	})
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, int64(500), inv.AmountExclTaxCents,
		"invoice must bill platform_fee_amount only, NOT include stripe_fee_amount")
	require.Len(t, inv.Items, 1)
	assert.Equal(t, int64(500), inv.Items[0].AmountCents)
}

func TestIssueFromMilestone_PremiumSkip(t *testing.T) {
	// Premium provider at funding time → PaymentRecord.PlatformFeeAmount==0
	// → skip emission, return (nil, nil).
	svc, repo, _ := newTestServiceFor(t)

	rec := validPaymentRecord(0) // Premium waiver
	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: uuid.New(),
	})
	require.NoError(t, err)
	assert.Nil(t, inv)
	assert.Empty(t, repo.persistedInvoices, "no invoice persisted for premium-waived record")
}

func TestIssueFromMilestone_Idempotent(t *testing.T) {
	svc, repo, profiles := newTestServiceFor(t)

	providerOrg := uuid.New()
	rec := validPaymentRecord(500)
	profiles.findByOrgFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return validProviderProfile(providerOrg), nil
	}

	// First call — issues.
	inv1, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: providerOrg,
	})
	require.NoError(t, err)
	require.NotNil(t, inv1)

	// Second call — must return the same row without re-persisting.
	repo.findPlatformFeeByMilFn = func(ctx context.Context, mid uuid.UUID) (*invoicing.Invoice, error) {
		assert.Equal(t, rec.MilestoneID, mid)
		return inv1, nil
	}
	inv2, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: providerOrg,
	})
	require.NoError(t, err)
	require.NotNil(t, inv2)
	assert.Equal(t, inv1.ID, inv2.ID)
	assert.Len(t, repo.persistedInvoices, 1, "no second persist on idempotent replay")
}

func TestIssueFromMilestone_RejectsInvalidInput(t *testing.T) {
	svc, _, _ := newTestServiceFor(t)

	tests := []struct {
		name    string
		in      invoicingapp.IssueFromMilestoneInput
		wantSub string
	}{
		{
			"nil payment record",
			invoicingapp.IssueFromMilestoneInput{ProviderOrganizationID: uuid.New()},
			"payment record required",
		},
		{
			"zero milestone id on record",
			invoicingapp.IssueFromMilestoneInput{
				PaymentRecord: &payment.PaymentRecord{
					ID: uuid.New(), PlatformFeeAmount: 500,
				},
				ProviderOrganizationID: uuid.New(),
			},
			"milestone id must be non-zero",
		},
		{
			"zero provider org id",
			invoicingapp.IssueFromMilestoneInput{
				PaymentRecord: validPaymentRecord(500),
			},
			"provider organization id required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inv, err := svc.IssueFromMilestone(context.Background(), tc.in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
			assert.Nil(t, inv)
		})
	}
}

func TestIssueFromMilestone_PropagatesDedupProbeFailure(t *testing.T) {
	svc, repo, _ := newTestServiceFor(t)

	wantErr := errors.New("db is down")
	repo.findPlatformFeeByMilFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return nil, wantErr
	}

	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          validPaymentRecord(500),
		ProviderOrganizationID: uuid.New(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, inv)
}

func TestIssueFromMilestone_PropagatesMissingBillingProfile(t *testing.T) {
	svc, _, profiles := newTestServiceFor(t)

	profiles.findByOrgFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return nil, invoicing.ErrNotFound
	}

	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          validPaymentRecord(500),
		ProviderOrganizationID: uuid.New(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without billing profile")
	assert.Nil(t, inv)
}

func TestIssueFromMilestone_DefaultsApprovedAtToNow(t *testing.T) {
	svc, repo, profiles := newTestServiceFor(t)
	providerOrg := uuid.New()
	profiles.findByOrgFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return validProviderProfile(providerOrg), nil
	}

	before := time.Now().UTC().Add(-1 * time.Second)
	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          validPaymentRecord(500),
		ProviderOrganizationID: providerOrg,
		// ApprovedAt left zero on purpose.
	})
	require.NoError(t, err)
	require.NotNil(t, inv)
	require.Len(t, repo.persistedInvoices, 1)
	after := time.Now().UTC().Add(1 * time.Second)
	persisted := repo.persistedInvoices[0]
	assert.False(t, persisted.ServicePeriodStart.Before(before), "ServicePeriodStart must default to now")
	assert.False(t, persisted.ServicePeriodEnd.After(after), "ServicePeriodEnd must default to now")
}

// TestIssueFromMilestone_SkipsIncompleteBillingProfile is the
// defense-in-depth gate added with the transfer.completed trigger
// move. When the billing_profile is loaded but missing any of the five
// universal fields (legal_name, country, address_line1, postal_code,
// city), the emission MUST be a silent no-op: (nil, nil) returned, no
// invoice persisted, the monthly safety-net scheduler retries on its
// next run.
//
// Note: ErrNotFound (no profile row at all) is a different code path —
// that one propagates as an error because it indicates a configuration
// bug at onboarding. The gate below only covers the "row exists but
// not yet hydrated" edge case.
func TestIssueFromMilestone_SkipsIncompleteBillingProfile(t *testing.T) {
	tests := []struct {
		name         string
		mutator      func(*invoicing.BillingProfile)
		missingLabel string
	}{
		{
			name:         "missing legal_name",
			mutator:      func(p *invoicing.BillingProfile) { p.LegalName = "" },
			missingLabel: "legal_name",
		},
		{
			name:         "missing country",
			mutator:      func(p *invoicing.BillingProfile) { p.Country = "" },
			missingLabel: "country",
		},
		{
			name:         "missing address_line1",
			mutator:      func(p *invoicing.BillingProfile) { p.AddressLine1 = "" },
			missingLabel: "address_line1",
		},
		{
			name:         "missing postal_code",
			mutator:      func(p *invoicing.BillingProfile) { p.PostalCode = "" },
			missingLabel: "postal_code",
		},
		{
			name:         "missing city",
			mutator:      func(p *invoicing.BillingProfile) { p.City = "" },
			missingLabel: "city",
		},
		{
			name: "whitespace-only legal_name",
			mutator: func(p *invoicing.BillingProfile) {
				p.LegalName = "   "
			},
			missingLabel: "legal_name (whitespace)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, profiles := newTestServiceFor(t)
			providerOrg := uuid.New()
			profiles.findByOrgFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
				profile := validProviderProfile(providerOrg)
				tc.mutator(profile)
				return profile, nil
			}

			inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
				PaymentRecord:          validPaymentRecord(500),
				ProviderOrganizationID: providerOrg,
			})

			require.NoError(t, err, "incomplete-profile path must not surface an error (%s)", tc.missingLabel)
			assert.Nil(t, inv, "no invoice issued when %s is missing", tc.missingLabel)
			assert.Empty(t, repo.persistedInvoices, "no invoice persisted when %s is missing", tc.missingLabel)
		})
	}
}

// TestIssueFromMilestone_IssuesWithEUVATNotValidated locks in the
// defense-in-depth gate's narrow scope: per the universal-field rule
// (legal_name + country + address), an EU provider whose VAT number is
// not yet validated MUST still get the invoice — country-specific
// rules (FR SIRET, EU validated VAT) are the operator's responsibility
// to fix via the Mes infos page, not a blocker for the per-milestone
// emission. The monthly consolidation also relies on this rule.
func TestIssueFromMilestone_IssuesWithEUVATNotValidated(t *testing.T) {
	svc, repo, profiles := newTestServiceFor(t)
	providerOrg := uuid.New()
	profiles.findByOrgFn = func(ctx context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		p := validProviderProfile(providerOrg)
		// Re-target to a non-FR EU country with no VAT validated.
		p.Country = "DE"
		p.VATNumber = ""
		p.VATValidatedAt = nil
		p.TaxID = ""
		return p, nil
	}

	inv, err := svc.IssueFromMilestone(context.Background(), invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          validPaymentRecord(500),
		ProviderOrganizationID: providerOrg,
	})
	require.NoError(t, err)
	require.NotNil(t, inv, "per-milestone path must NOT block on country-specific tax IDs — universal fields are sufficient")
	assert.Len(t, repo.persistedInvoices, 1)
}
