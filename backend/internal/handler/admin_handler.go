package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	adminapp "marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

type AdminHandler struct {
	svc *adminapp.Service
}

func NewAdminHandler(svc *adminapp.Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// GetDashboardStats handles GET /api/v1/admin/dashboard/stats.
func (h *AdminHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetDashboardStats(r.Context())
	if err != nil {
		slog.Error("admin dashboard stats", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get dashboard stats")
		return
	}

	recentSignups := make([]response.RecentSignupResponse, 0, len(stats.RecentSignups))
	for _, u := range stats.RecentSignups {
		recentSignups = append(recentSignups, response.NewRecentSignupResponse(u))
	}

	res.JSON(w, http.StatusOK, response.DashboardStatsResponse{
		TotalUsers:         stats.TotalUsers,
		UsersByRole:        stats.UsersByRole,
		ActiveUsers:        stats.ActiveUsers,
		SuspendedUsers:     stats.SuspendedUsers,
		BannedUsers:        stats.BannedUsers,
		TotalProposals:     stats.TotalProposals,
		ActiveProposals:    stats.ActiveProposals,
		TotalJobs:          stats.TotalJobs,
		OpenJobs:           stats.OpenJobs,
		TotalOrganizations: stats.TotalOrganizations,
		PendingInvitations: stats.PendingInvitations,
		RecentSignups:      recentSignups,
	})
}

// ListUsers handles GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))

	filters := repository.AdminUserFilters{
		Role:     r.URL.Query().Get("role"),
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
		Cursor:   r.URL.Query().Get("cursor"),
		Limit:    limit,
		Page:     page,
		Reported: r.URL.Query().Get("reported") == "true",
	}

	users, nextCursor, total, err := h.svc.ListUsers(r.Context(), filters)
	if err != nil {
		slog.Error("admin list users", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}

	items := make([]response.AdminUserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, response.NewAdminUserResponse(u))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// GetUser handles GET /api/v1/admin/users/{id}.
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	u, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewAdminUserResponse(u),
	})
}

