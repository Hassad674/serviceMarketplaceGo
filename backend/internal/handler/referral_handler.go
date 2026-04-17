package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

// ReferralHandler wires HTTP routes for the business-referral feature.
type ReferralHandler struct {
	svc *referralapp.Service
}

func NewReferralHandler(svc *referralapp.Service) *ReferralHandler {
	return &ReferralHandler{svc: svc}
}

// Create handles POST /api/v1/referrals
func (h *ReferralHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.CreateReferralRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_provider_id", "invalid provider id")
		return
	}
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_client_id", "invalid client id")
		return
	}

	var toggles *referralapp.SnapshotToggles
	if req.SnapshotToggles != nil {
		toggles = &referralapp.SnapshotToggles{
			IncludeExpertise:    req.SnapshotToggles.IncludeExpertise,
			IncludeExperience:   req.SnapshotToggles.IncludeExperience,
			IncludeRating:       req.SnapshotToggles.IncludeRating,
			IncludePricing:      req.SnapshotToggles.IncludePricing,
			IncludeRegion:       req.SnapshotToggles.IncludeRegion,
			IncludeLanguages:    req.SnapshotToggles.IncludeLanguages,
			IncludeAvailability: req.SnapshotToggles.IncludeAvailability,
		}
	}

	created, err := h.svc.CreateIntro(r.Context(), referralapp.CreateIntroInput{
		ReferrerID:           userID,
		ProviderID:           providerID,
		ClientID:             clientID,
		RatePct:              req.RatePct,
		DurationMonths:       req.DurationMonths,
		IntroMessageProvider: req.IntroMessageProvider,
		IntroMessageClient:   req.IntroMessageClient,
		SnapshotToggles:      toggles,
	})
	if err != nil {
		handleReferralError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewReferralResponse(created, userID))
}

// Get handles GET /api/v1/referrals/{id}
func (h *ReferralHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	referralID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid referral id")
		return
	}

	found, err := h.svc.GetByID(r.Context(), referralID, userID)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewReferralResponse(found, userID))
}

// ListMine handles GET /api/v1/referrals/me (referrer view).
func (h *ReferralHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	filter := filterFromQuery(r)
	rows, next, err := h.svc.ListByReferrer(r.Context(), userID, filter)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewReferralListResponse(rows, next, userID))
}

// ListIncoming handles GET /api/v1/referrals/incoming — the union of
// intros where the requesting user is either the provider party or the
// client party.
func (h *ReferralHandler) ListIncoming(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	filter := filterFromQuery(r)
	incomingProv, _, err := h.svc.ListIncomingForProvider(r.Context(), userID, filter)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	incomingCli, _, err := h.svc.ListIncomingForClient(r.Context(), userID, filter)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	// Merge (simple concatenation — UIs dedupe client-side if needed).
	merged := append(incomingProv, incomingCli...)
	res.JSON(w, http.StatusOK, response.NewReferralListResponse(merged, "", userID))
}

// Respond handles POST /api/v1/referrals/{id}/respond. The server infers
// the actor role from the viewer vs the referral parties, then dispatches
// to the right app-service method.
func (h *ReferralHandler) Respond(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	referralID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid referral id")
		return
	}

	var req request.RespondReferralRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	// Resolve the actor role by reading the referral and matching the JWT
	// against its parties. Ownership check lives inside the service but we
	// need the role here to pick the right method.
	found, err := h.svc.GetByID(r.Context(), referralID, userID)
	if err != nil {
		handleReferralError(w, err)
		return
	}

	// Lifecycle actions cancel/terminate are referrer-only and bypass the
	// respond/negotiate flow entirely.
	switch req.Action {
	case "cancel":
		updated, err := h.svc.Cancel(r.Context(), referralID, userID)
		if err != nil {
			handleReferralError(w, err)
			return
		}
		res.JSON(w, http.StatusOK, response.NewReferralResponse(updated, userID))
		return
	case "terminate":
		updated, err := h.svc.Terminate(r.Context(), referralID, userID)
		if err != nil {
			handleReferralError(w, err)
			return
		}
		res.JSON(w, http.StatusOK, response.NewReferralResponse(updated, userID))
		return
	}

	action, ok := negotiationActionFromString(req.Action)
	if !ok {
		res.Error(w, http.StatusBadRequest, "invalid_action", "action must be one of accept/reject/negotiate/cancel/terminate")
		return
	}
	input := referralapp.NewResponseInput(referralID, userID, action, req.NewRatePct, req.Message)

	var (
		updated *referral.Referral
		errDisp error
	)
	switch {
	case userID == found.ProviderID:
		updated, errDisp = h.svc.RespondAsProvider(r.Context(), input)
	case userID == found.ReferrerID:
		updated, errDisp = h.svc.RespondAsReferrer(r.Context(), input)
	case userID == found.ClientID:
		updated, errDisp = h.svc.RespondAsClient(r.Context(), input)
	default:
		res.Error(w, http.StatusForbidden, "forbidden", "not a party to this referral")
		return
	}
	if errDisp != nil {
		handleReferralError(w, errDisp)
		return
	}
	res.JSON(w, http.StatusOK, response.NewReferralResponse(updated, userID))
}

// ListNegotiations handles GET /api/v1/referrals/{id}/negotiations
func (h *ReferralHandler) ListNegotiations(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	referralID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid referral id")
		return
	}
	// Ownership check by loading the referral first.
	if _, err := h.svc.GetByID(r.Context(), referralID, userID); err != nil {
		handleReferralError(w, err)
		return
	}
	rows, err := h.svc.ListNegotiations(r.Context(), referralID)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewNegotiationList(rows))
}

