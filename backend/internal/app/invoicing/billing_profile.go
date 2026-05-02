package invoicing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// BillingProfileSnapshot is the value the handler renders. The service
// always returns the canonical pair (profile, missing_fields, is_complete)
// so the handler is a flat marshal — no extra computation.
type BillingProfileSnapshot struct {
	Profile       invoicing.BillingProfile
	MissingFields []invoicing.MissingField
	IsComplete    bool
}

// UpdateBillingProfileInput is the partial-update payload from the PUT
// endpoint. Empty strings are accepted (the user can deliberately clear
// optional fields). Completeness is checked at the gate, never on save.
type UpdateBillingProfileInput struct {
	ProfileType    string
	LegalName      string
	TradingName    string
	LegalForm      string
	TaxID          string
	VATNumber      string
	AddressLine1   string
	AddressLine2   string
	PostalCode     string
	City           string
	Country        string
	InvoicingEmail string
}

// VIESValidationSnapshot mirrors the slim shape the handler sends back
// after a /validate-vat call. We omit the raw VIES payload because the
// frontend has no use for the JSON blob — it stays on the row for legal
// proof.
type VIESValidationSnapshot struct {
	Valid          bool
	RegisteredName string
	CheckedAt      time.Time
}

// billingProfileOrgs is the local composite the invoicing flows need:
// FindByID (Reader) to load the org row + GetStripeAccount (StripeStore)
// to read its connected-account id. No segregated child covers both, so
// we compose locally — narrower than the wide port and explicit about
// the actual surface in use.
type billingProfileOrgs interface {
	repository.OrganizationReader
	repository.OrganizationStripeStore
}

// BillingProfileDeps groups optional dependencies for the Phase 6
// methods. Phase 4/5 callers can keep using the original NewService
// constructor and skip these — the methods that need the dep return a
// configured-with error when it's missing. This keeps Phase 4/5 wiring
// untouched and makes the new feature additive.
//
// Users + Organizations also drive the invoicing-email default: when a
// billing profile has no explicit invoicing_email, the read paths
// resolve the org's owner user and back-fill the snapshot with that
// account's email. Both deps are optional — when missing, the
// snapshot's invoicing_email simply stays empty (legacy behaviour).
type BillingProfileDeps struct {
	Organizations billingProfileOrgs
	Users         repository.UserReader
	StripeKYC     service.StripeKYCSnapshotReader
	VIESValidator service.VIESValidator
}

// SetBillingProfileDeps wires the optional dependencies the Phase 6
// flows need. Called from main.go right after NewService so the existing
// constructor signature stays stable. Safe to call multiple times — the
// last call wins. Passing a zero struct disables the corresponding
// methods.
func (s *Service) SetBillingProfileDeps(deps BillingProfileDeps) {
	s.organizations = deps.Organizations
	s.users = deps.Users
	s.stripeKYC = deps.StripeKYC
	s.vies = deps.VIESValidator
}

// ErrBillingProfileNoVAT is returned by ValidateBillingProfileVAT when
// the profile has no VAT number to validate. The handler maps it to a
// 400 with code "vat_number_required".
var ErrBillingProfileNoVAT = errors.New("invoicing: billing profile has no VAT number to validate")

// ErrBillingProfileFeatureDisabled is returned when the caller invokes a
// Phase 6 method that requires a dependency the orchestrator never
// wired (e.g. the Stripe sync without a StripeKYCSnapshotReader).
var ErrBillingProfileFeatureDisabled = errors.New("invoicing: billing profile feature partially disabled — missing dependency")

// GetBillingProfile returns the org's billing profile + completeness
// snapshot. Returns an empty stub when the org has no row yet — the
// frontend can render the empty form without a 404 dance.
//
// invoicing_email is back-filled with the org owner's account email
// when the row's value is empty, so the invoice generator and the
// frontend always see a usable address. The persisted column stays
// empty until the user explicitly overrides it.
func (s *Service) GetBillingProfile(ctx context.Context, organizationID uuid.UUID) (BillingProfileSnapshot, error) {
	if organizationID == uuid.Nil {
		return BillingProfileSnapshot{}, fmt.Errorf("invoicing: organization id required")
	}

	profile, err := s.profiles.FindByOrganization(ctx, organizationID)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			// Empty stub. The completeness check correctly returns the
			// universal "missing everything" list so the form renders
			// the right prompts on first visit.
			stub := invoicing.BillingProfile{
				OrganizationID: organizationID,
				ProfileType:    invoicing.ProfileBusiness,
			}
			s.fillDefaultInvoicingEmail(ctx, &stub)
			return snapshotOf(stub), nil
		}
		return BillingProfileSnapshot{}, fmt.Errorf("get billing profile: %w", err)
	}
	resolved := *profile
	s.fillDefaultInvoicingEmail(ctx, &resolved)
	return snapshotOf(resolved), nil
}

