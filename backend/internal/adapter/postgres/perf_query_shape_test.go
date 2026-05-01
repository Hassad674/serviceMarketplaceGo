package postgres_test

// PERF-B-08 / PERF-B-11 — query-shape regression tests.
//
// Without these, a future refactor could silently re-introduce the
// LEFT JOIN users on provider_id (which forced Postgres into a
// BitmapOr + nested-loop plan) — go test ./... would still pass but
// p50 latency on /api/v1/projects + /api/v1/wallet would balloon by
// 50–150 ms once production tables crossed ~10k rows for an org.
//
// The tests use sqlmock with the regexp matcher to assert two
// invariants per affected query:
//
//   1. The new query SHAPE is what the production adapter emits — the
//      WHERE predicate references the denormalized
//      provider_organization_id column (proposals + payment_records).
//
//   2. The OLD shape — `LEFT JOIN users provider_user` — is REJECTED.
//      sqlmock's regexp matcher fails the test if the adapter ever
//      regresses to the previous query.
//
// PERF-B-11: queryGetTotalUnread / queryGetTotalUnreadBatch must
// include `unread_count > 0` so the planner picks the partial index
// idx_conversation_read_state_user_unread (migration 074). The same
// shape test ensures a regression doesn't drop that predicate.

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/pkg/cursor"
)

// encodeTestCursor wraps cursor.Encode with a fixed test signature so
// the cursor branch tests don't repeat the same boilerplate.
func encodeTestCursor(createdAt time.Time, id uuid.UUID) string {
	return cursor.Encode(createdAt, id)
}

// minimalPaymentRecord builds a deterministic *payment.PaymentRecord
// with only the columns the INSERT touches populated. Tests that
// assert query SHAPE care about the SQL, not the row content — this
// helper keeps that distinction explicit.
func minimalPaymentRecord() *payment.PaymentRecord {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	return &payment.PaymentRecord{
		ID:                 uuid.New(),
		ProposalID:         uuid.New(),
		MilestoneID:        uuid.New(),
		ClientID:           uuid.New(),
		ProviderID:         uuid.New(),
		ProposalAmount:     50000,
		StripeFeeAmount:    1500,
		PlatformFeeAmount:  2500,
		ClientTotalAmount:  54000,
		ProviderPayout:     46000,
		Currency:           "EUR",
		Status:             payment.RecordStatusPending,
		TransferStatus:     payment.TransferPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// newPerfMockDB returns a sqlmock-backed adapter ready to use the
// regexp matcher. The matcher is REGEXP, not exact, so the assertions
// below pin partial query shape (column/JOIN presence) without
// hard-coding whitespace.
func newPerfMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	cleanup := func() { _ = db.Close() }
	t.Cleanup(cleanup)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet(), "unmet sqlmock expectations")
	})

	// Wire the adapter against the mock — note the postgres package
	// returns concrete types so we can't call db.Open variants.
	// Instead the caller creates the repo with this *sql.DB.
	_ = postgres.NewPaymentRecordRepository(db)
	return mock, cleanup
}

// ---------------------------------------------------------------------------
// PERF-B-08 — payment_records.ListByOrganization
// ---------------------------------------------------------------------------

func TestPaymentRecord_ListByOrganization_UsesDenormalizedProviderOrgColumn(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewPaymentRecordRepository(db)
	orgID := uuid.New()

	// Match the new shape — denormalized column reference. The match
	// is on the substring `provider_organization_id` and on the
	// ABSENCE of any `LEFT JOIN users` clause anywhere in the
	// statement.
	mock.ExpectQuery(`FROM payment_records pr\s+WHERE pr\.organization_id = \$1 OR pr\.provider_organization_id = \$1\s+ORDER BY pr\.created_at DESC`).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "proposal_id", "milestone_id", "client_id", "provider_id",
			"stripe_payment_intent_id", "stripe_transfer_id",
			"proposal_amount", "stripe_fee_amount", "platform_fee_amount",
			"client_total_amount", "provider_payout",
			"currency", "status", "transfer_status",
			"paid_at", "transferred_at", "created_at", "updated_at",
		}))

	_, qErr := repo.ListByOrganization(context.Background(), orgID)
	require.NoError(t, qErr)
	require.NoError(t, mock.ExpectationsWereMet(),
		"PERF-B-08 regression — query must use provider_organization_id, NOT a JOIN on users")
}

