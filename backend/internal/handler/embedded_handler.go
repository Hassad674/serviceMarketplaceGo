package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountsession"
	"github.com/stripe/stripe-go/v82/token"

	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// orgAccountStore is the trimmed view of OrganizationRepository this
// handler needs. Defined here so tests can stub it without pulling the
// full repository. Since phase R5 the Stripe Connect account lives on
// the organization (the merchant of record), not on an individual user.
//
// WithStripeAccountLock serialises concurrent check-and-create flows
// per org using a PostgreSQL transaction-scoped advisory lock
// (pg_advisory_xact_lock). Closes BUG-04: two concurrent
// CreateAccountSession requests from the same org would each see
// "no account" and create one, leaking an orphan Stripe account
// while only the second one was persisted in DB. By holding the
// lock for the entire check + create + persist sequence, only one
// goroutine ever sees the empty state.
//
// The implementation MUST run the callback inside a SQL transaction
// that takes pg_advisory_xact_lock(hashtext(org_id)) as its first
// statement so the lock is released exactly when the transaction
// commits / rolls back. Failures inside the callback are surfaced
// to the caller; the lock is always released by the transaction
// boundary regardless of success or panic.
type orgAccountStore interface {
	GetStripeAccount(ctx context.Context, orgID uuid.UUID) (accountID, country string, err error)
	SetStripeAccount(ctx context.Context, orgID uuid.UUID, accountID, country string) error
	ClearStripeAccount(ctx context.Context, orgID uuid.UUID) error

	// WithStripeAccountLock acquires a PG advisory lock keyed on org_id
	// for the lifetime of fn and releases it when fn returns. If the
	// store doesn't support advisory locks (test stubs), it MUST still
	// serialise concurrent calls per orgID via an in-process lock so
	// the handler's contract holds. fn's error is propagated as-is.
	WithStripeAccountLock(ctx context.Context, orgID uuid.UUID, fn func(ctx context.Context) error) error
}

// EmbeddedHandler serves the Stripe Connect Embedded Components endpoints.
// Stripe account ownership is persisted on the organizations table
// (phase R5) via OrganizationRepository — every operator of the team
// works against the same merchant account.
type EmbeddedHandler struct {
	orgs        orgAccountStore
	frontendURL string
}

// NewEmbeddedHandler wires the handler with the organization repository.
// frontendURL is the public web app URL (e.g., https://service-marketplace-go.vercel.app)
// used to build per-org profile URLs for Stripe business_profile.url.
func NewEmbeddedHandler(orgs orgAccountStore, frontendURL string) *EmbeddedHandler {
	return &EmbeddedHandler{
		orgs:        orgs,
		frontendURL: frontendURL,
	}
}

// accountSessionRequest is the body accepted when creating a session.
// Fields are required only when the user has NO existing connected account yet.
type accountSessionRequest struct {
	Country      string `json:"country"`
	BusinessType string `json:"business_type"`
}

type embeddedAccountSessionResponse struct {
	ClientSecret string `json:"client_secret"`
	AccountID    string `json:"account_id"`
	ExpiresAt    int64  `json:"expires_at"`
}

type embeddedAccountStatusResponse struct {
	AccountID                 string   `json:"account_id"`
	Country                   string   `json:"country"`
	BusinessType              string   `json:"business_type"`
	ChargesEnabled            bool     `json:"charges_enabled"`
	PayoutsEnabled            bool     `json:"payouts_enabled"`
	DetailsSubmitted          bool     `json:"details_submitted"`
	RequirementsCurrentlyDue  []string `json:"requirements_currently_due"`
	RequirementsPastDue       []string `json:"requirements_past_due"`
	RequirementsEventuallyDue []string `json:"requirements_eventually_due"`
	RequirementsPending       []string `json:"requirements_pending_verification"`
	RequirementsCount         int      `json:"requirements_count"`
	DisabledReason            string   `json:"disabled_reason,omitempty"`
}