// fillDefaultInvoicingEmail back-fills profile.InvoicingEmail with the
// org owner's account email when the row's value is empty. Mutates the
// passed profile in place. Best-effort — if the org or owner cannot be
// resolved (deps not wired, lookup error, etc.) the field stays empty
// and the caller proceeds; this method must NEVER fail a read flow.
//
// We default to the org's *owner* user (FindByOwnerUserID-style row)
// because the owner is the legal/economic principal of the org —
// matches the "organizations own state" rule in CLAUDE.md, where
// member emails are not authoritative for the org-level invoicing
// contact.
func (s *Service) fillDefaultInvoicingEmail(ctx context.Context, p *invoicing.BillingProfile) {
	if p == nil {
		return
	}
	if strings.TrimSpace(p.InvoicingEmail) != "" {
		return
	}
	if s.organizations == nil || s.users == nil {
		return
	}
	if p.OrganizationID == uuid.Nil {
		return
	}
	org, err := s.organizations.FindByID(ctx, p.OrganizationID)
	if err != nil || org == nil {
		return
	}
	if org.OwnerUserID == uuid.Nil {
		return
	}
	owner, err := s.users.GetByID(ctx, org.OwnerUserID)
	if err != nil || owner == nil {
		return
	}
	if email := strings.TrimSpace(owner.Email); email != "" {
		p.InvoicingEmail = email
	}
}

// UpdateBillingProfile upserts the profile. We accept partial saves —
// completeness is enforced at the gate, never on save. When the user
// changes vat_number, we clear vat_validated_at so the next save
// re-requires VIES validation.
func (s *Service) UpdateBillingProfile(ctx context.Context, organizationID uuid.UUID, in UpdateBillingProfileInput) (BillingProfileSnapshot, error) {
	if organizationID == uuid.Nil {
		return BillingProfileSnapshot{}, fmt.Errorf("invoicing: organization id required")
	}

	// Load existing or seed empty.
	existing, err := s.profiles.FindByOrganization(ctx, organizationID)
	now := time.Now().UTC()
	var profile invoicing.BillingProfile
	switch {
	case err == nil:
		profile = *existing
	case errors.Is(err, invoicing.ErrNotFound):
		profile = invoicing.BillingProfile{
			OrganizationID: organizationID,
			ProfileType:    invoicing.ProfileBusiness,
			CreatedAt:      now,
		}
	default:
		return BillingProfileSnapshot{}, fmt.Errorf("update billing profile: %w", err)
	}

	// Coerce and apply. Trim every text field so accidental whitespace
	// never silently passes the completeness check.
	if pt := strings.TrimSpace(in.ProfileType); pt != "" {
		profile.ProfileType = invoicing.ProfileType(pt)
	}
	if !profile.ProfileType.IsValid() {
		profile.ProfileType = invoicing.ProfileBusiness
	}

	// Detect VAT number change BEFORE we overwrite — clearing the
	// validation timestamp makes the user re-validate on next save.
	newVAT := strings.TrimSpace(in.VATNumber)
	if newVAT != strings.TrimSpace(profile.VATNumber) {
		profile.VATValidatedAt = nil
		profile.VATValidationPayload = nil
	}

	profile.LegalName = strings.TrimSpace(in.LegalName)
	profile.TradingName = strings.TrimSpace(in.TradingName)
	profile.LegalForm = strings.TrimSpace(in.LegalForm)
	profile.TaxID = strings.TrimSpace(in.TaxID)
	profile.VATNumber = newVAT
	profile.AddressLine1 = strings.TrimSpace(in.AddressLine1)
	profile.AddressLine2 = strings.TrimSpace(in.AddressLine2)
	profile.PostalCode = strings.TrimSpace(in.PostalCode)
	profile.City = strings.TrimSpace(in.City)
	profile.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	profile.InvoicingEmail = strings.TrimSpace(in.InvoicingEmail)
	profile.UpdatedAt = now

	if err := s.profiles.Upsert(ctx, &profile); err != nil {
		return BillingProfileSnapshot{}, fmt.Errorf("update billing profile: persist: %w", err)
	}
	return snapshotOf(profile), nil
}

