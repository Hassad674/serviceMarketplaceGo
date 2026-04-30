// Package audit defines the append-only audit log domain.
//
// The audit log is the permanent record of security-sensitive mutations
// performed against the system: authentication events, permission
// changes, role edits, ownership transfers, and other regulated actions.
//
// This package has no persistence responsibilities — see
// internal/port/repository/audit.go for the repository interface and
// internal/adapter/postgres/audit_repository.go for the implementation.
//
// Invariant: audit entries are immutable. The domain API exposes
// constructors but no setters, and the repository interface only has
// Log and List methods — Update and Delete do not exist and MUST NOT
// be added. A mistake in an audit entry is corrected by writing a
// compensating entry, never by editing the history.
package audit

import (
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Action identifies the type of event recorded. Values are snake_case
// and grouped by resource family (auth_*, team_*, org_*, …) so log
// analysis tools can filter by prefix without parsing metadata.
type Action string

// Canonical audit actions. New action keys must be added here so the
// codebase has a single source of truth for valid values and so a typo
// in a caller surfaces as a compile error rather than a silently
// mislabeled audit row.
const (
	// Authentication
	ActionLoginSuccess          Action = "auth.login_success"
	ActionLoginFailure          Action = "auth.login_failure"
	ActionLogout                Action = "auth.logout"
	ActionTokenRefresh          Action = "auth.token_refresh"
	ActionPasswordResetRequest  Action = "auth.password_reset_request"
	ActionPasswordResetComplete Action = "auth.password_reset_complete"

	// ActionTokenReuseDetected is recorded when a refresh token whose
	// JTI is already blacklisted (rotated or revoked) is presented
	// again to /auth/refresh. This is the canonical signal that a
	// refresh token was stolen — the legitimate user's next refresh
	// fails because the attacker rotated the pair first, OR vice
	// versa. SEC-06: the request returns 401 and this row is the
	// breadcrumb the SOC needs to start an investigation.
	ActionTokenReuseDetected Action = "auth.token_reuse_detected"

	// Team / organization
	ActionRolePermissionsChanged Action = "team.role_permissions_changed"
	ActionMemberRoleChanged      Action = "team.member_role_changed"
	ActionMemberRemoved          Action = "team.member_removed"
	ActionOwnershipTransferred   Action = "team.ownership_transferred"

	// Admin actions on user accounts (SEC-13). Emitted by the admin
	// service whenever a platform admin alters a user's account state.
	ActionAdminUserSuspend   Action = "admin.user_suspend"
	ActionAdminUserUnsuspend Action = "admin.user_unsuspend"
	ActionAdminUserBan       Action = "admin.user_ban"
	ActionAdminUserUnban     Action = "admin.user_unban"

	// Admin force-overrides on org ownership (SEC-13). Emitted when an
	// admin uses the "rescue an org with a missing/abusive Owner" flow.
	ActionAdminForceTransfer Action = "admin.force_transfer_ownership"

	// Authorization failures
	ActionAuthorizationDenied Action = "authz.denied"
)

// ResourceType is the kind of resource the audit entry refers to.
// Stored alongside ResourceID so a SELECT by (resource_type, resource_id)
// finds every event that touched a given resource.
type ResourceType string

const (
	ResourceTypeUser         ResourceType = "user"
	ResourceTypeOrganization ResourceType = "organization"
	ResourceTypeMember       ResourceType = "member"
	ResourceTypeInvitation   ResourceType = "invitation"
	ResourceTypeRole         ResourceType = "role"
)

// Entry is a single audit log row. Construct via NewEntry — the struct
// has public fields so the repository can scan directly into it, but
// callers outside this package should go through NewEntry to make sure
// the validation rules are enforced.
type Entry struct {
	ID           uuid.UUID
	UserID       *uuid.UUID // nil for system events (e.g. cron jobs)
	Action       Action
	ResourceType ResourceType
	ResourceID   *uuid.UUID
	Metadata     map[string]any // free-form, always non-nil (empty map if no data)
	IPAddress    *net.IP        // nil when unknown (background jobs, tests)
	CreatedAt    time.Time
}

// NewEntryInput groups the constructor arguments for Entry. Using a
// struct keeps the parameter list under the project's 4-arg limit as
// the audit system grows more call sites.
type NewEntryInput struct {
	UserID       *uuid.UUID
	Action       Action
	ResourceType ResourceType
	ResourceID   *uuid.UUID
	Metadata     map[string]any
	IPAddress    string // raw, validated and parsed into net.IP here
}

// NewEntry validates its input and returns an Entry ready to be
// persisted. Returns an error when the action or resource type is
// empty — every other field is optional.
func NewEntry(in NewEntryInput) (*Entry, error) {
	if strings.TrimSpace(string(in.Action)) == "" {
		return nil, ErrActionRequired
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	var ip *net.IP
	if in.IPAddress != "" {
		parsed := net.ParseIP(in.IPAddress)
		if parsed != nil {
			ip = &parsed
		}
	}

	return &Entry{
		ID:           uuid.New(),
		UserID:       in.UserID,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		Metadata:     metadata,
		IPAddress:    ip,
		CreatedAt:    time.Now().UTC(),
	}, nil
}
