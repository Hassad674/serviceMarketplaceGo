package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// OrgProvisioner is what the auth service needs from the organization
// service. Defined here (not in port/) because it describes a
// same-layer collaboration between two app services, not an external
// port to the outside world.
//
// A nil OrgProvisioner is allowed for tests that don't exercise the
// org provisioning path. Production code always wires a real one.
type OrgProvisioner interface {
	// CreateForOwner provisions a new organization owned by the given user.
	// Every user gets an org (agency/enterprise/provider_personal) since
	// phase R1.
	CreateForOwner(ctx context.Context, u *user.User) (*orgapp.Context, error)

	// ResolveContext returns the user's org membership and computed
	// permissions, or (nil, nil) when the user has no org.
	ResolveContext(ctx context.Context, userID uuid.UUID) (*orgapp.Context, error)
}

// orgContext is a local type alias so call sites read naturally.
// See internal/app/organization/service.go for the definition.
type orgContext = orgapp.Context

type RegisterInput struct {
	Email       string
	Password    string
	FirstName   string
	LastName    string
	DisplayName string
	Role        user.Role
}

type LoginInput struct {
	Email    string
	Password string
}

type ForgotPasswordInput struct {
	Email string
}

type ResetPasswordInput struct {
	Token       string
	NewPassword string
}

type AuthOutput struct {
	User         *user.User
	AccessToken  string
	RefreshToken string

	// Organization context — populated when the user belongs to an
	// organization (marketplace owners of type Agency/Enterprise,
	// and invited operators). Nil / empty for Providers.
	OrganizationID *uuid.UUID
	OrgRole        string
}

type Service struct {
	// users stays on the wide UserRepository — the auth service
	// straddles three segregated children (Reader for ExistsByEmail /
	// GetByID / GetByEmail, Writer for Create / Update, AuthStore for
	// TouchLastActive). Composing locally would cover almost the whole
	// wide port; keeping it wide is clearer.
	users                  repository.UserRepository
	resets                 repository.PasswordResetRepository
	hasher                 service.HasherService
	tokens                 service.TokenService
	email                  service.EmailService
	orgs                   OrgProvisioner         // may be nil in unit tests that don't exercise the org path
	moderationOrchestrator *appmoderation.Service // optional: when nil, display_name moderation is skipped
	// sessionSvc is used by ResetPassword to purge any existing
	// session after a successful reset (SEC-16). May be nil in unit
	// tests that don't exercise the reset path; production wires the
	// Redis adapter.
	sessionSvc       service.SessionService
	refreshBlacklist service.RefreshBlacklistService // SEC-06: when nil, refresh-token rotation defaults to issue-only (legacy behavior)
	// audits is the append-only audit log repository (SEC-13). Used
	// to record every authentication event for forensic purposes.
	// May be nil in unit tests; production wires the Postgres adapter.
	audits      repository.AuditRepository
	frontendURL string
}

// ServiceDeps groups the auth service dependencies to avoid a growing
// positional constructor (already at 6 args before adding the org provisioner).
type ServiceDeps struct {
	Users            repository.UserRepository
	Resets           repository.PasswordResetRepository
	Hasher           service.HasherService
	Tokens           service.TokenService
	Email            service.EmailService
	Orgs             OrgProvisioner
	Sessions         service.SessionService          // SEC-16: optional, purges sessions on password reset
	RefreshBlacklist service.RefreshBlacklistService // SEC-06: when set, refresh tokens rotate single-use through Redis
	Audits           repository.AuditRepository      // SEC-13: when set, auth events + token_reuse_detected are recorded
	FrontendURL      string
}

// NewService returns a fully wired auth service. Prefer NewServiceWithDeps
// in new callsites; this variant is kept for backward compatibility with
// existing tests.
func NewService(
	users repository.UserRepository,
	resets repository.PasswordResetRepository,
	hasher service.HasherService,
	tokens service.TokenService,
	email service.EmailService,
	frontendURL string,
) *Service {
	return &Service{
		users:       users,
		resets:      resets,
		hasher:      hasher,
		tokens:      tokens,
		email:       email,
		frontendURL: frontendURL,
	}
}

