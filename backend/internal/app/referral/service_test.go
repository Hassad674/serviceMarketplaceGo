package referral_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/domain/user"
	portservice "marketplace-backend/internal/port/service"
)

// testFixture bundles the service + all its mocks for use in individual tests.
type testFixture struct {
	svc      *referralapp.Service
	repo     *fakeReferralRepo
	users    *fakeUserRepo
	msgs     *fakeMessageSender
	notifier *fakeNotifier
	stripe   *fakeStripe
	reversal *fakeReversalService
	accounts *fakeStripeAccountResolver
}

func newTestFixture(t *testing.T, accountID string) *testFixture {
	t.Helper()
	repo := newFakeReferralRepo()
	users := newFakeUserRepo()
	msgs := &fakeMessageSender{}
	notifier := &fakeNotifier{}
	stripe := &fakeStripe{}
	reversal := &fakeReversalService{}
	accounts := &fakeStripeAccountResolver{accountID: accountID}
	snap := &fakeSnapshotLoader{
		provider: referral.ProviderSnapshot{Region: "IDF"},
	}

	svc := referralapp.NewService(referralapp.ServiceDeps{
		Referrals:        repo,
		Users:            users,
		Messages:         msgs,
		Notifications:    notifier,
		Stripe:           stripe,
		Reversals:        reversal,
		SnapshotProfiles: snap,
		StripeAccounts:   accounts,
	})

	return &testFixture{
		svc:      svc,
		repo:     repo,
		users:    users,
		msgs:     msgs,
		notifier: notifier,
		stripe:   stripe,
		reversal: reversal,
		accounts: accounts,
	}
}

// seedActors registers three users (referrer, provider, client) in the
// fake user repository with the correct roles and returns their IDs.
func (f *testFixture) seedActors(t *testing.T) (referrerID, providerID, clientID uuid.UUID) {
	t.Helper()
	referrerID = uuid.New()
	providerID = uuid.New()
	clientID = uuid.New()
	f.users.add(referrerID, user.RoleProvider, true)
	f.users.add(providerID, user.RoleProvider, false)
	f.users.add(clientID, user.RoleEnterprise, false)
	return
}

func (f *testFixture) createIntro(t *testing.T, referrerID, providerID, clientID uuid.UUID, rate float64) *referral.Referral {
	t.Helper()
	r, err := f.svc.CreateIntro(context.Background(), referralapp.CreateIntroInput{
		ReferrerID:           referrerID,
		ProviderID:           providerID,
		ClientID:             clientID,
		RatePct:              rate,
		DurationMonths:       6,
		IntroMessageProvider: "pitch provider",
		IntroMessageClient:   "pitch client",
	})
	require.NoError(t, err)
	return r
}

func TestCreateIntro_HappyPath(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)

	r := f.createIntro(t, refID, provID, cliID, 5)

	assert.Equal(t, referral.StatusPendingProvider, r.Status)
	assert.Equal(t, 5.0, r.RatePct)
	// One notification sent to the provider.
	assert.Equal(t, 1, f.notifier.typeCount(string(notification.TypeReferralIntroCreated)))
	// One initial negotiation row.
	negos, _ := f.repo.ListNegotiations(context.Background(), r.ID)
	require.Len(t, negos, 1)
	assert.Equal(t, referral.NegoActionProposed, negos[0].Action)
}

func TestCreateIntro_ReferrerNotEnabled(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	// Referrer role is provider but referrer_enabled is false → reject.
	f.users.add(referrerID, user.RoleProvider, false)
	f.users.add(providerID, user.RoleProvider, false)
	f.users.add(clientID, user.RoleEnterprise, false)

	_, err := f.svc.CreateIntro(context.Background(), referralapp.CreateIntroInput{
		ReferrerID:           referrerID,
		ProviderID:           providerID,
		ClientID:             clientID,
		RatePct:              5,
		DurationMonths:       6,
		IntroMessageProvider: "p",
		IntroMessageClient:   "c",
	})
	require.ErrorIs(t, err, referral.ErrReferrerRequired)
}

