package postgres_test

// Integration tests for ProfileViewRepository + SearchQueryStatsRepository.
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.
// Uses the shared `searchTestDB` helper from search_document_repository_test.go.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	domainstats "marketplace-backend/internal/domain/stats"
	"marketplace-backend/internal/port/repository"
)

// seedStatsOrg creates the bare minimum chain (user + organization)
// required for the FK on profile_view_events to validate.
// Returns the orgID; t.Cleanup wipes both rows.
func seedStatsOrg(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	ownerID := uuid.New()
	orgID := uuid.New()
	email := fmt.Sprintf("stats-%s@test.local", orgID.String()[:8])
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type, last_active_at)
		VALUES ($1, $2, 'hash', 'Stats', 'Owner', 'Stats Owner', 'provider', 'marketplace_owner', now())`,
		ownerID, email)
	require.NoError(t, err, "insert user")

	stripeAcct := "acct_test_stats_" + orgID.String()[:8]
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name, stripe_account_id)
		VALUES ($1, $2, 'provider_personal', 'Stats Org', $3)`,
		orgID, ownerID, stripeAcct)
	require.NoError(t, err, "insert organization")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM profile_view_events WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, ownerID)
	})
	return orgID
}

func recordViewWithCreatedAt(t *testing.T, db *sql.DB, orgID uuid.UUID, came domainstats.CameFrom, ip, ua string, q *string, pos *int, daysAgo int) {
	t.Helper()
	id := uuid.New()
	var qVal interface{}
	if q != nil {
		qVal = *q
	}
	var posVal interface{}
	if pos != nil {
		posVal = *pos
	}
	_, err := db.Exec(`
		INSERT INTO profile_view_events (id, organization_id, persona, viewer_ip_anonymized, viewer_ua_hash, came_from, search_query, search_position, created_at)
		VALUES ($1, $2, 'freelance', $3::inet, $4, $5, $6, $7, NOW() - ($8::int * INTERVAL '1 day'))`,
		id, orgID, ip, ua, string(came), qVal, posVal, daysAgo)
	require.NoError(t, err, "insert view event")
}

func TestProfileViewRepository_Record(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db)

	q := "go developer"
	pos := 2
	event, err := domainstats.NewViewEvent(domainstats.NewViewEventInput{
		OrganizationID:     orgID,
		Persona:            domainstats.PersonaFreelance,
		ViewerIPAnonymized: "203.0.113.0/24",
		ViewerUAHash:       "abc123",
		CameFrom:           domainstats.CameFromSearch,
		SearchQuery:        &q,
		SearchPosition:     &pos,
	})
	require.NoError(t, err)

	require.NoError(t, repo.Record(context.Background(), event))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM profile_view_events WHERE organization_id = $1`, orgID).Scan(&count))
	assert.Equal(t, 1, count)

	// Idempotent re-runs — fresh ID each call so a second Record with
	// the same domain input simply inserts a second row.
	event2, err := domainstats.NewViewEvent(domainstats.NewViewEventInput{
		OrganizationID:     orgID,
		Persona:            domainstats.PersonaFreelance,
		ViewerIPAnonymized: "203.0.113.0/24",
		ViewerUAHash:       "abc123",
		CameFrom:           domainstats.CameFromDirect,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Record(context.Background(), event2))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM profile_view_events WHERE organization_id = $1`, orgID).Scan(&count))
	assert.Equal(t, 2, count)
}

