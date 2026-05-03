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

// fakeUserAccountStore implements orgAccountStore for tests. Tracks every
// call so assertions can inspect what the handler invoked. The locks
// are emulated via a per-orgID sync.Mutex so concurrent tests exercise
// the same serialisation contract as the real PG advisory lock.
type fakeUserAccountStore struct {
	mu sync.Mutex

	// State
	accountID string
	country   string

	// Injected errors
	getErr   error
	setErr   error
	clearErr error
	// lockErr, when non-nil, is returned by WithStripeAccountLock
	// without invoking the callback — used to simulate a DB blip
	// while acquiring the advisory lock.
	lockErr error

	// Call tracking
	getCalls   int
	setCalls   int
	clearCalls int

	// Last-seen args
	lastSetAccountID string
	lastSetCountry   string
	lastClearUserID  uuid.UUID

	// orgLocks emulates the per-org PG advisory lock. Tests asserting
	// that two concurrent calls on the same org are serialised rely
	// on this lock being released at the end of the callback.
	orgLocks   map[uuid.UUID]*sync.Mutex
	orgLocksMu sync.Mutex

	// activePerOrg counts the in-flight callbacks per org. A value > 1
	// at any moment violates the BUG-04 contract — concurrent tests
	// inspect maxActive to assert serialisation held.
	activePerOrg map[uuid.UUID]int
	maxActive    int
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

// WithStripeAccountLock emulates the PG advisory lock contract: at
// most ONE callback runs per orgID at a time. Different orgs run
// concurrently. lockErr (when set) short-circuits before invoking fn.
// The callback's error is propagated as-is.
func (f *fakeUserAccountStore) WithStripeAccountLock(ctx context.Context, orgID uuid.UUID, fn func(ctx context.Context) error) error {
	if f.lockErr != nil {
		return f.lockErr
	}

	// Resolve / lazily create the org-scoped mutex.
	f.orgLocksMu.Lock()
	if f.orgLocks == nil {
		f.orgLocks = make(map[uuid.UUID]*sync.Mutex)
	}
	if f.activePerOrg == nil {
		f.activePerOrg = make(map[uuid.UUID]int)
	}
	mu, ok := f.orgLocks[orgID]
	if !ok {
		mu = &sync.Mutex{}
		f.orgLocks[orgID] = mu
	}
	f.orgLocksMu.Unlock()

	mu.Lock()
	// Track the in-flight count so concurrent tests can assert
	// "at most one callback per org at a time" — the BUG-04 contract.
	f.orgLocksMu.Lock()
	f.activePerOrg[orgID]++
	if f.activePerOrg[orgID] > f.maxActive {
		f.maxActive = f.activePerOrg[orgID]
	}
	f.orgLocksMu.Unlock()

	defer func() {
		f.orgLocksMu.Lock()
		f.activePerOrg[orgID]--
		f.orgLocksMu.Unlock()
		mu.Unlock()
	}()

	return fn(ctx)
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
	// Since phase R5, the embedded handler reads org_id from the
	// context — the tests use the same UUID for both so existing
	// assertions (last-seen id) keep working.
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, userID)
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
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestNoUser("DELETE", "/account-session", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, store.clearCalls)
}

func TestResetAccount_ValidUser_Returns204(t *testing.T) {
	store := &fakeUserAccountStore{accountID: "acct_to_delete"}
	h := NewEmbeddedHandler(store, "http://localhost:3001")
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
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 1, store.clearCalls) // still called (idempotent)
}

func TestResetAccount_DBError_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{clearErr: errors.New("db down")}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "db_error")
}

