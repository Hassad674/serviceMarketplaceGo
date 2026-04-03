package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	jobapp "marketplace-backend/internal/app/job"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type JobApplicationHandler struct {
	jobSvc *jobapp.Service
}

func NewJobApplicationHandler(svc *jobapp.Service) *JobApplicationHandler {
	return &JobApplicationHandler{jobSvc: svc}
}

func (h *JobApplicationHandler) ApplyToJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid job ID")
		return
	}

	var req request.ApplyToJobRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	app, err := h.jobSvc.ApplyToJob(r.Context(), jobapp.ApplyToJobInput{
		JobID:       jobID,
		ApplicantID: userID,
		Message:     req.Message,
		VideoURL:    req.VideoURL,
	})
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewJobApplicationResponse(app))
}

func (h *JobApplicationHandler) WithdrawApplication(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "applicationId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid application ID")
		return
	}

	if err := h.jobSvc.WithdrawApplication(r.Context(), appID, userID); err != nil {
		handleJobError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *JobApplicationHandler) ListJobApplications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid job ID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	items, nextCursor, err := h.jobSvc.ListJobApplications(r.Context(), jobID, userID, cursor, limit)
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewApplicationListResponse(items, nextCursor))
}

func (h *JobApplicationHandler) ListMyApplications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	items, nextCursor, err := h.jobSvc.ListMyApplications(r.Context(), userID, cursor, limit)
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewMyApplicationListResponse(items, nextCursor))
}

func (h *JobApplicationHandler) ContactApplicant(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid job ID")
		return
	}
	applicantID, err := uuid.Parse(chi.URLParam(r, "applicantId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid applicant ID")
		return
	}

	convID, err := h.jobSvc.ContactApplicant(r.Context(), jobID, userID, applicantID)
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.ContactApplicantResponse{
		ConversationID: convID.String(),
	})
}

func (h *JobApplicationHandler) ListOpenJobs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cursor := q.Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	filters := repository.JobListFilters{
		ApplicantType: q.Get("applicant_type"),
		BudgetType:    q.Get("budget_type"),
		Search:        q.Get("search"),
	}
	if skills := q.Get("skills"); skills != "" {
		filters.Skills = strings.Split(skills, ",")
	}
	if v := q.Get("min_budget"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filters.MinBudget = &n
		}
	}
	if v := q.Get("max_budget"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filters.MaxBudget = &n
		}
	}

	jobs, nextCursor, err := h.jobSvc.ListOpenJobs(r.Context(), filters, cursor, limit)
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewJobListResponse(jobs, nextCursor))
}

func (h *JobApplicationHandler) GetCredits(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	credits, err := h.jobSvc.GetCredits(r.Context(), userID)
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.CreditsResponse{Credits: credits})
}

func (h *JobApplicationHandler) ResetCredits(w http.ResponseWriter, r *http.Request) {
	if err := h.jobSvc.ResetWeeklyCredits(r.Context()); err != nil {
		res.Error(w, http.StatusInternalServerError, "reset_failed", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *JobApplicationHandler) HasApplied(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid job ID")
		return
	}

	applied, err := h.jobSvc.HasApplied(r.Context(), jobID, userID)
	if err != nil {
		handleJobError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.HasAppliedResponse{HasApplied: applied})
}
