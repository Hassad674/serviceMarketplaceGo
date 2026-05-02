package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// InvitationRateLimiter is the minimal shape the invitation service
// needs from the rate-limiting adapter. Defined locally (not in
// port/service) because it is a same-layer collaboration that nobody
// else in the backend consumes right now.
type InvitationRateLimiter interface {
	Allow(ctx context.Context, orgID uuid.UUID) (bool, error)
}

// InvitationService orchestrates the lifecycle of team invitations:
// send, validate, accept, resend, cancel, list.
//
// It reuses the organization & member repositories from the sibling
// Service but adds the user repository, the hasher (for password
// hashing at acceptance time), the email adapter, the rate limiter,
// and the notification sender for the org_invitation_accepted event.
type InvitationService struct {
	// orgs is narrowed to OrganizationReader — invitations only read
	// the org row to confirm existence + render the recipient email.
	orgs          repository.OrganizationReader
	members       repository.OrganizationMemberRepository
	invitations   repository.OrganizationInvitationRepository
	users         repository.UserRepository
	hasher        service.HasherService
	email         service.EmailService
	rateLimiter   InvitationRateLimiter
	notifications service.NotificationSender // nil when feature disabled
	frontendURL   string
}

// InvitationServiceDeps groups constructor arguments. Eight fields
// exceeds the project's 4-param rule, so we use a struct.
type InvitationServiceDeps struct {
	Orgs          repository.OrganizationReader
	Members       repository.OrganizationMemberRepository
	Invitations   repository.OrganizationInvitationRepository
	Users         repository.UserRepository
	Hasher        service.HasherService
	Email         service.EmailService
	RateLimiter   InvitationRateLimiter
	Notifications service.NotificationSender // optional — nil disables team notifications
	FrontendURL   string
}

func NewInvitationService(deps InvitationServiceDeps) *InvitationService {
	return &InvitationService{
		orgs:          deps.Orgs,
		members:       deps.Members,
		invitations:   deps.Invitations,
		users:         deps.Users,
		hasher:        deps.Hasher,
		email:         deps.Email,
		rateLimiter:   deps.RateLimiter,
		notifications: deps.Notifications,
		frontendURL:   deps.FrontendURL,
	}
}

// ---------------------------------------------------------------------------
// Send
// ---------------------------------------------------------------------------

// SendInvitationInput carries the data needed to create and send an
// invitation. Validated in NewInvitation (domain layer).
type SendInvitationInput struct {
	InviterUserID  uuid.UUID
	OrganizationID uuid.UUID
	Email          string
	FirstName      string
	LastName       string
	Title          string
	Role           organization.Role
}

// SendInvitation creates an invitation and sends the email.
//
// Permission: the caller must hold team.invite (Owner or Admin).
// Rate limit: at most 10 invitations per org per hour.
// Collision checks: the target email cannot already be a member of the
// org, cannot be an operator of ANOTHER org, and cannot belong to a
// Provider in V1 (use a different email for dual accounts).
//
// Returns the persisted invitation on success. The caller is expected
// to return it to the Owner/Admin so they can see the pending record in
// the team list immediately.
func (s *InvitationService) SendInvitation(ctx context.Context, in SendInvitationInput) (*organization.Invitation, error) {
	if err := s.requirePermission(ctx, in.InviterUserID, in.OrganizationID, organization.PermTeamInvite); err != nil {
		return nil, err
	}

	if !in.Role.CanBeInvitedAs() {
		return nil, organization.ErrCannotInviteAsOwner
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(in.Email))
	if err := s.checkEmailCollision(ctx, in.OrganizationID, normalizedEmail); err != nil {
		return nil, err
	}

	if s.rateLimiter != nil {
		allowed, err := s.rateLimiter.Allow(ctx, in.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("send invitation: rate limit check: %w", err)
		}
		if !allowed {
			return nil, ErrInvitationRateLimited
		}
	}

	org, err := s.orgs.FindByID(ctx, in.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("send invitation: find org: %w", err)
	}

	inv, err := organization.NewInvitation(organization.NewInvitationInput{
		OrganizationID:  in.OrganizationID,
		Email:           normalizedEmail,
		FirstName:       in.FirstName,
		LastName:        in.LastName,
		Title:           in.Title,
		Role:            in.Role,
		InvitedByUserID: in.InviterUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("send invitation: build entity: %w", err)
	}

	if err := s.invitations.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("send invitation: persist: %w", err)
	}

	// Fire-and-log the email. A delivery failure must not rollback the
	// persisted invitation — the Owner can always resend.
	if err := s.sendInvitationEmail(ctx, org, inv); err != nil {
		// Swallow the error but return the invitation so the caller
		// can display it with a warning. We don't propagate the error
		// because the invitation IS persisted and usable — a resend
		// will retry the email.
		return inv, fmt.Errorf("send invitation: email delivery failed: %w", err)
	}
	return inv, nil
}

