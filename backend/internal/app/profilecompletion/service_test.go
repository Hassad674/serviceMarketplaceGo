package profilecompletion_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/profilecompletion"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
)

// completionFixture wires a Service with a deterministic set of
// readers — every reader is configurable per test so we can exercise
// "all empty", "all filled", and the partial scenarios without
// duplicating boilerplate.
type completionFixture struct {
	users *userReaderStub
	orgs  *orgReaderStub
	deps  profilecompletion.Deps
}

func newFixture() *completionFixture {
	return &completionFixture{
		users: &userReaderStub{},
		orgs:  &orgReaderStub{},
	}
}

func (f *completionFixture) build(t *testing.T) *profilecompletion.Service {
	t.Helper()
	d := f.deps
	d.Users = f.users
	d.Organizations = f.orgs
	svc, err := profilecompletion.NewService(d)
	require.NoError(t, err)
	return svc
}

// ---------------------------------------------------------------
// Stubs (reader doubles)
// ---------------------------------------------------------------

type userReaderStub struct {
	user *user.User
	err  error
}

func (s *userReaderStub) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

type orgReaderStub struct {
	org *organization.Organization
	err error
}

func (s *orgReaderStub) FindByID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.org, nil
}

// fakeSharedReader builds the SharedProfile inline so each test case
// can declare the exact shape it cares about.
type fakeSharedReader struct {
	ph    string
	c     string
	cc    string
	langs []string
	err   error
}

func (s *fakeSharedReader) GetSharedProfile(_ context.Context, _ uuid.UUID) (*profilecompletion.SharedProfile, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &profilecompletion.SharedProfile{
		PhotoURL:              s.ph,
		City:                  s.c,
		CountryCode:           s.cc,
		LanguagesProfessional: s.langs,
	}, nil
}

// fakeFreelanceReader builds a FreelanceProfileSnapshot.
type fakeFreelanceReader struct {
	profileID    uuid.UUID
	title        string
	about        string
	video        string
	expertises   []string
	availability profile.AvailabilityStatus
	err          error
}

func (f *fakeFreelanceReader) GetByOrgID(_ context.Context, _ uuid.UUID) (*profilecompletion.FreelanceProfileSnapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &profilecompletion.FreelanceProfileSnapshot{
		ProfileID:          f.profileID,
		Title:              f.title,
		About:              f.about,
		VideoURL:           f.video,
		ExpertiseDomains:   f.expertises,
		AvailabilityStatus: f.availability,
	}, nil
}

// fakeLegacyReader returns a *profile.Profile pointer.
type fakeLegacyReader struct {
	p   *profile.Profile
	err error
}

func (f *fakeLegacyReader) GetByOrgID(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.p, nil
}

// skillsCounterStub returns a fixed count.
type skillsCounterStub struct {
	n   int
	err error
}

func (s *skillsCounterStub) CountByOrg(_ context.Context, _ uuid.UUID) (int, error) {
	return s.n, s.err
}

// socialLinksCounterStub returns per-persona counts.
type socialLinksCounterStub struct {
	freelance int
	referrer  int
	agency    int
	err       error
}

func (s *socialLinksCounterStub) CountByOrgPersona(_ context.Context, _ uuid.UUID, p profile.SocialLinkPersona) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	switch p {
	case profile.PersonaFreelance:
		return s.freelance, nil
	case profile.PersonaReferrer:
		return s.referrer, nil
	case profile.PersonaAgency:
		return s.agency, nil
	}
	return 0, nil
}

type portfolioCounterStub struct {
	n int
}

func (s *portfolioCounterStub) CountByOrganization(_ context.Context, _ uuid.UUID) (int, error) {
	return s.n, nil
}

type pricingExistsStub struct {
	exists bool
}

func (s *pricingExistsStub) ExistsByProfileID(_ context.Context, _ uuid.UUID) (bool, error) {
	return s.exists, nil
}

type legacyPricingCounterStub struct {
	n int
}

func (s *legacyPricingCounterStub) CountByOrgID(_ context.Context, _ uuid.UUID) (int, error) {
	return s.n, nil
}

