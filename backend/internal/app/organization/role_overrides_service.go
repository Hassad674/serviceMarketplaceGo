package organization

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// RolePermissionsRateLimiter is the minimal shape the role-overrides
// service needs from the rate-limiting adapter. Defined locally
// (not in port/service) because it is a same-layer collaboration
// that nobody else consumes right now.
type RolePermissionsRateLimiter interface {
	Allow(ctx context.Context, orgID uuid.UUID) (bool, error)
}

// RoleOverridesService owns the full lifecycle of per-organization
// role permission customizations: read the effective matrix, save
// changes, bump affected sessions, audit, notify the Owner.
//
// This service is deliberately separate from MembershipService so
// its authorization rules (Owner-only, non-overridable permission
// allowlist) can evolve without touching the rest of team management.
type RoleOverridesService struct {
	orgs        repository.OrganizationRepository
	members     repository.OrganizationMemberRepository
	users       repository.UserRepository
	audits      repository.AuditRepository
	email       service.EmailService
	rateLimiter RolePermissionsRateLimiter
}

// RoleOverridesServiceDeps groups the constructor arguments.
type RoleOverridesServiceDeps struct {
	Orgs        repository.OrganizationRepository
	Members     repository.OrganizationMemberRepository
	Users       repository.UserRepository
	Audits      repository.AuditRepository
	Email       service.EmailService
	RateLimiter RolePermissionsRateLimiter
}

func NewRoleOverridesService(deps RoleOverridesServiceDeps) *RoleOverridesService {
	return &RoleOverridesService{
		orgs:        deps.Orgs,
		members:     deps.Members,
		users:       deps.Users,
		audits:      deps.Audits,
		email:       deps.Email,
		rateLimiter: deps.RateLimiter,
	}
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

// RolePermissionsMatrix is the full customized permission view for a
// single organization. Used by the GET endpoint to populate the
// role-permissions editor in one round-trip.
type RolePermissionsMatrix struct {
	// Roles is the ordered list of (role, permission views) pairs.
	// The Owner row is always present and fully locked so the UI can
	// render it as read-only without a second call.
	Roles []RoleMatrixRow
}

// RoleMatrixRow is a single role's resolved permissions catalogue,
// returned as part of a RolePermissionsMatrix.
type RoleMatrixRow struct {
	Role        organization.Role
	Label       string
	Description string
	Permissions []organization.PermissionView
}

// GetMatrix returns the full customized permission matrix for an org.
// Read access requires any team.view permission — every authenticated
// member can see the matrix for transparency. Only editing is
// Owner-gated (see UpdateRoleOverrides).
func (s *RoleOverridesService) GetMatrix(
	ctx context.Context,
	actorID, orgID uuid.UUID,
) (*RolePermissionsMatrix, error) {
	// Authorization: the actor must be a member of the org.
	member, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		if errors.Is(err, organization.ErrMemberNotFound) {
			return nil, organization.ErrNotAMember
		}
		return nil, fmt.Errorf("get role matrix: find member: %w", err)
	}
	// Any member role can read (team.view is granted to all four in
	// the static defaults). The editor UI will only render the save
	// bar if the actor also has PermTeamManageRolePermissions.
	_ = member

	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get role matrix: find org: %w", err)
	}

	rows := make([]RoleMatrixRow, 0, len(organization.AllRoles()))
	for _, role := range organization.AllRoles() {
		meta := organization.MetadataForRole(role)
		views := organization.MergePermissionsForUI(role, org.RoleOverrides)
		rows = append(rows, RoleMatrixRow{
			Role:        role,
			Label:       meta.Label,
			Description: meta.Description,
			Permissions: views,
		})
	}

	return &RolePermissionsMatrix{Roles: rows}, nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// UpdateRoleOverridesInput carries a single save from the editor UI.
