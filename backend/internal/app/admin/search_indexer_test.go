package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

// newSearchSyncService builds an admin Service wired with the search
// sync collaborators on top of the base mocks. The org resolver maps
// every user to `org` so the happy-path assertions can target a known
// organization id.
func newSearchSyncService(org *organization.Organization, resolveErr error) (
	*Service,
	*mockUserRepo,
	*mockActorSearchIndexer,
) {
	users := &mockUserRepo{}
	indexer := &mockActorSearchIndexer{}
	resolver := &mockOrgResolver{
		findFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			if resolveErr != nil {
				return nil, resolveErr
			}
			return org, nil
		},
	}
	svc := NewService(ServiceDeps{
		Users:       users,
		Audit:       &mockAuditRepo{},
		SessionSvc:  &mockSessionService{},
		Broadcaster: &mockBroadcaster{},
	}).WithActorSearchIndexer(indexer, resolver)
	return svc, users, indexer
}

func makeOrg(ownerID uuid.UUID) *organization.Organization {
	return &organization.Organization{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
	}
}

// TestAdminService_ModerationSearchSync is the table-driven contract
// for the moderation ↔ Typesense sync:
//
//   - suspend / ban  → RemoveActor(orgID) called exactly once
//   - unsuspend / unban (user active again) → ReindexActor(orgID) once
//   - the org id passed to the indexer is the org the user OWNS
//     (proves delete/upsert target the right document)
func TestAdminService_ModerationSearchSync(t *testing.T) {
	uid := uuid.New()
	adminID := uuid.New()

	type call struct {
		name      string
		invoke    func(svc *Service) error
		wantRemove bool
		wantReindex bool
	}

	cases := []call{
		{
			name: "suspend removes actor",
			invoke: func(svc *Service) error {
				return svc.SuspendUser(context.Background(), adminID, uid, "spam", nil)
			},
			wantRemove: true,
		},
		{
			name: "ban removes actor",
			invoke: func(svc *Service) error {
				return svc.BanUser(context.Background(), adminID, uid, "fraud")
			},
			wantRemove: true,
		},
		{
			name: "unsuspend reindexes actor",
			invoke: func(svc *Service) error {
				return svc.UnsuspendUser(context.Background(), adminID, uid)
			},
			wantReindex: true,
		},
		{
			name: "unban reindexes actor",
			invoke: func(svc *Service) error {
				return svc.UnbanUser(context.Background(), adminID, uid)
			},
			wantReindex: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			org := makeOrg(uid)
			svc, users, indexer := newSearchSyncService(org, nil)
			users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return makeUser(uid), nil
			}

			require.NoError(t, tc.invoke(svc))

			removeCalls := indexer.snapshotRemoveCalls()
			reindexCalls := indexer.snapshotReindexCalls()

			if tc.wantRemove {
				require.Len(t, removeCalls, 1, "RemoveActor must fire once")
				assert.Equal(t, org.ID, removeCalls[0],
					"RemoveActor must target the org the user owns")
				assert.Empty(t, reindexCalls)
			}
			if tc.wantReindex {
				require.Len(t, reindexCalls, 1, "ReindexActor must fire once")
				assert.Equal(t, org.ID, reindexCalls[0],
					"ReindexActor must target the org the user owns")
				assert.Empty(t, removeCalls)
			}
		})
	}
}

