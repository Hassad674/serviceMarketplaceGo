package payment

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// ChargeService — dedicated tests proving the PaymentIntent lifecycle
// sub-service is independently testable. The fee-calculator dependency
// is exercised via a tiny inline stub (no need for a full WalletService).
// ---------------------------------------------------------------------------

// stubFeeCalc is the minimal platformFeeCalculator used by ChargeService
// tests. Two methods required, but the interface only has one.
type stubFeeCalc struct {
	fee int64
	err error
}

func (s *stubFeeCalc) computePlatformFee(_ context.Context, _ uuid.UUID, _ int64) (int64, error) {
	return s.fee, s.err
}

// chargeStubRecords is the focused store for charge tests. It supports
// the four methods the lifecycle actually uses. Thread-safe (mutex-
// guarded counters) so the same fixture can drive race tests without
// false positives from concurrent counter writes.
type chargeStubRecords struct {
	repository.PaymentRecordRepository
	mu             sync.Mutex
	byMilestone    *domain.PaymentRecord
	byProposal     *domain.PaymentRecord
	byPaymentInt   *domain.PaymentRecord
	byMilestoneErr error
	createCalls    int
	updateCalls    int
	updateErr      error
	createdRec     *domain.PaymentRecord
	updatedRec     *domain.PaymentRecord
}

func (c *chargeStubRecords) GetByMilestoneID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byMilestoneErr != nil {
		return nil, c.byMilestoneErr
	}
	if c.byMilestone == nil {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *c.byMilestone
	return &cp, nil
}

func (c *chargeStubRecords) GetByProposalID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byProposal == nil {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *c.byProposal
	return &cp, nil
}

func (c *chargeStubRecords) GetByPaymentIntentID(_ context.Context, _ string) (*domain.PaymentRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byPaymentInt == nil {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *c.byPaymentInt
	return &cp, nil
}

func (c *chargeStubRecords) Create(_ context.Context, r *domain.PaymentRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.createCalls++
	cp := *r
	c.createdRec = &cp
	return nil
}

func (c *chargeStubRecords) Update(_ context.Context, r *domain.PaymentRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updateCalls++
	if c.updateErr != nil {
		return c.updateErr
	}
	cp := *r
	c.updatedRec = &cp
	return nil
}

// chargeStubStripe captures every CreatePaymentIntent / GetPaymentIntent
// / ConstructWebhookEvent call made by the charge service. Thread-safe
// for race tests.
type chargeStubStripe struct {
	service.StripeService
	mu       sync.Mutex
	piResult *service.PaymentIntentResult
	piErr    error
	piCalls  int

	getPI      *service.PaymentIntentStatus
	getPIErr   error
	getPICalls int

	webhookEvent *service.StripeWebhookEvent
	webhookErr   error
}

func (s *chargeStubStripe) CreatePaymentIntent(_ context.Context, in service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.piCalls++
	if s.piErr != nil {
		return nil, s.piErr
	}
	if s.piResult == nil {
		// Default: fabricate a result keyed off the proposal id.
		return &service.PaymentIntentResult{
			PaymentIntentID: "pi_" + in.ProposalID,
			ClientSecret:    "cs_" + in.ProposalID,
		}, nil
	}
	return s.piResult, nil
}

func (s *chargeStubStripe) GetPaymentIntent(_ context.Context, _ string) (*service.PaymentIntentStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getPICalls++
	return s.getPI, s.getPIErr
}

func (s *chargeStubStripe) ConstructWebhookEvent(_ []byte, _ string) (*service.StripeWebhookEvent, error) {
	return s.webhookEvent, s.webhookErr
}

// ---------------------------------------------------------------------------
// CreatePaymentIntent — happy paths and edge cases
// ---------------------------------------------------------------------------

func TestChargeService_CreatePaymentIntent_NoStripe_ReturnsError(t *testing.T) {
	c := NewChargeService(ChargeServiceDeps{
		Records: &chargeStubRecords{}, FeeCalculator: &stubFeeCalc{},
	})
	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe not configured")
}

func TestChargeService_CreatePaymentIntent_NewMilestone_PersistsRecord(t *testing.T) {
	records := &chargeStubRecords{byMilestoneErr: domain.ErrPaymentRecordNotFound}
	stripe := &chargeStubStripe{}
	feeCalc := &stubFeeCalc{fee: 1500}

	c := NewChargeService(ChargeServiceDeps{
		Records: records, Stripe: stripe, FeeCalculator: feeCalc,
	})

	out, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEmpty(t, out.ClientSecret)
	assert.Equal(t, int64(1500), out.PlatformFee)
	assert.Equal(t, int64(48500), out.ProviderPayout)
	assert.Equal(t, 1, records.createCalls)
	require.NotNil(t, records.createdRec)
	assert.NotEmpty(t, records.createdRec.StripePaymentIntentID, "PI id must be persisted on the new record")
}

func TestChargeService_CreatePaymentIntent_FeeCalcFails_NoRecordCreated(t *testing.T) {
	records := &chargeStubRecords{byMilestoneErr: domain.ErrPaymentRecordNotFound}
	stripe := &chargeStubStripe{}
	feeCalc := &stubFeeCalc{err: errors.New("user gone")}

	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe, FeeCalculator: feeCalc})

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compute platform fee")
	assert.Equal(t, 0, records.createCalls, "no record may be persisted when fee resolution failed")
	assert.Equal(t, 0, stripe.piCalls, "Stripe must NOT be called when fee resolution failed")
}