// ---------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------

func providerUser() *user.User {
	uid := uuid.New()
	return &user.User{
		ID:    uid,
		Email: "p@example.com",
		Role:  user.RoleProvider,
	}
}

func agencyUser() *user.User {
	uid := uuid.New()
	return &user.User{
		ID:    uid,
		Email: "a@example.com",
		Role:  user.RoleAgency,
	}
}

func enterpriseUser() *user.User {
	uid := uuid.New()
	return &user.User{
		ID:    uid,
		Email: "e@example.com",
		Role:  user.RoleEnterprise,
	}
}

func providerOrg() *organization.Organization {
	return &organization.Organization{
		ID:          uuid.New(),
		OwnerUserID: uuid.New(),
		Type:        organization.OrgTypeProviderPersonal,
		Name:        "Solo",
	}
}

func agencyOrg() *organization.Organization {
	return &organization.Organization{
		ID:          uuid.New(),
		OwnerUserID: uuid.New(),
		Type:        organization.OrgTypeAgency,
		Name:        "Agency",
	}
}

func enterpriseOrg() *organization.Organization {
	return &organization.Organization{
		ID:          uuid.New(),
		OwnerUserID: uuid.New(),
		Type:        organization.OrgTypeEnterprise,
		Name:        "Enterprise",
	}
}

// ---------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------

func TestNewService_RequiresUsersReader(t *testing.T) {
	_, err := profilecompletion.NewService(profilecompletion.Deps{
		Organizations: &orgReaderStub{},
	})
	assert.Error(t, err)
}

func TestNewService_RequiresOrganizationsReader(t *testing.T) {
	_, err := profilecompletion.NewService(profilecompletion.Deps{
		Users: &userReaderStub{},
	})
	assert.Error(t, err)
}

// ---------------------------------------------------------------
// Compute — provider/freelance role
// ---------------------------------------------------------------

func TestCompute_Provider_AllEmpty_ZeroPercentExceptDefaults(t *testing.T) {
	f := newFixture()
	f.users.user = providerUser()
	f.orgs.org = providerOrg()

	// Every optional reader is nil → every "external" section
	// collapses to false. The freelance reader returns nil so
	// availability/title/about all evaluate as empty.

	svc := f.build(t)
	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)

	assert.Equal(t, "provider", r.Role)
	assert.Equal(t, "freelance", r.Persona)
	assert.Equal(t, 11, r.TotalSections, "freelance persona has 11 sections (billing/kyc dropped)")
	assert.Equal(t, 0, r.FilledSections)
	assert.Equal(t, 0, r.Percent)

	// Sanity-check the section keys are stable across releases.
	assertHasSectionKey(t, r, "photo")
	assertHasSectionKey(t, r, "title")
	assertHasSectionKey(t, r, "social_links")
	// billing_profile and kyc must NOT appear — they were dropped from
	// the freelance checklist.
	assertMissingSectionKey(t, r, "billing_profile")
	assertMissingSectionKey(t, r, "kyc")
}

func TestCompute_Provider_AllFilled_HundredPercent(t *testing.T) {
	f := newFixture()
	f.users.user = providerUser()
	org := providerOrg()
	f.orgs.org = org

	f.deps.Shared = &fakeSharedReader{ph: "photo.jpg", c: "Paris", cc: "FR", langs: []string{"fr"}}
	f.deps.FreelanceProfile = &fakeFreelanceReader{
		profileID:    uuid.New(),
		title:        "Senior dev",
		about:        "10 years",
		video:        "https://video",
		expertises:   []string{"backend"},
		availability: profile.AvailabilityNow,
	}
	f.deps.Skills = &skillsCounterStub{n: 5}
	f.deps.SocialLinks = &socialLinksCounterStub{freelance: 1}
	f.deps.FreelancePricing = &pricingExistsStub{exists: true}

	svc := f.build(t)
	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)

	assert.Equal(t, 11, r.TotalSections)
	assert.Equal(t, 11, r.FilledSections)
	assert.Equal(t, 100, r.Percent)
}

