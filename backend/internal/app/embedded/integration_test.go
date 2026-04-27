package embedded

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/organization"
	portservice "marketplace-backend/internal/port/service"
)

// These tests prove the full lifecycle chain from a raw Stripe webhook
// payload all the way to a user-facing notification. They are the
// marketplace-layer counterpart to the webhook_test.go fixtures in
// internal/adapter/stripe — those cover parsing, these cover routing.
//
// Scenario matrix validated here:
//   1. Fresh requirement appears post-activation        → notif dispatched
//   2. Document rejected with verification_document_expired → contextual notif
//   3. Account suspended (charges_enabled true → false) → urgent notif

// ----------------------------------------------------------------------
// Test helpers
// ----------------------------------------------------------------------

type capturedNotif struct {
	userID  uuid.UUID
	typ     notifdomain.NotificationType
	title   string
	body    string
	payload map[string]any
}

type capturingSink struct {
	calls []capturedNotif
}

func (c *capturingSink) Send(_ context.Context, userID uuid.UUID, t notifdomain.NotificationType,
	title, body string, meta map[string]any) error {
	c.calls = append(c.calls, capturedNotif{
		userID:  userID,
		typ:     t,
		title:   title,
		body:    body,
		payload: meta,
	})
	return nil
}

// memoryUserStore is a single in-memory OrgStore combining the
// account lookup + state persistence. Used by integration tests to
// simulate a real org_repository without a DB.
type memoryUserStore struct {
	orgID       uuid.UUID
	ownerUserID uuid.UUID
	state       *LastAccountState
}

func (m *memoryUserStore) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	return &organization.Organization{ID: m.orgID, OwnerUserID: m.ownerUserID}, nil
}

func (m *memoryUserStore) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	if m.state == nil {
		return nil, nil
	}
	return json.Marshal(m.state)
}

func (m *memoryUserStore) SaveStripeLastState(_ context.Context, _ uuid.UUID, raw []byte) error {
	var s LastAccountState
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	m.state = &s
	return nil
}

func setupChain(prev *LastAccountState) (*Notifier, *capturingSink, *memoryUserStore) {
	sink := &capturingSink{}
	store := &memoryUserStore{
		orgID:       uuid.New(),
		ownerUserID: uuid.MustParse("51a9b3e7-1dae-45ee-913a-d73733b20aae"),
		state:       prev,
	}
	n := NewNotifier(sink, store, 5*time.Minute)
	return n, sink, store
}

// ----------------------------------------------------------------------
// Scenario 1: fresh requirement appears post-activation
// ----------------------------------------------------------------------

func TestIntegration_WebhookAddsNewRequirement_EmitsContextualNotif(t *testing.T) {
	// Prior state: fully active account, no pending requirements
	notifier, sink, store := setupChain(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		DetailsSubmitted:        true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})

	// New webhook arrives: Stripe added a requirement (person verification document)
	snap := &portservice.StripeAccountSnapshot{
		AccountID:        "acct_1TIsgNPyy7y81FsB",
		Country:          "FR",
		BusinessType:     "company",
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
		CurrentlyDue:     []string{"person_1NzR.verification.document"},
	}

	err := notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	// Verify: user got a notification about the new requirement
	require.Len(t, sink.calls, 1, "expected exactly one notification (currently_due added)")
	n := sink.calls[0]
	assert.Equal(t, notifdomain.TypeStripeRequirements, n.typ)
	assert.Contains(t, n.title, "1 information requise")
	assert.Equal(t, "acct_1TIsgNPyy7y81FsB", n.payload["account_id"])
	assert.EqualValues(t, 1, n.payload["count"])

	// Verify: state persisted for future diffs
	assert.NotEmpty(t, store.state.CurrentlyDueHash)
}

// ----------------------------------------------------------------------
// Scenario 2: document rejection with specific error code
// ----------------------------------------------------------------------

func TestIntegration_DocumentExpired_EmitsSpecificFrenchMessage(t *testing.T) {
	// Prior state: account active, no known errors. HasEverActivated
	// is set so the requirement-class notifs are not gated by the
	// initial-onboarding suppression.
	notifier, sink, _ := setupChain(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		DetailsSubmitted:        true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})

	// Webhook: Stripe rejects the uploaded identity document
	snap := &portservice.StripeAccountSnapshot{
		AccountID:      "acct_1TIsgNPyy7y81FsB",
		Country:        "FR",
		BusinessType:   "individual",
		ChargesEnabled: true,
		PayoutsEnabled: true,
		CurrentlyDue:   []string{"individual.verification.document"},
		RequirementErrors: []portservice.StripeRequirementError{
			{
				Requirement: "individual.verification.document",
				Code:        "verification_document_expired",
				Reason:      "The document has expired.",
			},
		},
	}

	err := notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	// User should receive 2 notifs: generic "info requise" + specific "Document expiré"
	assert.GreaterOrEqual(t, len(sink.calls), 1)

	var docRejectedNotif *capturedNotif
	for i := range sink.calls {
		if sink.calls[i].title == "Document expiré" {
			docRejectedNotif = &sink.calls[i]
			break
		}
	}
	require.NotNil(t, docRejectedNotif, "expected a 'Document expiré' notification")
	assert.Contains(t, docRejectedNotif.body, "document fourni a expiré")
	assert.Equal(t, "verification_document_expired", docRejectedNotif.payload["code"])
	assert.Equal(t, "individual.verification.document", docRejectedNotif.payload["requirement"])
}

