package stripe

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/payment"
)

// TestCreateConnectedAccount_ForcesManualPayoutSchedule — the headline
// fix. Stripe Connect Custom accounts default to automatic daily
// payouts in FR (and most countries) which fires the moment KYC
// finishes. We force payout_schedule.interval = "manual" at creation
// so the only way funds leave Stripe is the wallet "Retirer" button.
//
// The test inspects the actual form body sent to Stripe rather than
// patching the SDK, so a future regression that drops the Settings
// param fails loudly here instead of silently in production.
func TestCreateConnectedAccount_ForcesManualPayoutSchedule(t *testing.T) {
	// The two paths of /v1/tokens (account token) + /v1/accounts each
	// get the same canned response — we only care about the form body
	// of the /v1/accounts call.
	cap, restore := stubBackends(t,
		`{"id":"acct_test_force","object":"account","external_accounts":{"data":[]}}`)
	defer restore()

	svc := NewService("sk_test_unit", "")
	dob := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)
	info := &payment.PaymentInfo{
		FirstName:     "Jean",
		LastName:      "Dupont",
		Phone:         "+33600000000",
		DateOfBirth:   dob,
		Address:       "1 rue de la Paix",
		City:          "Paris",
		PostalCode:    "75002",
		Country:       "FR",
		Nationality:   "FR",
		IBAN:          "FR1420041010050500013M02606",
		AccountHolder: "Jean Dupont",
	}

	// We don't care whether the call ultimately succeeds — Stripe's
	// stub only returns one canned account body. We care that the
	// /v1/accounts POST contained the manual schedule directive.
	_, _ = svc.CreateConnectedAccount(context.Background(), info, "127.0.0.1", "jean@example.com")

	cap.mu.Lock()
	defer cap.mu.Unlock()
	var accountsBody string
	for i, p := range cap.paths {
		if p == "/v1/accounts" {
			accountsBody = cap.bodies[i]
			break
		}
	}
	require.NotEmpty(t, accountsBody, "expected a POST to /v1/accounts in: %v", cap.paths)
	assert.True(t,
		strings.Contains(accountsBody, "settings[payouts][schedule][interval]=manual"),
		"account creation MUST set payout_schedule.interval=manual, got body: %s", accountsBody)
}

// TestUpdateConnectedAccount_ReassertsManualSchedule — defense-in-depth.
// Every account update re-sends the manual schedule so a stray edit in
// the Stripe Dashboard that flipped an account back to daily payouts
// is corrected the next time the user touches their billing form.
func TestUpdateConnectedAccount_ReassertsManualSchedule(t *testing.T) {
	cap, restore := stubBackends(t,
		`{"id":"acct_test_update","object":"account","external_accounts":{"data":[]}}`)
	defer restore()

	svc := NewService("sk_test_unit", "")
	dob := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)
	info := &payment.PaymentInfo{
		FirstName:     "Jean",
		LastName:      "Dupont",
		Phone:         "+33600000000",
		DateOfBirth:   dob,
		Address:       "1 rue de la Paix",
		City:          "Paris",
		PostalCode:    "75002",
		Country:       "FR",
		Nationality:   "FR",
		IBAN:          "FR1420041010050500013M02606",
		AccountHolder: "Jean Dupont",
	}
	_ = svc.UpdateConnectedAccount(context.Background(), "acct_test_update", info, "127.0.0.1", "jean@example.com")

	cap.mu.Lock()
	defer cap.mu.Unlock()
	var updateBody string
	for i, p := range cap.paths {
		if p == "/v1/accounts/acct_test_update" {
			updateBody = cap.bodies[i]
			break
		}
	}
	require.NotEmpty(t, updateBody, "expected a POST to /v1/accounts/acct_test_update in: %v", cap.paths)
	assert.True(t,
		strings.Contains(updateBody, "settings[payouts][schedule][interval]=manual"),
		"account update MUST re-assert payout_schedule.interval=manual, got body: %s", updateBody)
}
