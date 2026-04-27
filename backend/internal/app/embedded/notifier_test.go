package embedded

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/organization"
	portservice "marketplace-backend/internal/port/service"
)

/* ----------------------------- Fakes ----------------------------- */

type fakeSink struct {
	calls []sinkCall
	err   error
}

type sinkCall struct {
	userID uuid.UUID
	typ    notifdomain.NotificationType
	title  string
	body   string
	meta   map[string]any
}

func (f *fakeSink) Send(_ context.Context, userID uuid.UUID, t notifdomain.NotificationType, title, body string, metadata map[string]any) error {
	f.calls = append(f.calls, sinkCall{userID: userID, typ: t, title: title, body: body, meta: metadata})
	return f.err
}

// fakeUserStore implements OrgStore (named *UserStore for backward
// compatibility with existing test fixtures). Since phase R5 the
// notifier is keyed on the organization, not the user — this fake
// returns a stub Organization that carries both the org id used for
// state storage and the owner id used as the notification recipient.
type fakeUserStore struct {
	orgID        uuid.UUID
	ownerUserID  uuid.UUID
	lookupErr    error
	prev         *LastAccountState
	saved        *LastAccountState
}

func (f *fakeUserStore) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	return &organization.Organization{
		ID:          f.orgID,
		OwnerUserID: f.ownerUserID,
	}, nil
}

func (f *fakeUserStore) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	if f.prev == nil {
		return nil, nil
	}
	return json.Marshal(f.prev)
}

func (f *fakeUserStore) SaveStripeLastState(_ context.Context, _ uuid.UUID, state []byte) error {
	var s LastAccountState
	if err := json.Unmarshal(state, &s); err != nil {
		return err
	}
	f.saved = &s
	return nil
}

/* ----------------------------- Helpers ----------------------------- */

func snapshot(accountID string) *portservice.StripeAccountSnapshot {
	return &portservice.StripeAccountSnapshot{
		AccountID: accountID,
	}
}

func newTestNotifier(prev *LastAccountState) (*Notifier, *fakeSink, *fakeUserStore) {
	sink := &fakeSink{}
	store := &fakeUserStore{
		orgID:       uuid.New(),
		ownerUserID: uuid.New(),
		prev:        prev,
	}
	n := NewNotifier(
		sink,
		store,
		100*time.Millisecond, // short cooldown for tests
	)
	return n, sink, store
}

/* ----------------------------- Tests ----------------------------- */

func TestNotifier_NilSnapshot_ReturnsError(t *testing.T) {
	n, _, _ := newTestNotifier(nil)
	err := n.HandleAccountSnapshot(context.Background(), nil)
	assert.Error(t, err)
}

func TestNotifier_EmptyAccountID_ReturnsError(t *testing.T) {
	n, _, _ := newTestNotifier(nil)
	err := n.HandleAccountSnapshot(context.Background(), &portservice.StripeAccountSnapshot{})
	assert.Error(t, err)
}

func TestNotifier_LookupFails_ReturnsError(t *testing.T) {
	sink := &fakeSink{}
	store := &fakeUserStore{lookupErr: errors.New("not found")}
	n := NewNotifier(sink, store, time.Minute)
	err := n.HandleAccountSnapshot(context.Background(), snapshot("acct_1"))
	assert.Error(t, err)
	assert.Empty(t, sink.calls)
}

func TestNotifier_AccountActivated_SendsActivationNotif(t *testing.T) {
	n, sink, store := newTestNotifier(&LastAccountState{
		ChargesEnabled: false,
		PayoutsEnabled: false,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	// Activation transition emits two notifications: the existing
	// "Compte de paiement activé" + the new "Virements sortants activés"
	// one-shot for the first PayoutsEnabled=true transition.
	require.Len(t, sink.calls, 2)
	assert.Equal(t, notifdomain.TypeStripeAccountStatus, sink.calls[0].typ)
	assert.Contains(t, sink.calls[0].title, "activé")
	assert.NotNil(t, store.saved)
	assert.True(t, store.saved.ChargesEnabled)
	assert.True(t, store.saved.HasEverActivated, "latch must flip on activation")
	assert.True(t, store.saved.HasPayoutsEverActivated, "payouts latch must flip on activation")
}

func TestNotifier_NoPreviousState_ActivatedAccount_SendsActivationOnce(t *testing.T) {
	n, sink, _ := newTestNotifier(nil)
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	// "Compte de paiement activé" + first-payouts-activation positive notif.
	require.Len(t, sink.calls, 2)
	assert.Contains(t, sink.calls[0].title, "activé")
}

func TestNotifier_ChargesDisabled_SendsChargesDisabledNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = false
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Contains(t, sink.calls[0].title, "Paiements entrants")
}

func TestNotifier_PayoutsDisabled_SendsPayoutsDisabledNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = false

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Contains(t, sink.calls[0].title, "Virements sortants")
}

