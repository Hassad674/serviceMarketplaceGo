// Package gdpr implements the right-to-erasure + right-to-export
// orchestration for the marketplace API (P5 plan).
//
// The service has four user-facing entry points:
//
//   - ExportData: stream a ZIP of every personal datum the platform
//     holds for a user. Decision 1 of the P5 brief (manifest.json,
//     README.txt, one JSON file per domain).
//
//   - RequestDeletion: verify the user's password, refuse if the
//     user is the Owner of an org with active members (Decision 6),
//     issue a JWT with purpose=account_deletion (Decision 5), email
//     the confirmation link.
//
//   - ConfirmDeletion: validate the JWT, set users.deleted_at, lock
//     out the account (Decision 3).
//
//   - CancelDeletion: clear deleted_at while the 30-day window is
//     still open.
//
// The implementation is pure Go — every external concern (DB, JWT,
// email) is injected via a port interface so the service is fully
// unit-testable with sqlmock + stub services.
package gdpr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	domaingdpr "marketplace-backend/internal/domain/gdpr"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// JWTSigner is the minimal contract the service needs to issue + verify
// the deletion confirmation JWT. The auth feature wraps a different
// claim set so we keep this isolated rather than reuse the existing
// TokenService — different flows MUST NOT share signing keys with the
// access-token JWT.
type JWTSigner interface {
	Sign(claims jwt.Claims) (string, error)
	Parse(token string, claims jwt.Claims) error
}

// Service orchestrates the four GDPR endpoints.
type Service struct {
	repo repository.GDPRRepository
	// users is narrowed to UserReader — GDPR endpoints only resolve
	// the requesting user by id (GetByID); deletion / purge writes go
	// through the dedicated GDPRRepository contract.
	users         repository.UserReader
	hasher        service.HasherService
	email         service.EmailService
	signer        JWTSigner
	frontendURL   string
	clock         func() time.Time
	preferredLang func(*user.User) string
}

// ServiceDeps groups the constructor dependencies. Locale defaults to
// "fr" when PreferredLang is nil — French is the default audience of
// the marketplace (per project memory).
type ServiceDeps struct {
	Repo          repository.GDPRRepository
	Users         repository.UserReader
	Hasher        service.HasherService
	Email         service.EmailService
	Signer        JWTSigner
	FrontendURL   string
	Clock         func() time.Time
	PreferredLang func(*user.User) string
}

// NewService wires the GDPR service. Returns ErrSaltRequired-shaped
// configuration errors at boot via the wire helper, never here.
func NewService(deps ServiceDeps) *Service {
	clock := deps.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	pref := deps.PreferredLang
	if pref == nil {
		pref = func(*user.User) string { return "fr" }
	}
	return &Service{
		repo:          deps.Repo,
		users:         deps.Users,
		hasher:        deps.Hasher,
		email:         deps.Email,
		signer:        deps.Signer,
		frontendURL:   deps.FrontendURL,
		clock:         clock,
		preferredLang: pref,
	}
}