func TestCompute_Provider_PartialFilled_RoundedFraction(t *testing.T) {
	f := newFixture()
	f.users.user = providerUser()
	f.orgs.org = providerOrg()

	// Fill 4 sections out of 11 → 4*100/11 = 36 (integer division).
	f.deps.Shared = &fakeSharedReader{ph: "photo.jpg"}
	f.deps.FreelanceProfile = &fakeFreelanceReader{
		profileID: uuid.New(),
		title:     "Hi",
		about:     "Yo",
	}
	f.deps.Skills = &skillsCounterStub{n: 1}
	f.deps.SocialLinks = &socialLinksCounterStub{}

	svc := f.build(t)
	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)

	assert.Equal(t, 11, r.TotalSections)
	assert.Equal(t, 4, r.FilledSections)
	assert.Equal(t, 36, r.Percent)
}

// ---------------------------------------------------------------
// Compute — agency role
// ---------------------------------------------------------------

func TestCompute_Agency_EmptyAndFilled(t *testing.T) {
	f := newFixture()
	f.users.user = agencyUser()
	f.orgs.org = agencyOrg()

	svc := f.build(t)
	empty, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)
	assert.Equal(t, "agency", empty.Persona)
	assert.Equal(t, 10, empty.TotalSections, "agency persona has 10 sections (billing/kyc dropped)")
	assert.Equal(t, 0, empty.FilledSections)
	assertMissingSectionKey(t, empty, "billing_profile")
	assertMissingSectionKey(t, empty, "kyc")

	// Now fill every reader.
	org := agencyOrg()
	f.orgs.org = org

	f.deps.Shared = &fakeSharedReader{ph: "p.jpg", c: "Lyon", cc: "FR", langs: []string{"fr"}}
	f.deps.LegacyProfile = &fakeLegacyReader{p: &profile.Profile{
		Title:                 "Agency",
		About:                 "We do",
		AvailabilityStatus:    profile.AvailabilityNow,
		LanguagesProfessional: []string{"fr"},
	}}
	f.deps.Skills = &skillsCounterStub{n: 2}
	f.deps.SocialLinks = &socialLinksCounterStub{agency: 2}
	f.deps.Portfolio = &portfolioCounterStub{n: 3}
	f.deps.LegacyPricing = &legacyPricingCounterStub{n: 1}

	svc = f.build(t)
	full, err := svc.Compute(context.Background(), f.users.user.ID, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, full.TotalSections)
	assert.Equal(t, 10, full.FilledSections)
	assert.Equal(t, 100, full.Percent)
}

// ---------------------------------------------------------------
// Compute — enterprise role
// ---------------------------------------------------------------

func TestCompute_Enterprise_EmptyAndFilled(t *testing.T) {
	f := newFixture()
	f.users.user = enterpriseUser()
	f.orgs.org = enterpriseOrg()

	svc := f.build(t)
	empty, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)
	assert.Equal(t, "enterprise", empty.Persona)
	assert.Equal(t, 2, empty.TotalSections, "enterprise persona has 2 sections (photo + client_about)")
	assert.Equal(t, 0, empty.FilledSections)
	assertMissingSectionKey(t, empty, "billing_profile")
	assertMissingSectionKey(t, empty, "kyc")

	org := enterpriseOrg()
	f.orgs.org = org

	f.deps.Shared = &fakeSharedReader{ph: "p.jpg"}
	f.deps.LegacyProfile = &fakeLegacyReader{p: &profile.Profile{ClientDescription: "We buy"}}

	svc = f.build(t)
	full, err := svc.Compute(context.Background(), f.users.user.ID, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, full.TotalSections)
	assert.Equal(t, 2, full.FilledSections)
	assert.Equal(t, 100, full.Percent)
}

// ---------------------------------------------------------------
// Error propagation
// ---------------------------------------------------------------

func TestCompute_UserReaderError_Surfaces(t *testing.T) {
	f := newFixture()
	f.users.err = errors.New("boom")

	svc := f.build(t)
	_, err := svc.Compute(context.Background(), uuid.New(), uuid.New())
	assert.ErrorContains(t, err, "load user")
}

