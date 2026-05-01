package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/middleware"

	res "marketplace-backend/pkg/response"
)

// ProposalHandler is the legacy wide handler. It now acts as a thin
// composition facade over four focused handlers (lifecycle / payment /
// completion / admin) introduced by the Phase 3 SOLID decomposition.
//
// SRP rationale (Phase 3): the original handler bundled 22 methods
// across 4 distinct surfaces (audit QUAL-B SRP). The decomposition
// introduces:
//   - ProposalLifecycleHandler  — create / read / accept / decline /
//     modify / cancel + project listings (no money / no completion).
//   - ProposalPaymentHandler    — pay / confirm-payment / fund-milestone
//     (PaymentIntent + activation).
//   - ProposalCompletionHandler — request-completion / complete /
//     reject-completion + milestone-explicit submit / approve / reject.
//   - ProposalAdminHandler       — 5 admin-gated endpoints.
//
// The legacy ProposalHandler keeps its public method surface so all
// existing tests, all router wiring, and all external call sites
// compile unchanged. Each method is a one-line delegation to the
// corresponding focused handler.
type ProposalHandler struct {
	// Sub-handlers — owned & constructed by NewProposalHandler. The
	// four pointers cover the entire 22-method surface via delegation.
	lifecycle  *ProposalLifecycleHandler
	payment    *ProposalPaymentHandler
	completion *ProposalCompletionHandler
	admin      *ProposalAdminHandler

	// proposalSvc is preserved for the helper validateMilestoneMatchesCurrent
	// (called via the package-level validator helper that takes a
	// *proposalapp.Service). The public API of ProposalHandler stays
	// identical to its pre-Phase-3 shape.
	proposalSvc *proposalapp.Service
}

// NewProposalHandler wires the four focused sub-handlers and returns a
// composition facade. Existing callers continue to use the returned
// *ProposalHandler unchanged.
//
// paymentSvc may be nil when Stripe is not configured — only the
// payment sub-handler uses it, and it degrades gracefully.
func NewProposalHandler(svc *proposalapp.Service, paymentSvc *paymentapp.Service) *ProposalHandler {
	return &ProposalHandler{
		lifecycle:   NewProposalLifecycleHandler(svc),
		payment:     NewProposalPaymentHandler(svc, paymentSvc),
		completion:  NewProposalCompletionHandler(svc),
		admin:       NewProposalAdminHandler(svc),
		proposalSvc: svc,
	}
}

// ---------------------------------------------------------------------------
// Sub-handler accessors — for routers / tests that want the focused
// handler instead of the facade. Production code mostly uses the legacy
// methods below for backward compat; these accessors enable a future
// router refactor that wires sub-handlers directly.
// ---------------------------------------------------------------------------

// Lifecycle returns the lifecycle sub-handler.
func (h *ProposalHandler) Lifecycle() *ProposalLifecycleHandler { return h.lifecycle }

// Payment returns the payment sub-handler.
func (h *ProposalHandler) Payment() *ProposalPaymentHandler { return h.payment }

// Completion returns the completion sub-handler.
func (h *ProposalHandler) Completion() *ProposalCompletionHandler { return h.completion }

// Admin returns the admin sub-handler.
func (h *ProposalHandler) Admin() *ProposalAdminHandler { return h.admin }

// ---------------------------------------------------------------------------
// Lifecycle endpoints — delegated to ProposalLifecycleHandler
// ---------------------------------------------------------------------------

func (h *ProposalHandler) CreateProposal(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.CreateProposal(w, r)
}

func (h *ProposalHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.GetProposal(w, r)
}

func (h *ProposalHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.AcceptProposal(w, r)
}

func (h *ProposalHandler) DeclineProposal(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.DeclineProposal(w, r)
}

func (h *ProposalHandler) ModifyProposal(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.ModifyProposal(w, r)
}

func (h *ProposalHandler) CancelProposal(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.CancelProposal(w, r)
}

func (h *ProposalHandler) ListActiveProjects(w http.ResponseWriter, r *http.Request) {
	h.lifecycle.ListActiveProjects(w, r)
}

// ---------------------------------------------------------------------------
// Payment endpoints — delegated to ProposalPaymentHandler
// ---------------------------------------------------------------------------

func (h *ProposalHandler) PayProposal(w http.ResponseWriter, r *http.Request) {
	h.payment.PayProposal(w, r)
}

func (h *ProposalHandler) ConfirmPayment(w http.ResponseWriter, r *http.Request) {
	h.payment.ConfirmPayment(w, r)
}

func (h *ProposalHandler) FundMilestone(w http.ResponseWriter, r *http.Request) {
	h.payment.FundMilestone(w, r)
}

// ---------------------------------------------------------------------------
// Completion endpoints — delegated to ProposalCompletionHandler
// ---------------------------------------------------------------------------

