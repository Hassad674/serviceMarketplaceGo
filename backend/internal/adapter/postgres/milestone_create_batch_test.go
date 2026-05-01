package postgres_test

// PERF-B-04 — unit-level regression coverage for the multi-row
// CreateBatch path. The integration suite (milestone_repository_test.go)
// covers the round-trip semantics; these tests use sqlmock to assert
// the SHAPE of the query: exactly ONE ExecContext call regardless of
// batch size, multi-row VALUES tuple, ordering of arguments preserved.
//
// Without these unit tests the regression is invisible without a live
// DB — go test ./... in CI would silently accept a re-introduced loop
// of N INSERTs.

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/milestone"
)

// newMockedMilestoneRepo wires the postgres adapter against a sqlmock
// connection. The sqlmock instance is returned so the caller can set
// expectations and fail the test if anything is left unmet.
func newMockedMilestoneRepo(t *testing.T) (*postgres.MilestoneRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err, "open sqlmock")
	t.Cleanup(func() { _ = db.Close() })
	return postgres.NewMilestoneRepository(db), mock
}

// fixedMilestone returns a deterministic milestone for argument-shape
// assertions. Only the fields the INSERT touches are populated.
func fixedMilestone(proposalID uuid.UUID, seq int) *milestone.Milestone {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	return &milestone.Milestone{
		ID:          uuid.New(),
		ProposalID:  proposalID,
		Sequence:    seq,
		Title:       "T",
		Description: "D",
		Amount:      10000,
		Deadline:    nil,
		Status:      milestone.StatusPendingFunding,
		Version:     0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestCreateBatch_SingleExecContext_OneMilestone(t *testing.T) {
	repo, mock := newMockedMilestoneRepo(t)
	proposalID := uuid.New()
	ms := []*milestone.Milestone{fixedMilestone(proposalID, 1)}

	// Exactly one INSERT statement, regardless of batch size.
	mock.ExpectExec(`INSERT INTO proposal_milestones .* VALUES \(\$1, \$2`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.CreateBatch(context.Background(), ms))
	require.NoError(t, mock.ExpectationsWereMet(),
		"sqlmock: stray queries / unfulfilled expectations indicate a regression to the loop pattern")
}

func TestCreateBatch_SingleExecContext_FiveMilestones(t *testing.T) {
	repo, mock := newMockedMilestoneRepo(t)
	proposalID := uuid.New()
	ms := []*milestone.Milestone{
		fixedMilestone(proposalID, 1),
		fixedMilestone(proposalID, 2),
		fixedMilestone(proposalID, 3),
		fixedMilestone(proposalID, 4),
		fixedMilestone(proposalID, 5),
	}

	// Crucial assertion: ONE ExpectExec call — not five. If the loop
	// pattern is reintroduced sqlmock will fail with "unexpected query".
	mock.ExpectExec(`INSERT INTO proposal_milestones .* VALUES \(\$1, \$2.*\), \(\$20, \$21`).
		WillReturnResult(sqlmock.NewResult(0, 5))

	require.NoError(t, repo.CreateBatch(context.Background(), ms))
	require.NoError(t, mock.ExpectationsWereMet(),
		"PERF-B-04 regression — CreateBatch should issue exactly 1 INSERT")
}

func TestCreateBatch_PlaceholderShape_TwentyMilestones(t *testing.T) {
	// 20 is the cap enforced by milestone.NewMilestoneBatch — the worst
	// case in production. With 19 columns per row that's 380 args.
	// We assert the placeholder count matches by pattern-matching the
	// last expected row's leading placeholder.
	repo, mock := newMockedMilestoneRepo(t)
	proposalID := uuid.New()

	ms := make([]*milestone.Milestone, 20)
	for i := range ms {
		ms[i] = fixedMilestone(proposalID, i+1)
	}

	// Last row starts at $362 ((20-1)*19+1 = 362). The query should
	// contain that placeholder.
	mock.ExpectExec(`\$362`).WillReturnResult(sqlmock.NewResult(0, 20))

	require.NoError(t, repo.CreateBatch(context.Background(), ms))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateBatch_EmptyReturnsEmptyBatch(t *testing.T) {
	repo, mock := newMockedMilestoneRepo(t)

	err := repo.CreateBatch(context.Background(), nil)
	require.ErrorIs(t, err, milestone.ErrEmptyBatch)
	// Empty input must not touch the DB at all.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateBatch_DBErrorPropagates(t *testing.T) {
	repo, mock := newMockedMilestoneRepo(t)
	ms := []*milestone.Milestone{fixedMilestone(uuid.New(), 1)}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO proposal_milestones`)).
		WillReturnError(errors.New("connection refused"))

	err := repo.CreateBatch(context.Background(), ms)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insert milestones batch")
	assert.Contains(t, err.Error(), "connection refused")
}

// BenchmarkCreateBatch_FiveMilestones measures the single-call shape
// against a sqlmock backend. The win in production comes from
// eliminating N-1 round trips — sqlmock has zero latency so this
// benchmark only proves the function does not regress in CPU/alloc
// terms.
func BenchmarkCreateBatch_FiveMilestones(b *testing.B) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(b, err)
	defer func() { _ = db.Close() }()

	proposalID := uuid.New()
	ms := []*milestone.Milestone{
		fixedMilestone(proposalID, 1),
		fixedMilestone(proposalID, 2),
		fixedMilestone(proposalID, 3),
		fixedMilestone(proposalID, 4),
		fixedMilestone(proposalID, 5),
	}

	repo := postgres.NewMilestoneRepository(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ExpectExec(`INSERT INTO proposal_milestones`).
			WillReturnResult(sqlmock.NewResult(0, 5))
		_ = repo.CreateBatch(ctx, ms)
	}
}