// CreateAccountSession creates (or reuses) a Stripe Custom connected account
// for the authenticated user and returns a fresh Account Session client secret.
//
// POST /api/v1/payment-info/account-session
// Body: { "country": "FR", "business_type": "individual" }
func (h *EmbeddedHandler) CreateAccountSession(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	// Parse body — optional, only needed on first call.
	// Closes BUG-12: a malformed JSON body used to be silently swallowed
	// (the unmarshal error was discarded with `_ = ...`), leaving `req`
	// empty. The handler then surfaced "country is required" — a 500
	// that hid the real cause from the client. Empty body is still
	// allowed (the optional path), but a NON-EMPTY body that fails to
	// unmarshal now returns 400 invalid_json with a sanitized parser
	// hint so the client can fix its payload.
	req := accountSessionRequest{}
	if r.Body != nil && r.ContentLength > 0 {
		body, readErr := io.ReadAll(io.LimitReader(r.Body, 4096))
		if readErr != nil {
			slog.Warn("embedded: read body failed", "org_id", orgID, "error", readErr)
			res.Error(w, http.StatusBadRequest, "invalid_body", "failed to read request body")
			return
		}
		// Treat a body that is empty after read OR whitespace-only the
		// same as "no body": the handler's no-body path remains valid.
		if len(bytes.TrimSpace(body)) > 0 {
			if jsonErr := json.Unmarshal(body, &req); jsonErr != nil {
				slog.Warn("embedded: invalid JSON body",
					"org_id", orgID, "error", jsonErr.Error())
				res.Error(w, http.StatusBadRequest, "invalid_json", jsonErr.Error())
				return
			}
		}
	}
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Build per-org profile URL for Stripe business_profile.url,
	// adapting the path based on org type (freelancers vs agencies).
	// Stripe rejects localhost URLs, so use the production URL for dev.
	baseURL := h.frontendURL
	if strings.Contains(baseURL, "localhost") {
		baseURL = "https://service-marketplace-go.vercel.app"
	}
	role := middleware.GetRole(r.Context())
	profilePath := "freelancers"
	if role == "agency" {
		profilePath = "agencies"
	}
	profileURL := fmt.Sprintf("%s/%s/%s", baseURL, profilePath, orgID)
	accountID, err := h.resolveStripeAccount(ctx, orgID, req.Country, profileURL)
	if err != nil {
		slog.Error("embedded: resolve stripe account", "org_id", orgID, "error", err)
		// Detect Stripe cross-border country restriction and surface a
		// user-friendly 400 with a specific code.
		if strings.Contains(err.Error(), "cannot be created by platforms in") {
			res.Error(w, http.StatusBadRequest, "country_not_supported",
				"Ce pays n'est pas disponible depuis notre plateforme. Contactez notre support si vous pensez que c'est une erreur.")
			return
		}
		res.Error(w, http.StatusInternalServerError, "stripe_account_error", err.Error())
		return
	}

	sessionSecret, expiresAt, err := createOnboardingSession(accountID)
	if err != nil {
		slog.Error("embedded: create account session", "account_id", accountID, "error", err)
		res.Error(w, http.StatusInternalServerError, "stripe_session_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, embeddedAccountSessionResponse{
		ClientSecret: sessionSecret,
		AccountID:    accountID,
		ExpiresAt:    expiresAt,
	})
}

