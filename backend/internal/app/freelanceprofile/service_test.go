package freelanceprofile_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfreelance "marketplace-backend/internal/app/freelanceprofile"
	domainfreelance "marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/search"
)

// mockFreelanceProfileRepo is a hand-rolled mock for the tests in
// this file. Every method is a function field so a single test can
// swap behaviours without constructing a new mock type each time.
type mockFreelanceProfileRepo struct {
	getByOrgID             func(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error)
	getOrCreateByOrgID     func(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error)
	updateCore             func(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailability     func(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpertiseDomains func(ctx context.Context, orgID uuid.UUID, domains []string) error

	// Tx variants — used by the outbox path (BUG-05). Nil means
	// "delegate to the non-tx variant" so tests written before the
	// outbox refactor keep passing.
	updateCoreTx             func(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailabilityTx     func(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpertiseDomainsTx func(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, domains []string) error
}

func (m *mockFreelanceProfileRepo) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	return m.getByOrgID(ctx, orgID)
}
func (m *mockFreelanceProfileRepo) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	if m.getOrCreateByOrgID != nil {
		return m.getOrCreateByOrgID(ctx, orgID)
	}
	// Fallback to the strict read so tests that only wire getByOrgID
	// keep working — the service's owner path now calls
	// GetOrCreateByOrgID internally.
	return m.getByOrgID(ctx, orgID)
}
func (m *mockFreelanceProfileRepo) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	return m.updateCore(ctx, orgID, title, about, videoURL)
}
func (m *mockFreelanceProfileRepo) UpdateCoreTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, title, about, videoURL string) error {
	if m.updateCoreTx != nil {
		return m.updateCoreTx(ctx, tx, orgID, title, about, videoURL)
	}
	return m.updateCore(ctx, orgID, title, about, videoURL)
}
func (m *mockFreelanceProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	return m.updateAvailability(ctx, orgID, status)
}
func (m *mockFreelanceProfileRepo) UpdateAvailabilityTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	if m.updateAvailabilityTx != nil {
		return m.updateAvailabilityTx(ctx, tx, orgID, status)
	}
	return m.updateAvailability(ctx, orgID, status)
}
func (m *mockFreelanceProfileRepo) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	return m.updateExpertiseDomains(ctx, orgID, domains)
}
func (m *mockFreelanceProfileRepo) UpdateExpertiseDomainsTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, domains []string) error {
	if m.updateExpertiseDomainsTx != nil {
		return m.updateExpertiseDomainsTx(ctx, tx, orgID, domains)
	}
	return m.updateExpertiseDomains(ctx, orgID, domains)
}
func (m *mockFreelanceProfileRepo) UpdateVideo(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockFreelanceProfileRepo) GetVideoURL(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

// fakeSearchPublisher is a tiny stub for the freelanceprofile.SearchIndexPublisher
// port. Tracks each call so tests can assert on whether the tx-aware
// variant was invoked instead of the legacy one.
type fakeSearchPublisher struct {
	reindexCalls   int
	reindexTxCalls int
	reindexTxErr   error
}

func (f *fakeSearchPublisher) PublishReindex(_ context.Context, _ uuid.UUID, _ search.Persona) error {
	f.reindexCalls++
	return nil
}

func (f *fakeSearchPublisher) PublishReindexTx(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ search.Persona) error {
	f.reindexTxCalls++
	return f.reindexTxErr
}

// stubTxRunner runs the user fn synchronously with a non-nil *sql.Tx
// pointer (the publisher only checks for non-nil; the fake repo never
// dereferences it). When commitErr is set, the runner pretends the
// commit failed AFTER the fn returned nil — used to simulate a DB
// blip between fn returning nil and the actual COMMIT.
type stubTxRunner struct {
	calls     int
	commitErr error
	rollback  bool
}

func (s *stubTxRunner) RunInTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	s.calls++
	tx := &sql.Tx{}
	if err := fn(tx); err != nil {
		s.rollback = true
		return err
	}
	if s.commitErr != nil {
		s.rollback = true
		return s.commitErr
	}
	return nil
}

// RunInTxWithTenant satisfies the new repository.TxRunner contract
// added by Phase 5 Agent Q (RLS migration 125). The stub ignores the
// tenant ids — RLS is only enforced by Postgres, not by the stub —
// and delegates to RunInTx so the existing call-count assertions
// keep working unchanged.
func (s *stubTxRunner) RunInTxWithTenant(ctx context.Context, _, _ uuid.UUID, fn func(tx *sql.Tx) error) error {
	return s.RunInTx(ctx, fn)
}

