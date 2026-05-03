package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// SearchPinger is the narrow contract the health handler needs
// from the Typesense client. Nil disables the check — useful for
// unit tests and the rare worktree that boots without a Typesense
// dependency.
type SearchPinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db             *sql.DB
	searchPinger   SearchPinger
	searchRequired bool
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// WithSearchPinger attaches a Typesense client to the health
// check. Since phase 4 the query path has no SQL fallback, so any
// production deployment MUST pass `required=true` — a failed ping
// takes /ready red so load balancers rotate the instance out.
// The argument is still exposed so tests (and worktrees that
// intentionally boot without Typesense) can opt into a soft check.
func (h *HealthHandler) WithSearchPinger(pinger SearchPinger, required bool) *HealthHandler {
	h.searchPinger = pinger
	h.searchRequired = required
	return h
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	res.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		res.Error(w, http.StatusServiceUnavailable, "not_ready", "database is not reachable")
		return
	}
	if h.searchPinger != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.searchPinger.Ping(ctx); err != nil && h.searchRequired {
			res.Error(w, http.StatusServiceUnavailable, "not_ready", "search engine is not reachable")
			return
		}
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{"backend": "ok", "database": "error"}
	if err := h.db.PingContext(r.Context()); err == nil {
		result["database"] = "ok"
	}
	res.JSON(w, http.StatusOK, result)
}

func (h *HealthHandler) GetWords(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, word, created_at FROM test_words ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()

	type WordItem struct {
		ID        string `json:"id"`
		Word      string `json:"word"`
		CreatedAt string `json:"created_at"`
	}
	words := []WordItem{}
	for rows.Next() {
		var wi WordItem
		var t time.Time
		if err := rows.Scan(&wi.ID, &wi.Word, &t); err != nil {
			continue
		}
		wi.CreatedAt = t.Format(time.RFC3339)
		words = append(words, wi)
	}
	res.JSON(w, http.StatusOK, map[string]any{"words": words})
}

func (h *HealthHandler) AddWord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word string `json:"word"`
	}
	// F.5 B1: bound + reject unknown fields. Test endpoint, single-word.
	if err := jsondec.DecodeBody(w, r, &req, 1<<10); err != nil || req.Word == "" {
		res.Error(w, http.StatusBadRequest, "invalid_request", "word is required")
		return
	}
	_, err := h.db.ExecContext(r.Context(), "INSERT INTO test_words (word) VALUES ($1)", req.Word)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	res.JSON(w, http.StatusCreated, map[string]string{"status": "ok", "word": req.Word})
}

