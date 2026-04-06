package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests validate the REAL Stripe webhook signature verification — no
// mocks of the signing algorithm. They exercise the full chain from raw
// HTTP body + signature header to parsed AccountSnapshot.
//
// Goal: prove we can trust what Stripe sends us in production. A bad
// signature must be rejected; a good signature must be accepted.

// stripeSigningSecret is what Stripe would give you in the dashboard under
// "Webhook signing secrets". Always starts with "whsec_" in real life.
const testWebhookSecret = "whsec_test_signing_key_for_unit_tests_only_never_prod_12345"

// signStripePayload computes the signature header Stripe would include in
// the `Stripe-Signature` header. Format:
//
//	t=<timestamp>,v1=<hex-encoded HMAC-SHA256 of "<timestamp>.<payload>">
//
// Reference: https://docs.stripe.com/webhooks#verify-manually
func signStripePayload(payload []byte, secret string, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.%s", timestamp, string(payload))))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, sig)
}

// realAccountUpdatedPayload mimics what Stripe actually sends for an
// account.updated event (shape taken from a real test-mode webhook).
const realAccountUpdatedPayload = `{
  "id": "evt_1TIsgNPyy7y81FsB",
  "object": "event",
  "api_version": "2024-11-20.acacia",
  "created": 1775419800,
  "data": {
    "object": {
      "id": "acct_1TIsgNPyy7y81FsB",
      "object": "account",
      "country": "FR",
      "business_type": "company",
      "charges_enabled": true,
      "payouts_enabled": true,
      "details_submitted": true,
      "requirements": {
        "currently_due": ["person_1NzR.verification.document"],
        "eventually_due": ["person_1NzR.verification.document"],
        "past_due": [],
        "pending_verification": [],
        "disabled_reason": null,
        "errors": [
          {
            "requirement": "person_1NzR.verification.document",
            "code": "verification_document_expired",
            "reason": "The document has expired. Please upload a current one."
          }
        ]
      }
    }
  },
  "livemode": false,
  "type": "account.updated"
}`

// ----------------------------------------------------------------------
// Happy path: valid signature → event parsed correctly
// ----------------------------------------------------------------------

func TestConstructWebhookEvent_ValidSignature_ParsesAccountUpdated(t *testing.T) {
	svc := NewService("sk_test_not_used_here", testWebhookSecret)

	payload := []byte(realAccountUpdatedPayload)
	sig := signStripePayload(payload, testWebhookSecret, time.Now().Unix())

	event, err := svc.ConstructWebhookEvent(payload, sig)
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, "account.updated", event.Type)
	assert.Equal(t, "acct_1TIsgNPyy7y81FsB", event.AccountID)

	// AccountSnapshot should be populated with all the requirements details
	require.NotNil(t, event.AccountSnapshot)
	assert.Equal(t, "FR", event.AccountSnapshot.Country)
	assert.Equal(t, "company", event.AccountSnapshot.BusinessType)
	assert.True(t, event.AccountSnapshot.ChargesEnabled)
	assert.True(t, event.AccountSnapshot.PayoutsEnabled)
	assert.Len(t, event.AccountSnapshot.CurrentlyDue, 1)
	assert.Contains(t, event.AccountSnapshot.CurrentlyDue[0], "verification.document")
	require.Len(t, event.AccountSnapshot.RequirementErrors, 1)
	assert.Equal(t, "verification_document_expired", event.AccountSnapshot.RequirementErrors[0].Code)
}

// ----------------------------------------------------------------------
// Attack scenarios: bad signature, tampered payload, replay
// ----------------------------------------------------------------------

