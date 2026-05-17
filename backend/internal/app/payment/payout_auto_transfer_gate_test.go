package payment

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Volet 3 — money-categorisation regression
// (fix/wallet-kyc-billing-regression)
//
// Business rule (validated by the product owner):
//
//   mission validée + KYC OK  + billing OK  → Transféré (auto-transfer)
//   mission validée + KYC KO  (any billing) → Disponible (NO transfer)
//   mission validée + KYC OK  + billing KO  → Disponible (NO transfer)
//   mission validée + KYC OK  + billing err → Disponible (conservative)
//
// ProviderReadyForAutoTransfer is the single gate the proposal
// milestone-approval path consults. When it returns false the
// milestone is NOT auto-transferred — the payment record stays
// Succeeded+TransferPending which the wallet projection classifies
// as AvailableAmount (NOT re-escrowed, NOT transferred). Escrow only
// covers the pre-validation period (covered by classifyRecordBucket
// tests in wallet_escrow_split_test.go).
// ---------------------------------------------------------------------------

// stubBillingGate is a configurable service.BillingProfileGate.
type stubBillingGate struct {
	mu       sync.Mutex
	complete bool
	err      error
	calls    int
}

func (g *stubBillingGate) IsBillingProfileComplete(_ context.Context, _ uuid.UUID) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if g.err != nil {
		return false, g.err
	}
	return g.complete, nil
}

var _ service.BillingProfileGate = (*stubBillingGate)(nil)

// kycStripeStub returns a payouts-enabled (or not) Stripe account.
type kycStripeStub struct {
	service.StripeService
	payoutsEnabled bool
	getAccountErr  error
}

func (s *kycStripeStub) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	if s.getAccountErr != nil {
		return nil, s.getAccountErr
	}
	return &service.StripeAccountInfo{
		ChargesEnabled: true,
		PayoutsEnabled: s.payoutsEnabled,
	}, nil
}

func newAutoTransferGateService(
	stripeAccountID string,
	payoutsEnabled bool,
	gate service.BillingProfileGate,
	getAccountErr error,
) *PayoutService {
	orgs := &payoutStubOrgs{stripeAccountID: stripeAccountID}
	stripe := &kycStripeStub{payoutsEnabled: payoutsEnabled, getAccountErr: getAccountErr}
	p := NewPayoutService(PayoutServiceDeps{
		Records:       &payoutStubRecords{},
		Organizations: orgs,
		Stripe:        stripe,
	})
	if gate != nil {
		p.SetBillingProfileGate(gate)
	}
	return p
}

// TestProviderReadyForAutoTransfer_Matrix walks the full
// {KYC} × {billing} matrix and asserts the auto-transfer decision.
func TestProviderReadyForAutoTransfer_Matrix(t *testing.T) {
	tests := []struct {
		name            string
		stripeAccountID string
		payoutsEnabled  bool
		gate            *stubBillingGate
		getAccountErr   error
		wantReady       bool
		wantErr         bool
	}{
		{
			name:            "KYC ok + billing ok → ready (auto-transfer / Transféré)",
			stripeAccountID: "acct_ok",
			payoutsEnabled:  true,
			gate:            &stubBillingGate{complete: true},
			wantReady:       true,
		},
		{
			name:            "KYC ok + billing INCOMPLETE → NOT ready (funds stay Disponible)",
			stripeAccountID: "acct_ok",
			payoutsEnabled:  true,
			gate:            &stubBillingGate{complete: false},
			wantReady:       false,
		},
		{
			name:            "KYC KO (payouts disabled) + billing ok → NOT ready (Disponible)",
			stripeAccountID: "acct_ok",
			payoutsEnabled:  false,
			gate:            &stubBillingGate{complete: true},
			wantReady:       false,
		},
		{
			name:            "KYC KO + billing KO → NOT ready (Disponible)",
			stripeAccountID: "acct_ok",
			payoutsEnabled:  false,
			gate:            &stubBillingGate{complete: false},
			wantReady:       false,
		},
		{
			name:            "no Stripe account at all → NOT ready (Disponible)",
			stripeAccountID: "",
			payoutsEnabled:  true,
			gate:            &stubBillingGate{complete: true},
			wantReady:       false,
		},
		{
			name:            "KYC ok + billing gate ERRORS → NOT ready (conservative: Disponible, no err surfaced)",
			stripeAccountID: "acct_ok",
			payoutsEnabled:  true,
			gate:            &stubBillingGate{err: errors.New("db blip")},
			wantReady:       false,
			wantErr:         false,
		},
		{
			name:            "KYC probe ERRORS → NOT ready, error surfaced (caller defaults to manual)",
			stripeAccountID: "acct_ok",
			payoutsEnabled:  true,
			gate:            &stubBillingGate{complete: true},
			getAccountErr:   errors.New("stripe down"),
			wantReady:       false,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAutoTransferGateService(
				tt.stripeAccountID, tt.payoutsEnabled, tt.gate, tt.getAccountErr,
			)
			ready, err := p.ProviderReadyForAutoTransfer(context.Background(), uuid.New())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantReady, ready)
		})
	}
}

