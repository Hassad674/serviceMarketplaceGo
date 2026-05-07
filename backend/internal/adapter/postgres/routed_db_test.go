package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/system"
)

// newRoutedDBPair builds two sqlmock-backed pools and the matching
// RoutedDB. Returns both raw mock handles so the tests can assert
// expectations on the right pool.
func newRoutedDBPair(t *testing.T) (*RoutedDB, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	appDB, appMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = appDB.Close() })

	adminDB, adminMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = adminDB.Close() })

	r, err := NewRoutedDB(appDB, adminDB)
	require.NoError(t, err)
	return r, appMock, adminMock
}

func TestRoutedDB_New_RejectsNilPools(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := NewRoutedDB(nil, db); err == nil {
		t.Fatal("expected error for nil app pool")
	}
	if _, err := NewRoutedDB(db, nil); err == nil {
		t.Fatal("expected error for nil admin pool")
	}
}

func TestRoutedDB_QueryContext_RoutesToAppByDefault(t *testing.T) {
	t.Parallel()
	r, appMock, adminMock := newRoutedDBPair(t)

	appMock.ExpectQuery(`SELECT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(1))

	ctx := context.Background()
	rows, err := r.QueryContext(ctx, "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	assert.NoError(t, appMock.ExpectationsWereMet(),
		"app pool must have been hit")
	assert.NoError(t, adminMock.ExpectationsWereMet(),
		"admin pool must NOT have been hit (no expectations set)")
}

func TestRoutedDB_QueryContext_RoutesToAdminWhenSystemActor(t *testing.T) {
	t.Parallel()
	r, appMock, adminMock := newRoutedDBPair(t)

	adminMock.ExpectQuery(`SELECT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(1))

	ctx := system.WithSystemActor(context.Background())
	rows, err := r.QueryContext(ctx, "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	assert.NoError(t, adminMock.ExpectationsWereMet(),
		"admin pool must have been hit")
	assert.NoError(t, appMock.ExpectationsWereMet(),
		"app pool must NOT have been hit (no expectations set)")
}

func TestRoutedDB_QueryRowContext_RoutesByContext(t *testing.T) {
	t.Parallel()
	r, appMock, adminMock := newRoutedDBPair(t)

	appMock.ExpectQuery(`SELECT 'app'`).
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow("app"))
	adminMock.ExpectQuery(`SELECT 'admin'`).
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow("admin"))

	var v string
	err := r.QueryRowContext(context.Background(), "SELECT 'app'").Scan(&v)
	require.NoError(t, err)
	assert.Equal(t, "app", v)

	err = r.QueryRowContext(system.WithSystemActor(context.Background()), "SELECT 'admin'").Scan(&v)
	require.NoError(t, err)
	assert.Equal(t, "admin", v)

	assert.NoError(t, appMock.ExpectationsWereMet())
	assert.NoError(t, adminMock.ExpectationsWereMet())
}

func TestRoutedDB_ExecContext_RoutesToAppByDefault(t *testing.T) {
	t.Parallel()
	r, appMock, _ := newRoutedDBPair(t)

	appMock.ExpectExec(`UPDATE foo SET bar = 1`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	res, err := r.ExecContext(context.Background(), "UPDATE foo SET bar = 1")
	require.NoError(t, err)
	rows, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	assert.NoError(t, appMock.ExpectationsWereMet())
}

func TestRoutedDB_ExecContext_RoutesToAdminWhenSystemActor(t *testing.T) {
	t.Parallel()
	r, _, adminMock := newRoutedDBPair(t)

	adminMock.ExpectExec(`DELETE FROM stale`).
		WillReturnResult(sqlmock.NewResult(0, 5))

	res, err := r.ExecContext(system.WithSystemActor(context.Background()), "DELETE FROM stale")
	require.NoError(t, err)
	rows, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(5), rows)

	assert.NoError(t, adminMock.ExpectationsWereMet())
}

func TestRoutedDB_BeginTx_RoutesByContext(t *testing.T) {
	t.Parallel()
	r, appMock, adminMock := newRoutedDBPair(t)

	appMock.ExpectBegin()
	appMock.ExpectCommit()
	adminMock.ExpectBegin()
	adminMock.ExpectCommit()

	tx, err := r.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	stx, err := r.BeginTx(system.WithSystemActor(context.Background()), nil)
	require.NoError(t, err)
	require.NoError(t, stx.Commit())

	assert.NoError(t, appMock.ExpectationsWereMet())
	assert.NoError(t, adminMock.ExpectationsWereMet())
}

func TestRoutedDB_PingContext_ChecksBothPools(t *testing.T) {
	t.Parallel()
	appDB, appMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer appDB.Close()
	adminDB, adminMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer adminDB.Close()

	r, err := NewRoutedDB(appDB, adminDB)
	require.NoError(t, err)

	appMock.ExpectPing()
	adminMock.ExpectPing()

	require.NoError(t, r.PingContext(context.Background()))
	assert.NoError(t, appMock.ExpectationsWereMet())
	assert.NoError(t, adminMock.ExpectationsWereMet())
}

func TestRoutedDB_PingContext_FailsWhenAdminPoolDown(t *testing.T) {
	t.Parallel()
	appDB, appMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer appDB.Close()
	adminDB, adminMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer adminDB.Close()

	r, err := NewRoutedDB(appDB, adminDB)
	require.NoError(t, err)

	wantErr := errors.New("admin down")
	appMock.ExpectPing()
	adminMock.ExpectPing().WillReturnError(wantErr)

	err = r.PingContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin pool ping")
}

func TestRoutedDB_AppPool_AdminPool_ReturnTheUnderlyingDBs(t *testing.T) {
	t.Parallel()
	appDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer appDB.Close()
	adminDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer adminDB.Close()

	r, err := NewRoutedDB(appDB, adminDB)
	require.NoError(t, err)

	// Identity comparison — the helpers must hand back the same
	// pointers that were passed in, otherwise the few callers that
	// reach into a specific pool would silently target a different
	// connection than the one they configured.
	assert.Same(t, appDB, r.AppPool())
	assert.Same(t, adminDB, r.AdminPool())
}

// pickPool is unexported; verify routing through the public surface
// to keep the test surface minimal.
var _ = sql.ErrNoRows
