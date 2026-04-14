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

	tests := []struct {
		name           string
		amount         int64
		stripeFee      int64
		wantPlatform   int64
		wantClientTotal int64
		wantPayout     int64
	}{
		{
			name:           "standard 10000 centimes (100€)",
			amount:         10000,
			stripeFee:      175,
			wantPlatform:   500,
			wantClientTotal: 10175,
			wantPayout:     9500,
		},
		{
			name:           "small amount 1000 centimes (10€)",
			amount:         1000,
			stripeFee:      40,
			wantPlatform:   50,
			wantClientTotal: 1040,
			wantPayout:     950,
		},
		{
			name:           "large amount 100000 centimes (1000€)",
			amount:         100000,
			stripeFee:      1525,
			wantPlatform:   5000,
			wantClientTotal: 101525,
			wantPayout:     95000,
		},
		{
			name:           "zero amount",
			amount:         0,
			stripeFee:      0,
			wantPlatform:   0,
			wantClientTotal: 0,
			wantPayout:     0,
		},
		{
			name:           "amount not evenly divisible by 100",
			amount:         9999,
			stripeFee:      175,
			wantPlatform:   499,
			wantClientTotal: 10174,
			wantPayout:     9500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := NewPaymentRecord(proposalID, milestoneID, clientID, providerID, tt.amount, tt.stripeFee)

			assert.Equal(t, proposalID, rec.ProposalID)
			assert.Equal(t, milestoneID, rec.MilestoneID)
			assert.Equal(t, clientID, rec.ClientID)
			assert.Equal(t, providerID, rec.ProviderID)
			assert.Equal(t, tt.amount, rec.ProposalAmount)
			assert.Equal(t, tt.stripeFee, rec.StripeFeeAmount)
			assert.Equal(t, tt.wantPlatform, rec.PlatformFeeAmount)
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

func TestPaymentRecord_MarkPaid(t *testing.T) {
	t.Run("success from pending", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)

		err := rec.MarkPaid()

		require.NoError(t, err)
		assert.Equal(t, RecordStatusSucceeded, rec.Status)
		assert.NotNil(t, rec.PaidAt)
	})

	t.Run("fails from succeeded", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)
		_ = rec.MarkPaid()

		err := rec.MarkPaid()

		assert.ErrorIs(t, err, ErrPaymentNotPending)
	})

	t.Run("fails from failed", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)
		rec.MarkFailed()

		err := rec.MarkPaid()

		assert.ErrorIs(t, err, ErrPaymentNotPending)
	})
}

func TestPaymentRecord_MarkFailed(t *testing.T) {
	rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)

	rec.MarkFailed()

	assert.Equal(t, RecordStatusFailed, rec.Status)
}

func TestPaymentRecord_MarkTransferred(t *testing.T) {
	t.Run("success from succeeded+pending transfer", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)
		_ = rec.MarkPaid()

		err := rec.MarkTransferred("tr_abc123")

		require.NoError(t, err)
		assert.Equal(t, TransferCompleted, rec.TransferStatus)
		assert.Equal(t, "tr_abc123", rec.StripeTransferID)
		assert.NotNil(t, rec.TransferredAt)
	})

	t.Run("fails when payment not succeeded", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)

		err := rec.MarkTransferred("tr_abc")

		assert.ErrorIs(t, err, ErrPaymentNotSucceeded)
	})

	t.Run("fails when transfer already done", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)
		_ = rec.MarkPaid()
		_ = rec.MarkTransferred("tr_first")

		err := rec.MarkTransferred("tr_second")

		assert.ErrorIs(t, err, ErrTransferAlreadyDone)
	})

	t.Run("fails when transfer failed then retried", func(t *testing.T) {
		rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)
		_ = rec.MarkPaid()
		rec.MarkTransferFailed()

		err := rec.MarkTransferred("tr_retry")

		assert.ErrorIs(t, err, ErrTransferAlreadyDone)
	})
}

func TestPaymentRecord_MarkTransferFailed(t *testing.T) {
	rec := NewPaymentRecord(uuid.New(), uuid.New(), uuid.New(), uuid.New(), 10000, 175)

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
