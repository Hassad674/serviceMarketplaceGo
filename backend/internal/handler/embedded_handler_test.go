package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v82"

	"marketplace-backend/internal/handler/middleware"
)

// --- Stripe mock backend -----------------------------------------------------
//
// The embedded handler calls Stripe APIs directly via globals (stripe.Key and
// the package-level Backends). We install a fake Backend implementation for
// both APIBackend and ConnectBackend so tests can exercise Stripe code paths
// deterministically without real HTTP calls.

// stripeMockBackend implements stripe.Backend.
type stripeMockBackend struct {
	// callFn is invoked for every Call(). If it returns a non-nil error, the
	// SDK surfaces it to the caller. If it returns nil, the response json
	// payload is written into v via reflection (we emulate only the subset
	// the handler actually inspects: id, client_secret, expires_at).
	callFn func(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error
}

func (m *stripeMockBackend) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	if m.callFn != nil {
		return m.callFn(method, path, key, params, v)
	}
	return errors.New("stripe mock: no call handler configured")
}

func (m *stripeMockBackend) CallStreaming(_, _, _ string, _ stripe.ParamsContainer, _ stripe.StreamingLastResponseSetter) error {
	return errors.New("stripe mock: streaming not supported")
}

func (m *stripeMockBackend) CallRaw(_, _, _ string, _ []byte, _ *stripe.Params, _ stripe.LastResponseSetter) error {
	return errors.New("stripe mock: raw not supported")
}

func (m *stripeMockBackend) CallMultipart(_, _, _, _ string, _ *bytes.Buffer, _ *stripe.Params, _ stripe.LastResponseSetter) error {
	return errors.New("stripe mock: multipart not supported")
}

func (m *stripeMockBackend) SetMaxNetworkRetries(_ int64) {}

// installMockStripeBackend wires a mock backend for both APIBackend and
// ConnectBackend (Connect is used by account.* and accountsession.*) and
// returns a restore function to revert to the originals.
func installMockStripeBackend(t *testing.T, backend *stripeMockBackend) func() {
	t.Helper()
	originalKey := stripe.Key
	stripe.Key = "sk_test_mock"
	originalAPI := stripe.GetBackend(stripe.APIBackend)
	originalConnect := stripe.GetBackend(stripe.ConnectBackend)
	stripe.SetBackend(stripe.APIBackend, backend)
	stripe.SetBackend(stripe.ConnectBackend, backend)
	return func() {
		stripe.SetBackend(stripe.APIBackend, originalAPI)
		stripe.SetBackend(stripe.ConnectBackend, originalConnect)
		stripe.Key = originalKey
	}
}

// stripeAlwaysErrorBackend returns an error for every call — used when the
// handler should NEVER reach Stripe (failure should occur earlier).
func stripeAlwaysErrorBackend() *stripeMockBackend {
	return &stripeMockBackend{
		callFn: func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("stripe mock: unexpected API call")
		},
	}
}

// --- Test helpers ------------------------------------------------------------

func newEmbeddedTestHandler(t *testing.T) (*EmbeddedHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	h := NewEmbeddedHandler(db)
	return h, mock, func() { _ = db.Close() }
}

func withEmbeddedUser(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

// --- CreateAccountSession: auth ---------------------------------------------

func TestCreateAccountSession_MissingUser_Returns401(t *testing.T) {
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

// --- CreateAccountSession: validation (no existing account) ------------------

func TestCreateAccountSession_NoExistingAccount_MissingCountry_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	// No existing account. Must NOT call Stripe because validation fails first.
	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{"business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "country is required")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_InvalidBusinessType_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "invalid_type"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "business_type must be")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_EmptyBusinessType_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "business_type must be")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_MissingBody_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	// Body is nil -> country="" -> validation fails
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "country is required")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_EmptyBody_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "country is required")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_MalformedJSON_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	// Malformed JSON: json.Unmarshal fails silently, fields stay empty
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "country is required")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- CreateAccountSession: DB errors ----------------------------------------

