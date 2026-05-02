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

// AccountType distinguishes between users who self-registered in a
// marketplace role (agencies, enterprises, providers) and operators who
// were invited into an existing organization and have no standalone
// marketplace identity.
//
// Operators inherit their org's marketplace role (agency or enterprise)
// in the Role field, so existing queries that filter by role keep working
// naturally. AccountType is the orthogonal dimension that tells us whether
// the user is the founder of their account or a delegated operator.
type AccountType string

const (
	// AccountTypeMarketplaceOwner is a user who self-registered
	// as an Agency, Enterprise, or Provider. They own their account.
	AccountTypeMarketplaceOwner AccountType = "marketplace_owner"

	// AccountTypeOperator is a user who was invited into an existing
	// organization. They have no independent marketplace identity
	// (no public profile, not searchable, no personal wallet).
	AccountTypeOperator AccountType = "operator"
)

func (a AccountType) IsValid() bool {
	return a == AccountTypeMarketplaceOwner || a == AccountTypeOperator
}

func (a AccountType) String() string {
	return string(a)
}

type User struct {
	ID              uuid.UUID
	Email           string
	HashedPassword  string
	FirstName       string
	LastName        string
	DisplayName     string
	Role            Role
	AccountType     AccountType
	ReferrerEnabled bool

	// SessionVersion is incremented every time the user's effective
	// permissions change (role bumped/demoted, removed from org,
	// suspended, password changed). The JWT carries the session version
	// at issuance time, and the auth middleware compares against the
	// current value on every request. A mismatch triggers immediate 401
	// — this is the revocation mechanism that gives us "immediate"
	// effect on sensitive security actions.
	SessionVersion int

	EmailNotificationsEnabled bool

	IsAdmin             bool
	Status              UserStatus
	SuspendedAt         *time.Time
	SuspensionReason    string
	SuspensionExpiresAt *time.Time
	BannedAt            *time.Time
	BanReason           string
	OrganizationID      *uuid.UUID
	LinkedInID          *string
	GoogleID            *string
	EmailVerified       bool

	// DeletedAt anchors the GDPR soft-delete flow (migration 132).
	// Set to a non-nil timestamp when the user confirms deletion via
	// the email link; cleared when they cancel. The daily purge cron
	// hard-deletes when DeletedAt is older than 30 days. While set,
	// every read filters the user out and login is refused with the
	// account_scheduled_for_deletion code.
	DeletedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsScheduledForDeletion reports whether the GDPR soft-delete flag
// is currently set. Used by login + middleware to refuse access.
func (u *User) IsScheduledForDeletion() bool {
	return u.DeletedAt != nil
}

// NewUser creates a new user with validated fields.
// Email and password validation should be done via value objects before calling this.
func NewUser(email string, hashedPassword string, firstName, lastName, displayName string, role Role) (*User, error) {
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}

	now := time.Now()
	return &User{
		ID:                        uuid.New(),
		Email:                     email,
		HashedPassword:            hashedPassword,
		FirstName:                 firstName,
		LastName:                  lastName,
		DisplayName:               displayName,
		Role:                      role,
		AccountType:               AccountTypeMarketplaceOwner,
		EmailNotificationsEnabled: true,
		Status:                    StatusActive,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}, nil
}

// NewOperator creates a user with AccountType=operator, used exclusively
// by the organization invitation acceptance flow. Operators inherit the
// marketplace role (agency or enterprise) of the organization they join,
// so the role passed here must match organization.Type.
func NewOperator(email, hashedPassword, firstName, lastName, displayName string, role Role) (*User, error) {
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}
	if role == RoleProvider {
		// Providers are solo — they cannot become operators of an org.
		return nil, ErrInvalidRole
	}

	now := time.Now()
	return &User{
		ID:                        uuid.New(),
		Email:                     email,
		HashedPassword:            hashedPassword,
		FirstName:                 firstName,
		LastName:                  lastName,
		DisplayName:               displayName,
		Role:                      role,
		AccountType:               AccountTypeOperator,
		EmailNotificationsEnabled: true,
		Status:                    StatusActive,
		CreatedAt:                 now,
		UpdatedAt:                 now,
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
