package referral_test

// Manual mocks for the ports consumed by the referral app service. Each mock
// is a struct of function fields the test sets up per scenario. This is the
// project's manual-mock convention (see backend/mock/ for the broader style,
// but since these mocks live only inside the referral package tests we keep
// them alongside the test file).

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Compile-time interface checks.
var (
	_ repository.ReferralRepository = (*fakeReferralRepo)(nil)
	_ repository.UserRepository     = (*fakeUserRepo)(nil)
	_ service.MessageSender         = (*fakeMessageSender)(nil)
	_ service.NotificationSender    = (*fakeNotifier)(nil)
	_ service.StripeService         = (*fakeStripe)(nil)
	_ service.StripeTransferReversalService = (*fakeReversalService)(nil)
)

// fakeReferralRepo is an in-memory stand-in for repository.ReferralRepository.
type fakeReferralRepo struct {
	mu              sync.Mutex
	rows            map[uuid.UUID]*referral.Referral
	negotiations    []*referral.Negotiation
	attributions    map[uuid.UUID]*referral.Attribution // keyed by proposal_id
	attributionsByID map[uuid.UUID]*referral.Attribution // keyed by attribution id
	commissions     map[string]*referral.Commission // keyed by attribution_id:milestone_id
}

func newFakeReferralRepo() *fakeReferralRepo {
	return &fakeReferralRepo{
		rows:             make(map[uuid.UUID]*referral.Referral),
		attributions:     make(map[uuid.UUID]*referral.Attribution),
		attributionsByID: make(map[uuid.UUID]*referral.Attribution),
		commissions:      make(map[string]*referral.Commission),
	}
}

func (f *fakeReferralRepo) Create(ctx context.Context, r *referral.Referral) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.rows {
		if existing.ProviderID == r.ProviderID && existing.ClientID == r.ClientID && existing.Status.LocksCouple() {
			return referral.ErrCoupleLocked
		}
	}
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}

func (f *fakeReferralRepo) GetByID(ctx context.Context, id uuid.UUID) (*referral.Referral, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	if !ok {
		return nil, referral.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeReferralRepo) Update(ctx context.Context, r *referral.Referral) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.rows[r.ID]; !ok {
		return referral.ErrNotFound
	}
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}

func (f *fakeReferralRepo) FindActiveByCouple(ctx context.Context, providerID, clientID uuid.UUID) (*referral.Referral, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ProviderID == providerID && r.ClientID == clientID && r.Status.LocksCouple() {
			cp := *r
			return &cp, nil
		}
	}
	return nil, referral.ErrNotFound
}

