package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	jobapp "marketplace-backend/internal/app/job"
	jobdomain "marketplace-backend/internal/domain/job"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type JobHandler struct {
	jobSvc *jobapp.Service
}

func NewJobHandler(svc *jobapp.Service) *JobHandler {
	return &JobHandler{jobSvc: svc}
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	var req request.CreateJobRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	j, err := h.jobSvc.CreateJob(r.Context(), jobapp.CreateJobInput{
		CreatorID:        userID,
		Title:            req.Title,
		Description:      req.Description,
		Skills:           req.Skills,
		ApplicantType:    req.ApplicantType,
		BudgetType:       req.BudgetType,
		MinBudget:        req.MinBudget,
		MaxBudget:        req.MaxBudget,
		PaymentFrequency: req.PaymentFrequency,
		DurationWeeks:    req.DurationWeeks,
		IsIndefinite:     req.IsIndefinite,
		DescriptionType:  req.DescriptionType,
		VideoURL:         req.VideoURL,
	})
	if err != nil {
		handleJobError(w, err)
		return
	}
	res.JSON(w, http.StatusCreated, response.NewJobResponse(j))
}

func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_job_id", "id must be a valid UUID")
		return
	}
	j, err := h.jobSvc.GetJob(r.Context(), jobID)
	if err != nil {
		handleJobError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewJobResponse(j))
}

// ListMyJobs returns the jobs belonging to the authenticated user's
// organization. All operators of the same org see the same list —
// the Stripe Dashboard shared-workspace model.
func (h *JobHandler) ListMyJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}
	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	jobs, nextCursor, err := h.jobSvc.ListOrgJobsWithCounts(r.Context(), orgID, userID, cursorStr, limit)
	if err != nil {
		handleJobError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewJobWithCountsListResponse(jobs, nextCursor))
}

func (h *JobHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_job_id", "id must be a valid UUID")
		return
	}
	var req request.CreateJobRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	j, err := h.jobSvc.UpdateJob(r.Context(), jobapp.UpdateJobInput{
		JobID:            jobID,
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		Skills:           req.Skills,
		ApplicantType:    req.ApplicantType,
		BudgetType:       req.BudgetType,
		MinBudget:        req.MinBudget,
		MaxBudget:        req.MaxBudget,
		PaymentFrequency: req.PaymentFrequency,
		DurationWeeks:    req.DurationWeeks,
		IsIndefinite:     req.IsIndefinite,
		DescriptionType:  req.DescriptionType,
		VideoURL:         req.VideoURL,
	})
	if err != nil {
		handleJobError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewJobResponse(j))
}

func (h *JobHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_job_id", "id must be a valid UUID")
		return
	}
	if err := h.jobSvc.DeleteJob(r.Context(), jobID, userID); err != nil {
		handleJobError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobHandler) MarkApplicationsViewed(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_job_id", "id must be a valid UUID")
		return
	}
	if err := h.jobSvc.MarkApplicationsViewed(r.Context(), jobID, userID); err != nil {
		handleJobError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobHandler) CloseJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_job_id", "id must be a valid UUID")
		return
	}
	if err := h.jobSvc.CloseJob(r.Context(), jobID, userID); err != nil {
		handleJobError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (h *JobHandler) ReopenJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_job_id", "id must be a valid UUID")
		return
	}
	if err := h.jobSvc.ReopenJob(r.Context(), jobID, userID); err != nil {
		handleJobError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "open"})
}

func handleJobError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, jobdomain.ErrJobNotFound):
		res.Error(w, http.StatusNotFound, "job_not_found", err.Error())
	case errors.Is(err, jobdomain.ErrNotOwner):
		res.Error(w, http.StatusForbidden, "not_owner", err.Error())
	case errors.Is(err, jobdomain.ErrAlreadyClosed):
		res.Error(w, http.StatusConflict, "already_closed", err.Error())
	case errors.Is(err, jobdomain.ErrAlreadyOpen):
		res.Error(w, http.StatusConflict, "already_open", err.Error())
	case errors.Is(err, jobdomain.ErrUnauthorizedRole):
		res.Error(w, http.StatusForbidden, "unauthorized_role", err.Error())
	case errors.Is(err, jobdomain.ErrEmptyTitle):
		res.Error(w, http.StatusBadRequest, "empty_title", err.Error())
	case errors.Is(err, jobdomain.ErrTitleTooLong):
		res.Error(w, http.StatusBadRequest, "title_too_long", err.Error())
	case errors.Is(err, jobdomain.ErrEmptyDescription):
		res.Error(w, http.StatusBadRequest, "empty_description", err.Error())
	case errors.Is(err, jobdomain.ErrTooManySkills):
		res.Error(w, http.StatusBadRequest, "too_many_skills", err.Error())
	case errors.Is(err, jobdomain.ErrInvalidApplicantType):
		res.Error(w, http.StatusBadRequest, "invalid_applicant_type", err.Error())
	case errors.Is(err, jobdomain.ErrInvalidBudgetType):
		res.Error(w, http.StatusBadRequest, "invalid_budget_type", err.Error())
	case errors.Is(err, jobdomain.ErrInvalidBudget):
		res.Error(w, http.StatusBadRequest, "invalid_budget", err.Error())
	case errors.Is(err, jobdomain.ErrMinExceedsMax):
		res.Error(w, http.StatusBadRequest, "min_exceeds_max", err.Error())
	case errors.Is(err, jobdomain.ErrInvalidPaymentFrequency):
		res.Error(w, http.StatusBadRequest, "invalid_payment_frequency", err.Error())
	case errors.Is(err, jobdomain.ErrInvalidDescriptionType):
		res.Error(w, http.StatusBadRequest, "invalid_description_type", err.Error())
	case errors.Is(err, jobdomain.ErrVideoURLRequired):
		res.Error(w, http.StatusBadRequest, "video_url_required", err.Error())
	case errors.Is(err, jobdomain.ErrApplicationNotFound):
		res.Error(w, http.StatusNotFound, "application_not_found", err.Error())
	case errors.Is(err, jobdomain.ErrAlreadyApplied):
		res.Error(w, http.StatusConflict, "already_applied", err.Error())
	case errors.Is(err, jobdomain.ErrCannotApplyToOwnJob):
		res.Error(w, http.StatusForbidden, "cannot_apply_own_job", err.Error())
	case errors.Is(err, jobdomain.ErrCannotApplyToClosed):
		res.Error(w, http.StatusBadRequest, "job_closed", err.Error())
	case errors.Is(err, jobdomain.ErrNotApplicant):
		res.Error(w, http.StatusForbidden, "not_applicant", err.Error())
	case errors.Is(err, jobdomain.ErrApplicantTypeMismatch):
		res.Error(w, http.StatusForbidden, "applicant_type_mismatch", err.Error())
	case errors.Is(err, jobdomain.ErrEmptyApplicationMessage):
		res.Error(w, http.StatusBadRequest, "empty_message", err.Error())
	case errors.Is(err, jobdomain.ErrApplicationMessageTooLong):
		res.Error(w, http.StatusBadRequest, "message_too_long", err.Error())
	case errors.Is(err, jobdomain.ErrNoCreditsLeft):
		res.Error(w, http.StatusTooManyRequests, "no_credits_left", err.Error())
	case errors.Is(err, userdomain.ErrKYCRestricted):
		res.Error(w, http.StatusForbidden, "kyc_restricted", "Your account is restricted. Set up your payment info to lift this restriction.")
	default:
		slog.Error("unhandled job error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
