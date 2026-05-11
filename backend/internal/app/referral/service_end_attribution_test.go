package referral_test

// Unit tests for Service.EndIntroAttribution — the WALLET-UNIFY Run A
// "Terminer l'intro" use case. Covers:
//   - Happy path (audit + notifs + ended_at populated).
//   - Idempotency (2nd call returns the already-ended row with no
//     extra audit or notification).
//   - RBAC failure (cross-tenant caller → ErrNotAuthorized).
//   - Not found (bogus attribution id).
//
// Uses the existing fakeReferralRepo / fakeAuditRepo / fakeNotifier
// in mocks_test.go and the newTestFixture builder in service_test.go.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/referral"
)

// seedActiveAttribution creates a fully wired referral + active
// attribution and returns the parent referral id + attribution id.
// The attribution is freshly inserted (ended_at NULL).
func seedActiveAttribution(t *testing.T, f *testFixture) (
	referrerID, providerID, clientID, refID, attID uuid.UUID,
) {
	t.Helper()
	ctx := context.Background()
	referrerID, providerID, clientID = f.seedActors(t)
	parent := f.createIntro(t, referrerID, providerID, clientID, 5)
	refID = parent.ID

	att, err := referral.NewAttribution(referral.NewAttributionInput{
		ReferralID:      refID,
		ProposalID:      uuid.New(),
		ProviderID:      providerID,
		ClientID:        clientID,
		RatePctSnapshot: 5,
	})
	require.NoError(t, err)
	require.NoError(t, f.repo.CreateAttribution(ctx, att))
	return referrerID, providerID, clientID, refID, att.ID
}

func TestEndIntroAttribution_Success(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	referrerID, providerID, clientID, refID, attID := seedActiveAttribution(t, f)
	_ = refID

	// Drain notifications recorded during CreateIntro so we measure
	// only the EndIntroAttribution emissions.
	startNotif := len(f.notifier.notifications)

	got, err := f.svc.EndIntroAttribution(context.Background(), attID, referrerID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.True(t, got.IsEnded(), "EndedAt must be populated on the returned attribution")

	// Audit row emitted.
	entries := f.audits.entriesOfAction(audit.ActionReferralIntroAttributionEnded)
	require.Len(t, entries, 1, "exactly one audit entry recorded")
	assert.Equal(t, audit.ResourceTypeReferralAttribution, entries[0].ResourceType)
	require.NotNil(t, entries[0].ResourceID)
	assert.Equal(t, attID, *entries[0].ResourceID)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, referrerID, *entries[0].UserID)
	assert.Contains(t, entries[0].Metadata, "referral_id")
	assert.Contains(t, entries[0].Metadata, "proposal_id")
	assert.Contains(t, entries[0].Metadata, "ended_at")

	// Both parties received the notification (typeCount = +2 since
	// fanOut with nil orgMembers falls back to [anchor]).
	terminatedCount := f.notifier.typeCount(string(notification.TypeReferralIntroTerminated))
	assert.GreaterOrEqual(t, terminatedCount, 2, "provider + client must receive the end notification")

	// One notif to provider AND one notif to client (by user id).
	assert.GreaterOrEqual(t,
		f.notifier.toUserTypeCount(providerID, string(notification.TypeReferralIntroTerminated)), 1,
		"provider must receive end notification")
	assert.GreaterOrEqual(t,
		f.notifier.toUserTypeCount(clientID, string(notification.TypeReferralIntroTerminated)), 1,
		"client must receive end notification")
	assert.GreaterOrEqual(t, len(f.notifier.notifications), startNotif+2)
}

func TestEndIntroAttribution_Idempotent(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	referrerID, _, _, _, attID := seedActiveAttribution(t, f)

	// First call — emits audit + notifs.
	_, err := f.svc.EndIntroAttribution(context.Background(), attID, referrerID)
	require.NoError(t, err)
	auditCountFirst := len(f.audits.entriesOfAction(audit.ActionReferralIntroAttributionEnded))
	notifCountFirst := f.notifier.typeCount(string(notification.TypeReferralIntroTerminated))

	// Second call — must return the same row, no extra audit, no
	// extra notifs.
	got, err := f.svc.EndIntroAttribution(context.Background(), attID, referrerID)
	require.NoError(t, err, "second call must succeed idempotently")
	require.NotNil(t, got)
	assert.True(t, got.IsEnded())

	assert.Equal(t, auditCountFirst,
		len(f.audits.entriesOfAction(audit.ActionReferralIntroAttributionEnded)),
		"idempotent re-end must NOT emit a second audit entry")
	assert.Equal(t, notifCountFirst,
		f.notifier.typeCount(string(notification.TypeReferralIntroTerminated)),
		"idempotent re-end must NOT emit additional notifications")
}

func TestEndIntroAttribution_NotOwner(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	_, _, _, _, attID := seedActiveAttribution(t, f)

	// A different user (not the apporteur) tries to end. The service
	// rejects with ErrNotAuthorized before touching the DB.
	stranger := uuid.New()
	_, err := f.svc.EndIntroAttribution(context.Background(), attID, stranger)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)

	// No mutation, no audit, no notifs.
	got, ferr := f.repo.FindAttributionByID(context.Background(), attID)
	require.NoError(t, ferr)
	assert.False(t, got.IsEnded(), "cross-tenant attempt must not mutate the row")
	assert.Empty(t, f.audits.entriesOfAction(audit.ActionReferralIntroAttributionEnded))
}

func TestEndIntroAttribution_NotFound(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	referrerID, _, _ := f.seedActors(t)

	_, err := f.svc.EndIntroAttribution(context.Background(), uuid.New(), referrerID)
	require.ErrorIs(t, err, referral.ErrAttributionNotFound)
	assert.Empty(t, f.audits.entriesOfAction(audit.ActionReferralIntroAttributionEnded))
}

// TestEndIntroAttribution_RaceAlreadyEnded simulates the case where
// another caller ended the attribution between our service's load and
// our service's UPDATE. The repository surfaces ErrAttributionAlreadyEnded;
// the service treats it as an idempotent success — reloads the row and
// returns it without emitting a duplicate audit or notification.
func TestEndIntroAttribution_RaceAlreadyEnded(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	referrerID, _, _, _, attID := seedActiveAttribution(t, f)
	ctx := context.Background()

	// Pre-end the attribution via a direct repo call — this simulates
	// another concurrent caller winning the race. The service's
	// pre-check will see "already ended" via FindAttributionByID and
	// short-circuit BEFORE the EndAttribution call, returning the
	// idempotent success path without auditing.
	require.NoError(t, f.repo.EndAttribution(ctx, attID, referrerID))

	got, err := f.svc.EndIntroAttribution(ctx, attID, referrerID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsEnded())
	// No audit emitted (idempotent re-call).
	assert.Empty(t, f.audits.entriesOfAction(audit.ActionReferralIntroAttributionEnded))
}
