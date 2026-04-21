package stripe

import (
	"fmt"
	"testing"
	"time"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the subscription-specific paths added to the
// Stripe webhook adapter in Phase B: subscription state projection,
// lookup_key parsing, and end-to-end event construction from a signed
// payload. No mocking of the Stripe SDK — we use the real signer
// defined in webhook_signature_test.go so a regression in Stripe's
// verification surface fails these tests loudly.

func TestToSubscriptionSnapshot_FullPayload(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour).Unix()
	end := time.Now().Add(30 * 24 * time.Hour).Unix()
	sub := &stripe.Subscription{
		ID:                "sub_abc",
		Status:            stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd: true,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					Price: &stripe.Price{
						ID:        "price_premium",
						LookupKey: "premium_freelance_monthly",
					},
					CurrentPeriodStart: start,
					CurrentPeriodEnd:   end,
				},
			},
		},
	}

	snap := toSubscriptionSnapshot(sub)

	assert.Equal(t, "sub_abc", snap.ID)
	assert.Equal(t, "active", snap.Status)
	assert.Equal(t, "price_premium", snap.PriceID)
	assert.True(t, snap.CancelAtPeriodEnd)
	assert.Equal(t, start, snap.CurrentPeriodStart.Unix())
	assert.Equal(t, end, snap.CurrentPeriodEnd.Unix())
}

func TestToSubscriptionSnapshot_NoItems_FieldsZeroed(t *testing.T) {
	sub := &stripe.Subscription{
		ID:     "sub_empty",
		Status: stripe.SubscriptionStatusIncomplete,
		Items:  &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{}},
	}

	snap := toSubscriptionSnapshot(sub)

	assert.Equal(t, "sub_empty", snap.ID)
	assert.Equal(t, "incomplete", snap.Status)
	assert.Empty(t, snap.PriceID)
	assert.True(t, snap.CurrentPeriodStart.IsZero())
	assert.True(t, snap.CurrentPeriodEnd.IsZero())
}

func TestParsePlanCycleFromSubscription_AllCombinations(t *testing.T) {
	tests := []struct {
		lookupKey string
		wantPlan  string
		wantCycle string
	}{
		{"premium_freelance_monthly", "freelance", "monthly"},
		{"premium_freelance_annual", "freelance", "annual"},
		{"premium_agency_monthly", "agency", "monthly"},
		{"premium_agency_annual", "agency", "annual"},
	}
	for _, tc := range tests {
		t.Run(tc.lookupKey, func(t *testing.T) {
			sub := &stripe.Subscription{
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{Price: &stripe.Price{LookupKey: tc.lookupKey}},
					},
				},
			}
			plan, cycle := parsePlanCycleFromSubscription(sub)
			assert.Equal(t, tc.wantPlan, plan)
			assert.Equal(t, tc.wantCycle, cycle)
		})
	}
}

func TestParsePlanCycleFromSubscription_UnknownPrefix_ReturnsEmpty(t *testing.T) {
	sub := &stripe.Subscription{
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{Price: &stripe.Price{LookupKey: "basic_freelance_monthly"}},
			},
		},
	}
	plan, cycle := parsePlanCycleFromSubscription(sub)
	assert.Empty(t, plan)
	assert.Empty(t, cycle)
}

func TestParsePlanCycleFromSubscription_MissingLookupKey(t *testing.T) {
	sub := &stripe.Subscription{
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{Price: &stripe.Price{}},
			},
		},
	}
	plan, cycle := parsePlanCycleFromSubscription(sub)
	assert.Empty(t, plan)
	assert.Empty(t, cycle)
}

// Real-shape webhook bodies — truncated to the fields the adapter reads,
// still valid JSON. The signatures are computed below with
// signStripePayload (reused from webhook_signature_test.go).

const realSubscriptionCreatedPayload = `{
  "id": "evt_sub_created_1",
  "object": "event",
  "api_version": "2024-11-20.acacia",
  "created": 1777000000,
  "type": "customer.subscription.created",
  "data": {
    "object": {
      "id": "sub_1NXabc",
      "object": "subscription",
      "status": "active",
      "cancel_at_period_end": true,
      "metadata": {"user_id": "550e8400-e29b-41d4-a716-446655440000"},
      "items": {
        "object": "list",
        "data": [{
          "id": "si_1",
          "price": {
            "id": "price_1",
            "object": "price",
            "lookup_key": "premium_freelance_monthly"
          },
          "current_period_start": 1777000000,
          "current_period_end": 1779500000
        }]
      }
    }
  }
}`

