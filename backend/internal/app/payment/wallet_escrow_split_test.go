package payment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	milestonedomain "marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/payment"
)

// ─── Stubs ────────────────────────────────────────────────────────────

// stubMilestoneStatusReader implements MilestoneStatusReader. Records
// every call so the batching contract can be asserted.
type stubMilestoneStatusReader struct {
	mu       sync.Mutex
	statuses map[uuid.UUID]milestonedomain.MilestoneStatus
	calls    int
	lastIDs  []uuid.UUID
	err      error
}

func (s *stubMilestoneStatusReader) StatusByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]milestonedomain.MilestoneStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.lastIDs = append([]uuid.UUID(nil), ids...)
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[uuid.UUID]milestonedomain.MilestoneStatus, len(ids))
	for _, id := range ids {
		if v, ok := s.statuses[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

// ─── classifyRecordBucket — pure table-driven matrix ─────────────────

// TestClassifyRecordBucket_Matrix exhaustively covers every relevant
// combination of (payment_status, transfer_status, milestone.status)
// to lock the dispatch contract in. ≥ 8 cases as required by the brief.
func TestClassifyRecordBucket_Matrix(t *testing.T) {
	milestoneID := uuid.New()
	tests := []struct {
		name         string
		paymentStatus  domain.PaymentRecordStatus
		transferStatus domain.TransferStatus
		milestoneSt    milestonedomain.MilestoneStatus
		omitStatus     bool // when true, the milestone id is NOT in the lookup map
		nilMilestoneID bool // when true, MilestoneID = uuid.Nil
		wantBucket     recordBucket
	}{
		{
			name:           "transfer completed → transferred (regardless of milestone)",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferCompleted,
			milestoneSt:    milestonedomain.StatusReleased,
			wantBucket:     bucketTransferred,
		},
		{
			name:           "transfer completed with submitted milestone — defensive transferred",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferCompleted,
			milestoneSt:    milestonedomain.StatusSubmitted,
			wantBucket:     bucketTransferred,
		},
		{
			name:           "succeeded+pending+funded → escrow",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusFunded,
			wantBucket:     bucketEscrow,
		},
		{
			name:           "succeeded+pending+submitted → escrow",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusSubmitted,
			wantBucket:     bucketEscrow,
		},
		{
			name:           "succeeded+pending+disputed → escrow",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusDisputed,
			wantBucket:     bucketEscrow,
		},
		{
			name:           "succeeded+pending+approved → available (the headline case)",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusApproved,
			wantBucket:     bucketAvailable,
		},
		{
			// Volet 3 regression guard: client completed the mission
			// (Approve→Release) but provider KYC/billing is incomplete
			// so the auto-transfer was deferred. transfer_status stays
			// pending → money is drainable manually, NOT transferred.
			// Must be Available (Disponible), never Transferred.
			name:           "succeeded+pending+released → available (transfer deferred, KYC/billing incomplete)",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusReleased,
			wantBucket:     bucketAvailable,
		},
		{
			name:           "succeeded+pending+pending_funding → skip (data corruption)",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusPendingFunding,
			wantBucket:     bucketSkip,
		},
		{
			name:           "succeeded+pending+cancelled → skip",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusCancelled,
			wantBucket:     bucketSkip,
		},
		{
			name:           "succeeded+pending+refunded → skip",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusRefunded,
			wantBucket:     bucketSkip,
		},
		{
			name:           "succeeded+pending with missing status → escrow (conservative)",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			omitStatus:     true,
			wantBucket:     bucketEscrow,
		},
		{
			name:           "succeeded+pending with nil milestone id → escrow",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferPending,
			nilMilestoneID: true,
			wantBucket:     bucketEscrow,
		},
		{
			name:           "pending payment → skip",
			paymentStatus:  domain.RecordStatusPending,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusFunded,
			wantBucket:     bucketSkip,
		},
		{
			name:           "failed payment → skip",
			paymentStatus:  domain.RecordStatusFailed,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusFunded,
			wantBucket:     bucketSkip,
		},
		{
			name:           "refunded payment → skip",
			paymentStatus:  domain.RecordStatusRefunded,
			transferStatus: domain.TransferPending,
			milestoneSt:    milestonedomain.StatusApproved,
			wantBucket:     bucketSkip,
		},
		{
			name:           "transfer failed status → skip (not yet completed, not the happy path)",
			paymentStatus:  domain.RecordStatusSucceeded,
			transferStatus: domain.TransferFailed,
			milestoneSt:    milestonedomain.StatusApproved,
			wantBucket:     bucketSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &domain.PaymentRecord{
				ID:             uuid.New(),
				ProposalID:     uuid.New(),
				MilestoneID:    milestoneID,
				ProviderPayout: 100,
				Status:         tt.paymentStatus,
				TransferStatus: tt.transferStatus,
			}
			if tt.nilMilestoneID {
				r.MilestoneID = uuid.Nil
			}
			statuses := map[uuid.UUID]milestonedomain.MilestoneStatus{}
			if !tt.omitStatus && !tt.nilMilestoneID {
				statuses[milestoneID] = tt.milestoneSt
			}
			got := classifyRecordBucket(r, statuses)
			assert.Equal(t, tt.wantBucket, got)
		})
	}
}

// ─── Integration: GetWalletOverview with milestone reader ─────────────

// TestWalletList_EscrowVsAvailable_Split — the integration matrix the
// brief asks for. Records exercise every bucket and the totals must
// match the sum of buckets without double-count.
func TestWalletList_EscrowVsAvailable_Split(t *testing.T) {
	mFunded := uuid.New()
	mSubmitted := uuid.New()
	mDisputed := uuid.New()
	mApproved1 := uuid.New()
	mApproved2 := uuid.New()
	mReleased := uuid.New()
	now := time.Now()

	rec := func(milestoneID uuid.UUID, status domain.PaymentRecordStatus, transfer domain.TransferStatus, payout int64) *domain.PaymentRecord {
		return &domain.PaymentRecord{
			ID:             uuid.New(),
			ProposalID:     uuid.New(),
			MilestoneID:    milestoneID,
			Status:         status,
			TransferStatus: transfer,
			ProposalAmount: payout + 50,
			ProviderPayout: payout,
			CreatedAt:      now,
		}
	}

	records := []*domain.PaymentRecord{
		rec(mFunded, domain.RecordStatusSucceeded, domain.TransferPending, 100),    // escrow
		rec(mSubmitted, domain.RecordStatusSucceeded, domain.TransferPending, 200), // escrow
		rec(mDisputed, domain.RecordStatusSucceeded, domain.TransferPending, 300),  // escrow
		rec(mApproved1, domain.RecordStatusSucceeded, domain.TransferPending, 400), // available
		rec(mApproved2, domain.RecordStatusSucceeded, domain.TransferPending, 500), // available
		rec(mReleased, domain.RecordStatusSucceeded, domain.TransferCompleted, 1000), // transferred
	}
	statuses := map[uuid.UUID]milestonedomain.MilestoneStatus{
		mFunded:    milestonedomain.StatusFunded,
		mSubmitted: milestonedomain.StatusSubmitted,
		mDisputed:  milestonedomain.StatusDisputed,
		mApproved1: milestonedomain.StatusApproved,
		mApproved2: milestonedomain.StatusApproved,
		mReleased:  milestonedomain.StatusReleased,
	}
	reader := &stubMilestoneStatusReader{statuses: statuses}

	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: records},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{stripeAccountID: "acct_test"},
		Stripe:        &walletStubStripe{},
	})
	wallet.SetMilestoneStatusReader(reader)

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, ov)

	// Escrow buckets sum to 100+200+300 = 600
	assert.Equal(t, int64(600), ov.EscrowAmount, "funded+submitted+disputed must sum into escrow")
	// Available buckets sum to 400+500 = 900
	assert.Equal(t, int64(900), ov.AvailableAmount, "approved milestones must sum into available")
	// Transferred = 1000
	assert.Equal(t, int64(1000), ov.TransferredAmount)
	// Crucially, escrow != available — the bug we're fixing.
	assert.NotEqual(t, ov.EscrowAmount, ov.AvailableAmount,
		"escrow and available must be computed independently")
	// And the sum (no double-count): 600+900+1000 = 2500.
	total := ov.EscrowAmount + ov.AvailableAmount + ov.TransferredAmount
	assert.Equal(t, int64(2500), total, "sum equals the legitimate total — no double-count")
}

