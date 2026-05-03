package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- helpers ---

func newTestService(
	userRepo *mockUserRepo,
	resetRepo *mockPasswordResetRepo,
	hasher *mockHasher,
	tokens *mockTokenService,
	email *mockEmailService,
) *Service {
	if userRepo == nil {
		userRepo = &mockUserRepo{}
	}
	if resetRepo == nil {
		resetRepo = &mockPasswordResetRepo{}
	}
	if hasher == nil {
		hasher = &mockHasher{}
	}
	if tokens == nil {
		tokens = &mockTokenService{}
	}
	if email == nil {
		email = &mockEmailService{}
	}
	return NewService(userRepo, resetRepo, hasher, tokens, email, "https://example.com")
}

func validRegisterInput() RegisterInput {
	return RegisterInput{
		Email:       "test@example.com",
		Password:    "StrongPass1!",
		FirstName:   "John",
		LastName:    "Doe",
		DisplayName: "John D.",
		Role:        user.RoleProvider,
	}
}

// --- Register tests ---

func TestAuthService_Register_Success(t *testing.T) {
	var createdUser *user.User

	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		createFn: func(_ context.Context, u *user.User) error {
			createdUser = u
			return nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Register(context.Background(), validRegisterInput())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.User)
	assert.Equal(t, "test@example.com", result.User.Email)
	assert.Equal(t, "John", result.User.FirstName)
	assert.Equal(t, "Doe", result.User.LastName)
	assert.Equal(t, "John D.", result.User.DisplayName)
	assert.Equal(t, user.RoleProvider, result.User.Role)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.NotNil(t, createdUser, "user should be persisted")
}

func TestAuthService_Register_AllRoles(t *testing.T) {
	roles := []user.Role{user.RoleAgency, user.RoleEnterprise, user.RoleProvider}

	for _, role := range roles {
		t.Run("role_"+string(role), func(t *testing.T) {
			userRepo := &mockUserRepo{
				existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
					return false, nil
				},
			}

			svc := newTestService(userRepo, nil, nil, nil, nil)
			input := validRegisterInput()
			input.Role = role

			result, err := svc.Register(context.Background(), input)

			require.NoError(t, err)
			assert.Equal(t, role, result.User.Role)
		})
	}
}

func TestAuthService_Register_EmailAlreadyExists_SilentDuplicate(t *testing.T) {
	// F.5 S5: anti-enumeration. A duplicate registration must NOT
	// leak the typed ErrEmailAlreadyExists to the caller — the service
	// returns a SilentDuplicate output instead, and the handler maps
	// it to a neutral 202 Accepted. A probe cannot tell a fresh
	// registration apart from a duplicate via the response shape.
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Register(context.Background(), validRegisterInput())

	require.NoError(t, err, "duplicate must not leak via typed error")
	require.NotNil(t, result, "duplicate must return a neutral output")
	assert.True(t, result.SilentDuplicate, "duplicate flag must be set")
	assert.Nil(t, result.User, "no user payload on duplicate")
	assert.Empty(t, result.AccessToken, "no access token on duplicate")
	assert.Empty(t, result.RefreshToken, "no refresh token on duplicate")
}

// TestAuthService_Register_DoesNotEnumerate hardens F.5 S5: a duplicate
// registration MUST send a security-signal email to the legitimate
// owner of the address, and MUST NOT leak any field that distinguishes
// it from a fresh registration on the wire.
func TestAuthService_Register_DoesNotEnumerate(t *testing.T) {
	mockEmail := &mockEmailService{}
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(userRepo, nil, nil, nil, mockEmail)

	out, err := svc.Register(context.Background(), validRegisterInput())
	require.NoError(t, err)
	require.NotNil(t, out)
	require.True(t, out.SilentDuplicate)

	// Security signal email was dispatched to the legitimate owner.
	require.Len(t, mockEmail.notifications, 1, "duplicate must trigger one signal email")
	notif := mockEmail.notifications[0]
	assert.Equal(t, validRegisterInput().Email, notif.To)
	assert.NotEmpty(t, notif.Subject)
	assert.NotEmpty(t, notif.HTML)

	// Wire-shape check: the public-facing fields MUST be empty so the
	// handler emits a neutral 202 — distinguishable only via mock
	// inspection, never via the API response.
	assert.Empty(t, out.AccessToken)
	assert.Empty(t, out.RefreshToken)
	assert.Nil(t, out.OrganizationID)
	assert.Empty(t, out.OrgRole)
}

