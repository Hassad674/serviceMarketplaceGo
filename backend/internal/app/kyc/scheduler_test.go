package kyc

// Unit tests for the KYC enforcement scheduler. The scheduler walks
// every org in `kyc_pending` and emits a tier notification (day0, day3,
// day7, day14) based on how long ago the org first earned funds.
//
// Coverage targets:
//   - tick(): all four tier branches (day0/3/7/14) including the
//     restriction tier crossing 14 days
//   - dedupe via KYCRestrictionNotifiedAt persisted state
//   - computePendingAmount aggregating only succeeded+pending payment
//     records for the org
//   - Run() honoring ctx.Done — must return cleanly without leaking a
//     goroutine
//   - mustJSON edge cases (nil, primitives, nested map)
//   - tick gracefully handling repo and send errors

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// ─── mocks ────────────────────────────────────────────────────────────

type kycMockOrgRepo struct {
	mu sync.Mutex

	listKYCPendingFn          func(ctx context.Context) ([]*organization.Organization, error)
	saveKYCNotificationFn     func(ctx context.Context, orgID uuid.UUID, state map[string]time.Time) error
	saveKYCCalls              []saveKYCNotifCall
}

type saveKYCNotifCall struct {
	OrgID uuid.UUID
	State map[string]time.Time
}

// kycMockOrgRepo implements the narrowed scheduler dependency
// (OrganizationReader + OrganizationStripeStore). The legacy 22-method
// stub was shrunk to the 13 methods the two segregated children
// actually expose — every dropped method was unused by the scheduler
// and only there to satisfy the wide port.
var (
	_ repository.OrganizationReader      = (*kycMockOrgRepo)(nil)
	_ repository.OrganizationStripeStore = (*kycMockOrgRepo)(nil)
)

func (m *kycMockOrgRepo) FindByID(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
	return &organization.Organization{ID: id}, nil
}
func (m *kycMockOrgRepo) FindByOwnerUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (m *kycMockOrgRepo) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (m *kycMockOrgRepo) CountAll(_ context.Context) (int, error) { return 0, nil }
func (m *kycMockOrgRepo) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	return nil, nil
}
func (m *kycMockOrgRepo) ListKYCPending(ctx context.Context) ([]*organization.Organization, error) {
	if m.listKYCPendingFn != nil {
		return m.listKYCPendingFn(ctx)
	}
	return nil, nil
}
func (m *kycMockOrgRepo) ListWithStripeAccount(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *kycMockOrgRepo) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *kycMockOrgRepo) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *kycMockOrgRepo) SetStripeAccount(_ context.Context, _ uuid.UUID, _ string, _ string) error {
	return nil
}
func (m *kycMockOrgRepo) ClearStripeAccount(_ context.Context, _ uuid.UUID) error { return nil }
func (m *kycMockOrgRepo) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *kycMockOrgRepo) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *kycMockOrgRepo) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *kycMockOrgRepo) SaveKYCNotificationState(ctx context.Context, orgID uuid.UUID, state map[string]time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make(map[string]time.Time, len(state))
	for k, v := range state {
		cp[k] = v
	}
	m.saveKYCCalls = append(m.saveKYCCalls, saveKYCNotifCall{OrgID: orgID, State: cp})
	if m.saveKYCNotificationFn != nil {
		return m.saveKYCNotificationFn(ctx, orgID, state)
	}
	return nil
}

func (m *kycMockOrgRepo) snapshotSaveKYCCalls() []saveKYCNotifCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]saveKYCNotifCall, len(m.saveKYCCalls))
	copy(out, m.saveKYCCalls)
	return out
}

type kycMockRecordRepo struct {
	listByOrgFn func(ctx context.Context, orgID uuid.UUID) ([]*payment.PaymentRecord, error)
}

var _ repository.PaymentRecordRepository = (*kycMockRecordRepo)(nil)

func (m *kycMockRecordRepo) Create(_ context.Context, _ *payment.PaymentRecord) error { return nil }
func (m *kycMockRecordRepo) GetByID(_ context.Context, _ uuid.UUID) (*payment.PaymentRecord, error) {
	return nil, nil
}
func (m *kycMockRecordRepo) GetByIDForOrg(_ context.Context, _, _ uuid.UUID) (*payment.PaymentRecord, error) {
	return nil, nil
}
func (m *kycMockRecordRepo) GetByProposalID(_ context.Context, _ uuid.UUID) (*payment.PaymentRecord, error) {
	return nil, nil
}
func (m *kycMockRecordRepo) ListByProposalID(_ context.Context, _ uuid.UUID) ([]*payment.PaymentRecord, error) {
	return nil, nil
}
func (m *kycMockRecordRepo) GetByMilestoneID(_ context.Context, _ uuid.UUID) (*payment.PaymentRecord, error) {
	return nil, nil
}
func (m *kycMockRecordRepo) GetByPaymentIntentID(_ context.Context, _ string) (*payment.PaymentRecord, error) {
	return nil, nil
}
func (m *kycMockRecordRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*payment.PaymentRecord, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, orgID)
	}
	return nil, nil
}
func (m *kycMockRecordRepo) Update(_ context.Context, _ *payment.PaymentRecord) error { return nil }