// ----------------------------------------------------------------------
// Scenario 3: account suspended (charges disabled transition)
// ----------------------------------------------------------------------

func TestIntegration_AccountSuspended_EmitsUrgentNotif(t *testing.T) {
	// Prior state: account was fully active
	notifier, sink, store := setupChain(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		DetailsSubmitted:        true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})

	// Webhook: Stripe suspended charges + payouts due to past_due requirements
	snap := &portservice.StripeAccountSnapshot{
		AccountID:        "acct_1TIsgNPyy7y81FsB",
		Country:          "FR",
		BusinessType:     "company",
		ChargesEnabled:   false,
		PayoutsEnabled:   false,
		DetailsSubmitted: true,
		PastDue:          []string{"company.verification.document"},
		DisabledReason:   "requirements.past_due",
	}

	err := notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	// Should have at least: charges_disabled notif + past_due notif + account_disabled notif
	assert.GreaterOrEqual(t, len(sink.calls), 2)

	// Expect a charges-disabled notification
	foundChargesDisabled := false
	foundPastDue := false
	for _, n := range sink.calls {
		if n.title == "Paiements entrants suspendus" {
			foundChargesDisabled = true
			assert.Equal(t, "charges_disabled", n.payload["status"])
		}
		if n.title == "Action urgente — délai dépassé" {
			foundPastDue = true
			assert.Equal(t, true, n.payload["urgent"])
		}
	}
	assert.True(t, foundChargesDisabled, "expected 'Paiements entrants suspendus' notif")
	assert.True(t, foundPastDue, "expected past_due urgent notif")

	// State persisted with new disabled reason
	assert.False(t, store.state.ChargesEnabled)
	assert.Equal(t, "requirements.past_due", store.state.DisabledReason)
}

// ----------------------------------------------------------------------
// Bonus scenario: eventually_due informs user without blocking
// ----------------------------------------------------------------------

func TestIntegration_EventuallyDueAdded_EmitsNonUrgentNotif(t *testing.T) {
	notifier, sink, _ := setupChain(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		DetailsSubmitted:        true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})

	snap := &portservice.StripeAccountSnapshot{
		AccountID:        "acct_new",
		Country:          "US",
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
		EventuallyDue:    []string{"individual.verification.additional_document"},
	}

	err := notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(sink.calls), 1)
	foundEventuallyDue := false
	for _, n := range sink.calls {
		if n.payload["count"] != nil && n.payload["urgent"] == false {
			foundEventuallyDue = true
			assert.Contains(t, n.title, "à prévoir")
		}
	}
	assert.True(t, foundEventuallyDue, "expected non-urgent eventually_due notif")
}

// ----------------------------------------------------------------------
// Bonus scenario: multiple transitions in single webhook → multiple notifs
// ----------------------------------------------------------------------

func TestIntegration_MultipleSimultaneousChanges_AllDispatched(t *testing.T) {
	// Prior state: account active with no issues
	notifier, sink, _ := setupChain(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		DetailsSubmitted:        true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})

	// Webhook: charges suspended AND past_due added AND doc rejected → user gets full picture
	snap := &portservice.StripeAccountSnapshot{
		AccountID:      "acct_big_update",
		Country:        "FR",
		ChargesEnabled: false,
		PayoutsEnabled: false,
		PastDue:        []string{"company.verification.document"},
		DisabledReason: "requirements.past_due",
		RequirementErrors: []portservice.StripeRequirementError{
			{
				Requirement: "company.verification.document",
				Code:        "verification_document_fraudulent",
				Reason:      "Document appears altered.",
			},
		},
	}

	err := notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	// Should emit: charges_disabled + past_due + doc_rejected + account_disabled
	assert.GreaterOrEqual(t, len(sink.calls), 3,
		"expected at least 3 notifications for multi-change webhook")

	titles := make([]string, 0, len(sink.calls))
	for _, n := range sink.calls {
		titles = append(titles, n.title)
	}
	t.Logf("notifications emitted: %v", titles)
}

// ----------------------------------------------------------------------
// Regression: identical webhooks don't spam the user
// ----------------------------------------------------------------------

func TestIntegration_IdenticalWebhookTwice_NoDuplicateNotifs(t *testing.T) {
	notifier, sink, store := setupChain(nil)

	snap := &portservice.StripeAccountSnapshot{
		AccountID:      "acct_idempotent",
		Country:        "FR",
		ChargesEnabled: true,
		PayoutsEnabled: true,
	}

	// First call
	err := notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	firstCount := len(sink.calls)
	assert.GreaterOrEqual(t, firstCount, 1)

	// Second call — Stripe retried the webhook with identical payload
	// (state is persisted between calls via the memory store)
	_ = store // state is captured internally

	err = notifier.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	// No new notifs: state was already persisted, diff returns empty
	assert.Equal(t, firstCount, len(sink.calls),
		"identical snapshot should not produce new notifs")
}