const realSubscriptionDeletedPayload = `{
  "id": "evt_sub_deleted_1",
  "object": "event",
  "type": "customer.subscription.deleted",
  "data": {
    "object": {
      "id": "sub_1NXabc",
      "object": "subscription",
      "status": "canceled",
      "items": {"data": [{"price": {"lookup_key": "premium_freelance_monthly"}}]}
    }
  }
}`

const realInvoicePaymentFailedPayload = `{
  "id": "evt_inv_failed_1",
  "object": "event",
  "type": "invoice.payment_failed",
  "data": {
    "object": {
      "id": "in_1",
      "object": "invoice",
      "parent": {
        "type": "subscription_details",
        "subscription_details": {
          "subscription": "sub_1NXabc"
        }
      }
    }
  }
}`

func TestConstructWebhookEvent_SubscriptionCreated(t *testing.T) {
	svc := &Service{webhookSecret: testWebhookSecret}
	now := time.Now().Unix()
	sig := signStripePayload([]byte(realSubscriptionCreatedPayload), testWebhookSecret, now)

	event, err := svc.ConstructWebhookEvent([]byte(realSubscriptionCreatedPayload), sig)

	require.NoError(t, err)
	assert.Equal(t, "customer.subscription.created", event.Type)
	assert.Equal(t, "evt_sub_created_1", event.EventID, "event id MUST be surfaced for idempotency")
	assert.False(t, event.SubscriptionDeleted)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", event.SubscriptionUserID)
	assert.Equal(t, "freelance", event.SubscriptionPlan)
	assert.Equal(t, "monthly", event.SubscriptionCycle)
	require.NotNil(t, event.SubscriptionSnapshot)
	assert.Equal(t, "sub_1NXabc", event.SubscriptionSnapshot.ID)
	assert.Equal(t, "active", event.SubscriptionSnapshot.Status)
	assert.True(t, event.SubscriptionSnapshot.CancelAtPeriodEnd)
}

func TestConstructWebhookEvent_SubscriptionDeleted(t *testing.T) {
	svc := &Service{webhookSecret: testWebhookSecret}
	now := time.Now().Unix()
	sig := signStripePayload([]byte(realSubscriptionDeletedPayload), testWebhookSecret, now)

	event, err := svc.ConstructWebhookEvent([]byte(realSubscriptionDeletedPayload), sig)

	require.NoError(t, err)
	assert.Equal(t, "customer.subscription.deleted", event.Type)
	assert.True(t, event.SubscriptionDeleted, "deleted flag MUST be true on .deleted events")
	require.NotNil(t, event.SubscriptionSnapshot)
	assert.Equal(t, "sub_1NXabc", event.SubscriptionSnapshot.ID)
}

func TestConstructWebhookEvent_InvoicePaymentFailed(t *testing.T) {
	svc := &Service{webhookSecret: testWebhookSecret}
	now := time.Now().Unix()
	sig := signStripePayload([]byte(realInvoicePaymentFailedPayload), testWebhookSecret, now)

	event, err := svc.ConstructWebhookEvent([]byte(realInvoicePaymentFailedPayload), sig)

	require.NoError(t, err)
	assert.Equal(t, "invoice.payment_failed", event.Type)
	assert.Equal(t, "sub_1NXabc", event.InvoiceSubscriptionID)
	assert.True(t, event.InvoicePaymentFailed)
}

func TestConstructWebhookEvent_InvalidSignature_Rejected(t *testing.T) {
	svc := &Service{webhookSecret: testWebhookSecret}

	// Sign with a DIFFERENT secret — the verification MUST fail.
	bogus := signStripePayload([]byte(realSubscriptionCreatedPayload), "whsec_wrong_secret", time.Now().Unix())

	_, err := svc.ConstructWebhookEvent([]byte(realSubscriptionCreatedPayload), bogus)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature", "a wrong-secret replay MUST be rejected")
}

// compile-time guard: keep fmt imported even if no test references it in
// future refactors (avoids churn in test files).
var _ = fmt.Sprintf