func TestNotifier_SameState_NoNotification(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	assert.Empty(t, sink.calls)
}

func TestNotifier_CurrentlyDueAdded_SendsRequirementsNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
		CurrentlyDueHash:        "",
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.CurrentlyDue = []string{"individual.verification.document"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Contains(t, sink.calls[0].title, "1 information requise")
	assert.Equal(t, notifdomain.TypeStripeRequirements, sink.calls[0].typ)
}

func TestNotifier_MultipleCurrentlyDue_DuringInitialOnboarding_NoRequirementNotif(t *testing.T) {
	// On the very first webhook with no prior state and no activation
	// yet, the requirement notifications must be suppressed — the user
	// is on the KYC page filling things out and the page itself surfaces
	// requirements. Spamming the bell is what we are fixing.
	n, sink, _ := newTestNotifier(nil)
	snap := snapshot("acct_1")
	snap.ChargesEnabled = false
	snap.CurrentlyDue = []string{"individual.verification.document", "individual.phone"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	for _, c := range sink.calls {
		assert.NotEqual(t, notifdomain.TypeStripeRequirements, c.typ,
			"requirement notifs must be suppressed during initial onboarding")
	}
}

func TestNotifier_MultipleCurrentlyDue_AfterActivation_PluralizesCorrectly(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.CurrentlyDue = []string{"individual.verification.document", "individual.phone"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	foundReq := false
	for _, c := range sink.calls {
		if c.typ == notifdomain.TypeStripeRequirements && contains(c.title, "2 informations requises") {
			foundReq = true
		}
	}
	assert.True(t, foundReq, "expected plural requirements notification")
}

func TestNotifier_CurrentlyDueSameHash_NoNotification(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
		CurrentlyDueHash:        "individual.verification.document",
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.CurrentlyDue = []string{"individual.verification.document"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	assert.Empty(t, sink.calls)
}

func TestNotifier_PastDueAdded_SendsUrgentNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.PastDue = []string{"individual.verification.document"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Contains(t, sink.calls[0].title, "urgente")
	assert.Equal(t, true, sink.calls[0].meta["urgent"])
}

func TestNotifier_DocumentRejected_SendsDocRejectionNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{
			Requirement: "individual.verification.document",
			Code:        "verification_document_expired",
			Reason:      "expired",
		},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Contains(t, sink.calls[0].title, "expiré")
	assert.Equal(t, "verification_document_expired", sink.calls[0].meta["code"])
}

func TestNotifier_DocumentBlurry_FriendlyMessage(t *testing.T) {
	// Prior activation required for document-error notifs to fire.
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Code: "verification_document_too_blurry", Requirement: "individual.verification.document"},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sink.calls), 1)
	foundBlurry := false
	for _, c := range sink.calls {
		if contains(c.title, "illisible") || contains(c.body, "floue") {
			foundBlurry = true
		}
	}
	assert.True(t, foundBlurry)
}

func TestNotifier_ErrorFraudulent_NeutralMessage(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Code: "verification_document_fraudulent", Requirement: "individual.verification.document"},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	foundFraud := false
	for _, c := range sink.calls {
		if contains(c.title, "refusé") {
			foundFraud = true
		}
	}
	assert.True(t, foundFraud)
}

func TestNotifier_UnknownErrorCode_UsesGenericFallback(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Code: "zzz_unknown_code", Requirement: "individual.phone"},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	foundGeneric := false
	for _, c := range sink.calls {
		if contains(c.title, "Action requise") {
			foundGeneric = true
		}
	}
	assert.True(t, foundGeneric)
}

func TestNotifier_SameErrorCodeTwice_NoRepeatNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
		ErrorCodes:              []string{"individual.verification.document:verification_document_expired"},
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Code: "verification_document_expired", Requirement: "individual.verification.document"},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	assert.Empty(t, sink.calls)
}

