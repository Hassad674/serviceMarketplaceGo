package stats_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	appstats "marketplace-backend/internal/app/stats"
	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/port/repository"
)

func newSvc(views *fakeProfileViewRepo, kw *fakeKeywordsRepo, app *fakeApplicationsRepo) *appstats.Service {
	return appstats.NewService(views, kw, app)
}

func TestService_Record_HappyPath(t *testing.T) {
	t.Parallel()
	captured := []*domainstats.ViewEvent{}
	views := &fakeProfileViewRepo{RecordFn: func(_ context.Context, e *domainstats.ViewEvent) error {
		captured = append(captured, e)
		return nil
	}}
	svc := newSvc(views, nil, nil)

	q := "go developer"
	pos := 4
	out, err := svc.Record(context.Background(), appstats.RecordViewInput{
		OrganizationID: uuid.New(),
		Persona:        domainstats.PersonaFreelance,
		RawIP:          "203.0.113.42",
		UserAgent:      "Mozilla/5.0",
		CameFrom:       domainstats.CameFromSearch,
		SearchQuery:    &q,
		SearchPosition: &pos,
	})
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Len(t, captured, 1)
	assert.Equal(t, "203.0.113.0/24", captured[0].ViewerIPAnonymized)
	assert.Equal(t, appstats.HashUserAgent("Mozilla/5.0"), captured[0].ViewerUAHash)
}

func TestService_Record_DomainValidation(t *testing.T) {
	t.Parallel()
	svc := newSvc(&fakeProfileViewRepo{}, nil, nil)
	_, err := svc.Record(context.Background(), appstats.RecordViewInput{
		// missing org id triggers ErrOrgIDRequired
		Persona:   domainstats.PersonaAgency,
		RawIP:     "10.0.0.1",
		UserAgent: "x",
		CameFrom:  domainstats.CameFromDirect,
	})
	assert.ErrorIs(t, err, domainstats.ErrOrgIDRequired)
}

func TestService_Record_RepoError(t *testing.T) {
	t.Parallel()
	views := &fakeProfileViewRepo{RecordFn: func(context.Context, *domainstats.ViewEvent) error {
		return errors.New("db down")
	}}
	svc := newSvc(views, nil, nil)
	_, err := svc.Record(context.Background(), appstats.RecordViewInput{
		OrganizationID: uuid.New(),
		Persona:        domainstats.PersonaFreelance,
		RawIP:          "10.0.0.1",
		UserAgent:      "x",
		CameFrom:       domainstats.CameFromDirect,
	})
	assert.ErrorContains(t, err, "stats: persist view event")
}

func TestService_Record_NotWired(t *testing.T) {
	t.Parallel()
	svc := appstats.NewService(nil, nil, nil)
	_, err := svc.Record(context.Background(), appstats.RecordViewInput{
		OrganizationID: uuid.New(),
		Persona:        domainstats.PersonaFreelance,
		RawIP:          "10.0.0.1",
		UserAgent:      "x",
		CameFrom:       domainstats.CameFromDirect,
	})
	assert.ErrorContains(t, err, "not wired")
}

func TestService_GetVisibility_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	views := &fakeProfileViewRepo{AggregateVisibilityFn: func(_ context.Context, f repository.VisibilityFilter) (*domainstats.Visibility, error) {
		assert.Equal(t, orgID, f.OrganizationID)
		assert.Equal(t, domainstats.Period30Days, f.PeriodDays)
		return &domainstats.Visibility{TotalViews: 42, UniqueViewers: 12}, nil
	}}
	svc := newSvc(views, nil, nil)
	got, err := svc.GetVisibility(context.Background(), orgID, 30)
	assert.NoError(t, err)
	assert.Equal(t, 42, got.TotalViews)
	assert.NotNil(t, got.Series) // populated even when nil from repo
}

func TestService_GetVisibility_InvalidPeriod(t *testing.T) {
	t.Parallel()
	svc := newSvc(&fakeProfileViewRepo{}, nil, nil)
	_, err := svc.GetVisibility(context.Background(), uuid.New(), 31)
	assert.ErrorIs(t, err, domainstats.ErrPeriodInvalid)
}

func TestService_GetVisibility_NilOrg(t *testing.T) {
	t.Parallel()
	svc := newSvc(&fakeProfileViewRepo{}, nil, nil)
	_, err := svc.GetVisibility(context.Background(), uuid.Nil, 30)
	assert.ErrorIs(t, err, domainstats.ErrOrgIDRequired)
}