// ExportData builds the export aggregate for the user. The handler
// streams it to a ZIP without further processing.
func (s *Service) ExportData(ctx context.Context, userID uuid.UUID) (*domaingdpr.Export, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.IsScheduledForDeletion() {
		return nil, user.ErrAccountScheduledForDeletion
	}
	exp, err := s.repo.LoadExport(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := exp.Validate(); err != nil {
		return nil, err
	}
	exp.Locale = s.preferredLang(u)
	return exp, nil
}

// RequestDeletionInput groups the request body of POST
// /me/account/request-deletion. Password is the user's current
// password — verified against the hashed value before any side
// effect.
type RequestDeletionInput struct {
	UserID   uuid.UUID
	Password string
}

// RequestDeletionResult tells the handler what side effects already
// landed. EmailSentTo is the address the confirmation email went to,
// which the handler echoes back to the frontend so the UX can show
// "we sent an email to xx@yy.com — check your inbox".
type RequestDeletionResult struct {
	EmailSentTo string
	ExpiresAt   time.Time
}

// RequestDeletion is the entry point for the password-protected
// "I want to delete my account" form. On success it sends the
// confirmation email but does NOT set deleted_at — that only
// happens once the user clicks the link in their inbox.
func (s *Service) RequestDeletion(ctx context.Context, in RequestDeletionInput) (*RequestDeletionResult, error) {
	u, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.IsScheduledForDeletion() {
		// Idempotent — re-send the email, do not refuse.
		return s.dispatchConfirmationEmail(ctx, u)
	}
	if err := s.hasher.Compare(u.HashedPassword, in.Password); err != nil {
		return nil, user.ErrInvalidCredentials
	}

	blocked, err := s.repo.FindOwnedOrgsBlockingDeletion(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("check blocking orgs: %w", err)
	}
	if len(blocked) > 0 {
		return nil, domaingdpr.NewOwnerBlockedError(blocked)
	}

	return s.dispatchConfirmationEmail(ctx, u)
}

// dispatchConfirmationEmail issues the JWT, builds the link, and
// hands the rendered email to the email service. Extracted so
// RequestDeletion can re-send for idempotency without re-running
// the password check.
func (s *Service) dispatchConfirmationEmail(ctx context.Context, u *user.User) (*RequestDeletionResult, error) {
	now := s.clock()
	expires := now.Add(domaingdpr.ConfirmationTokenTTL)

	claims := jwt.MapClaims{
		"sub":     u.ID.String(),
		"purpose": domaingdpr.ConfirmationTokenPurpose,
		"iat":     now.Unix(),
		"exp":     expires.Unix(),
		"jti":     uuid.New().String(),
	}
	token, err := s.signer.Sign(claims)
	if err != nil {
		return nil, fmt.Errorf("sign confirmation token: %w", err)
	}

	confirmURL := s.frontendURL + "/account/confirm-deletion?token=" + token
	subject, html := renderConfirmationEmail(s.preferredLang(u), confirmEmailParams{
		FirstName:  u.FirstName,
		ConfirmURL: confirmURL,
		ExpiresAt:  expires,
	})

	if err := s.email.SendNotification(ctx, u.Email, subject, html); err != nil {
		return nil, fmt.Errorf("send confirmation email: %w", err)
	}
	return &RequestDeletionResult{
		EmailSentTo: u.Email,
		ExpiresAt:   expires,
	}, nil
}

// ConfirmDeletion validates the JWT and sets users.deleted_at. The
// underlying SoftDelete is COALESCE-based so a duplicate click is a
// no-op and returns 200.
type ConfirmDeletionResult struct {
	UserID      uuid.UUID
	DeletedAt   time.Time
	HardDeleteAt time.Time
}

func (s *Service) ConfirmDeletion(ctx context.Context, token string) (*ConfirmDeletionResult, error) {
	claims := jwt.MapClaims{}
	if err := s.signer.Parse(token, claims); err != nil {
		return nil, fmt.Errorf("parse confirmation token: %w", user.ErrUnauthorized)
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != domaingdpr.ConfirmationTokenPurpose {
		return nil, user.ErrUnauthorized
	}
	subStr, _ := claims["sub"].(string)
	userID, err := uuid.Parse(subStr)
	if err != nil {
		return nil, user.ErrUnauthorized
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	deletedAt, err := s.repo.SoftDelete(ctx, u.ID, s.clock())
	if err != nil {
		return nil, err
	}
	return &ConfirmDeletionResult{
		UserID:       u.ID,
		DeletedAt:    deletedAt,
		HardDeleteAt: domaingdpr.ScheduledHardDeleteAt(deletedAt),
	}, nil
}

// CancelDeletion clears deleted_at if the user is currently in the
// 30-day cooldown. Idempotent: a cancel for a non-deleted user
// returns NoOp=true so the handler can render a friendly "nothing
// to cancel" page.
type CancelDeletionResult struct {
	NoOp bool
}

func (s *Service) CancelDeletion(ctx context.Context, userID uuid.UUID) (*CancelDeletionResult, error) {
	cancelled, err := s.repo.CancelDeletion(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &CancelDeletionResult{NoOp: !cancelled}, nil
}

// PurgeOnce is the entry point for the cron scheduler. It walks one
// batch of purgeable users and runs PurgeUser for each. Returns the
// number of rows actually purged + any non-fatal errors. Errors on
// individual rows are logged by the scheduler but do not abort the
// batch — a poisoned row should never block subsequent purges.
type PurgeBatchResult struct {
	Examined int
	Purged   int
	Errors   []error
}

// PurgeOnce runs one batch of the scheduled hard-delete. Used by the
// scheduler in scheduler.go and by integration tests that exercise
// the full flow.
func (s *Service) PurgeOnce(ctx context.Context, salt string, batchSize int) (*PurgeBatchResult, error) {
	if salt == "" {
		return nil, domaingdpr.ErrSaltRequired
	}
	now := s.clock()
	cutoff := now.Add(-domaingdpr.PurgeWindow)

	ids, err := s.repo.ListPurgeable(ctx, cutoff, batchSize)
	if err != nil {
		return nil, fmt.Errorf("list purgeable: %w", err)
	}

	out := &PurgeBatchResult{Examined: len(ids)}
	for _, id := range ids {
		ok, err := s.repo.PurgeUser(ctx, id, cutoff, salt)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Errorf("purge %s: %w", id, err))
			continue
		}
		if ok {
			out.Purged++
		}
	}
	return out, nil
}

// errIs is a typed local helper that re-exports errors.Is so callers
// can compose without pulling errors directly. Kept here for symmetry
// with the auth service.
func errIs(err, target error) bool {
	return errors.Is(err, target)
}

var _ = errIs
