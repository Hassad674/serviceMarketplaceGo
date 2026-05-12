package referral

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// stubOrgReader satisfies repository.OrganizationReader. Only the
// FindByUserID branch is exercised by the resolver — every other
// method panics so an accidental new call surfaces in tests.
type stubOrgReader struct {
	byUserID func(ctx context.Context, userID uuid.UUID) (*organization.Organization, error)
}

func (s *stubOrgReader) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	panic("FindByID must not be called by the display-name resolver")
}
func (s *stubOrgReader) FindByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error) {
	panic("FindByOwnerUserID must not be called by the display-name resolver")
}
func (s *stubOrgReader) FindByUserID(ctx context.Context, userID uuid.UUID) (*organization.Organization, error) {
	if s.byUserID == nil {
		return nil, nil
	}
	return s.byUserID(ctx, userID)
}
func (s *stubOrgReader) FindByStripeAccountID(ctx context.Context, stripeAccountID string) (*organization.Organization, error) {
	panic("FindByStripeAccountID must not be called")
}
func (s *stubOrgReader) CountAll(ctx context.Context) (int, error) {
	panic("CountAll must not be called")
}
func (s *stubOrgReader) ListKYCPending(ctx context.Context) ([]*organization.Organization, error) {
	panic("ListKYCPending must not be called")
}
func (s *stubOrgReader) ListWithStripeAccount(ctx context.Context) ([]uuid.UUID, error) {
	panic("ListWithStripeAccount must not be called")
}

// stubUserReader satisfies repository.UserReader. Same approach as
// stubOrgReader — only GetByID is wired.
type stubUserReader struct {
	byID func(ctx context.Context, id uuid.UUID) (*user.User, error)
}

func (s *stubUserReader) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if s.byID == nil {
		return nil, nil
	}
	return s.byID(ctx, id)
}
func (s *stubUserReader) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	panic("GetByEmail must not be called")
}
func (s *stubUserReader) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	panic("ExistsByEmail must not be called")
}
func (s *stubUserReader) ListAdmin(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, error) {
	panic("ListAdmin must not be called")
}
func (s *stubUserReader) CountAdmin(ctx context.Context, filters repository.AdminUserFilters) (int, error) {
	panic("CountAdmin must not be called")
}
func (s *stubUserReader) CountByRole(ctx context.Context) (map[string]int, error) {
	panic("CountByRole must not be called")
}
func (s *stubUserReader) CountByStatus(ctx context.Context) (map[string]int, error) {
	panic("CountByStatus must not be called")
}
func (s *stubUserReader) RecentSignups(ctx context.Context, limit int) ([]*user.User, error) {
	panic("RecentSignups must not be called")
}

// TestPartyDisplayNameResolver covers the three resolution branches:
//   - User owns an agency org → org name wins
//   - User owns an enterprise org → org name wins
//   - User has no org (or org is provider_personal) → FullName fallback
func TestPartyDisplayNameResolver_ResolveDisplayName(t *testing.T) {
	userID := uuid.New()

	cases := []struct {
		name      string
		orgFn     func(ctx context.Context, userID uuid.UUID) (*organization.Organization, error)
		userFn    func(ctx context.Context, id uuid.UUID) (*user.User, error)
		expectOut string
	}{
		{
			name: "agency org → org name wins",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{
					ID:   uuid.New(),
					Type: organization.OrgTypeAgency,
					Name: "Atelier Lumen",
				}, nil
			},
			userFn:    nil,
			expectOut: "Atelier Lumen",
		},
		{
			name: "enterprise org → org name wins",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{
					ID:   uuid.New(),
					Type: organization.OrgTypeEnterprise,
					Name: "Banque du Sud",
				}, nil
			},
			expectOut: "Banque du Sud",
		},
		{
			name: "provider_personal org → falls back to FullName",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{
					ID:   uuid.New(),
					Type: organization.OrgTypeProviderPersonal,
					Name: "ignored",
				}, nil
			},
			userFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{FirstName: "Marie", LastName: "Curie"}, nil
			},
			expectOut: "Marie Curie",
		},
		{
			name: "no org → FullName fallback",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return nil, nil
			},
			userFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{FirstName: "Ada", LastName: "Lovelace"}, nil
			},
			expectOut: "Ada Lovelace",
		},
		{
			name: "org lookup error → FullName fallback",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return nil, errors.New("transient")
			},
			userFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{FirstName: "Grace", LastName: "Hopper"}, nil
			},
			expectOut: "Grace Hopper",
		},
		{
			name: "user not found → empty string (UI degrades to placeholder)",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return nil, nil
			},
			userFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return nil, errors.New("not found")
			},
			expectOut: "",
		},
		{
			name: "blank org name → falls through to FullName",
			orgFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{
					ID:   uuid.New(),
					Type: organization.OrgTypeAgency,
					Name: "   ",
				}, nil
			},
			userFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{FirstName: "Linus", LastName: "Torvalds"}, nil
			},
			expectOut: "Linus Torvalds",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			users := &stubUserReader{byID: tc.userFn}
			orgs := &stubOrgReader{byUserID: tc.orgFn}
			res := NewOrgFirstPartyDisplayNameResolver(users, orgs)
			got, err := res.ResolveDisplayName(context.Background(), userID)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectOut, got)
		})
	}
}

// TestPartyDisplayNameResolver_NilSafe ensures the resolver is safe to
// call with nil dependencies — production should never wire nil, but
// minimal smoke tests do, and a nil-panic at request time would crash
// the apporteur detail page.
func TestPartyDisplayNameResolver_NilSafe(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *OrgFirstPartyDisplayNameResolver
		got, err := r.ResolveDisplayName(context.Background(), uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, "", got)
	})
	t.Run("nil users + nil orgs", func(t *testing.T) {
		r := NewOrgFirstPartyDisplayNameResolver(nil, nil)
		got, err := r.ResolveDisplayName(context.Background(), uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, "", got)
	})
	t.Run("nil orgs only → falls through to user lookup", func(t *testing.T) {
		users := &stubUserReader{byID: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return &user.User{FirstName: "Solo", LastName: "User"}, nil
		}}
		r := NewOrgFirstPartyDisplayNameResolver(users, nil)
		got, err := r.ResolveDisplayName(context.Background(), uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, "Solo User", got)
	})
}
