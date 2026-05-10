package gdpr

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domaingdpr "marketplace-backend/internal/domain/gdpr"
	portservice "marketplace-backend/internal/port/service"
)

// stubStorage is the minimum StorageService satisfying the contract,
// with hooks to capture BulkDelete inputs and inject failure
// behaviors. Only the methods PurgeOnce reaches are wired; the others
// panic so an accidental call surfaces immediately.
type stubStorage struct {
	mu             sync.Mutex
	bulkDeleteCall []string
	bulkDeleteFn   func(ctx context.Context, keys []string) ([]portservice.BulkDeleteResult, error)
}

func (s *stubStorage) Upload(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
	panic("stub: Upload not used")
}
func (s *stubStorage) Delete(_ context.Context, _ string) error {
	panic("stub: Delete not used")
}
func (s *stubStorage) BulkDelete(ctx context.Context, keys []string) ([]portservice.BulkDeleteResult, error) {
	s.mu.Lock()
	s.bulkDeleteCall = append(s.bulkDeleteCall, keys...)
	s.mu.Unlock()
	if s.bulkDeleteFn != nil {
		return s.bulkDeleteFn(ctx, keys)
	}
	out := make([]portservice.BulkDeleteResult, len(keys))
	for i, k := range keys {
		out[i] = portservice.BulkDeleteResult{Key: k}
	}
	return out, nil
}
func (s *stubStorage) GetPublicURL(_ string) string { return "" }
func (s *stubStorage) GetPresignedUploadURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	panic("stub: GetPresignedUploadURL not used")
}
func (s *stubStorage) GetPresignedDownloadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	panic("stub: GetPresignedDownloadURL not used")
}
func (s *stubStorage) GetPresignedDownloadURLAsAttachment(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	panic("stub: GetPresignedDownloadURLAsAttachment not used")
}
func (s *stubStorage) Download(_ context.Context, _ string) ([]byte, error) {
	panic("stub: Download not used")
}

// Compile-time check.
var _ portservice.StorageService = (*stubStorage)(nil)

// TestPurgeOnce_StorageStep_Success asserts the happy path: the
// scheduler gathers keys, bulk-deletes them, records an audit row
// with every key as purged, then runs the SQL anonymization.
func TestPurgeOnce_StorageStep_Success(t *testing.T) {
	userID := uuid.New()
	keys := []string{"avatars/u1.jpg", "videos/v1.mp4", "kyc/u1-id.pdf"}
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	purgeCalls := 0
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{userID}, nil
		},
		purgeFn: func(_ context.Context, id uuid.UUID, _ time.Time, _ string) (bool, error) {
			purgeCalls++
			return true, nil
		},
		listKeysFn: func(_ context.Context, id uuid.UUID) ([]string, error) {
			require.Equal(t, userID, id)
			return keys, nil
		},
	}
	storage := &stubStorage{}

	svc := newServiceForTest(t, ServiceDeps{
		Repo:    repo,
		Users:   &stubUserRepo{},
		Storage: storage,
		Clock:   func() time.Time { return now },
	})

	res, err := svc.PurgeOnce(context.Background(), "salt", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Purged)
	assert.Equal(t, 1, purgeCalls, "SQL purge runs after the storage step")
	assert.ElementsMatch(t, keys, storage.bulkDeleteCall)
	require.Len(t, repo.recordedAudits, 1)
	audit := repo.recordedAudits[0]
	assert.Equal(t, userID, audit.UserID)
	assert.ElementsMatch(t, keys, audit.PurgedKeys)
	assert.Empty(t, audit.FailedKeys)
	assert.Empty(t, audit.Errors)
	assert.Equal(t, now, audit.PurgedAt)
}

// TestPurgeOnce_StorageStep_PartialFailure asserts that per-key
// errors land in failed_keys + errors but the SQL purge still runs
// and the manifest is recorded — DB anonymization is the legal
// floor and must not be skipped on transient R2 failures.
func TestPurgeOnce_StorageStep_PartialFailure(t *testing.T) {
	userID := uuid.New()
	keys := []string{"good/k1", "bad/k2", "good/k3"}

	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{userID}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
		listKeysFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return keys, nil
		},
	}
	storage := &stubStorage{
		bulkDeleteFn: func(_ context.Context, ks []string) ([]portservice.BulkDeleteResult, error) {
			out := make([]portservice.BulkDeleteResult, len(ks))
			for i, k := range ks {
				out[i] = portservice.BulkDeleteResult{Key: k}
				if k == "bad/k2" {
					out[i].Err = errors.New("AccessDenied")
				}
			}
			return out, nil
		},
	}

	svc := newServiceForTest(t, ServiceDeps{
		Repo: repo, Users: &stubUserRepo{}, Storage: storage,
	})

	res, err := svc.PurgeOnce(context.Background(), "salt", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Purged, "SQL purge runs even when 1/3 R2 keys fail")
	require.Len(t, repo.recordedAudits, 1)
	audit := repo.recordedAudits[0]
	assert.ElementsMatch(t, []string{"good/k1", "good/k3"}, audit.PurgedKeys)
	assert.ElementsMatch(t, []string{"bad/k2"}, audit.FailedKeys)
	require.Len(t, audit.Errors, 1)
	assert.Contains(t, audit.Errors[0], "bad/k2")
	assert.Contains(t, audit.Errors[0], "AccessDenied")
}