func TestCreateIntro_InvalidProviderRole(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	f.users.add(referrerID, user.RoleProvider, true)
	f.users.add(providerID, user.RoleEnterprise, false) // invalid — enterprise can't be a "provider party"
	f.users.add(clientID, user.RoleEnterprise, false)

	_, err := f.svc.CreateIntro(context.Background(), referralapp.CreateIntroInput{
		ReferrerID:           referrerID,
		ProviderID:           providerID,
		ClientID:             clientID,
		RatePct:              5,
		IntroMessageProvider: "p",
		IntroMessageClient:   "c",
	})
	require.ErrorIs(t, err, referral.ErrInvalidProviderRole)
}

func TestRespondAsProvider_AcceptMovesToPendingClient(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	updated, err := f.svc.RespondAsProvider(context.Background(), referralapp.NewResponseInput(r.ID, provID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)
	assert.Equal(t, referral.StatusPendingClient, updated.Status)
	// Referrer gets the "provider accepted" confirmation.
	assert.Equal(t, 1, f.notifier.typeCount(string(notification.TypeReferralIntroAcceptedByProvider)))
	// Client gets a "new intro awaiting your decision" notification too —
	// this is the fan-out guarantee: both sides of the transition are
	// informed, not just the side that just acted.
	assert.GreaterOrEqual(t, f.notifier.toUserTypeCount(cliID, string(notification.TypeReferralIntroCreated)), 1,
		"client must be notified when the provider accepts and status moves to pending_client")
}

func TestRespondAsProvider_CounterMovesToPendingReferrer(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	updated, err := f.svc.RespondAsProvider(context.Background(), referralapp.NewResponseInput(r.ID, provID, referral.NegoActionCountered, 3, "too high"))
	require.NoError(t, err)
	assert.Equal(t, referral.StatusPendingReferrer, updated.Status)
	assert.Equal(t, 3.0, updated.RatePct)
	assert.Equal(t, 2, updated.Version)
	assert.Equal(t, 1, f.notifier.typeCount(string(notification.TypeReferralIntroNegotiated)))
}

func TestRespondAsReferrer_AcceptCounter(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	// Provider counters.
	_, err := f.svc.RespondAsProvider(context.Background(), referralapp.NewResponseInput(r.ID, provID, referral.NegoActionCountered, 3, ""))
	require.NoError(t, err)

	// Referrer accepts the counter.
	updated, err := f.svc.RespondAsReferrer(context.Background(), referralapp.NewResponseInput(r.ID, refID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)
	assert.Equal(t, referral.StatusPendingClient, updated.Status)
}

func TestRespondAsClient_AcceptActivatesAndOpensConversation(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	// Bring to pending_client via provider accept.
	_, err := f.svc.RespondAsProvider(context.Background(), referralapp.NewResponseInput(r.ID, provID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)

	updated, err := f.svc.RespondAsClient(context.Background(), referralapp.NewResponseInput(r.ID, cliID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)
	assert.Equal(t, referral.StatusActive, updated.Status)
	require.NotNil(t, updated.ActivatedAt)
	require.NotNil(t, updated.ExpiresAt)

	// Phase B: three distinct conv pairs are used through the lifecycle —
	// apporteur↔provider (from creation), apporteur↔client (from
	// pending_client), provider↔client (at activation). The apporteur
	// is NEVER a participant of the provider↔client conv.
	pairs := map[string]bool{}
	for _, c := range f.msgs.convsCreated {
		pairs[convPairKey(c.UserA, c.UserB)] = true
	}
	assert.True(t, pairs[convPairKey(refID, provID)], "apporteur↔provider conv must exist")
	assert.True(t, pairs[convPairKey(refID, cliID)], "apporteur↔client conv must exist")
	assert.True(t, pairs[convPairKey(provID, cliID)], "provider↔client conv must exist")

	// The activation system message lands in the provider↔client conv
	// as well as in the two apporteur-facing conv pairs.
	activations := f.msgs.sysMessagesOfType("referral_intro_activated")
	assert.GreaterOrEqual(t, len(activations), 3,
		"activation posts in provider↔client, apporteur↔provider, apporteur↔client")

	// Activation notification sent to the three parties.
	assert.GreaterOrEqual(t, f.notifier.typeCount(string(notification.TypeReferralIntroActivated)), 3)
}

func TestCreateIntro_PostsSystemMessageInApporteurProviderConv(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	sent := f.msgs.sysMessagesOfType("referral_intro_sent")
	require.Len(t, sent, 1, "one intro_sent message in apporteur↔provider conv on creation")

	// Metadata must carry referral_id + rate_pct so the widget can render.
	require.Contains(t, string(sent[0].Metadata), r.ID.String())
	require.Contains(t, string(sent[0].Metadata), `"rate_pct":5`)
}

func TestProviderCounter_PostsNegotiatedSystemMessage(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	_, err := f.svc.RespondAsProvider(context.Background(), referralapp.NewResponseInput(r.ID, provID, referral.NegoActionCountered, 3, "too high"))
	require.NoError(t, err)

	negotiated := f.msgs.sysMessagesOfType("referral_intro_negotiated")
	require.Len(t, negotiated, 1, "one negotiated message per counter offer")
	// The sender is the actor who just counter-offered (the provider).
	assert.Equal(t, provID, negotiated[0].SenderID)
	// Metadata carries the new rate so the widget shows the fresh offer.
	assert.Contains(t, string(negotiated[0].Metadata), `"rate_pct":3`)
}

func TestClientAccept_StripsRateFromApporteurClientMessage(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	_, err := f.svc.RespondAsProvider(context.Background(), referralapp.NewResponseInput(r.ID, provID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)

	// Provider accept transitions to pending_client and posts intro_sent
	// messages in BOTH apporteur↔provider (rate visible) AND apporteur↔client
	// (rate stripped — Modèle A).
	sent := f.msgs.sysMessagesOfType("referral_intro_sent")
	withRate := 0
	withoutRate := 0
	for _, m := range sent {
		if strings.Contains(string(m.Metadata), `"rate_pct":5`) {
			withRate++
			continue
		}
		withoutRate++
	}
	assert.GreaterOrEqual(t, withRate, 1, "apporteur↔provider intro_sent must carry the rate")
	assert.GreaterOrEqual(t, withoutRate, 1, "apporteur↔client intro_sent must strip the rate (Modèle A)")
	_ = refID
	_ = r
}

func TestCancel_ReferrerOnly(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)

	// Provider cannot cancel.
	_, err := f.svc.Cancel(context.Background(), r.ID, provID)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)

	// Referrer can.
	updated, err := f.svc.Cancel(context.Background(), r.ID, refID)
	require.NoError(t, err)
	assert.Equal(t, referral.StatusCancelled, updated.Status)
}

func TestAttributor_NoActiveReferral_NoOp(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	// No intro — attributor should no-op.
	err := f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(uuid.New(), uuid.New(), uuid.New()))
	require.NoError(t, err)
}

func TestAttributor_CreatesAttributionOnActiveReferral(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	err := f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID))
	require.NoError(t, err)

	// Verify attribution row exists.
	att, err := f.repo.FindAttributionByProposal(context.Background(), proposalID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, att.ReferralID)
	assert.Equal(t, 5.0, att.RatePctSnapshot)
}

