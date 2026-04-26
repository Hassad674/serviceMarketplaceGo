package email_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/email"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/service"
)

// stubEmailService implements service.EmailService and records every
// SendNotification call, so tests can assert subject/body/recipient
// without spinning a real Resend client.
type stubEmailService struct {
	calls []notificationCall
}

type notificationCall struct {
	to      string
	subject string
	html    string
}

func (s *stubEmailService) SendPasswordReset(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *stubEmailService) SendNotification(_ context.Context, to, subject, html string) error {
	s.calls = append(s.calls, notificationCall{to: to, subject: subject, html: html})
	return nil
}

func (s *stubEmailService) SendTeamInvitation(_ context.Context, _ service.TeamInvitationEmailInput) error {
	return nil
}

func (s *stubEmailService) SendRolePermissionsChanged(_ context.Context, _ service.RolePermissionsChangedEmailInput) error {
	return nil
}

func newFinalizedInvoice(t *testing.T, recipientCountry, recipientEmail string) *invoicing.Invoice {
	t.Helper()
	itemID := uuid.New()
	now := time.Now()
	inv := &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000123",
		RecipientOrganizationID: uuid.New(),
		RecipientSnapshot: invoicing.RecipientInfo{
			OrganizationID: uuid.NewString(),
			ProfileType:    "business",
			LegalName:      "Recipient SARL",
			Country:        recipientCountry,
			Email:          recipientEmail,
		},
		IssuerSnapshot: invoicing.IssuerInfo{
			LegalName: "Marketplace Service SAS",
			SIRET:     "87891296300012",
			Country:   "FR",
		},
		IssuedAt:           now,
		ServicePeriodStart: now.AddDate(0, -1, 0),
		ServicePeriodEnd:   now,
		Currency:           "EUR",
		AmountExclTaxCents: 12000,
		VATAmountCents:     0,
		AmountInclTaxCents: 12000,
		TaxRegime:          invoicing.RegimeFRFranchiseBase,
		MentionsRendered:   []string{"TVA non applicable, art. 293 B du CGI"},
		SourceType:         invoicing.SourceSubscription,
		Status:             invoicing.StatusIssued,
		FinalizedAt:        &now,
		Items: []invoicing.InvoiceItem{
			{
				ID:             itemID,
				Description:    "Premium subscription — April 2026",
				Quantity:       1,
				UnitPriceCents: 12000,
				AmountCents:    12000,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return inv
}

func TestDeliverer_DeliverInvoice_FrenchRecipient(t *testing.T) {
	stub := &stubEmailService{}
	d := email.NewDeliverer(stub)

	inv := newFinalizedInvoice(t, "FR", "client@example.fr")

	err := d.DeliverInvoice(context.Background(), inv, []byte("ignored-pdf-bytes"), "https://files.example.com/inv/FAC-000123.pdf")
	require.NoError(t, err)
	require.Len(t, stub.calls, 1)

	call := stub.calls[0]
	assert.Equal(t, "client@example.fr", call.to)
	assert.Equal(t, "Facture FAC-000123 disponible", call.subject)
	assert.Contains(t, call.html, "FAC-000123")
	assert.Contains(t, call.html, "120,00 €")
	assert.Contains(t, call.html, "https://files.example.com/inv/FAC-000123.pdf")
	assert.Contains(t, call.html, "87891296300012", "SIRET footer must be present")
}

func TestDeliverer_DeliverInvoice_EnglishRecipient(t *testing.T) {
	stub := &stubEmailService{}
	d := email.NewDeliverer(stub)

	inv := newFinalizedInvoice(t, "US", "client@example.com")

	err := d.DeliverInvoice(context.Background(), inv, nil, "https://files.example.com/inv/FAC-000123.pdf")
	require.NoError(t, err)
	require.Len(t, stub.calls, 1)

	call := stub.calls[0]
	assert.Equal(t, "client@example.com", call.to)
	assert.Equal(t, "Invoice FAC-000123 available", call.subject)
	assert.Contains(t, call.html, "Your invoice is available")
	assert.Contains(t, call.html, "EUR 120.00")
}

func TestDeliverer_DeliverInvoice_NoRecipientEmail_Errors(t *testing.T) {
	stub := &stubEmailService{}
	d := email.NewDeliverer(stub)

	inv := newFinalizedInvoice(t, "FR", "")

	err := d.DeliverInvoice(context.Background(), inv, nil, "https://example.com/x.pdf")
	require.Error(t, err)
	assert.Empty(t, stub.calls, "no email must be sent when recipient address is missing")
}

func TestDeliverer_DeliverCreditNote_FrenchRecipient(t *testing.T) {
	stub := &stubEmailService{}
	d := email.NewDeliverer(stub)

	now := time.Now()
	cn := &invoicing.CreditNote{
		ID:                      uuid.New(),
		Number:                  "AV-000007",
		OriginalInvoiceID:       uuid.New(),
		RecipientOrganizationID: uuid.New(),
		RecipientSnapshot: invoicing.RecipientInfo{
			LegalName: "Recipient SARL",
			Country:   "FR",
			Email:     "client@example.fr",
		},
		IssuerSnapshot: invoicing.IssuerInfo{
			LegalName: "Marketplace Service SAS",
			SIRET:     "87891296300012",
			Country:   "FR",
		},
		IssuedAt:           now,
		Reason:             "Refund of subscription overcharge",
		Currency:           "EUR",
		AmountExclTaxCents: 5000,
		AmountInclTaxCents: 5000,
		TaxRegime:          invoicing.RegimeFRFranchiseBase,
		FinalizedAt:        &now,
	}

	err := d.DeliverCreditNote(context.Background(), cn, nil, "https://files.example.com/avoirs/AV-000007.pdf")
	require.NoError(t, err)
	require.Len(t, stub.calls, 1)

	call := stub.calls[0]
	assert.Equal(t, "client@example.fr", call.to)
	assert.Equal(t, "Avoir AV-000007 disponible", call.subject)
	assert.Contains(t, call.html, "AV-000007")
	assert.Contains(t, call.html, "50,00 €")
	assert.Contains(t, call.html, "https://files.example.com/avoirs/AV-000007.pdf")
}
