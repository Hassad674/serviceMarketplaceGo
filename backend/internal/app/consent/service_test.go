package consent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appconsent "marketplace-backend/internal/app/consent"
	"marketplace-backend/internal/domain/consent"
)

type stubRepo struct {
	created *consent.Entry
	err     error
}

func (s *stubRepo) Create(_ context.Context, entry *consent.Entry) error {
	s.created = entry
	return s.err
}

func TestService_Record_Success(t *testing.T) {
	repo := &stubRepo{}
	svc := appconsent.NewService(repo)

	entry, err := svc.Record(context.Background(), appconsent.RecordInput{
		SessionID:  "sess-9",
		Categories: []string{"analytics"},
		Action:     consent.ActionAcceptAll,
		RawIP:      "203.0.113.42",
		UserAgent:  "Mozilla/5.0",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if entry == nil || entry.ID == uuid.Nil {
		t.Fatalf("expected entry with ID")
	}
	if entry.IPAnonymized != "203.0.x.x" {
		t.Errorf("expected truncated ipv4, got %q", entry.IPAnonymized)
	}
	if len(entry.UserAgentHash) != 64 {
		t.Errorf("expected 64-hex sha256, got %d chars", len(entry.UserAgentHash))
	}
	if repo.created == nil || repo.created.ID != entry.ID {
		t.Errorf("expected repo to receive entry %v", entry.ID)
	}
}

func TestService_Record_RejectsInvalidInput(t *testing.T) {
	repo := &stubRepo{}
	svc := appconsent.NewService(repo)

	_, err := svc.Record(context.Background(), appconsent.RecordInput{
		Categories: []string{"analytics"},
		Action:     "garbage",
		RawIP:      "1.2.3.4",
		UserAgent:  "ua",
	})
	if !errors.Is(err, consent.ErrInvalidAction) {
		t.Errorf("got %v, want ErrInvalidAction", err)
	}
	if repo.created != nil {
		t.Errorf("expected repo NOT to be called on invalid input")
	}
}

func TestService_Record_PropagatesRepoError(t *testing.T) {
	repoErr := errors.New("boom")
	repo := &stubRepo{err: repoErr}
	svc := appconsent.NewService(repo)

	_, err := svc.Record(context.Background(), appconsent.RecordInput{
		Categories: []string{"analytics"},
		Action:     consent.ActionRefuseAll,
		RawIP:      "1.2.3.4",
		UserAgent:  "ua",
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected wrapped repo err, got %v", err)
	}
}