func TestAuthService_Register_WeakPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Short1!"},
		{"no uppercase", "alllower1"},
		{"no lowercase", "ALLUPPER1"},
		{"no digit", "NoDigitsHere"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil)
			input := validRegisterInput()
			input.Password = tt.password

			result, err := svc.Register(context.Background(), input)

			assert.ErrorIs(t, err, user.ErrWeakPassword)
			assert.Nil(t, result)
		})
	}
}

func TestAuthService_Register_InvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil)
			input := validRegisterInput()
			input.Email = tt.email

			result, err := svc.Register(context.Background(), input)

			assert.ErrorIs(t, err, user.ErrInvalidEmail)
			assert.Nil(t, result)
		})
	}
}

func TestAuthService_Register_InvalidRole(t *testing.T) {
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)
	input := validRegisterInput()
	input.Role = user.Role("invalid")

	result, err := svc.Register(context.Background(), input)

	assert.ErrorIs(t, err, user.ErrInvalidRole)
	assert.Nil(t, result)
}

func TestAuthService_Register_HasherFailure(t *testing.T) {
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}
	hasher := &mockHasher{
		hashFn: func(_ string) (string, error) {
			return "", fmt.Errorf("hasher internal error")
		},
	}

	svc := newTestService(userRepo, nil, hasher, nil, nil)

	result, err := svc.Register(context.Background(), validRegisterInput())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash password")
	assert.Nil(t, result)
}

func TestAuthService_Register_CreateUserFailure(t *testing.T) {
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		createFn: func(_ context.Context, _ *user.User) error {
			return fmt.Errorf("database connection lost")
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Register(context.Background(), validRegisterInput())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create user")
	assert.Nil(t, result)
}

func TestAuthService_Register_TokenGenerationFailure(t *testing.T) {
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}
	tokens := &mockTokenService{
		generateAccessFn: func(_ service.AccessTokenInput) (string, error) {
			return "", fmt.Errorf("token generation failed")
		},
	}

	svc := newTestService(userRepo, nil, nil, tokens, nil)

	result, err := svc.Register(context.Background(), validRegisterInput())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access token")
	assert.Nil(t, result)
}

func TestAuthService_Register_EmailLowercased(t *testing.T) {
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, email string) (bool, error) {
			// The email should arrive lowercased
			assert.Equal(t, "upper@example.com", email)
			return false, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)
	input := validRegisterInput()
	input.Email = "UPPER@EXAMPLE.COM"

	result, err := svc.Register(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "upper@example.com", result.User.Email)
}

func TestAuthService_Register_TrimsWhitespace(t *testing.T) {
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)
	input := RegisterInput{
		Email:       "test@example.com",
		Password:    "StrongPass1!",
		FirstName:   "  John  ",
		LastName:    "  Doe  ",
		DisplayName: "  John D.  ",
		Role:        user.RoleProvider,
	}

	result, err := svc.Register(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "John", result.User.FirstName)
	assert.Equal(t, "Doe", result.User.LastName)
	assert.Equal(t, "John D.", result.User.DisplayName)
}

// --- Login tests ---

func TestAuthService_Login_Success(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "login@example.com",
		HashedPassword: "hashed_CorrectPass1!",
		FirstName:      "Jane",
		LastName:       "Doe",
		Role:           user.RoleEnterprise,
	}

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, email string) (*user.User, error) {
			if email == "login@example.com" {
				return existingUser, nil
			}
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "login@example.com",
		Password: "CorrectPass1!",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, existingUser.ID, result.User.ID)
	assert.Equal(t, "login@example.com", result.User.Email)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "login@example.com",
		HashedPassword: "hashed_CorrectPass1!",
		Role:           user.RoleAgency,
	}

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "login@example.com",
		Password: "WrongPassword1!",
	})

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Nil(t, result)
}

func TestAuthService_Login_NonExistentEmail(t *testing.T) {
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "nobody@example.com",
		Password: "SomePassword1!",
	})

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Nil(t, result)
}

func TestAuthService_Login_InvalidEmailFormat(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "not-an-email",
		Password: "SomePassword1!",
	})

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Nil(t, result)
}

func TestAuthService_Login_EmptyEmail(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "",
		Password: "SomePassword1!",
	})

	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
	assert.Nil(t, result)
}

