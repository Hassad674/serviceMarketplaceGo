package referral_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

func validCommissionInput() referral.NewCommissionInput {
	return referral.NewCommissionInput{
		AttributionID:    uuid.New(),
		MilestoneID:      uuid.New(),
		GrossAmountCents: 100_00, // 100 EUR
		RatePct:          5,
		Currency:         "EUR",
	}
}

func TestNewCommission_Valid(t *testing.T) {
	in := validCommissionInput()
	c, err := referral.NewCommission(in)
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, referral.CommissionPending, c.Status)
	assert.Equal(t, int64(5_00), c.CommissionCents) // 5% of 100 EUR = 5 EUR
	assert.Equal(t, "EUR", c.Currency)
}

func TestNewCommission_DefaultsCurrencyToEUR(t *testing.T) {
	in := validCommissionInput()
	in.Currency = ""
	c, err := referral.NewCommission(in)
	require.NoError(t, err)
	assert.Equal(t, "EUR", c.Currency)
}

func TestNewCommission_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*referral.NewCommissionInput)
		want   error
	}{
		{"nil attribution id", func(in *referral.NewCommissionInput) { in.AttributionID = uuid.Nil }, referral.ErrNotAuthorized},
		{"nil milestone id", func(in *referral.NewCommissionInput) { in.MilestoneID = uuid.Nil }, referral.ErrNotAuthorized},
		{"zero gross amount", func(in *referral.NewCommissionInput) { in.GrossAmountCents = 0 }, referral.ErrInsufficientGrossAmount},
		{"negative gross amount", func(in *referral.NewCommissionInput) { in.GrossAmountCents = -100 }, referral.ErrInsufficientGrossAmount},
		{"rate negative", func(in *referral.NewCommissionInput) { in.RatePct = -1 }, referral.ErrRateOutOfRange},
		{"rate too high", func(in *referral.NewCommissionInput) { in.RatePct = 51 }, referral.ErrRateOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validCommissionInput()
			tt.mutate(&in)
			c, err := referral.NewCommission(in)
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, c)
		})
	}
}

func TestComputeCommissionCents_Truncation(t *testing.T) {
	// 5% of 1000 EUR = 50 EUR exactly
	in := referral.NewCommissionInput{
		AttributionID:    uuid.New(),
		MilestoneID:      uuid.New(),
		GrossAmountCents: 1000_00,
		RatePct:          5,
	}
	c, err := referral.NewCommission(in)
	require.NoError(t, err)
	assert.Equal(t, int64(50_00), c.CommissionCents)

	// 3.7% of 333 cents = 12.321 → truncated to 12
	in.GrossAmountCents = 333
	in.RatePct = 3.7
	c, err = referral.NewCommission(in)
	require.NoError(t, err)
	assert.Equal(t, int64(12), c.CommissionCents)

	// 0% gives 0 cents
	in.RatePct = 0
	c, err = referral.NewCommission(in)
	require.NoError(t, err)
	assert.Equal(t, int64(0), c.CommissionCents)
}

func TestCommission_MarkPaid(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	require.NoError(t, c.MarkPaid("tr_test_123"))
	assert.Equal(t, referral.CommissionPaid, c.Status)
	assert.Equal(t, "tr_test_123", c.StripeTransferID)
	require.NotNil(t, c.PaidAt)
}

func TestCommission_MarkPaid_OnlyFromPending(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	require.NoError(t, c.MarkPendingKYC())
	err := c.MarkPaid("tr_test")
	require.ErrorIs(t, err, referral.ErrCommissionNotPayable)
}

func TestCommission_MarkPendingKYC(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	require.NoError(t, c.MarkPendingKYC())
	assert.Equal(t, referral.CommissionPendingKYC, c.Status)
}

func TestCommission_MarkFailed(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	require.NoError(t, c.MarkFailed("stripe error"))
	assert.Equal(t, referral.CommissionFailed, c.Status)
	assert.Equal(t, "stripe error", c.FailureReason)
}

func TestCommission_MarkCancelled(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	require.NoError(t, c.MarkCancelled())
	assert.Equal(t, referral.CommissionCancelled, c.Status)
}

func TestCommission_MarkCancelled_NotFromPaidOrTerminal(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	_ = c.MarkPaid("tr_x")
	err := c.MarkCancelled()
	require.ErrorIs(t, err, referral.ErrClawbackNotApplicable)
}

func TestCommission_ApplyClawback(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	require.NoError(t, c.MarkPaid("tr_x"))
	require.NoError(t, c.ApplyClawback("trr_clawback"))
	assert.Equal(t, referral.CommissionClawedBack, c.Status)
	assert.Equal(t, "trr_clawback", c.StripeReversalID)
	require.NotNil(t, c.ClawedBackAt)
}

func TestCommission_ApplyClawback_OnlyFromPaid(t *testing.T) {
	c, _ := referral.NewCommission(validCommissionInput())
	err := c.ApplyClawback("trr_x")
	require.ErrorIs(t, err, referral.ErrClawbackNotApplicable)
}

func TestClawbackAmountCents(t *testing.T) {
	tests := []struct {
		name           string
		commissionCents int64
		grossCents      int64
		refundedCents   int64
		want           int64
	}{
		{"full refund", 500, 10_000, 10_000, 500},
		{"half refund", 500, 10_000, 5_000, 250},
		{"third refund", 500, 9_000, 3_000, 166}, // 500 * 3000 / 9000 = 166.66 → 166 truncated
		{"refund larger than gross caps", 500, 10_000, 20_000, 500},
		{"zero refund", 500, 10_000, 0, 0},
		{"zero commission", 0, 10_000, 5_000, 0},
		{"zero gross", 500, 0, 5_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := referral.ClawbackAmountCents(tt.commissionCents, tt.grossCents, tt.refundedCents)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommissionStatus_IsValid(t *testing.T) {
	assert.True(t, referral.CommissionPending.IsValid())
	assert.True(t, referral.CommissionPaid.IsValid())
	assert.True(t, referral.CommissionClawedBack.IsValid())
	assert.False(t, referral.CommissionStatus("zombie").IsValid())
}
