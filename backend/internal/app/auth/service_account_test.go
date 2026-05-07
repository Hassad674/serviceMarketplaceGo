package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
)

// --- ChangeEmail ---

func TestAuthService_ChangeEmail_Success(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "old@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}

	var updated *user.User
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			require.Equal(t, uid, id)
			return existing, nil
		},
		existsByEmailFn: func(_ context.Context, email string) (bool, error) {
			require.Equal(t, "new@example.com", email)
			return false, nil
		},
		updateFn: func(_ context.Context, u *user.User) error {
			updated = u
			return nil
		},
	}

	svc, auditRepo, sessionSvc := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)

	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewEmail:        "NEW@example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "new@example.com", result.Email, "email is normalised (lowercase)")
	require.NotNil(t, updated, "user should be persisted")
	assert.Equal(t, "new@example.com", updated.Email)

	// session kill switch
	bumps := userRepo.snapshotBumpCalls()
	require.Len(t, bumps, 1)
	assert.Equal(t, uid, bumps[0])
	deletes := sessionSvc.snapshotDeleteCalls()
	require.Len(t, deletes, 1)
	assert.Equal(t, uid, deletes[0])

	// audit
	entries := auditRepo.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionChangeEmail, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, uid, *entries[0].UserID)
	assert.Equal(t, "old@example.com", entries[0].Metadata["old_email"])
	assert.Equal(t, "new@example.com", entries[0].Metadata["new_email"])
}

func TestAuthService_ChangeEmail_WrongCurrentPassword(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "old@example.com",
		HashedPassword: "hashed_Right1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
	}

	svc, auditRepo, _ := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)

	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uid,
		CurrentPassword: "Wrong1Pass!",
		NewEmail:        "new@example.com",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Empty(t, auditRepo.snapshot(), "no audit row on credential failure")
	assert.Empty(t, userRepo.snapshotBumpCalls(), "no session bump on failure")
}

func TestAuthService_ChangeEmail_SameEmail(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "same@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	// Same after lowercasing — must still be rejected.
	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewEmail:        "SAME@example.com",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, user.ErrSameEmail)
}

func TestAuthService_ChangeEmail_AlreadyTaken(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "old@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewEmail:        "taken@example.com",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, user.ErrEmailAlreadyExists)
}

func TestAuthService_ChangeEmail_InvalidNewEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil)
			result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
				UserID:          uuid.New(),
				CurrentPassword: "CurrentPass1!",
				NewEmail:        tt.email,
			})
			assert.Nil(t, result)
			assert.ErrorIs(t, err, user.ErrInvalidEmail)
		})
	}
}

func TestAuthService_ChangeEmail_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uuid.New(),
		CurrentPassword: "CurrentPass1!",
		NewEmail:        "new@example.com",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrUserNotFound)
}

func TestAuthService_ChangeEmail_RaceOnUniqueIndex(t *testing.T) {
	// ExistsByEmail says "free", but the UPDATE trips the unique
	// index because a concurrent registration won the race. The
	// repository surfaces ErrEmailAlreadyExists — the service must
	// pass it through unchanged so the handler can map it to 409.
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "old@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		updateFn: func(_ context.Context, _ *user.User) error {
			return user.ErrEmailAlreadyExists
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewEmail:        "new@example.com",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, user.ErrEmailAlreadyExists)
}