// ---------------------------------------------------------------------------
// Validate (public, for the acceptance page)
// ---------------------------------------------------------------------------

// ValidatedInvitation is the public view of a pending invitation returned
// to the invitation acceptance page so it can pre-fill the form.
type ValidatedInvitation struct {
	Invitation   *organization.Invitation
	Organization *organization.Organization
}

// ValidateToken returns the invitation associated with the given token
// if (and only if) it is in pending state and not expired. Used by the
// public /api/v1/invitations/validate?token=X endpoint which does not
// require auth.
func (s *InvitationService) ValidateToken(ctx context.Context, token string) (*ValidatedInvitation, error) {
	if token == "" {
		return nil, organization.ErrInvitationNotFound
	}
	inv, err := s.invitations.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if inv.IsExpired() {
		// Flip state to expired on read, so subsequent lookups stay
		// consistent without waiting for the background sweeper.
		_ = inv.MarkExpired()
		_ = s.invitations.Update(ctx, inv)
		return nil, organization.ErrInvitationExpired
	}
	if inv.Status != organization.InvitationStatusPending {
		return nil, organization.ErrInvitationAlreadyUsed
	}
	org, err := s.orgs.FindByID(ctx, inv.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("validate token: find org: %w", err)
	}
	return &ValidatedInvitation{Invitation: inv, Organization: org}, nil
}

// ---------------------------------------------------------------------------
// Accept (public, creates the operator account)
// ---------------------------------------------------------------------------

// AcceptInvitationInput bundles the fields the public endpoint needs
// to turn a pending invitation into a live operator account.
type AcceptInvitationInput struct {
	Token    string
	Password string // plain text, will be hashed
}

// AcceptInvitationResult is what the handler returns to the caller so
// the frontend can log the new operator in immediately after accepting.
type AcceptInvitationResult struct {
	User         *user.User
	OrgContext   *Context
	Organization *organization.Organization
	Member       *organization.Member
}