// TestPaymentRecord_ListByOrganization_RejectsOldJoinShape proves
// that the adapter does NOT emit the legacy JOIN — sqlmock will fail
// "unexpected query" if a future change re-introduces the JOIN.
//
// The mechanism: we expect ONLY the new shape (no JOIN). If the
// adapter emits the old shape it won't match the regexp, sqlmock
// errors out, the test fails — exactly the regression guard we want.
func TestPaymentRecord_ListByOrganization_RejectsOldJoinShape(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewPaymentRecordRepository(db)
	orgID := uuid.New()

	// New shape only — no `LEFT JOIN users`.
	mock.ExpectQuery(`^[^J]*FROM payment_records pr WHERE pr\.organization_id`).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "proposal_id", "milestone_id", "client_id", "provider_id",
			"stripe_payment_intent_id", "stripe_transfer_id",
			"proposal_amount", "stripe_fee_amount", "platform_fee_amount",
			"client_total_amount", "provider_payout",
			"currency", "status", "transfer_status",
			"paid_at", "transferred_at", "created_at", "updated_at",
		}))

	_, qErr := repo.ListByOrganization(context.Background(), orgID)
	require.NoError(t, qErr)
}

// TestPaymentRecord_ListByOrganization_ScansRows proves the row scan
// loop materialises records correctly through the new query shape.
// Complements the SHAPE assertions above so we know the function
// behaviour didn't regress when the JOIN was removed.
func TestPaymentRecord_ListByOrganization_ScansRows(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewPaymentRecordRepository(db)
	orgID := uuid.New()

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	rec1ID := uuid.New()
	rec2ID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "proposal_id", "milestone_id", "client_id", "provider_id",
		"stripe_payment_intent_id", "stripe_transfer_id",
		"proposal_amount", "stripe_fee_amount", "platform_fee_amount",
		"client_total_amount", "provider_payout",
		"currency", "status", "transfer_status",
		"paid_at", "transferred_at", "created_at", "updated_at",
	}).AddRow(
		rec1ID, uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		"pi_a", "tr_a",
		int64(50000), int64(1500), int64(2500),
		int64(54000), int64(46000),
		"EUR", "succeeded", "pending",
		nil, nil, now, now,
	).AddRow(
		rec2ID, uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		"pi_b", "tr_b",
		int64(20000), int64(600), int64(1000),
		int64(21600), int64(18400),
		"EUR", "pending", "pending",
		nil, nil, now, now,
	)

	mock.ExpectQuery(`FROM payment_records pr\s+WHERE pr\.organization_id = \$1 OR pr\.provider_organization_id = \$1`).
		WithArgs(orgID).
		WillReturnRows(rows)

	out, qErr := repo.ListByOrganization(context.Background(), orgID)
	require.NoError(t, qErr)
	require.Len(t, out, 2)
	assert.Equal(t, rec1ID, out[0].ID)
	assert.Equal(t, "EUR", out[0].Currency)
	assert.EqualValues(t, 50000, out[0].ProposalAmount)
	assert.Equal(t, rec2ID, out[1].ID)
}

// ---------------------------------------------------------------------------
// PERF-B-08 — proposals.ListActiveProjectsByOrganization
// ---------------------------------------------------------------------------

func TestProposalRepo_ListActiveProjectsByOrganization_UsesDenormalizedProviderOrg(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewProposalRepository(db)
	orgID := uuid.New()

	// New shape: WHERE p.organization_id OR p.provider_organization_id
	// — no LEFT JOIN users.
	mock.ExpectQuery(`FROM proposals p\s+WHERE \(p\.organization_id = \$1 OR p\.provider_organization_id = \$1\)`).
		WithArgs(orgID, 21). // limit+1 = 21 for default page 20
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "conversation_id", "sender_id", "recipient_id",
			"title", "description", "amount", "deadline",
			"status", "parent_id", "version",
			"client_id", "provider_id", "metadata",
			"active_dispute_id", "last_dispute_id",
			"accepted_at", "declined_at", "paid_at", "completed_at",
			"created_at", "updated_at",
		}))

	_, _, qErr := repo.ListActiveProjectsByOrganization(context.Background(), orgID, "", 20)
	require.NoError(t, qErr)
	require.NoError(t, mock.ExpectationsWereMet(),
		"PERF-B-08 regression — proposals.ListActiveProjectsByOrg must use provider_organization_id directly")
}