func TestAuthService_ChangeEmail_SuspendedUser(t *testing.T) {
	uid := uuid.New()
	suspended := &user.User{
		ID:             uid,
		Email:          "old@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}
	suspended.Suspend("policy", nil)

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return suspended, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.ChangeEmail(context.Background(), ChangeEmailInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewEmail:        "new@example.com",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

// --- ChangePassword ---

func TestAuthService_ChangePassword_Success(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}

	var updated *user.User
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
		updateFn: func(_ context.Context, u *user.User) error {
			updated = u
			return nil
		},
	}

	svc, auditRepo, sessionSvc := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewPassword:     "NewStrong1Pass!",
	})

	require.NoError(t, err)
	require.NotNil(t, updated, "user should be persisted")
	assert.Equal(t, "hashed_NewStrong1Pass!", updated.HashedPassword)

	// session kill switch
	bumps := userRepo.snapshotBumpCalls()
	require.Len(t, bumps, 1)
	assert.Equal(t, uid, bumps[0])
	deletes := sessionSvc.snapshotDeleteCalls()
	require.Len(t, deletes, 1)
	assert.Equal(t, uid, deletes[0])

	// audit — strict invariant: no password material in metadata
	entries := auditRepo.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionChangePassword, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, uid, *entries[0].UserID)
	assert.Empty(t, entries[0].Metadata, "audit metadata must NOT carry password material")
}

func TestAuthService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_Right1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
	}

	svc, auditRepo, _ := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uid,
		CurrentPassword: "Wrong1Pass!",
		NewPassword:     "NewStrong1Pass!",
	})

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Empty(t, auditRepo.snapshot(), "no audit row on credential failure")
	assert.Empty(t, userRepo.snapshotBumpCalls(), "no session bump on failure")
}

func TestAuthService_ChangePassword_SamePassword(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewPassword:     "CurrentPass1!",
	})

	assert.ErrorIs(t, err, user.ErrSamePassword)
}

func TestAuthService_ChangePassword_WeakNewPassword(t *testing.T) {
	tests := []struct {
		name        string
		newPassword string
	}{
		{"too short", "Short1!"},
		{"no uppercase", "alllower1pass!"},
		{"no lowercase", "ALLUPPER1PASS!"},
		{"no digit", "NoDigitsHerePass!"},
		{"no special", "NoSpecial1Char"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil)
			err := svc.ChangePassword(context.Background(), ChangePasswordInput{
				UserID:          uuid.New(),
				CurrentPassword: "CurrentPass1!",
				NewPassword:     tt.newPassword,
			})
			assert.ErrorIs(t, err, user.ErrWeakPassword)
		})
	}
}

func TestAuthService_ChangePassword_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uuid.New(),
		CurrentPassword: "CurrentPass1!",
		NewPassword:     "NewStrong1Pass!",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrUserNotFound)
}

func TestAuthService_ChangePassword_HasherFailure(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
	}
	hasher := &mockHasher{
		// First call (verify current) succeeds. Second call (collision
		// check via Compare against new) returns invalid, third call
		// (Hash for the rotated password) fails.
		compareFn: func(hashed, password string) error {
			if hashed == "hashed_"+password {
				return nil
			}
			return user.ErrInvalidCredentials
		},
		hashFn: func(_ string) (string, error) {
			return "", errors.New("hash failure")
		},
	}

	svc := newTestService(userRepo, nil, hasher, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewPassword:     "NewStrong1Pass!",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hash")
}

func TestAuthService_ChangePassword_SuspendedUser(t *testing.T) {
	uid := uuid.New()
	suspended := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}
	suspended.Suspend("policy", nil)

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return suspended, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewPassword:     "NewStrong1Pass!",
	})

	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

// TestAuthService_ChangePassword_BumpFailureDoesNotFailRotation guards
// the policy: a Redis blip on session_version bump must not surface
// as a 5xx to the caller — the password is already rotated and
// refusing the call would put the user in a worse state.
func TestAuthService_ChangePassword_BumpFailureDoesNotFailRotation(t *testing.T) {
	uid := uuid.New()
	existing := &user.User{
		ID:             uid,
		Email:          "user@example.com",
		HashedPassword: "hashed_CurrentPass1!",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return existing, nil
		},
		bumpErr: fmt.Errorf("redis cluster down"),
	}

	svc, _, _ := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:          uid,
		CurrentPassword: "CurrentPass1!",
		NewPassword:     "NewStrong1Pass!",
	})

	assert.NoError(t, err, "rotation must succeed even when bump fails")
}
