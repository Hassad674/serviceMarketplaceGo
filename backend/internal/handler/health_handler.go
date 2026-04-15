package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	res "marketplace-backend/pkg/response"
)

// SearchPinger is the narrow contract the health handler needs
// from the Typesense client. A nil implementation disables the
// Typesense check (useful when SEARCH_ENGINE=sql).
type SearchPinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db            *sql.DB
	searchPinger  SearchPinger
	searchRequired bool
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// WithSearchPinger attaches a Typesense client to the health
// check. The `required` flag controls whether a failed Typesense
// ping is treated as fatal (503) or as a best-effort signal:
//   - required=true when SEARCH_ENGINE=typesense — the query
//     path depends on Typesense being healthy, so a failure is
//     an outage.
//   - required=false when SEARCH_ENGINE=sql — the query path
//     falls back to Postgres, so Typesense can be flaky without
//     taking the whole backend out of rotation.
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Word == "" {
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