// AcceptInvitation creates a new operator user, makes them a member of
// the inviting organization with the role stored on the invitation, and
// marks the invitation as accepted — all in a single DB transaction
// via the invitation repository's AcceptInvitationTx method.
//
// The resulting user has account_type=operator and inherits the
// marketplace role of the organization (agency or enterprise).
//
// The caller is expected to immediately issue login tokens for the new
// user (see auth service), so the operator lands on the dashboard in
// one click after password entry.
func (s *InvitationService) AcceptInvitation(ctx context.Context, in AcceptInvitationInput) (*AcceptInvitationResult, error) {
	validated, err := s.ValidateToken(ctx, in.Token)
	if err != nil {
		return nil, err
	}
	inv := validated.Invitation
	org := validated.Organization

	if _, err := user.NewPassword(in.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("accept invitation: hash password: %w", err)
	}

	// Final collision check: make sure no one else has registered with
	// this email between ValidateToken and now (race window is small
	// but not zero).
	if exists, _ := s.users.ExistsByEmail(ctx, inv.Email); exists {
		return nil, organization.ErrAlreadyMember
	}

	marketplaceRole := marketplaceRoleForOrgType(org.Type)
	displayName := strings.TrimSpace(inv.FirstName + " " + inv.LastName)

	newOperator, err := user.NewOperator(
		inv.Email,
		hashedPassword,
		inv.FirstName,
		inv.LastName,
		displayName,
		marketplaceRole,
	)
	if err != nil {
		return nil, fmt.Errorf("accept invitation: build operator user: %w", err)
	}
	// Denormalize the invited org onto the new operator's users row, so
	// single-row lookups (JWT refresh, /me, resource backfills) see it
	// without joining organization_members.
	orgID := inv.OrganizationID
	newOperator.OrganizationID = &orgID

	newMember, err := organization.NewMember(inv.OrganizationID, newOperator.ID, inv.Role, inv.Title)
	if err != nil {
		return nil, fmt.Errorf("accept invitation: build member: %w", err)
	}

	if err := inv.Accept(); err != nil {
		return nil, fmt.Errorf("accept invitation: mark accepted: %w", err)
	}

	if err := s.invitations.AcceptInvitationTx(ctx, inv, newOperator, newMember); err != nil {
		return nil, fmt.Errorf("accept invitation: persist: %w", err)
	}

	// Notify the original inviter (Owner or Admin who sent it).
	// Best-effort: failures are swallowed inside notifyInvitationAccepted.
	notifyInvitationAccepted(ctx, s.notifications, inv.InvitedByUserID, newOperator, org, inv.ID)

	return &AcceptInvitationResult{
		User:         newOperator,
		Organization: org,
		Member:       newMember,
		OrgContext: &Context{
			Organization: org,
			Member:       newMember,
			// Honor the org's custom permission overrides right from
			// the first session so the new operator sees the same UI
			// state as existing members on their very first request.
			Permissions: organization.EffectivePermissionsFor(newMember.Role, org.RoleOverrides),
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Resend / Cancel / List
// ---------------------------------------------------------------------------

// ResendInvitation generates a fresh token and resends the email. The
// original invitation row stays (same id) — only the token, timestamps,
// and email delivery are refreshed. Rate-limited like SendInvitation.
func (s *InvitationService) ResendInvitation(ctx context.Context, actorID, orgID, invitationID uuid.UUID) (*organization.Invitation, error) {
	if err := s.requirePermission(ctx, actorID, orgID, organization.PermTeamInvite); err != nil {
		return nil, err
	}

	inv, err := s.invitations.FindByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if inv.OrganizationID != orgID {
		return nil, organization.ErrInvitationNotFound
	}

	if s.rateLimiter != nil {
		allowed, err := s.rateLimiter.Allow(ctx, orgID)
		if err != nil {
			return nil, fmt.Errorf("resend invitation: rate limit check: %w", err)
		}
		if !allowed {
			return nil, ErrInvitationRateLimited
		}
	}

	// Rebuild a fresh invitation entity to regenerate the token and
	// reset the timestamps. We preserve the persisted id so the row
	// doesn't move in the UI.
	freshInput := organization.NewInvitationInput{
		OrganizationID:  inv.OrganizationID,
		Email:           inv.Email,
		FirstName:       inv.FirstName,
		LastName:        inv.LastName,
		Title:           inv.Title,
		Role:            inv.Role,
		InvitedByUserID: actorID,
	}
	refreshed, err := organization.NewInvitation(freshInput)
	if err != nil {
		return nil, fmt.Errorf("resend invitation: rebuild: %w", err)
	}
	inv.Token = refreshed.Token
	inv.ExpiresAt = refreshed.ExpiresAt
	inv.Status = organization.InvitationStatusPending
	inv.AcceptedAt = nil
	inv.CancelledAt = nil
	inv.InvitedByUserID = actorID
	inv.UpdatedAt = refreshed.UpdatedAt

	if err := s.invitations.Update(ctx, inv); err != nil {
		return nil, fmt.Errorf("resend invitation: persist: %w", err)
	}

	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("resend invitation: find org: %w", err)
	}
	if err := s.sendInvitationEmail(ctx, org, inv); err != nil {
		return inv, fmt.Errorf("resend invitation: email delivery failed: %w", err)
	}
	return inv, nil
}

// CancelInvitation marks a pending invitation as cancelled. The row
// stays in the DB for audit purposes. Idempotent-ish: returns an error
// when the invitation is not pending (so the UI can surface "already
// used" or "already cancelled" distinctly).
func (s *InvitationService) CancelInvitation(ctx context.Context, actorID, orgID, invitationID uuid.UUID) error {
	if err := s.requirePermission(ctx, actorID, orgID, organization.PermTeamInvite); err != nil {
		return err
	}
	inv, err := s.invitations.FindByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv.OrganizationID != orgID {
		return organization.ErrInvitationNotFound
	}
	if err := inv.Cancel(); err != nil {
		return err
	}
	return s.invitations.Update(ctx, inv)
}

// ListPending returns the organization's pending invitations, most
// recent first. Requires team.view.
func (s *InvitationService) ListPending(ctx context.Context, actorID, orgID uuid.UUID, cursor string, limit int) ([]*organization.Invitation, string, error) {
	if err := s.requirePermission(ctx, actorID, orgID, organization.PermTeamView); err != nil {
		return nil, "", err
	}
	return s.invitations.List(ctx, repository.ListInvitationsParams{
		OrganizationID: orgID,
		StatusFilter:   organization.InvitationStatusPending,
		Cursor:         cursor,
		Limit:          limit,
	})
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// requirePermission ensures the actor is a member of the org with the
// given permission. Returns ErrNotAMember or ErrPermissionDenied.
//
// Resolves the org's per-role overrides before evaluating — an Owner
// who has granted "invite" to Members must see Members pass this check
// even though the static defaults deny it.
func (s *InvitationService) requirePermission(ctx context.Context, actorID, orgID uuid.UUID, perm organization.Permission) error {
	member, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		if errors.Is(err, organization.ErrMemberNotFound) {
			return organization.ErrNotAMember
		}
		return fmt.Errorf("permission check: %w", err)
	}
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("permission check: load org: %w", err)
	}
	if !organization.HasEffectivePermission(member.Role, perm, org.RoleOverrides) {
		return organization.ErrPermissionDenied
	}
	return nil
}

// checkEmailCollision runs the three pre-send checks on an email:
//   - already a member of THIS org           → ErrAlreadyMember
//   - already has a pending invitation here  → ErrAlreadyInvited
//   - already has a marketplace account      → ErrAlreadyMember (V1: use different email)
func (s *InvitationService) checkEmailCollision(ctx context.Context, orgID uuid.UUID, email string) error {
	// 1. Existing user check (any account with this email)
	existing, err := s.users.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		// The email belongs to an existing account. V1 forbids
		// reusing it as an operator (enforces "one email = one
		// account" invariant). The user must use a different email.
		return organization.ErrAlreadyMember
	}
	// user.ErrUserNotFound is expected and fine — we want "no existing user"

	// 2. Pending invitation for the same email in THIS org
	if _, err := s.invitations.FindPendingByOrgAndEmail(ctx, orgID, email); err == nil {
		return organization.ErrAlreadyInvited
	}
	return nil
}