func TestAuthService_Login_TokenGenerationFailure(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "login@example.com",
		HashedPassword: "hashed_CorrectPass1!",
		Role:           user.RoleAgency,
	}

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	tokens := &mockTokenService{
		generateAccessFn: func(_ service.AccessTokenInput) (string, error) {
			return "", fmt.Errorf("signing key unavailable")
		},
	}

	svc := newTestService(userRepo, nil, nil, tokens, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "login@example.com",
		Password: "CorrectPass1!",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access token")
	assert.Nil(t, result)
}

// --- RefreshToken tests ---

func TestAuthService_RefreshToken_Success(t *testing.T) {
	existingUser := &user.User{
		ID:    uuid.New(),
		Email: "refresh@example.com",
		Role:  user.RoleProvider,
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == existingUser.ID {
				return existingUser, nil
			}
			return nil, user.ErrUserNotFound
		},
	}

	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:    existingUser.ID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
		generateAccessFn: func(_ service.AccessTokenInput) (string, error) {
			return "new_access_token", nil
		},
		generateRefreshFn: func(userID uuid.UUID) (string, error) {
			return "new_refresh_token", nil
		},
	}

	svc := newTestService(userRepo, nil, nil, tokens, nil)

	result, err := svc.RefreshToken(context.Background(), "valid_refresh_token")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, existingUser.ID, result.User.ID)
	assert.Equal(t, "new_access_token", result.AccessToken)
	assert.Equal(t, "new_refresh_token", result.RefreshToken)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return nil, fmt.Errorf("invalid token")
		},
	}

	svc := newTestService(nil, nil, nil, tokens, nil)

	result, err := svc.RefreshToken(context.Background(), "invalid_token")

	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, result)
}

func TestAuthService_RefreshToken_UserNotFound(t *testing.T) {
	missingID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:    missingID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, tokens, nil)

	result, err := svc.RefreshToken(context.Background(), "token_for_deleted_user")

	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, result)
}

func TestAuthService_RefreshToken_EmptyToken(t *testing.T) {
	tokens := &mockTokenService{
		validateRefreshFn: func(token string) (*service.TokenClaims, error) {
			return nil, fmt.Errorf("empty token")
		},
	}

	svc := newTestService(nil, nil, nil, tokens, nil)

	result, err := svc.RefreshToken(context.Background(), "")

	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, result)
}

// --- GetMe tests ---

func TestAuthService_GetMe_Success(t *testing.T) {
	expectedUser := &user.User{
		ID:        uuid.New(),
		Email:     "me@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.RoleAgency,
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == expectedUser.ID {
				return expectedUser, nil
			}
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.GetMe(context.Background(), expectedUser.ID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedUser.ID, result.ID)
	assert.Equal(t, "me@example.com", result.Email)
	assert.Equal(t, "John", result.FirstName)
}

func TestAuthService_GetMe_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.GetMe(context.Background(), uuid.New())

	assert.ErrorIs(t, err, user.ErrUserNotFound)
	assert.Nil(t, result)
}

// --- ForgotPassword tests ---

func TestAuthService_ForgotPassword_Success(t *testing.T) {
	existingUser := &user.User{
		ID:    uuid.New(),
		Email: "forgot@example.com",
	}

	var resetCreated bool
	var emailSent bool

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, email string) (*user.User, error) {
			if email == "forgot@example.com" {
				return existingUser, nil
			}
			return nil, user.ErrUserNotFound
		},
	}
	resetRepo := &mockPasswordResetRepo{
		createFn: func(_ context.Context, pr *repository.PasswordReset) error {
			resetCreated = true
			assert.Equal(t, existingUser.ID, pr.UserID)
			assert.NotEmpty(t, pr.Token)
			assert.True(t, pr.ExpiresAt.After(time.Now()))
			return nil
		},
	}
	emailSvc := &mockEmailService{
		sendPasswordResetFn: func(_ context.Context, to string, resetURL string) error {
			emailSent = true
			assert.Equal(t, "forgot@example.com", to)
			assert.Contains(t, resetURL, "https://example.com/reset-password?token=")
			return nil
		},
	}

	svc := newTestService(userRepo, resetRepo, nil, nil, emailSvc)

	err := svc.ForgotPassword(context.Background(), ForgotPasswordInput{
		Email: "forgot@example.com",
	})

	assert.NoError(t, err)
	assert.True(t, resetCreated, "reset token should be created")
	assert.True(t, emailSent, "email should be sent")
}

