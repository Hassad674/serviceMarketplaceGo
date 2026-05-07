package receipt

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/receipt"
)

// fakeReceiptRepo is the in-memory fake used by the service tests.
// It records the last call args so each test can assert that the
// service correctly forwarded the org id (defense-in-depth filter).
type fakeReceiptRepo struct {
	receipts map[uuid.UUID]*domain.Receipt
	listFn   func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Receipt, string, error)
	getErr   error

	lastListOrgID uuid.UUID
	lastGetID     uuid.UUID
	lastGetOrgID  uuid.UUID
}

func (f *fakeReceiptRepo) ListForOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Receipt, string, error) {
	f.lastListOrgID = orgID
	if f.listFn != nil {
		return f.listFn(ctx, orgID, cursor, limit)
	}
	out := make([]*domain.Receipt, 0)
	for _, r := range f.receipts {
		if r.IsParty(orgID) {
			out = append(out, r)
		}
	}
	return out, "", nil
}

func (f *fakeReceiptRepo) GetForOrganization(ctx context.Context, receiptID, orgID uuid.UUID) (*domain.Receipt, error) {
	f.lastGetID = receiptID
	f.lastGetOrgID = orgID
	if f.getErr != nil {
		return nil, f.getErr
	}
	rec, ok := f.receipts[receiptID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if !rec.IsParty(orgID) {
		return nil, domain.ErrForbidden
	}
	return rec, nil
}

type fakeRenderer struct {
	out []byte
	err error
}

func (f *fakeRenderer) RenderReceipt(ctx context.Context, rec *domain.Receipt, language string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func sampleReceipt(clientOrg, providerOrg uuid.UUID) *domain.Receipt {
	return &domain.Receipt{
		ID:                uuid.New(),
		PaymentRecordID:   uuid.New(),
		AmountCents:       12000,
		Currency:          "EUR",
		SnapshotAvailable: true,
		Client:            &domain.PartyBilling{OrganizationID: clientOrg, Name: "Acme"},
		Provider:          &domain.PartyBilling{OrganizationID: providerOrg, Name: "Freelance Co"},
	}
}

func TestService_List_HappyPath(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec}}
	svc := NewService(ServiceDeps{Repo: repo})

	page, err := svc.List(context.Background(), clientOrg, "", 50)
	require.NoError(t, err)
	require.NotNil(t, page)
	assert.Len(t, page.Receipts, 1)
	assert.Equal(t, clientOrg, repo.lastListOrgID)
}

func TestService_List_NilRepo(t *testing.T) {
	svc := NewService(ServiceDeps{})
	_, err := svc.List(context.Background(), uuid.New(), "", 20)
	assert.Error(t, err)
}

func TestService_List_NilOrg(t *testing.T) {
	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	svc := NewService(ServiceDeps{Repo: repo})
	_, err := svc.List(context.Background(), uuid.Nil, "", 20)
	assert.Error(t, err)
}

func TestService_List_EmptyResultsAreNonNil(t *testing.T) {
	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	svc := NewService(ServiceDeps{Repo: repo})
	page, err := svc.List(context.Background(), uuid.New(), "", 20)
	require.NoError(t, err)
	assert.NotNil(t, page.Receipts)
	assert.Empty(t, page.Receipts)
}

func TestService_Get_HappyPath(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec}}
	svc := NewService(ServiceDeps{Repo: repo})

	got, err := svc.Get(context.Background(), rec.ID, providerOrg)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
	assert.Equal(t, rec.ID, repo.lastGetID)
	assert.Equal(t, providerOrg, repo.lastGetOrgID)
}

func TestService_Get_NotFound(t *testing.T) {
	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	svc := NewService(ServiceDeps{Repo: repo})
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Get_Forbidden_WhenCallerIsNotParty(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	other := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec}}
	svc := NewService(ServiceDeps{Repo: repo})

	_, err := svc.Get(context.Background(), rec.ID, other)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestService_Get_DefenseInDepth_RepoLeaksRecord(t *testing.T) {
	// Even if the repository misbehaves and returns a row the caller
	// is not a party on, the service must reject with ErrForbidden.
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	other := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{
		receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec},
		// Override Get to skip the IsParty check, simulating a buggy adapter.
	}
	repo.getErr = nil
	repoBuggy := &buggyRepo{rec: rec}
	svc := NewService(ServiceDeps{Repo: repoBuggy})

	_, err := svc.Get(context.Background(), rec.ID, other)
	assert.ErrorIs(t, err, ErrForbidden)
}

// buggyRepo always returns the receipt regardless of the caller — used
// to verify the service's defense-in-depth IsParty check.
type buggyRepo struct{ rec *domain.Receipt }

func (b *buggyRepo) ListForOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Receipt, string, error) {
	return []*domain.Receipt{b.rec}, "", nil
}

func (b *buggyRepo) GetForOrganization(ctx context.Context, receiptID, orgID uuid.UUID) (*domain.Receipt, error) {
	return b.rec, nil
}

func TestService_RenderPDF_NoRenderer(t *testing.T) {
	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	svc := NewService(ServiceDeps{Repo: repo})
	_, _, err := svc.RenderPDF(context.Background(), uuid.New(), uuid.New(), "fr")
	assert.ErrorIs(t, err, ErrPDFRendererUnavailable)
}

func TestService_RenderPDF_HappyPath(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec}}
	renderer := &fakeRenderer{out: []byte("PDFBYTES")}
	svc := NewService(ServiceDeps{Repo: repo, Renderer: renderer})

	pdf, got, err := svc.RenderPDF(context.Background(), rec.ID, clientOrg, "fr")
	require.NoError(t, err)
	assert.Equal(t, []byte("PDFBYTES"), pdf)
	assert.Equal(t, rec.ID, got.ID)
}

func TestService_RenderPDF_PropagatesAuthorizationError(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec}}
	renderer := &fakeRenderer{out: []byte("never")}
	svc := NewService(ServiceDeps{Repo: repo, Renderer: renderer})

	_, _, err := svc.RenderPDF(context.Background(), rec.ID, uuid.New(), "fr")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestService_RenderPDF_RendererError(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	rec := sampleReceipt(clientOrg, providerOrg)

	repo := &fakeReceiptRepo{receipts: map[uuid.UUID]*domain.Receipt{rec.ID: rec}}
	rendererErr := errors.New("chromedp boom")
	renderer := &fakeRenderer{err: rendererErr}
	svc := NewService(ServiceDeps{Repo: repo, Renderer: renderer})

	_, _, err := svc.RenderPDF(context.Background(), rec.ID, clientOrg, "fr")
	assert.Error(t, err)
}