// SyncBillingProfileFromStripeKYC fills empty fields from the org's
// Stripe Connect account. Never overwrites user-edited values. Sets
// synced_from_kyc_at when at least one field was filled.
func (s *Service) SyncBillingProfileFromStripeKYC(ctx context.Context, organizationID uuid.UUID) (BillingProfileSnapshot, error) {
	if organizationID == uuid.Nil {
		return BillingProfileSnapshot{}, fmt.Errorf("invoicing: organization id required")
	}
	if s.organizations == nil || s.stripeKYC == nil {
		return BillingProfileSnapshot{}, ErrBillingProfileFeatureDisabled
	}

	// Resolve Stripe account id from the org. GetStripeAccount returns
	// (id, country, err) — we only need id at this layer.
	stripeAccountID, _, err := s.organizations.GetStripeAccount(ctx, organizationID)
	if err != nil {
		return BillingProfileSnapshot{}, fmt.Errorf("sync billing profile: load org stripe account: %w", err)
	}
	if strings.TrimSpace(stripeAccountID) == "" {
		// No Stripe Connect account yet (fresh provider, KYC not
		// started). Returning an error here would crash the embed
		// flow — the page calls this on mount to pre-fill the form
		// and a 500 surfaces as a generic load error in the WebView.
		// Instead, return whatever profile snapshot we already have
		// (or an empty one for first-time users) so the form renders
		// correctly and the user fills it manually.
		existing, ferr := s.profiles.FindByOrganization(ctx, organizationID)
		if ferr == nil {
			return snapshotOf(*existing), nil
		}
		if errors.Is(ferr, invoicing.ErrNotFound) {
			return snapshotOf(invoicing.BillingProfile{
				OrganizationID: organizationID,
				ProfileType:    invoicing.ProfileBusiness,
			}), nil
		}
		return BillingProfileSnapshot{}, fmt.Errorf("sync billing profile: load existing: %w", ferr)
	}

	snap, err := s.stripeKYC.GetAccountKYCSnapshot(ctx, stripeAccountID)
	if err != nil {
		return BillingProfileSnapshot{}, fmt.Errorf("sync billing profile: stripe kyc fetch: %w", err)
	}

	// Load existing or seed empty.
	existing, ferr := s.profiles.FindByOrganization(ctx, organizationID)
	now := time.Now().UTC()
	var profile invoicing.BillingProfile
	switch {
	case ferr == nil:
		profile = *existing
	case errors.Is(ferr, invoicing.ErrNotFound):
		profile = invoicing.BillingProfile{
			OrganizationID: organizationID,
			ProfileType:    invoicing.ProfileBusiness,
			CreatedAt:      now,
		}
	default:
		return BillingProfileSnapshot{}, fmt.Errorf("sync billing profile: load: %w", ferr)
	}

	// Merge: fill empty fields only.
	filled := false
	fillIfEmpty := func(target *string, src string) {
		if strings.TrimSpace(*target) == "" && strings.TrimSpace(src) != "" {
			*target = strings.TrimSpace(src)
			filled = true
		}
	}
	// profile_type only flips to a strong default when it would otherwise
	// be empty / "business" placeholder AND Stripe has a clear answer.
	if snap.BusinessType == "individual" && profile.ProfileType == invoicing.ProfileBusiness && profile.LegalName == "" {
		profile.ProfileType = invoicing.ProfileIndividual
		filled = true
	}
	fillIfEmpty(&profile.LegalName, snap.LegalName)
	fillIfEmpty(&profile.AddressLine1, snap.AddressLine1)
	fillIfEmpty(&profile.AddressLine2, snap.AddressLine2)
	fillIfEmpty(&profile.City, snap.City)
	fillIfEmpty(&profile.PostalCode, snap.PostalCode)
	fillIfEmpty(&profile.TaxID, snap.TaxID)
	fillIfEmpty(&profile.InvoicingEmail, snap.SupportEmail)
	if strings.TrimSpace(profile.Country) == "" && strings.TrimSpace(snap.Country) != "" {
		profile.Country = strings.ToUpper(strings.TrimSpace(snap.Country))
		filled = true
	}

	if filled {
		stamp := now
		profile.SyncedFromKYCAt = &stamp
	}
	profile.UpdatedAt = now

	if err := s.profiles.Upsert(ctx, &profile); err != nil {
		return BillingProfileSnapshot{}, fmt.Errorf("sync billing profile: persist: %w", err)
	}
	return snapshotOf(profile), nil
}