// ListAttributions handles GET /api/v1/referrals/{id}/attributions.
// Returns the proposals attributed during the exclusivity window with
// the proposal title + status and commission aggregates. Commission
// amounts are stripped from the client's DTO (Modèle A).
func (h *ReferralHandler) ListAttributions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	referralID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid referral id")
		return
	}

	// Resolve parties for the client-viewer check used by the DTO.
	parent, err := h.svc.GetByID(r.Context(), referralID, userID)
	if err != nil {
		handleReferralError(w, err)
		return
	}

	rows, err := h.svc.ListAttributionsWithStats(r.Context(), referralID, userID)
	if err != nil {
		handleReferralError(w, err)
		return
	}

	// Map the app-level struct onto the handler-level shape (same
	// field names / types — compile-time safe via the shared alias).
	mapped := make([]struct {
		Attribution               *referral.Attribution
		ProposalTitle             string
		ProposalStatus            string
		TotalCommissionCents      int64
		PendingCommissionCents    int64
		ClawedBackCommissionCents int64
		EscrowCommissionCents     int64
		MilestonesPaid            int
		MilestonesPending         int
		MilestonesTotal           int
	}, 0, len(rows))
	for _, a := range rows {
		mapped = append(mapped, struct {
			Attribution               *referral.Attribution
			ProposalTitle             string
			ProposalStatus            string
			TotalCommissionCents      int64
			PendingCommissionCents    int64
			ClawedBackCommissionCents int64
			EscrowCommissionCents     int64
			MilestonesPaid            int
			MilestonesPending         int
			MilestonesTotal           int
		}{
			Attribution:               a.Attribution,
			ProposalTitle:             a.ProposalTitle,
			ProposalStatus:            a.ProposalStatus,
			TotalCommissionCents:      a.TotalCommissionCents,
			PendingCommissionCents:    a.PendingCommissionCents,
			ClawedBackCommissionCents: a.ClawedBackCommissionCents,
			EscrowCommissionCents:     a.EscrowCommissionCents,
			MilestonesPaid:            a.MilestonesPaid,
			MilestonesPending:         a.MilestonesPending,
			MilestonesTotal:           a.MilestonesTotal,
		})
	}
	res.JSON(w, http.StatusOK, response.NewAttributionListFromStats(mapped, userID, parent.ClientID))
}

// ListCommissions handles GET /api/v1/referrals/{id}/commissions.
// Reserved for the apporteur and the provider party — the client is
// blocked with 403 so there is no way (even via the URL) to peek at
// commission amounts.
func (h *ReferralHandler) ListCommissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	referralID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid referral id")
		return
	}

	parent, err := h.svc.GetByID(r.Context(), referralID, userID)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	if userID == parent.ClientID {
		res.Error(w, http.StatusForbidden, "forbidden", "clients cannot read commissions")
		return
	}

	rows, err := h.svc.ListCommissionsByReferral(r.Context(), referralID, userID)
	if err != nil {
		handleReferralError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, response.NewCommissionList(rows))
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func filterFromQuery(r *http.Request) repository.ReferralListFilter {
	q := r.URL.Query()
	filter := repository.ReferralListFilter{
		Cursor: q.Get("cursor"),
	}
	if statuses, ok := q["status"]; ok {
		for _, s := range statuses {
			filter.Statuses = append(filter.Statuses, referral.Status(s))
		}
	}
	return filter
}

func negotiationActionFromString(s string) (referral.NegotiationAction, bool) {
	switch s {
	case "accept":
		return referral.NegoActionAccepted, true
	case "reject":
		return referral.NegoActionRejected, true
	case "negotiate", "counter":
		return referral.NegoActionCountered, true
	case "propose":
		return referral.NegoActionProposed, true
	}
	return "", false
}

func handleReferralError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, referral.ErrNotFound):
		res.Error(w, http.StatusNotFound, "referral_not_found", err.Error())
	case errors.Is(err, referral.ErrAttributionNotFound):
		res.Error(w, http.StatusNotFound, "attribution_not_found", err.Error())
	case errors.Is(err, referral.ErrCommissionNotFound):
		res.Error(w, http.StatusNotFound, "commission_not_found", err.Error())
	case errors.Is(err, referral.ErrNotAuthorized):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, referral.ErrCoupleLocked):
		res.Error(w, http.StatusConflict, "referral_couple_locked", err.Error())
	case errors.Is(err, referral.ErrSelfReferral),
		errors.Is(err, referral.ErrSameOrganization),
		errors.Is(err, referral.ErrInvalidProviderRole),
		errors.Is(err, referral.ErrInvalidClientRole),
		errors.Is(err, referral.ErrReferrerRequired),
		errors.Is(err, referral.ErrRateOutOfRange),
		errors.Is(err, referral.ErrDurationOutOfRange),
		errors.Is(err, referral.ErrEmptyMessage),
		errors.Is(err, referral.ErrMessageTooLong),
		errors.Is(err, referral.ErrSnapshotInvalid):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, referral.ErrInvalidTransition),
		errors.Is(err, referral.ErrAlreadyTerminal):
		res.Error(w, http.StatusConflict, "invalid_transition", err.Error())
	default:
		slog.Error("referral handler error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