func TestDistributor_PaysCommission(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "paid", string(result))
	require.Len(t, f.stripe.transfers, 1)
	assert.Equal(t, int64(5000), f.stripe.transfers[0].Amount) // 5% of 100000 = 5000 cents
	assert.Equal(t, "acct_referrer", f.stripe.transfers[0].DestinationAccount)
}

func TestDistributor_PendingKYCWhenNoAccount(t *testing.T) {
	f := newTestFixture(t, "") // no stripe account
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "pending_kyc", string(result))
	assert.Empty(t, f.stripe.transfers)
	assert.Equal(t, 1, f.notifier.typeCount(string(notification.TypeReferralCommissionPendingKYC)))
}

func TestDistributor_Idempotent(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))

	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// Second call on the same milestone must skip and NOT create a second transfer.
	result, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)
	assert.Equal(t, "skipped", string(result))
	assert.Len(t, f.stripe.transfers, 1)
}

func TestClawback_FullRefundReversesCommission(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	err = f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(milestoneID, 1000_00, 1000_00))
	require.NoError(t, err)
	require.Len(t, f.reversal.reversals, 1)
	assert.Equal(t, int64(5000), f.reversal.reversals[0].Amount) // full commission reversed
	assert.Equal(t, 1, f.notifier.typeCount(string(notification.TypeReferralCommissionClawedBack)))
}