// TestProposalRepo_ListActiveProjectsByOrganization_CursorVariant
// covers the keyset-pagination branch of the same function. Same shape
// invariant — no JOIN on users, denormalized column reference.
func TestProposalRepo_ListActiveProjectsByOrganization_CursorVariant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewProposalRepository(db)
	orgID := uuid.New()

	// Build a real cursor — the function calls cursor.Decode internally
	// so we use the matching encoder.
	cursorStr := encodeTestCursor(time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC), uuid.New())

	mock.ExpectQuery(`FROM proposals p\s+WHERE \(p\.organization_id = \$1 OR p\.provider_organization_id = \$1\).*\(p\.created_at, p\.id\) < \(\$2, \$3\)`).
		WithArgs(orgID, sqlmock.AnyArg(), sqlmock.AnyArg(), 21).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "conversation_id", "sender_id", "recipient_id",
			"title", "description", "amount", "deadline",
			"status", "parent_id", "version",
			"client_id", "provider_id", "metadata",
			"active_dispute_id", "last_dispute_id",
			"accepted_at", "declined_at", "paid_at", "completed_at",
			"created_at", "updated_at",
		}))

	_, _, qErr := repo.ListActiveProjectsByOrganization(context.Background(), orgID, cursorStr, 20)
	require.NoError(t, qErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// PERF-B-08 — proposals.IsOrgAuthorizedForProposal
// ---------------------------------------------------------------------------

func TestProposalRepo_IsOrgAuthorized_UsesDenormalizedColumn(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewProposalRepository(db)
	proposalID := uuid.New()
	orgID := uuid.New()

	mock.ExpectQuery(`FROM proposals p\s+WHERE p\.id = \$1\s+AND \(p\.organization_id = \$2 OR p\.provider_organization_id = \$2\)`).
		WithArgs(proposalID, orgID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	ok, qErr := repo.IsOrgAuthorizedForProposal(context.Background(), proposalID, orgID)
	require.NoError(t, qErr)
	assert.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet(),
		"PERF-B-08 regression — IsOrgAuthorized must use the denormalized provider_organization_id column")
}

// ---------------------------------------------------------------------------
// PERF-B-11 — conversation_read_state.GetTotalUnread uses partial index
// ---------------------------------------------------------------------------

func TestConversationRepo_GetTotalUnread_UsesPartialIndexPredicate(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewConversationRepository(db)
	userID := uuid.New()

	// The predicate `unread_count > 0` is what makes Postgres pick
	// idx_conversation_read_state_user_unread (migration 074, partial
	// index WHERE unread_count > 0). Rows with unread_count = 0
	// contribute 0 to SUM so they can be safely filtered out.
	mock.ExpectQuery(`FROM conversation_read_state\s+WHERE user_id = \$1\s+AND unread_count > 0`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(7)))

	got, qErr := repo.GetTotalUnread(context.Background(), userID)
	require.NoError(t, qErr)
	assert.Equal(t, 7, got)
	require.NoError(t, mock.ExpectationsWereMet(),
		"PERF-B-11 regression — query must filter on unread_count > 0 to hit the partial index")
}

func TestConversationRepo_GetTotalUnreadBatch_UsesPartialIndexPredicate(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewConversationRepository(db)
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}

	mock.ExpectQuery(`FROM conversation_read_state\s+WHERE user_id = ANY\(\$1\)\s+AND unread_count > 0\s+GROUP BY user_id`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "sum"}))

	_, qErr := repo.GetTotalUnreadBatch(context.Background(), userIDs)
	require.NoError(t, qErr)
	require.NoError(t, mock.ExpectationsWereMet(),
		"PERF-B-11 regression — batch query must filter on unread_count > 0")
}

// ---------------------------------------------------------------------------
// PERF-B-08 — payment_records.Create denormalizes provider_organization_id
// ---------------------------------------------------------------------------

func TestPaymentRecord_Create_PopulatesProviderOrganizationID(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := postgres.NewPaymentRecordRepository(db)

	// The INSERT must contain BOTH organization_id (resolved from
	// organization_members keyed on the client) AND
	// provider_organization_id (resolved from users keyed on the
	// provider). Migration 131 introduced the second column.
	mock.ExpectExec(`INSERT INTO payment_records.*organization_id, provider_organization_id.*SELECT organization_id FROM organization_members WHERE user_id = \$4 LIMIT 1.*SELECT organization_id FROM users WHERE id = \$5`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Build a minimal payment record — only the fields the INSERT
	// touches need to be populated. The repo layer uses ptrString +
	// timestamps so we feed deterministic values.
	rec := minimalPaymentRecord()
	require.NoError(t, repo.Create(context.Background(), rec))
	require.NoError(t, mock.ExpectationsWereMet())
}

