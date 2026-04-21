package subscription_test

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/subscription"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Interface-satisfaction assertions. If a port grows a method, the
// compiler fails here — catches drift before it reaches runtime.
var (
	_ repository.SubscriptionRepository         = (*mockSubRepo)(nil)
	_ repository.UserRepository                 = (*mockUserRepo)(nil)
	_ repository.ProviderMilestoneAmountsReader = (*mockAmountsReader)(nil)
	_ service.StripeSubscriptionService         = (*mockStripe)(nil)
)

// --- mockSubRepo ---

type mockSubRepo struct {
	createFn         func(ctx context.Context, s *domain.Subscription) error
	findOpenByUserFn func(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	findByStripeIDFn func(ctx context.Context, stripeSubID string) (*domain.Subscription, error)
	updateFn         func(ctx context.Context, s *domain.Subscription) error
}

func (m *mockSubRepo) Create(ctx context.Context, s *domain.Subscription) error {
	if m.createFn != nil {
		return m.createFn(ctx, s)
	}
	return nil
}
func (m *mockSubRepo) FindOpenByUser(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	if m.findOpenByUserFn != nil {
		return m.findOpenByUserFn(ctx, userID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockSubRepo) FindByStripeID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	if m.findByStripeIDFn != nil {
		return m.findByStripeIDFn(ctx, stripeSubID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockSubRepo) Update(ctx context.Context, s *domain.Subscription) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, s)
	}
	return nil
}

// --- mockUserRepo — minimal implementation of the full UserRepository ---

type mockUserRepo struct {
	user *domainuser.User
	err  error
}

func (m *mockUserRepo) Create(_ context.Context, _ *domainuser.User) error { return nil }
func (m *mockUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*domainuser.User, error) {
	return m.user, m.err
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*domainuser.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Update(_ context.Context, _ *domainuser.User) error { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error        { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*domainuser.User, string, error) {
	return nil, "", nil
}
func (m *mockUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) CountByRole(_ context.Context) (map[string]int, error)   { return nil, nil }
func (m *mockUserRepo) CountByStatus(_ context.Context) (map[string]int, error) { return nil, nil }
func (m *mockUserRepo) RecentSignups(_ context.Context, _ int) ([]*domainuser.User, error) {
	return nil, nil
}
func (m *mockUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

// --- mockAmountsReader ---

type mockAmountsReader struct {
	amounts []int64
	err     error
}

func (m *mockAmountsReader) ListProviderMilestoneAmountsSince(
	_ context.Context, _ uuid.UUID, _ time.Time,
) ([]int64, error) {
	return m.amounts, m.err
}

// --- mockStripe ---

type mockStripe struct {
	ensureCustomerFn           func(ctx context.Context, userID, email, name string) (string, error)
	createCheckoutSessionFn    func(ctx context.Context, in service.CreateCheckoutSessionInput) (string, error)
	resolvePriceIDFn           func(ctx context.Context, lookupKey string) (string, error)
	updateCancelAtPeriodEndFn  func(ctx context.Context, stripeSubID string, cancelAtEnd bool) (service.SubscriptionSnapshot, error)
	changeCycleFn              func(ctx context.Context, stripeSubID, newPriceID string) (service.SubscriptionSnapshot, error)
	createPortalSessionFn      func(ctx context.Context, customerID, returnURL string) (string, error)

	lastCreateCheckoutInput *service.CreateCheckoutSessionInput // captured for assertions
}

func (m *mockStripe) EnsureCustomer(ctx context.Context, userID, email, name string) (string, error) {
	if m.ensureCustomerFn != nil {
		return m.ensureCustomerFn(ctx, userID, email, name)
	}
	return "cus_default", nil
}

func (m *mockStripe) CreateCheckoutSession(ctx context.Context, in service.CreateCheckoutSessionInput) (string, error) {
	// Capture for assertions
	copied := in
	m.lastCreateCheckoutInput = &copied
	if m.createCheckoutSessionFn != nil {
		return m.createCheckoutSessionFn(ctx, in)
	}
	return "https://checkout.stripe.test/" + in.PriceID, nil
}

func (m *mockStripe) ResolvePriceID(ctx context.Context, lookupKey string) (string, error) {
	if m.resolvePriceIDFn != nil {
		return m.resolvePriceIDFn(ctx, lookupKey)
	}
	return "price_" + lookupKey, nil
}

func (m *mockStripe) UpdateCancelAtPeriodEnd(ctx context.Context, stripeSubID string, cancelAtEnd bool) (service.SubscriptionSnapshot, error) {
	if m.updateCancelAtPeriodEndFn != nil {
		return m.updateCancelAtPeriodEndFn(ctx, stripeSubID, cancelAtEnd)
	}
	return service.SubscriptionSnapshot{
		ID:                stripeSubID,
		Status:            "active",
		CancelAtPeriodEnd: cancelAtEnd,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
	}, nil
}

func (m *mockStripe) ChangeCycle(ctx context.Context, stripeSubID, newPriceID string) (service.SubscriptionSnapshot, error) {
	if m.changeCycleFn != nil {
		return m.changeCycleFn(ctx, stripeSubID, newPriceID)
	}
	return service.SubscriptionSnapshot{
		ID:                stripeSubID,
		Status:            "active",
		PriceID:           newPriceID,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(365 * 24 * time.Hour),
	}, nil
}

func (m *mockStripe) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	if m.createPortalSessionFn != nil {
		return m.createPortalSessionFn(ctx, customerID, returnURL)
	}
	return "https://portal.stripe.test/" + customerID, nil
}
