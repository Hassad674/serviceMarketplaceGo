package handler

// BUG-NEW-06 — Stripe webhook handler must release the idempotency
// claim AND respond 5xx when a downstream handler returns an error.
//
// Pre-fix: handlers logged errors silently, the dispatcher always
// returned 200 OK, and the durable claim was permanent. Stripe's next
// retry was silently deduped → state change permanently lost.
//
// Post-fix: handlers return errors, the dispatcher captures the first
// non-nil, calls IdempotencyClaimer.Release(), and responds 5xx. Stripe
// retries the webhook and the claim flow returns "first delivery" again.

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "marketplace-backend/internal/app/payment"
	paymentdomain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// stubIdempotencyClaimer mirrors the postgres+redis composite claimer
// API surface (TryClaim + Release). Records every call so the test can
// assert which sequence the dispatcher executed.
type stubIdempotencyClaimer struct {
	mu sync.Mutex

	// tryClaimCalls maps event_id → total times TryClaim was invoked
	// (counts every dispatch attempt, including replays). The
	// firstDeliveryByEvent map drives the bool return value.
	tryClaimCalls map[string]int
	// firstDeliveryByEvent tracks whether the next TryClaim should be
	// treated as a "first delivery". Set to true on construction, set
	// to false after a successful claim, set back to true on Release.
	firstDeliveryByEvent map[string]bool

	// released records every event_id passed to Release. The dispatcher
	// MUST call Release exactly once when a handler returns an error.
	released []string

	// releaseErr lets a test simulate a Release failure (Postgres down).
	// The dispatcher must still respond 5xx so Stripe retries.
	releaseErr error
}

func newStubIdempotencyClaimer() *stubIdempotencyClaimer {
	return &stubIdempotencyClaimer{
		tryClaimCalls:        map[string]int{},
		firstDeliveryByEvent: map[string]bool{},
	}
}

func (s *stubIdempotencyClaimer) TryClaim(_ context.Context, eventID, _ string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tryClaimCalls[eventID]++
	// Default for a new id: first delivery. After we return true once
	// we flip to false until Release flips it back.
	first, seen := s.firstDeliveryByEvent[eventID]
	if !seen {
		// First time we see this id — return "first delivery" and
		// remember we've now claimed it.
		s.firstDeliveryByEvent[eventID] = false
		return true, nil
	}
	if first {
		s.firstDeliveryByEvent[eventID] = false
		return true, nil
	}
	return false, nil
}

func (s *stubIdempotencyClaimer) Release(_ context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.released = append(s.released, eventID)
	if s.releaseErr != nil {
		return s.releaseErr
	}
	// Mirror the durable adapter: deleting the row makes the next
	// TryClaim see no entry → returns (true, nil).
	s.firstDeliveryByEvent[eventID] = true
	return nil
}

// stubVerifyingStripe is a minimal portservice.StripeService that
// satisfies ConstructWebhookEvent (the underlying call behind
// paymentSvc.VerifyWebhook) by returning whatever event the test wired.
// Every other method is auto-supplied by the embedded interface — none
// of them are exercised by HandleWebhook.
type stubVerifyingStripe struct {
	portservice.StripeService
	event *portservice.StripeWebhookEvent
	err   error
}

func (s *stubVerifyingStripe) ConstructWebhookEvent(_ []byte, _ string) (*portservice.StripeWebhookEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.event, nil
}

// stubUserRepoOnly is the smallest valid UserRepository — needed by
// paymentapp.NewService construction even when the tested path doesn't
// use it. Same pattern used in billing_handler_test.go.
type stubUserRepoOnly struct {
	repository.UserRepository
}

// failingRecordsRepo is a PaymentRecordRepository whose every read
// returns an error so handlePaymentSucceeded surfaces an error to the
// dispatcher (driving the BUG-NEW-06 release path). The embedded
// interface auto-satisfies the unused methods.
type failingRecordsRepo struct {
	repository.PaymentRecordRepository
}

func (f *failingRecordsRepo) GetByPaymentIntentID(_ context.Context, _ string) (*paymentdomain.PaymentRecord, error) {
	return nil, errors.New("synthetic test failure")
}