func TestChargeService_CreatePaymentIntent_StripeFails_NoRecordCreated(t *testing.T) {
	records := &chargeStubRecords{byMilestoneErr: domain.ErrPaymentRecordNotFound}
	stripe := &chargeStubStripe{piErr: errors.New("stripe rate limit")}
	feeCalc := &stubFeeCalc{fee: 1500}

	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe, FeeCalculator: feeCalc})

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create payment intent")
	assert.Equal(t, 0, records.createCalls, "no record may be persisted when Stripe rejected")
}

func TestChargeService_CreatePaymentIntent_RecordPersistFails_PropagatesError(t *testing.T) {
	// chargeStubRecordsFailingCreate composes a fresh chargeStubRecords
	// (no sync.Mutex copy — Mutex must never be copied after first use).
	failingCreate := &chargeStubRecordsFailingCreate{}
	failingCreate.byMilestoneErr = domain.ErrPaymentRecordNotFound
	stripe := &chargeStubStripe{}
	feeCalc := &stubFeeCalc{fee: 1500}

	c := NewChargeService(ChargeServiceDeps{Records: failingCreate, Stripe: stripe, FeeCalculator: feeCalc})

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persist payment record")
}

type chargeStubRecordsFailingCreate struct {
	chargeStubRecords
}

func (c *chargeStubRecordsFailingCreate) Create(_ context.Context, _ *domain.PaymentRecord) error {
	return errors.New("write conflict")
}

// ---------------------------------------------------------------------------
// MarkPaymentSucceeded — comprehensive matrix (mirrors the legacy table
// test in service_stripe_test.go but exercised directly on ChargeService)
// ---------------------------------------------------------------------------

func TestChargeService_MarkPaymentSucceeded_VerifiesStripe(t *testing.T) {
	tests := []struct {
		name           string
		piID           string
		stripeStatus   string
		stripeErr      error
		stripeNilResp  bool
		recordStatus   domain.PaymentRecordStatus
		wantErr        error
		wantUpdate     bool
		wantStripeCall bool
	}{
		{"stripe says succeeded", "pi", "succeeded", nil, false, domain.RecordStatusPending, nil, true, true},
		{"stripe says requires_payment_method", "pi", "requires_payment_method", nil, false, domain.RecordStatusPending, domain.ErrPaymentNotConfirmed, false, true},
		{"stripe says processing", "pi", "processing", nil, false, domain.RecordStatusPending, domain.ErrPaymentNotConfirmed, false, true},
		{"stripe nil PI", "pi", "", nil, true, domain.RecordStatusPending, domain.ErrPaymentNotConfirmed, false, true},
		{"missing PI id", "", "", nil, false, domain.RecordStatusPending, domain.ErrPaymentNotConfirmed, false, false},
		{"already succeeded — idempotent", "pi", "succeeded", nil, false, domain.RecordStatusSucceeded, nil, false, false},
		{"already refunded — idempotent", "pi", "succeeded", nil, false, domain.RecordStatusRefunded, nil, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &domain.PaymentRecord{
				ID:                    uuid.New(),
				ProposalID:            uuid.New(),
				StripePaymentIntentID: tc.piID,
				Status:                tc.recordStatus,
			}
			records := &chargeStubRecords{byProposal: rec}
			stripe := &chargeStubStripe{}
			if tc.stripeErr != nil {
				stripe.getPIErr = tc.stripeErr
			} else if !tc.stripeNilResp {
				stripe.getPI = &service.PaymentIntentStatus{
					PaymentIntentID: tc.piID,
					Status:          tc.stripeStatus,
				}
			}

			c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe})
			err := c.MarkPaymentSucceeded(context.Background(), rec.ProposalID)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			if tc.wantUpdate {
				assert.Equal(t, 1, records.updateCalls)
			} else {
				assert.Equal(t, 0, records.updateCalls)
			}
			if tc.wantStripeCall {
				assert.Equal(t, 1, stripe.getPICalls)
			} else {
				assert.Equal(t, 0, stripe.getPICalls)
			}
		})
	}
}

