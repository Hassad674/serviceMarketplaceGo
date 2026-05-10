package session

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLoginMethod_IsValid(t *testing.T) {
	cases := []struct {
		name string
		m    LoginMethod
		want bool
	}{
		{"password", LoginMethodPassword, true},
		{"invitation", LoginMethodInvitation, true},
		{"token_bridge", LoginMethodTokenBridge, true},
		{"refresh", LoginMethodRefresh, true},
		{"admin_impersonation", LoginMethodAdminImpersonation, true},
		{"empty rejected", LoginMethod(""), false},
		{"unknown rejected", LoginMethod("sso"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.m.IsValid())
		})
	}
}

func validInput() NewInput {
	return NewInput{
		UserID:        uuid.New(),
		JTI:           uuid.New().String(),
		UserAgentHash: "deadbeefcafef00d",
		IPAnonymized:  "192.0.2.0/24",
		LoginMethod:   LoginMethodPassword,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}
}

func TestNew_validation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*NewInput)
		wantErr error
	}{
		{
			name:   "valid password session",
			mutate: func(*NewInput) {},
		},
		{
			name:    "missing user id",
			mutate:  func(in *NewInput) { in.UserID = uuid.Nil },
			wantErr: ErrUserIDRequired,
		},
		{
			name:    "missing jti",
			mutate:  func(in *NewInput) { in.JTI = "" },
			wantErr: ErrJTIRequired,
		},
		{
			name:    "missing user agent",
			mutate:  func(in *NewInput) { in.UserAgentHash = "" },
			wantErr: ErrUserAgentRequired,
		},
		{
			name:    "missing ip",
			mutate:  func(in *NewInput) { in.IPAnonymized = "" },
			wantErr: ErrIPRequired,
		},
		{
			name:    "blank ip",
			mutate:  func(in *NewInput) { in.IPAnonymized = "   " },
			wantErr: ErrIPRequired,
		},
		{
			name:    "invalid method",
			mutate:  func(in *NewInput) { in.LoginMethod = LoginMethod("sso") },
			wantErr: ErrInvalidLoginMethod,
		},
		{
			name:    "expires in the past",
			mutate:  func(in *NewInput) { in.ExpiresAt = time.Now().Add(-time.Hour) },
			wantErr: ErrExpiresAtPast,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)
			s, err := New(in)
			if tc.wantErr != nil {
				assert.Nil(t, s)
				assert.True(t, errors.Is(err, tc.wantErr), "want %v got %v", tc.wantErr, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, s)
			assert.NotEqual(t, uuid.Nil, s.ID)
			assert.False(t, s.CreatedAt.IsZero())
			assert.Equal(t, s.CreatedAt, s.LastUsedAt)
			assert.Nil(t, s.RevokedAt)
		})
	}
}

func TestSession_Active(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	revoked := now.Add(-time.Hour)
	cases := []struct {
		name string
		s    Session
		want bool
	}{
		{
			name: "fresh session is active",
			s:    Session{ExpiresAt: now.Add(time.Hour)},
			want: true,
		},
		{
			name: "expired session is inactive",
			s:    Session{ExpiresAt: now.Add(-time.Minute)},
			want: false,
		},
		{
			name: "revoked but not yet expired is inactive",
			s:    Session{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked},
			want: false,
		},
		{
			name: "revoked and expired is inactive",
			s:    Session{ExpiresAt: now.Add(-time.Hour), RevokedAt: &revoked},
			want: false,
		},
		{
			name: "exactly at expiry counts as expired",
			s:    Session{ExpiresAt: now},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.Active(now))
		})
	}
}

func TestLoginMethod_String(t *testing.T) {
	assert.Equal(t, "password", LoginMethodPassword.String())
	assert.Equal(t, "refresh", LoginMethodRefresh.String())
}