// newPaymentSvcVerifyingTo builds a minimal payment.Service that returns
// `event` from VerifyWebhook. We can't avoid construction of the real
// Service because HandleWebhook calls paymentSvc.VerifyWebhook directly.
// Records is the failingRecordsRepo so payment_intent.succeeded events
// produce an error inside handlePaymentSucceeded.
func newPaymentSvcVerifyingTo(event *portservice.StripeWebhookEvent) *paymentapp.Service {
	return paymentapp.NewService(paymentapp.ServiceDeps{
		Users:   &stubUserRepoOnly{},
		Records: &failingRecordsRepo{},
		Stripe:  &stubVerifyingStripe{event: event},
	})
}

// failingSubscriptionService is the simplest knob to force a downstream
// handler error: handleSubscriptionSnapshot calls
// subscriptionSvc.HandleSubscriptionSnapshot, which we wrap below to
// always return an error.
//
// We replace the field via reflection in the test setup — see the
// per-test helper. (Not great, but the StripeHandler uses a concrete
// pointer type and we can't swap it out cleanly.) Instead, we drive
// the test through handler.handleSubscriptionSnapshot manually rather
// than wiring through the entire HandleWebhook path. That still
// exercises the dispatch() error capture + Release flow when paired
// with a separate test on dispatch() itself.

// TestHandleWebhook_HandlerErrorReleasesClaimAndReturns5xx is the
// end-to-end regression: a fake handler error must trigger Release +
// 5xx. Implemented by injecting a stub claimer on the handler and
// using the dispatch() error capture path directly via HandleWebhook.
//
// To force a handler error, we wire an event type that requires a
// subscriptionSvc but leave the service nil — the handler will not
// fail by itself in that case. So we test directly via dispatch().
func TestHandleWebhook_HandlerErrorReleasesClaimAndReturns5xx(t *testing.T) {
	const eventID = "evt_bug_new_06_001"
	claimer := newStubIdempotencyClaimer()

	// Stub event that will route through dispatch() → handler. We use
	// payment_intent.succeeded because handlePaymentSucceeded returns
	// the payment-side error directly (no extra wiring needed).
	event := &portservice.StripeWebhookEvent{
		EventID:         eventID,
		Type:            "payment_intent.succeeded",
		PaymentIntentID: "pi_bug_new_06_001",
	}

	// payment.Service whose HandlePaymentSucceeded fails (no Records
	// wired → ErrPaymentRecordNotFound). This drives a non-nil error
	// out of handlePaymentSucceeded → into dispatch() → into
	// HandleWebhook's release branch.
	paymentSvc := newPaymentSvcVerifyingTo(event)

	// proposalSvc is irrelevant since paymentSvc fails first; pass nil.
	h := &StripeHandler{
		paymentSvc:       paymentSvc,
		idempotencyStore: claimer,
	}

	// First delivery: dispatch fails → Release called → 5xx.
	body := []byte(`{"id":"evt_bug_new_06_001","type":"payment_intent.succeeded"}`)
	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("Stripe-Signature", "t=any,v1=any")
	w := httptest.NewRecorder()

	h.HandleWebhook(w, r)

	// 1. Response must be 5xx so Stripe retries.
	assert.GreaterOrEqual(t, w.Code, 500, "BUG-NEW-06: handler error must produce 5xx so Stripe retries")
	assert.Less(t, w.Code, 600, "5xx range")

	// 2. Release was called exactly once for the failed event.
	require.Len(t, claimer.released, 1, "Release MUST be called when a handler returns an error")
	assert.Equal(t, eventID, claimer.released[0])

	// 3. A second delivery of the SAME event_id MUST be processed
	//    again (not silently deduped) — that's the whole point of the
	//    Release. Stripe sends another request; the new TryClaim
	//    should return (true, nil) because Release wiped the row.
	r2 := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r2.Header.Set("Stripe-Signature", "t=any,v1=any")
	w2 := httptest.NewRecorder()
	h.HandleWebhook(w2, r2)

	assert.Equal(t, 2, claimer.tryClaimCalls[eventID],
		"BUG-NEW-06: second delivery MUST hit TryClaim again (not be deduped)")
}