func TestService_GetVisibility_RepoNilReturn(t *testing.T) {
	t.Parallel()
	views := &fakeProfileViewRepo{AggregateVisibilityFn: func(context.Context, repository.VisibilityFilter) (*domainstats.Visibility, error) {
		return nil, nil
	}}
	svc := newSvc(views, nil, nil)
	got, err := svc.GetVisibility(context.Background(), uuid.New(), 7)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, []domainstats.DailyBucket{}, got.Series)
}

func TestService_GetKeywords_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	kw := &fakeKeywordsRepo{TopKeywordsForOrgFn: func(_ context.Context, f repository.KeywordFilter) ([]domainstats.KeywordRow, error) {
		assert.Equal(t, orgID, f.OrganizationID)
		assert.Equal(t, 10, f.Limit)
		return []domainstats.KeywordRow{{Keyword: "go", Count: 5, AvgPosition: 1.2}}, nil
	}}
	svc := newSvc(nil, kw, nil)
	got, err := svc.GetKeywords(context.Background(), orgID, 30, 0)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "go", got[0].Keyword)
}

func TestService_GetKeywords_NilRowsBecomesEmptySlice(t *testing.T) {
	t.Parallel()
	kw := &fakeKeywordsRepo{TopKeywordsForOrgFn: func(context.Context, repository.KeywordFilter) ([]domainstats.KeywordRow, error) {
		return nil, nil
	}}
	svc := newSvc(nil, kw, nil)
	got, err := svc.GetKeywords(context.Background(), uuid.New(), 30, 5)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestService_GetKeywords_LimitClamped(t *testing.T) {
	t.Parallel()
	captured := 0
	kw := &fakeKeywordsRepo{TopKeywordsForOrgFn: func(_ context.Context, f repository.KeywordFilter) ([]domainstats.KeywordRow, error) {
		captured = f.Limit
		return nil, nil
	}}
	svc := newSvc(nil, kw, nil)
	_, err := svc.GetKeywords(context.Background(), uuid.New(), 30, 9999)
	assert.NoError(t, err)
	assert.Equal(t, 100, captured)
}

func TestService_GetKeywords_RepoError(t *testing.T) {
	t.Parallel()
	kw := &fakeKeywordsRepo{TopKeywordsForOrgFn: func(context.Context, repository.KeywordFilter) ([]domainstats.KeywordRow, error) {
		return nil, errors.New("db down")
	}}
	svc := newSvc(nil, kw, nil)
	_, err := svc.GetKeywords(context.Background(), uuid.New(), 30, 10)
	assert.ErrorContains(t, err, "stats: top keywords")
}

func TestService_GetEnterpriseApplications_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	app := &fakeApplicationsRepo{AggregateApplicationsFn: func(_ context.Context, f repository.VisibilityFilter) (*domainstats.ApplicationsTimeSeries, error) {
		assert.Equal(t, orgID, f.OrganizationID)
		return &domainstats.ApplicationsTimeSeries{TotalCount: 7}, nil
	}}
	svc := newSvc(nil, nil, app)
	got, err := svc.GetEnterpriseApplications(context.Background(), orgID, 30)
	assert.NoError(t, err)
	assert.Equal(t, 7, got.TotalCount)
	assert.NotNil(t, got.Series)
}

func TestService_GetEnterpriseApplications_InvalidPeriod(t *testing.T) {
	t.Parallel()
	svc := newSvc(nil, nil, &fakeApplicationsRepo{})
	_, err := svc.GetEnterpriseApplications(context.Background(), uuid.New(), 17)
	assert.ErrorIs(t, err, domainstats.ErrPeriodInvalid)
}

func TestService_GetEnterpriseApplications_NotWired(t *testing.T) {
	t.Parallel()
	svc := appstats.NewService(nil, nil, nil)
	_, err := svc.GetEnterpriseApplications(context.Background(), uuid.New(), 30)
	assert.ErrorContains(t, err, "not wired")
}

func TestService_GetVisibility_NotWired(t *testing.T) {
	t.Parallel()
	svc := appstats.NewService(nil, nil, nil)
	_, err := svc.GetVisibility(context.Background(), uuid.New(), 30)
	assert.ErrorContains(t, err, "not wired")
}

func TestService_GetKeywords_NotWired(t *testing.T) {
	t.Parallel()
	svc := appstats.NewService(nil, nil, nil)
	_, err := svc.GetKeywords(context.Background(), uuid.New(), 30, 10)
	assert.ErrorContains(t, err, "not wired")
}

func TestHashUserAgent(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", appstats.HashUserAgent(""))
	assert.Equal(t, "", appstats.HashUserAgent("   "))
	a := appstats.HashUserAgent("Mozilla/5.0")
	b := appstats.HashUserAgent("Mozilla/5.0")
	c := appstats.HashUserAgent("Chrome")
	assert.Len(t, a, 64) // hex sha-256
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}