type kycMockNotifier struct {
	mu     sync.Mutex
	sent   []portservice.NotificationInput
	sendFn func(ctx context.Context, input portservice.NotificationInput) error
}

var _ portservice.NotificationSender = (*kycMockNotifier)(nil)

func (m *kycMockNotifier) Send(ctx context.Context, input portservice.NotificationInput) error {
	m.mu.Lock()
	m.sent = append(m.sent, input)
	m.mu.Unlock()
	if m.sendFn != nil {
		return m.sendFn(ctx, input)
	}
	return nil
}

func (m *kycMockNotifier) snapshot() []portservice.NotificationInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]portservice.NotificationInput, len(m.sent))
	copy(out, m.sent)
	return out
}

// ─── tests ────────────────────────────────────────────────────────────

func makeOrg(id, ownerID uuid.UUID, daysAgo int) *organization.Organization {
	earned := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)
	return &organization.Organization{
		ID:                       id,
		OwnerUserID:              ownerID,
		Type:                     organization.OrgTypeProviderPersonal,
		KYCFirstEarningAt:        &earned,
		KYCRestrictionNotifiedAt: map[string]time.Time{},
	}
}

func TestNewScheduler_WiresDeps(t *testing.T) {
	orgs := &kycMockOrgRepo{}
	records := &kycMockRecordRepo{}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       records,
		Notifications: notifier,
	})
	require.NotNil(t, s)
}

func TestScheduler_Tick_NoOrgs_NoOp(t *testing.T) {
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return nil, nil
		},
	}
	records := &kycMockRecordRepo{}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{Organizations: orgs, Records: records, Notifications: notifier})
	s.tick(context.Background())
	assert.Empty(t, notifier.snapshot())
}

func TestScheduler_Tick_ListErrorIsLoggedAndSwallowed(t *testing.T) {
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return nil, errors.New("db down")
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: notifier,
	})
	// Must not panic; just early-returns.
	s.tick(context.Background())
	assert.Empty(t, notifier.snapshot())
}

func TestScheduler_Tick_Day0_FirstNotification(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{makeOrg(orgID, ownerID, 0)}, nil
		},
	}
	records := &kycMockRecordRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*payment.PaymentRecord, error) {
			return nil, nil
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{Organizations: orgs, Records: records, Notifications: notifier})

	s.tick(context.Background())

	sent := notifier.snapshot()
	require.Len(t, sent, 1, "day0 must fire for a freshly-pending org")
	assert.Equal(t, ownerID, sent[0].UserID, "notification target is the org owner")
	assert.Equal(t, string(notifdomain.TypeKYCReminder), sent[0].Type)

	saveCalls := orgs.snapshotSaveKYCCalls()
	require.Len(t, saveCalls, 1)
	assert.Equal(t, orgID, saveCalls[0].OrgID)
	assert.Contains(t, saveCalls[0].State, "day0",
		"persisted state must include the just-fired tier so the next tick dedupes")
}

func TestScheduler_Tick_Day3_FiresBothDay0AndDay3(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{makeOrg(orgID, ownerID, 4)}, nil
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: notifier,
	})

	s.tick(context.Background())

	// At 4 days elapsed, both day0 (>= 0) and day3 (>= 3) tiers should fire.
	sent := notifier.snapshot()
	require.Len(t, sent, 2, "day0 and day3 fire together on first observation at 4 days")
}

func TestScheduler_Tick_Day14_RestrictionType(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{makeOrg(orgID, ownerID, 15)}, nil
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: notifier,
	})

	s.tick(context.Background())

	// All four tiers fire on first observation (cold-start).
	sent := notifier.snapshot()
	require.Len(t, sent, 4)
	// The day14 tier carries the RESTRICTION type — money path: this is
	// the boundary that flips the org wallet to "blocked".
	last := sent[len(sent)-1]
	assert.Equal(t, string(notifdomain.TypeKYCRestriction), last.Type,
		"day14 tier must use TypeKYCRestriction so the wallet middleware blocks new actions")
}

func TestScheduler_Tick_DedupesAlreadyNotifiedTiers(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	org := makeOrg(orgID, ownerID, 4)
	// Already notified day0 and day3 — only day7+day14 should be relevant
	// (but the org is at 4 days, so neither qualifies).
	org.KYCRestrictionNotifiedAt = map[string]time.Time{
		"day0": time.Now().Add(-time.Hour),
		"day3": time.Now().Add(-time.Minute),
	}
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{org}, nil
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: notifier,
	})

	s.tick(context.Background())
	sent := notifier.snapshot()
	assert.Empty(t, sent, "tiers already notified must not re-fire")
}