// TestAdminService_SearchSync_FailureIsNonBlocking proves a Typesense
// publish failure NEVER fails the moderation action — the DB status
// flip is the source of truth.
func TestAdminService_SearchSync_FailureIsNonBlocking(t *testing.T) {
	uid := uuid.New()
	adminID := uuid.New()

	t.Run("suspend succeeds despite RemoveActor error", func(t *testing.T) {
		org := makeOrg(uid)
		svc, users, indexer := newSearchSyncService(org, nil)
		indexer.removeErr = errors.New("typesense down")
		users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return makeUser(uid), nil
		}

		err := svc.SuspendUser(context.Background(), adminID, uid, "spam", nil)
		require.NoError(t, err, "search failure must not fail the suspension")

		// The DB status flip still happened.
		updates := users.snapshotUpdateCalls()
		require.Len(t, updates, 1)
		assert.Equal(t, user.StatusSuspended, updates[0].Status)
		// And the indexer was still attempted.
		assert.Len(t, indexer.snapshotRemoveCalls(), 1)
	})

	t.Run("unban succeeds despite ReindexActor error", func(t *testing.T) {
		org := makeOrg(uid)
		svc, users, indexer := newSearchSyncService(org, nil)
		indexer.reindexErr = errors.New("typesense down")
		users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return makeUser(uid), nil
		}

		err := svc.UnbanUser(context.Background(), adminID, uid)
		require.NoError(t, err, "search failure must not fail the unban")

		updates := users.snapshotUpdateCalls()
		require.Len(t, updates, 1)
		assert.Equal(t, user.StatusActive, updates[0].Status)
		assert.Len(t, indexer.snapshotReindexCalls(), 1)
	})

	t.Run("org resolution failure is swallowed", func(t *testing.T) {
		svc, users, indexer := newSearchSyncService(nil, errors.New("db blip"))
		users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return makeUser(uid), nil
		}

		err := svc.BanUser(context.Background(), adminID, uid, "fraud")
		require.NoError(t, err, "org resolve failure must not fail the ban")
		assert.Empty(t, indexer.snapshotRemoveCalls(),
			"no indexer call when the org cannot be resolved")
	})

	t.Run("user owning no org is a clean no-op", func(t *testing.T) {
		svc, users, indexer := newSearchSyncService(nil, organization.ErrOrgNotFound)
		users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return makeUser(uid), nil
		}

		err := svc.SuspendUser(context.Background(), adminID, uid, "spam", nil)
		require.NoError(t, err)
		assert.Empty(t, indexer.snapshotRemoveCalls())
		assert.Empty(t, indexer.snapshotReindexCalls())
	})

	t.Run("resolver returning nil org without error is a clean no-op", func(t *testing.T) {
		// Defensive: a resolver that returns (nil, nil) must not panic
		// and must not call the indexer.
		svc, users, indexer := newSearchSyncService(nil, nil) // org nil, err nil
		users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return makeUser(uid), nil
		}

		err := svc.BanUser(context.Background(), adminID, uid, "fraud")
		require.NoError(t, err)
		assert.Empty(t, indexer.snapshotRemoveCalls())
	})
}

// TestAdminService_SearchSync_NilIndexerNoPanic proves the admin
// service stays bootable without the search engine: when no indexer /
// resolver is wired, the moderation actions run normally and never
// panic.
func TestAdminService_SearchSync_NilIndexerNoPanic(t *testing.T) {
	svc, users, _, _, _ := newTestService() // no WithActorSearchIndexer
	uid := uuid.New()
	adminID := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}

	assert.NotPanics(t, func() {
		require.NoError(t, svc.SuspendUser(context.Background(), adminID, uid, "spam", nil))
		require.NoError(t, svc.BanUser(context.Background(), adminID, uid, "fraud"))
		require.NoError(t, svc.UnsuspendUser(context.Background(), adminID, uid))
		require.NoError(t, svc.UnbanUser(context.Background(), adminID, uid))
	})
}

// TestAdminService_reindexActorIfActive_StatusGuard exercises the
// StatusActive guard directly: a non-active (or nil) user must NOT be
// resurrected in the index, an active one must. This guarantees an
// unban cannot revive an actor that is still suspended (and vice
// versa) if the two flags ever diverge.
func TestAdminService_reindexActorIfActive_StatusGuard(t *testing.T) {
	uid := uuid.New()
	org := makeOrg(uid)

	tests := []struct {
		name       string
		user       *user.User
		wantCalled bool
	}{
		{
			name:       "nil user → no reindex",
			user:       nil,
			wantCalled: false,
		},
		{
			name:       "suspended user → no reindex",
			user:       &user.User{ID: uid, Status: user.StatusSuspended},
			wantCalled: false,
		},
		{
			name:       "banned user → no reindex",
			user:       &user.User{ID: uid, Status: user.StatusBanned},
			wantCalled: false,
		},
		{
			name:       "active user → reindex fires",
			user:       &user.User{ID: uid, Status: user.StatusActive},
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, indexer := newSearchSyncService(org, nil)
			svc.reindexActorIfActive(context.Background(), tt.user, "unit")
			calls := indexer.snapshotReindexCalls()
			if tt.wantCalled {
				require.Len(t, calls, 1)
				assert.Equal(t, org.ID, calls[0])
			} else {
				assert.Empty(t, calls)
			}
		})
	}
}

// TestAdminService_SearchSync_PartialWiringDisabled proves that
// wiring only one of (indexer, resolver) disables the sync — the
// invariant documented on WithActorSearchIndexer.
func TestAdminService_SearchSync_PartialWiringDisabled(t *testing.T) {
	uid := uuid.New()
	adminID := uuid.New()
	org := makeOrg(uid)
	indexer := &mockActorSearchIndexer{}

	svc := NewService(ServiceDeps{
		Users:       &mockUserRepo{getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return makeUser(uid), nil }},
		Audit:       &mockAuditRepo{},
		SessionSvc:  &mockSessionService{},
		Broadcaster: &mockBroadcaster{},
	}).WithActorSearchIndexer(indexer, nil) // resolver nil → disabled

	require.NoError(t, svc.SuspendUser(context.Background(), adminID, uid, "spam", nil))
	assert.Empty(t, indexer.snapshotRemoveCalls(),
		"nil resolver must disable the sync even with an indexer wired")
	_ = org
}