// ResetAccount deletes the test embedded account mapping for the user so
// the next session creation spawns a fresh Stripe connected account with
// new country/business_type. Only useful for the test page.
//
// DELETE /api/v1/payment-info/account-session
func (h *EmbeddedHandler) ResetAccount(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.orgs.ClearStripeAccount(ctx, orgID); err != nil {
		slog.Error("embedded: reset account", "org_id", orgID, "error", err)
		res.Error(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAccountStatus returns the current state of the user's Stripe connected
// account: capabilities, requirements, verification status.
//
// GET /api/v1/payment-info/account-status
func (h *EmbeddedHandler) GetAccountStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	accountID, _, err := h.orgs.GetStripeAccount(ctx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res.Error(w, http.StatusNotFound, "no_account", "no stripe account for this organization yet")
			return
		}
		res.Error(w, http.StatusInternalServerError, "lookup_error", err.Error())
		return
	}
	if accountID == "" {
		res.Error(w, http.StatusNotFound, "no_account", "no stripe account for this organization yet")
		return
	}

	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		slog.Error("embedded: retrieve account", "account_id", accountID, "error", err)
		res.Error(w, http.StatusInternalServerError, "stripe_error", err.Error())
		return
	}

	resp := embeddedAccountStatusResponse{
		AccountID:        acct.ID,
		Country:          acct.Country,
		ChargesEnabled:   acct.ChargesEnabled,
		PayoutsEnabled:   acct.PayoutsEnabled,
		DetailsSubmitted: acct.DetailsSubmitted,
	}
	if acct.BusinessType != "" {
		resp.BusinessType = string(acct.BusinessType)
	}
	if acct.Requirements != nil {
		resp.RequirementsCurrentlyDue = acct.Requirements.CurrentlyDue
		resp.RequirementsPastDue = acct.Requirements.PastDue
		resp.RequirementsEventuallyDue = acct.Requirements.EventuallyDue
		resp.RequirementsPending = acct.Requirements.PendingVerification
		resp.DisabledReason = string(acct.Requirements.DisabledReason)
		resp.RequirementsCount = len(acct.Requirements.CurrentlyDue) +
			len(acct.Requirements.PastDue) +
			len(acct.Requirements.EventuallyDue)
	}

	res.JSON(w, http.StatusOK, resp)
}

// resolveStripeAccount returns an existing Stripe account ID for the
// organization or creates a fresh Custom account and persists the
// mapping. When reusing an existing account, it re-applies
// business_profile pre-fill (idempotent) so fields Stripe would
// otherwise ask (URL, MCC, product description) are filled regardless
// of when the account was originally created.
//
// BUG-04: the entire check-and-create sequence now runs under a
// PG advisory lock keyed on org_id (via h.orgs.WithStripeAccountLock).
// Without the lock, two concurrent CreateAccountSession requests on
// the same org would each see "no account" and each create one — the
// second persisted overwrote the first, leaking an orphan Stripe
// account that the platform was billed for but never used.
//
// The lock scope is the whole resolveStripeAccount call (lookup +
// optional create + persist). The fast path "row already had an
// account id" still benefits from the lock: it makes
// syncBusinessProfile idempotent under contention without a window
// where two callers race the same Stripe Update.
func (h *EmbeddedHandler) resolveStripeAccount(
	ctx context.Context,
	orgID uuid.UUID,
	country, platformURL string,
) (string, error) {
	var resolvedAccountID string
	lockErr := h.orgs.WithStripeAccountLock(ctx, orgID, func(ctx context.Context) error {
		existing, _, err := h.orgs.GetStripeAccount(ctx, orgID)
		if err == nil && existing != "" {
			// Ensure business_profile is always populated on the connected
			// account so Stripe does not re-ask for website URL / MCC /
			// description. Inside the lock so a concurrent retry doesn't
			// race the same Stripe Update.
			if updErr := syncBusinessProfile(existing, platformURL); updErr != nil {
				slog.Warn("embedded: sync business_profile failed (non-fatal)",
					"account_id", existing, "error", updErr)
			}
			resolvedAccountID = existing
			return nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lookup account: %w", err)
		}

		if country == "" {
			return fmt.Errorf("country is required to create a new account")
		}

		accountID, err := createStripeCustomAccount(country, platformURL)
		if err != nil {
			return fmt.Errorf("create stripe account: %w", err)
		}

		if err := h.orgs.SetStripeAccount(ctx, orgID, accountID, country); err != nil {
			return fmt.Errorf("persist account id: %w", err)
		}
		resolvedAccountID = accountID
		return nil
	})
	if lockErr != nil {
		return "", lockErr
	}
	return resolvedAccountID, nil
}