func TestChargeService_MarkPaymentSucceeded_NoStripeWired(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:                    uuid.New(),
		ProposalID:            uuid.New(),
		StripePaymentIntentID: "pi",
		Status:                domain.RecordStatusPending,
	}
	records := &chargeStubRecords{byProposal: rec}
	c := NewChargeService(ChargeServiceDeps{Records: records})

	err := c.MarkPaymentSucceeded(context.Background(), rec.ProposalID)
	require.Error(t, err, "without Stripe, verification cannot happen — must refuse")
	assert.Contains(t, err.Error(), "stripe service not configured")
}

func TestChargeService_MarkPaymentSucceeded_GetByProposalErr(t *testing.T) {
	records := &chargeStubRecords{byProposal: nil} // returns ErrPaymentRecordNotFound
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: &chargeStubStripe{}})

	err := c.MarkPaymentSucceeded(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find record")
}

func TestChargeService_MarkPaymentSucceeded_StripeAPIError_Wrapped(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:                    uuid.New(),
		ProposalID:            uuid.New(),
		StripePaymentIntentID: "pi",
		Status:                domain.RecordStatusPending,
	}
	records := &chargeStubRecords{byProposal: rec}
	stripe := &chargeStubStripe{getPIErr: errors.New("stripe down")}

	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe})
	err := c.MarkPaymentSucceeded(context.Background(), rec.ProposalID)
	require.Error(t, err)
	// Wrapped, not converted to ErrPaymentNotConfirmed (the API error
	// is transient and the caller should retry).
	assert.NotErrorIs(t, err, domain.ErrPaymentNotConfirmed)
}

// ---------------------------------------------------------------------------
// HandlePaymentSucceeded — webhook flow
// ---------------------------------------------------------------------------

func TestChargeService_HandlePaymentSucceeded_HappyPath_UpdatesRecord(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:         uuid.New(),
		ProposalID: uuid.New(),
		Status:     domain.RecordStatusPending,
	}
	records := &chargeStubRecords{byPaymentInt: rec}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: &chargeStubStripe{}})

	gotProposalID, err := c.HandlePaymentSucceeded(context.Background(), "pi_test")
	require.NoError(t, err)
	assert.Equal(t, rec.ProposalID, gotProposalID)
	assert.Equal(t, 1, records.updateCalls)
	require.NotNil(t, records.updatedRec)
	assert.Equal(t, domain.RecordStatusSucceeded, records.updatedRec.Status)
}

func TestChargeService_HandlePaymentSucceeded_AlreadySucceeded_Idempotent(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:         uuid.New(),
		ProposalID: uuid.New(),
		Status:     domain.RecordStatusSucceeded,
	}
	records := &chargeStubRecords{byPaymentInt: rec}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: &chargeStubStripe{}})

	gotProposalID, err := c.HandlePaymentSucceeded(context.Background(), "pi_test")
	require.NoError(t, err)
	assert.Equal(t, rec.ProposalID, gotProposalID)
	assert.Equal(t, 0, records.updateCalls, "no Update on a record that is already succeeded — idempotent webhook replay")
}

func TestChargeService_HandlePaymentSucceeded_RecordNotFound(t *testing.T) {
	records := &chargeStubRecords{}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: &chargeStubStripe{}})

	_, err := c.HandlePaymentSucceeded(context.Background(), "pi_unknown")
	require.Error(t, err)
}

func TestChargeService_HandlePaymentSucceeded_UpdatePersistFails(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:         uuid.New(),
		ProposalID: uuid.New(),
		Status:     domain.RecordStatusPending,
	}
	records := &chargeStubRecords{
		byPaymentInt: rec,
		updateErr:    errors.New("conflict"),
	}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: &chargeStubStripe{}})

	_, err := c.HandlePaymentSucceeded(context.Background(), "pi_test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update record")
}

// ---------------------------------------------------------------------------
// VerifyWebhook — pass-through to Stripe adapter
// ---------------------------------------------------------------------------

func TestChargeService_VerifyWebhook_NoStripe_Errors(t *testing.T) {
	c := NewChargeService(ChargeServiceDeps{Records: &chargeStubRecords{}})
	_, err := c.VerifyWebhook([]byte("payload"), "sig")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe not configured")
}

func TestChargeService_VerifyWebhook_DelegatesToStripe(t *testing.T) {
	want := &service.StripeWebhookEvent{Type: "payment_intent.succeeded"}
	stripe := &chargeStubStripe{webhookEvent: want}
	c := NewChargeService(ChargeServiceDeps{Records: &chargeStubRecords{}, Stripe: stripe})

	got, err := c.VerifyWebhook([]byte("payload"), "sig")
	require.NoError(t, err)
	assert.Same(t, want, got)
}

