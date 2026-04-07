package middleware

import (
	"net/http"
	"time"

	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// RequireKYCCompliant blocks providers/agencies who have earned available
// funds but haven't completed Stripe KYC within 14 days. Must be chained
// AFTER middleware.Auth.
//
// Enterprises pass through unconditionally (they pay, they don't receive).
// Users with no earnings pass through (no KYC deadline started).
// Users who completed KYC pass through.
func RequireKYCCompliant(users repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			// Only providers and agencies can be KYC-blocked
			if role != "provider" && role != "agency" {
				next.ServeHTTP(w, r)
				return
			}

			userID, ok := GetUserID(r.Context())
			if !ok {
				res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
				return
			}

			u, err := users.GetByID(r.Context(), userID)
			if err != nil {
				// Fail open: if we can't check, let the request through
				// rather than blocking legitimate users.
				next.ServeHTTP(w, r)
				return
			}

			if u.IsKYCBlocked() {
				deadline := u.KYCFirstEarningAt.Add(14 * 24 * time.Hour)
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
