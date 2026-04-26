package invoicing_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/invoicing"
)

func validItem() invoicing.InvoiceItem {
	return invoicing.InvoiceItem{
		ID:             uuid.New(),
		Description:    "Premium Agence — avril 2026",
		Quantity:       1,
		UnitPriceCents: 4900,
		AmountCents:    4900,
	}
}

func validNewInvoiceInput() invoicing.NewInvoiceInput {
	return invoicing.NewInvoiceInput{
		RecipientOrganizationID: uuid.New(),
		Recipient: invoicing.RecipientInfo{
			OrganizationID: uuid.New().String(),
			LegalName:      "ACME SARL",
			Country:        "FR",
		},
		Issuer: invoicing.IssuerInfo{
			LegalName: "Hassad Smara",
			Country:   "FR",
		},
		ServicePeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ServicePeriodEnd:   time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		SourceType:         invoicing.SourceSubscription,
		StripeEventID:      "evt_test",
		Items:              []invoicing.InvoiceItem{validItem()},
	}
}

func TestNewInvoice_HappyPath(t *testing.T) {
	in := validNewInvoiceInput()
	inv, err := invoicing.NewInvoice(in)

	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, invoicing.StatusDraft, inv.Status)
	assert.Equal(t, int64(4900), inv.AmountExclTaxCents)
	assert.Equal(t, int64(4900), inv.AmountInclTaxCents)
	assert.Equal(t, invoicing.RegimeFRFranchiseBase, inv.TaxRegime)
	assert.NotEmpty(t, inv.MentionsRendered)
	assert.False(t, inv.IsFinalized())
	assert.Equal(t, "EUR", inv.Currency)
	assert.NotEqual(t, uuid.Nil, inv.ID)
}

func TestNewInvoice_DerivesRegimeFromCountries(t *testing.T) {
	tests := []struct {
		recipientCountry string
		recipientVAT     string
		want             invoicing.TaxRegime
	}{
		{"FR", "", invoicing.RegimeFRFranchiseBase},
		{"DE", "DE123456789", invoicing.RegimeEUReverseCharge},
		{"US", "", invoicing.RegimeOutOfScopeEU},
	}
	for _, tc := range tests {
		t.Run(tc.recipientCountry, func(t *testing.T) {
			in := validNewInvoiceInput()
			in.Recipient.Country = tc.recipientCountry
			in.Recipient.VATNumber = tc.recipientVAT
			inv, err := invoicing.NewInvoice(in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, inv.TaxRegime)
		})
	}
}

func TestNewInvoice_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*invoicing.NewInvoiceInput)
		wantErr error
	}{
		{
			"zero org id",
			func(in *invoicing.NewInvoiceInput) { in.RecipientOrganizationID = uuid.Nil },
			invoicing.ErrInvalidOrganization,
		},
		{
			"unknown source type",
			func(in *invoicing.NewInvoiceInput) { in.SourceType = invoicing.SourceType("garbage") },
			invoicing.ErrInvalidSourceType,
		},
		{
			"period inverted",
			func(in *invoicing.NewInvoiceInput) {
				in.ServicePeriodEnd = in.ServicePeriodStart.Add(-1 * time.Hour)
			},
			invoicing.ErrInvalidPeriod,
		},
		{
			"empty items",
			func(in *invoicing.NewInvoiceInput) { in.Items = nil },
			invoicing.ErrEmptyItems,
		},
		{
			"missing recipient country",
			func(in *invoicing.NewInvoiceInput) { in.Recipient.Country = "" },
			invoicing.ErrCountryRequired,
		},
		{
			"item amount mismatch",
			func(in *invoicing.NewInvoiceInput) {
				in.Items[0].UnitPriceCents = 100
				in.Items[0].Quantity = 2
				in.Items[0].AmountCents = 999 // should be 200
			},
			invoicing.ErrItemAmountMismatch,
		},
		{
			"negative amount",
			func(in *invoicing.NewInvoiceInput) {
				in.Items[0].UnitPriceCents = -100
				in.Items[0].AmountCents = -100
			},
			invoicing.ErrInvalidAmount,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validNewInvoiceInput()
			tc.mutate(&in)
			_, err := invoicing.NewInvoice(in)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestInvoice_Finalize(t *testing.T) {
	inv, err := invoicing.NewInvoice(validNewInvoiceInput())
	require.NoError(t, err)

	require.NoError(t, inv.Finalize("FAC-000001", "invoices/abc/FAC-000001.pdf"))

	assert.Equal(t, "FAC-000001", inv.Number)
	assert.Equal(t, "invoices/abc/FAC-000001.pdf", inv.PDFR2Key)
	assert.Equal(t, invoicing.StatusIssued, inv.Status)
	assert.True(t, inv.IsFinalized())
}

func TestInvoice_Finalize_AlreadyFinalized(t *testing.T) {
	inv, _ := invoicing.NewInvoice(validNewInvoiceInput())
	require.NoError(t, inv.Finalize("FAC-000001", "key"))

	err := inv.Finalize("FAC-000002", "key2")
	assert.ErrorIs(t, err, invoicing.ErrAlreadyFinalized)
}

func TestInvoice_Finalize_RequiresNumberAndKey(t *testing.T) {
	inv, _ := invoicing.NewInvoice(validNewInvoiceInput())
	assert.ErrorIs(t, inv.Finalize("", "key"), invoicing.ErrInvalidNumber)

	inv2, _ := invoicing.NewInvoice(validNewInvoiceInput())
	assert.ErrorIs(t, inv2.Finalize("FAC-000001", ""), invoicing.ErrPDFKeyRequired)
}

func TestInvoice_MarkCredited(t *testing.T) {
	inv, _ := invoicing.NewInvoice(validNewInvoiceInput())
	require.NoError(t, inv.Finalize("FAC-000001", "key"))

	require.NoError(t, inv.MarkCredited())
	assert.Equal(t, invoicing.StatusCredited, inv.Status)
}

func TestInvoice_MarkCredited_RejectsDraft(t *testing.T) {
	inv, _ := invoicing.NewInvoice(validNewInvoiceInput())
	err := inv.MarkCredited()
	assert.ErrorIs(t, err, invoicing.ErrAlreadyFinalized)
}

func TestInvoiceItem_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*invoicing.InvoiceItem)
		wantErr error
	}{
		{"happy", func(it *invoicing.InvoiceItem) {}, nil},
		{"empty desc", func(it *invoicing.InvoiceItem) { it.Description = "" }, invoicing.ErrInvalidAmount},
		{"zero qty", func(it *invoicing.InvoiceItem) { it.Quantity = 0 }, invoicing.ErrInvalidAmount},
		{"negative price", func(it *invoicing.InvoiceItem) { it.UnitPriceCents = -10 }, invoicing.ErrInvalidAmount},
		{"mismatch", func(it *invoicing.InvoiceItem) { it.AmountCents = 9999 }, invoicing.ErrItemAmountMismatch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			it := validItem()
			tc.mutate(&it)
			err := it.Validate()
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSourceType_IsValid(t *testing.T) {
	assert.True(t, invoicing.SourceSubscription.IsValid())
	assert.True(t, invoicing.SourceMonthlyCommission.IsValid())
	assert.False(t, invoicing.SourceType("nope").IsValid())
}

func TestStatus_IsValid(t *testing.T) {
	assert.True(t, invoicing.StatusDraft.IsValid())
	assert.True(t, invoicing.StatusIssued.IsValid())
	assert.True(t, invoicing.StatusCredited.IsValid())
	assert.False(t, invoicing.Status("nope").IsValid())
}
