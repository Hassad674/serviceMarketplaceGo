package payment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaymentRecord(t *testing.T) {
	proposalID := uuid.New()
	milestoneID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	// The fee is no longer computed by the domain (used to be hardcoded 5%).
	// The app layer now provides the platform fee from the billing schedule,
	// so tests assert that the constructor faithfully records the values it
	// receives — that's the new contract.
	tests := []struct {
		name            string
		amount          int64
		stripeFee       int64
		platformFee     int64
		wantClientTotal int64
		wantPayout      int64
	}{
		{
			name:            "freelance tier 1 — 150€ milestone, 9€ fee",
			amount:          15000,
			stripeFee:       250,
			platformFee:     900,
			wantClientTotal: 15250,
			wantPayout:      14100,
		},
		{
			name:            "freelance tier 2 — 500€ milestone, 15€ fee",
			amount:          50000,
			stripeFee:       775,
			platformFee:     1500,
			wantClientTotal: 50775,
			wantPayout:      48500,
		},
		{
			name:            "freelance tier 3 — 2000€ milestone, 25€ fee",
			amount:          200000,
			stripeFee:       3025,
			platformFee:     2500,
			wantClientTotal: 203025,
			wantPayout:      197500,
		},
		{
			name:            "agency tier 2 — 1000€ milestone, 39€ fee",
			amount:          100000,
			stripeFee:       1525,
			platformFee:     3900,
			wantClientTotal: 101525,
			wantPayout:      96100,
		},
		{
			name:            "premium subscriber — fee waived",
			amount:          50000,
			stripeFee:       775,
			platformFee:     0,
			wantClientTotal: 50775,
			wantPayout:      50000,
		},
		{
			name:            "zero amount, zero fee",
			amount:          0,
			stripeFee:       0,
			platformFee:     0,
			wantClientTotal: 0,
			wantPayout:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := NewPaymentRecord(proposalID, milestoneID, clientID, providerID, tt.amount, tt.stripeFee, tt.platformFee)

			assert.Equal(t, proposalID, rec.ProposalID)
			assert.Equal(t, milestoneID, rec.MilestoneID)
			assert.Equal(t, clientID, rec.ClientID)
			assert.Equal(t, providerID, rec.ProviderID)
			assert.Equal(t, tt.amount, rec.ProposalAmount)
			assert.Equal(t, tt.stripeFee, rec.StripeFeeAmount)
			assert.Equal(t, tt.platformFee, rec.PlatformFeeAmount)
			assert.Equal(t, tt.wantClientTotal, rec.ClientTotalAmount)
			assert.Equal(t, tt.wantPayout, rec.ProviderPayout)
			assert.Equal(t, "eur", rec.Currency)
			assert.Equal(t, RecordStatusPending, rec.Status)
			assert.Equal(t, TransferPending, rec.TransferStatus)
			assert.NotEqual(t, uuid.Nil, rec.ID)
			assert.Nil(t, rec.PaidAt)
			assert.Nil(t, rec.TransferredAt)
		})
	}
}

// newFixtureRecord is a compact helper for state-machine tests — the fee
// value itself is irrelevant to status transitions.
func newFixtureRecord() *PaymentRecord {
	return NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175, 1500)
}

func TestPaymentRecord_MarkPaid(t *testing.T) {
	t.Run("success from pending", func(t *testing.T) {
		rec := newFixtureRecord()

		err := rec.MarkPaid()

		require.NoError(t, err)
		assert.Equal(t, RecordStatusSucceeded, rec.Status)
		assert.NotNil(t, rec.PaidAt)
	})

	t.Run("fails from succeeded", func(t *testing.T) {
		rec := newFixtureRecord()
		_ = rec.MarkPaid()

		err := rec.MarkPaid()

		assert.ErrorIs(t, err, ErrPaymentNotPending)
	})

	t.Run("fails from failed", func(t *testing.T) {
		rec := newFixtureRecord()
		require.NoError(t, rec.MarkFailed()) // Pending → Failed

		err := rec.MarkPaid()

		assert.ErrorIs(t, err, ErrPaymentNotPending)
	})
}

func TestPaymentRecord_MarkFailed(t *testing.T) {
	rec := newFixtureRecord()

	err := rec.MarkFailed()
	require.NoError(t, err)
	assert.Equal(t, RecordStatusFailed, rec.Status)
}

func TestPaymentRecord_MarkTransferred(t *testing.T) {
	t.Run("success from succeeded+pending transfer", func(t *testing.T) {
		rec := newFixtureRecord()
		_ = rec.MarkPaid()

		err := rec.MarkTransferred("tr_abc123")

		require.NoError(t, err)
		assert.Equal(t, TransferCompleted, rec.TransferStatus)
		assert.Equal(t, "tr_abc123", rec.StripeTransferID)
		assert.NotNil(t, rec.TransferredAt)
	})

	t.Run("fails when payment not succeeded", func(t *testing.T) {
		rec := newFixtureRecord()

		err := rec.MarkTransferred("tr_abc")

		assert.ErrorIs(t, err, ErrPaymentNotSucceeded)
	})

	t.Run("fails when transfer already done", func(t *testing.T) {
		rec := newFixtureRecord()
		_ = rec.MarkPaid()
		_ = rec.MarkTransferred("tr_first")

		err := rec.MarkTransferred("tr_second")

		assert.ErrorIs(t, err, ErrTransferAlreadyDone)
	})

	t.Run("fails when transfer failed then retried", func(t *testing.T) {
		rec := newFixtureRecord()
		_ = rec.MarkPaid()
		rec.MarkTransferFailed()

		err := rec.MarkTransferred("tr_retry")

		assert.ErrorIs(t, err, ErrTransferAlreadyDone)
	})
}

func TestPaymentRecord_MarkTransferFailed(t *testing.T) {
	rec := newFixtureRecord()

	rec.MarkTransferFailed()

	assert.Equal(t, TransferFailed, rec.TransferStatus)
}

func TestEstimateStripeFee(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		wantFee int64
	}{
		{
			name:    "100€ (10000 centimes)",
			amount:  10000,
			wantFee: 175,
		},
		{
			name:    "10€ (1000 centimes)",
			amount:  1000,
			wantFee: 40,
		},
		{
			name:    "1€ (100 centimes)",
			amount:  100,
			wantFee: 27,
		},
		{
			name:    "1000€ (100000 centimes)",
			amount:  100000,
			wantFee: 1525,
		},
		{
			name:    "zero amount",
			amount:  0,
			wantFee: 25,
		},
		{
			name:    "1 centime",
			amount:  1,
			wantFee: 26,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := EstimateStripeFee(tt.amount)
			assert.Equal(t, tt.wantFee, fee)
		})
	}
}