func (h *ProposalHandler) RequestCompletion(w http.ResponseWriter, r *http.Request) {
	h.completion.RequestCompletion(w, r)
}

func (h *ProposalHandler) CompleteProposal(w http.ResponseWriter, r *http.Request) {
	h.completion.CompleteProposal(w, r)
}

func (h *ProposalHandler) RejectCompletion(w http.ResponseWriter, r *http.Request) {
	h.completion.RejectCompletion(w, r)
}

func (h *ProposalHandler) SubmitMilestone(w http.ResponseWriter, r *http.Request) {
	h.completion.SubmitMilestone(w, r)
}

func (h *ProposalHandler) ApproveMilestone(w http.ResponseWriter, r *http.Request) {
	h.completion.ApproveMilestone(w, r)
}

func (h *ProposalHandler) RejectMilestone(w http.ResponseWriter, r *http.Request) {
	h.completion.RejectMilestone(w, r)
}

// ---------------------------------------------------------------------------
// Admin endpoints — delegated to ProposalAdminHandler
// ---------------------------------------------------------------------------

func (h *ProposalHandler) AdminActivateProposal(w http.ResponseWriter, r *http.Request) {
	h.admin.AdminActivateProposal(w, r)
}

func (h *ProposalHandler) AdminListBonusLog(w http.ResponseWriter, r *http.Request) {
	h.admin.AdminListBonusLog(w, r)
}

func (h *ProposalHandler) AdminListPendingBonusLog(w http.ResponseWriter, r *http.Request) {
	h.admin.AdminListPendingBonusLog(w, r)
}

func (h *ProposalHandler) AdminApproveBonusEntry(w http.ResponseWriter, r *http.Request) {
	h.admin.AdminApproveBonusEntry(w, r)
}

func (h *ProposalHandler) AdminRejectBonusEntry(w http.ResponseWriter, r *http.Request) {
	h.admin.AdminRejectBonusEntry(w, r)
}

// ---------------------------------------------------------------------------
// Backward-compat: validateMilestoneMatchesCurrent stays on the wide
// handler so any out-of-tree caller keeps compiling. New code should
// call the package-level helper.
// ---------------------------------------------------------------------------

func (h *ProposalHandler) validateMilestoneMatchesCurrent(w http.ResponseWriter, r *http.Request, proposalID, expectedMilestoneID uuid.UUID) bool {
	return validateMilestoneMatchesCurrent(w, r, h.proposalSvc, proposalID, expectedMilestoneID)
}

// ---------------------------------------------------------------------------
// Shared package-level helpers used by every focused sub-handler
// ---------------------------------------------------------------------------

// convertMilestoneInputs maps the request DTO milestone slice onto the
// proposal app service's MilestoneInput type. Returns an error if a
// deadline string is malformed (RFC3339 or YYYY-MM-DD only).
func convertMilestoneInputs(in []request.MilestoneInputRequest) ([]proposalapp.MilestoneInput, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]proposalapp.MilestoneInput, 0, len(in))
	for _, m := range in {
		var deadline *time.Time
		if m.Deadline != "" {
			t, err := time.Parse(time.RFC3339, m.Deadline)
			if err != nil {
				t, err = time.Parse("2006-01-02", m.Deadline)
				if err != nil {
					return nil, errors.New("milestone deadline must be RFC3339 or YYYY-MM-DD")
				}
			}
			deadline = &t
		}
		out = append(out, proposalapp.MilestoneInput{
			Sequence:    m.Sequence,
			Title:       m.Title,
			Description: m.Description,
			Amount:      m.Amount,
			Deadline:    deadline,
		})
	}
	return out, nil
}

// convertDocumentInputs maps the request DTO document slice onto the
// proposal app service's DocumentInput type.
func convertDocumentInputs(inputs []request.DocumentInput) []proposalapp.DocumentInput {
	docs := make([]proposalapp.DocumentInput, len(inputs))
	for i, d := range inputs {
		docs[i] = proposalapp.DocumentInput{
			Filename: d.Filename,
			URL:      d.URL,
			Size:     d.Size,
			MimeType: d.MimeType,
		}
	}
	return docs
}

// validateMilestoneMatchesCurrent fetches the current active milestone
// of the proposal and asserts its id matches the one in the URL.
// Writes a 409 to the response on mismatch and returns false.
//
// Package-level helper because every milestone-explicit sub-handler
// (FundMilestone, SubmitMilestone, ApproveMilestone, RejectMilestone)
// runs the same check. Takes the proposal service explicitly so the
// helper does not need to be a method on a specific handler.
func validateMilestoneMatchesCurrent(w http.ResponseWriter, r *http.Request, svc *proposalapp.Service, proposalID, expectedMilestoneID uuid.UUID) bool {
	current, err := svc.ListMilestones(r.Context(), proposalID)
	if err != nil {
		handleProposalError(w, err)
		return false
	}
	active := findFirstActiveMilestone(current)
	if active == nil {
		res.Error(w, http.StatusConflict, "no_active_milestone", "no active milestone on this proposal")
		return false
	}
	if active.ID != expectedMilestoneID {
		res.Error(w, http.StatusConflict, "stale_milestone",
			"the milestone id in the URL does not match the current active milestone — refresh and retry")
		return false
	}
	return true
}