// sendInvitationEmail renders and dispatches the invitation email. The
// org's owner name is used as the "inviter name" display — the actual
// inviter might be an Admin, but for V1 we use the org's public face.
func (s *InvitationService) sendInvitationEmail(ctx context.Context, org *organization.Organization, inv *organization.Invitation) error {
	inviter, err := s.users.GetByID(ctx, inv.InvitedByUserID)
	inviterName := "l'équipe"
	orgName := "l'organisation"
	if err == nil && inviter != nil {
		inviterName = inviter.DisplayName
		if inviterName == "" {
			inviterName = strings.TrimSpace(inviter.FirstName + " " + inviter.LastName)
		}
	}
	owner, ownerErr := s.users.GetByID(ctx, org.OwnerUserID)
	if ownerErr == nil && owner != nil && owner.DisplayName != "" {
		orgName = owner.DisplayName
	}

	acceptURL := strings.TrimRight(s.frontendURL, "/") + "/invitation/" + inv.Token

	return s.email.SendTeamInvitation(ctx, service.TeamInvitationEmailInput{
		To:               inv.Email,
		OrgName:          orgName,
		OrgType:          org.Type.String(),
		InviterName:      inviterName,
		InviteeFirstName: inv.FirstName,
		Role:             inv.Role.String(),
		AcceptURL:        acceptURL,
		ExpiresAt:        inv.ExpiresAt,
	})
}

// marketplaceRoleForOrgType maps an organization type to the marketplace
// role an invited operator should inherit. The operator takes their
// org's role so existing queries keyed by role continue to work.
func marketplaceRoleForOrgType(t organization.OrgType) user.Role {
	switch t {
	case organization.OrgTypeEnterprise:
		return user.RoleEnterprise
	default:
		return user.RoleAgency
	}
}

// ErrInvitationRateLimited is returned when the org has hit the hourly
// invitation cap. Mapped to HTTP 429 at the handler layer.
var ErrInvitationRateLimited = errors.New("invitation rate limit reached: try again later")