// TestPurgeOnce_StorageStep_BatchTransportError verifies that when
// BulkDelete itself returns an error (every key already marked
// failed by the adapter), the SQL purge still runs and an audit
// row is still written.
func TestPurgeOnce_StorageStep_BatchTransportError(t *testing.T) {
	userID := uuid.New()
	keys := []string{"k1", "k2"}

	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{userID}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
		listKeysFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return keys, nil
		},
	}
	storage := &stubStorage{
		bulkDeleteFn: func(_ context.Context, ks []string) ([]portservice.BulkDeleteResult, error) {
			out := make([]portservice.BulkDeleteResult, len(ks))
			for i, k := range ks {
				out[i] = portservice.BulkDeleteResult{Key: k, Err: errors.New("transport")}
			}
			return out, errors.New("network down")
		},
	}

	svc := newServiceForTest(t, ServiceDeps{
		Repo: repo, Users: &stubUserRepo{}, Storage: storage,
	})

	res, err := svc.PurgeOnce(context.Background(), "salt", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Purged, "SQL purge runs on transport-level R2 failures")
	require.Len(t, repo.recordedAudits, 1)
	audit := repo.recordedAudits[0]
	assert.Empty(t, audit.PurgedKeys)
	assert.ElementsMatch(t, keys, audit.FailedKeys)
}

// TestPurgeOnce_StorageStep_NoStorage verifies the legacy DB-only
// behavior: when Storage is nil, no key-listing or BulkDelete call
// is attempted and no audit row is written. The SQL purge runs as
// before.
func TestPurgeOnce_StorageStep_NoStorage(t *testing.T) {
	userID := uuid.New()
	listKeysCalls := 0

	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{userID}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
		listKeysFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			listKeysCalls++
			return nil, nil
		},
	}

	svc := newServiceForTest(t, ServiceDeps{
		Repo: repo, Users: &stubUserRepo{},
		// Storage is intentionally nil
	})

	res, err := svc.PurgeOnce(context.Background(), "salt", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Purged)
	assert.Zero(t, listKeysCalls, "no storage -> no key listing")
	assert.Empty(t, repo.recordedAudits, "no storage -> no audit row")
}

// TestPurgeOnce_StorageStep_NoKeys exercises the edge case where the
// user has no uploaded media. An audit row is STILL written
// (compliance: the cron looked at the user) but it has zero keys.
func TestPurgeOnce_StorageStep_NoKeys(t *testing.T) {
	userID := uuid.New()
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{userID}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
		listKeysFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return nil, nil
		},
	}
	storage := &stubStorage{}

	svc := newServiceForTest(t, ServiceDeps{
		Repo: repo, Users: &stubUserRepo{}, Storage: storage,
	})

	res, err := svc.PurgeOnce(context.Background(), "salt", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Purged)
	assert.Empty(t, storage.bulkDeleteCall, "no keys -> no BulkDelete call")
	require.Len(t, repo.recordedAudits, 1, "audit still written so the cron can prove it ran")
	assert.Equal(t, 0, repo.recordedAudits[0].KeysCount())
}

// TestPurgeOnce_StorageStep_ListKeysError verifies that when the
// repo cannot enumerate keys (DB error mid-tx), the cron still
// writes a manifest with the error captured and proceeds with SQL
// anonymization.
func TestPurgeOnce_StorageStep_ListKeysError(t *testing.T) {
	userID := uuid.New()
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{userID}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
		listKeysFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return nil, errors.New("connection refused")
		},
	}
	storage := &stubStorage{}

	svc := newServiceForTest(t, ServiceDeps{
		Repo: repo, Users: &stubUserRepo{}, Storage: storage,
	})

	res, err := svc.PurgeOnce(context.Background(), "salt", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Purged, "SQL anonymization still runs on key-list failures")
	assert.Empty(t, storage.bulkDeleteCall)
	require.Len(t, repo.recordedAudits, 1)
	require.NotEmpty(t, repo.recordedAudits[0].Errors)
	assert.Contains(t, repo.recordedAudits[0].Errors[0], "list keys")
}

// TestStoragePurgeManifest_Validate ensures the domain entity rejects
// a manifest with a zero UUID — protects audit insertion from
// silently writing rows attributable to nobody.
func TestStoragePurgeManifest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		m       domaingdpr.StoragePurgeManifest
		wantErr error
	}{
		{
			name:    "zero user_id is rejected",
			m:       domaingdpr.StoragePurgeManifest{},
			wantErr: domaingdpr.ErrNoUserID,
		},
		{
			name: "valid manifest passes",
			m:    domaingdpr.StoragePurgeManifest{UserID: uuid.New()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestStoragePurgeManifest_Helpers exercises KeysCount + HasFailures.
func TestStoragePurgeManifest_Helpers(t *testing.T) {
	m := domaingdpr.StoragePurgeManifest{
		UserID:     uuid.New(),
		Keys:       []string{"a", "b", "c"},
		PurgedKeys: []string{"a", "b"},
		FailedKeys: []string{"c"},
	}
	assert.Equal(t, 3, m.KeysCount())
	assert.True(t, m.HasFailures())

	mClean := domaingdpr.StoragePurgeManifest{UserID: uuid.New()}
	assert.False(t, mClean.HasFailures())
	assert.Equal(t, 0, mClean.KeysCount())
}
