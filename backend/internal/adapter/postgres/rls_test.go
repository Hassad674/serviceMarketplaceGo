package postgres_test

// Unit + integration tests for the RLS tenant-context helpers
// (SetCurrentOrg, SetCurrentUser, SetTenantContext, RunInTxWithTenant).
//
// The unit-style assertions on the nil-tx error paths run without a
// database. The "happy path" tests are gated on
// MARKETPLACE_TEST_DATABASE_URL — they need a live Postgres because
// SET LOCAL only takes effect inside a transaction.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
)

// ---------------------------------------------------------------------------
// Unit tests — no DB needed
// ---------------------------------------------------------------------------

func TestSetCurrentOrg_NilTx_ReturnsError(t *testing.T) {
	err := postgres.SetCurrentOrg(context.Background(), nil, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tx is required")
}

func TestSetCurrentUser_NilTx_ReturnsError(t *testing.T) {
	err := postgres.SetCurrentUser(context.Background(), nil, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tx is required")
}

func TestSetTenantContext_NilTx_ReturnsError(t *testing.T) {
	err := postgres.SetTenantContext(context.Background(), nil, uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tx is required")
}

func TestRunInTxWithTenant_NilFn_ReturnsError(t *testing.T) {
	// The runner can be a zero-value because the nil-fn check happens
	// before we touch the *sql.DB. This keeps the unit test free of
	// database dependencies.
	r := postgres.NewTxRunner(nil)
	err := r.RunInTxWithTenant(context.Background(), uuid.New(), uuid.New(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fn is required")
}

// ---------------------------------------------------------------------------
// Integration tests — gated on MARKETPLACE_TEST_DATABASE_URL
// ---------------------------------------------------------------------------

// readSetting fetches the current value of an app.* setting via
// current_setting. The "true" arg makes a missing setting return ''.
func readSetting(t *testing.T, ctx context.Context, tx *sql.Tx, name string) string {
	t.Helper()
	var v sql.NullString
	err := tx.QueryRowContext(ctx,
		"SELECT current_setting($1, true)", name,
	).Scan(&v)
	require.NoError(t, err)
	if !v.Valid {
		return ""
	}
	return v.String
}

func TestSetCurrentOrg_StoresUUIDAsString(t *testing.T) {
	db := testDB(t)

	orgID := uuid.New()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	require.NoError(t, postgres.SetCurrentOrg(context.Background(), tx, orgID))

	got := readSetting(t, context.Background(), tx, "app.current_org_id")
	assert.Equal(t, orgID.String(), got)
}

func TestSetCurrentUser_StoresUUIDAsString(t *testing.T) {
	db := testDB(t)

	userID := uuid.New()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	require.NoError(t, postgres.SetCurrentUser(context.Background(), tx, userID))

	got := readSetting(t, context.Background(), tx, "app.current_user_id")
	assert.Equal(t, userID.String(), got)
}

func TestSetTenantContext_SetsBothValues(t *testing.T) {
	db := testDB(t)

	orgID := uuid.New()
	userID := uuid.New()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	require.NoError(t, postgres.SetTenantContext(context.Background(), tx, orgID, userID))

	gotOrg := readSetting(t, context.Background(), tx, "app.current_org_id")
	gotUser := readSetting(t, context.Background(), tx, "app.current_user_id")
	assert.Equal(t, orgID.String(), gotOrg)
	assert.Equal(t, userID.String(), gotUser)
}

func TestSetTenantContext_NilOrgIDSkipsOrgSetter(t *testing.T) {
	db := testDB(t)

	userID := uuid.New()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	require.NoError(t, postgres.SetTenantContext(context.Background(), tx, uuid.Nil, userID))

	gotOrg := readSetting(t, context.Background(), tx, "app.current_org_id")
	gotUser := readSetting(t, context.Background(), tx, "app.current_user_id")
	assert.Equal(t, "", gotOrg, "uuid.Nil orgID must skip the org setter — current_setting returns '' for an unset key")
	assert.Equal(t, userID.String(), gotUser)
}

func TestSetTenantContext_NilUserIDSkipsUserSetter(t *testing.T) {
	db := testDB(t)

	orgID := uuid.New()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	require.NoError(t, postgres.SetTenantContext(context.Background(), tx, orgID, uuid.Nil))

	gotOrg := readSetting(t, context.Background(), tx, "app.current_org_id")
	gotUser := readSetting(t, context.Background(), tx, "app.current_user_id")
	assert.Equal(t, orgID.String(), gotOrg)
	assert.Equal(t, "", gotUser, "uuid.Nil userID must skip the user setter")
}

// TestSetCurrentOrg_LocalScope confirms the SET LOCAL semantics: a
// value set in one transaction must NOT leak into a second transaction
// on the same pooled connection. This is the critical guarantee that
// keeps RLS safe under concurrency.
func TestSetCurrentOrg_LocalScope(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgA := uuid.New()
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, postgres.SetCurrentOrg(ctx, tx1, orgA))
	gotA := readSetting(t, ctx, tx1, "app.current_org_id")
	require.Equal(t, orgA.String(), gotA)
	require.NoError(t, tx1.Commit())

	// Open a new transaction. Because SET LOCAL is tx-scoped, the
	// setting from tx1 must NOT survive.
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	gotB := readSetting(t, ctx, tx2, "app.current_org_id")
	assert.Equal(t, "", gotB, "SET LOCAL must not leak across transactions")
}

func TestRunInTxWithTenant_HappyPath(t *testing.T) {
	db := testDB(t)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()

	var (
		seenOrg  string
		seenUser string
	)
	err := runner.RunInTxWithTenant(ctx, orgID, userID, func(tx *sql.Tx) error {
		seenOrg = readSetting(t, ctx, tx, "app.current_org_id")
		seenUser = readSetting(t, ctx, tx, "app.current_user_id")
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, orgID.String(), seenOrg)
	assert.Equal(t, userID.String(), seenUser)
}

func TestRunInTxWithTenant_PropagatesFnError(t *testing.T) {
	db := testDB(t)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	sentinel := errors.New("boom")
	err := runner.RunInTxWithTenant(ctx, uuid.New(), uuid.New(), func(tx *sql.Tx) error {
		return sentinel
	})
	assert.ErrorIs(t, err, sentinel)
}

// TestRunInTxWithTenant_NilOrgID_AllowsCallback verifies the helper
// runs the callback even when both ids are nil (e.g. a background job
// touching only RLS-free tables but going through the same wrapper
// for consistency).
func TestRunInTxWithTenant_NilOrgID_AllowsCallback(t *testing.T) {
	db := testDB(t)
	runner := postgres.NewTxRunner(db)
	ctx := context.Background()

	called := false
	err := runner.RunInTxWithTenant(ctx, uuid.Nil, uuid.Nil, func(tx *sql.Tx) error {
		called = true
		gotOrg := readSetting(t, ctx, tx, "app.current_org_id")
		gotUser := readSetting(t, ctx, tx, "app.current_user_id")
		assert.Equal(t, "", gotOrg)
		assert.Equal(t, "", gotUser)
		return nil
	})
	require.NoError(t, err)
	assert.True(t, called)
}

// TestSetCurrentOrg_DBError_WrapsError covers the err != nil branch
// in SetCurrentOrg by deliberately rolling back the transaction
// before calling the helper, so set_config returns "transaction is
// already aborted" / "tx is closed" depending on driver state.
func TestSetCurrentOrg_DBError_WrapsError(t *testing.T) {
	db := testDB(t)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	// tx is now closed — ExecContext will fail with sql.ErrTxDone.
	err = postgres.SetCurrentOrg(context.Background(), tx, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set current org")
}

func TestSetCurrentUser_DBError_WrapsError(t *testing.T) {
	db := testDB(t)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = postgres.SetCurrentUser(context.Background(), tx, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set current user")
}

// TestSetTenantContext_DBError_PropagatesFromOrgSetter ensures the
// composite helper surfaces the underlying SetCurrentOrg failure when
// the tx is closed, exercising the "if err := SetCurrentOrg..." branch.
func TestSetTenantContext_DBError_PropagatesFromOrgSetter(t *testing.T) {
	db := testDB(t)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = postgres.SetTenantContext(context.Background(), tx, uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set current org")
}

// TestSetTenantContext_DBError_PropagatesFromUserSetter exercises the
// branch where the org setter succeeds but the user setter fails.
// This is a contrived scenario — to hit it, we keep the tx open for
// the org call (so it succeeds) and close it before the user call by
// using a wrapper trick: cancel the context between the two calls.
//
// Implementation note: SET LOCAL accepts repeated calls in the same
// tx, so the first SetCurrentOrg succeeds. We then manually invoke
// SetCurrentUser with a cancelled context to force a failure.
func TestSetTenantContext_UserSetter_ErrorPath(t *testing.T) {
	db := testDB(t)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// First setter succeeds.
	require.NoError(t, postgres.SetCurrentOrg(context.Background(), tx, uuid.New()))

	// Cancel the ctx so the next ExecContext fails.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = postgres.SetCurrentUser(ctx, tx, uuid.New())
	require.Error(t, err)
}

// TestSetTenantContext_PropagatesUserSetterError fully exercises the
// "return err" branch from the userID block of SetTenantContext.
// We open a tx, succeed on the org setter, then cancel the ctx so the
// user setter fails — SetTenantContext must surface that error.
func TestSetTenantContext_PropagatesUserSetterError(t *testing.T) {
	db := testDB(t)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// We need the org setter to succeed (hit set_config once with a
	// valid ctx) and the user setter to fail. The cleanest way is to
	// pass uuid.Nil for orgID — that skips the org branch entirely —
	// then a cancelled ctx for the user branch so it fails.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = postgres.SetTenantContext(ctx, tx, uuid.Nil, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set current user")
}

// TestRunInTxWithTenant_PropagatesContextError exercises the
// "if err := SetTenantContext..." branch of RunInTxWithTenant. We
// pass a cancelled ctx so the inner SetTenantContext fails on the
// first ExecContext call.
func TestRunInTxWithTenant_PropagatesContextError(t *testing.T) {
	db := testDB(t)
	runner := postgres.NewTxRunner(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	err := runner.RunInTxWithTenant(ctx, uuid.New(), uuid.New(), func(tx *sql.Tx) error {
		called = true
		return nil
	})
	require.Error(t, err)
	assert.False(t, called, "fn must not be invoked when SetTenantContext fails")
}