// NewServiceWithDeps is the struct-based constructor used by main.go wiring.
// Accepts the organization provisioner alongside the legacy deps.
func NewServiceWithDeps(deps ServiceDeps) *Service {
	return &Service{
		users:            deps.Users,
		resets:           deps.Resets,
		hasher:           deps.Hasher,
		tokens:           deps.Tokens,
		email:            deps.Email,
		orgs:             deps.Orgs,
		sessionSvc:       deps.Sessions,
		refreshBlacklist: deps.RefreshBlacklist,
		audits:           deps.Audits,
		frontendURL:      deps.FrontendURL,
	}
}

// logAudit is a fire-and-forget audit emission helper. Failures are
// logged via slog but never returned to the caller — audit
// completeness is best-effort by policy (see CLAUDE.md "Audit
// logging" + port/repository/audit.go interface comment).
//
// A nil audits repository is fine and skips the call entirely; this
// keeps unit tests that don't wire the audit path simple.
func (s *Service) logAudit(ctx context.Context, in audit.NewEntryInput) {
	if s.audits == nil {
		return
	}
	entry, err := audit.NewEntry(in)
	if err != nil {
		slog.Warn("audit: build entry failed", "action", in.Action, "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("audit: insert failed", "action", in.Action, "error", err)
	}
}

// SetModerationOrchestrator wires the central moderation pipeline.
// Optional: when nil, display_name moderation is skipped (used by
// tests and minimal wiring scenarios). In production this MUST be set
// otherwise toxic display names will pass through registration.
func (s *Service) SetModerationOrchestrator(svc *appmoderation.Service) {
	s.moderationOrchestrator = svc
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil, err
	}

	if _, err := user.NewPassword(input.Password); err != nil {
		return nil, err
	}

	exists, err := s.users.ExistsByEmail(ctx, email.String())
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, user.ErrEmailAlreadyExists
	}

	hashedPassword, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	u, err := user.NewUser(email.String(), hashedPassword, strings.TrimSpace(input.FirstName), strings.TrimSpace(input.LastName), strings.TrimSpace(input.DisplayName), input.Role)
	if err != nil {
		return nil, err
	}

	// Synchronous moderation gate on the public-facing identity. We
	// concatenate display_name + first_name + last_name into a single
	// scan so the engine catches a toxic full name even when the
	// individual fields scrape under the per-field threshold. The
	// content_id is the freshly-minted user.ID — admins can later
	// trace the blocked attempt back to a (failed) registration.
	if err := s.moderateDisplayName(ctx, u); err != nil {
		return nil, err
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Provision an organization for Agency/Enterprise self-registrations.
	// Providers stay solo. If the org provisioner is not wired (e.g. in
	// tests), we skip this step — the user is created but without an org.
	orgCtx, err := s.provisionOrgForNewUser(ctx, u)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(buildAccessInput(u, orgCtx))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return buildAuthOutput(u, orgCtx, accessToken, refreshToken), nil
}

// provisionOrgForNewUser creates an organization for every newly
// registered user. Agencies and enterprises get a company org (type
// mirrors the role), providers get a provider_personal org. Returns
// nil only when no provisioner is wired (tests).
func (s *Service) provisionOrgForNewUser(ctx context.Context, u *user.User) (*orgContext, error) {
	if s.orgs == nil {
		return nil, nil
	}

	orgCtx, err := s.orgs.CreateForOwner(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("provision organization: %w", err)
	}
	return orgCtx, nil
}

