package profilepricing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainpricing "marketplace-backend/internal/domain/profilepricing"
)

// ---- Upsert ----

func TestService_Upsert_HappyPath_ProviderDirectDaily(t *testing.T) {
	orgID := uuid.New()
	var persisted *domainpricing.Pricing

	repo := &mockPricingRepo{
		UpsertFn: func(_ context.Context, p *domainpricing.Pricing) error {
			persisted = p
			return nil
		},
	}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeProviderPersonal, false, nil
		},
	}
	svc := NewService(repo, resolver)

	p, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: orgID,
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeDaily,
		MinAmount:      60000,
		Currency:       "EUR",
		Note:           "TJM standard",
		Negotiable:     true,
	})

	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotNil(t, persisted, "repo.Upsert should have been called")
	assert.Equal(t, orgID, p.OrganizationID)
	assert.Equal(t, domainpricing.KindDirect, p.Kind)
	assert.Equal(t, domainpricing.TypeDaily, p.Type)
	assert.Equal(t, int64(60000), p.MinAmount)
	assert.Equal(t, "EUR", p.Currency)
	assert.True(t, p.Negotiable)
}

func TestService_Upsert_Agency_DailyRejected(t *testing.T) {
	repo := &mockPricingRepo{
		UpsertFn: func(_ context.Context, _ *domainpricing.Pricing) error { return nil },
	}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeAgency, false, nil
		},
	}
	svc := NewService(repo, resolver)

	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeDaily,
		MinAmount:      60000,
		Currency:       "EUR",
	})
	assert.ErrorIs(t, err, domainpricing.ErrTypeNotAllowedForOrg)
}

func TestService_Upsert_Agency_HourlyRejected(t *testing.T) {
	repo := &mockPricingRepo{}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeAgency, false, nil
		},
	}
	svc := NewService(repo, resolver)

	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeHourly,
		MinAmount:      7500,
		Currency:       "EUR",
	})
	assert.ErrorIs(t, err, domainpricing.ErrTypeNotAllowedForOrg)
}

func TestService_Upsert_Agency_ProjectFromAccepted(t *testing.T) {
	repo := &mockPricingRepo{
		UpsertFn: func(_ context.Context, _ *domainpricing.Pricing) error { return nil },
	}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeAgency, false, nil
		},
	}
	svc := NewService(repo, resolver)

	p, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeProjectFrom,
		MinAmount:      1000000,
		Currency:       "EUR",
	})
	require.NoError(t, err)
	assert.Equal(t, domainpricing.TypeProjectFrom, p.Type)
}

// TestService_Upsert_Agency_V1RejectsDeprecatedDirectTypes asserts
// the V1 pricing simplification: agency orgs on the direct kind may
// only declare project_from. project_range (previously legal) must
// fail fast with ErrPricingTypeNotAllowed and MUST NOT reach the
// repository. The other direct types (daily/hourly) are already
// rejected upstream by the kind-level whitelist, so they do not
// need an additional V1 case here.
func TestService_Upsert_Agency_V1RejectsDeprecatedDirectTypes(t *testing.T) {
	cases := []struct {
		name  string
		input UpsertInput
	}{
		{
			name: "project_range",
			input: UpsertInput{
				Kind:      domainpricing.KindDirect,
				Type:      domainpricing.TypeProjectRange,
				MinAmount: 1000000,
				MaxAmount: ptrInt64(5000000),
				Currency:  "EUR",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upsertCalled := false
			repo := &mockPricingRepo{
				UpsertFn: func(_ context.Context, _ *domainpricing.Pricing) error {
					upsertCalled = true
					return nil
				},
			}
			resolver := &mockOrgInfoResolver{
				GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
					return domainpricing.OrgTypeAgency, false, nil
				},
			}
			svc := NewService(repo, resolver)

			tc.input.OrganizationID = uuid.New()
			_, err := svc.Upsert(context.Background(), tc.input)
			assert.ErrorIs(t, err, domainpricing.ErrPricingTypeNotAllowed)
			assert.False(t, upsertCalled, "deprecated agency type must never reach the repository")
		})
	}
}

// ptrInt64 is a tiny helper shared by the V1 tests — kept here so
// we do not leak a package-level helper just for a single call site.
func ptrInt64(v int64) *int64 { return &v }

func TestService_Upsert_HappyPath_ReferralCommissionPct(t *testing.T) {
	repo := &mockPricingRepo{
		UpsertFn: func(_ context.Context, _ *domainpricing.Pricing) error { return nil },
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	max := int64(1500)
	p, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindReferral,
		Type:           domainpricing.TypeCommissionPct,
		MinAmount:      500,
		MaxAmount:      &max,
		Currency:       "pct",
	})
	require.NoError(t, err)
	assert.Equal(t, domainpricing.KindReferral, p.Kind)
	assert.Equal(t, "pct", p.Currency)
}

func TestService_Upsert_RoleCheck_RejectsReferralForAgency(t *testing.T) {
	repo := &mockPricingRepo{} // Upsert should never be called
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeAgency, false, nil
		},
	}
	svc := NewService(repo, resolver)

	max := int64(1500)
	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindReferral,
		Type:           domainpricing.TypeCommissionPct,
		MinAmount:      500,
		MaxAmount:      &max,
		Currency:       "pct",
	})

	assert.ErrorIs(t, err, domainpricing.ErrKindNotAllowedForRole)
}

func TestService_Upsert_RoleCheck_RejectsAnythingForEnterprise(t *testing.T) {
	repo := &mockPricingRepo{}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeEnterprise, false, nil
		},
	}
	svc := NewService(repo, resolver)

	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeDaily,
		MinAmount:      100,
		Currency:       "EUR",
	})
	assert.ErrorIs(t, err, domainpricing.ErrKindNotAllowedForRole)
}