func TestScheduler_Tick_NilFirstEarning_Skipped(t *testing.T) {
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{
				{ID: uuid.New(), OwnerUserID: uuid.New()},
			}, nil
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: notifier,
	})
	s.tick(context.Background())
	assert.Empty(t, notifier.snapshot())
}

func TestScheduler_Tick_NotificationFailure_DoesNotMarkAsSent(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{makeOrg(orgID, ownerID, 0)}, nil
		},
	}
	notifier := &kycMockNotifier{
		sendFn: func(_ context.Context, _ portservice.NotificationInput) error {
			return errors.New("fcm down")
		},
	}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: notifier,
	})

	s.tick(context.Background())

	saveCalls := orgs.snapshotSaveKYCCalls()
	assert.Empty(t, saveCalls, "if no tier fired successfully, no state should be persisted")
}

func TestScheduler_Tick_BodyIncludesAmountWhenPositive(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return []*organization.Organization{makeOrg(orgID, ownerID, 0)}, nil
		},
	}
	records := &kycMockRecordRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*payment.PaymentRecord, error) {
			return []*payment.PaymentRecord{
				{Status: payment.RecordStatusSucceeded, TransferStatus: payment.TransferPending, ProviderPayout: 5000_00},
			}, nil
		},
	}
	notifier := &kycMockNotifier{}
	s := NewScheduler(SchedulerDeps{Organizations: orgs, Records: records, Notifications: notifier})
	s.tick(context.Background())

	sent := notifier.snapshot()
	require.NotEmpty(t, sent)
	assert.Contains(t, sent[0].Body, "5000€", "body must include the pending €amount when > 0")
}

// ─── computePendingAmount ─────────────────────────────────────────────

func TestComputePendingAmount_OnlyCountsSucceededAndPending(t *testing.T) {
	orgID := uuid.New()
	records := &kycMockRecordRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*payment.PaymentRecord, error) {
			return []*payment.PaymentRecord{
				{Status: payment.RecordStatusSucceeded, TransferStatus: payment.TransferPending, ProviderPayout: 1000},
				{Status: payment.RecordStatusSucceeded, TransferStatus: payment.TransferCompleted, ProviderPayout: 2000}, // already paid out
				{Status: payment.RecordStatusFailed, TransferStatus: payment.TransferPending, ProviderPayout: 3000},      // failed
				{Status: payment.RecordStatusSucceeded, TransferStatus: payment.TransferPending, ProviderPayout: 4000},
			}, nil
		},
	}
	s := NewScheduler(SchedulerDeps{Records: records})
	got := s.computePendingAmount(context.Background(), orgID)
	assert.Equal(t, int64(5000), got, "only succeeded+pending rows must contribute (1000 + 4000 = 5000)")
}

func TestComputePendingAmount_RepoError_ReturnsZero(t *testing.T) {
	records := &kycMockRecordRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*payment.PaymentRecord, error) {
			return nil, errors.New("db")
		},
	}
	s := NewScheduler(SchedulerDeps{Records: records})
	got := s.computePendingAmount(context.Background(), uuid.New())
	assert.Zero(t, got, "repo error must degrade to 0, never panic")
}

func TestComputePendingAmount_EmptyList(t *testing.T) {
	records := &kycMockRecordRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*payment.PaymentRecord, error) {
			return nil, nil
		},
	}
	s := NewScheduler(SchedulerDeps{Records: records})
	got := s.computePendingAmount(context.Background(), uuid.New())
	assert.Zero(t, got)
}

// ─── mustJSON ─────────────────────────────────────────────────────────

func TestMustJSON_PrimitiveMap(t *testing.T) {
	out := mustJSON(map[string]any{"a": 1, "b": "x"})
	require.NotEmpty(t, out)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))
	assert.EqualValues(t, 1, decoded["a"])
	assert.Equal(t, "x", decoded["b"])
}

func TestMustJSON_Nil(t *testing.T) {
	out := mustJSON(nil)
	assert.Equal(t, "null", string(out))
}

// ─── Run + ctx cancellation ───────────────────────────────────────────

func TestScheduler_Run_StopsOnContextCancel(t *testing.T) {
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			return nil, nil
		},
	}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: &kycMockNotifier{},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Let the immediate tick run.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// ok — Run returned cleanly on ctx cancellation
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop within 1s of context cancellation — goroutine leak")
	}
}

func TestScheduler_Run_TicksOnInterval(t *testing.T) {
	var ticks int
	var mu sync.Mutex
	orgs := &kycMockOrgRepo{
		listKYCPendingFn: func(_ context.Context) ([]*organization.Organization, error) {
			mu.Lock()
			ticks++
			mu.Unlock()
			return nil, nil
		},
	}
	s := NewScheduler(SchedulerDeps{
		Organizations: orgs,
		Records:       &kycMockRecordRepo{},
		Notifications: &kycMockNotifier{},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	s.Run(ctx, 20*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, ticks, 2,
		"Run must fire one immediate tick + at least one ticker tick within 60ms at 20ms interval")
}