// TestWalletList_BatchedMilestoneFetch proves there is no N+1 query:
// 10 records with distinct milestone ids must trigger EXACTLY ONE
// StatusByIDs call. Regression pin — a future refactor that loops
// per-record will trip this test loudly.
func TestWalletList_BatchedMilestoneFetch(t *testing.T) {
	const N = 10
	records := make([]*domain.PaymentRecord, 0, N)
	statuses := map[uuid.UUID]milestonedomain.MilestoneStatus{}
	for i := 0; i < N; i++ {
		mid := uuid.New()
		records = append(records, &domain.PaymentRecord{
			ID:             uuid.New(),
			ProposalID:     uuid.New(),
			MilestoneID:    mid,
			Status:         domain.RecordStatusSucceeded,
			TransferStatus: domain.TransferPending,
			ProposalAmount: 100,
			ProviderPayout: 100,
			CreatedAt:      time.Now(),
		})
		// Mix approved + funded — exercises both buckets in a single batch.
		if i%2 == 0 {
			statuses[mid] = milestonedomain.StatusApproved
		} else {
			statuses[mid] = milestonedomain.StatusFunded
		}
	}
	reader := &stubMilestoneStatusReader{statuses: statuses}

	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: records},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	wallet.SetMilestoneStatusReader(reader)

	_, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)

	assert.Equal(t, 1, reader.calls, "milestone batch lookup must run EXACTLY ONCE — N+1 forbidden")
	assert.Len(t, reader.lastIDs, N, "the single batch carried every distinct milestone id")
}