// ValidateBillingProfileVAT calls VIES on the profile's vat_number,
// stores the result, and returns the slim snapshot.
func (s *Service) ValidateBillingProfileVAT(ctx context.Context, organizationID uuid.UUID) (VIESValidationSnapshot, error) {
	if organizationID == uuid.Nil {
		return VIESValidationSnapshot{}, fmt.Errorf("invoicing: organization id required")
	}
	if s.vies == nil {
		return VIESValidationSnapshot{}, ErrBillingProfileFeatureDisabled
	}

	profile, err := s.profiles.FindByOrganization(ctx, organizationID)
	if err != nil {
		return VIESValidationSnapshot{}, fmt.Errorf("validate vat: load profile: %w", err)
	}
	vat := strings.TrimSpace(profile.VATNumber)
	if vat == "" {
		return VIESValidationSnapshot{}, ErrBillingProfileNoVAT
	}
	country := strings.ToUpper(strings.TrimSpace(profile.Country))
	if country == "" {
		// VIES needs a country code. Best-effort: derive from the VAT
		// number's first two letters when the user hasn't filled the
		// country field yet.
		if len(vat) >= 2 {
			country = strings.ToUpper(vat[:2])
		}
	}

	res, err := s.vies.Validate(ctx, country, vat)
	if err != nil {
		return VIESValidationSnapshot{}, fmt.Errorf("validate vat: vies: %w", err)
	}

	now := time.Now().UTC()
	if res.Valid {
		profile.VATValidatedAt = &now
		profile.VATValidationPayload = res.RawPayload
	} else {
		// Negative result — clear the timestamp so the gate keeps blocking.
		profile.VATValidatedAt = nil
		profile.VATValidationPayload = res.RawPayload
	}
	profile.UpdatedAt = now
	if err := s.profiles.Upsert(ctx, profile); err != nil {
		return VIESValidationSnapshot{}, fmt.Errorf("validate vat: persist: %w", err)
	}

	return VIESValidationSnapshot{
		Valid:          res.Valid,
		RegisteredName: res.RegisteredName,
		CheckedAt:      now,
	}, nil
}

// GetBillingProfileSnapshotForStripe returns the slim projection of the
// org's billing profile that the subscription app pushes onto the
// Stripe Customer record before creating an Embedded Checkout session.
// Implements port/service.BillingProfileSnapshotReader so subscription
// can stay decoupled from the invoicing module.
//
// Returns a zero-value snapshot (every field empty) when the org has no
// billing_profile row yet — the caller decides whether to skip the
// Stripe Customer.Update in that case (see snap.IsEmpty()).
func (s *Service) GetBillingProfileSnapshotForStripe(ctx context.Context, organizationID uuid.UUID) (service.BillingProfileStripeSnapshot, error) {
	if organizationID == uuid.Nil {
		return service.BillingProfileStripeSnapshot{}, fmt.Errorf("invoicing: organization id required")
	}
	profile, err := s.profiles.FindByOrganization(ctx, organizationID)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			return service.BillingProfileStripeSnapshot{}, nil
		}
		return service.BillingProfileStripeSnapshot{}, fmt.Errorf("get billing snapshot for stripe: %w", err)
	}
	resolved := *profile
	s.fillDefaultInvoicingEmail(ctx, &resolved)
	return service.BillingProfileStripeSnapshot{
		LegalName:      resolved.LegalName,
		AddressLine1:   resolved.AddressLine1,
		AddressLine2:   resolved.AddressLine2,
		PostalCode:     resolved.PostalCode,
		City:           resolved.City,
		Country:        resolved.Country,
		InvoicingEmail: resolved.InvoicingEmail,
		VATNumber:      resolved.VATNumber,
	}, nil
}

// IsBillingProfileComplete is the read-only gate the wallet/subscribe
// handlers call before a payout/subscribe. Returns (true, nil) when the
// profile passes domain.CheckCompleteness; otherwise the missing fields
// list is bubbled up so the handler can include it in the 403 payload.
//
// First-time callers (no row yet) ALWAYS yield "incomplete" with the
// universal missing list — they have not seeded anything yet.
func (s *Service) IsBillingProfileComplete(ctx context.Context, organizationID uuid.UUID) (bool, []invoicing.MissingField, error) {
	if organizationID == uuid.Nil {
		return false, nil, fmt.Errorf("invoicing: organization id required")
	}
	profile, err := s.profiles.FindByOrganization(ctx, organizationID)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			stub := invoicing.BillingProfile{
				OrganizationID: organizationID,
				ProfileType:    invoicing.ProfileBusiness,
			}
			missing := invoicing.CheckCompleteness(stub)
			return false, missing, nil
		}
		return false, nil, fmt.Errorf("billing profile completeness probe: %w", err)
	}
	missing := invoicing.CheckCompleteness(*profile)
	return len(missing) == 0, missing, nil
}

// snapshotOf packages a profile + its computed completeness into the
// canonical handler payload.
func snapshotOf(p invoicing.BillingProfile) BillingProfileSnapshot {
	missing := invoicing.CheckCompleteness(p)
	return BillingProfileSnapshot{
		Profile:       p,
		MissingFields: missing,
		IsComplete:    len(missing) == 0,
	}
}
