package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/handler/middleware"
)

// mountAdminRoutes wires the entire /admin sub-tree. Gated by THREE
// layered middlewares — defense-in-depth, every layer can independently
// refuse the request:
//
//  1. `auth` — must produce a valid identity. 401 otherwise.
//  2. `RequireRole("admin")` — primary role gate. 403 if the JWT/session
//     does not carry the `admin` role string. SEC-FINAL-03.
//  3. `RequireAdmin` — secondary `is_admin` flag gate. 403 if the
//     boolean is false (covers the case where a future migration
//     changes the role taxonomy but leaves the flag as the
//     authoritative source).
//
// The two role checks are intentional: a single-source check would
// fail open if an admin's role string drifts (rename, capitalization,
// etc.). With both checks an attacker would need to compromise both
// the role and the flag — the defense is multiplicative.
//
// The block is split into focused helpers so each admin domain
// (users, conversations, jobs, reviews, moderation, media, disputes,
// proposals, credits, search) keeps its own neighbourhood.
func mountAdminRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Admin == nil {
		return
	}
	r.Route("/admin", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.RequireRole("admin"))
		r.Use(middleware.RequireAdmin())
		r.Use(middleware.NoCache)
		mountAdminUsersRoutes(r, deps)
		mountAdminConversationsRoutes(r, deps)
		mountAdminJobsRoutes(r, deps)
		mountAdminMessageModerationRoutes(r, deps)
		mountAdminReviewsRoutes(r, deps)
		mountAdminUnifiedModerationRoutes(r, deps)
		mountAdminMediaRoutes(r, deps)
		mountAdminDisputesRoutes(r, deps)
		mountAdminProposalRoutes(r, deps)
		mountAdminTeamRoutes(r, deps)
		mountAdminSearchRoutes(r, deps)
		mountAdminInvoicingRoutes(r, deps)
	})
}

func mountAdminUsersRoutes(r chi.Router, deps RouterDeps) {
	r.Get("/dashboard/stats", deps.Admin.GetDashboardStats)
	r.Get("/users", deps.Admin.ListUsers)
	r.Get("/users/{id}", deps.Admin.GetUser)
	r.Post("/users/{id}/suspend", deps.Admin.SuspendUser)
	r.Post("/users/{id}/unsuspend", deps.Admin.UnsuspendUser)
	r.Post("/users/{id}/ban", deps.Admin.BanUser)
	r.Post("/users/{id}/unban", deps.Admin.UnbanUser)
	// Report management endpoints — keyed by user
	r.Get("/users/{id}/reports", deps.Admin.ListUserReports)
	// Admin notification counters
	r.Get("/notifications", deps.Admin.GetNotificationCounters)
	r.Post("/notifications/{category}/reset", deps.Admin.ResetNotificationCounter)
}

func mountAdminConversationsRoutes(r chi.Router, deps RouterDeps) {
	// Conversation moderation endpoints
	r.Get("/conversations", deps.Admin.ListConversations)
	r.Get("/conversations/{id}", deps.Admin.GetConversation)
	r.Get("/conversations/{id}/messages", deps.Admin.GetConversationMessages)
	r.Get("/conversations/{id}/reports", deps.Admin.ListConversationReports)
	// Generic report resolution
	r.Post("/reports/{id}/resolve", deps.Admin.ResolveReport)
}

func mountAdminJobsRoutes(r chi.Router, deps RouterDeps) {
	// Job admin endpoints
	r.Get("/jobs", deps.Admin.ListJobs)
	r.Get("/jobs/{id}", deps.Admin.GetAdminJob)
	r.Get("/jobs/{id}/reports", deps.Admin.ListJobReports)
	r.Delete("/jobs/{id}", deps.Admin.DeleteAdminJob)
	r.Get("/job-applications", deps.Admin.ListJobApplications)
	r.Delete("/job-applications/{id}", deps.Admin.DeleteJobApplication)
}

func mountAdminMessageModerationRoutes(r chi.Router, deps RouterDeps) {
	// Message moderation action endpoints
	r.Post("/messages/{id}/approve-moderation", deps.Admin.ApproveMessageModeration)
	r.Post("/messages/{id}/hide", deps.Admin.HideMessage)
	r.Post("/messages/{id}/restore-moderation", deps.Admin.RestoreMessageModeration)
}

func mountAdminReviewsRoutes(r chi.Router, deps RouterDeps) {
	// Review admin endpoints
	r.Get("/reviews", deps.Admin.ListReviews)
	r.Get("/reviews/{id}", deps.Admin.GetReview)
	r.Delete("/reviews/{id}", deps.Admin.DeleteReview)
	r.Get("/reviews/{id}/reports", deps.Admin.ListReviewReports)
	r.Post("/reviews/{id}/approve-moderation", deps.Admin.ApproveReviewModeration)
	r.Post("/reviews/{id}/restore-moderation", deps.Admin.RestoreReviewModeration)
}