func TestAuthService_ForgotPassword_NonExistentEmail_NoError(t *testing.T) {
	// Must return nil to not reveal whether email exists
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	err := svc.ForgotPassword(context.Background(), ForgotPasswordInput{
		Email: "nobody@example.com",
	})

	assert.NoError(t, err, "should not reveal that email does not exist")
}

func TestAuthService_ForgotPassword_InvalidEmail_NoError(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	err := svc.ForgotPassword(context.Background(), ForgotPasswordInput{
		Email: "invalid-email",
	})

	assert.NoError(t, err, "should not reveal email validation details")
}

// --- ResetPassword tests ---

func TestAuthService_ResetPassword_Success(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "reset@example.com",
		HashedPassword: "old_hashed_password",
	}
	resetID := uuid.New()

	var passwordUpdated bool
	var tokenMarkedUsed bool

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == existingUser.ID {
				return existingUser, nil
			}
			return nil, user.ErrUserNotFound
		},
		updateFn: func(_ context.Context, u *user.User) error {
			passwordUpdated = true
			assert.NotEqual(t, "old_hashed_password", u.HashedPassword)
			return nil
		},
	}
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, token string) (*repository.PasswordReset, error) {
			if token == "valid-reset-token" {
				return &repository.PasswordReset{
					ID:        resetID,
					UserID:    existingUser.ID,
					Token:     "valid-reset-token",
					ExpiresAt: time.Now().Add(30 * time.Minute),
					Used:      false,
				}, nil
			}
			return nil, user.ErrUnauthorized
		},
		markUsedFn: func(_ context.Context, id uuid.UUID) error {
			tokenMarkedUsed = true
			assert.Equal(t, resetID, id)
			return nil
		},
	}

	svc := newTestService(userRepo, resetRepo, nil, nil, nil)

	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "valid-reset-token",
		NewPassword: "NewStrongPass1!",
	})

	assert.NoError(t, err)
	assert.True(t, passwordUpdated, "password should be updated")
	assert.True(t, tokenMarkedUsed, "reset token should be marked as used")
}

func TestAuthService_ResetPassword_WeakNewPassword(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "some-token",
		NewPassword: "weak",
	})

	assert.ErrorIs(t, err, user.ErrWeakPassword)
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, _ string) (*repository.PasswordReset, error) {
			return nil, user.ErrUnauthorized
		},
	}

	svc := newTestService(nil, resetRepo, nil, nil, nil)

	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "invalid-token",
		NewPassword: "NewStrongPass1!",
	})

	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

func TestAuthService_ResetPassword_ExpiredToken(t *testing.T) {
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, _ string) (*repository.PasswordReset, error) {
			return &repository.PasswordReset{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				Token:     "expired-token",
				ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
				Used:      false,
			}, nil
		},
	}

	svc := newTestService(nil, resetRepo, nil, nil, nil)

	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "expired-token",
		NewPassword: "NewStrongPass1!",
	})

	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

func TestAuthService_ResetPassword_AlreadyUsedToken(t *testing.T) {
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, _ string) (*repository.PasswordReset, error) {
			return &repository.PasswordReset{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				Token:     "used-token",
				ExpiresAt: time.Now().Add(30 * time.Minute),
				Used:      true, // already used
			}, nil
		},
	}

	svc := newTestService(nil, resetRepo, nil, nil, nil)

	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "used-token",
		NewPassword: "NewStrongPass1!",
	})

	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

// --- EnableReferrer tests ---

func TestAuthService_EnableReferrer_Success(t *testing.T) {
	providerUser := &user.User{
		ID:              uuid.New(),
		Email:           "provider@example.com",
		Role:            user.RoleProvider,
		ReferrerEnabled: false,
	}

	var updatedUser *user.User

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == providerUser.ID {
				return providerUser, nil
			}
			return nil, user.ErrUserNotFound
		},
		updateFn: func(_ context.Context, u *user.User) error {
			updatedUser = u
			return nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.EnableReferrer(context.Background(), providerUser.ID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.ReferrerEnabled, "referrer should be enabled")
	assert.NotNil(t, updatedUser, "user should be persisted")
	assert.True(t, updatedUser.ReferrerEnabled, "persisted user should have referrer enabled")
}