// findFirstActiveMilestone returns the lowest-sequence non-terminal
// milestone, or nil if none. Mirrors milestone.FindCurrentActive
// without importing the milestone domain helpers (kept local to limit
// the handler-layer surface).
func findFirstActiveMilestone(milestones []*milestonedomain.Milestone) *milestonedomain.Milestone {
	var current *milestonedomain.Milestone
	for _, m := range milestones {
		if m.IsTerminal() {
			continue
		}
		if current == nil || m.Sequence < current.Sequence {
			current = m
		}
	}
	return current
}

// requireAuthContext is the small helper used by every handler that
// needs both user_id and organization_id from the JWT context.
// Centralised here so the sub-handlers all share the same shape.
func requireAuthContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, true
}

// parseProposalAndMilestoneID extracts and validates the two URL params
// shared by every milestone-explicit endpoint. Writes a 400 on parse
// failure and returns ok=false.
func parseProposalAndMilestoneID(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "proposal id must be a valid UUID")
		return uuid.Nil, uuid.Nil, false
	}
	milestoneID, err := uuid.Parse(chi.URLParam(r, "mid"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_milestone_id", "milestone id must be a valid UUID")
		return uuid.Nil, uuid.Nil, false
	}
	return proposalID, milestoneID, true
}

// handleProposalError maps proposal + user domain errors to HTTP
// responses. Centralised here so every sub-handler shares a single
// translation table — adding a new domain error means a single edit.
func handleProposalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, proposaldomain.ErrProposalNotFound):
		res.Error(w, http.StatusNotFound, "proposal_not_found", err.Error())
	case errors.Is(err, proposaldomain.ErrNotAuthorized):
		res.Error(w, http.StatusForbidden, "not_authorized", err.Error())
	case errors.Is(err, proposaldomain.ErrCannotModify):
		res.Error(w, http.StatusForbidden, "cannot_modify", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidStatus):
		res.Error(w, http.StatusConflict, "invalid_status", err.Error())
	case errors.Is(err, proposaldomain.ErrSameUser):
		res.Error(w, http.StatusBadRequest, "same_user", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidRoleCombination):
		res.Error(w, http.StatusBadRequest, "invalid_role_combination", err.Error())
	case errors.Is(err, proposaldomain.ErrEmptyTitle):
		res.Error(w, http.StatusBadRequest, "empty_title", err.Error())
	case errors.Is(err, proposaldomain.ErrEmptyDescription):
		res.Error(w, http.StatusBadRequest, "empty_description", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidAmount):
		res.Error(w, http.StatusBadRequest, "invalid_amount", err.Error())
	case errors.Is(err, proposaldomain.ErrBelowMinimumAmount):
		res.Error(w, http.StatusBadRequest, "below_minimum_amount", err.Error())
	case errors.Is(err, milestonedomain.ErrMilestonesNotSequential):
		res.Error(w, http.StatusBadRequest, "milestones_not_sequential", err.Error())
	case errors.Is(err, milestonedomain.ErrMilestoneDeadlineAfterProject):
		res.Error(w, http.StatusBadRequest, "milestone_deadline_after_project", err.Error())
	case errors.Is(err, proposaldomain.ErrNotProvider):
		res.Error(w, http.StatusForbidden, "not_provider", err.Error())
	case errors.Is(err, proposaldomain.ErrNotClient):
		res.Error(w, http.StatusForbidden, "not_client", err.Error())
	case errors.Is(err, proposaldomain.ErrProviderKYCNotReady):
		// 412 Precondition Failed: the resource state is fine, but a
		// pre-condition (provider has finished Stripe onboarding) is
		// not met. The client should ask the provider to complete
		// payouts setup and retry. Message in French to match the
		// existing user-facing messaging style on the proposal flow.
		res.Error(w, http.StatusPreconditionFailed, "provider_kyc_incomplete",
			"Le prestataire doit terminer son onboarding Stripe avant que ce jalon puisse être libéré. Demande-lui de finaliser sa configuration de paiement.")
	case errors.Is(err, userdomain.ErrKYCRestricted):
		res.Error(w, http.StatusForbidden, "kyc_restricted", "Your account is restricted. Set up your payment info to lift this restriction.")
	default:
		slog.Error("unhandled proposal error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
