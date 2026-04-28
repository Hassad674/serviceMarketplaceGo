package stripe

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripeapi "github.com/stripe/stripe-go/v82"

	portservice "marketplace-backend/internal/port/service"
)

// stubBackends installs an httptest server as Stripe's API + Connect
// backends and returns a function that captures every request body.
// The Stripe SDK reads stripeapi.Key + the global backends, so tests
// that share state must restore everything in their cleanup. The
// caller serializes its assertions on captured.bodies under capMu.
type captured struct {
	mu     sync.Mutex
	paths  []string
	bodies []string
}

func stubBackends(t *testing.T, response string) (cap *captured, restore func()) {
	t.Helper()
	cap = &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		cap.mu.Lock()
		cap.paths = append(cap.paths, r.URL.Path)
		cap.bodies = append(cap.bodies, string(body))
		cap.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))

	prevAPI := stripeapi.GetBackend(stripeapi.APIBackend)
	prevConn := stripeapi.GetBackend(stripeapi.ConnectBackend)
	prevKey := stripeapi.Key
	stripeapi.Key = "sk_test_unit"
	api := stripeapi.GetBackendWithConfig(stripeapi.APIBackend, &stripeapi.BackendConfig{
		URL:           stripeapi.String(srv.URL),
		LeveledLogger: &stripeapi.LeveledLogger{Level: stripeapi.LevelNull},
	})
	conn := stripeapi.GetBackendWithConfig(stripeapi.ConnectBackend, &stripeapi.BackendConfig{
		URL:           stripeapi.String(srv.URL),
		LeveledLogger: &stripeapi.LeveledLogger{Level: stripeapi.LevelNull},
	})
	stripeapi.SetBackend(stripeapi.APIBackend, api)
	stripeapi.SetBackend(stripeapi.ConnectBackend, conn)

	return cap, func() {
		srv.Close()
		stripeapi.SetBackend(stripeapi.APIBackend, prevAPI)
		stripeapi.SetBackend(stripeapi.ConnectBackend, prevConn)
		stripeapi.Key = prevKey
	}
}

// TestUpdatePayoutSchedule_PostsManualInterval — the adapter must POST
// settings[payouts][schedule][interval]=manual to /v1/accounts/<id>.
// This is the surface used by the stripe-payout-schedule-backfill CLI.
func TestUpdatePayoutSchedule_PostsManualInterval(t *testing.T) {
	cap, restore := stubBackends(t, `{"id":"acct_123","object":"account"}`)
	defer restore()

	svc := NewService("sk_test_unit", "")
	err := svc.UpdatePayoutSchedule(context.Background(), "acct_123", "manual")
	require.NoError(t, err)

	cap.mu.Lock()
	defer cap.mu.Unlock()
	require.Len(t, cap.bodies, 1, "exactly one Stripe call")
	assert.Equal(t, "/v1/accounts/acct_123", cap.paths[0])
	assert.Contains(t, cap.bodies[0], "settings[payouts][schedule][interval]=manual",
		"manual schedule must be on the wire — without this, FR accounts auto-payout daily")
}

// TestUpdatePayoutSchedule_GuardsEmptyInputs — defensive checks before
// hitting Stripe at all. A bad input would burn a network round-trip
// for nothing and surface an opaque "missing path parameter" error
// from the SDK.
func TestUpdatePayoutSchedule_GuardsEmptyInputs(t *testing.T) {
	svc := NewService("sk_test_unit", "")
	err := svc.UpdatePayoutSchedule(context.Background(), "", "manual")
	assert.Error(t, err, "empty account id must be rejected")

	err = svc.UpdatePayoutSchedule(context.Background(), "acct_123", "")
	assert.Error(t, err, "empty interval must be rejected")
}

// TestCreatePayout_PostsToConnectedAccount — the adapter must hit
// /v1/payouts AND set the Stripe-Account header to the connected
// account id. Without that header the payout would target the
// platform balance instead of the user's connected account, debiting
// the wrong balance.
func TestCreatePayout_PostsToConnectedAccount(t *testing.T) {
	// Spin up our own server so we can capture the Stripe-Account
	// header — stubBackends only records body + path.
	var capturedAccount, capturedPath, capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAccount = r.Header.Get("Stripe-Account")
		capturedPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		capturedBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"po_123","object":"payout"}`))
	}))
	defer srv.Close()

	prevAPI := stripeapi.GetBackend(stripeapi.APIBackend)
	prevKey := stripeapi.Key
	stripeapi.Key = "sk_test_unit"
	stripeapi.SetBackend(stripeapi.APIBackend,
		stripeapi.GetBackendWithConfig(stripeapi.APIBackend, &stripeapi.BackendConfig{
			URL:           stripeapi.String(srv.URL),
			LeveledLogger: &stripeapi.LeveledLogger{Level: stripeapi.LevelNull},
		}))
	defer func() {
		stripeapi.SetBackend(stripeapi.APIBackend, prevAPI)
		stripeapi.Key = prevKey
	}()

	svc := NewService("sk_test_unit", "")
	id, err := svc.CreatePayout(context.Background(), portservice.CreatePayoutInput{
		ConnectedAccountID: "acct_xyz",
		Amount:             1500,
		Currency:           "eur",
		IdempotencyKey:     "payout_idem_1",
		Description:        "Wallet payout",
	})
	require.NoError(t, err)
	assert.Equal(t, "po_123", id)
	assert.Equal(t, "/v1/payouts", capturedPath)
	assert.Equal(t, "acct_xyz", capturedAccount,
		"Stripe-Account header MUST point at the connected account; otherwise the payout debits the wrong balance")
	assert.True(t, strings.Contains(capturedBody, "amount=1500"), "amount on wire: %s", capturedBody)
	assert.True(t, strings.Contains(capturedBody, "currency=eur"), "currency on wire: %s", capturedBody)
	assert.True(t, strings.Contains(capturedBody, "description=Wallet+payout"), "description on wire: %s", capturedBody)
}

// TestCreatePayout_ValidatesInputs — every bad input is rejected
// before hitting Stripe, so we never spend a round-trip + idempotency
// key on a request that can't possibly succeed.
func TestCreatePayout_ValidatesInputs(t *testing.T) {
	svc := NewService("sk_test_unit", "")
	cases := []struct {
		name  string
		input portservice.CreatePayoutInput
	}{
		{"empty connected account", portservice.CreatePayoutInput{Amount: 100, Currency: "eur", IdempotencyKey: "k"}},
		{"zero amount", portservice.CreatePayoutInput{ConnectedAccountID: "acct", Amount: 0, Currency: "eur", IdempotencyKey: "k"}},
		{"negative amount", portservice.CreatePayoutInput{ConnectedAccountID: "acct", Amount: -1, Currency: "eur", IdempotencyKey: "k"}},
		{"empty currency", portservice.CreatePayoutInput{ConnectedAccountID: "acct", Amount: 100, IdempotencyKey: "k"}},
		{"missing idempotency key", portservice.CreatePayoutInput{ConnectedAccountID: "acct", Amount: 100, Currency: "eur"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.CreatePayout(context.Background(), c.input)
			assert.Error(t, err)
		})
	}
}
