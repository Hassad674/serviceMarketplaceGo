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
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/handler/middleware"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fakeUserAccountStore implements userAccountStore for tests. Tracks every
// call so assertions can inspect what the handler invoked.
type fakeUserAccountStore struct {
	mu sync.Mutex

	// State
	accountID string
	country   string

	// Injected errors
	getErr   error
	setErr   error
	clearErr error

	// Call tracking
	getCalls   int
	setCalls   int
	clearCalls int

	// Last-seen args
	lastSetAccountID string
	lastSetCountry   string
	lastClearUserID  uuid.UUID
}

func (f *fakeUserAccountStore) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.getErr != nil {
		return "", "", f.getErr
	}
	if f.accountID == "" {
		return "", "", sql.ErrNoRows
	}
	return f.accountID, f.country, nil
}

func (f *fakeUserAccountStore) SetStripeAccount(_ context.Context, _ uuid.UUID, accountID, country string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.accountID = accountID
	f.country = country
	f.lastSetAccountID = accountID
	f.lastSetCountry = country
	return nil
}

func (f *fakeUserAccountStore) ClearStripeAccount(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clearCalls++
	f.lastClearUserID = userID
	if f.clearErr != nil {
		return f.clearErr
	}
	f.accountID = ""
	f.country = ""
	return nil
}

func makeRequestWithUser(method, path string, body []byte, userID uuid.UUID) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := context.WithValue(r.Context(), middleware.ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

func makeRequestNoUser(method, path string, body []byte) *http.Request {
	if body != nil {
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		return r
	}
	return httptest.NewRequest(method, path, nil)
}

// ---------------------------------------------------------------------------
// ResetAccount
// ---------------------------------------------------------------------------

func TestResetAccount_NoUser_Returns401(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestNoUser("DELETE", "/account-session", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, store.clearCalls)
}

func TestResetAccount_ValidUser_Returns204(t *testing.T) {
	store := &fakeUserAccountStore{accountID: "acct_to_delete"}
	h := NewEmbeddedHandler(store)
	userID := uuid.New()

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, userID))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 1, store.clearCalls)
	assert.Equal(t, userID, store.lastClearUserID)
	assert.Equal(t, "", store.accountID) // cleared
}

func TestResetAccount_IdempotentNoRows(t *testing.T) {
	store := &fakeUserAccountStore{} // no existing account
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 1, store.clearCalls) // still called (idempotent)
}

func TestResetAccount_DBError_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{clearErr: errors.New("db down")}
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "db_error")
}

func TestResetAccount_ResponseContentType_None(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestResetAccount_ConcurrentCalls_AllIndependent(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))
		}()
	}
	wg.Wait()

	require.Equal(t, 10, store.clearCalls)
}

func TestResetAccount_UsesRequestingUserID(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	userID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, userID))

	assert.Equal(t, userID, store.lastClearUserID)
}

// ---------------------------------------------------------------------------
// GetAccountStatus (DB path only — Stripe API call paths need real Stripe)
// ---------------------------------------------------------------------------

func TestGetAccountStatus_NoUser_Returns401(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, makeRequestNoUser("GET", "/account-status", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetAccountStatus_NoAccount_Returns404(t *testing.T) {
	store := &fakeUserAccountStore{} // no account → GetStripeAccount returns ErrNoRows
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, makeRequestWithUser("GET", "/account-status", nil, uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "no_account")
	assert.Equal(t, 1, store.getCalls)
}

func TestGetAccountStatus_DBError_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{getErr: errors.New("connection lost")}
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, makeRequestWithUser("GET", "/account-status", nil, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "lookup_error")
}

// ---------------------------------------------------------------------------
// CreateAccountSession — input validation paths (no Stripe API calls)
// ---------------------------------------------------------------------------

func TestCreateAccountSession_NoUser_Returns401(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestNoUser("POST", "/account-session", body))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateAccountSession_NoBody_NoExistingAccount_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{} // no account
	h := NewEmbeddedHandler(store)

	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "country is required")
}