// TestProviderReadyForAutoTransfer_NilGate_DegradesToKYCOnly pins the
// "invoicing feature disabled" contract: with no billing gate wired,
// readiness is the pre-fix KYC-only behaviour so the payment feature
// stays bootable without invoicing.
func TestProviderReadyForAutoTransfer_NilGate_DegradesToKYCOnly(t *testing.T) {
	t.Run("KYC ok, no gate → ready", func(t *testing.T) {
		p := newAutoTransferGateService("acct_ok", true, nil, nil)
		ready, err := p.ProviderReadyForAutoTransfer(context.Background(), uuid.New())
		assert.NoError(t, err)
		assert.True(t, ready)
	})
	t.Run("KYC ko, no gate → NOT ready", func(t *testing.T) {
		p := newAutoTransferGateService("acct_ok", false, nil, nil)
		ready, err := p.ProviderReadyForAutoTransfer(context.Background(), uuid.New())
		assert.NoError(t, err)
		assert.False(t, ready)
	})
}

// TestProviderReadyForAutoTransfer_BillingGateNotCalledWhenKYCFails
// proves we short-circuit on the KYC gate — no point hitting the
// billing read when the provider cannot receive payouts at all.
func TestProviderReadyForAutoTransfer_BillingGateNotCalledWhenKYCFails(t *testing.T) {
	gate := &stubBillingGate{complete: true}
	p := newAutoTransferGateService("acct_ok", false, gate, nil)
	ready, err := p.ProviderReadyForAutoTransfer(context.Background(), uuid.New())
	assert.NoError(t, err)
	assert.False(t, ready)
	assert.Equal(t, 0, gate.calls, "billing gate must not be consulted once KYC fails")
}

// TestProviderReadyForAutoTransfer_FacadeDelegates threads the parent
// Service facade end-to-end so the wiring point used by main.go is
// covered.
func TestProviderReadyForAutoTransfer_FacadeDelegates(t *testing.T) {
	orgs := &payoutStubOrgs{stripeAccountID: "acct_ok"}
	stripe := &kycStripeStub{payoutsEnabled: true}
	payout := NewPayoutService(PayoutServiceDeps{
		Records:       &payoutStubRecords{},
		Organizations: orgs,
		Stripe:        stripe,
	})
	gate := &stubBillingGate{complete: false}
	svc := &Service{payout: payout}
	svc.SetBillingProfileGate(gate)

	ready, err := svc.ProviderReadyForAutoTransfer(context.Background(), uuid.New())
	assert.NoError(t, err)
	assert.False(t, ready, "facade must surface the billing-incomplete defer")
	assert.Equal(t, 1, gate.calls)
}
