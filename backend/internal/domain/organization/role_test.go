package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{"owner", RoleOwner, true},
		{"admin", RoleAdmin, true},
		{"member", RoleMember, true},
		{"viewer", RoleViewer, true},
		{"unknown", Role("superadmin"), false},
		{"empty", Role(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.role.IsValid())
		})
	}
}

func TestRole_String(t *testing.T) {
	assert.Equal(t, "owner", RoleOwner.String())
	assert.Equal(t, "admin", RoleAdmin.String())
	assert.Equal(t, "member", RoleMember.String())
	assert.Equal(t, "viewer", RoleViewer.String())
}

func TestRole_CanBeInvitedAs(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, false},
		{RoleAdmin, true},
		{RoleMember, true},
		{RoleViewer, true},
		{Role("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.role.CanBeInvitedAs())
		})
	}
}

func TestRole_IsElevated(t *testing.T) {
	assert.True(t, RoleOwner.IsElevated())
	assert.True(t, RoleAdmin.IsElevated())
	assert.False(t, RoleMember.IsElevated())
	assert.False(t, RoleViewer.IsElevated())
}

func TestRole_Level(t *testing.T) {
	tests := []struct {
		role Role
		want int
	}{
		{RoleOwner, 4},
		{RoleAdmin, 3},
		{RoleMember, 2},
		{RoleViewer, 1},
		{Role("unknown"), 0},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.role.Level())
		})
	}
}

func TestIsDemotion(t *testing.T) {
	tests := []struct {
		name string
		from Role
		to   Role
		want bool
	}{
		{"admin to member is demotion", RoleAdmin, RoleMember, true},
		{"admin to viewer is demotion", RoleAdmin, RoleViewer, true},
		{"member to viewer is demotion", RoleMember, RoleViewer, true},
		{"owner to admin is demotion", RoleOwner, RoleAdmin, true},
		{"viewer to member is promotion", RoleViewer, RoleMember, false},
		{"member to admin is promotion", RoleMember, RoleAdmin, false},
		{"viewer to admin is promotion", RoleViewer, RoleAdmin, false},
		{"admin to admin is lateral", RoleAdmin, RoleAdmin, false},
		{"member to member is lateral", RoleMember, RoleMember, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDemotion(tt.from, tt.to))
		})
	}
}
