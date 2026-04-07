package handler

import (
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

// userAccountStore is the trimmed view of UserRepository this handler needs.
// Defined here so tests can stub it without pulling the full repository.
type userAccountStore interface {
	GetStripeAccount(ctx context.Context, userID uuid.UUID) (accountID, country string, err error)
	SetStripeAccount(ctx context.Context, userID uuid.UUID, accountID, country string) error
	ClearStripeAccount(ctx context.Context, userID uuid.UUID) error
}

// EmbeddedHandler serves the Stripe Connect Embedded Components endpoints.
// Stripe account ownership is persisted on the users table (migration 040)
// via the UserRepository — no per-handler table required.
type EmbeddedHandler struct {
	users       userAccountStore
	frontendURL string
}

// NewEmbeddedHandler wires the handler with the user repository.
// frontendURL is the public web app URL (e.g., https://service-marketplace-go.vercel.app)
// used to build per-user profile URLs for Stripe business_profile.url.
func NewEmbeddedHandler(users userAccountStore, frontendURL string) *EmbeddedHandler {
	return &EmbeddedHandler{
		users:       users,
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
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	// Parse body — optional, only needed on first call
	req := accountSessionRequest{}
	if r.Body != nil && r.ContentLength > 0 {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		_ = json.Unmarshal(body, &req)
	}
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Build per-user profile URL for Stripe business_profile.url
	// adapting the path based on user role (freelancers vs agencies)
	role := middleware.GetRole(r.Context())
	profilePath := "freelancers"
	if role == "agency" {
		profilePath = "agencies"
	}
	profileURL := fmt.Sprintf("%s/%s/%s", h.frontendURL, profilePath, userID)
	accountID, err := h.resolveStripeAccount(ctx, userID, req.Country, profileURL)
	if err != nil {
		slog.Error("embedded: resolve stripe account", "user_id", userID, "error", err)
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
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.users.ClearStripeAccount(ctx, userID); err != nil {
		slog.Error("embedded: reset account", "user_id", userID, "error", err)
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
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	accountID, _, err := h.users.GetStripeAccount(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res.Error(w, http.StatusNotFound, "no_account", "no stripe account for this user yet")
			return
		}
		res.Error(w, http.StatusInternalServerError, "lookup_error", err.Error())
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

// resolveStripeAccount returns an existing Stripe account ID for the user
// or creates a fresh Custom account and persists the mapping. When reusing
// an existing account, it re-applies business_profile pre-fill (idempotent)
// so fields Stripe would otherwise ask (URL, MCC, product description) are
// filled regardless of when the account was originally created.
func (h *EmbeddedHandler) resolveStripeAccount(
	ctx context.Context,
	userID uuid.UUID,
	country, platformURL string,
) (string, error) {
	existing, _, err := h.users.GetStripeAccount(ctx, userID)
	if err == nil && existing != "" {
		// Ensure business_profile is always populated on the connected account
		// so Stripe does not re-ask for website URL / MCC / description.
		if updErr := syncBusinessProfile(existing, platformURL); updErr != nil {
			slog.Warn("embedded: sync business_profile failed (non-fatal)",
				"account_id", existing, "error", updErr)
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("lookup account: %w", err)
	}

	if country == "" {
		return "", fmt.Errorf("country is required to create a new account")
	}

	accountID, err := createStripeCustomAccount(country, platformURL)
	if err != nil {
		return "", fmt.Errorf("create stripe account: %w", err)
	}

	if err := h.users.SetStripeAccount(ctx, userID, accountID, country); err != nil {
		return "", fmt.Errorf("persist account id: %w", err)
	}
	return accountID, nil
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
			ProductDescription: stripe.String("Freelance and agency services provided via the marketplace platform."),
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
			ProductDescription: stripe.String("Freelance and agency services provided via the marketplace platform."),
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
