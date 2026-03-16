package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"marketplace-backend/pkg/response"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
					"request_id", GetRequestID(r.Context()),
				)
				response.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
