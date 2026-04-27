package invoicing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// These tests cover the invoicing-email default behaviour added when
// the "Email de facturation" form field was removed from the billing
// profile form. The service must back-fill profile.InvoicingEmail with
// the org owner's account email on every read path so invoice
// generation, the Stripe Customer enrichment, and the frontend always
// see a usable address.

func TestGetBillingProfile_DefaultsInvoicingEmailFromOwnerAccount(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	ownerID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return &domain.BillingProfile{
			OrganizationID: orgID,
			ProfileType:    domain.ProfileBusiness,
			LegalName:      "Acme",
			InvoicingEmail: "", // empty -> service must back-fill from owner
		}, nil
	}

	orgRepo := stubOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			require.Equal(t, orgID, id)
			return &organization.Organization{ID: orgID, OwnerUserID: ownerID}, nil
		},
	}
	userRepo := stubUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			require.Equal(t, ownerID, id)
			return &user.User{ID: ownerID, Email: "owner@acme.example"}, nil
		},
	}
	svc.SetBillingProfileDeps(invoicingapp.BillingProfileDeps{
		Organizations: orgRepo,
		Users:         userRepo,
	})

	snap, err := svc.GetBillingProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "owner@acme.example", snap.Profile.InvoicingEmail,
		"empty invoicing_email must be back-filled with the owner's account email")
}

func TestGetBillingProfile_KeepsExplicitInvoicingEmail(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	ownerID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return &domain.BillingProfile{
			OrganizationID: orgID,
			ProfileType:    domain.ProfileBusiness,
			InvoicingEmail: "billing-override@acme.example", // user-set
		}, nil
	}
	orgRepo := stubOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: orgID, OwnerUserID: ownerID}, nil
		},
	}
	userRepo := stubUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return &user.User{ID: ownerID, Email: "owner@acme.example"}, nil
		},
	}
	svc.SetBillingProfileDeps(invoicingapp.BillingProfileDeps{
		Organizations: orgRepo,
		Users:         userRepo,
	})

	snap, err := svc.GetBillingProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "billing-override@acme.example", snap.Profile.InvoicingEmail,
		"explicit invoicing_email must NOT be overwritten by the owner default")
}

func TestGetBillingProfile_EmptyStubAlsoDefaultsEmail(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	ownerID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return nil, domain.ErrNotFound
	}
	orgRepo := stubOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: orgID, OwnerUserID: ownerID}, nil
		},
	}
	userRepo := stubUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return &user.User{ID: ownerID, Email: "owner@acme.example"}, nil
		},
	}
	svc.SetBillingProfileDeps(invoicingapp.BillingProfileDeps{
		Organizations: orgRepo,
		Users:         userRepo,
	})

	snap, err := svc.GetBillingProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "owner@acme.example", snap.Profile.InvoicingEmail,
		"first-time empty stub must also receive the owner-email default")
}

func TestGetBillingProfileSnapshotForStripe_DefaultsInvoicingEmail(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	ownerID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return &domain.BillingProfile{
			OrganizationID: orgID,
			LegalName:      "Acme",
			InvoicingEmail: "",
		}, nil
	}
	orgRepo := stubOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: orgID, OwnerUserID: ownerID}, nil
		},
	}
	userRepo := stubUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return &user.User{ID: ownerID, Email: "owner@acme.example"}, nil
		},
	}
	svc.SetBillingProfileDeps(invoicingapp.BillingProfileDeps{
		Organizations: orgRepo,
		Users:         userRepo,
	})

	snap, err := svc.GetBillingProfileSnapshotForStripe(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "owner@acme.example", snap.InvoicingEmail,
		"Stripe snapshot must surface the owner-email default just like the frontend snapshot")
}

func TestGetBillingProfile_NoDepsLeavesEmailEmpty(t *testing.T) {
	// When the optional Users/Organizations deps are not wired (legacy
	// callers, /remove-feature pruning), the read paths must still
	// return a snapshot — the email simply stays empty.
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return &domain.BillingProfile{OrganizationID: orgID, InvoicingEmail: ""}, nil
	}

	snap, err := svc.GetBillingProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Empty(t, snap.Profile.InvoicingEmail)
}

// --- minimal stub repos ---

// stubOrgRepo implements repository.OrganizationRepository for the
// invoicing email-default tests. Only FindByID is exercised; every
// other method is a no-op default safe enough for the few callers.
type stubOrgRepo struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
}

func (s stubOrgRepo) Create(context.Context, *organization.Organization) error {
	return nil
}
func (s stubOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (s stubOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return nil, organization.ErrOrgNotFound
}
func (s stubOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (s stubOrgRepo) FindByUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (s stubOrgRepo) Update(context.Context, *organization.Organization) error { return nil }
func (s stubOrgRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (s stubOrgRepo) SaveRoleOverrides(context.Context, uuid.UUID, organization.RoleOverrides) error {
	return nil
}
func (s stubOrgRepo) CountAll(context.Context) (int, error) { return 0, nil }
func (s stubOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (s stubOrgRepo) ListKYCPending(context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (s stubOrgRepo) ListWithStripeAccount(context.Context) ([]uuid.UUID, error) { return nil, nil }
func (s stubOrgRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (s stubOrgRepo) GetStripeAccountByUserID(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (s stubOrgRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (s stubOrgRepo) ClearStripeAccount(context.Context, uuid.UUID) error { return nil }
func (s stubOrgRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (s stubOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error {
	return nil
}
func (s stubOrgRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (s stubOrgRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}

var _ repository.OrganizationRepository = stubOrgRepo{}

// stubUserRepo implements repository.UserRepository for the invoicing
// email-default tests. Only GetByID is exercised; every other method is
// a no-op default.
type stubUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
}

func (s stubUserRepo) Create(context.Context, *user.User) error { return nil }
func (s stubUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (s stubUserRepo) GetByEmail(context.Context, string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s stubUserRepo) Update(context.Context, *user.User) error { return nil }
func (s stubUserRepo) Delete(context.Context, uuid.UUID) error  { return nil }
func (s stubUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}
func (s stubUserRepo) ListAdmin(context.Context, repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (s stubUserRepo) CountAdmin(context.Context, repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (s stubUserRepo) CountByRole(context.Context) (map[string]int, error)   { return nil, nil }
func (s stubUserRepo) CountByStatus(context.Context) (map[string]int, error) { return nil, nil }
func (s stubUserRepo) RecentSignups(context.Context, int) ([]*user.User, error) {
	return nil, nil
}
func (s stubUserRepo) BumpSessionVersion(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (s stubUserRepo) GetSessionVersion(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (s stubUserRepo) UpdateEmailNotificationsEnabled(context.Context, uuid.UUID, bool) error {
	return nil
}
func (s stubUserRepo) TouchLastActive(context.Context, uuid.UUID) error { return nil }

var _ repository.UserRepository = stubUserRepo{}