// TestWalletList_BatchedMilestoneFetch_DedupesIDs — two records on the
// SAME milestone id (one-time proposal with retry record etc.) must
// not trigger a duplicated id in the batch payload.
func TestWalletList_BatchedMilestoneFetch_DedupesIDs(t *testing.T) {
	mid := uuid.New()
	records := []*domain.PaymentRecord{
		{
			ID: uuid.New(), ProposalID: uuid.New(), MilestoneID: mid,
			Status: domain.RecordStatusSucceeded, TransferStatus: domain.TransferPending,
			ProviderPayout: 100, CreatedAt: time.Now(),
		},
		{
			ID: uuid.New(), ProposalID: uuid.New(), MilestoneID: mid,
			Status: domain.RecordStatusSucceeded, TransferStatus: domain.TransferPending,
			ProviderPayout: 50, CreatedAt: time.Now(),
		},
	}
	reader := &stubMilestoneStatusReader{
		statuses: map[uuid.UUID]milestonedomain.MilestoneStatus{
			mid: milestonedomain.StatusApproved,
		},
	}
	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: records},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	wallet.SetMilestoneStatusReader(reader)

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)

	assert.Equal(t, 1, reader.calls)
	require.Len(t, reader.lastIDs, 1, "duplicate milestone ids must be deduped in the batch")
	assert.Equal(t, mid, reader.lastIDs[0])
	assert.Equal(t, int64(150), ov.AvailableAmount, "both records route to available via the same approval")
}

// TestWalletService_GetWalletOverview_NoMilestoneReader_DegradesToEscrowOnly
// pins the conservative-default contract: when no MilestoneStatusReader
// is wired, every paid+pending record stays in escrow and AvailableAmount
// is zero. We MUST NEVER silently mark unverified funds as drainable —
// that would let a provider drain escrow funds via the withdraw endpoint.
func TestWalletService_GetWalletOverview_NoMilestoneReader_DegradesToEscrowOnly(t *testing.T) {
	r := &domain.PaymentRecord{
		ID: uuid.New(), ProposalID: uuid.New(), MilestoneID: uuid.New(),
		Status: domain.RecordStatusSucceeded, TransferStatus: domain.TransferPending,
		ProviderPayout: 12345, CreatedAt: time.Now(),
	}
	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: []*domain.PaymentRecord{r}},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	// SetMilestoneStatusReader is NOT called — production must boot
	// safely in that mode (worktrees without the proposal feature
	// wired).

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(12345), ov.EscrowAmount)
	assert.Zero(t, ov.AvailableAmount, "no milestone reader → AvailableAmount MUST be zero")
}

