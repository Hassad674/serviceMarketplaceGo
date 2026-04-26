package invoicing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/invoicing"
)

func TestDetermineRegime(t *testing.T) {
	tests := []struct {
		name             string
		issuerCountry    string
		recipientCountry string
		hasValidVAT      bool
		want             invoicing.TaxRegime
	}{
		// Same country = domestic. Issuer is auto-entrepreneur en franchise.
		{"FR issuer, FR recipient", "FR", "FR", false, invoicing.RegimeFRFranchiseBase},
		{"FR issuer, FR recipient with VAT (irrelevant)", "FR", "FR", true, invoicing.RegimeFRFranchiseBase},

		// Other EU + valid VAT → reverse charge applies.
		{"FR issuer, DE recipient with VAT", "FR", "DE", true, invoicing.RegimeEUReverseCharge},
		{"FR issuer, IT recipient with VAT", "FR", "IT", true, invoicing.RegimeEUReverseCharge},
		{"FR issuer, BE recipient with VAT", "FR", "BE", true, invoicing.RegimeEUReverseCharge},

		// Other EU + no validated VAT → fall back to fr_franchise_base
		// (defense in depth — the upstream completeness gate should
		// have rejected this state).
		{"FR issuer, DE recipient no VAT", "FR", "DE", false, invoicing.RegimeFRFranchiseBase},

		// Outside EU = out of scope. Brexit means UK is now in this bucket.
		{"FR issuer, US recipient", "FR", "US", false, invoicing.RegimeOutOfScopeEU},
		{"FR issuer, UK recipient (post-Brexit)", "FR", "GB", true, invoicing.RegimeOutOfScopeEU},
		{"FR issuer, CH recipient", "FR", "CH", true, invoicing.RegimeOutOfScopeEU},
		{"FR issuer, JP recipient", "FR", "JP", false, invoicing.RegimeOutOfScopeEU},

		// Lowercase + whitespace tolerance.
		{"lowercase + whitespace recipient", "FR", " de ", true, invoicing.RegimeEUReverseCharge},

		// Empty recipient country defaults to fr_franchise_base
		// (safest mention — never claims an unauthorised reverse charge).
		{"empty recipient country", "FR", "", true, invoicing.RegimeFRFranchiseBase},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := invoicing.DetermineRegime(tc.issuerCountry, tc.recipientCountry, tc.hasValidVAT)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsEUCountry(t *testing.T) {
	assert.True(t, invoicing.IsEUCountry("FR"))
	assert.True(t, invoicing.IsEUCountry("fr")) // case-insensitive
	assert.True(t, invoicing.IsEUCountry("DE"))
	assert.True(t, invoicing.IsEUCountry("HR")) // Croatia (joined 2013)
	assert.False(t, invoicing.IsEUCountry("GB"), "UK left in 2020")
	assert.False(t, invoicing.IsEUCountry("US"))
	assert.False(t, invoicing.IsEUCountry("CH"), "Switzerland never joined")
	assert.False(t, invoicing.IsEUCountry(""))
}

func TestTaxRegime_IsValid(t *testing.T) {
	assert.True(t, invoicing.RegimeFRFranchiseBase.IsValid())
	assert.True(t, invoicing.RegimeEUReverseCharge.IsValid())
	assert.True(t, invoicing.RegimeOutOfScopeEU.IsValid())
	assert.False(t, invoicing.TaxRegime("nonsense").IsValid())
	assert.False(t, invoicing.TaxRegime("").IsValid())
}
