package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripeapi "github.com/stripe/stripe-go/v82"
)

// TestSelectAccounts_All — the --org=all branch returns every
// connected account (stripe_account_id non-null) ordered by id, so
// the operator can predict the order of the dry-run output.
func TestSelectAccounts_All(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	orgA := uuid.New()
	orgB := uuid.New()
	mock.ExpectQuery("SELECT id, stripe_account_id FROM organizations").
		WillReturnRows(sqlmock.NewRows([]string{"id", "stripe_account_id"}).
			AddRow(orgA, "acct_1").
			AddRow(orgB, "acct_2"))

	rows, err := selectAccounts(context.Background(), db, "all")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "acct_1", rows[0].AccountID)
	assert.Equal(t, "acct_2", rows[1].AccountID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSelectAccounts_SingleOrg — passing a uuid restricts the result
// to one organization. When the org row exists and has an account id,
// the function returns exactly one row.
func TestSelectAccounts_SingleOrg(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	orgID := uuid.New()
	mock.ExpectQuery(`SELECT stripe_account_id FROM organizations WHERE id = \$1`).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_only"))

	rows, err := selectAccounts(context.Background(), db, orgID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, orgID, rows[0].OrgID)
	assert.Equal(t, "acct_only", rows[0].AccountID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSelectAccounts_OrgWithoutAccount — a typo or an org that never
// onboarded onto Stripe must error out instead of silently no-oping
// the script. The operator wants a hard signal that nothing happened.
func TestSelectAccounts_OrgWithoutAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	orgID := uuid.New()
	mock.ExpectQuery(`SELECT stripe_account_id FROM organizations WHERE id = \$1`).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow(nil))

	_, err = selectAccounts(context.Background(), db, orgID.String())
	assert.Error(t, err, "no connected account → hard error, not silent skip")
}

// TestSelectAccounts_RejectsBadOrgFlag — invalid uuid must error
// before hitting the DB so the operator notices the typo immediately.
func TestSelectAccounts_RejectsBadOrgFlag(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	_, err = selectAccounts(context.Background(), db, "not-a-uuid")
	assert.Error(t, err)
}

// TestSetManual_PostsManualScheduleToStripe — the actual Stripe call
// hits /v1/accounts/<id> with settings[payouts][schedule][interval]=
// manual. This is the load-bearing assertion for the backfill: a
// regression that drops the body parameter would silently leave
// thousands of accounts on the daily auto-payout schedule.
func TestSetManual_PostsManualScheduleToStripe(t *testing.T) {
	var capturedPath, capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"acct_test","object":"account"}`))
	}))
	defer srv.Close()

	prevAPI := stripeapi.GetBackend(stripeapi.APIBackend)
	prevKey := stripeapi.Key
	stripeapi.Key = "sk_test_unit"
	stripeapi.SetBackend(stripeapi.APIBackend,
		stripeapi.GetBackendWithConfig(stripeapi.APIBackend, &stripeapi.BackendConfig{
			URL:           stripeapi.String(srv.URL),
			LeveledLogger: &stripeapi.LeveledLogger{Level: stripeapi.LevelNull},
		}))
	defer func() {
		stripeapi.SetBackend(stripeapi.APIBackend, prevAPI)
		stripeapi.Key = prevKey
	}()

	err := setManual(context.Background(), "acct_test")
	require.NoError(t, err)
	assert.Equal(t, "/v1/accounts/acct_test", capturedPath)
	assert.Contains(t, capturedBody, "settings[payouts][schedule][interval]=manual")
}