func TestCompute_OrgReaderError_Surfaces(t *testing.T) {
	f := newFixture()
	f.users.user = providerUser()
	f.orgs.err = errors.New("kaput")

	svc := f.build(t)
	_, err := svc.Compute(context.Background(), uuid.New(), uuid.New())
	assert.ErrorContains(t, err, "load org")
}

// Billing/KYC sections were dropped from the freelance / agency /
// enterprise checklists — completion percent no longer queries the
// billing profile reader. This regression check ensures neither key
// is shipped on the wire so a future re-introduction is an explicit
// product decision, not a silent re-add.
func TestCompute_FreelanceDoesNotShipBillingOrKyc(t *testing.T) {
	f := newFixture()
	f.users.user = providerUser()
	f.orgs.org = providerOrg()

	svc := f.build(t)
	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)
	for _, s := range r.Sections {
		assert.NotEqual(t, "billing_profile", string(s.Key),
			"billing_profile must not be shipped — dropped from checklist")
		assert.NotEqual(t, "kyc", string(s.Key),
			"kyc must not be shipped — dropped from checklist")
	}
}

// TestCompute_AgencyDoesNotShipBillingOrKyc mirrors the freelance
// regression for the agency persona.
func TestCompute_AgencyDoesNotShipBillingOrKyc(t *testing.T) {
	f := newFixture()
	f.users.user = agencyUser()
	f.orgs.org = agencyOrg()

	svc := f.build(t)
	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)
	for _, s := range r.Sections {
		assert.NotEqual(t, "billing_profile", string(s.Key))
		assert.NotEqual(t, "kyc", string(s.Key))
	}
}

// TestCompute_EnterpriseShipsTwoSections asserts the enterprise
// checklist is exactly photo + client_about — billing/kyc were the
// only other items, and dropping them leaves 2 sections.
func TestCompute_EnterpriseShipsTwoSections(t *testing.T) {
	f := newFixture()
	f.users.user = enterpriseUser()
	f.orgs.org = enterpriseOrg()

	svc := f.build(t)
	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, r.TotalSections)
	keys := make(map[string]bool, len(r.Sections))
	for _, s := range r.Sections {
		keys[string(s.Key)] = true
	}
	assert.True(t, keys["photo"], "enterprise must include photo")
	assert.True(t, keys["client_about"], "enterprise must include client_about")
	assert.False(t, keys["billing_profile"])
	assert.False(t, keys["kyc"])
}

// ---------------------------------------------------------------
// Section payload
// ---------------------------------------------------------------

func TestCompute_SectionsCarryLabelKeyAndCompletionPath(t *testing.T) {
	f := newFixture()
	f.users.user = providerUser()
	f.orgs.org = providerOrg()
	svc := f.build(t)

	r, err := svc.Compute(context.Background(), f.users.user.ID, f.orgs.org.ID)
	require.NoError(t, err)
	for _, s := range r.Sections {
		assert.NotEmpty(t, s.LabelKey, "every section must carry a label key")
		assert.Contains(t, s.LabelKey, "profile.completion.section.",
			"label key must be the dotted i18n path")
		assert.NotEmpty(t, s.CompletionPath, "every section must carry a completion path")
	}
}

// ---------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------

func assertHasSectionKey(t *testing.T, r *profilecompletion.Report, key string) {
	t.Helper()
	for _, s := range r.Sections {
		if string(s.Key) == key {
			return
		}
	}
	t.Errorf("missing section key %q", key)
}

// assertMissingSectionKey is the dual of assertHasSectionKey — it
// fails the test when the report carries a key that should not be
// shipped (e.g. dropped billing/kyc sections).
func assertMissingSectionKey(t *testing.T, r *profilecompletion.Report, key string) {
	t.Helper()
	for _, s := range r.Sections {
		if string(s.Key) == key {
			t.Errorf("unexpected section key %q present", key)
			return
		}
	}
}
