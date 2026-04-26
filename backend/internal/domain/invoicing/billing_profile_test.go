package invoicing_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/invoicing"
)

func validatedAt() *time.Time {
	t := time.Now().Add(-1 * time.Hour)
	return &t
}

func completeFRBusiness() invoicing.BillingProfile {
	return invoicing.BillingProfile{
		OrganizationID: uuid.New(),
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "ACME SARL",
		LegalForm:      "SARL",
		TaxID:          "12345678900012", // 14 digits SIRET
		AddressLine1:   "1 boulevard Haussmann",
		PostalCode:     "75009",
		City:           "Paris",
		Country:        "FR",
		InvoicingEmail: "billing@acme.fr",
	}
}

func completeDEBusiness() invoicing.BillingProfile {
	return invoicing.BillingProfile{
		OrganizationID: uuid.New(),
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "ACME GmbH",
		VATNumber:      "DE123456789",
		VATValidatedAt: validatedAt(),
		AddressLine1:   "Berliner Straße 5",
		PostalCode:     "10117",
		City:           "Berlin",
		Country:        "DE",
		InvoicingEmail: "billing@acme.de",
	}
}

func completeUSBusiness() invoicing.BillingProfile {
	return invoicing.BillingProfile{
		OrganizationID: uuid.New(),
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "ACME Corp",
		AddressLine1:   "1 Main St",
		PostalCode:     "10001",
		City:           "New York",
		Country:        "US",
		InvoicingEmail: "billing@acme.com",
	}
}

func TestCheckCompleteness_FRComplete(t *testing.T) {
	got := invoicing.CheckCompleteness(completeFRBusiness())
	assert.Empty(t, got)
}

func TestCheckCompleteness_FRMissingSIRET(t *testing.T) {
	p := completeFRBusiness()
	p.TaxID = ""
	got := invoicing.CheckCompleteness(p)
	require := assert.New(t)
	require.Len(got, 1)
	require.Equal("tax_id", got[0].Field)
}

func TestCheckCompleteness_EUComplete(t *testing.T) {
	got := invoicing.CheckCompleteness(completeDEBusiness())
	assert.Empty(t, got)
}

func TestCheckCompleteness_EUMissingVAT(t *testing.T) {
	p := completeDEBusiness()
	p.VATNumber = ""
	got := invoicing.CheckCompleteness(p)
	require := assert.New(t)
	require.Len(got, 1)
	require.Equal("vat_number", got[0].Field)
}

func TestCheckCompleteness_EUVATNotValidated(t *testing.T) {
	p := completeDEBusiness()
	p.VATValidatedAt = nil
	got := invoicing.CheckCompleteness(p)
	require := assert.New(t)
	require.Len(got, 1)
	require.Equal("vat_number", got[0].Field)
	require.Contains(got[0].Reason, "validated against VIES")
}

func TestCheckCompleteness_USComplete(t *testing.T) {
	got := invoicing.CheckCompleteness(completeUSBusiness())
	assert.Empty(t, got, "outside the EU only universal fields are required")
}

func TestCheckCompleteness_UniversalFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*invoicing.BillingProfile)
		wantKey string
	}{
		{"missing legal name", func(p *invoicing.BillingProfile) { p.LegalName = "" }, "legal_name"},
		{"missing address", func(p *invoicing.BillingProfile) { p.AddressLine1 = "" }, "address_line1"},
		{"missing postal", func(p *invoicing.BillingProfile) { p.PostalCode = "" }, "postal_code"},
		{"missing city", func(p *invoicing.BillingProfile) { p.City = "" }, "city"},
		{"missing country", func(p *invoicing.BillingProfile) { p.Country = "" }, "country"},
		{"missing email", func(p *invoicing.BillingProfile) { p.InvoicingEmail = "" }, "invoicing_email"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := completeFRBusiness()
			tc.mutate(&p)
			got := invoicing.CheckCompleteness(p)
			require := assert.New(t)
			require.NotEmpty(got)
			fields := make(map[string]struct{}, len(got))
			for _, m := range got {
				fields[m.Field] = struct{}{}
			}
			require.Contains(fields, tc.wantKey)
		})
	}
}

func TestBillingProfile_IsComplete(t *testing.T) {
	assert.True(t, completeFRBusiness().IsComplete())
	p := completeFRBusiness()
	p.TaxID = ""
	assert.False(t, p.IsComplete())
}

func TestProfileType_IsValid(t *testing.T) {
	assert.True(t, invoicing.ProfileIndividual.IsValid())
	assert.True(t, invoicing.ProfileBusiness.IsValid())
	assert.False(t, invoicing.ProfileType("nope").IsValid())
}