// syncBusinessProfile updates an existing connected account's business_profile
// to ensure URL/MCC/description are always set. Idempotent.
//
// MCC 8999 = "Professional Services" — best generic match for a B2B
// marketplace of freelancers and agencies.
func syncBusinessProfile(accountID, platformURL string) error {
	_, err := account.Update(accountID, &stripe.AccountParams{
		BusinessProfile: &stripe.AccountBusinessProfileParams{
			URL:                stripe.String(platformURL),
			MCC:                stripe.String("8999"),
			ProductDescription: stripe.String("Professional services provided through our B2B marketplace platform. Clients pay upfront when a proposal is accepted, funds are held in escrow via Stripe Connect, and released to the provider upon successful delivery."),
		},
	})
	return err
}

// createStripeCustomAccount creates a Stripe Custom connected account with
// card_payments + transfers capabilities, pre-filling business_profile to
// skip redundant KYC fields (website URL, MCC, product description).
//
// FR platforms require an Account Token when creating Custom accounts where
// controller[requirement_collection]=application. business_type is NOT
// pre-filled — Stripe's Embedded onboarding asks the user directly.
func createStripeCustomAccount(country, platformURL string) (string, error) {
	tokenParams := &stripe.TokenParams{
		Account: &stripe.TokenAccountParams{
			TOSShownAndAccepted: stripe.Bool(true),
		},
	}
	tok, err := token.New(tokenParams)
	if err != nil {
		return "", fmt.Errorf("create account token: %w", err)
	}

	params := &stripe.AccountParams{
		Type:         stripe.String(string(stripe.AccountTypeCustom)),
		Country:      stripe.String(country),
		AccountToken: stripe.String(tok.ID),
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
		BusinessProfile: &stripe.AccountBusinessProfileParams{
			URL:                stripe.String(platformURL),
			MCC:                stripe.String("8999"), // Professional Services (B2B generic)
			ProductDescription: stripe.String("Professional services provided through our B2B marketplace platform. Clients pay upfront when a proposal is accepted, funds are held in escrow via Stripe Connect, and released to the provider upon successful delivery."),
		},
	}
	acct, err := account.New(params)
	if err != nil {
		return "", err
	}
	return acct.ID, nil
}

// createOnboardingSession creates a short-lived Account Session with
// account_onboarding + account_management + notification_banner components
// enabled. Used by the production payment-info-v2 page.
//
// DisableStripeUserAuthentication is true because we use Custom accounts.
// external_account_collection is true so users can edit their bank account
// (IBAN) from the AccountManagement component.
func createOnboardingSession(accountID string) (string, int64, error) {
	params := &stripe.AccountSessionParams{
		Account: stripe.String(accountID),
		Components: &stripe.AccountSessionComponentsParams{
			AccountOnboarding: &stripe.AccountSessionComponentsAccountOnboardingParams{
				Enabled: stripe.Bool(true),
				Features: &stripe.AccountSessionComponentsAccountOnboardingFeaturesParams{
					DisableStripeUserAuthentication: stripe.Bool(true),
				},
			},
			AccountManagement: &stripe.AccountSessionComponentsAccountManagementParams{
				Enabled: stripe.Bool(true),
				Features: &stripe.AccountSessionComponentsAccountManagementFeaturesParams{
					DisableStripeUserAuthentication: stripe.Bool(true),
					ExternalAccountCollection:       stripe.Bool(true),
				},
			},
			NotificationBanner: &stripe.AccountSessionComponentsNotificationBannerParams{
				Enabled: stripe.Bool(true),
				Features: &stripe.AccountSessionComponentsNotificationBannerFeaturesParams{
					DisableStripeUserAuthentication: stripe.Bool(true),
					ExternalAccountCollection:       stripe.Bool(true),
				},
			},
		},
	}
	sess, err := accountsession.New(params)
	if err != nil {
		return "", 0, err
	}
	return sess.ClientSecret, sess.ExpiresAt, nil
}
