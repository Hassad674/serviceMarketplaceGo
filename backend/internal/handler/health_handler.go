package handler

import (
	"database/sql"
	"net/http"

	res "marketplace-backend/pkg/response"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	res.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		res.Error(w, http.StatusServiceUnavailable, "not_ready", "database is not reachable")
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
