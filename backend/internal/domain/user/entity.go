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

type User struct {
	ID              uuid.UUID
	Email           string
	HashedPassword  string
	FirstName       string
	LastName        string
	DisplayName     string
	Role            Role
	ReferrerEnabled bool
	IsAdmin         bool
	OrganizationID  *uuid.UUID
	LinkedInID      *string
	GoogleID        *string
	EmailVerified   bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