func TestChargeService_VerifyWebhook_StripeFails(t *testing.T) {
	stripe := &chargeStubStripe{webhookErr: errors.New("bad sig")}
	c := NewChargeService(ChargeServiceDeps{Records: &chargeStubRecords{}, Stripe: stripe})

	_, err := c.VerifyWebhook([]byte("payload"), "bad")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Race tests — concurrent CreatePaymentIntent calls must not corrupt state
// ---------------------------------------------------------------------------

func TestChargeService_CreatePaymentIntent_Concurrent_NoRace(t *testing.T) {
	records := &chargeStubRecords{byMilestoneErr: domain.ErrPaymentRecordNotFound}
	stripe := &chargeStubStripe{}
	feeCalc := &stubFeeCalc{fee: 1500}

	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe, FeeCalculator: feeCalc})

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
				ProposalID:     uuid.New(),
				MilestoneID:    uuid.New(),
				ClientID:       uuid.New(),
				ProviderID:     uuid.New(),
				ProposalAmount: 50000,
			})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	// All N calls produced a record. The exact ordering of createCalls
	// is non-deterministic, but the sum is.
	assert.Equal(t, goroutines, records.createCalls)
}

// ---------------------------------------------------------------------------
// CreatePaymentIntent existing-record path (BUG-09 location 1)
// ---------------------------------------------------------------------------

func TestChargeService_CreatePaymentIntentFromExisting_PersistsNewPI(t *testing.T) {
	existing := &domain.PaymentRecord{
		ID:                    uuid.New(),
		ProposalID:            uuid.New(),
		MilestoneID:           uuid.New(),
		ClientID:              uuid.New(),
		ProviderID:            uuid.New(),
		StripePaymentIntentID: "", // empty triggers persist branch
		Currency:              "eur",
		ProposalAmount:        1000,
		ClientTotalAmount:     1100,
		ProviderPayout:        950,
		StripeFeeAmount:       50,
		PlatformFeeAmount:     50,
		Status:                domain.RecordStatusPending,
	}
	records := &chargeStubRecords{byMilestone: existing}
	stripe := &chargeStubStripe{
		piResult: &service.PaymentIntentResult{
			PaymentIntentID: "pi_refreshed",
			ClientSecret:    "cs_refreshed",
		},
	}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe, FeeCalculator: &stubFeeCalc{}})

	out, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:  existing.ProposalID,
		MilestoneID: existing.MilestoneID,
		ClientID:    existing.ClientID,
		ProviderID:  existing.ProviderID,
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "cs_refreshed", out.ClientSecret)
	assert.Equal(t, 1, records.updateCalls)
}

func TestChargeService_CreatePaymentIntentFromExisting_DBPersistFails_SurfacesError(t *testing.T) {
	existing := &domain.PaymentRecord{
		ID:                    uuid.New(),
		ProposalID:            uuid.New(),
		MilestoneID:           uuid.New(),
		StripePaymentIntentID: "", // empty triggers patched branch
		Currency:              "eur",
		Status:                domain.RecordStatusPending,
	}
	records := &chargeStubRecords{byMilestone: existing, updateErr: errors.New("db blip")}
	stripe := &chargeStubStripe{piResult: &service.PaymentIntentResult{PaymentIntentID: "pi_new"}}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe})

	out, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:  existing.ProposalID,
		MilestoneID: existing.MilestoneID,
	})
	require.Error(t, err, "BUG-09: DB blip on PI re-fetch persist must NOT be swallowed")
	assert.Nil(t, out)
}

func TestChargeService_CreatePaymentIntentFromExisting_NonEmptyPI_NoUpdate(t *testing.T) {
	existing := &domain.PaymentRecord{
		ID:                    uuid.New(),
		ProposalID:            uuid.New(),
		MilestoneID:           uuid.New(),
		StripePaymentIntentID: "pi_already_set", // not empty → no Update
		Currency:              "eur",
		Status:                domain.RecordStatusPending,
		ClientTotalAmount:     1100,
	}
	records := &chargeStubRecords{byMilestone: existing}
	stripe := &chargeStubStripe{
		piResult: &service.PaymentIntentResult{PaymentIntentID: "pi_already_set", ClientSecret: "cs_yes"},
	}
	c := NewChargeService(ChargeServiceDeps{Records: records, Stripe: stripe})

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:  existing.ProposalID,
		MilestoneID: existing.MilestoneID,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, records.updateCalls, "no Update should fire when PI id is already populated")
}