func (f *fakeReferralRepo) ListByReferrer(ctx context.Context, referrerID uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Referral
	for _, r := range f.rows {
		if r.ReferrerID == referrerID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, "", nil
}

func (f *fakeReferralRepo) ListIncomingForProvider(ctx context.Context, providerID uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Referral
	for _, r := range f.rows {
		if r.ProviderID == providerID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, "", nil
}

func (f *fakeReferralRepo) ListIncomingForClient(ctx context.Context, clientID uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Referral
	for _, r := range f.rows {
		if r.ClientID == clientID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, "", nil
}

func (f *fakeReferralRepo) AppendNegotiation(ctx context.Context, n *referral.Negotiation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *n
	f.negotiations = append(f.negotiations, &cp)
	return nil
}

func (f *fakeReferralRepo) ListNegotiations(ctx context.Context, referralID uuid.UUID) ([]*referral.Negotiation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Negotiation
	for _, n := range f.negotiations {
		if n.ReferralID == referralID {
			cp := *n
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeReferralRepo) CreateAttribution(ctx context.Context, a *referral.Attribution) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.attributions[a.ProposalID]; exists {
		return nil // ON CONFLICT DO NOTHING semantics
	}
	cp := *a
	f.attributions[a.ProposalID] = &cp
	f.attributionsByID[a.ID] = &cp
	return nil
}

func (f *fakeReferralRepo) FindAttributionByProposal(ctx context.Context, proposalID uuid.UUID) (*referral.Attribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.attributions[proposalID]
	if !ok {
		return nil, referral.ErrAttributionNotFound
	}
	cp := *a
	return &cp, nil
}

func (f *fakeReferralRepo) FindAttributionByID(ctx context.Context, id uuid.UUID) (*referral.Attribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.attributionsByID[id]
	if !ok {
		return nil, referral.ErrAttributionNotFound
	}
	cp := *a
	return &cp, nil
}

func (f *fakeReferralRepo) ListAttributionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Attribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Attribution
	for _, a := range f.attributions {
		if a.ReferralID == referralID {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeReferralRepo) CreateCommission(ctx context.Context, c *referral.Commission) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s:%s", c.AttributionID, c.MilestoneID)
	if _, exists := f.commissions[key]; exists {
		return referral.ErrCommissionAlreadyExists
	}
	cp := *c
	f.commissions[key] = &cp
	return nil
}

func (f *fakeReferralRepo) UpdateCommission(ctx context.Context, c *referral.Commission) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s:%s", c.AttributionID, c.MilestoneID)
	if _, exists := f.commissions[key]; !exists {
		return referral.ErrCommissionNotFound
	}
	cp := *c
	f.commissions[key] = &cp
	return nil
}

func (f *fakeReferralRepo) FindCommissionByMilestone(ctx context.Context, milestoneID uuid.UUID) (*referral.Commission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.commissions {
		if c.MilestoneID == milestoneID {
			cp := *c
			return &cp, nil
		}
	}
	return nil, referral.ErrCommissionNotFound
}

func (f *fakeReferralRepo) ListCommissionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Commission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Commission
	for _, c := range f.commissions {
		if a, ok := f.attributionsByID[c.AttributionID]; ok && a.ReferralID == referralID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeReferralRepo) ListPendingKYCByReferrer(ctx context.Context, referrerID uuid.UUID) ([]*referral.Commission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Commission
	for _, c := range f.commissions {
		if c.Status != referral.CommissionPendingKYC {
			continue
		}
		a, ok := f.attributionsByID[c.AttributionID]
		if !ok {
			continue
		}
		if r, ok := f.rows[a.ReferralID]; ok && r.ReferrerID == referrerID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeReferralRepo) ListExpiringIntros(ctx context.Context, cutoff time.Time, limit int) ([]*referral.Referral, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Referral
	for _, r := range f.rows {
		if r.Status.IsPending() && r.LastActionAt.Before(cutoff) {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeReferralRepo) ListExpiringActives(ctx context.Context, now time.Time, limit int) ([]*referral.Referral, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*referral.Referral
	for _, r := range f.rows {
		if r.Status == referral.StatusActive && r.ExpiresAt != nil && r.ExpiresAt.Before(now) {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeReferralRepo) CountByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.Status]int, error) {
	return nil, nil
}

func (f *fakeReferralRepo) SumCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.CommissionStatus]int64, error) {
	return nil, nil
}

// The fake must satisfy both signatures of ListExpiring* — the repository
// interface uses time.Time, so we need matching signatures:
func (f *fakeReferralRepo) listExpiringIntrosReal() {}

// fakeUserRepo is a minimal user repository for role validation in tests.
type fakeUserRepo struct {
	users map[uuid.UUID]*user.User
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{users: map[uuid.UUID]*user.User{}} }

func (f *fakeUserRepo) add(id uuid.UUID, role user.Role, referrerEnabled bool) {
	f.users[id] = &user.User{ID: id, Role: role, ReferrerEnabled: referrerEnabled}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error { return nil }
func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}
func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) { return nil, nil }
func (f *fakeUserRepo) Update(ctx context.Context, u *user.User) error                   { return nil }
func (f *fakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error                   { return nil }
func (f *fakeUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error)    { return false, nil }
func (f *fakeUserRepo) ListAdmin(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (f *fakeUserRepo) CountAdmin(ctx context.Context, filters repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (f *fakeUserRepo) CountByRole(ctx context.Context) (map[string]int, error)   { return nil, nil }
func (f *fakeUserRepo) CountByStatus(ctx context.Context) (map[string]int, error) { return nil, nil }
func (f *fakeUserRepo) RecentSignups(ctx context.Context, limit int) ([]*user.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) BumpSessionVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (f *fakeUserRepo) GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (f *fakeUserRepo) UpdateEmailNotificationsEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return nil
}

// fakeMessageSender tracks system messages without doing anything else.
type fakeMessageSender struct {
	mu         sync.Mutex
	convsCreated []service.FindOrCreateConversationInput
	sysMessages  []service.SystemMessageInput
}

func (f *fakeMessageSender) FindOrCreateConversation(ctx context.Context, in service.FindOrCreateConversationInput) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.convsCreated = append(f.convsCreated, in)
	return uuid.New(), nil
}

func (f *fakeMessageSender) SendSystemMessage(ctx context.Context, in service.SystemMessageInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sysMessages = append(f.sysMessages, in)
	return nil
}

// fakeNotifier records all notifications.
type fakeNotifier struct {
	mu            sync.Mutex
	notifications []service.NotificationInput
}

func (f *fakeNotifier) Send(ctx context.Context, in service.NotificationInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifications = append(f.notifications, in)
	return nil
}

func (f *fakeNotifier) typeCount(t string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := 0
	for _, n := range f.notifications {
		if n.Type == t {
			c++
		}
	}
	return c
}

// fakeStripe tracks Stripe calls.
type fakeStripe struct {
	mu        sync.Mutex
	transfers []service.CreateTransferInput
	failNext  bool
}

func (f *fakeStripe) CreatePaymentIntent(ctx context.Context, input service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	return nil, nil
}
func (f *fakeStripe) CreateTransfer(ctx context.Context, input service.CreateTransferInput) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return "", fmt.Errorf("stripe boom")
	}
	f.transfers = append(f.transfers, input)
	return "tr_" + input.IdempotencyKey, nil
}
func (f *fakeStripe) ConstructWebhookEvent(payload []byte, signature string) (*service.StripeWebhookEvent, error) {
	return nil, nil
}
func (f *fakeStripe) GetAccount(ctx context.Context, accountID string) (*service.StripeAccountInfo, error) {
	return nil, nil
}
func (f *fakeStripe) CreateRefund(ctx context.Context, paymentIntentID string, amount int64) (string, error) {
	return "", nil
}

// fakeReversalService tracks reversal calls.
type fakeReversalService struct {
	mu       sync.Mutex
	reversals []service.CreateTransferReversalInput
}

func (f *fakeReversalService) CreateTransferReversal(ctx context.Context, in service.CreateTransferReversalInput) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reversals = append(f.reversals, in)
	return "trr_" + in.IdempotencyKey, nil
}

// fakeSnapshotLoader returns a canned snapshot.
type fakeSnapshotLoader struct {
	provider referral.ProviderSnapshot
	client   referral.ClientSnapshot
}

func (f *fakeSnapshotLoader) LoadProvider(ctx context.Context, userID uuid.UUID) (referral.ProviderSnapshot, error) {
	return f.provider, nil
}

func (f *fakeSnapshotLoader) LoadClient(ctx context.Context, userID uuid.UUID) (referral.ClientSnapshot, error) {
	return f.client, nil
}

// fakeStripeAccountResolver returns the pre-set account id for every user.
type fakeStripeAccountResolver struct {
	accountID string
}

func (f *fakeStripeAccountResolver) ResolveStripeAccountID(ctx context.Context, userID uuid.UUID) (string, error) {
	return f.accountID, nil
}

// _ unused to silence "imported but not used" in the test package when a test
// doesn't touch all helpers.
var _ = json.Marshal