func TestResetAccount_ResponseContentType_None(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.ResetAccount(rec, makeRequestWithUser("DELETE", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestResetAccount_ConcurrentCalls_AllIndependent(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

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
	h := NewEmbeddedHandler(store, "http://localhost:3001")

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
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, makeRequestNoUser("GET", "/account-status", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetAccountStatus_NoAccount_Returns404(t *testing.T) {
	store := &fakeUserAccountStore{} // no account → GetStripeAccount returns ErrNoRows
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, makeRequestWithUser("GET", "/account-status", nil, uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "no_account")
	assert.Equal(t, 1, store.getCalls)
}

func TestGetAccountStatus_DBError_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{getErr: errors.New("connection lost")}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.GetAccountStatus(rec, makeRequestWithUser("GET", "/account-status", nil, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	// F.5 S4: the raw err.Error() must NOT reach the response. The
	// sanitized code is "db_error".
	assert.Contains(t, rec.Body.String(), "db_error")
	assert.NotContains(t, rec.Body.String(), "connection lost",
		"F.5 S4: raw error text must never leak to the client")
}

// ---------------------------------------------------------------------------
// CreateAccountSession — input validation paths (no Stripe API calls)
// ---------------------------------------------------------------------------

func TestCreateAccountSession_NoUser_Returns401(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body, _ := json.Marshal(map[string]string{"country": "FR", "business_type": "individual"})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestNoUser("POST", "/account-session", body))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateAccountSession_NoBody_NoExistingAccount_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{} // no account
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", nil, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	// F.5 S4: sanitized — body must NOT contain "country is required"
	// (raw err.Error() leak). The sanitized response surfaces a generic
	// stripe_error code.
	assert.Contains(t, rec.Body.String(), "stripe_error")
	assert.NotContains(t, rec.Body.String(), "country is required",
		"F.5 S4: raw error text must never leak to the client")
}

func TestCreateAccountSession_EmptyBody_MissingCountry_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body, _ := json.Marshal(map[string]string{})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	// F.5 S4: sanitized — see TestCreateAccountSession_NoBody_NoExistingAccount.
	assert.Contains(t, rec.Body.String(), "stripe_error")
	assert.NotContains(t, rec.Body.String(), "country is required",
		"F.5 S4: raw error text must never leak to the client")
}


func TestCreateAccountSession_DBLookupError_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{getErr: errors.New("db down")}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body, _ := json.Marshal(map[string]string{
		"country":       "FR",
		"business_type": "individual",
	})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// BUG-12 — malformed JSON body now surfaces a clear 400 invalid_json,
// instead of being silently swallowed and surfacing as "country is
// required" 500. The previous behaviour (assertion: 500) was a bug.
func TestCreateAccountSession_MalformedJSON_Returns400InvalidJSON(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body := []byte(`{this is not valid json`)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_json")
	// Lookup must NOT have run — we exited at body parsing.
	assert.Equal(t, 0, store.getCalls,
		"resolveStripeAccount must not run when JSON parsing fails")
}

// BUG-12 — a well-formed but type-wrong body (e.g. an array where an
// object is expected) is also invalid_json: json.Unmarshal returns a
// json.UnmarshalTypeError, which is still a parser failure from the
// handler's point of view.
func TestCreateAccountSession_TypeMismatchJSON_Returns400InvalidJSON(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	// Well-formed JSON, but an array where an object is expected.
	body := []byte(`["not", "an", "object"]`)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_json")
	assert.Equal(t, 0, store.getCalls)
}

// BUG-12 — whitespace-only body ("    ") is treated like no body. The
// optional-body legacy path is preserved: the handler still goes
// looking for an existing account and surfaces "country is required"
// when it cannot find one.
func TestCreateAccountSession_WhitespaceOnlyBody_TreatedAsNoBody(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body := []byte(`     `)
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	// We made it past parsing — into the "country required" path. F.5 S4:
	// the user-facing message is sanitized to a generic stripe_error.
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "stripe_error")
	assert.NotContains(t, rec.Body.String(), "country is required",
		"F.5 S4: raw error text must never leak to the client")
	// resolveStripeAccount was invoked (passed parsing).
	assert.Equal(t, 1, store.getCalls)
}

// BUG-12 — the legacy "no body at all" path must keep working. This
// is the path mobile clients on existing accounts hit when refreshing
// their session with a sentinel POST.
func TestCreateAccountSession_NoBody_PreservesLegacyBehaviour(t *testing.T) {
	store := &fakeUserAccountStore{accountID: "acct_existing", country: "FR"}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	// No body, but the user has an existing account → resolve should
	// short-circuit on the lookup. The Stripe API call after will fail
	// because we have no fixture, surfacing 500. We do NOT assert on
	// the exact terminal status — only that we never returned
	// invalid_json or country_required for an existing-account user.
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", nil, uuid.New()))

	assert.NotContains(t, rec.Body.String(), "invalid_json")
}

// BUG-12 — a valid JSON body with the expected fields keeps producing
// the legacy success path (modulo the Stripe API which is mocked at
// the store layer, not here).
func TestCreateAccountSession_ValidJSON_ReachesResolveStep(t *testing.T) {
	// Inject a getErr so we can confirm we made it past parsing.
	store := &fakeUserAccountStore{getErr: errors.New("post-parse exit")}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body, _ := json.Marshal(map[string]string{
		"country":       "FR",
		"business_type": "individual",
	})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	// Past-parse confirmation.
	assert.NotContains(t, rec.Body.String(), "invalid_json")
	assert.Equal(t, 1, store.getCalls)
}

func TestCreateAccountSession_EmptyBodyLength_NoExistingAccount_Returns500(t *testing.T) {
	store := &fakeUserAccountStore{}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	r := httptest.NewRequest("POST", "/account-session", bytes.NewReader([]byte{}))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = 0
	uid := uuid.New()
	ctx := context.WithValue(r.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	r = r.WithContext(ctx)

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
			h := NewEmbeddedHandler(store, "http://localhost:3001")
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


// ---------------------------------------------------------------------------
// Cross-border error translation (country not supported from FR platform)
// ---------------------------------------------------------------------------

func TestCreateAccountSession_CrossBorderError_Returns400(t *testing.T) {
	// Simulate the Stripe error surfacing via lookup (which reports the
	// upstream create-account failure message).
	store := &fakeUserAccountStore{
		getErr: errors.New("create stripe account: cannot be created by platforms in FR"),
	}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	body, _ := json.Marshal(map[string]string{
		"country":       "IN",
		"business_type": "individual",
	})
	rec := httptest.NewRecorder()
	h.CreateAccountSession(rec, makeRequestWithUser("POST", "/account-session", body, uuid.New()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "country_not_supported")
}

// ---------------------------------------------------------------------------
// BUG-04 — concurrent CreateAccountSession on the same org must serialise
// through the advisory lock. Without the fix, two simultaneous requests
// would both observe "no account" and create one each, leaving an orphan
// Stripe account.
// ---------------------------------------------------------------------------

// TestResolveStripeAccount_LockSerialisesCheckAndCreate calls
// resolveStripeAccount directly (the path that holds the lock) and
// asserts that two concurrent callers on the same org never see the
// "no account" state at the same time. The fake locker records the
// peak in-flight count per org; at most one callback per org may run
// at a time.
func TestResolveStripeAccount_LockSerialisesCheckAndCreate(t *testing.T) {
	const callers = 10
	store := &fakeUserAccountStore{}

	// Inject getErr so the callback exits before issuing a real Stripe
	// call. We're proving the LOCK contract, not the create branch.
	store.getErr = errors.New("post-lock fast exit")

	h := NewEmbeddedHandler(store, "http://localhost:3001")
	orgID := uuid.New()

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.resolveStripeAccount(context.Background(), orgID, "FR", "https://example.com")
		}()
	}
	wg.Wait()

	// BUG-04 invariant: at most ONE callback per org runs concurrently.
	store.orgLocksMu.Lock()
	defer store.orgLocksMu.Unlock()
	assert.Equal(t, 1, store.maxActive,
		"BUG-04: concurrent calls on the same org must serialise through the lock")
	// All 10 callers must have hit the GetStripeAccount path inside
	// the locked section — proving the lock didn't drop any of them.
	assert.Equal(t, callers, store.getCalls,
		"every concurrent caller must run inside the lock")
}

// TestResolveStripeAccount_DifferentOrgsRunConcurrently is the converse:
// the lock must NOT serialise different orgs against each other. Each
// org has its own lock key, so 10 different orgs can resolve in parallel.
func TestResolveStripeAccount_DifferentOrgsRunConcurrently(t *testing.T) {
	const callers = 10
	store := &fakeUserAccountStore{
		getErr: errors.New("post-lock fast exit"),
	}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		orgID := uuid.New() // each goroutine uses a distinct org
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.resolveStripeAccount(context.Background(), orgID, "FR", "https://example.com")
		}()
	}
	wg.Wait()

	// All 10 callers ran but none observed peak-active > 1 for its
	// own org because each had a unique key. The fake's maxActive is
	// global per-org, so it will have value 1 (one call per org), not
	// callers-many.
	store.orgLocksMu.Lock()
	defer store.orgLocksMu.Unlock()
	assert.LessOrEqual(t, store.maxActive, 1,
		"different orgs must NOT serialise against each other")
	assert.Equal(t, callers, store.getCalls)
}

// TestResolveStripeAccount_LockErrSurfaced verifies that when
// WithStripeAccountLock itself fails (e.g. PG advisory lock acquisition
// timed out), the error is surfaced and no Stripe call is attempted.
func TestResolveStripeAccount_LockErrSurfaced(t *testing.T) {
	store := &fakeUserAccountStore{
		lockErr: errors.New("advisory lock acquisition timed out"),
	}
	h := NewEmbeddedHandler(store, "http://localhost:3001")

	_, err := h.resolveStripeAccount(context.Background(), uuid.New(), "FR", "https://example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "advisory lock acquisition timed out")
	assert.Equal(t, 0, store.getCalls,
		"GetStripeAccount must NOT run when the lock cannot be acquired")
}

// TestResolveStripeAccount_ExistingAccount_LockHeldThroughSync proves
// the fast path "row already had an account id" still runs inside the
// lock — important because the syncBusinessProfile call inside it
// would otherwise race a concurrent retry.
func TestResolveStripeAccount_ExistingAccount_LockHeldThroughSync(t *testing.T) {
	store := &fakeUserAccountStore{accountID: "acct_existing", country: "FR"}
	h := NewEmbeddedHandler(store, "http://localhost:3001")
	orgID := uuid.New()

	const callers = 5
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.resolveStripeAccount(context.Background(), orgID, "FR", "https://example.com")
		}()
	}
	wg.Wait()

	store.orgLocksMu.Lock()
	defer store.orgLocksMu.Unlock()
	assert.Equal(t, 1, store.maxActive,
		"the fast path also runs under the per-org lock")
}
