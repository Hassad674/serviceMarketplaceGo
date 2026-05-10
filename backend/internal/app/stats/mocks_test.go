package stats_test

import (
	"context"

	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/port/repository"
)

// fakeProfileViewRepo is the in-line mock for repository.ProfileViewRepository
// used by service_test.go. Each method holds a function field so
// tests can inject closures inline (no codegen).
type fakeProfileViewRepo struct {
	RecordFn              func(ctx context.Context, event *domainstats.ViewEvent) error
	AggregateVisibilityFn func(ctx context.Context, f repository.VisibilityFilter) (*domainstats.Visibility, error)
}

func (f *fakeProfileViewRepo) Record(ctx context.Context, e *domainstats.ViewEvent) error {
	if f.RecordFn != nil {
		return f.RecordFn(ctx, e)
	}
	return nil
}

func (f *fakeProfileViewRepo) AggregateVisibility(ctx context.Context, filter repository.VisibilityFilter) (*domainstats.Visibility, error) {
	if f.AggregateVisibilityFn != nil {
		return f.AggregateVisibilityFn(ctx, filter)
	}
	return nil, nil
}

type fakeKeywordsRepo struct {
	TopKeywordsForOrgFn func(ctx context.Context, f repository.KeywordFilter) ([]domainstats.KeywordRow, error)
}

func (f *fakeKeywordsRepo) TopKeywordsForOrg(ctx context.Context, filter repository.KeywordFilter) ([]domainstats.KeywordRow, error) {
	if f.TopKeywordsForOrgFn != nil {
		return f.TopKeywordsForOrgFn(ctx, filter)
	}
	return nil, nil
}

type fakeApplicationsRepo struct {
	AggregateApplicationsFn func(ctx context.Context, f repository.VisibilityFilter) (*domainstats.ApplicationsTimeSeries, error)
}

func (f *fakeApplicationsRepo) AggregateApplications(ctx context.Context, filter repository.VisibilityFilter) (*domainstats.ApplicationsTimeSeries, error) {
	if f.AggregateApplicationsFn != nil {
		return f.AggregateApplicationsFn(ctx, filter)
	}
	return nil, nil
}
