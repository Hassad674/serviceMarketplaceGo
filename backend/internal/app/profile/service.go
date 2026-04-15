package profileapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/internal/search"
)

// SearchIndexPublisher is the narrow port the legacy agency profile
// service uses to trigger a Typesense reindex after a profile
// mutation. Optional — a nil publisher is treated as a no-op so the
// search engine can be disabled without breaking the legacy profile
// flow. Defined locally to avoid cross-feature imports.
type SearchIndexPublisher interface {
	PublishReindex(ctx context.Context, orgID uuid.UUID, persona search.Persona) error
}

// Service is the application layer for the organization profile.
// It orchestrates the repository port and — for the Tier 1
// location block — an optional Geocoder dependency that turns
// (city, country) into decimal coordinates without blocking the
// save flow on failure.
//
// Every Tier 1 write method (UpdateLocation / UpdateLanguages /
// UpdateAvailability) is independent: failing one does not roll
// back the others, and a client can edit the blocks in isolation
// via the dedicated endpoints.
type Service struct {
	profiles repository.ProfileRepository

	// geocoder is optional. When nil, UpdateLocation persists the
	// location block without coordinates (latitude/longitude stay
	// NULL). This keeps the service usable in tests and in
	// development environments that do not run with an HTTP
	// geocoder available.
	geocoder service.Geocoder

	// searchIndex is the optional Typesense reindex publisher. Nil
	// in tests + when the search engine is disabled.
	searchIndex SearchIndexPublisher
}

// NewService wires the profile service with its mandatory
// dependency. Use WithGeocoder to attach an optional geocoder
// without breaking existing call sites that predate the Tier 1
// completion.
func NewService(profiles repository.ProfileRepository) *Service {
	return &Service{profiles: profiles}
}

// WithGeocoder sets (or replaces) the optional geocoder dependency
// and returns the same service for fluent wiring in main.go.
// Passing nil is a no-op so a default-then-override pattern stays
// safe. See handler.ProfileHandler.WithSkillsReader for the same
// idiom applied to a different optional collaborator.
func (s *Service) WithGeocoder(g service.Geocoder) *Service {
	if g != nil {
		s.geocoder = g
	}
	return s
}

// WithSearchIndexPublisher attaches a Typesense reindex publisher.
// Returns the same service for fluent wiring in main.go. Passing
// nil is allowed and disables publishing.
func (s *Service) WithSearchIndexPublisher(p SearchIndexPublisher) *Service {
	s.searchIndex = p
	return s
}

// publishReindex is the best-effort wrapper. Logged but never
// returned so a degraded search engine cannot block a profile
// update.
func (s *Service) publishReindex(ctx context.Context, orgID uuid.UUID) {
	if s.searchIndex == nil {
		return
	}
	if err := s.searchIndex.PublishReindex(ctx, orgID, search.PersonaAgency); err != nil {
		slog.Warn("legacy profile: search reindex publish failed",
			"org_id", orgID, "error", err)
	}
}

func (s *Service) SearchPublic(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error) {
	results, nextCursor, err := s.profiles.SearchPublic(ctx, orgTypeFilter, referrerOnly, cursor, limit)
	if err != nil {
		return nil, "", fmt.Errorf("search public profiles: %w", err)
	}
	return results, nextCursor, nil
}

func (s *Service) GetProfile(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	p, err := s.profiles.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return p, nil
}

type UpdateProfileInput struct {
	Title                string
	About                string
	PhotoURL             string
	PresentationVideoURL string
	ReferrerAbout        string
	ReferrerVideoURL     string
}

func (s *Service) UpdateProfile(ctx context.Context, orgID uuid.UUID, input UpdateProfileInput) (*profile.Profile, error) {
	p, err := s.profiles.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	applyUpdates(p, input)

	if err := s.profiles.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	s.publishReindex(ctx, orgID)
	return p, nil
}

func applyUpdates(p *profile.Profile, input UpdateProfileInput) {
	if input.Title != "" {
		p.Title = input.Title
	}
	if input.About != "" {
		p.About = input.About
	}
	if input.PhotoURL != "" {
		p.PhotoURL = input.PhotoURL
	}
	if input.PresentationVideoURL != "" {
		p.PresentationVideoURL = input.PresentationVideoURL
	}
	if input.ReferrerAbout != "" {
		p.ReferrerAbout = input.ReferrerAbout
	}
	if input.ReferrerVideoURL != "" {
		p.ReferrerVideoURL = input.ReferrerVideoURL
	}
}

// ---------------------------------------------------------------
// Tier 1 completion use cases (migration 083)
// ---------------------------------------------------------------