func TestAuthService_EnableReferrer_NonProviderRole(t *testing.T) {
	tests := []struct {
		name string
		role user.Role
	}{
		{"agency cannot be referrer", user.RoleAgency},
		{"enterprise cannot be referrer", user.RoleEnterprise},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonProviderUser := &user.User{
				ID:   uuid.New(),
				Role: tt.role,
			}

			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					return nonProviderUser, nil
				},
			}

			svc := newTestService(userRepo, nil, nil, nil, nil)

			result, err := svc.EnableReferrer(context.Background(), nonProviderUser.ID)

			assert.ErrorIs(t, err, user.ErrInvalidRole)
			assert.Nil(t, result)
		})
	}
}

func TestAuthService_EnableReferrer_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.EnableReferrer(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enable referrer")
	assert.Nil(t, result)
}

func TestAuthService_EnableReferrer_UpdateFailure(t *testing.T) {
	providerUser := &user.User{
		ID:   uuid.New(),
		Role: user.RoleProvider,
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return providerUser, nil
		},
		updateFn: func(_ context.Context, _ *user.User) error {
			return fmt.Errorf("database connection lost")
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.EnableReferrer(context.Background(), providerUser.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enable referrer")
	assert.Nil(t, result)
}

func TestAuthService_EnableReferrer_AlreadyEnabled(t *testing.T) {
	// Should succeed idempotently even if already enabled
	providerUser := &user.User{
		ID:              uuid.New(),
		Role:            user.RoleProvider,
		ReferrerEnabled: true,
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return providerUser, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.EnableReferrer(context.Background(), providerUser.ID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.ReferrerEnabled)
}

func TestAuthService_EnableReferrer_SetsUpdatedAt(t *testing.T) {
	providerUser := &user.User{
		ID:        uuid.New(),
		Role:      user.RoleProvider,
		UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return providerUser, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.EnableReferrer(context.Background(), providerUser.ID)

	require.NoError(t, err)
	assert.True(t, result.UpdatedAt.After(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		"updated_at should be refreshed to a recent time")
}

// --- Login: suspended / banned user tests ---

func TestAuthService_Login_SuspendedUser_ReturnsSuspendedError(t *testing.T) {
	suspendedUser := &user.User{
		ID:             uuid.New(),
		Email:          "suspended@example.com",
		HashedPassword: "hashed_CorrectPass1!",
		Role:           user.RoleProvider,
		Status:         user.StatusActive,
	}
	suspendedUser.Suspend("policy violation", nil)

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return suspendedUser, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "suspended@example.com",
		Password: "CorrectPass1!",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrAccountSuspended)

	var statusErr *user.AccountStatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, "policy violation", statusErr.Reason)
}

func TestAuthService_Login_BannedUser_ReturnsBannedError(t *testing.T) {
	bannedUser := &user.User{
		ID:             uuid.New(),
		Email:          "banned@example.com",
		HashedPassword: "hashed_CorrectPass1!",
		Role:           user.RoleAgency,
		Status:         user.StatusActive,
	}
	bannedUser.Ban("repeated violations")

	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return bannedUser, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, nil, nil)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:    "banned@example.com",
		Password: "CorrectPass1!",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrAccountBanned)

	var statusErr *user.AccountStatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, "repeated violations", statusErr.Reason)
}

// --- RefreshToken: suspended / banned user tests ---

func TestAuthService_RefreshToken_SuspendedUser_ReturnsError(t *testing.T) {
	suspendedUser := &user.User{
		ID:     uuid.New(),
		Email:  "suspended@example.com",
		Role:   user.RoleProvider,
		Status: user.StatusActive,
	}
	suspendedUser.Suspend("auto-suspension", nil)

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == suspendedUser.ID {
				return suspendedUser, nil
			}
			return nil, user.ErrUserNotFound
		},
	}
	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:    suspendedUser.ID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, tokens, nil)

	result, err := svc.RefreshToken(context.Background(), "valid_refresh_token")

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrAccountSuspended)
}

func TestAuthService_RefreshToken_BannedUser_ReturnsError(t *testing.T) {
	bannedUser := &user.User{
		ID:     uuid.New(),
		Email:  "banned@example.com",
		Role:   user.RoleEnterprise,
		Status: user.StatusActive,
	}
	bannedUser.Ban("permanent ban")

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == bannedUser.ID {
				return bannedUser, nil
			}
			return nil, user.ErrUserNotFound
		},
	}
	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:    bannedUser.ID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
	}

	svc := newTestService(userRepo, nil, nil, tokens, nil)

	result, err := svc.RefreshToken(context.Background(), "valid_refresh_token")

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrAccountBanned)
}