func TestClawback_PartialRefundProportional(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, f.svc.CreateAttributionIfExists(context.Background(), attrInputFor(proposalID, provID, cliID)))
	milestoneID := uuid.New()
	_, err := f.svc.DistributeIfApplicable(context.Background(), distInputFor(proposalID, milestoneID, 1000_00))
	require.NoError(t, err)

	// Refund half of the milestone.
	err = f.svc.ClawbackIfApplicable(context.Background(), clawbackInput(milestoneID, 500_00, 1000_00))
	require.NoError(t, err)
	require.Len(t, f.reversal.reversals, 1)
	assert.Equal(t, int64(2500), f.reversal.reversals[0].Amount) // half of 5000
}

func TestExpireStaleIntros(t *testing.T) {
	f := newTestFixture(t, "acct_xyz")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	// Rewind last_action_at.
	stored, _ := f.repo.GetByID(context.Background(), r.ID)
	stored.LastActionAt = time.Now().UTC().Add(-20 * 24 * time.Hour)
	require.NoError(t, f.repo.Update(context.Background(), stored))

	count, err := f.svc.ExpireStaleIntros(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	after, _ := f.repo.GetByID(context.Background(), r.ID)
	assert.Equal(t, referral.StatusExpired, after.Status)
}

// ─── Test helpers ─────────────────────────────────────────────────────────

// bringToActive walks a fresh referral from pending_provider → active so
// attributor/distributor tests can start from a stable known state.
func bringToActive(t *testing.T, svc *referralapp.Service, r *referral.Referral, provID, cliID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := svc.RespondAsProvider(ctx, referralapp.NewResponseInput(r.ID, provID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)
	_, err = svc.RespondAsClient(ctx, referralapp.NewResponseInput(r.ID, cliID, referral.NegoActionAccepted, 0, ""))
	require.NoError(t, err)
}

func attrInputFor(proposalID, providerID, clientID uuid.UUID) portservice.ReferralAttributorInput {
	return portservice.ReferralAttributorInput{
		ProposalID: proposalID,
		ProviderID: providerID,
		ClientID:   clientID,
	}
}

func distInputFor(proposalID, milestoneID uuid.UUID, grossCents int64) portservice.ReferralCommissionDistributorInput {
	return portservice.ReferralCommissionDistributorInput{
		ProposalID:       proposalID,
		MilestoneID:      milestoneID,
		GrossAmountCents: grossCents,
		Currency:         "EUR",
	}
}

func clawbackInput(milestoneID uuid.UUID, refundedCents, grossCents int64) portservice.ReferralClawbackInput {
	return portservice.ReferralClawbackInput{
		MilestoneID:   milestoneID,
		RefundedCents: refundedCents,
		GrossCents:    grossCents,
	}
}
