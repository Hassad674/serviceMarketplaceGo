package subscription_test

// End-to-end integration test for Phase B.1: the full Premium vertical
// slice end to end, all the way to a real Postgres row and a
// fee-waived payment_record.
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL so CI / fresh checkouts
// auto-skip when the env var is absent. Stripe is mocked at the
// service-port boundary — we do NOT hit the real Stripe API in this
// test (too slow, flaky, and would require test clock setup). The
// webhook-driven transitions are simulated by calling the app service
// directly with a SubscriptionSnapshot, which is exactly what the
// webhook dispatcher does in production.

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsub "marketplace-backend/internal/app/subscription"
	paymentapp "marketplace-backend/internal/app/payment"
	pgadapter "marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	domain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/port/service"
)

func e2eTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping subscription E2E")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "open test database")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx), "ping test database")
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// e2eProposalFixture inserts a conversation + proposal + milestone triple
// so payment_record inserts satisfy their FKs. Returns the milestone id
// (the only one the payment layer needs as input). Registers cleanup.
func e2eProposalFixture(t *testing.T, db *sql.DB, clientID, providerID uuid.UUID, amountCents int64) uuid.UUID {
	t.Helper()
	convID := uuid.New()
	_, err := db.Exec(`INSERT INTO conversations (id, created_at, updated_at) VALUES ($1, now(), now())`, convID)
	require.NoError(t, err)

	proposalID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO proposals (
			id, conversation_id, sender_id, recipient_id, title, description,
			amount, status, version, client_id, provider_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'E2E Proposal', 'desc',
			$5, 'pending', 1, $3, $4, now(), now())`,
		proposalID, convID, clientID, providerID, amountCents,
	)
	require.NoError(t, err)

	milestoneID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO proposal_milestones (
			id, proposal_id, sequence, title, description, amount, status, created_at, updated_at
		) VALUES ($1, $2, 1, 'E2E Milestone', 'desc', $3, 'pending_funding', now(), now())`,
		milestoneID, proposalID, amountCents,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM payment_records WHERE milestone_id = $1`, milestoneID)
		_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE proposal_id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})
	return milestoneID
}

// e2eInsertUser creates a user AND their personal organization so
// subscription flows (org-scoped since migration 119) have something to
// attach to. Returns the user id; the caller fetches the org_id via
// users.organization_id when needed (see e2eUserOrg).
func e2eInsertUser(t *testing.T, db *sql.DB, role string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	orgID := uuid.New()
	email := userID.String()[:8] + "@e2e.local"
	// users.organization_id + organizations.owner_user_id form a cycle.
	// Insert user with NULL org, the org with the owner ref, then link.
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'x', 'E2E', 'User', 'E2E User', $3)`,
		userID, email, role,
	)
	require.NoError(t, err)
	orgType := role
	if role == "provider" {
		orgType = "provider_personal" // matches organizations_type_check
	}
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, $3, $4)`,
		orgID, userID, orgType, "E2E Org "+userID.String()[:8],
	)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM payment_records WHERE provider_id = $1 OR client_id = $1`, userID)
		_, _ = db.Exec(`DELETE FROM subscriptions WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, userID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
	})
	return userID
}

// e2eUserOrg reads back the organization_id associated with a test user.
// Isolated so the test reads stay tight to the SQL contract.
func e2eUserOrg(t *testing.T, db *sql.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	var orgID uuid.UUID
	err := db.QueryRow(`SELECT organization_id FROM users WHERE id = $1`, userID).Scan(&orgID)
	require.NoError(t, err)
	return orgID
}

// e2eStripe implements both StripeSubscriptionService AND the minimal
// StripeService methods the payment feature calls inside CreatePaymentIntent.
// We don't care about Stripe correctness in this test — we care about the
// end-to-end integration inside OUR code.
type e2eStripe struct {
	createdClientSecret string
}

// --- StripeSubscriptionService ---

func (e *e2eStripe) EnsureCustomer(_ context.Context, userID, _, _ string) (string, error) {
	return "cus_e2e_" + userID[:8], nil
}
func (e *e2eStripe) CreateCheckoutSession(_ context.Context, in service.CreateCheckoutSessionInput) (string, error) {
	e.createdClientSecret = "cs_e2e_" + in.PriceID
	return e.createdClientSecret, nil
}
func (e *e2eStripe) EnrichCustomerWithBillingProfile(_ context.Context, _ string, _ service.BillingProfileStripeSnapshot) error {
	return nil
}
func (e *e2eStripe) ResolvePriceID(_ context.Context, lookupKey string) (string, error) {
	return "price_" + lookupKey, nil
}
func (e *e2eStripe) UpdateCancelAtPeriodEnd(_ context.Context, subID string, cancel bool) (service.SubscriptionSnapshot, error) {
	return service.SubscriptionSnapshot{
		ID: subID, Status: "active", CancelAtPeriodEnd: cancel,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}
func (e *e2eStripe) ChangeCycleImmediate(_ context.Context, subID, newPriceID string) (service.SubscriptionSnapshot, error) {
	return service.SubscriptionSnapshot{
		ID: subID, Status: "active", PriceID: newPriceID,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(365 * 24 * time.Hour),
	}, nil
}
func (e *e2eStripe) ScheduleCycleChange(_ context.Context, subID, newPriceID string) (service.ScheduledCycleChange, error) {
	effectiveAt := time.Now().Add(365 * 24 * time.Hour)
	return service.ScheduledCycleChange{
		ScheduleID:  "sched_e2e_" + subID,
		EffectiveAt: effectiveAt,
		Snapshot: service.SubscriptionSnapshot{
			ID: subID, Status: "active", PriceID: "price_current",
			CurrentPeriodStart: time.Now(), CurrentPeriodEnd: effectiveAt,
		},
	}, nil
}
func (e *e2eStripe) ReleaseSchedule(_ context.Context, _ string) error { return nil }
func (e *e2eStripe) PreviewCycleChange(_ context.Context, _ string, _ string, prorateImmediately bool) (service.InvoicePreview, error) {
	amount := int64(0)
	if prorateImmediately {
		amount = 41900
	}
	return service.InvoicePreview{
		AmountDueCents: amount, Currency: "eur",
		PeriodStart: time.Now(), PeriodEnd: time.Now().Add(365 * 24 * time.Hour),
	}, nil
}
func (e *e2eStripe) CreatePortalSession(_ context.Context, customerID, _ string) (string, error) {
	return "https://portal.stripe.test/" + customerID, nil
}

// e2ePaymentStripe implements the smaller StripeService surface the
// payment feature consumes. The test only exercises CreatePaymentIntent.
type e2ePaymentStripe struct{}

func (e *e2ePaymentStripe) CreatePaymentIntent(_ context.Context, in service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	return &service.PaymentIntentResult{
		PaymentIntentID: "pi_e2e_" + in.ProposalID,
		ClientSecret:    "cs_e2e_" + in.ProposalID,
		AmountTotal:     in.AmountCentimes,
	}, nil
}
func (e *e2ePaymentStripe) CreateTransfer(_ context.Context, _ service.CreateTransferInput) (string, error) {
	return "tr_e2e", nil
}
func (e *e2ePaymentStripe) ConstructWebhookEvent(_ []byte, _ string) (*service.StripeWebhookEvent, error) {
	return nil, nil
}
func (e *e2ePaymentStripe) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	return &service.StripeAccountInfo{ChargesEnabled: true, PayoutsEnabled: true}, nil
}
func (e *e2ePaymentStripe) CreateRefund(_ context.Context, _ string, _ int64) (string, error) {
	return "re_e2e", nil
}

// TestSubscriptionE2E_FullLifecycle exercises the entire Premium arc —
// from a free user's first subscribe to a milestone after the
// subscription expires — through the REAL app services and postgres
// repositories. Only the Stripe boundary is stubbed.
//
// The story:
//
//  1. A provider user exists. They have no subscription.
//  2. Creating a payment record charges the normal platform fee.
//  3. The user calls Subscribe → Stripe checkout URL returned.
//  4. Stripe's `customer.subscription.created` webhook lands → the
//     subscription app service persists an active row.
//  5. IsActive returns true; the Redis cache is coherent.
//  6. A new payment record is created → platform_fee is waived (= 0).
//  7. ToggleAutoRenew(false) flips cancel_at_period_end to TRUE.
//  8. Stripe's `customer.subscription.deleted` webhook lands → row
//     transitions to canceled; IsActive returns false after invalidation.
//  9. A new payment record is created AGAIN → platform_fee reapplied.
func TestSubscriptionE2E_FullLifecycle(t *testing.T) {
	db := e2eTestDB(t)

	// ---- Build the real dependency graph (like main.go) ----
	userRepo := pgadapter.NewUserRepository(db)
	_ = userRepo // only used for the subscription app service below

	subRepo := pgadapter.NewSubscriptionRepository(db)
	amountsRepo := pgadapter.NewProviderMilestoneAmountsRepository(db)
	paymentRepo := pgadapter.NewPaymentRecordRepository(db)

	// Real mini-redis — exercises the cache + invalidation path exactly
	// like production.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	stripeSub := &e2eStripe{}
	subSvc := appsub.NewService(appsub.ServiceDeps{
		Subscriptions: subRepo,
		Users:         userRepo,
		Amounts:       amountsRepo,
		Stripe:        stripeSub,
		LookupKeys:    appsub.DefaultLookupKeys(),
		URLs: appsub.URLs{
			CheckoutReturn: "https://app.test/subscribe/return?session_id={CHECKOUT_SESSION_ID}",
			PortalReturn:   "https://app.test/billing",
		},
	})

	// Redis-cached reader, wrapped around the real app service.
	cachedReader := redisadapter.NewCachedSubscriptionReader(redisClient, subSvc, redisadapter.DefaultSubscriptionCacheTTL)

	paymentStripe := &e2ePaymentStripe{}
	paymentSvc := paymentapp.NewService(paymentapp.ServiceDeps{
		Records: paymentRepo,
		Users:   userRepo,
		Stripe:  paymentStripe,
	})
	paymentSvc.SetSubscriptionReader(cachedReader)

	ctx := context.Background()
	providerID := e2eInsertUser(t, db, "provider")
	clientID := e2eInsertUser(t, db, "enterprise")

	// ---- Step 1: free user → normal fee ----
	t.Log("Step 1: free user creates a payment intent, normal fee applies")
	milestone1 := e2eProposalFixture(t, db, clientID, providerID, 50000)
	out1, err := paymentSvc.CreatePaymentIntent(ctx, service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    milestone1,
		ClientID:       clientID,
		ProviderID:     providerID,
		ProposalAmount: 50000, // 500€, freelance tier 2 = 15€
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1500), out1.PlatformFee, "free user MUST pay the grid fee")

	// ---- Step 2: subscribe → checkout URL + no local row yet ----
	t.Log("Step 2: user subscribes, checkout URL returned")
	providerOrgID := e2eUserOrg(t, db, providerID)
	subOut, err := subSvc.Subscribe(ctx, appsub.SubscribeInput{
		OrganizationID: providerOrgID,
		ActorUserID:    providerID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
		AutoRenew:      false,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, subOut.ClientSecret)
	assert.Equal(t, stripeSub.createdClientSecret, subOut.ClientSecret)
	// No DB row yet — the webhook creates it.
	_, err = subRepo.FindOpenByOrganization(ctx, providerOrgID)
	assert.ErrorIs(t, err, domain.ErrNotFound, "no row until the webhook lands")

	// ---- Step 3: simulate customer.subscription.created webhook ----
	t.Log("Step 3: simulate webhook — subscription becomes active")
	now := time.Now().UTC().Truncate(time.Second)
	snap := service.SubscriptionSnapshot{
		ID:                "sub_e2e_" + providerID.String()[:8],
		Status:            "active",
		PriceID:           "price_premium_freelance_monthly",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
		CancelAtPeriodEnd: true,
	}
	err = subSvc.RegisterFromCheckout(
		ctx,
		providerOrgID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_e2e_"+providerID.String()[:8],
		snap,
	)
	require.NoError(t, err)

	// Cache must be flushed so IsActive sees the new state. In production
	// the webhook handler calls subscriptionCache.Invalidate explicitly;
	// here we do the same since we're invoking the app service directly.
	_ = cachedReader.Invalidate(ctx, providerID)

	// ---- Step 4: IsActive true + cache path ----
	t.Log("Step 4: verify Premium is active")
	active, err := cachedReader.IsActive(ctx, providerID)
	require.NoError(t, err)
	assert.True(t, active, "Premium MUST be active after webhook")

	// Second call hits Redis (warm cache).
	active2, _ := cachedReader.IsActive(ctx, providerID)
	assert.True(t, active2)

	// ---- Step 5: fee-waived payment record ----
	t.Log("Step 5: Premium user's payment record pays ZERO fee")
	milestone2 := e2eProposalFixture(t, db, clientID, providerID, 50000)
	out2, err := paymentSvc.CreatePaymentIntent(ctx, service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    milestone2,
		ClientID:       clientID,
		ProviderID:     providerID,
		ProposalAmount: 50000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), out2.PlatformFee, "Premium MUST waive the platform fee")
	assert.Equal(t, int64(50000), out2.ProviderPayout, "net payout = full amount")

	// DB row reflects it too.
	var feeInDB int64
	err = db.QueryRow(`SELECT platform_fee_amount FROM payment_records WHERE id = $1`, out2.PaymentRecordID).Scan(&feeInDB)
	require.NoError(t, err)
	assert.Equal(t, int64(0), feeInDB, "persisted fee MUST be zero")

	// ---- Step 6: toggle auto-renew on ----
	t.Log("Step 6: user toggles auto-renew on")
	updated, err := subSvc.ToggleAutoRenew(ctx, providerOrgID, true)
	require.NoError(t, err)
	assert.False(t, updated.CancelAtPeriodEnd, "auto_renew=true → cancel_at_period_end=false")

	// ---- Step 7: simulate subscription.deleted webhook ----
	t.Log("Step 7: simulate webhook — subscription canceled")
	err = subSvc.HandleSubscriptionSnapshot(ctx, service.SubscriptionSnapshot{
		ID: snap.ID,
	}, true)
	require.NoError(t, err)
	_ = cachedReader.Invalidate(ctx, providerID)

	// ---- Step 8: fee reapplied on a fresh payment record ----
	t.Log("Step 8: after cancel, fees reapply")
	active3, _ := cachedReader.IsActive(ctx, providerID)
	assert.False(t, active3, "Premium MUST be revoked after subscription.deleted")

	milestone3 := e2eProposalFixture(t, db, clientID, providerID, 50000)
	out3, err := paymentSvc.CreatePaymentIntent(ctx, service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    milestone3,
		ClientID:       clientID,
		ProviderID:     providerID,
		ProposalAmount: 50000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1500), out3.PlatformFee, "post-cancel MUST return to grid fee")
}