func TestService_Upsert_RoleCheck_ProviderWithoutReferrer_RejectsReferralKind(t *testing.T) {
	repo := &mockPricingRepo{}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return domainpricing.OrgTypeProviderPersonal, false, nil
		},
	}
	svc := NewService(repo, resolver)

	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindReferral,
		Type:           domainpricing.TypeCommissionFlat,
		MinAmount:      500,
		Currency:       "EUR",
	})
	assert.ErrorIs(t, err, domainpricing.ErrKindNotAllowedForRole)
}

func TestService_Upsert_ResolverError_Propagates(t *testing.T) {
	want := errors.New("db blew up")
	repo := &mockPricingRepo{}
	resolver := &mockOrgInfoResolver{
		GetOrgInfoFn: func(_ context.Context, _ uuid.UUID) (string, bool, error) {
			return "", false, want
		},
	}
	svc := NewService(repo, resolver)

	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeDaily,
		MinAmount:      1,
		Currency:       "EUR",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, want)
	assert.Contains(t, err.Error(), "resolve org")
}

func TestService_Upsert_DomainValidationFailure_DoesNotPersist(t *testing.T) {
	var upsertCalled bool
	repo := &mockPricingRepo{
		UpsertFn: func(_ context.Context, _ *domainpricing.Pricing) error {
			upsertCalled = true
			return nil
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	// Missing max for a range type.
	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeProjectRange,
		MinAmount:      100,
		Currency:       "EUR",
	})
	assert.ErrorIs(t, err, domainpricing.ErrRangeRequiredForType)
	assert.False(t, upsertCalled)
}

func TestService_Upsert_PersistError_Propagates(t *testing.T) {
	want := errors.New("db unreachable")
	repo := &mockPricingRepo{
		UpsertFn: func(_ context.Context, _ *domainpricing.Pricing) error {
			return want
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	_, err := svc.Upsert(context.Background(), UpsertInput{
		OrganizationID: uuid.New(),
		Kind:           domainpricing.KindDirect,
		Type:           domainpricing.TypeDaily,
		MinAmount:      1000,
		Currency:       "EUR",
	})
	assert.ErrorIs(t, err, want)
	assert.Contains(t, err.Error(), "persist")
}

// ---- GetForOrg ----

func TestService_GetForOrg_Success(t *testing.T) {
	orgID := uuid.New()
	expected := []*domainpricing.Pricing{
		{OrganizationID: orgID, Kind: domainpricing.KindDirect, Type: domainpricing.TypeDaily, MinAmount: 50000, Currency: "EUR"},
	}
	repo := &mockPricingRepo{
		FindByOrgIDFn: func(_ context.Context, id uuid.UUID) ([]*domainpricing.Pricing, error) {
			assert.Equal(t, orgID, id)
			return expected, nil
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	got, err := svc.GetForOrg(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_GetForOrg_RepoError(t *testing.T) {
	repo := &mockPricingRepo{
		FindByOrgIDFn: func(_ context.Context, _ uuid.UUID) ([]*domainpricing.Pricing, error) {
			return nil, fmt.Errorf("down")
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	_, err := svc.GetForOrg(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile pricing get")
}

// ---- GetForOrgsBatch ----

func TestService_GetForOrgsBatch_Passthrough(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	expected := map[uuid.UUID][]*domainpricing.Pricing{
		ids[0]: {{Kind: domainpricing.KindDirect}},
		ids[1]: {},
	}
	repo := &mockPricingRepo{
		ListByOrgIDsFn: func(_ context.Context, got []uuid.UUID) (map[uuid.UUID][]*domainpricing.Pricing, error) {
			assert.ElementsMatch(t, ids, got)
			return expected, nil
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	out, err := svc.GetForOrgsBatch(context.Background(), ids)
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestService_GetForOrgsBatch_RepoError(t *testing.T) {
	repo := &mockPricingRepo{
		ListByOrgIDsFn: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]*domainpricing.Pricing, error) {
			return nil, fmt.Errorf("nope")
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	_, err := svc.GetForOrgsBatch(context.Background(), []uuid.UUID{uuid.New()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile pricing batch get")
}

// ---- DeleteKind ----

func TestService_DeleteKind_Success(t *testing.T) {
	var called bool
	repo := &mockPricingRepo{
		DeleteByKindFn: func(_ context.Context, _ uuid.UUID, k domainpricing.PricingKind) error {
			called = true
			assert.Equal(t, domainpricing.KindDirect, k)
			return nil
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	err := svc.DeleteKind(context.Background(), uuid.New(), domainpricing.KindDirect)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_DeleteKind_InvalidKind_NoRepoCall(t *testing.T) {
	var called bool
	repo := &mockPricingRepo{
		DeleteByKindFn: func(_ context.Context, _ uuid.UUID, _ domainpricing.PricingKind) error {
			called = true
			return nil
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	err := svc.DeleteKind(context.Background(), uuid.New(), domainpricing.PricingKind("wild"))
	assert.ErrorIs(t, err, domainpricing.ErrInvalidKind)
	assert.False(t, called)
}

func TestService_DeleteKind_RepoError_Wrapped(t *testing.T) {
	repo := &mockPricingRepo{
		DeleteByKindFn: func(_ context.Context, _ uuid.UUID, _ domainpricing.PricingKind) error {
			return errors.New("db unreachable")
		},
	}
	svc := NewService(repo, &mockOrgInfoResolver{})

	err := svc.DeleteKind(context.Background(), uuid.New(), domainpricing.KindDirect)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile pricing delete")
}
