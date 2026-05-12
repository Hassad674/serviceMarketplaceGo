package referral

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// stubResolver is the smallest possible PartyDisplayNameResolver — used
// to drive Service.ResolvePartyDisplayName through every branch.
type stubResolver struct {
	out string
	err error
}

func (s *stubResolver) ResolveDisplayName(_ context.Context, _ uuid.UUID) (string, error) {
	return s.out, s.err
}

// TestService_ResolvePartyDisplayName_Wrapper covers the three short-
// circuit paths on the Service wrapper: nil receiver, nil resolver,
// resolver-returns-error. None of them must propagate an error to the
// caller — the handler reads the empty string and the DTO omits the
// field.
func TestService_ResolvePartyDisplayName_Wrapper(t *testing.T) {
	t.Run("nil service receiver", func(t *testing.T) {
		var s *Service
		got := s.ResolvePartyDisplayName(context.Background(), uuid.New())
		assert.Equal(t, "", got)
	})
	t.Run("resolver not wired", func(t *testing.T) {
		s := &Service{}
		got := s.ResolvePartyDisplayName(context.Background(), uuid.New())
		assert.Equal(t, "", got)
	})
	t.Run("resolver error → empty string", func(t *testing.T) {
		s := &Service{partyDisplayNames: &stubResolver{err: errors.New("boom")}}
		got := s.ResolvePartyDisplayName(context.Background(), uuid.New())
		assert.Equal(t, "", got)
	})
	t.Run("resolver returns name", func(t *testing.T) {
		s := &Service{partyDisplayNames: &stubResolver{out: "Acme Inc."}}
		got := s.ResolvePartyDisplayName(context.Background(), uuid.New())
		assert.Equal(t, "Acme Inc.", got)
	})
}

// TestNewService_WiresPartyDisplayNames verifies the constructor wires
// the new dependency. Cheap, but catches a future regression where the
// dependency is added to ServiceDeps but forgotten in NewService.
func TestNewService_WiresPartyDisplayNames(t *testing.T) {
	stub := &stubResolver{out: "Bravo"}
	svc := NewService(ServiceDeps{PartyDisplayNames: stub})
	got := svc.ResolvePartyDisplayName(context.Background(), uuid.New())
	assert.Equal(t, "Bravo", got)
}