func TestProfileViewRepository_AggregateVisibility(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db)

	// 100 events: 50 unique IPs, across the last 30 days. Mix of
	// search/direct so the search counters are non-zero.
	q := "go developer"
	for i := 0; i < 100; i++ {
		ip := fmt.Sprintf("198.51.%d.0/24", i%50) // 50 unique networks
		ua := fmt.Sprintf("ua-%d", i%50)
		daysAgo := i % 30 // 0..29 days
		var query *string
		var pos *int
		came := domainstats.CameFromDirect
		if i%2 == 0 {
			came = domainstats.CameFromSearch
			query = &q
			p := (i % 10) + 1
			pos = &p
		}
		recordViewWithCreatedAt(t, db, orgID, came, ip, ua, query, pos, daysAgo)
	}

	// Insert a "stale" event 60 days ago — must NOT be counted in
	// the 30-day window.
	recordViewWithCreatedAt(t, db, orgID, domainstats.CameFromDirect, "10.0.0.0/24", "stale-ua", nil, nil, 60)

	got, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
	})
	require.NoError(t, err)
	assert.Equal(t, orgID.String(), got.OrganizationID)
	assert.Equal(t, domainstats.Period30Days, got.PeriodDays)
	assert.Equal(t, 100, got.TotalViews, "stale event must be excluded from 30d window")
	assert.Equal(t, 50, got.UniqueViewers)
	assert.Equal(t, 50, got.SearchAppearances)
	assert.Greater(t, got.AvgSearchPosition, 0.0)
	assert.Less(t, got.AvgSearchPosition, 11.0)
	assert.NotEmpty(t, got.Series)

	// Sanity: 7-day window must be a strict subset.
	got7, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period7Days,
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, got7.TotalViews, got.TotalViews)
}

func TestProfileViewRepository_DailySeries_UniqueVsTotal(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db)

	// Day -1: 2 distinct fingerprints, 3 total events (one viewer
	// hits the profile twice). Day -2: 1 fingerprint, 1 event.
	// The resulting series MUST report Count >= Unique on every day,
	// and Day -1 must show Count=3 / Unique=2.
	recordViewWithCreatedAt(t, db, orgID, domainstats.CameFromDirect, "10.10.1.0/24", "ua-A", nil, nil, 1)
	recordViewWithCreatedAt(t, db, orgID, domainstats.CameFromDirect, "10.10.1.0/24", "ua-A", nil, nil, 1) // same viewer, second hit
	recordViewWithCreatedAt(t, db, orgID, domainstats.CameFromDirect, "10.10.2.0/24", "ua-B", nil, nil, 1) // different viewer same day
	recordViewWithCreatedAt(t, db, orgID, domainstats.CameFromDirect, "10.10.3.0/24", "ua-C", nil, nil, 2)

	got, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
	})
	require.NoError(t, err)
	require.Len(t, got.Series, 2, "expected one bucket per distinct day")

	for _, b := range got.Series {
		assert.LessOrEqual(t, b.Unique, b.Count, "unique always <= total")
	}

	// Series is ordered ASC by day so [0] is the older bucket (day -2).
	older, newer := got.Series[0], got.Series[1]
	assert.Equal(t, 1, older.Count)
	assert.Equal(t, 1, older.Unique)
	assert.Equal(t, 3, newer.Count, "newer day saw 3 total events")
	assert.Equal(t, 2, newer.Unique, "newer day saw 2 distinct viewers")
}

func TestProfileViewRepository_DailySeries_Period365(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db)

	// Spread one event per quarter across the last year.
	for _, daysAgo := range []int{1, 90, 180, 270, 360} {
		recordViewWithCreatedAt(t, db, orgID,
			domainstats.CameFromDirect,
			fmt.Sprintf("10.20.%d.0/24", daysAgo%200),
			fmt.Sprintf("ua-%d", daysAgo),
			nil, nil, daysAgo)
	}

	got, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period365Days,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, got.TotalViews, "365d window must include the year-old event")
	assert.GreaterOrEqual(t, len(got.Series), 4, "at least 4 distinct days in the series")
}

func TestProfileViewRepository_AggregateVisibility_EmptyOrg(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db) // org exists, no events

	got, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, got.TotalViews)
	assert.Equal(t, 0, got.UniqueViewers)
	assert.Equal(t, 0, got.SearchAppearances)
	assert.Equal(t, 0.0, got.AvgSearchPosition)
	assert.NotNil(t, got.Series)
	assert.Empty(t, got.Series)
}