func TestCreateAccountSession_LookupDBError_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(errors.New("db connection lost"))

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "lookup account")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- CreateAccountSession: existing account happy path -----------------------

func TestCreateAccountSession_ExistingAccount_ReturnsSession(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	// Mock Stripe: Account.Update (sync profile) + AccountSession.New both succeed.
	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && strings.HasPrefix(path, "/v1/accounts/acct_existing"):
				// syncBusinessProfile: account.Update
				_ = json.Unmarshal([]byte(`{"id":"acct_existing"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/account_sessions":
				_ = json.Unmarshal([]byte(`{"client_secret":"cs_test_123","expires_at":1700000000,"account":"acct_existing"}`), v)
				return nil
			}
			return errors.New("unexpected call: " + method + " " + path)
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_existing"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp embeddedAccountSessionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "cs_test_123", resp.ClientSecret)
	assert.Equal(t, "acct_existing", resp.AccountID)
	assert.Equal(t, int64(1700000000), resp.ExpiresAt)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_ExistingAccount_SyncBusinessProfileFails_StillReturnsSession(t *testing.T) {
	// syncBusinessProfile failure is non-fatal (logged warning only).
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && strings.HasPrefix(path, "/v1/accounts/acct_existing"):
				return errors.New("stripe: account update rate-limited")
			case method == http.MethodPost && path == "/v1/account_sessions":
				_ = json.Unmarshal([]byte(`{"client_secret":"cs_test_xyz","expires_at":1700001234,"account":"acct_existing"}`), v)
				return nil
			}
			return errors.New("unexpected call")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_existing"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp embeddedAccountSessionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "cs_test_xyz", resp.ClientSecret)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_ExistingAccount_SessionCreationFails_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && strings.HasPrefix(path, "/v1/accounts/acct_existing"):
				_ = json.Unmarshal([]byte(`{"id":"acct_existing"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/account_sessions":
				return errors.New("stripe: session creation failed")
			}
			return errors.New("unexpected call")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_existing"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "stripe_session_error")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- CreateAccountSession: new account creation ------------------------------

func TestCreateAccountSession_NoExistingAccount_ValidInputs_CreatesAccount(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && path == "/v1/tokens":
				_ = json.Unmarshal([]byte(`{"id":"tok_test_123"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/accounts":
				_ = json.Unmarshal([]byte(`{"id":"acct_new_abc"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/account_sessions":
				_ = json.Unmarshal([]byte(`{"client_secret":"cs_test_new","expires_at":1700002000,"account":"acct_new_abc"}`), v)
				return nil
			}
			return errors.New("unexpected call: " + method + " " + path)
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)
	mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
		WithArgs(uid, "acct_new_abc", "FR").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp embeddedAccountSessionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "acct_new_abc", resp.AccountID)
	assert.Equal(t, "cs_test_new", resp.ClientSecret)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_CompanyBusinessType_CreatesAccount(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && path == "/v1/tokens":
				_ = json.Unmarshal([]byte(`{"id":"tok_co"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/accounts":
				_ = json.Unmarshal([]byte(`{"id":"acct_de_company"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/account_sessions":
				_ = json.Unmarshal([]byte(`{"client_secret":"cs_co","expires_at":1700003000,"account":"acct_de_company"}`), v)
				return nil
			}
			return errors.New("unexpected call")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)
	mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
		WithArgs(uid, "acct_de_company", "DE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"country": "DE", "business_type": "company"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_LowercaseCountry_NormalizedToUpper(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && path == "/v1/tokens":
				_ = json.Unmarshal([]byte(`{"id":"tok_a"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/accounts":
				_ = json.Unmarshal([]byte(`{"id":"acct_us"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/account_sessions":
				_ = json.Unmarshal([]byte(`{"client_secret":"cs_us","expires_at":1,"account":"acct_us"}`), v)
				return nil
			}
			return errors.New("unexpected")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)
	// Country "us" must be normalized to "US" before persistence
	mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
		WithArgs(uid, "acct_us", "US").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"country": "us", "business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_WhitespaceInputs_Trimmed(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && path == "/v1/tokens":
				_ = json.Unmarshal([]byte(`{"id":"tok_x"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/accounts":
				_ = json.Unmarshal([]byte(`{"id":"acct_gb"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/account_sessions":
				_ = json.Unmarshal([]byte(`{"client_secret":"cs_gb","expires_at":1,"account":"acct_gb"}`), v)
				return nil
			}
			return errors.New("unexpected")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)
	mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
		WithArgs(uid, "acct_gb", "GB").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"country": "  gb  ", "business_type": " Company "})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_TokenCreationFails_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			if method == http.MethodPost && path == "/v1/tokens" {
				return errors.New("stripe: token creation failed")
			}
			return errors.New("unexpected call: " + method + " " + path)
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "create account token")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_AccountCreationFails_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && path == "/v1/tokens":
				_ = json.Unmarshal([]byte(`{"id":"tok_ok"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/accounts":
				return errors.New("stripe: account creation failed")
			}
			return errors.New("unexpected call")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "create stripe account")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCreateAccountSession_NoExistingAccount_PersistFails_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			switch {
			case method == http.MethodPost && path == "/v1/tokens":
				_ = json.Unmarshal([]byte(`{"id":"tok_ok"}`), v)
				return nil
			case method == http.MethodPost && path == "/v1/accounts":
				_ = json.Unmarshal([]byte(`{"id":"acct_ok"}`), v)
				return nil
			}
			return errors.New("unexpected call")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)
	mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
		WithArgs(uid, "acct_ok", "FR").
		WillReturnError(errors.New("db write failed"))

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "persist account id")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// Table-driven test across multiple supported countries.
func TestCreateAccountSession_MultipleCountries_AllSucceed(t *testing.T) {
	countries := []string{"FR", "US", "DE", "GB", "ES", "IT", "NL", "CA", "AU"}

	for _, country := range countries {
		t.Run(country, func(t *testing.T) {
			h, mockDB, cleanup := newEmbeddedTestHandler(t)
			defer cleanup()

			expectedAcct := "acct_" + strings.ToLower(country)
			backend := &stripeMockBackend{
				callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
					switch {
					case method == http.MethodPost && path == "/v1/tokens":
						_ = json.Unmarshal([]byte(`{"id":"tok_x"}`), v)
						return nil
					case method == http.MethodPost && path == "/v1/accounts":
						_ = json.Unmarshal([]byte(`{"id":"`+expectedAcct+`"}`), v)
						return nil
					case method == http.MethodPost && path == "/v1/account_sessions":
						_ = json.Unmarshal([]byte(`{"client_secret":"cs_x","expires_at":1,"account":"`+expectedAcct+`"}`), v)
						return nil
					}
					return errors.New("unexpected")
				},
			}
			restore := installMockStripeBackend(t, backend)
			defer restore()

			uid := uuid.New()
			mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
				WithArgs(uid).
				WillReturnError(sql.ErrNoRows)
			mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
				WithArgs(uid, expectedAcct, country).
				WillReturnResult(sqlmock.NewResult(1, 1))

			body, _ := json.Marshal(map[string]string{"country": country, "business_type": "individual"})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withEmbeddedUser(req, uid)
			rec := httptest.NewRecorder()
			h.CreateAccountSession(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

// --- ResetAccount ------------------------------------------------------------

func TestResetAccount_MissingUser_Returns401(t *testing.T) {
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/payment-info/account-session", nil)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestResetAccount_NoExistingAccount_Returns204(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	uid := uuid.New()
	// DELETE is idempotent: 0 rows affected still returns 204
	mockDB.ExpectExec(`DELETE FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestResetAccount_ExistingAccount_DeletesAndReturns204(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	uid := uuid.New()
	mockDB.ExpectExec(`DELETE FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestResetAccount_DBError_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	uid := uuid.New()
	mockDB.ExpectExec(`DELETE FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(errors.New("db is down"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "db_error")
	assert.Contains(t, rec.Body.String(), "db is down")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestResetAccount_DeletesExactlyOwnRow(t *testing.T) {
	// Verify the DELETE query is scoped by user_id — critical for tenant isolation.
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	uid := uuid.New()
	mockDB.ExpectExec(`DELETE FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid). // anything other than uid would fail sqlmock
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- GetAccountStatus --------------------------------------------------------

func TestGetAccountStatus_MissingUser_Returns401(t *testing.T) {
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestGetAccountStatus_NoAccountInDB_Returns404(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "no_account")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAccountStatus_DBError_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	restore := installMockStripeBackend(t, stripeAlwaysErrorBackend())
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(errors.New("connection timeout"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "lookup_error")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAccountStatus_AccountExists_StripeSucceeds_Returns200(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			if method == http.MethodGet && strings.HasPrefix(path, "/v1/accounts/acct_present") {
				_ = json.Unmarshal([]byte(`{
					"id":"acct_present",
					"country":"FR",
					"business_type":"individual",
					"charges_enabled":true,
					"payouts_enabled":false,
					"details_submitted":true,
					"requirements":{"currently_due":["external_account"],"past_due":[]}
				}`), v)
				return nil
			}
			return errors.New("unexpected")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_present"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp embeddedAccountStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "acct_present", resp.AccountID)
	assert.Equal(t, "FR", resp.Country)
	assert.Equal(t, "individual", resp.BusinessType)
	assert.True(t, resp.ChargesEnabled)
	assert.False(t, resp.PayoutsEnabled)
	assert.True(t, resp.DetailsSubmitted)
	assert.Equal(t, []string{"external_account"}, resp.RequirementsCurrentlyDue)
	assert.Equal(t, 1, resp.RequirementsCount)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAccountStatus_AccountExists_StripeFails_Returns500(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			if method == http.MethodGet && strings.HasPrefix(path, "/v1/accounts/") {
				return errors.New("stripe api: account not found")
			}
			return errors.New("unexpected")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_broken"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "stripe_error")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAccountStatus_AccountWithPastDueRequirements(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			if method == http.MethodGet && strings.HasPrefix(path, "/v1/accounts/") {
				_ = json.Unmarshal([]byte(`{
					"id":"acct_late",
					"country":"US",
					"business_type":"company",
					"charges_enabled":false,
					"payouts_enabled":false,
					"details_submitted":true,
					"requirements":{"currently_due":["tos_acceptance.date"],"past_due":["individual.id_number","external_account"]}
				}`), v)
				return nil
			}
			return errors.New("unexpected")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_late"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp embeddedAccountStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "company", resp.BusinessType)
	assert.Equal(t, 2, len(resp.RequirementsPastDue))
	assert.Equal(t, 3, resp.RequirementsCount)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAccountStatus_AccountWithoutRequirements_CountIsZero(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			if method == http.MethodGet && strings.HasPrefix(path, "/v1/accounts/") {
				_ = json.Unmarshal([]byte(`{
					"id":"acct_clean",
					"country":"FR",
					"business_type":"individual",
					"charges_enabled":true,
					"payouts_enabled":true,
					"details_submitted":true,
					"requirements":{"currently_due":[],"past_due":[]}
				}`), v)
				return nil
			}
			return errors.New("unexpected")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_clean"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp embeddedAccountStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.ChargesEnabled)
	assert.True(t, resp.PayoutsEnabled)
	assert.Equal(t, 0, resp.RequirementsCount)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- findAccountID (via GetAccountStatus routing) ----------------------------

func TestFindAccountID_RowExists_ReturnsID(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	h := NewEmbeddedHandler(db)

	uid := uuid.New()
	mock.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"stripe_account_id"}).AddRow("acct_123"))

	got, err := h.findAccountID(context.Background(), uid)
	assert.NoError(t, err)
	assert.Equal(t, "acct_123", got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAccountID_NoRow_ReturnsErrNoRows(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	h := NewEmbeddedHandler(db)

	uid := uuid.New()
	mock.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	got, err := h.findAccountID(context.Background(), uid)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAccountID_DBError_PropagatesError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	h := NewEmbeddedHandler(db)

	uid := uuid.New()
	mock.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(errors.New("connection refused"))

	got, err := h.findAccountID(context.Background(), uid)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, sql.ErrNoRows)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- persistAccountID --------------------------------------------------------

func TestPersistAccountID_NewRow_InsertsSuccessfully(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	h := NewEmbeddedHandler(db)

	uid := uuid.New()
	mock.ExpectExec(`INSERT INTO test_embedded_accounts \(user_id, stripe_account_id, country\)`).
		WithArgs(uid, "acct_new", "FR").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = h.persistAccountID(context.Background(), uid, "acct_new", "FR")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPersistAccountID_ExistingRow_UpdatesViaOnConflict(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	h := NewEmbeddedHandler(db)

	uid := uuid.New()
	// ON CONFLICT updates an existing row: simulated by returning 1 rows affected
	// (which for upsert still matches sqlmock's NewResult regardless of insert/update).
	mock.ExpectExec(`ON CONFLICT \(user_id\) DO UPDATE`).
		WithArgs(uid, "acct_replaced", "US").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = h.persistAccountID(context.Background(), uid, "acct_replaced", "US")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPersistAccountID_DBError_PropagatesError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	h := NewEmbeddedHandler(db)

	uid := uuid.New()
	mock.ExpectExec(`INSERT INTO test_embedded_accounts`).
		WithArgs(uid, "acct_x", "FR").
		WillReturnError(errors.New("unique constraint violation"))

	err = h.persistAccountID(context.Background(), uid, "acct_x", "FR")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unique constraint")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- syncBusinessProfile -----------------------------------------------------

func TestSyncBusinessProfile_Success(t *testing.T) {
	backend := &stripeMockBackend{
		callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
			if method == http.MethodPost && strings.HasPrefix(path, "/v1/accounts/acct_sync") {
				_ = json.Unmarshal([]byte(`{"id":"acct_sync"}`), v)
				return nil
			}
			return errors.New("unexpected call: " + method + " " + path)
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	err := syncBusinessProfile("acct_sync", "https://example.com")
	assert.NoError(t, err)
}

func TestSyncBusinessProfile_StripeError_Propagates(t *testing.T) {
	backend := &stripeMockBackend{
		callFn: func(_, _, _ string, _ stripe.ParamsContainer, _ stripe.LastResponseSetter) error {
			return errors.New("stripe: unauthorized")
		},
	}
	restore := installMockStripeBackend(t, backend)
	defer restore()

	err := syncBusinessProfile("acct_bad", "https://example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

// --- NewEmbeddedHandler ------------------------------------------------------

func TestNewEmbeddedHandler_SetsDefaultPlatformURL(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := NewEmbeddedHandler(db)
	assert.NotNil(t, h)
	assert.Equal(t, db, h.db)
	assert.Equal(t, "https://marketplace-service.com", h.platformURL)
}

// --- Input normalization edge cases (table-driven) ---------------------------

func TestCreateAccountSession_InputNormalization(t *testing.T) {
	// This verifies the Country -> ToUpper+Trim and BusinessType -> ToLower+Trim
	// normalization logic by inspecting the DB INSERT args and Stripe token call.
	tests := []struct {
		name           string
		inputCountry   string
		inputBizType   string
		expectCountry  string
		expectValid    bool
		expectErrorMsg string
	}{
		{"lowercase country", "fr", "individual", "FR", true, ""},
		{"mixed case country", "Us", "individual", "US", true, ""},
		{"trimmed country", "  DE  ", "individual", "DE", true, ""},
		{"uppercase business_type", "FR", "INDIVIDUAL", "FR", true, ""},
		{"mixed case business_type", "FR", "Company", "FR", true, ""},
		{"spaces in business_type", "FR", "  individual  ", "FR", true, ""},
		{"empty country", "", "individual", "", false, "country is required"},
		{"whitespace only country", "   ", "individual", "", false, "country is required"},
		{"empty business_type", "FR", "", "", false, "business_type must be"},
		{"whitespace only business_type", "FR", "   ", "", false, "business_type must be"},
		{"bogus business_type", "FR", "partnership", "", false, "business_type must be"},
		{"numeric business_type", "FR", "123", "", false, "business_type must be"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, mockDB, cleanup := newEmbeddedTestHandler(t)
			defer cleanup()

			backend := &stripeMockBackend{
				callFn: func(method, path, _ string, _ stripe.ParamsContainer, v stripe.LastResponseSetter) error {
					switch {
					case method == http.MethodPost && path == "/v1/tokens":
						_ = json.Unmarshal([]byte(`{"id":"tok_norm"}`), v)
						return nil
					case method == http.MethodPost && path == "/v1/accounts":
						_ = json.Unmarshal([]byte(`{"id":"acct_norm"}`), v)
						return nil
					case method == http.MethodPost && path == "/v1/account_sessions":
						_ = json.Unmarshal([]byte(`{"client_secret":"cs_norm","expires_at":1,"account":"acct_norm"}`), v)
						return nil
					}
					return errors.New("unexpected call")
				},
			}
			restore := installMockStripeBackend(t, backend)
			defer restore()

			uid := uuid.New()
			mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
				WithArgs(uid).
				WillReturnError(sql.ErrNoRows)
			if tc.expectValid {
				mockDB.ExpectExec(`INSERT INTO test_embedded_accounts`).
					WithArgs(uid, "acct_norm", tc.expectCountry).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			body, _ := json.Marshal(map[string]string{"country": tc.inputCountry, "business_type": tc.inputBizType})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withEmbeddedUser(req, uid)
			rec := httptest.NewRecorder()
			h.CreateAccountSession(rec, req)

			if tc.expectValid {
				assert.Equal(t, http.StatusOK, rec.Code, "expected 200 for %q/%q", tc.inputCountry, tc.inputBizType)
			} else {
				assert.Equal(t, http.StatusInternalServerError, rec.Code)
				assert.Contains(t, rec.Body.String(), tc.expectErrorMsg)
			}
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

// --- Method routing sanity (defensive): handlers don't mutate wrong methods --

func TestCreateAccountSession_WrongMethod_StillProcessed(t *testing.T) {
	// The handler function itself does not check r.Method — routing does.
	// This confirms that if someone accidentally wires a GET to it, it still
	// executes (and hits auth check first). Defensive test for future regressions.
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-session", nil)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestResetAccount_WrongMethod_StillProcessed(t *testing.T) {
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetAccountStatus_WrongMethod_StillProcessed(t *testing.T) {
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-status", nil)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Response content-type verification --------------------------------------

func TestCreateAccountSession_Unauthorized_SetsJSONContentType(t *testing.T) {
	h, _, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", nil)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestGetAccountStatus_NotFound_SetsJSONContentType(t *testing.T) {
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	uid := uuid.New()
	mockDB.ExpectQuery(`SELECT stripe_account_id FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/account-status", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- ResetAccount: headers ---------------------------------------------------

func TestResetAccount_Success_NoContentTypeHeader(t *testing.T) {
	// 204 No Content must not have a body — verify that.
	h, mockDB, cleanup := newEmbeddedTestHandler(t)
	defer cleanup()

	uid := uuid.New()
	mockDB.ExpectExec(`DELETE FROM test_embedded_accounts WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/payment-info/account-session", nil)
	req = withEmbeddedUser(req, uid)
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 0, rec.Body.Len())
	assert.NoError(t, mockDB.ExpectationsWereMet())
}