func mountAdminUnifiedModerationRoutes(r chi.Router, deps RouterDeps) {
	// Unified moderation queue
	r.Get("/moderation", deps.Admin.ListModerationItems)
	r.Get("/moderation/count", deps.Admin.ModerationCount)
	// Generic restore endpoint covering Phase 2 content types
	// (profile_about, profile_title, job_title, job_description,
	// proposal_description, job_application_message,
	// user_display_name). The legacy per-type routes above
	// (.../messages/{id}/restore-moderation,
	// .../reviews/{id}/restore-moderation) keep working.
	r.Post("/moderation/{content_type}/{content_id}/restore", deps.Admin.RestoreModerationGeneric)
}

func mountAdminMediaRoutes(r chi.Router, deps RouterDeps) {
	// Media moderation endpoints
	r.Get("/media", deps.Admin.ListMedia)
	r.Get("/media/{id}", deps.Admin.GetMediaDetail)
	r.Post("/media/{id}/approve", deps.Admin.ApproveMedia)
	r.Post("/media/{id}/reject", deps.Admin.RejectMedia)
	r.Delete("/media/{id}", deps.Admin.DeleteMedia)
}

func mountAdminDisputesRoutes(r chi.Router, deps RouterDeps) {
	// Dispute admin endpoints
	if deps.AdminDispute == nil {
		return
	}
	r.Get("/disputes", deps.AdminDispute.ListDisputes)
	r.Get("/disputes/{id}", deps.AdminDispute.GetAdminDispute)
	r.Post("/disputes/{id}/resolve", deps.AdminDispute.ResolveDispute)
	r.Post("/disputes/{id}/force-escalate", deps.AdminDispute.ForceEscalate)
	r.Post("/disputes/{id}/ai-chat", deps.AdminDispute.AskAI)
	r.Post("/disputes/{id}/ai-budget", deps.AdminDispute.IncreaseAIBudget)
	r.Get("/disputes/count", deps.AdminDispute.CountDisputes)
}

func mountAdminProposalRoutes(r chi.Router, deps RouterDeps) {
	// Proposal admin endpoints (force activate for testing)
	if deps.Proposal != nil {
		r.Post("/proposals/{id}/activate", deps.Proposal.AdminActivateProposal)
	}

	// Job credit admin endpoints
	if deps.JobApplication != nil {
		r.Post("/credits/reset", deps.JobApplication.ResetCredits)
		r.Post("/credits/reset/{userId}", deps.JobApplication.ResetCreditsForUser)
	}

	// Credit bonus fraud log endpoints
	if deps.Proposal != nil {
		r.Get("/credits/bonus-log", deps.Proposal.AdminListBonusLog)
		r.Get("/credits/bonus-log/pending", deps.Proposal.AdminListPendingBonusLog)
		r.Post("/credits/bonus-log/{id}/approve", deps.Proposal.AdminApproveBonusEntry)
		r.Post("/credits/bonus-log/{id}/reject", deps.Proposal.AdminRejectBonusEntry)
	}
}

func mountAdminTeamRoutes(r chi.Router, deps RouterDeps) {
	// Team admin endpoints (Phase 6).
	// All five are gated by the same RequireAdmin middleware
	// applied to this whole group. Reads live under /users,
	// mutations live under /organizations.
	r.Get("/users/{id}/organization", deps.Admin.GetUserOrganization)
	r.Post("/organizations/{id}/force-transfer", deps.Admin.ForceTransferOwnership)
	r.Patch("/organizations/{id}/members/{userID}", deps.Admin.ForceUpdateMemberRole)
	r.Delete("/organizations/{id}/members/{userID}", deps.Admin.ForceRemoveMember)
	r.Delete("/organizations/{id}/invitations/{invID}", deps.Admin.ForceCancelInvitation)
}

func mountAdminSearchRoutes(r chi.Router, deps RouterDeps) {
	// Search analytics dashboard — admin-only aggregates over
	// the search_queries table. Gated by the outer RequireAdmin
	// middleware; the handler re-checks defensively.
	if deps.AdminSearchStats == nil {
		return
	}
	r.Get("/search/stats", deps.AdminSearchStats.GetStats)
}

func mountAdminInvoicingRoutes(r chi.Router, deps RouterDeps) {
	// Admin invoicing corrections — manual credit-note issuance.
	// Same RequireAdmin gate as every sibling under /admin.
	if deps.AdminCreditNote != nil {
		r.Post("/invoices/{id}/credit-note", deps.AdminCreditNote.Issue)
	}

	// Admin "all invoices ever emitted" listing + PDF redirect.
	// Wired separately from the credit-note handler so each
	// admin surface stays removable in isolation.
	if deps.AdminInvoice != nil {
		r.Get("/invoices", deps.AdminInvoice.List)
		r.Get("/invoices/{id}/pdf", deps.AdminInvoice.GetPDF)
	}
}