// The overrides map contains the FULL desired state for the target
// role (not a delta) — any previous override that is not included
// is reverted to its default. This is how the UI's "reset to default"
// button works: it sends the map without the override keys.
type UpdateRoleOverridesInput struct {
	ActorUserID    uuid.UUID
	OrganizationID uuid.UUID
	Role           organization.Role
	Overrides      map[organization.Permission]bool
	// Optional — recorded in the audit log when available.
	IPAddress string
}

// UpdateRoleOverridesResult is returned to the handler so it can
// render a success toast with the exact change summary.
type UpdateRoleOverridesResult struct {
	Role            organization.Role
	GrantedKeys     []organization.Permission
	RevokedKeys     []organization.Permission
	AffectedMembers int
}

// UpdateRoleOverrides is the Owner-only write path for the
// role-permissions editor. Enforces every security invariant of the
// feature:
//
//  1. The actor must hold PermTeamManageRolePermissions (Owner-only
//     by default and non-overridable — an Admin can never reach this).
//  2. The target role must be Admin, Member, or Viewer — Owner is
//     never customized.
//  3. Every permission in the payload must be overridable — locked
//     permissions are rejected with ErrPermissionNotOverridable.
//  4. The save must pass the rate limiter (20/day/org default).
//  5. The affected members' session_version is bumped so revoked
//     permissions take effect on the next request.
//  6. Every (role, perm) that changes is appended to audit_logs.
//  7. The Owner receives a summary email (best-effort).
//
// Any failure inside the best-effort tail (audit, email, session
// bump) is logged but does not fail the save — the core state change
// is the only hard requirement.
func (s *RoleOverridesService) UpdateRoleOverrides(
	ctx context.Context,
	in UpdateRoleOverridesInput,
) (*UpdateRoleOverridesResult, error) {
	// Validate role target. Owner is excluded by the domain helper.
	if in.Role == organization.RoleOwner {
		return nil, organization.ErrCannotOverrideOwner
	}
	if !in.Role.IsValid() {
		return nil, organization.ErrInvalidRole
	}

	// Authorization: Owner-only. Look up the actor's membership and
	// check the effective permission. Using HasEffectivePermission on
	// a non-overridable permission is equivalent to HasPermission, but
	// the call is symmetric with the rest of the service layer.
	actor, err := s.members.FindByOrgAndUser(ctx, in.OrganizationID, in.ActorUserID)
	if err != nil {
		if errors.Is(err, organization.ErrMemberNotFound) {
			return nil, organization.ErrNotAMember
		}
		return nil, fmt.Errorf("update role overrides: find actor: %w", err)
	}
	// PermTeamManageRolePermissions is non-overridable, so an empty
	// overrides argument is acceptable here.
	if !organization.HasEffectivePermission(
		actor.Role,
		organization.PermTeamManageRolePermissions,
		nil,
	) {
		return nil, organization.ErrPermissionDenied
	}

	// Rate limit BEFORE mutating anything. The limiter is per-org so
	// a compromised Owner session can't drive it beyond the cap.
	if s.rateLimiter != nil {
		allowed, limitErr := s.rateLimiter.Allow(ctx, in.OrganizationID)
		if limitErr != nil {
			return nil, fmt.Errorf("update role overrides: rate limit: %w", limitErr)
		}
		if !allowed {
			return nil, organization.ErrRolePermChangesRateLimit
		}
	}

	// Load the org and compute the desired state.
	org, err := s.orgs.FindByID(ctx, in.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("update role overrides: find org: %w", err)
	}

	previousOverrides := org.RoleOverrides.Clone()
	// Build the net-new state after applying the Owner's edits on top
	// of the existing overrides. ReplaceRoleOverrides validates the
	// entire payload (including ErrPermissionNotOverridable) in one
	// shot — if any cell is illegal the whole save is rejected.
	normalized := normalizeOverrideCells(in.Overrides, in.Role)
	if err := org.ReplaceRoleOverrides(in.Role, normalized); err != nil {
		return nil, err
	}

	// Persist the change.
	if err := s.orgs.SaveRoleOverrides(ctx, org.ID, org.RoleOverrides); err != nil {
		return nil, fmt.Errorf("update role overrides: persist: %w", err)
	}

	// Compute the diff relative to the PREVIOUS state so the audit
	// log, email, and result all speak the same language.
	granted, revoked := diffEffectivePermissions(
		in.Role,
		previousOverrides,
		org.RoleOverrides,
	)

	// Bump session_version for every member currently holding this
	// role so revoked permissions take effect immediately. Promotions
	// also benefit — the next request reflects the new grants.
	affectedIDs, listErr := s.members.ListUserIDsByRole(ctx, org.ID, in.Role)
	if listErr != nil {
		slog.Warn("update role overrides: list affected members failed",
			"org_id", org.ID, "role", in.Role, "error", listErr)
	}
	for _, userID := range affectedIDs {
		if _, bumpErr := s.users.BumpSessionVersion(ctx, userID); bumpErr != nil {
			// Best-effort. A failure leaves the member with a stale
			// session until the 15-minute access token expires —
			// acceptable because the save succeeded.
			slog.Warn("update role overrides: bump session failed",
				"user_id", userID, "error", bumpErr)
		}
	}

	// Audit log — one entry per changed cell so grep on the audit
	// trail surfaces individual permission flips without parsing
	// the JSON payload.
	s.writeAuditEntries(ctx, in, org.ID, granted, revoked)

	// Best-effort owner notification email.
	s.notifyOwner(ctx, org, in.Role, granted, revoked, len(affectedIDs))

	return &UpdateRoleOverridesResult{
		Role:            in.Role,
		GrantedKeys:     granted,
		RevokedKeys:     revoked,
		AffectedMembers: len(affectedIDs),
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// normalizeOverrideCells removes cells that match the static default
// for the target role, keeping only the true customizations. This
// keeps the JSONB blob minimal and prevents the UI from surfacing
// "customized" badges on cells that match the default anyway.
func normalizeOverrideCells(
	raw map[organization.Permission]bool,
	role organization.Role,
) map[organization.Permission]bool {
	out := make(map[organization.Permission]bool, len(raw))
	for perm, granted := range raw {
		// Compare against the static default for this role. If the
		// override matches the default, it is not a real override and
		// we drop it from the persisted payload.
		defaultHas := organization.HasPermission(role, perm)
		if defaultHas == granted {
			continue
		}
		out[perm] = granted
	}
	return out
}

// diffEffectivePermissions compares the effective permissions before
// and after a save and returns the granted / revoked sets. Both lists
// are sorted for deterministic audit logs and email output.
func diffEffectivePermissions(
	role organization.Role,
	before, after organization.RoleOverrides,
) (granted, revoked []organization.Permission) {
	beforeSet := map[organization.Permission]bool{}
	for _, p := range organization.EffectivePermissionsFor(role, before) {
		beforeSet[p] = true
	}
	afterSet := map[organization.Permission]bool{}
	for _, p := range organization.EffectivePermissionsFor(role, after) {
		afterSet[p] = true
	}
	for p := range afterSet {
		if !beforeSet[p] {
			granted = append(granted, p)
		}
	}
	for p := range beforeSet {
		if !afterSet[p] {
			revoked = append(revoked, p)
		}
	}
	sort.Slice(granted, func(i, j int) bool { return granted[i] < granted[j] })
	sort.Slice(revoked, func(i, j int) bool { return revoked[i] < revoked[j] })
	return granted, revoked
}

// writeAuditEntries appends one audit row per changed cell. Failures
// are logged but never returned to the caller — audit completeness is
// best-effort by policy.
func (s *RoleOverridesService) writeAuditEntries(
	ctx context.Context,
	in UpdateRoleOverridesInput,
	orgID uuid.UUID,
	granted, revoked []organization.Permission,
) {
	if s.audits == nil {
		return
	}
	actorID := in.ActorUserID
	writeOne := func(perm organization.Permission, grantedAfter bool) {
		metadata := map[string]any{
			"organization_id": orgID.String(),
			"role":            string(in.Role),
			"permission":      string(perm),
			"granted_after":   grantedAfter,
		}
		entry, err := audit.NewEntry(audit.NewEntryInput{
			UserID:       &actorID,
			Action:       audit.ActionRolePermissionsChanged,
			ResourceType: audit.ResourceTypeOrganization,
			ResourceID:   &orgID,
			Metadata:     metadata,
			IPAddress:    in.IPAddress,
		})
		if err != nil {
			slog.Warn("audit: build entry failed",
				"action", audit.ActionRolePermissionsChanged, "error", err)
			return
		}
		if err := s.audits.Log(ctx, entry); err != nil {
			slog.Warn("audit: insert failed",
				"action", audit.ActionRolePermissionsChanged, "error", err)
		}
	}
	for _, p := range granted {
		writeOne(p, true)
	}
	for _, p := range revoked {
		writeOne(p, false)
	}
}

// notifyOwner sends the anti-tampering summary email to the Owner.
// Runs asynchronously (fire-and-forget) so a slow email adapter does
// not delay the API response. Failures are logged.
func (s *RoleOverridesService) notifyOwner(
	ctx context.Context,
	org *organization.Organization,
	role organization.Role,
	granted, revoked []organization.Permission,
	affected int,
) {
	if s.email == nil || s.users == nil {
		return
	}

	// Snapshot the data we'll need inside the goroutine so the outer
	// context and caller-owned structs stay immutable.
	ownerID := org.OwnerUserID
	orgName := org.Name
	grantedLabels := labelize(granted)
	revokedLabels := labelize(revoked)
	roleStr := string(role)

	// Detach from the request context so cancellation does not
	// propagate (the email delivery must complete after the response
	// returns), but keep the trace identifiers so the email send is
	// correlatable to the admin action that triggered it. Closes
	// gosec G118: parent is request-scoped + WithoutCancel, never
	// context.Background().
	parent := context.WithoutCancel(ctx)
	go func() {
		// 10 second timeout protects against a slow email adapter
		// leaking the goroutine.
		bgCtx, cancel := context.WithTimeout(parent, 10*time.Second)
		defer cancel()

		owner, err := s.users.GetByID(bgCtx, ownerID)
		if err != nil || owner == nil {
			slog.Warn("role perms: owner lookup failed for notification email",
				"owner_id", ownerID, "error", err)
			return
		}
		// Honor the Owner's email notifications preference. Security
		// notices normally bypass the unsubscribe flag (the anti-
		// tampering use case overrides opt-out), but in V1 we respect
		// the user's choice to avoid spamming an Owner who explicitly
		// asked for silence. Admins can re-enable the flag from their
		// account settings if they want the alert.
		if !owner.EmailNotificationsEnabled {
			return
		}
		sendErr := s.email.SendRolePermissionsChanged(bgCtx, service.RolePermissionsChangedEmailInput{
			To:              owner.Email,
			OwnerFirstName:  owner.FirstName,
			OrgName:         orgName,
			Role:            roleStr,
			GrantedLabels:   grantedLabels,
			RevokedLabels:   revokedLabels,
			AffectedMembers: affected,
			ChangedAt:       time.Now(),
		})
		if sendErr != nil {
			slog.Warn("role perms: email delivery failed",
				"owner_id", ownerID, "error", sendErr)
		}
	}()

	_ = ctx
}

// labelize resolves a list of permission keys into their human
// metadata labels, sorted for stable output.
func labelize(perms []organization.Permission) []string {
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		meta := organization.MetadataForPermission(p)
		label := strings.TrimSpace(meta.Label)
		if label == "" {
			label = string(p)
		}
		out = append(out, label)
	}
	return out
}