// TestWalletService_GetWalletOverview_MilestoneReaderError_DegradesToEscrow
// is the soft-fail contract: if the batch call errors (DB blip, RLS
// misconfig), the wallet must NOT 500 — it falls back to the
// conservative escrow-only branch and surfaces a slog warning. The
// withdraw endpoint then sees available=0 and refuses to drain — safe
// degradation.
func TestWalletService_GetWalletOverview_MilestoneReaderError_DegradesToEscrow(t *testing.T) {
	mid := uuid.New()
	r := &domain.PaymentRecord{
		ID: uuid.New(), ProposalID: uuid.New(), MilestoneID: mid,
		Status: domain.RecordStatusSucceeded, TransferStatus: domain.TransferPending,
		ProviderPayout: 200, CreatedAt: time.Now(),
	}
	reader := &stubMilestoneStatusReader{err: errors.New("milestone db blip")}
	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: []*domain.PaymentRecord{r}},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	wallet.SetMilestoneStatusReader(reader)

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err, "milestone reader errors MUST NOT take down the wallet")
	assert.Equal(t, int64(200), ov.EscrowAmount)
	assert.Zero(t, ov.AvailableAmount, "fail-soft: every paid+pending record falls into escrow")
}

// TestWalletService_SetMilestoneStatusReader_Service ensures the
// parent-level setter on *Service correctly threads through to the
// wallet sub-service.
func TestWalletService_SetMilestoneStatusReader_Service(t *testing.T) {
	svc := NewService(ServiceDeps{
		Records:       &walletStubRecords{},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	reader := &stubMilestoneStatusReader{statuses: map[uuid.UUID]milestonedomain.MilestoneStatus{}}
	svc.SetMilestoneStatusReader(reader)
	assert.Same(t, reader, svc.Wallet().milestones,
		"parent setter must thread the reader into the wallet sub-service")
}

// TestWalletService_GetWalletOverview_ApprovedRecord_RoutesToAvailable
// is the headline regression test — proves the user-reported bug
// (escrow == available) is fixed for the simplest possible scenario.
func TestWalletService_GetWalletOverview_ApprovedRecord_RoutesToAvailable(t *testing.T) {
	mid := uuid.New()
	r := &domain.PaymentRecord{
		ID: uuid.New(), ProposalID: uuid.New(), MilestoneID: mid,
		Status: domain.RecordStatusSucceeded, TransferStatus: domain.TransferPending,
		ProviderPayout: 31207_00, // matches the user's screenshot
		CreatedAt:      time.Now(),
	}
	reader := &stubMilestoneStatusReader{statuses: map[uuid.UUID]milestonedomain.MilestoneStatus{
		mid: milestonedomain.StatusApproved,
	}}
	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: []*domain.PaymentRecord{r}},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	wallet.SetMilestoneStatusReader(reader)

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(31207_00), ov.AvailableAmount, "approved milestone → available")
	assert.Zero(t, ov.EscrowAmount, "approved milestone must NOT also appear in escrow (the bug)")
}

// TestWalletService_GetWalletOverview_FundedRecord_RoutesToEscrow
// is the dual: a funded (paid but not yet client-approved) milestone
// stays in escrow and never leaks into available.
func TestWalletService_GetWalletOverview_FundedRecord_RoutesToEscrow(t *testing.T) {
	mid := uuid.New()
	r := &domain.PaymentRecord{
		ID: uuid.New(), ProposalID: uuid.New(), MilestoneID: mid,
		Status: domain.RecordStatusSucceeded, TransferStatus: domain.TransferPending,
		ProviderPayout: 5000_00,
		CreatedAt:      time.Now(),
	}
	reader := &stubMilestoneStatusReader{statuses: map[uuid.UUID]milestonedomain.MilestoneStatus{
		mid: milestonedomain.StatusFunded,
	}}
	wallet := NewWalletService(WalletServiceDeps{
		Records:       &walletStubRecords{rows: []*domain.PaymentRecord{r}},
		Users:         &walletStubUsers{},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	wallet.SetMilestoneStatusReader(reader)

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(5000_00), ov.EscrowAmount)
	assert.Zero(t, ov.AvailableAmount, "funded milestone is NOT retire-eligible")
}