func TestCreateAccountSession_EmptyBody_MissingCountry_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	body, _ := json.Marshal(map[string]string{})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "country is required")
}

func TestCreateAccountSession_InvalidBusinessType_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	body, _ := json.Marshal(map[string]string{
		"country":       "FR",
		"business_type": "partnership",
	})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "business_type must be")
}

func TestCreateAccountSession_DBLookupError_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{getErr: errors.New("db down")}
	h := NewEmbeddedHandler(store)

	body, _ := json.Marshal(map[string]string{
		"country":       "FR",
		"business_type": "individual",
	})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateAccountSession_MalformedJSON_NoExistingAccount_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	body := []byte(`{this is not valid json`)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	// Malformed body silently treated as empty → still requires country
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateAccountSession_EmptyBodyLength_NoExistingAccount_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store)

	r := httptest.NewRequest("POST", "/account-session", bytes.NewReader([]byte{}))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = 0
	r = r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyUserID, uuid.New()))

	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, r)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ---------------------------------------------------------------------------
// Input normalization (validation passes → handler hits next step)
// ---------------------------------------------------------------------------

func TestCreateAccountSession_CountryNormalization(t *testing.T) {
	cases := []struct {
		name    string
		country string
	}{
		{"lowercase", "fr"},
		{"uppercase", "FR"},
		{"whitespace trimmed", "  FR  "},
		{"mixed case", "Fr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Inject a getErr so the handler exits right after normalization.
			store := &fakeUserAccountStore{getErr: errors.New("post-validation exit")}
			h := NewEmbeddedHandler(store)
			body, _ := json.Marshal(map[string]string{
				"country":       tc.country,
				"business_type": "individual",
			})
			rec := httptest.NewRecorder()
			h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

			// 500 confirms we made it past country normalization
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.NotContains(t, strings.ToLower(rec.Body.String()), "country is required")
		})
	}
}

func TestCreateAccountSession_BusinessTypeNormalization(t *testing.T) {
	validCases := []string{"individual", "Individual", "INDIVIDUAL", "  individual  ", "company", "Company"}
	for _, input := range validCases {
		t.Run(input, func(t *testing.T) {
			store := &fakeUserAccountStore{getErr: errors.New("post-validation exit")}
			h := NewEmbeddedHandler(store)
			body, _ := json.Marshal(map[string]string{
				"country":       "FR",
				"business_type": input,
			})
			rec := httptest.NewRecorder()
			h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			// Should NOT fail with "business_type must be" — it passed validation
			assert.NotContains(t, strings.ToLower(rec.Body.String()), "business_type must be")
		})
	}

	invalidCases := []string{"partnership", "sole_proprietor", "LLC", ""}
	for _, input := range invalidCases {
		t.Run("invalid_"+input, func(t *testing.T) {
			store := &fakeUserAccountStore{}
			h := NewEmbeddedHandler(store)
			body, _ := json.Marshal(map[string]string{
				"country":       "FR",
				"business_type": input,
			})
			rec := httptest.NewRecorder()
			h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Contains(t, strings.ToLower(rec.Body.String()), "business_type must be")
		})
	}
}

// ---------------------------------------------------------------------------
// Cross-border error translation (country not supported from FR platform)
// ---------------------------------------------------------------------------

func TestCreateAccountSession_CrossBorderError_Returns400(t *testing.T) {
	// Simulate the Stripe error surfacing via lookup (which reports the
	// upstream create-account failure message).
	store := &fakeUserAccountStore{
		getErr: errors.New("create stripe account: cannot be created by platforms in FR"),
	}
	h := NewEmbeddedHandler(store)

	body, _ := json.Marshal(map[string]string{
		"country":       "IN",
		"business_type": "individual",
	})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "country_not_supported")
}