// newStubView returns a minimal FreelanceProfileView suitable for
// tests that do not care about the payload shape, only whether
// something non-nil was returned.
func newStubView(orgID uuid.UUID) *repository.FreelanceProfileView {
	return &repository.FreelanceProfileView{
		Profile: &domainfreelance.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{},
		},
	}
}

func TestService_GetByOrgID_PassesThroughRepoResult(t *testing.T) {
	orgID := uuid.New()
	stub := newStubView(orgID)
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			assert.Equal(t, orgID, id)
			return stub, nil
		},
	}
	svc := appfreelance.NewService(repo)

	got, err := svc.GetByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, stub, got)
}

func TestService_GetByOrgID_WrapsRepoError(t *testing.T) {
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return nil, domainfreelance.ErrProfileNotFound
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.GetByOrgID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domainfreelance.ErrProfileNotFound)
}

// GetPublicByOrgID is the public read path used by the listing pages.
// Strict: must NEVER lazily create a profile (that path is for the
// owner's editor only). Must wrap a missing-row error so callers can
// detect "this org has no public freelance profile".
func TestService_GetPublicByOrgID_PassesThroughRepoResult(t *testing.T) {
	orgID := uuid.New()
	stub := newStubView(orgID)
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			assert.Equal(t, orgID, id, "GetPublicByOrgID must use GetByOrgID, NOT GetOrCreate")
			return stub, nil
		},
	}
	svc := appfreelance.NewService(repo)

	got, err := svc.GetPublicByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, stub, got)
}

func TestService_GetPublicByOrgID_WrapsNotFoundError(t *testing.T) {
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return nil, domainfreelance.ErrProfileNotFound
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.GetPublicByOrgID(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, domainfreelance.ErrProfileNotFound)
}

func TestService_GetPublicByOrgID_DoesNotLazilyCreate(t *testing.T) {
	// Critical contract: GetPublicByOrgID must never call GetOrCreate.
	// A consumer browsing another org's profile must NOT silently
	// provision a row for that org — that's a side effect we
	// definitely don't want from a read endpoint.
	getOrCreateCalls := 0
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return nil, domainfreelance.ErrProfileNotFound
		},
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			getOrCreateCalls++
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, _ = svc.GetPublicByOrgID(context.Background(), uuid.New())

	assert.Equal(t, 0, getOrCreateCalls,
		"GetPublicByOrgID must NEVER call GetOrCreateByOrgID — it's a public read")
}

// GetFreelanceProfileIDByOrgID resolves the surrogate profile ID
// for the pricing handler. Uses the lazy GetOrCreate path because
// the pricing editor may be opened before the profile row exists.
func TestService_GetFreelanceProfileIDByOrgID_ReturnsProfileID(t *testing.T) {
	orgID := uuid.New()
	stub := newStubView(orgID)
	repo := &mockFreelanceProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			assert.Equal(t, orgID, id)
			return stub, nil
		},
	}
	svc := appfreelance.NewService(repo)

	got, err := svc.GetFreelanceProfileIDByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, stub.Profile.ID, got)
}

func TestService_GetFreelanceProfileIDByOrgID_WrapsRepoError(t *testing.T) {
	boom := errors.New("repo blew up")
	repo := &mockFreelanceProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return nil, boom
		},
	}
	svc := appfreelance.NewService(repo)

	got, err := svc.GetFreelanceProfileIDByOrgID(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, uuid.Nil, got, "error path must return uuid.Nil so callers can branch on the boundary")
}