func TestNotifier_NewErrorAfterOldOne_OnlyNewTriggers(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
		ErrorCodes:              []string{"individual.verification.document:verification_document_expired"},
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Code: "verification_document_expired", Requirement: "individual.verification.document"},
		{Code: "verification_failed_address_match", Requirement: "individual.address"},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Equal(t, "verification_failed_address_match", sink.calls[0].meta["code"])
}

func TestNotifier_AccountDisabled_SendsDisabledNotif(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = false
	snap.PayoutsEnabled = false
	snap.DisabledReason = "rejected.fraud"

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	// Expect 2 notifs: charges_disabled + account_disabled
	assert.GreaterOrEqual(t, len(sink.calls), 1)
	foundDisabled := false
	for _, c := range sink.calls {
		if contains(c.title, "restreint") || contains(c.body, "fraude") {
			foundDisabled = true
		}
	}
	assert.True(t, foundDisabled)
}

func TestNotifier_SameDisabledReason_NoRepeat(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled: false,
		PayoutsEnabled: false,
		DisabledReason: "rejected.fraud",
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = false
	snap.PayoutsEnabled = false
	snap.DisabledReason = "rejected.fraud"

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	assert.Empty(t, sink.calls)
}

func TestNotifier_Cooldown_SuppressesSecondCall(t *testing.T) {
	// Use a payouts-disabled→enabled transition (the cooldown bucket
	// is per Key="payouts_disabled"). On the first transition we go
	// from PayoutsEnabled=false→true, which fires "payouts_disabled"
	// nothing — let's actually exercise an existing already-activated
	// account suspending then resuming charges.
	n, sink, store := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = false
	snap.PayoutsEnabled = true

	// First call — charges_disabled notif sent
	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)
	assert.Contains(t, sink.calls[0].title, "Paiements entrants")

	// Reset store.prev to trigger the transition again
	store.prev = &LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	}

	// Second call with same transition — should be suppressed by cooldown
	err = n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	assert.Len(t, sink.calls, 1) // unchanged
}

func TestNotifier_Cooldown_ExpiresAfterTTL(t *testing.T) {
	sink := &fakeSink{}
	store := &fakeUserStore{
		orgID:       uuid.New(),
		ownerUserID: uuid.New(),
		prev: &LastAccountState{
			ChargesEnabled:          true,
			PayoutsEnabled:          true,
			HasEverActivated:        true,
			HasPayoutsEverActivated: true,
		},
	}
	n := NewNotifier(sink, store, 10*time.Millisecond)

	snap := snapshot("acct_1")
	snap.ChargesEnabled = false
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.Len(t, sink.calls, 1)

	time.Sleep(20 * time.Millisecond)
	store.prev = &LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	}

	err = n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	assert.Len(t, sink.calls, 2)
}

func TestNotifier_SinkFailure_DoesNotCrash(t *testing.T) {
	sink := &fakeSink{err: errors.New("sink down")}
	store := &fakeUserStore{orgID: uuid.New(), ownerUserID: uuid.New()}
	n := NewNotifier(sink, store, time.Minute)

	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	// Best-effort — overall call should not fail even if a notification failed
	require.NoError(t, err)
}

func TestNotifier_DefaultCooldown_ZeroMeansFiveMinutes(t *testing.T) {
	sink := &fakeSink{}
	store := &fakeUserStore{orgID: uuid.New(), ownerUserID: uuid.New()}
	n := NewNotifier(sink, store, 0)
	assert.Equal(t, 5*time.Minute, n.ttl)
}

func TestNotifier_SnapshotToState_EmptyRequirements(t *testing.T) {
	snap := snapshot("acct_1")
	state := snapshotToState(snap)
	assert.Equal(t, "", state.CurrentlyDueHash)
	assert.Equal(t, "", state.PastDueHash)
	assert.Empty(t, state.ErrorCodes)
}

func TestNotifier_SnapshotToState_WithRequirements(t *testing.T) {
	snap := snapshot("acct_1")
	snap.CurrentlyDue = []string{"b", "a", "c"}
	snap.PastDue = []string{"z"}
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Requirement: "a", Code: "err1"},
	}
	state := snapshotToState(snap)
	// Hash should be alphabetically sorted
	assert.Equal(t, "a|b|c", state.CurrentlyDueHash)
	assert.Equal(t, "z", state.PastDueHash)
	assert.Equal(t, []string{"a:err1"}, state.ErrorCodes)
}

