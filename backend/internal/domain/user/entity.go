package user

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAgency     Role = "agency"
	RoleEnterprise Role = "enterprise"
	RoleProvider   Role = "provider"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAgency, RoleEnterprise, RoleProvider:
		return true
	}
	return false
}

func (r Role) String() string {
	return string(r)
}

type UserStatus string

const (
	StatusActive    UserStatus = "active"
	StatusSuspended UserStatus = "suspended"
	StatusBanned    UserStatus = "banned"
)

type User struct {
	ID              uuid.UUID
	Email           string
	HashedPassword  string
	FirstName       string
	LastName        string
	DisplayName     string
	Role            Role
	ReferrerEnabled bool
	IsAdmin             bool
	Status              UserStatus
	SuspendedAt         *time.Time
	SuspensionReason    string
	SuspensionExpiresAt *time.Time
	BannedAt            *time.Time
	BanReason           string
	OrganizationID      *uuid.UUID
	LinkedInID      *string
	GoogleID        *string
	EmailVerified   bool

	// Stripe Connect account (Embedded Components) — see migration 040.
	// All three are nil/empty until the user starts payment setup.
	StripeAccountID      *string
	StripeAccountCountry *string
	// StripeLastState is the last-seen Stripe account snapshot used by the
	// embedded Notifier to diff incoming webhooks. Opaque JSON, owned by
	// internal/app/embedded.
	StripeLastState []byte

	// KYC enforcement (migration 044). Set once when the first mission
	// completes with funds available. Used to compute the 14-day deadline.
	KYCFirstEarningAt       *time.Time
	KYCRestrictionNotifiedAt map[string]time.Time // tier → timestamp

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsKYCBlocked returns true if the user has earned available funds, has NOT
// completed KYC, and 14 days have elapsed since the first earning.
func (u *User) IsKYCBlocked() bool {
	if u.HasKYCCompleted() {
		return false
	}
	if u.KYCFirstEarningAt == nil {
		return false
	}
	return time.Since(*u.KYCFirstEarningAt) >= 14*24*time.Hour
}

// HasKYCCompleted returns true when a Stripe account exists.
func (u *User) HasKYCCompleted() bool {
	return u.StripeAccountID != nil && *u.StripeAccountID != ""
}

// KYCDaysRemaining returns the number of days before restriction kicks in.
// Returns -1 if not applicable (no earnings or KYC done).
// Returns 0 if already restricted.
func (u *User) KYCDaysRemaining() int {
	if u.HasKYCCompleted() || u.KYCFirstEarningAt == nil {
		return -1
	}
	remaining := 14*24*time.Hour - time.Since(*u.KYCFirstEarningAt)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}

// NewUser creates a new user with validated fields.
// Email and password validation should be done via value objects before calling this.
func NewUser(email string, hashedPassword string, firstName, lastName, displayName string, role Role) (*User, error) {
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}

	now := time.Now()
	return &User{
		ID:             uuid.New(),
		Email:          email,
		HashedPassword: hashedPassword,
		FirstName:      firstName,
		LastName:       lastName,
		DisplayName:    displayName,
		Role:           role,
		Status:         StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) EnableReferrer() {
	u.ReferrerEnabled = true
}

func (u *User) DisableReferrer() {
	u.ReferrerEnabled = false
}

func (u *User) CanBeReferrer() bool {
	return u.Role == RoleProvider
}

func (u *User) Suspend(reason string, expiresAt *time.Time) {
	now := time.Now()
	u.Status = StatusSuspended
	u.SuspendedAt = &now
	u.SuspensionReason = reason
	u.SuspensionExpiresAt = expiresAt
	u.UpdatedAt = now
}

func (u *User) Unsuspend() {
	u.Status = StatusActive
	u.SuspendedAt = nil
	u.SuspensionReason = ""
	u.SuspensionExpiresAt = nil
	u.UpdatedAt = time.Now()
}

func (u *User) Ban(reason string) {
	now := time.Now()
	u.Status = StatusBanned
	u.BannedAt = &now
	u.BanReason = reason
	u.UpdatedAt = now
}

func (u *User) Unban() {
	u.Status = StatusActive
	u.BannedAt = nil
	u.BanReason = ""
	u.UpdatedAt = time.Now()
}

func (u *User) IsSuspended() bool {
	if u.Status != StatusSuspended {
		return false
	}
	if u.SuspensionExpiresAt != nil && u.SuspensionExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

func (u *User) IsBanned() bool {
	return u.Status == StatusBanned
}
