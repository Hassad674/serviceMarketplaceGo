package organization

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DefaultInvitationDuration is how long a newly-generated invitation stays
// valid before it expires. 7 days was chosen as a balance between giving
// the recipient enough time to act (including weekends and holidays) and
// not letting stale invitations accumulate.
const DefaultInvitationDuration = 7 * 24 * time.Hour

// invitationTokenBytes is the length of the random token used to match a
// click-through to an invitation row. 32 bytes = 256 bits = unguessable
// without a collision of cosmic improbability. Encoded in hex, that's 64
// characters in the URL.
const invitationTokenBytes = 32

// maxNameLength caps first/last name inputs to prevent obvious abuse.
const maxNameLength = 100

// InvitationStatus tracks where an invitation is in its lifecycle.
type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusCancelled InvitationStatus = "cancelled"
	InvitationStatusExpired   InvitationStatus = "expired"
)

// IsValid reports whether the status is a known value.
func (s InvitationStatus) IsValid() bool {
	switch s {
	case InvitationStatusPending, InvitationStatusAccepted, InvitationStatusCancelled, InvitationStatusExpired:
		return true
	}
	return false
}

// Invitation represents a pending (or completed) invitation for a new
// operator to join an organization.
//
// The token is generated at construction time and is the only identifier
// the recipient ever sees — they click /invitation/{token}, the backend
// looks it up, validates status+expiry, and either presents the registration
// form (new user) or promotes the existing user's access (they already
// have an account — deferred to V2; V1 always requires a fresh account).
type Invitation struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Email           string
	FirstName       string
	LastName        string
	Title           string
	Role            Role
	Token           string
	InvitedByUserID uuid.UUID
	Status          InvitationStatus
	ExpiresAt       time.Time
	AcceptedAt      *time.Time
	CancelledAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewInvitationInput gathers the fields required to create an invitation.
// Grouped into a struct because NewInvitation would otherwise take 8
// positional arguments, violating the project's 4-param rule.
type NewInvitationInput struct {
	OrganizationID  uuid.UUID
	Email           string
	FirstName       string
	LastName        string
	Title           string
	Role            Role
	InvitedByUserID uuid.UUID
	Duration        time.Duration // optional — zero value falls back to DefaultInvitationDuration
}

// NewInvitation creates a validated invitation with a freshly-generated
// token and the given expiration. If Duration is zero, it defaults to
// DefaultInvitationDuration (7 days).
//
// The Role must be one of Admin/Member/Viewer — inviting as Owner is
// forbidden (use the transfer ownership flow instead).
func NewInvitation(in NewInvitationInput) (*Invitation, error) {
	if in.OrganizationID == uuid.Nil || in.InvitedByUserID == uuid.Nil {
		return nil, ErrInvitationNotFound
	}
	if !in.Role.IsValid() {
		return nil, ErrInvalidRole
	}
	if !in.Role.CanBeInvitedAs() {
		return nil, ErrCannotInviteAsOwner
	}

	email := normalizeEmail(in.Email)
	if !isPlausibleEmail(email) {
		return nil, ErrInvalidEmail
	}
	if strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" {
		return nil, ErrNameRequired
	}
	if len(in.FirstName) > maxNameLength || len(in.LastName) > maxNameLength {
		return nil, ErrNameTooLong
	}
	if len(in.Title) > maxTitleLength {
		return nil, ErrTitleTooLong
	}

	token, err := generateInvitationToken()
	if err != nil {
		return nil, err
	}

	duration := in.Duration
	if duration <= 0 {
		duration = DefaultInvitationDuration
	}
	now := time.Now()
	return &Invitation{
		ID:              uuid.New(),
		OrganizationID:  in.OrganizationID,
		Email:           email,
		FirstName:       strings.TrimSpace(in.FirstName),
		LastName:        strings.TrimSpace(in.LastName),
		Title:           strings.TrimSpace(in.Title),
		Role:            in.Role,
		Token:           token,
		InvitedByUserID: in.InvitedByUserID,
		Status:          InvitationStatusPending,
		ExpiresAt:       now.Add(duration),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// IsExpired reports whether the invitation has passed its expiry window.
// It considers the status field as well: an already-accepted or cancelled
// invitation is never "expired" (it's in its terminal state for a
// different reason).
func (i *Invitation) IsExpired() bool {
	if i.Status != InvitationStatusPending {
		return false
	}
	return time.Now().After(i.ExpiresAt)
}

// Accept marks the invitation as accepted. Returns an error if the
// invitation is not in a state that permits acceptance (already used,
// cancelled, or expired).
func (i *Invitation) Accept() error {
	switch i.Status {
	case InvitationStatusAccepted:
		return ErrInvitationAlreadyUsed
	case InvitationStatusCancelled:
		return ErrInvitationCancelled
	case InvitationStatusExpired:
		return ErrInvitationExpired
	}
	if i.IsExpired() {
		// Flip to the expired status so subsequent reads see a consistent
		// value without waiting for a background sweeper.
		i.Status = InvitationStatusExpired
		i.UpdatedAt = time.Now()
		return ErrInvitationExpired
	}

	now := time.Now()
	i.Status = InvitationStatusAccepted
	i.AcceptedAt = &now
	i.UpdatedAt = now
	return nil
}

// Cancel marks the invitation as cancelled. Idempotent in the sense that
// cancelling an already-cancelled invitation returns an error (so the
// caller knows nothing changed) rather than silently no-op'ing.
func (i *Invitation) Cancel() error {
	if i.Status != InvitationStatusPending {
		return ErrInvalidInvitationStatus
	}
	now := time.Now()
	i.Status = InvitationStatusCancelled
	i.CancelledAt = &now
	i.UpdatedAt = now
	return nil
}

// MarkExpired transitions a pending invitation to expired. Called by the
// background sweeper that runs periodically to clean up stale invitations.
// Returns an error if the invitation is not pending.
func (i *Invitation) MarkExpired() error {
	if i.Status != InvitationStatusPending {
		return ErrInvalidInvitationStatus
	}
	i.Status = InvitationStatusExpired
	i.UpdatedAt = time.Now()
	return nil
}

// generateInvitationToken returns a 64-character hex string from 32 bytes
// of crypto-random data. A collision probability at 1 billion invitations
// is still under 10^-40, so we never check for token uniqueness; the DB
// UNIQUE constraint is a safety net, not a normal code path.
func generateInvitationToken() (string, error) {
	buf := make([]byte, invitationTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// normalizeEmail lowercases and trims the email. We intentionally do NOT
// canonicalize gmail-style dots ("john.doe" == "johndoe") because that's
// a Google-specific convention and treating it as universal would cause
// false duplicates on other providers.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isPlausibleEmail runs a minimal sanity check on the input. Full RFC
// validation is overkill and would reject valid addresses; real validation
// happens when the invitation email is actually delivered.
func isPlausibleEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return false
	}
	if strings.IndexByte(email[at+1:], '.') < 0 {
		return false
	}
	return true
}
