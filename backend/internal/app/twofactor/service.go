// Package twofactor implements the application service for the email
// 2FA flow. It is the orchestration layer between the auth flow (which
// asks "is 2FA required for this login?"), the postgres adapter (which
// stores and reads the challenge rows), the bcrypt hasher (which keeps
// the plaintext code out of the DB), and the email service (which
// delivers the code to the user's mailbox).
//
// The package is intentionally narrow: two public methods,
// RequestChallenge and VerifyChallenge. Enable/disable of the flag
// itself lives on the auth handler because it requires re-auth and
// session-version bumping that belong with the credential rotation
// flows.
package twofactor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Service orchestrates the email 2FA challenge lifecycle. It depends
// on narrow port interfaces — never on adapters — so the postgres,
// bcrypt, and email implementations can be swapped without touching
// this package.
type Service struct {
	challenges repository.TwoFactorChallengeRepository
	hasher     service.HasherService
	email      service.EmailService
	audits     repository.AuditRepository // optional: when nil, audit rows are skipped
}

// ServiceDeps groups the constructor arguments to stay under the
// project's 4-arg ceiling. Audits is optional — unit tests that don't
// exercise the audit path leave it nil.
type ServiceDeps struct {
	Challenges repository.TwoFactorChallengeRepository
	Hasher     service.HasherService
	Email      service.EmailService
	Audits     repository.AuditRepository
}

// NewService wires the dependencies. Returns a fully usable service
// when at minimum Challenges, Hasher, and Email are non-nil.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		challenges: deps.Challenges,
		hasher:     deps.Hasher,
		email:      deps.Email,
		audits:     deps.Audits,
	}
}

// RequestChallengeInput groups the request-side metadata. ClientIP and
// UserAgentHash are best-effort forensic fields — passing empty
// strings is fine and the row simply records NULL on those columns.
// EmailTo is the user's address; resolving it from userID is the
// caller's job (the auth service has the user row already, so an
// extra DB read here would be wasteful).
type RequestChallengeInput struct {
	UserID        uuid.UUID
	EmailTo       string
	ClientIP      string
	UserAgentHash string
}

// RequestChallenge generates a fresh 6-digit code, persists the
// bcrypt-hashed copy, and emails the plaintext to the user. The
// plaintext NEVER touches the database — it lives only in the email
// payload and the in-memory variable that goes out of scope when this
// function returns.
//
// Returns the freshly-created challenge so the caller can echo its id
// to the client (the verify endpoint accepts user_id alone, but
// returning the id is useful for logging and future challenge_id
// flows). Errors from email delivery are returned to the caller so
// the login path can refuse to gate on a 2FA code that never reached
// the user.
func (s *Service) RequestChallenge(ctx context.Context, in RequestChallengeInput) (*twofactor.Challenge, error) {
	if in.UserID == uuid.Nil {
		return nil, twofactor.ErrUserIDRequired
	}

	code, err := twofactor.GenerateCode()
	if err != nil {
		return nil, fmt.Errorf("twofactor: generate code: %w", err)
	}

	hashed, err := s.hasher.Hash(code)
	if err != nil {
		return nil, fmt.Errorf("twofactor: hash code: %w", err)
	}

	challenge, err := twofactor.New(twofactor.NewChallengeInput{
		UserID:        in.UserID,
		CodeHash:      hashed,
		ClientIP:      in.ClientIP,
		UserAgentHash: in.UserAgentHash,
	})
	if err != nil {
		return nil, fmt.Errorf("twofactor: build challenge: %w", err)
	}

	if err := s.challenges.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("twofactor: persist challenge: %w", err)
	}

	if in.EmailTo != "" {
		if sendErr := s.sendChallengeEmail(ctx, in.EmailTo, code); sendErr != nil {
			// Email delivery failure is surfaced to the caller. The
			// challenge row stays in the DB — it will expire naturally
			// in 10 minutes and the user can request a fresh code via
			// the resend endpoint (B.6.2 follow-up). Logging the error
			// here gives the SOC a breadcrumb if the SMTP backend is
			// down.
			slog.Error("twofactor: send challenge email failed",
				"user_id", in.UserID, "error", sendErr)
			return challenge, fmt.Errorf("twofactor: send email: %w", sendErr)
		}
	}

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &in.UserID,
		Action:       AuditActionChallengeIssued,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &in.UserID,
		Metadata: map[string]any{
			"challenge_id": challenge.ID.String(),
		},
		IPAddress: in.ClientIP,
	})

	return challenge, nil
}

// VerifyChallengeInput is the verify path's narrow argument bundle.
// Code is the plaintext the user typed; the service bcrypt-compares
// against the latest pending row's hash.
type VerifyChallengeInput struct {
	UserID uuid.UUID
	Code   string
}

