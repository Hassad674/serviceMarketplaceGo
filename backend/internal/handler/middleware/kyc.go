package middleware

import (
	"net/http"
	"time"

	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// RequireKYCCompliant blocks providers/agencies whose organization has
// earned available funds but hasn't completed Stripe KYC within 14
// days. Must be chained AFTER middleware.Auth. Since phase R5 the KYC
// state lives on the organization (the merchant of record), so the
// check resolves the caller's org and inspects its KYC deadline.
//
// Enterprises pass through unconditionally (they pay, they don't receive).
// Orgs with no earnings pass through (no KYC deadline started).
// Orgs that have completed KYC pass through.
//
// Narrowed to OrganizationReader: the middleware only ever calls
// FindByID. Passing the wide OrganizationRepository at wiring time
// still satisfies the interface via structural typing.
func RequireKYCCompliant(orgs repository.OrganizationReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			// Only providers and agencies can be KYC-blocked
			if role != "provider" && role != "agency" {
				next.ServeHTTP(w, r)
				return
			}

			orgID, ok := GetOrganizationID(r.Context())
			if !ok {
				res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
				return
			}

			org, err := orgs.FindByID(r.Context(), orgID)
			if err != nil {
				// Fail open: if we can't check, let the request through
				// rather than blocking legitimate operators.
				next.ServeHTTP(w, r)
				return
			}

			if org.IsKYCBlocked() {
				deadline := org.KYCFirstEarningAt.Add(14 * 24 * time.Hour)
				res.JSON(w, http.StatusForbidden, map[string]any{
					"error": map[string]any{
						"code":    "kyc_restricted",
						"message": "Your account is restricted. Set up your payment info to lift this restriction.",
					},
					"kyc_deadline": deadline.Format(time.RFC3339),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