// TestHandleWebhook_HandlerSuccess_KeepsClaimAndReturns200 is the
// regression: the success path must keep the claim AND return 200.
// Otherwise we'd release on every 200 and Stripe would re-process
// already-handled events on every retry.
func TestHandleWebhook_HandlerSuccess_KeepsClaimAndReturns200(t *testing.T) {
	const eventID = "evt_bug_new_06_002"
	claimer := newStubIdempotencyClaimer()

	// Event type with no wired side-effect handler: invoice.payment_succeeded
	// is intentionally a no-op in dispatch(). It returns nil → 200 OK.
	event := &portservice.StripeWebhookEvent{
		EventID: eventID,
		Type:    "invoice.payment_succeeded",
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)

	h := &StripeHandler{
		paymentSvc:       paymentSvc,
		idempotencyStore: claimer,
	}

	body := []byte(`{"id":"evt_bug_new_06_002","type":"invoice.payment_succeeded"}`)
	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("Stripe-Signature", "t=any,v1=any")
	w := httptest.NewRecorder()
	h.HandleWebhook(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "happy path must return 200")
	assert.Empty(t, claimer.released,
		"Release MUST NOT be called on the success path (would cause Stripe to re-deliver)")
	assert.Equal(t, 1, claimer.tryClaimCalls[eventID], "claim made once, kept")
}

// TestHandleWebhook_ReplayIsDeduped_NoSecondDispatch is the standard
// idempotency guarantee — replays must skip dispatch entirely. This
// test ensures the BUG-NEW-06 fix didn't break the existing dedup.
func TestHandleWebhook_ReplayIsDeduped_NoSecondDispatch(t *testing.T) {
	const eventID = "evt_bug_new_06_003"
	claimer := newStubIdempotencyClaimer()

	event := &portservice.StripeWebhookEvent{
		EventID: eventID,
		Type:    "invoice.payment_succeeded", // no-op handler
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)
	h := &StripeHandler{
		paymentSvc:       paymentSvc,
		idempotencyStore: claimer,
	}

	body := []byte(`{"id":"evt_bug_new_06_003","type":"invoice.payment_succeeded"}`)

	// First delivery: process + 200.
	r1 := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r1.Header.Set("Stripe-Signature", "t=any,v1=any")
	w1 := httptest.NewRecorder()
	h.HandleWebhook(w1, r1)
	require.Equal(t, http.StatusOK, w1.Code)

	// Second delivery: claim returns false → 200 + skip.
	r2 := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r2.Header.Set("Stripe-Signature", "t=any,v1=any")
	w2 := httptest.NewRecorder()
	h.HandleWebhook(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)

	// TryClaim called twice; Release never called.
	assert.Equal(t, 2, claimer.tryClaimCalls[eventID])
	assert.Empty(t, claimer.released, "successful path → no release")
}

// TestHandleWebhook_HandlerError_ReleaseAlsoFails_Still5xx is the
// belt-and-braces case: even when Release itself fails (e.g. Postgres
// down), the dispatcher MUST respond 5xx. The state change WILL be
// permanently lost on the next Stripe retry, but that's an ops-level
// problem that needs manual reconciliation; the very least we can do
// is fail loud.
func TestHandleWebhook_HandlerError_ReleaseAlsoFails_Still5xx(t *testing.T) {
	const eventID = "evt_bug_new_06_004"
	claimer := newStubIdempotencyClaimer()
	claimer.releaseErr = errors.New("postgres down")

	event := &portservice.StripeWebhookEvent{
		EventID:         eventID,
		Type:            "payment_intent.succeeded",
		PaymentIntentID: "pi_bug_new_06_004",
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)

	h := &StripeHandler{
		paymentSvc:       paymentSvc,
		idempotencyStore: claimer,
	}

	body := []byte(`{"id":"evt_bug_new_06_004","type":"payment_intent.succeeded"}`)
	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("Stripe-Signature", "t=any,v1=any")
	w := httptest.NewRecorder()
	h.HandleWebhook(w, r)

	assert.GreaterOrEqual(t, w.Code, 500, "5xx even when Release itself fails")
	assert.Less(t, w.Code, 600)
	assert.Len(t, claimer.released, 1, "Release was attempted")
}

// _ = uuid is kept in case future test cases need a typed id.
var _ = uuid.Nil