// ---------------------------------------------------------------------
// SEC-13: audit log emission for auth events
// ---------------------------------------------------------------------

// newTestServiceWithAudit returns a service with an audit + session
// service wired in, exposing both mocks so the test can assert on
// emitted entries / DeleteByUserID calls.
func newTestServiceWithAudit(
	userRepo *mockUserRepo,
	resetRepo *mockPasswordResetRepo,
	hasher *mockHasher,
	tokens *mockTokenService,
	email *mockEmailService,
) (*Service, *mockAuditRepo, *mockSessionService) {
	if userRepo == nil {
		userRepo = &mockUserRepo{}
	}
	if resetRepo == nil {
		resetRepo = &mockPasswordResetRepo{}
	}
	if hasher == nil {
		hasher = &mockHasher{}
	}
	if tokens == nil {
		tokens = &mockTokenService{}
	}
	if email == nil {
		email = &mockEmailService{}
	}
	auditRepo := &mockAuditRepo{}
	sessionSvc := &mockSessionService{}
	svc := NewServiceWithDeps(ServiceDeps{
		Users:       userRepo,
		Resets:      resetRepo,
		Hasher:      hasher,
		Tokens:      tokens,
		Email:       email,
		Sessions:    sessionSvc,
		Audits:      auditRepo,
		FrontendURL: "https://example.com",
	})
	return svc, auditRepo, sessionSvc
}

func TestAuthService_Login_EmitsLoginSuccessAudit(t *testing.T) {
	existingUser := &user.User{
		ID:             uuid.New(),
		Email:          "audit@example.com",
		HashedPassword: "hashed_CorrectPass1",
		Role:           user.RoleProvider,
	}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}

	svc, auditRepo, _ := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)

	_, err := svc.Login(context.Background(), LoginInput{
		Email:    "audit@example.com",
		Password: "CorrectPass1",
	})
	require.NoError(t, err)

	entries := auditRepo.snapshot()
	require.Len(t, entries, 1, "exactly one audit row expected on login success")
	assert.Equal(t, audit.ActionLoginSuccess, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, existingUser.ID, *entries[0].UserID)
	assert.Equal(t, "audit@example.com", entries[0].Metadata["email"])
}

func TestAuthService_Login_FailureEmitsLoginFailureAudit(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*mockUserRepo, LoginInput)
		wantReason string
	}{
		{
			name: "invalid email format",
			setup: func() (*mockUserRepo, LoginInput) {
				return &mockUserRepo{}, LoginInput{Email: "not-an-email", Password: "x"}
			},
			wantReason: "invalid_email_format",
		},
		{
			name: "user not found",
			setup: func() (*mockUserRepo, LoginInput) {
				repo := &mockUserRepo{
					getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
						return nil, user.ErrUserNotFound
					},
				}
				return repo, LoginInput{Email: "ghost@example.com", Password: "Password1"}
			},
			wantReason: "user_not_found",
		},
		{
			name: "invalid password",
			setup: func() (*mockUserRepo, LoginInput) {
				u := &user.User{ID: uuid.New(), Email: "real@example.com", HashedPassword: "hashed_Right1"}
				repo := &mockUserRepo{
					getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
						return u, nil
					},
				}
				return repo, LoginInput{Email: "real@example.com", Password: "Wrong1Pass"}
			},
			wantReason: "invalid_password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, input := tt.setup()
			svc, auditRepo, _ := newTestServiceWithAudit(repo, nil, nil, nil, nil)

			_, err := svc.Login(context.Background(), input)
			require.Error(t, err)

			entries := auditRepo.snapshot()
			require.Len(t, entries, 1)
			assert.Equal(t, audit.ActionLoginFailure, entries[0].Action)
			assert.Equal(t, tt.wantReason, entries[0].Metadata["reason"])
		})
	}
}

func TestAuthService_Login_FiveFailuresProduceFiveAuditRows(t *testing.T) {
	repo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	svc, auditRepo, _ := newTestServiceWithAudit(repo, nil, nil, nil, nil)

	for i := 0; i < 5; i++ {
		_, err := svc.Login(context.Background(), LoginInput{
			Email:    fmt.Sprintf("attempt%d@example.com", i),
			Password: "BadPassword1",
		})
		require.Error(t, err)
	}

	entries := auditRepo.snapshot()
	assert.Len(t, entries, 5, "5 failed logins produce 5 audit rows")
	for _, e := range entries {
		assert.Equal(t, audit.ActionLoginFailure, e.Action)
	}
}