func TestNotifier_HashFields_DeterministicOrdering(t *testing.T) {
	h1 := hashFields([]string{"b", "a", "c"})
	h2 := hashFields([]string{"c", "b", "a"})
	h3 := hashFields([]string{"a", "c", "b"})
	assert.Equal(t, h1, h2)
	assert.Equal(t, h2, h3)
}

func TestNotifier_HumanizeDisabledReason(t *testing.T) {
	tests := map[string]string{
		"requirements.past_due":            "informations requises non fournies à temps",
		"requirements.pending_verification": "vérification en cours",
		"listed":                            "compte sur une liste de vérification",
		"rejected.fraud":                    "suspicion de fraude",
		"rejected.terms_of_service":         "violation des conditions d'utilisation",
		"unknown_code":                      "unknown_code",
	}
	for code, expected := range tests {
		assert.Equal(t, expected, humanizeDisabledReason(code), "code=%s", code)
	}
}

func TestNotifier_PluralS(t *testing.T) {
	assert.Equal(t, "", pluralS(0))
	assert.Equal(t, "", pluralS(1))
	assert.Equal(t, "s", pluralS(2))
	assert.Equal(t, "s", pluralS(10))
}

func TestNotifier_ErrorMessageFor_AllKnownCodes(t *testing.T) {
	codes := []string{
		"verification_document_expired",
		"verification_document_too_blurry",
		"verification_document_not_readable",
		"verification_document_name_mismatch",
		"verification_document_nationality_mismatch",
		"verification_document_fraudulent",
		"verification_document_manipulated",
		"verification_failed_address_match",
		"verification_failed_id_number_match",
		"invalid_value_other",
	}
	for _, code := range codes {
		title, body := errorMessageFor(code)
		assert.NotEmpty(t, title, "code=%s", code)
		assert.NotEmpty(t, body, "code=%s", code)
	}
}

func TestNotifier_MetadataContainsAccountID(t *testing.T) {
	n, sink, _ := newTestNotifier(nil)
	snap := snapshot("acct_xyz")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	// 2 notifs: account_active + payouts_first_activated.
	require.GreaterOrEqual(t, len(sink.calls), 1)
	for _, c := range sink.calls {
		assert.Equal(t, "acct_xyz", c.meta["account_id"])
	}
}