// SuspendUser handles POST /api/v1/admin/users/{id}/suspend.
func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var body struct {
		Reason    string  `json:"reason"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if body.Reason == "" {
		res.Error(w, http.StatusBadRequest, "validation_error", "reason is required")
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "validation_error", "expires_at must be RFC3339 format")
			return
		}
		expiresAt = &t
	}

	// BUG-NEW-09 — pass the AUTHENTICATED admin's user id from JWT
	// context as the audit actor; the URL-derived `id` is the target
	// being suspended.
	adminID, _ := middleware.GetUserID(r.Context())
	if err := h.svc.SuspendUser(r.Context(), adminID, id, body.Reason, expiresAt); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user suspended"})
}

// UnsuspendUser handles POST /api/v1/admin/users/{id}/unsuspend.
func (h *AdminHandler) UnsuspendUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	adminID, _ := middleware.GetUserID(r.Context())
	if err := h.svc.UnsuspendUser(r.Context(), adminID, id); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user unsuspended"})
}

// BanUser handles POST /api/v1/admin/users/{id}/ban.
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if body.Reason == "" {
		res.Error(w, http.StatusBadRequest, "validation_error", "reason is required")
		return
	}

	// BUG-NEW-09 — actor=admin (from JWT), resource=URL id.
	adminID, _ := middleware.GetUserID(r.Context())
	if err := h.svc.BanUser(r.Context(), adminID, id, body.Reason); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user banned"})
}

// UnbanUser handles POST /api/v1/admin/users/{id}/unban.
func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	adminID, _ := middleware.GetUserID(r.Context())
	if err := h.svc.UnbanUser(r.Context(), adminID, id); err != nil {
		handleAdminError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "user unbanned"})
}

// ListConversations handles GET /api/v1/admin/conversations.
func (h *AdminHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))
	sort := r.URL.Query().Get("sort")
	filter := r.URL.Query().Get("filter")

	conversations, nextCursor, total, err := h.svc.ListConversations(r.Context(), cursorStr, limit, page, sort, filter)
	if err != nil {
		slog.Error("admin list conversations", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list conversations")
		return
	}

	items := make([]response.AdminConversationResponse, 0, len(conversations))
	for _, c := range conversations {
		items = append(items, response.NewAdminConversationResponse(c))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// GetConversation handles GET /api/v1/admin/conversations/{id}.
func (h *AdminHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	conv, err := h.svc.GetConversation(r.Context(), id)
	if err != nil {
		slog.Error("admin get conversation", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get conversation")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewAdminConversationResponse(*conv),
	})
}

// GetConversationMessages handles GET /api/v1/admin/conversations/{id}/messages.
func (h *AdminHandler) GetConversationMessages(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 50)

	messages, nextCursor, err := h.svc.GetConversationMessages(r.Context(), id, cursorStr, limit)
	if err != nil {
		slog.Error("admin get conversation messages", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get messages")
		return
	}

	items := make([]response.AdminMessageResponse, 0, len(messages))
	for _, m := range messages {
		items = append(items, response.NewAdminMessageResponse(m))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// ListConversationReports handles GET /api/v1/admin/conversations/{id}/reports.
func (h *AdminHandler) ListConversationReports(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	reports, err := h.svc.ListConversationReports(r.Context(), id)
	if err != nil {
		slog.Error("admin list conversation reports", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reports")
		return
	}

	items := make([]response.AdminReportResponse, 0, len(reports))
	for _, rp := range reports {
		items = append(items, response.NewAdminReportResponse(rp))
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": items})
}

// ListUserReports handles GET /api/v1/admin/users/{id}/reports.
func (h *AdminHandler) ListUserReports(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	against, filed, err := h.svc.ListUserReports(r.Context(), id)
	if err != nil {
		slog.Error("admin list user reports", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reports")
		return
	}

	againstResp := make([]response.AdminReportResponse, 0, len(against))
	for _, rp := range against {
		againstResp = append(againstResp, response.NewAdminReportResponse(rp))
	}
	filedResp := make([]response.AdminReportResponse, 0, len(filed))
	for _, rp := range filed {
		filedResp = append(filedResp, response.NewAdminReportResponse(rp))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"reports_against": againstResp,
		"reports_filed":   filedResp,
	})
}

// ResolveReport handles POST /api/v1/admin/reports/{id}/resolve.
func (h *AdminHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var body struct {
		Status    string `json:"status"`
		AdminNote string `json:"admin_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if body.Status != "resolved" && body.Status != "dismissed" {
		res.Error(w, http.StatusBadRequest, "validation_error", "status must be resolved or dismissed")
		return
	}
	if body.AdminNote == "" {
		res.Error(w, http.StatusBadRequest, "validation_error", "admin_note is required")
		return
	}

	adminID, _ := middleware.GetUserID(r.Context())

	if err := h.svc.ResolveReport(r.Context(), id, body.Status, body.AdminNote, adminID); err != nil {
		slog.Error("admin resolve report", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to resolve report")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{"message": "report resolved"})
}

// ListJobs handles GET /api/v1/admin/jobs.
func (h *AdminHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	filter := r.URL.Query().Get("filter")
	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))

	jobs, nextCursor, total, err := h.svc.ListJobs(r.Context(), status, search, sort, filter, cursorStr, limit, page)
	if err != nil {
		slog.Error("admin list jobs", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list jobs")
		return
	}

	items := make([]response.AdminJobResponse, 0, len(jobs))
	for _, j := range jobs {
		items = append(items, response.NewAdminJobResponse(j))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// GetAdminJob handles GET /api/v1/admin/jobs/{id}.
func (h *AdminHandler) GetAdminJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		slog.Error("admin get job", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get job")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewAdminJobResponse(*job),
	})
}

// DeleteAdminJob handles DELETE /api/v1/admin/jobs/{id}.
func (h *AdminHandler) DeleteAdminJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.DeleteJob(r.Context(), id); err != nil {
		slog.Error("admin delete job", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to delete job")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListJobApplications handles GET /api/v1/admin/job-applications.
func (h *AdminHandler) ListJobApplications(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	filter := r.URL.Query().Get("filter")
	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))

	apps, nextCursor, total, err := h.svc.ListJobApplications(r.Context(), jobID, search, sort, filter, cursorStr, limit, page)
	if err != nil {
		slog.Error("admin list job applications", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list applications")
		return
	}

	items := make([]response.AdminJobApplicationResponse, 0, len(apps))
	for _, a := range apps {
		items = append(items, response.NewAdminJobApplicationResponse(a))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// ListJobReports handles GET /api/v1/admin/jobs/{id}/reports.
func (h *AdminHandler) ListJobReports(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	reports, err := h.svc.ListJobReports(r.Context(), id)
	if err != nil {
		slog.Error("admin list job reports", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reports")
		return
	}

	items := make([]response.AdminReportResponse, 0, len(reports))
	for _, rp := range reports {
		items = append(items, response.NewAdminReportResponse(rp))
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": items})
}

// DeleteJobApplication handles DELETE /api/v1/admin/job-applications/{id}.
func (h *AdminHandler) DeleteJobApplication(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminUserID(r)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.DeleteJobApplication(r.Context(), id); err != nil {
		slog.Error("admin delete job application", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to delete application")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseAdminUserID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

func parsePage(s string) int {
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil || val < 1 {
		return 0
	}
	return val
}

func totalPages(total int, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}

func handleAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		res.Error(w, http.StatusNotFound, "user_not_found", err.Error())
	default:
		slog.Error("unhandled admin error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
