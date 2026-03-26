package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_ValidRoles(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		hash        string
		firstName   string
		lastName    string
		displayName string
		role        Role
	}{
		{
			name:        "valid agency user",
			email:       "agency@example.com",
			hash:        "hashed_password_123",
			firstName:   "John",
			lastName:    "Doe",
			displayName: "John D.",
			role:        RoleAgency,
		},
		{
			name:        "valid enterprise user",
			email:       "enterprise@example.com",
			hash:        "hashed_password_456",
			firstName:   "Jane",
			lastName:    "Smith",
			displayName: "Jane S.",
			role:        RoleEnterprise,
		},
		{
			name:        "valid provider user",
			email:       "provider@example.com",
			hash:        "hashed_password_789",
			firstName:   "Bob",
			lastName:    "Martin",
			displayName: "Bob M.",
			role:        RoleProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewUser(tt.email, tt.hash, tt.firstName, tt.lastName, tt.displayName, tt.role)

			require.NoError(t, err)
			require.NotNil(t, u)

			assert.NotEmpty(t, u.ID, "ID should be generated")
			assert.Equal(t, tt.email, u.Email)
			assert.Equal(t, tt.hash, u.HashedPassword)
			assert.Equal(t, tt.firstName, u.FirstName)
			assert.Equal(t, tt.lastName, u.LastName)
			assert.Equal(t, tt.displayName, u.DisplayName)
			assert.Equal(t, tt.role, u.Role)
			assert.False(t, u.ReferrerEnabled, "referrer should be disabled by default")
			assert.False(t, u.IsAdmin, "should not be admin by default")
			assert.Nil(t, u.OrganizationID, "organization should be nil by default")
			assert.Nil(t, u.LinkedInID, "linkedIn should be nil by default")
			assert.Nil(t, u.GoogleID, "google should be nil by default")
			assert.False(t, u.EmailVerified, "email should not be verified by default")
			assert.False(t, u.CreatedAt.IsZero(), "created_at should be set")
			assert.False(t, u.UpdatedAt.IsZero(), "updated_at should be set")
		})
	}
}

func TestNewUser_InvalidRole_ReturnsError(t *testing.T) {
	invalidRoles := []Role{
		Role("invalid"),
		Role("admin"),
		Role(""),
		Role("AGENCY"),
		Role("Provider"),
	}

	for _, role := range invalidRoles {
		t.Run("role_"+string(role), func(t *testing.T) {
			u, err := NewUser("test@example.com", "hash", "First", "Last", "Display", role)

			assert.ErrorIs(t, err, ErrInvalidRole)
			assert.Nil(t, u)
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role  Role
		valid bool
	}{
		{RoleAgency, true},
		{RoleEnterprise, true},
		{RoleProvider, true},
		{Role("invalid"), false},
		{Role(""), false},
		{Role("admin"), false},
		{Role("AGENCY"), false},
		{Role("Enterprise"), false},
	}

	for _, tt := range tests {
		t.Run("role_"+string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.role.IsValid())
		})
	}
}

func TestRole_String(t *testing.T) {
	assert.Equal(t, "agency", RoleAgency.String())
	assert.Equal(t, "enterprise", RoleEnterprise.String())
	assert.Equal(t, "provider", RoleProvider.String())
}

func TestUser_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		expected  string
	}{
		{"standard names", "John", "Doe", "John Doe"},
		{"single character", "J", "D", "J D"},
		{"empty first name", "", "Doe", " Doe"},
		{"empty last name", "John", "", "John "},
		{"both empty", "", "", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{FirstName: tt.firstName, LastName: tt.lastName}
			assert.Equal(t, tt.expected, u.FullName())
		})
	}
}

func TestUser_CanBeReferrer(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{"provider can be referrer", RoleProvider, true},
		{"agency cannot be referrer", RoleAgency, false},
		{"enterprise cannot be referrer", RoleEnterprise, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Role: tt.role}
			assert.Equal(t, tt.expected, u.CanBeReferrer())
		})
	}
}

func TestUser_EnableReferrer(t *testing.T) {
	u := &User{Role: RoleProvider, ReferrerEnabled: false}

	u.EnableReferrer()
	assert.True(t, u.ReferrerEnabled)
}

func TestUser_DisableReferrer(t *testing.T) {
	u := &User{Role: RoleProvider, ReferrerEnabled: true}

	u.DisableReferrer()
	assert.False(t, u.ReferrerEnabled)
}

func TestUser_EnableDisableReferrer_Toggle(t *testing.T) {
	u := &User{Role: RoleProvider, ReferrerEnabled: false}

	u.EnableReferrer()
	assert.True(t, u.ReferrerEnabled)

	u.DisableReferrer()
	assert.False(t, u.ReferrerEnabled)

	u.EnableReferrer()
	assert.True(t, u.ReferrerEnabled)
}