func TestNotifier_PersistsNewState(t *testing.T) {
	n, _, store := newTestNotifier(&LastAccountState{})
	snap := snapshot("acct_1")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.CurrentlyDue = []string{"foo"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	require.NotNil(t, store.saved)
	assert.True(t, store.saved.ChargesEnabled)
	assert.True(t, store.saved.PayoutsEnabled)
	assert.Equal(t, "foo", store.saved.CurrentlyDueHash)
}

func TestNotifier_DiffErrors_NoPrev_AllNew(t *testing.T) {
	prev := &LastAccountState{ErrorCodes: nil}
	cur := &LastAccountState{ErrorCodes: []string{"req1:code1", "req2:code2"}}
	diff := diffErrors(prev, cur)
	assert.Len(t, diff, 2)
}

func TestNotifier_DiffErrors_PrevHasAll_NoneNew(t *testing.T) {
	prev := &LastAccountState{ErrorCodes: []string{"req1:code1", "req2:code2"}}
	cur := &LastAccountState{ErrorCodes: []string{"req1:code1", "req2:code2"}}
	diff := diffErrors(prev, cur)
	assert.Empty(t, diff)
}

func TestNotifier_DiffErrors_NilPrev_AllNew(t *testing.T) {
	cur := &LastAccountState{ErrorCodes: []string{"req1:code1"}}
	diff := diffErrors(nil, cur)
	assert.Len(t, diff, 1)
}

/* ----------------------- KYC onboarding gating ----------------------- */

// TestNotifier_OnboardingState_NoRequirementNotifications covers the
// fix for Task B: when the user just created a Stripe Connect account
// and Stripe immediately reports a long currently_due / eventually_due
// list, the notification bell must stay quiet — the user is on the
// KYC page filling things out, the page itself surfaces requirements.
func TestNotifier_OnboardingState_NoRequirementNotifications(t *testing.T) {
	// No prior state → first webhook for a freshly-created account.
	n, sink, store := newTestNotifier(nil)
	snap := snapshot("acct_new")
	snap.ChargesEnabled = false
	snap.PayoutsEnabled = false
	// A realistic load-on-creation: 11 currently_due + 4 eventually_due.
	snap.CurrentlyDue = []string{
		"individual.verification.document",
		"individual.address.line1", "individual.address.city",
		"individual.address.postal_code", "individual.first_name",
		"individual.last_name", "individual.dob.day", "individual.dob.month",
		"individual.dob.year", "individual.id_number", "individual.phone",
	}
	snap.EventuallyDue = []string{
		"external_account", "individual.verification.additional_document",
		"tos_acceptance.date", "tos_acceptance.ip",
	}
	snap.RequirementErrors = []portservice.StripeRequirementError{
		{Requirement: "individual.verification.document",
			Code: "verification_document_expired"},
	}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	// Zero notifications: not a single requirement-class notif must
	// fire. The user has not yet activated, so the bell stays quiet.
	for _, c := range sink.calls {
		assert.NotEqual(t, notifdomain.TypeStripeRequirements, c.typ,
			"requirement notifs must be suppressed during initial onboarding (got: %s)", c.title)
	}

	// State is still persisted so the next webhook can diff against it.
	require.NotNil(t, store.saved)
	assert.False(t, store.saved.HasEverActivated)
	assert.False(t, store.saved.HasPayoutsEverActivated)
}

// TestNotifier_AlreadyActivated_RequirementNotificationsFire is the
// counterpart: once the user has been active at least once in the
// past, Stripe adding new requirements DOES warrant a notification
// (it usually means an additional verification round, e.g. annual
// re-attestation).
func TestNotifier_AlreadyActivated_RequirementNotificationsFire(t *testing.T) {
	n, sink, _ := newTestNotifier(&LastAccountState{
		ChargesEnabled:          true,
		PayoutsEnabled:          true,
		HasEverActivated:        true,
		HasPayoutsEverActivated: true,
	})
	snap := snapshot("acct_existing")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true
	snap.CurrentlyDue = []string{"individual.verification.additional_document"}

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	foundReq := false
	for _, c := range sink.calls {
		if c.typ == notifdomain.TypeStripeRequirements {
			foundReq = true
		}
	}
	assert.True(t, foundReq,
		"after first activation, requirement notifs must fire on new currently_due")
}

// TestNotifier_PayoutsFirstActivation_FiresOnce covers Task C: the
// first time we see PayoutsEnabled flip from false to true, fire the
// "Virements sortants activés" notification. Subsequent webhooks
// reporting PayoutsEnabled=true again must NOT re-fire.
func TestNotifier_PayoutsFirstActivation_FiresOnce(t *testing.T) {
	// Prev: payouts not yet activated.
	n, sink, store := newTestNotifier(&LastAccountState{
		ChargesEnabled:          false,
		PayoutsEnabled:          false,
		HasEverActivated:        false,
		HasPayoutsEverActivated: false,
	})
	snap := snapshot("acct_xyz")
	snap.ChargesEnabled = true
	snap.PayoutsEnabled = true

	err := n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)

	// Find the payouts-activated notif
	foundPayoutsActivated := 0
	for _, c := range sink.calls {
		if contains(c.title, "Virements sortants activés") {
			foundPayoutsActivated++
			assert.Equal(t, "payouts_activated", c.meta["status"])
		}
	}
	assert.Equal(t, 1, foundPayoutsActivated,
		"first PayoutsEnabled=true transition must fire the activation notif exactly once")

	// Latch must be set on the saved state.
	require.NotNil(t, store.saved)
	assert.True(t, store.saved.HasPayoutsEverActivated)

	// Second webhook with the same state — must NOT re-fire.
	store.prev = store.saved
	beforeCount := len(sink.calls)
	err = n.HandleAccountSnapshot(context.Background(), snap)
	require.NoError(t, err)
	for _, c := range sink.calls[beforeCount:] {
		assert.NotContains(t, c.title, "Virements sortants activés",
			"payouts-first-activated notif must not re-fire after the latch is set")
	}
}

/* ----------------------------- utils ----------------------------- */

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