func TestConstructWebhookEvent_WrongSecret_Rejected(t *testing.T) {
	svc := NewService("sk_test", "whsec_correct_secret")

	payload := []byte(realAccountUpdatedPayload)
	// Attacker signs with their own secret
	sig := signStripePayload(payload, "whsec_attacker_fake", time.Now().Unix())

	event, err := svc.ConstructWebhookEvent(payload, sig)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestConstructWebhookEvent_TamperedPayload_Rejected(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)

	originalPayload := []byte(realAccountUpdatedPayload)
	sig := signStripePayload(originalPayload, testWebhookSecret, time.Now().Unix())

	// Attacker changes payload AFTER it was signed
	tamperedPayload := []byte(`{"id":"evt_attacker","type":"account.updated","data":{"object":{"id":"acct_attacker"}}}`)

	event, err := svc.ConstructWebhookEvent(tamperedPayload, sig)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestConstructWebhookEvent_MissingSignatureHeader_Rejected(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)
	payload := []byte(realAccountUpdatedPayload)

	event, err := svc.ConstructWebhookEvent(payload, "")
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestConstructWebhookEvent_MalformedSignatureHeader_Rejected(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)
	payload := []byte(realAccountUpdatedPayload)

	event, err := svc.ConstructWebhookEvent(payload, "garbage-signature-not-t-or-v1")
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestConstructWebhookEvent_VeryOldTimestamp_Rejected(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)
	payload := []byte(realAccountUpdatedPayload)

	// Signature is valid but timestamp is from a day ago — Stripe SDK
	// rejects events older than its tolerance window (default 5 minutes)
	// to prevent replay attacks.
	oneDayAgo := time.Now().Add(-24 * time.Hour).Unix()
	sig := signStripePayload(payload, testWebhookSecret, oneDayAgo)

	event, err := svc.ConstructWebhookEvent(payload, sig)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestConstructWebhookEvent_FutureTimestamp_StillAccepted(t *testing.T) {
	// Timestamp slightly in the future (clock skew) should be accepted.
	svc := NewService("sk_test", testWebhookSecret)
	payload := []byte(realAccountUpdatedPayload)

	futureTimestamp := time.Now().Add(30 * time.Second).Unix()
	sig := signStripePayload(payload, testWebhookSecret, futureTimestamp)

	event, err := svc.ConstructWebhookEvent(payload, sig)
	require.NoError(t, err)
	assert.NotNil(t, event)
}

// ----------------------------------------------------------------------
// Edge: empty payload
// ----------------------------------------------------------------------

func TestConstructWebhookEvent_EmptyPayload_Rejected(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)
	sig := signStripePayload([]byte(""), testWebhookSecret, time.Now().Unix())

	event, err := svc.ConstructWebhookEvent([]byte(""), sig)
	assert.Error(t, err)
	assert.Nil(t, event)
}

// ----------------------------------------------------------------------
// Happy path: other event types also parse correctly
// ----------------------------------------------------------------------

func TestConstructWebhookEvent_PaymentIntentSucceeded_Parsed(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)

	payload := []byte(`{
		"id": "evt_pi",
		"object": "event",
		"type": "payment_intent.succeeded",
		"data": {
			"object": {
				"id": "pi_1abc",
				"object": "payment_intent",
				"amount": 10000,
				"currency": "eur"
			}
		}
	}`)
	sig := signStripePayload(payload, testWebhookSecret, time.Now().Unix())

	event, err := svc.ConstructWebhookEvent(payload, sig)
	require.NoError(t, err)
	assert.Equal(t, "payment_intent.succeeded", event.Type)
	assert.Equal(t, "pi_1abc", event.PaymentIntentID)
	// AccountSnapshot NOT populated for payment_intent events
	assert.Nil(t, event.AccountSnapshot)
}

func TestConstructWebhookEvent_CapabilityUpdated_ParsesAccountContext(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)

	payload := []byte(`{
		"id": "evt_cap",
		"object": "event",
		"type": "capability.updated",
		"data": {
			"object": {
				"id": "acct_cap_test",
				"object": "account",
				"country": "FR",
				"charges_enabled": false,
				"payouts_enabled": true,
				"requirements": {
					"currently_due": [],
					"eventually_due": [],
					"past_due": [],
					"pending_verification": [],
					"disabled_reason": "requirements.past_due",
					"errors": []
				}
			}
		}
	}`)
	sig := signStripePayload(payload, testWebhookSecret, time.Now().Unix())

	event, err := svc.ConstructWebhookEvent(payload, sig)
	require.NoError(t, err)
	assert.Equal(t, "capability.updated", event.Type)
	require.NotNil(t, event.AccountSnapshot)
	assert.False(t, event.AccountSnapshot.ChargesEnabled)
	assert.Equal(t, "requirements.past_due", event.AccountSnapshot.DisabledReason)
}

// ----------------------------------------------------------------------
// Robustness: malformed Stripe data
// ----------------------------------------------------------------------

func TestConstructWebhookEvent_InvalidJSON_Rejected(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)

	payload := []byte(`{not valid json`)
	sig := signStripePayload(payload, testWebhookSecret, time.Now().Unix())

	event, err := svc.ConstructWebhookEvent(payload, sig)
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestConstructWebhookEvent_UnknownEventType_ReturnsTypeOnly(t *testing.T) {
	svc := NewService("sk_test", testWebhookSecret)

	// Unknown event types should still parse successfully — we just don't
	// populate the typed fields. The handler can log+skip them.
	payload := []byte(`{
		"id": "evt_unk",
		"object": "event",
		"type": "invoice.created",
		"data": {"object": {"id": "in_1"}}
	}`)
	sig := signStripePayload(payload, testWebhookSecret, time.Now().Unix())

	event, err := svc.ConstructWebhookEvent(payload, sig)
	require.NoError(t, err)
	assert.Equal(t, "invoice.created", event.Type)
	assert.Empty(t, event.PaymentIntentID)
	assert.Empty(t, event.AccountID)
}