// VerifyChallenge looks up the latest pending challenge, bcrypt-checks
// the code, and either marks it used (success) or decrements
// attempts_left (mismatch). Returns the matched challenge on success
// so the caller can include its id in audit trails.
//
// Error mapping (each wraps a domain sentinel for handler matching):
//
//   - twofactor.ErrChallengeNotFound — no pending row
//   - twofactor.ErrChallengeExpired — pending row past expires_at
//   - twofactor.ErrAttemptsExhausted — attempts_left already at 0
//   - twofactor.ErrCodeMismatch — bcrypt compare failed (attempts_left
//     decremented as a side effect)
func (s *Service) VerifyChallenge(ctx context.Context, in VerifyChallengeInput) (*twofactor.Challenge, error) {
	if in.UserID == uuid.Nil {
		return nil, twofactor.ErrUserIDRequired
	}
	if in.Code == "" {
		return nil, twofactor.ErrCodeMismatch
	}

	challenge, err := s.challenges.FindLatestPendingForUser(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrTwoFactorChallengeNotFound) {
			s.logAudit(ctx, audit.NewEntryInput{
				UserID:       &in.UserID,
				Action:       AuditActionChallengeFailure,
				ResourceType: audit.ResourceTypeUser,
				ResourceID:   &in.UserID,
				Metadata:     map[string]any{"reason": "no_pending_challenge"},
			})
			return nil, twofactor.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("twofactor: load challenge: %w", err)
	}

	if challenge.IsExpired() {
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &in.UserID,
			Action:       AuditActionChallengeFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &in.UserID,
			Metadata: map[string]any{
				"reason":       "expired",
				"challenge_id": challenge.ID.String(),
			},
		})
		return nil, twofactor.ErrChallengeExpired
	}

	if challenge.AttemptsLeft <= 0 {
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &in.UserID,
			Action:       AuditActionChallengeFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &in.UserID,
			Metadata: map[string]any{
				"reason":       "attempts_exhausted",
				"challenge_id": challenge.ID.String(),
			},
		})
		return nil, twofactor.ErrAttemptsExhausted
	}

	if compareErr := s.hasher.Compare(challenge.CodeHash, in.Code); compareErr != nil {
		// Decrement first so a flaky DB does not give the attacker a
		// free retry — the comparison failed, so the attempt counts
		// regardless of whether the row mutation persists. Errors on
		// the decrement are logged and swallowed because the user-
		// visible outcome is "wrong code" either way.
		if decErr := s.challenges.DecrementAttempts(ctx, challenge.ID); decErr != nil {
			slog.Warn("twofactor: decrement attempts failed",
				"challenge_id", challenge.ID, "error", decErr)
		}
		s.logAudit(ctx, audit.NewEntryInput{
			UserID:       &in.UserID,
			Action:       AuditActionChallengeFailure,
			ResourceType: audit.ResourceTypeUser,
			ResourceID:   &in.UserID,
			Metadata: map[string]any{
				"reason":       "code_mismatch",
				"challenge_id": challenge.ID.String(),
			},
		})
		return nil, twofactor.ErrCodeMismatch
	}

	if err := s.challenges.MarkUsed(ctx, challenge.ID); err != nil {
		return nil, fmt.Errorf("twofactor: mark used: %w", err)
	}
	challenge.MarkUsed()

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &in.UserID,
		Action:       AuditActionChallengeSuccess,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &in.UserID,
		Metadata: map[string]any{
			"challenge_id": challenge.ID.String(),
		},
	})

	return challenge, nil
}

// sendChallengeEmail renders the plaintext code into the same minimal
// HTML envelope every other transactional email uses. We don't ship a
// dedicated template file because the body is two short paragraphs —
// a string literal here is more readable than yet another template
// indirection.
func (s *Service) sendChallengeEmail(ctx context.Context, to, code string) error {
	subject := "Code de vérification — Marketplace Service"
	html := fmt.Sprintf(
		`<p>Bonjour,</p>`+
			`<p>Voici ton code de vérification : <strong style="font-size:18px;letter-spacing:2px">%s</strong></p>`+
			`<p>Ce code est valable 10 minutes. Si tu n'es pas à l'origine de cette demande, tu peux ignorer cet email — personne d'autre n'a accès à ton compte sans ce code.</p>`,
		code,
	)
	return s.email.SendNotification(ctx, to, subject, html)
}

// logAudit is a fire-and-forget audit emission helper. Failures are
// logged via slog but never returned to the caller — audit
// completeness is best-effort by policy (mirrors auth.Service).
func (s *Service) logAudit(ctx context.Context, in audit.NewEntryInput) {
	if s.audits == nil {
		return
	}
	entry, err := audit.NewEntry(in)
	if err != nil {
		slog.Warn("twofactor: build audit entry failed",
			"action", in.Action, "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("twofactor: insert audit entry failed",
			"action", in.Action, "error", err)
	}
}