func TestService_UpdateCore_TrimsAndRefetches(t *testing.T) {
	orgID := uuid.New()
	var gotTitle, gotAbout, gotVideo string
	repo := &mockFreelanceProfileRepo{
		updateCore: func(ctx context.Context, id uuid.UUID, title, about, videoURL string) error {
			gotTitle = title
			gotAbout = about
			gotVideo = videoURL
			return nil
		},
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{
		Title:    "  Senior Go Engineer  ",
		About:    "\nBuilds marketplaces.\n",
		VideoURL: " https://example.com/v.mp4 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "Senior Go Engineer", gotTitle)
	assert.Equal(t, "Builds marketplaces.", gotAbout)
	assert.Equal(t, "https://example.com/v.mp4", gotVideo)
}

func TestService_UpdateCore_PropagatesRepoError(t *testing.T) {
	boom := errors.New("database exploded")
	repo := &mockFreelanceProfileRepo{
		updateCore: func(ctx context.Context, id uuid.UUID, _, _, _ string) error {
			return boom
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateCore(context.Background(), uuid.New(), appfreelance.UpdateCoreInput{})
	assert.ErrorIs(t, err, boom)
}

func TestService_UpdateAvailability_ValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"valid now", "available_now", false},
		{"valid soon", "available_soon", false},
		{"valid not", "not_available", false},
		{"empty", "", true},
		{"unknown", "maybe", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockFreelanceProfileRepo{
				updateAvailability: func(ctx context.Context, id uuid.UUID, status profile.AvailabilityStatus) error {
					return nil
				},
				getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
					return newStubView(id), nil
				},
			}
			svc := appfreelance.NewService(repo)
			_, err := svc.UpdateAvailability(context.Background(), uuid.New(), tc.raw)
			if tc.wantErr {
				assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_UpdateExpertise_NormalizesInput(t *testing.T) {
	var captured []string
	repo := &mockFreelanceProfileRepo{
		updateExpertiseDomains: func(ctx context.Context, id uuid.UUID, domains []string) error {
			captured = domains
			return nil
		},
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateExpertise(context.Background(), uuid.New(),
		[]string{"  development ", "design_ui_ux", "", "development"},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"development", "design_ui_ux"}, captured,
		"trimmed + deduped, preserving first-occurrence order")
}

func TestService_UpdateExpertise_NilInputYieldsEmptySlice(t *testing.T) {
	var captured []string
	captured = []string{"sentinel"}
	repo := &mockFreelanceProfileRepo{
		updateExpertiseDomains: func(ctx context.Context, id uuid.UUID, domains []string) error {
			captured = domains
			return nil
		},
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateExpertise(context.Background(), uuid.New(), nil)
	require.NoError(t, err)
	assert.NotNil(t, captured)
	assert.Empty(t, captured)
}

// ---------------------------------------------------------------------------
// BUG-05 — outbox path: when a TxRunner + SearchIndexPublisher are
// both wired, every mutation must run repo.UpdateXxxTx and
// publisher.PublishReindexTx in the same transaction.
// ---------------------------------------------------------------------------

func TestService_UpdateCore_Outbox_UsesTxAndScheduleTx(t *testing.T) {
	orgID := uuid.New()
	var (
		updateTxCalls int
		passedTx      *sql.Tx
	)

	repo := &mockFreelanceProfileRepo{
		updateCoreTx: func(_ context.Context, tx *sql.Tx, _ uuid.UUID, _, _, _ string) error {
			updateTxCalls++
			passedTx = tx
			return nil
		},
		updateCore: func(_ context.Context, _ uuid.UUID, _, _, _ string) error {
			t.Fatalf("non-tx UpdateCore must NOT be called when outbox is wired")
			return nil
		},
		getByOrgID: func(_ context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	pub := &fakeSearchPublisher{}
	runner := &stubTxRunner{}

	svc := appfreelance.NewService(repo).
		WithSearchIndexPublisher(pub).
		WithTxRunner(runner)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{
		Title: "Title", About: "About",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, runner.calls, "TxRunner.RunInTx must be invoked exactly once")
	assert.Equal(t, 1, updateTxCalls, "repo.UpdateCoreTx must be called from inside the tx")
	assert.NotNil(t, passedTx, "repo must receive the live *sql.Tx")
	assert.Equal(t, 0, pub.reindexCalls, "non-tx Publish must NOT fire on the outbox path")
	assert.Equal(t, 1, pub.reindexTxCalls, "PublishReindexTx must fire exactly once inside the tx")
	assert.False(t, runner.rollback, "happy path commits — no rollback")
}

func TestService_UpdateCore_Outbox_PublishFailureRollsBack(t *testing.T) {
	orgID := uuid.New()
	var updateTxCalls int

	repo := &mockFreelanceProfileRepo{
		updateCoreTx: func(_ context.Context, _ *sql.Tx, _ uuid.UUID, _, _, _ string) error {
			updateTxCalls++
			return nil
		},
	}
	publishErr := errors.New("typesense down")
	pub := &fakeSearchPublisher{reindexTxErr: publishErr}
	runner := &stubTxRunner{}

	svc := appfreelance.NewService(repo).
		WithSearchIndexPublisher(pub).
		WithTxRunner(runner)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{Title: "X"})
	require.Error(t, err, "publisher tx error must surface to the caller — outbox guarantee")
	assert.True(t, runner.rollback, "rollback must be triggered when PublishReindexTx fails")
	assert.Equal(t, 1, updateTxCalls, "repo write happened, but commit must not reach Postgres")
}

func TestService_UpdateCore_Outbox_RepoFailureRollsBack(t *testing.T) {
	repoErr := errors.New("update failed")
	repo := &mockFreelanceProfileRepo{
		updateCoreTx: func(_ context.Context, _ *sql.Tx, _ uuid.UUID, _, _, _ string) error {
			return repoErr
		},
	}
	pub := &fakeSearchPublisher{}
	runner := &stubTxRunner{}

	svc := appfreelance.NewService(repo).
		WithSearchIndexPublisher(pub).
		WithTxRunner(runner)

	_, err := svc.UpdateCore(context.Background(), uuid.New(), appfreelance.UpdateCoreInput{Title: "X"})
	require.Error(t, err, "repo failure must propagate")
	assert.ErrorIs(t, err, repoErr)
	assert.True(t, runner.rollback)
	assert.Equal(t, 0, pub.reindexTxCalls, "publisher must NOT be invoked after repo error")
}

func TestService_UpdateCore_Outbox_CommitFailureSurfaces(t *testing.T) {
	repo := &mockFreelanceProfileRepo{
		updateCoreTx: func(_ context.Context, _ *sql.Tx, _ uuid.UUID, _, _, _ string) error {
			return nil
		},
	}
	commitErr := errors.New("commit failed")
	pub := &fakeSearchPublisher{}
	runner := &stubTxRunner{commitErr: commitErr}

	svc := appfreelance.NewService(repo).
		WithSearchIndexPublisher(pub).
		WithTxRunner(runner)

	_, err := svc.UpdateCore(context.Background(), uuid.New(), appfreelance.UpdateCoreInput{Title: "X"})
	require.Error(t, err, "commit failure must propagate so the caller knows the write was rolled back")
}

func TestService_UpdateAvailability_Outbox_UsesTxPath(t *testing.T) {
	repo := &mockFreelanceProfileRepo{
		updateAvailabilityTx: func(_ context.Context, _ *sql.Tx, _ uuid.UUID, _ profile.AvailabilityStatus) error {
			return nil
		},
		getByOrgID: func(_ context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	pub := &fakeSearchPublisher{}
	runner := &stubTxRunner{}

	svc := appfreelance.NewService(repo).
		WithSearchIndexPublisher(pub).
		WithTxRunner(runner)

	_, err := svc.UpdateAvailability(context.Background(), uuid.New(), "available_now")
	require.NoError(t, err)
	assert.Equal(t, 1, runner.calls)
	assert.Equal(t, 1, pub.reindexTxCalls)
	assert.Equal(t, 0, pub.reindexCalls)
}

func TestService_UpdateExpertise_Outbox_UsesTxPath(t *testing.T) {
	var capturedDomains []string
	repo := &mockFreelanceProfileRepo{
		updateExpertiseDomainsTx: func(_ context.Context, _ *sql.Tx, _ uuid.UUID, domains []string) error {
			capturedDomains = domains
			return nil
		},
		getByOrgID: func(_ context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	pub := &fakeSearchPublisher{}
	runner := &stubTxRunner{}

	svc := appfreelance.NewService(repo).
		WithSearchIndexPublisher(pub).
		WithTxRunner(runner)

	_, err := svc.UpdateExpertise(context.Background(), uuid.New(), []string{"  ml ", "ml", "design"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ml", "design"}, capturedDomains, "normalization happens before tx start")
	assert.Equal(t, 1, runner.calls)
	assert.Equal(t, 1, pub.reindexTxCalls)
}

// Backward-compat: when no TxRunner is wired the legacy hors-tx path
// remains active so existing tests / dev setups keep working.
func TestService_UpdateCore_NoTxRunner_FallsBackToLegacyPath(t *testing.T) {
	var (
		updateCalls   int
		updateTxCalls int
	)
	repo := &mockFreelanceProfileRepo{
		updateCore: func(_ context.Context, _ uuid.UUID, _, _, _ string) error {
			updateCalls++
			return nil
		},
		updateCoreTx: func(_ context.Context, _ *sql.Tx, _ uuid.UUID, _, _, _ string) error {
			updateTxCalls++
			return nil
		},
		getByOrgID: func(_ context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	pub := &fakeSearchPublisher{}

	// Publisher attached but no TxRunner → legacy path.
	svc := appfreelance.NewService(repo).WithSearchIndexPublisher(pub)

	_, err := svc.UpdateCore(context.Background(), uuid.New(), appfreelance.UpdateCoreInput{Title: "X"})
	require.NoError(t, err)
	assert.Equal(t, 1, updateCalls, "legacy non-tx Update must run when TxRunner is not wired")
	assert.Equal(t, 0, updateTxCalls)
	assert.Equal(t, 1, pub.reindexCalls, "legacy hors-tx publish must run as fallback")
	assert.Equal(t, 0, pub.reindexTxCalls)
}

// --- Cache invalidation hook ---

// stubFreelanceInvalidator captures every Invalidate call.
type stubFreelanceInvalidator struct {
	calls []uuid.UUID
	err   error
}

func (s *stubFreelanceInvalidator) Invalidate(_ context.Context, orgID uuid.UUID) error {
	s.calls = append(s.calls, orgID)
	return s.err
}

func TestFreelance_UpdateCore_FiresCacheInvalidator(t *testing.T) {
	orgID := uuid.New()
	repo := &mockFreelanceProfileRepo{
		getByOrgID:         func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		getOrCreateByOrgID: func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		updateCore:         func(_ context.Context, _ uuid.UUID, _, _, _ string) error { return nil },
	}
	inv := &stubFreelanceInvalidator{}
	svc := appfreelance.NewService(repo).WithCacheInvalidator(inv)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{Title: "Hello"})
	require.NoError(t, err)
	require.Len(t, inv.calls, 1)
	assert.Equal(t, orgID, inv.calls[0])
}

func TestFreelance_UpdateAvailability_FiresCacheInvalidator(t *testing.T) {
	orgID := uuid.New()
	repo := &mockFreelanceProfileRepo{
		getByOrgID:         func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		getOrCreateByOrgID: func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		updateAvailability: func(_ context.Context, _ uuid.UUID, _ profile.AvailabilityStatus) error { return nil },
	}
	inv := &stubFreelanceInvalidator{}
	svc := appfreelance.NewService(repo).WithCacheInvalidator(inv)

	_, err := svc.UpdateAvailability(context.Background(), orgID, "available_now")
	require.NoError(t, err)
	require.Len(t, inv.calls, 1)
}

func TestFreelance_UpdateExpertise_FiresCacheInvalidator(t *testing.T) {
	orgID := uuid.New()
	repo := &mockFreelanceProfileRepo{
		getByOrgID:             func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		getOrCreateByOrgID:     func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		updateExpertiseDomains: func(_ context.Context, _ uuid.UUID, _ []string) error { return nil },
	}
	inv := &stubFreelanceInvalidator{}
	svc := appfreelance.NewService(repo).WithCacheInvalidator(inv)

	_, err := svc.UpdateExpertise(context.Background(), orgID, []string{"development"})
	require.NoError(t, err)
	require.Len(t, inv.calls, 1)
}

func TestFreelance_PersistenceFailure_DoesNotInvalidate(t *testing.T) {
	// A failed DB write must NOT invalidate the cache, otherwise
	// readers re-populate from a stale row, defeating cache-aside.
	orgID := uuid.New()
	repo := &mockFreelanceProfileRepo{
		updateCore: func(_ context.Context, _ uuid.UUID, _, _, _ string) error {
			return errors.New("connection lost")
		},
	}
	inv := &stubFreelanceInvalidator{}
	svc := appfreelance.NewService(repo).WithCacheInvalidator(inv)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{Title: "X"})
	require.Error(t, err)
	assert.Empty(t, inv.calls, "failed DB write must NOT invalidate")
}

func TestFreelance_InvalidatorErrorDoesNotFailWrite(t *testing.T) {
	orgID := uuid.New()
	repo := &mockFreelanceProfileRepo{
		getByOrgID:         func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		getOrCreateByOrgID: func(_ context.Context, _ uuid.UUID) (*repository.FreelanceProfileView, error) { return &repository.FreelanceProfileView{}, nil },
		updateCore:         func(_ context.Context, _ uuid.UUID, _, _, _ string) error { return nil },
	}
	inv := &stubFreelanceInvalidator{err: errors.New("redis down")}
	svc := appfreelance.NewService(repo).WithCacheInvalidator(inv)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{Title: "X"})
	require.NoError(t, err, "flaky cache must not unwind a successful DB write")
	assert.Len(t, inv.calls, 1)
}

func TestFreelance_WithCacheInvalidator_NilReceiver(t *testing.T) {
	var svc *appfreelance.Service
	assert.Nil(t, svc.WithCacheInvalidator(nil))
}