func TestAuthService_RefreshToken_EmitsTokenRefreshAudit(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "ref@example.com", Role: user.RoleProvider}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return u, nil
		},
	}
	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}

	svc, auditRepo, _ := newTestServiceWithAudit(userRepo, nil, nil, tokens, nil)
	_, err := svc.RefreshToken(context.Background(), "any-token")
	require.NoError(t, err)

	entries := auditRepo.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionTokenRefresh, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, u.ID, *entries[0].UserID)
}

func TestAuthService_Logout_EmitsLogoutAudit(t *testing.T) {
	svc, auditRepo, _ := newTestServiceWithAudit(nil, nil, nil, nil, nil)
	userID := uuid.New()
	svc.Logout(context.Background(), userID)

	entries := auditRepo.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionLogout, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, userID, *entries[0].UserID)
}

func TestAuthService_ForgotPassword_EmitsPasswordResetRequestAudit(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "forgot@example.com"}
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return u, nil
		},
	}

	svc, auditRepo, _ := newTestServiceWithAudit(userRepo, nil, nil, nil, nil)
	err := svc.ForgotPassword(context.Background(), ForgotPasswordInput{Email: "forgot@example.com"})
	require.NoError(t, err)

	entries := auditRepo.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionPasswordResetRequest, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, u.ID, *entries[0].UserID)
	assert.Equal(t, "forgot@example.com", entries[0].Metadata["email"])
}

// ---------------------------------------------------------------------
// SEC-16: ResetPassword bumps session_version + purges sessions + audit
// ---------------------------------------------------------------------

func TestAuthService_ResetPassword_BumpsSessionVersionAndPurgesSessions(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "reset@example.com", HashedPassword: "old_hashed"}
	resetID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil },
		updateFn:  func(_ context.Context, _ *user.User) error { return nil },
	}
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, _ string) (*repository.PasswordReset, error) {
			return &repository.PasswordReset{
				ID:        resetID,
				UserID:    u.ID,
				Token:     "valid-token",
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	svc, auditRepo, sessionSvc := newTestServiceWithAudit(userRepo, resetRepo, nil, nil, nil)
	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "valid-token",
		NewPassword: "NewStrong1Pass!",
	})
	require.NoError(t, err)

	// SEC-16 invariant 1: session_version was bumped exactly once
	bumpCalls := userRepo.snapshotBumpCalls()
	require.Len(t, bumpCalls, 1, "expected 1 BumpSessionVersion call")
	assert.Equal(t, u.ID, bumpCalls[0])

	// SEC-16 invariant 2: every Redis session for the user was deleted
	deleteCalls := sessionSvc.snapshotDeleteCalls()
	require.Len(t, deleteCalls, 1, "expected 1 DeleteByUserID call")
	assert.Equal(t, u.ID, deleteCalls[0])

	// SEC-13 invariant: audit row written
	entries := auditRepo.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionPasswordResetComplete, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, u.ID, *entries[0].UserID)
}

// TestAuthService_ResetPassword_BumpFailureDoesNotFailReset documents
// the policy: a BumpSessionVersion failure is logged but doesn't fail
// the reset itself — refusing the call would put the user in a worse
// state ("password is changed but… error?"). This guards the policy
// against an over-eager refactor that propagates the error.
func TestAuthService_ResetPassword_BumpFailureDoesNotFailReset(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "r@example.com", HashedPassword: "h"}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil },
		updateFn:  func(_ context.Context, _ *user.User) error { return nil },
		bumpErr:   fmt.Errorf("redis cluster down"),
	}
	resetRepo := &mockPasswordResetRepo{
		getByTokenFn: func(_ context.Context, _ string) (*repository.PasswordReset, error) {
			return &repository.PasswordReset{
				ID:        uuid.New(),
				UserID:    u.ID,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	svc, _, _ := newTestServiceWithAudit(userRepo, resetRepo, nil, nil, nil)
	err := svc.ResetPassword(context.Background(), ResetPasswordInput{
		Token:       "x",
		NewPassword: "NewStrong1Pass!",
	})
	assert.NoError(t, err, "reset must succeed even when bump fails")
}
