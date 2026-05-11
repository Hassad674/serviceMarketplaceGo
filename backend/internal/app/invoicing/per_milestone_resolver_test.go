package invoicing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/domain/payment"
)

// fakePaymentsReader satisfies invoicingapp.PaymentRecordReader.
type fakePaymentsReader struct {
	getFn func(ctx context.Context, milestoneID uuid.UUID) (*payment.PaymentRecord, error)
}

func (f *fakePaymentsReader) GetByMilestoneID(ctx context.Context, milestoneID uuid.UUID) (*payment.PaymentRecord, error) {
	if f.getFn != nil {
		return f.getFn(ctx, milestoneID)
	}
	return nil, errors.New("not configured")
}

// fakeOrgReader satisfies invoicingapp.OrganizationOfUserReader.
type fakeOrgReader struct {
	findFn func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

func (f *fakeOrgReader) FindByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	if f.findFn != nil {
		return f.findFn(ctx, userID)
	}
	return uuid.Nil, errors.New("not configured")
}

func TestPerMilestoneInvoicerAdapter_NilSvc_NoOp(t *testing.T) {
	var a *invoicingapp.PerMilestoneInvoicerAdapter
	err := a.IssueFromMilestone(context.Background(), uuid.New())
	assert.NoError(t, err, "nil adapter is a silent no-op (feature disabled)")
}

func TestPerMilestoneInvoicerAdapter_RejectsZeroMilestoneID(t *testing.T) {
	svc, _, _ := newTestServiceFor(t)
	a := invoicingapp.NewPerMilestoneInvoicerAdapter(svc, &fakePaymentsReader{}, &fakeOrgReader{})
	err := a.IssueFromMilestone(context.Background(), uuid.Nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "milestone id required")
}

func TestPerMilestoneInvoicerAdapter_PropagatesPaymentLookupError(t *testing.T) {
	svc, _, _ := newTestServiceFor(t)
	wantErr := errors.New("db boom")
	a := invoicingapp.NewPerMilestoneInvoicerAdapter(svc, &fakePaymentsReader{
		getFn: func(_ context.Context, _ uuid.UUID) (*payment.PaymentRecord, error) {
			return nil, wantErr
		},
	}, &fakeOrgReader{})
	err := a.IssueFromMilestone(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestPerMilestoneInvoicerAdapter_PropagatesOrgLookupError(t *testing.T) {
	svc, _, _ := newTestServiceFor(t)
	rec := validPaymentRecord(500)
	wantErr := errors.New("user not found")
	a := invoicingapp.NewPerMilestoneInvoicerAdapter(svc,
		&fakePaymentsReader{
			getFn: func(_ context.Context, _ uuid.UUID) (*payment.PaymentRecord, error) {
				return rec, nil
			},
		},
		&fakeOrgReader{
			findFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, wantErr
			},
		},
	)
	err := a.IssueFromMilestone(context.Background(), rec.MilestoneID)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestPerMilestoneInvoicerAdapter_RejectsZeroOrg(t *testing.T) {
	svc, _, _ := newTestServiceFor(t)
	rec := validPaymentRecord(500)
	a := invoicingapp.NewPerMilestoneInvoicerAdapter(svc,
		&fakePaymentsReader{
			getFn: func(_ context.Context, _ uuid.UUID) (*payment.PaymentRecord, error) {
				return rec, nil
			},
		},
		&fakeOrgReader{
			findFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, nil
			},
		},
	)
	err := a.IssueFromMilestone(context.Background(), rec.MilestoneID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no organization")
}

func TestPerMilestoneInvoicerAdapter_HappyPath(t *testing.T) {
	svc, repo, profiles := newTestServiceFor(t)
	providerOrg := uuid.New()
	rec := validPaymentRecord(500)
	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return validProviderProfile(providerOrg), nil
	}
	a := invoicingapp.NewPerMilestoneInvoicerAdapter(svc,
		&fakePaymentsReader{
			getFn: func(_ context.Context, mid uuid.UUID) (*payment.PaymentRecord, error) {
				assert.Equal(t, rec.MilestoneID, mid)
				return rec, nil
			},
		},
		&fakeOrgReader{
			findFn: func(_ context.Context, userID uuid.UUID) (uuid.UUID, error) {
				assert.Equal(t, rec.ProviderID, userID)
				return providerOrg, nil
			},
		},
	)
	err := a.IssueFromMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err)
	require.Len(t, repo.persistedInvoices, 1)
	assert.True(t, repo.persistedInvoices[0].IsPlatformFee())
}