// UpdateLocationInput groups the user-facing inputs for the
// location block so the service signature stays under the 4-param
// budget. When the caller already has canonical coordinates (e.g.
// from the web / mobile city autocomplete powered by BAN + Photon),
// it MAY pass non-nil Latitude + Longitude and the service will
// trust them verbatim — skipping the server-side geocoder entirely.
// That saves a 2s bounded round-trip on every save and removes a
// needless external dependency from the happy path. Legacy or
// programmatic callers that omit the coordinates still fall back to
// the optional server-side Geocoder.
type UpdateLocationInput struct {
	City           string
	CountryCode    string
	Latitude       *float64
	Longitude      *float64
	WorkMode       []string
	TravelRadiusKm *int
}

// UpdateLocation persists the org's location block. The method:
//
//  1. Normalizes city (trim), country code (upper + trim), and
//     work modes (dedup + filter).
//  2. Validates the country code via the domain helper.
//  3. If the caller supplied both lat and lng (non-nil), trusts
//     them as-is — the client-side autocomplete already resolved a
//     canonical municipality + coordinates. Otherwise attempts a
//     best-effort server-side geocode via the injected Geocoder
//     with a 2s bounded sub-context; any failure is logged at WARN
//     and the save proceeds without coordinates.
//  4. Delegates to ProfileRepository.UpdateLocation which
//     rewrites the entire location block atomically.
//
// Empty city + empty country code is a valid payload that clears
// the location block (NULL lat/lng, empty arrays). The UI uses
// this path when a user deletes their location.
func (s *Service) UpdateLocation(ctx context.Context, orgID uuid.UUID, input UpdateLocationInput) error {
	city := strings.TrimSpace(input.City)
	country := strings.ToUpper(strings.TrimSpace(input.CountryCode))
	if err := profile.ValidateCountryCode(country); err != nil {
		return fmt.Errorf("update location: %w", err)
	}
	workMode := profile.NormalizeWorkModes(input.WorkMode)

	lat, lng := s.resolveCoordinates(ctx, orgID, city, country, input.Latitude, input.Longitude)

	if err := s.profiles.UpdateLocation(ctx, orgID, repository.LocationInput{
		City:           city,
		CountryCode:    country,
		Latitude:       lat,
		Longitude:      lng,
		WorkMode:       workMode,
		TravelRadiusKm: input.TravelRadiusKm,
	}); err != nil {
		return fmt.Errorf("update location: persist: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return nil
}

// resolveCoordinates trusts client-supplied lat/lng when both are
// non-nil — that is the fast path used by the web/mobile autocomplete.
// When either is nil it falls back to the server-side geocoder so
// admin tooling and programmatic writes keep working without having
// to embed a geocoding client of their own.
func (s *Service) resolveCoordinates(
	ctx context.Context,
	orgID uuid.UUID,
	city, country string,
	clientLat, clientLng *float64,
) (*float64, *float64) {
	if clientLat != nil && clientLng != nil {
		return clientLat, clientLng
	}
	return s.tryGeocode(ctx, orgID, city, country)
}

// tryGeocode attempts a best-effort geocoding call. Returns nil
// pointers when the geocoder is unavailable, when the inputs do
// not allow a meaningful lookup (empty city or country), or when
// the provider fails. Extracted so UpdateLocation stays under the
// 50-line cap and so the fallback path can be tested in isolation.
func (s *Service) tryGeocode(ctx context.Context, orgID uuid.UUID, city, country string) (*float64, *float64) {
	if s.geocoder == nil || city == "" || country == "" {
		return nil, nil
	}
	gctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	la, ln, err := s.geocoder.Geocode(gctx, city, country)
	if err != nil {
		slog.Warn("geocoding failed for profile location",
			"org_id", orgID.String(),
			"city", city,
			"country", country,
			"error", err)
		return nil, nil
	}
	return &la, &ln
}

// UpdateLanguages replaces the two language arrays atomically.
// The inputs are normalized + deduped via the domain helpers so
// the repository writes a canonical shape.
func (s *Service) UpdateLanguages(ctx context.Context, orgID uuid.UUID, professional, conversational []string) error {
	pro := profile.NormalizeLanguageCodes(professional)
	conv := profile.NormalizeLanguageCodes(conversational)
	if err := s.profiles.UpdateLanguages(ctx, orgID, pro, conv); err != nil {
		return fmt.Errorf("update languages: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return nil
}

// UpdateAvailability patches the direct and/or referrer availability
// slots. Each slot is independent: passing nil means "leave this
// column untouched", passing a non-nil value validates and writes
// it. Callers are expected to supply at least one non-nil pointer.
// This split lets the freelance profile page and the referrer
// profile page mutate their own column without clobbering the other.
func (s *Service) UpdateAvailability(ctx context.Context, orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) error {
	if direct == nil && referrer == nil {
		return profile.ErrInvalidAvailabilityStatus
	}
	if direct != nil && !direct.IsValid() {
		return profile.ErrInvalidAvailabilityStatus
	}
	if referrer != nil && !referrer.IsValid() {
		return profile.ErrInvalidAvailabilityStatus
	}
	if err := s.profiles.UpdateAvailability(ctx, orgID, direct, referrer); err != nil {
		return fmt.Errorf("update availability: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return nil
}