func TestProfileViewRepository_AggregateVisibility_OrgIsolation(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgA := seedStatsOrg(t, db)
	orgB := seedStatsOrg(t, db)

	for i := 0; i < 5; i++ {
		recordViewWithCreatedAt(t, db, orgA, domainstats.CameFromDirect, "10.1.1.0/24", "ua-A", nil, nil, 1)
	}
	for i := 0; i < 7; i++ {
		recordViewWithCreatedAt(t, db, orgB, domainstats.CameFromDirect, "10.2.2.0/24", "ua-B", nil, nil, 1)
	}

	gotA, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgA, PeriodDays: domainstats.Period30Days})
	require.NoError(t, err)
	gotB, err := repo.AggregateVisibility(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgB, PeriodDays: domainstats.Period30Days})
	require.NoError(t, err)

	assert.Equal(t, 5, gotA.TotalViews, "org A must only see its own rows")
	assert.Equal(t, 7, gotB.TotalViews, "org B must only see its own rows")
}

func TestSearchQueryStatsRepository_TopKeywordsForOrg(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewSearchQueryStatsRepository(db)
	orgID := seedStatsOrg(t, db)

	// Seed search_queries rows with clicks pointing to orgID.
	// 5 rows for "go developer" at positions 1..5
	// 3 rows for "react senior"
	// 1 row for "data engineer"
	type seed struct {
		query string
		pos   int
		days  int
	}
	rows := []seed{
		{"Go Developer", 1, 1},
		{"go developer", 2, 2},
		{"go developer", 3, 3},
		{"go developer", 4, 4},
		{"go developer", 5, 5},
		{"react senior", 2, 2},
		{"react senior", 3, 3},
		{"REACT senior", 4, 4},
		{"data engineer", 1, 1},
	}
	for i, r := range rows {
		searchID := fmt.Sprintf("test-%s-%d", orgID.String()[:8], i)
		_, err := db.Exec(`
			INSERT INTO search_queries (search_id, query, persona, results_count, latency_ms, clicked_result_id, clicked_position, created_at)
			VALUES ($1, $2, 'freelance', 50, 30, $3, $4, NOW() - ($5::int * INTERVAL '1 day'))`,
			searchID, r.query, orgID, r.pos, r.days)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM search_queries WHERE clicked_result_id = $1`, orgID)
	})

	got, err := repo.TopKeywordsForOrg(context.Background(), repository.KeywordFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
		Limit:          10,
	})
	require.NoError(t, err)
	require.Len(t, got, 3, "expected 3 distinct lowercased keywords")

	// Order is by count DESC then keyword ASC.
	assert.Equal(t, "go developer", got[0].Keyword)
	assert.Equal(t, 5, got[0].Count)
	assert.InDelta(t, 3.0, got[0].AvgPosition, 0.001) // mean of 1..5

	assert.Equal(t, "react senior", got[1].Keyword)
	assert.Equal(t, 3, got[1].Count)

	assert.Equal(t, "data engineer", got[2].Keyword)
	assert.Equal(t, 1, got[2].Count)

	// Limit=2 must clip to top 2.
	clipped, err := repo.TopKeywordsForOrg(context.Background(), repository.KeywordFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
		Limit:          2,
	})
	require.NoError(t, err)
	assert.Len(t, clipped, 2)
}

func TestSearchQueryStatsRepository_TopKeywordsForOrg_Empty(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewSearchQueryStatsRepository(db)
	orgID := seedStatsOrg(t, db)

	got, err := repo.TopKeywordsForOrg(context.Background(), repository.KeywordFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
		Limit:          10,
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestProfileViewRepository_AggregateApplications(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db)

	// Seed a job belonging to the org and a few applications across
	// the last 10 days. job.creator_id is required (FK to users); we
	// reuse the org's owner user.
	var ownerID uuid.UUID
	require.NoError(t, db.QueryRow(`SELECT owner_user_id FROM organizations WHERE id = $1`, orgID).Scan(&ownerID))

	jobID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO jobs (id, creator_id, organization_id, title, description, budget_type, min_budget, max_budget)
		VALUES ($1, $2, $3, 'Stats Job', 'Test', 'fixed', 1000, 5000)`,
		jobID, ownerID, orgID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM job_applications WHERE job_id = $1`, jobID)
		_, _ = db.Exec(`DELETE FROM jobs WHERE id = $1`, jobID)
	})

	// 10 applicants — each seeded with their own user + organization
	// so the NOT NULL applicant_organization_id FK is satisfied.
	type applicant struct {
		userID, orgID uuid.UUID
	}
	applicants := make([]applicant, 10)
	for i := range applicants {
		uid := uuid.New()
		oid := uuid.New()
		applicants[i] = applicant{userID: uid, orgID: oid}
		email := fmt.Sprintf("appl-%s-%d@stats.test", orgID.String()[:8], i)
		_, err := db.Exec(`
			INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
			VALUES ($1, $2, 'h', 'Appl', 'Test', 'Appl Test', 'provider', 'marketplace_owner')`,
			uid, email)
		require.NoError(t, err)
		stripeAcct := "acct_test_appl_" + oid.String()[:8]
		_, err = db.Exec(`
			INSERT INTO organizations (id, owner_user_id, type, name, stripe_account_id)
			VALUES ($1, $2, 'provider_personal', 'Appl Org', $3)`,
			oid, uid, stripeAcct)
		require.NoError(t, err)
		_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, oid, uid)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, a := range applicants {
			_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, a.userID)
			_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, a.orgID)
		}
	})

	for i, a := range applicants {
		appID := uuid.New()
		// Spread across the last 10 days.
		_, err := db.Exec(`
			INSERT INTO job_applications (id, job_id, applicant_id, applicant_organization_id, message, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'msg', NOW() - ($5::int * INTERVAL '1 day'), NOW())`,
			appID, jobID, a.userID, a.orgID, i)
		require.NoError(t, err)
	}

	// Stale application 60 days ago — must be excluded from the 30d window.
	staleUserID := uuid.New()
	staleOrgID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
		VALUES ($1, $2, 'h', 'Stale', 'X', 'Stale', 'provider', 'marketplace_owner')`,
		staleUserID, fmt.Sprintf("stale-%s@stats.test", orgID.String()[:8]))
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name, stripe_account_id)
		VALUES ($1, $2, 'provider_personal', 'Stale Org', $3)`,
		staleOrgID, staleUserID, "acct_test_stale_"+staleOrgID.String()[:8])
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, staleOrgID, staleUserID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO job_applications (id, job_id, applicant_id, applicant_organization_id, message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'msg', NOW() - INTERVAL '60 days', NOW())`,
		uuid.New(), jobID, staleUserID, staleOrgID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, staleUserID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, staleOrgID)
	})

	got, err := repo.AggregateApplications(context.Background(), repository.VisibilityFilter{
		OrganizationID: orgID,
		PeriodDays:     domainstats.Period30Days,
	})
	require.NoError(t, err)
	assert.Equal(t, 10, got.TotalCount, "stale 60d application excluded")
	assert.NotEmpty(t, got.Series)
}

// Quick sanity check that the IP truncation goes round-trip correctly
// (and rejects junk inputs at the INET cast layer).
func TestProfileViewRepository_Record_RejectsBadIP(t *testing.T) {
	db := searchTestDB(t)
	repo := postgres.NewProfileViewRepository(db)
	orgID := seedStatsOrg(t, db)

	// Synthesize a domain event whose IP would NOT round-trip through
	// the INET column. We bypass NewViewEvent's validator by hand-
	// crafting the struct so the error surfaces from the DB driver.
	bad := &domainstats.ViewEvent{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		Persona:            domainstats.PersonaFreelance,
		ViewerIPAnonymized: "not-an-ip",
		ViewerUAHash:       "x",
		CameFrom:           domainstats.CameFromDirect,
		CreatedAt:          time.Now(),
	}
	err := repo.Record(context.Background(), bad)
	assert.Error(t, err, "INET cast must reject malformed input")
}