// buildAccessInput prepares the TokenService input from a user and an
// optional org context. The session_version is copied from the user's
// current value so the auth middleware has a reference to compare
// future requests against.
//
// The Permissions claim is populated from orgCtx.Permissions — the
// already-resolved effective permission set that honors per-org role
// overrides. The middleware consumes this list as its fast-path,
// which is how customized permissions take effect on every endpoint
// without a DB round-trip on the hot path.
func buildAccessInput(u *user.User, orgCtx *orgContext) service.AccessTokenInput {
	input := service.AccessTokenInput{
		UserID:         u.ID,
		Role:           u.Role.String(),
		IsAdmin:        u.IsAdmin,
		SessionVersion: u.SessionVersion,
	}
	if orgCtx != nil && orgCtx.Organization != nil && orgCtx.Member != nil {
		orgID := orgCtx.Organization.ID
		input.OrganizationID = &orgID
		input.OrgRole = orgCtx.Member.Role.String()
		if len(orgCtx.Permissions) > 0 {
			perms := make([]string, 0, len(orgCtx.Permissions))
			for _, p := range orgCtx.Permissions {
				perms = append(perms, string(p))
			}
			input.Permissions = perms
		}
	}
	return input
}

// buildAuthOutput assembles the auth output with optional org context.
func buildAuthOutput(u *user.User, orgCtx *orgContext, access, refresh string) *AuthOutput {
	out := &AuthOutput{
		User:         u,
		AccessToken:  access,
		RefreshToken: refresh,
	}
	if orgCtx != nil && orgCtx.Organization != nil && orgCtx.Member != nil {
		orgID := orgCtx.Organization.ID
		out.OrganizationID = &orgID
		out.OrgRole = orgCtx.Member.Role.String()
	}
	return out
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		s.logAudit(ctx, audit.NewEntryInput{
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			Metadata: map[string]any{
				"email":  input.Email,
				"reason": "invalid_email_format",
			},
		})
		return nil, user.ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, email.String())
	if err != nil {
		s.logAudit(ctx, audit.NewEntryInput{
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			Metadata: map[string]any{
				"email":  email.String(),
				"reason": "user_not_found",
			},
		})
		return nil, user.ErrInvalidCredentials
	}

	if err := s.hasher.Compare(u.HashedPassword, input.Password); err != nil {
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &u.ID,
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &u.ID,
			Metadata: map[string]any{
				"email":  email.String(),
				"reason": "invalid_password",
			},
		})
		return nil, user.ErrInvalidCredentials
	}

	if u.IsScheduledForDeletion() {
		// P5 (GDPR): refuse login for users whose deleted_at is
		// set. The frontend uses the typed error code to redirect
		// to /account/cancel-deletion if the user wants to keep
		// the account.
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &u.ID,
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &u.ID,
			Metadata: map[string]any{
				"email":  email.String(),
				"reason": "account_scheduled_for_deletion",
			},
		})
		reason := ""
		if u.DeletedAt != nil {
			reason = u.DeletedAt.Format(time.RFC3339)
		}
		return nil, user.NewScheduledForDeletionError(reason)
	}
	if u.IsSuspended() {
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &u.ID,
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &u.ID,
			Metadata: map[string]any{
				"email":  email.String(),
				"reason": "account_suspended",
			},
		})
		return nil, user.NewSuspendedError(u.SuspensionReason)
	}
	if u.IsBanned() {
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &u.ID,
			Action:       audit.ActionLoginFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &u.ID,
			Metadata: map[string]any{
				"email":  email.String(),
				"reason": "account_banned",
			},
		})
		return nil, user.NewBannedError(u.BanReason)
	}

	orgCtx, err := s.resolveOrgContext(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(buildAccessInput(u, orgCtx))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Best-effort: bump last_active_at so the Typesense indexer can
	// rank recently-active profiles higher. A failure here must
	// never block a successful login — log and move on.
	if err := s.users.TouchLastActive(ctx, u.ID); err != nil {
		slog.Warn("auth: touch last_active_at on login failed", "user_id", u.ID, "error", err)
	}

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &u.ID,
		Action:       audit.ActionLoginSuccess,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &u.ID,
		Metadata: map[string]any{
			"email": email.String(),
		},
	})

	return buildAuthOutput(u, orgCtx, accessToken, refreshToken), nil
}
