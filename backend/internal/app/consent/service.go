// Package consent implements the consent_log Record use case.
//
// The handler builds a RecordInput from the HTTP request (anonymized
// IP, hashed UA, derived session id) then calls Record. Record
// validates the input via the domain constructor and persists via
// the repository port — no other side effects.
package consent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/consent"
	"marketplace-backend/internal/domain/gdpr"
	"marketplace-backend/internal/port/repository"
)

// Service exposes the Record use case. Constructed once at boot in
// cmd/api/wire_*.go and shared across handler invocations.
type Service struct {
	repo repository.ConsentLogRepository
}

// NewService is the canonical constructor. The repo dependency is the
// port interface — tests inject a fake; production wires the
// postgres.ConsentLogRepository.
func NewService(repo repository.ConsentLogRepository) *Service {
	return &Service{repo: repo}
}

// RecordInput is the handler-shaped input. The service derives the
// IPAnonymized / UserAgentHash itself so the handler stays a thin DTO
// adapter.
type RecordInput struct {
	UserID     *uuid.UUID
	SessionID  string
	Categories []string
	Action     consent.Action
	RawIP      string
	UserAgent  string
}

// Record builds a domain.Entry from the input, validates it, and
// persists it. Returns the persisted entry so the handler can echo
// the canonical id back to the client (handy for client-side
// debugging without exposing internal state).
func (s *Service) Record(ctx context.Context, in RecordInput) (*consent.Entry, error) {
	entry, err := consent.New(consent.NewInput{
		UserID:        in.UserID,
		SessionID:     in.SessionID,
		Categories:    in.Categories,
		Action:        in.Action,
		IPAnonymized:  gdpr.TruncateIP(in.RawIP),
		UserAgentHash: hashUserAgent(in.UserAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("consent: build entry: %w", err)
	}
	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("consent: persist entry: %w", err)
	}
	return entry, nil
}

// hashUserAgent returns the hex-encoded SHA-256 of the UA. An empty
// string maps to "" (the domain validator will catch that).
func hashUserAgent(ua string) string {
	if ua == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ua))
	return hex.EncodeToString(sum[:])
}
