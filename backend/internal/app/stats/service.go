// Package stats implements the per-org public stats use cases:
//   - Record(view event) — invoked by the tracking middleware
//   - GetVisibility(org, days) — totals + daily time series
//   - GetKeywords(org, days, limit) — top search keywords
//   - GetEnterpriseApplications(org, days) — applications time series
//
// The service is stateless and depends only on port interfaces. The
// adapters live in adapter/postgres; tests inject mocks through the
// same constructor.
package stats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"

	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/port/repository"
)

// Service exposes the four stats use cases.
type Service struct {
	views        repository.ProfileViewRepository
	keywords     repository.SearchQueryStatsRepository
	applications repository.EnterpriseApplicationsStatsRepository
}

// NewService is the canonical constructor. Each repository is
// optional at construction time — passing nil disables the
// corresponding use case (the caller must check IsXxxAvailable).
// Production wiring always passes non-nil values; tests mock as
// needed.
func NewService(
	views repository.ProfileViewRepository,
	keywords repository.SearchQueryStatsRepository,
	applications repository.EnterpriseApplicationsStatsRepository,
) *Service {
	return &Service{views: views, keywords: keywords, applications: applications}
}

// RecordViewInput is the handler-shaped input for Record. The service
// derives the IPAnonymized + UAHash itself so the handler stays a
// thin DTO adapter.
type RecordViewInput struct {
	OrganizationID uuid.UUID
	Persona        domainstats.Persona
	ViewerUserID   *uuid.UUID
	RawIP          string
	UserAgent      string
	CameFrom       domainstats.CameFrom
	SearchQuery    *string
	SearchPosition *int
	ReferrerURL    *string
}

// Record builds a domain.ViewEvent from the input, validates it, and
// persists it through the repository port. Returns the persisted
// event so the caller can emit telemetry on success.
//
// The function is invoked from a fire-and-forget goroutine in the
// tracking middleware — errors are logged at the call site, never
// surfaced to the end user.
func (s *Service) Record(ctx context.Context, in RecordViewInput) (*domainstats.ViewEvent, error) {
	if s == nil || s.views == nil {
		return nil, fmt.Errorf("stats: profile-view repository not wired")
	}
	event, err := domainstats.NewViewEvent(domainstats.NewViewEventInput{
		OrganizationID:     in.OrganizationID,
		Persona:            in.Persona,
		ViewerUserID:       in.ViewerUserID,
		ViewerIPAnonymized: domainstats.AnonymizeIP(in.RawIP),
		ViewerUAHash:       HashUserAgent(in.UserAgent),
		CameFrom:           in.CameFrom,
		SearchQuery:        in.SearchQuery,
		SearchPosition:     in.SearchPosition,
		ReferrerURL:        in.ReferrerURL,
	})
	if err != nil {
		return nil, fmt.Errorf("stats: build view event: %w", err)
	}
	if err := s.views.Record(ctx, event); err != nil {
		return nil, fmt.Errorf("stats: persist view event: %w", err)
	}
	return event, nil
}

// GetVisibility returns the per-window stats aggregate for the
// requesting org. periodDays must be one of 7/30/90 — anything else
// yields ErrPeriodInvalid.
func (s *Service) GetVisibility(ctx context.Context, orgID uuid.UUID, periodDays int) (*domainstats.Visibility, error) {
	if s == nil || s.views == nil {
		return nil, fmt.Errorf("stats: profile-view repository not wired")
	}
	period, err := domainstats.ParsePeriodDays(periodDays)
	if err != nil {
		return nil, err
	}
	if orgID == uuid.Nil {
		return nil, domainstats.ErrOrgIDRequired
	}
	out, err := s.views.AggregateVisibility(ctx, repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     period,
	})
	if err != nil {
		return nil, fmt.Errorf("stats: aggregate visibility: %w", err)
	}
	if out == nil {
		out = &domainstats.Visibility{
			OrganizationID: orgID.String(),
			PeriodDays:     period,
			Series:         []domainstats.DailyBucket{},
		}
	}
	if out.Series == nil {
		out.Series = []domainstats.DailyBucket{}
	}
	return out, nil
}

// GetKeywords returns the top N keywords for the requesting org.
// limit is clamped to [1..100] via stats.ClampLimit — values outside
// that range are accepted (and clamped) so the handler can pass the
// raw query param without re-validation.
func (s *Service) GetKeywords(ctx context.Context, orgID uuid.UUID, periodDays, limit int) ([]domainstats.KeywordRow, error) {
	if s == nil || s.keywords == nil {
		return nil, fmt.Errorf("stats: search-query repository not wired")
	}
	period, err := domainstats.ParsePeriodDays(periodDays)
	if err != nil {
		return nil, err
	}
	if orgID == uuid.Nil {
		return nil, domainstats.ErrOrgIDRequired
	}
	rows, err := s.keywords.TopKeywordsForOrg(ctx, repository.KeywordFilter{
		OrganizationID: orgID,
		PeriodDays:     period,
		Limit:          domainstats.ClampLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("stats: top keywords: %w", err)
	}
	if rows == nil {
		rows = []domainstats.KeywordRow{}
	}
	return rows, nil
}

// GetEnterpriseApplications returns the applications time series for
// the requesting org.
func (s *Service) GetEnterpriseApplications(ctx context.Context, orgID uuid.UUID, periodDays int) (*domainstats.ApplicationsTimeSeries, error) {
	if s == nil || s.applications == nil {
		return nil, fmt.Errorf("stats: applications repository not wired")
	}
	period, err := domainstats.ParsePeriodDays(periodDays)
	if err != nil {
		return nil, err
	}
	if orgID == uuid.Nil {
		return nil, domainstats.ErrOrgIDRequired
	}
	out, err := s.applications.AggregateApplications(ctx, repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     period,
	})
	if err != nil {
		return nil, fmt.Errorf("stats: aggregate applications: %w", err)
	}
	if out == nil {
		out = &domainstats.ApplicationsTimeSeries{
			OrganizationID: orgID.String(),
			PeriodDays:     period,
			Series:         []domainstats.DailyBucket{},
		}
	}
	if out.Series == nil {
		out.Series = []domainstats.DailyBucket{}
	}
	return out, nil
}

// HashUserAgent returns the hex-encoded SHA-256 of the UA. An empty
// string maps to "" (the domain validator catches that on the way
// in). Exported so the handler-side helper can reuse the same
// canonical hash for unit-test fixtures.
func HashUserAgent(ua string) string {
	t := strings.TrimSpace(ua)
	if t == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}
