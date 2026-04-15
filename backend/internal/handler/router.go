package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type RouterDeps struct {
	Auth           *AuthHandler
	Invitation     *InvitationHandler
	Team           *TeamHandler
	RoleOverrides  *RoleOverridesHandler
	Profile        *ProfileHandler
	ProfilePricing *ProfilePricingHandler

	// Split-profile handlers (migrations 096-104). These are the
	// new persona-specific surface for provider_personal orgs; the
	// legacy Profile/ProfilePricing handlers continue to serve the
	// agency path until a follow-up refactor. Every field is
	// optional so a worktree without the split wiring can still
	// boot — only the corresponding routes are registered when the
	// pointer is non-nil.
	FreelanceProfile        *FreelanceProfileHandler
	FreelancePricing        *FreelancePricingHandler
	FreelanceProfileVideo   *FreelanceProfileVideoHandler
	ReferrerProfile         *ReferrerProfileHandler
	ReferrerPricing         *ReferrerPricingHandler
	ReferrerProfileVideo    *ReferrerProfileVideoHandler
	OrganizationShared      *OrganizationSharedProfileHandler

	Upload         *UploadHandler
	Health         *HealthHandler
	Messaging      *MessagingHandler
	Proposal       *ProposalHandler
	Job            *JobHandler
	JobApplication *JobApplicationHandler
	Review         *ReviewHandler
	Call           *CallHandler
	SocialLink          *SocialLinkHandler // legacy agency-scoped handler
	FreelanceSocialLink *SocialLinkHandler // persona=freelance handler
	ReferrerSocialLink  *SocialLinkHandler // persona=referrer handler
	Embedded       *EmbeddedHandler
	Notification   *NotificationHandler
	Stripe         *StripeHandler
	Report         *ReportHandler
	Wallet         *WalletHandler
	Admin          *AdminHandler
	Portfolio      *PortfolioHandler
	ProjectHistory *ProjectHistoryHandler
	Dispute        *DisputeHandler
	AdminDispute   *AdminDisputeHandler
	Skill          *SkillHandler
	Referral       *ReferralHandler
	Search         *SearchHandler // optional — nil when Typesense is disabled
	WSHandler      http.HandlerFunc
	Config         *config.Config
	TokenService   service.TokenService
	SessionService service.SessionService
	UserRepo       repository.UserRepository // for KYC middleware
}

func NewRouter(deps RouterDeps) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	limiter := middleware.NewRateLimiter(10, 20) // 10 req/s, burst 20
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(deps.Config.AllowedOrigins))
	r.Use(limiter.Middleware)

	// Health routes
	r.Get("/health", deps.Health.Health)
	r.Get("/ready", deps.Health.Ready)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", deps.Auth.Register)
			r.Post("/login", deps.Auth.Login)
			r.Post("/refresh", deps.Auth.Refresh)
			r.Post("/forgot-password", deps.Auth.ForgotPassword)
			r.Post("/reset-password", deps.Auth.ResetPassword)

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/me", deps.Auth.Me)
				r.Get("/ws-token", deps.Auth.WSToken)
				r.Post("/logout", deps.Auth.Logout)
				r.Put("/referrer-enable", deps.Auth.EnableReferrer)
			})
		})

		// Team invitation routes — public acceptance endpoints + protected
		// management endpoints nested under /organizations/{orgID}/invitations.
		if deps.Invitation != nil {
			// Public: validate a token and accept an invitation.
			r.Get("/invitations/validate", deps.Invitation.Validate)
			r.Post("/invitations/accept", deps.Invitation.Accept)

			r.Route("/organizations/{orgID}/invitations", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Post("/", deps.Invitation.Send)
				r.Get("/", deps.Invitation.List)
				r.Post("/{invID}/resend", deps.Invitation.Resend)
				r.Delete("/{invID}", deps.Invitation.Cancel)
			})
		}

		// Team management routes — list/edit/remove members + transfer
		// ownership. All protected by auth middleware; each method
		// enforces its own permission checks at the service layer.
		if deps.Team != nil {
			// Static org-scoped routes that do NOT take an orgID URL
			// param go above the {orgID} route group so chi resolves
			// them correctly. role-definitions is a global catalogue
			// (R13: team page "About roles" panel + edit modal preview).
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/organizations/role-definitions", deps.Team.RoleDefinitions)
			})

			r.Route("/organizations/{orgID}", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)

				r.Get("/members", deps.Team.ListMembers)
				r.Patch("/members/{userID}", deps.Team.UpdateMember)
				r.Delete("/members/{userID}", deps.Team.RemoveMember)
				r.Post("/leave", deps.Team.Leave)

				r.Post("/transfer", deps.Team.InitiateTransfer)
				r.Delete("/transfer", deps.Team.CancelTransfer)
				r.Post("/transfer/accept", deps.Team.AcceptTransfer)
				r.Post("/transfer/decline", deps.Team.DeclineTransfer)

				// Role permissions editor (R17 — per-org customization).
				// GET is readable by any org member (every role holds
				// team.view in the defaults). PATCH is Owner-only and
				// additionally defense-in-depth gated by the service
				// layer. The middleware fast-path uses the Owner-only
				// PermTeamManageRolePermissions permission which is
				// itself non-overridable.
				if deps.RoleOverrides != nil {
					r.Get("/role-permissions", deps.RoleOverrides.GetMatrix)
					r.With(middleware.RequirePermission(organization.PermTeamManageRolePermissions)).
						Patch("/role-permissions", deps.RoleOverrides.UpdateMatrix)
				}
			})
		}

		// Profile routes (authenticated, permission-gated)
		r.Route("/profile", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
			r.Use(middleware.NoCache)
			r.Get("/", deps.Profile.GetMyProfile)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.Profile.UpdateMyProfile)
			// Expertise domains — same "edit profile" permission as the
			// main profile fields. The feature is hard-disabled for
			// enterprise orgs at the service layer (403 forbidden).
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/expertise", deps.Profile.UpdateMyExpertise)
			// Profile skills (authenticated). Same permission as expertise
			// — both are public-profile decorations shared by the whole
			// org. The feature is hard-disabled for enterprise orgs at
			// the service layer (403 forbidden).
			if deps.Skill != nil {
				r.Get("/skills", deps.Skill.GetMyProfileSkills)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/skills", deps.Skill.PutMyProfileSkills)
			}
			// Profile Tier 1 completion (migration 083): location,
			// languages, availability blocks. Same edit-profile
			// permission as the main profile fields — all three
			// are public profile decorations shared by the whole org.
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/location", deps.Profile.UpdateMyLocation)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/languages", deps.Profile.UpdateMyLanguages)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/availability", deps.Profile.UpdateMyAvailability)

			// Profile pricing (migration 083). Wired through a
			// dedicated handler (ProfilePricingHandler) to preserve
			// the feature-isolation principle — deleting the
			// pricing feature means deleting that file + wiring
			// without touching ProfileHandler.
			if deps.ProfilePricing != nil {
				r.Get("/pricing", deps.ProfilePricing.ListMyPricing)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/pricing", deps.ProfilePricing.UpsertMyPricing)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/pricing/{kind}", deps.ProfilePricing.DeleteMyPricingByKind)
			}
		})

		// Split-profile routes (migrations 096-104). Mounted in
		// their own route groups so a worktree without the split
		// handlers wired in boots cleanly — the `if` guards keep
		// each feature fully removable.
		if deps.FreelanceProfile != nil {
			r.Route("/freelance-profile", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/", deps.FreelanceProfile.GetMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.FreelanceProfile.UpdateMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/availability", deps.FreelanceProfile.UpdateMyAvailability)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/expertise", deps.FreelanceProfile.UpdateMyExpertise)
				if deps.FreelanceProfileVideo != nil {
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Post("/video", deps.FreelanceProfileVideo.Upload)
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/video", deps.FreelanceProfileVideo.Delete)
				}
				if deps.FreelancePricing != nil {
					r.Get("/pricing", deps.FreelancePricing.GetMy)
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/pricing", deps.FreelancePricing.UpsertMy)
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/pricing", deps.FreelancePricing.DeleteMy)
				}
			})
		}
		if deps.ReferrerProfile != nil {
			r.Route("/referrer-profile", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/", deps.ReferrerProfile.GetMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.ReferrerProfile.UpdateMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/availability", deps.ReferrerProfile.UpdateMyAvailability)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/expertise", deps.ReferrerProfile.UpdateMyExpertise)
				if deps.ReferrerProfileVideo != nil {
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Post("/video", deps.ReferrerProfileVideo.Upload)
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/video", deps.ReferrerProfileVideo.Delete)
				}
				if deps.ReferrerPricing != nil {
					r.Get("/pricing", deps.ReferrerPricing.GetMy)
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/pricing", deps.ReferrerPricing.UpsertMy)
					r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/pricing", deps.ReferrerPricing.DeleteMy)
				}
			})
		}
		if deps.OrganizationShared != nil {
			r.Route("/organization", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/shared", deps.OrganizationShared.GetSharedProfile)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/location", deps.OrganizationShared.UpdateLocation)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/languages", deps.OrganizationShared.UpdateLanguages)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/photo", deps.OrganizationShared.UpdatePhoto)
			})
		}

		// Skills catalog (mixed: public browse/autocomplete, authenticated create)
		if deps.Skill != nil {
			// Public catalog reads — no auth required so the discovery
			// UI can surface skills to anonymous visitors.
			r.Get("/skills/catalog", deps.Skill.GetCuratedByExpertise)
			r.Get("/skills/autocomplete", deps.Skill.Autocomplete)

			// Authenticated: create a new user-contributed skill from
			// the "Create X" autocomplete option. Permission-gated by
			// the same edit-profile grant as the profile skills PUT.
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Post("/skills", deps.Skill.CreateUserSkill)
			})
		}

		// Upload routes (authenticated, permission-gated)
		r.Route("/upload", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
			r.Use(middleware.NoCache)
			// Profile-related uploads require org profile edit permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(organization.PermOrgProfileEdit))
				r.Post("/photo", deps.Upload.UploadPhoto)
				r.Post("/video", deps.Upload.UploadVideo)
				r.Delete("/video", deps.Upload.DeleteVideo)
				r.Post("/referrer-video", deps.Upload.UploadReferrerVideo)
				r.Delete("/referrer-video", deps.Upload.DeleteReferrerVideo)
				r.Post("/portfolio-image", deps.Upload.UploadPortfolioImage)
				r.Post("/portfolio-video", deps.Upload.UploadPortfolioVideo)
			})
			// Review video upload requires review permission
			r.With(middleware.RequirePermission(organization.PermReviewsRespond)).Post("/review-video", deps.Upload.UploadReviewVideo)
		})

		// Public profiles (keyed by organization id since phase R2)
		r.Get("/profiles/search", deps.Profile.SearchProfiles)
		r.Get("/profiles/{orgId}", deps.Profile.GetPublicProfile)

		// Typesense-backed search routes (phase 2). Both endpoints
		// require an authenticated user — anonymous browsing of
		// the listing pages still goes through the legacy
		// /profiles/search SQL path until phase 4 retires it.
		if deps.Search != nil {
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/search/key", deps.Search.ScopedKey)
				r.Get("/search", deps.Search.Search)
			})
		}
		if deps.ProjectHistory != nil {
			r.Get("/profiles/{orgId}/project-history", deps.ProjectHistory.ListByOrganization)
		}

		// Public read routes for the split-profile personas
		// (provider_personal orgs only). Keyed by organization_id
		// so the URL scheme stays symmetrical with the legacy
		// /profiles/{orgId} and the frontend's existing routes.
		if deps.FreelanceProfile != nil {
			r.Get("/freelance-profiles/{orgID}", deps.FreelanceProfile.GetPublic)
		}
		if deps.ReferrerProfile != nil {
			r.Get("/referrer-profiles/{orgID}", deps.ReferrerProfile.GetPublic)
		}

		// Messaging routes (authenticated, permission-gated)
		if deps.Messaging != nil {
			r.Route("/messaging", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				// Read operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermMessagingView))
					r.Get("/conversations", deps.Messaging.ListConversations)
					r.Get("/conversations/{id}/messages", deps.Messaging.ListMessages)
					r.Post("/conversations/{id}/read", deps.Messaging.MarkAsRead)
					r.Get("/unread-count", deps.Messaging.GetTotalUnread)
				})
				// Write operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermMessagingSend))
					r.Post("/conversations", deps.Messaging.StartConversation)
					r.Post("/conversations/{id}/messages", deps.Messaging.SendMessage)
					r.Put("/messages/{id}", deps.Messaging.EditMessage)
					r.Delete("/messages/{id}", deps.Messaging.DeleteMessage)
					r.Post("/upload-url", deps.Messaging.GetPresignedURL)
				})
			})
		}

		// Proposal routes (authenticated, permission-gated)
		if deps.Proposal != nil {
			r.Route("/proposals", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.With(middleware.RequirePermission(organization.PermProposalsView)).Get("/{id}", deps.Proposal.GetProposal)
				r.With(middleware.RequirePermission(organization.PermProposalsCreate)).Post("/", deps.Proposal.CreateProposal)
				r.With(middleware.RequirePermission(organization.PermProposalsCreate)).Post("/{id}/modify", deps.Proposal.ModifyProposal)
				// Respond actions (accept, decline, pay, complete flow)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermProposalsRespond))
					r.Post("/{id}/accept", deps.Proposal.AcceptProposal)
					r.Post("/{id}/decline", deps.Proposal.DeclineProposal)
					// Legacy endpoints (one-time mode shortcut, kept for
					// backward compatibility — they delegate to the same
					// proposal service methods as the new milestone-explicit
					// routes below).
					r.Post("/{id}/pay", deps.Proposal.PayProposal)
					r.Post("/{id}/confirm-payment", deps.Proposal.ConfirmPayment)
					r.Post("/{id}/request-completion", deps.Proposal.RequestCompletion)
					r.Post("/{id}/complete", deps.Proposal.CompleteProposal)
					r.Post("/{id}/reject-completion", deps.Proposal.RejectCompletion)
					// Phase 5: milestone-explicit endpoints. The {mid}
					// segment is validated against the current active
					// milestone — a stale client view (someone else has
					// moved the proposal forward) yields 409 Conflict
					// instead of silently mutating the wrong milestone.
					r.Post("/{id}/milestones/{mid}/fund", deps.Proposal.FundMilestone)
					r.Post("/{id}/milestones/{mid}/submit", deps.Proposal.SubmitMilestone)
					r.Post("/{id}/milestones/{mid}/approve", deps.Proposal.ApproveMilestone)
					r.Post("/{id}/milestones/{mid}/reject", deps.Proposal.RejectMilestone)
					// Boundary cancel (no money in flight). Either side
					// may initiate — the proposal service's
					// requireOrgIsParticipant check handles auth.
					r.Post("/{id}/cancel", deps.Proposal.CancelProposal)
				})
			})
			r.Route("/projects", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Use(middleware.RequirePermission(organization.PermProposalsView))
				r.Get("/", deps.Proposal.ListActiveProjects)
			})
		}

		// Job routes (authenticated, permission-gated)
		if deps.Job != nil {
			r.Route("/jobs", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)

				// View operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermJobsView))
					r.Get("/mine", deps.Job.ListMyJobs)
					r.Get("/{id}", deps.Job.GetJob)
					r.Post("/{id}/mark-viewed", deps.Job.MarkApplicationsViewed)
					if deps.JobApplication != nil {
						r.Get("/open", deps.JobApplication.ListOpenJobs)
						r.Get("/credits", deps.JobApplication.GetCredits)
						r.Get("/{id}/applications", deps.JobApplication.ListJobApplications)
						r.Get("/{id}/has-applied", deps.JobApplication.HasApplied)
					}
				})

				// Create
				r.With(middleware.RequirePermission(organization.PermJobsCreate)).Post("/", deps.Job.CreateJob)

				// Edit
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermJobsEdit))
					r.Put("/{id}", deps.Job.UpdateJob)
					r.Post("/{id}/close", deps.Job.CloseJob)
					r.Post("/{id}/reopen", deps.Job.ReopenJob)
				})

				// Delete (Owner/Admin only)
				r.With(middleware.RequirePermission(organization.PermJobsDelete)).Delete("/{id}", deps.Job.DeleteJob)

				// Application actions (proposal + messaging permissions)
				if deps.JobApplication != nil {
					r.With(middleware.RequirePermission(organization.PermProposalsView)).Get("/applications/mine", deps.JobApplication.ListMyApplications)
					r.With(middleware.RequirePermission(organization.PermProposalsCreate)).Post("/{id}/apply", deps.JobApplication.ApplyToJob)
					r.With(middleware.RequirePermission(organization.PermProposalsCreate)).Delete("/applications/{applicationId}", deps.JobApplication.WithdrawApplication)
					r.With(middleware.RequirePermission(organization.PermMessagingSend)).Post("/{id}/applications/{applicantId}/contact", deps.JobApplication.ContactApplicant)
				}
			})
		}

		// Review routes (mixed: public reads, authenticated writes)
		if deps.Review != nil {
			r.Route("/reviews", func(r chi.Router) {
				// Public: read reviews and average ratings (keyed by org)
				r.Get("/org/{orgId}", deps.Review.ListByOrganization)
				r.Get("/average/{orgId}", deps.Review.GetAverageRating)

				// Authenticated: create reviews and check eligibility
				r.Group(func(r chi.Router) {
					r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
					r.Use(middleware.NoCache)
					r.With(middleware.RequirePermission(organization.PermReviewsRespond)).Post("/", deps.Review.CreateReview)
					r.With(middleware.RequirePermission(organization.PermProposalsView)).Get("/can-review/{proposalId}", deps.Review.CanReview)
				})
			})
		}

		// Report routes (authenticated)
		if deps.Report != nil {
			r.Route("/reports", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Post("/", deps.Report.CreateReport)
				r.Get("/mine", deps.Report.ListMyReports)
			})
		}

		// Social link routes — legacy agency-scoped path. Kept for
		// backwards compatibility with the agency profile flow.
		if deps.SocialLink != nil {
			// Public: read agency social links
			r.Get("/profiles/{orgId}/social-links", deps.SocialLink.ListPublicSocialLinks)

			// Authenticated: manage own agency social links
			r.Route("/profile/social-links", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/", deps.SocialLink.ListMySocialLinks)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.SocialLink.UpsertSocialLink)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/{platform}", deps.SocialLink.DeleteSocialLink)
			})
		}

		// Freelance persona social link routes — independent set
		// scoped to the freelance identity of provider_personal users.
		if deps.FreelanceSocialLink != nil {
			r.Get("/freelance-profiles/{orgId}/social-links", deps.FreelanceSocialLink.ListPublicSocialLinks)

			r.Route("/freelance-profile/social-links", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/", deps.FreelanceSocialLink.ListMySocialLinks)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.FreelanceSocialLink.UpsertSocialLink)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/{platform}", deps.FreelanceSocialLink.DeleteSocialLink)
			})
		}

		// Referrer persona social link routes — independent set
		// scoped to the apporteur d'affaires identity.
		if deps.ReferrerSocialLink != nil {
			r.Get("/referrer-profiles/{orgId}/social-links", deps.ReferrerSocialLink.ListPublicSocialLinks)

			r.Route("/referrer-profile/social-links", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/", deps.ReferrerSocialLink.ListMySocialLinks)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.ReferrerSocialLink.UpsertSocialLink)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/{platform}", deps.ReferrerSocialLink.DeleteSocialLink)
			})
		}

		// Portfolio routes (mixed: public reads, authenticated writes)
		if deps.Portfolio != nil {
			// Public: read portfolio for an organization
			r.Get("/portfolio/org/{orgId}", deps.Portfolio.ListPortfolioByOrganization)
			r.Get("/portfolio/{id}", deps.Portfolio.GetPortfolioItem)

			// Authenticated: manage own portfolio (org profile edit permission)
			r.Route("/portfolio", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Use(middleware.RequirePermission(organization.PermOrgProfileEdit))
				r.Post("/", deps.Portfolio.CreatePortfolioItem)
				r.Put("/reorder", deps.Portfolio.ReorderPortfolio)
				r.Put("/{id}", deps.Portfolio.UpdatePortfolioItem)
				r.Delete("/{id}", deps.Portfolio.DeletePortfolioItem)
			})
		}

		// Call routes (authenticated, permission-gated)
		if deps.Call != nil {
			r.Route("/calls", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.With(middleware.RequirePermission(organization.PermMessagingSend)).Post("/initiate", deps.Call.InitiateCall)
				// Accept/decline/end are receiving-side actions — view permission is sufficient
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermMessagingView))
					r.Post("/{id}/accept", deps.Call.AcceptCall)
					r.Post("/{id}/decline", deps.Call.DeclineCall)
					r.Post("/{id}/end", deps.Call.EndCall)
				})
			})
		}

		// Payment info routes — all served by Embedded Components now.
		if deps.Embedded != nil {
			r.Route("/payment-info", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.With(middleware.RequirePermission(organization.PermBillingView)).Get("/account-status", deps.Embedded.GetAccountStatus)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermKYCManage))
					r.Post("/account-session", deps.Embedded.CreateAccountSession)
					r.Delete("/account-session", deps.Embedded.ResetAccount)
				})
			})
		}

		// Notification routes (authenticated)
		if deps.Notification != nil {
			r.Route("/notifications", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Get("/", deps.Notification.ListNotifications)
				r.Get("/unread-count", deps.Notification.GetUnreadCount)
				r.Post("/{id}/read", deps.Notification.MarkAsRead)
				r.Post("/read-all", deps.Notification.MarkAllAsRead)
				r.Delete("/{id}", deps.Notification.DeleteNotification)
				r.Get("/preferences", deps.Notification.GetPreferences)
				r.Put("/preferences", deps.Notification.UpdatePreferences)
				r.Patch("/preferences/bulk-email", deps.Notification.BulkUpdateEmailPreferences)
				r.Post("/device-token", deps.Notification.RegisterDeviceToken)
			})
		}

		// Identity documents are now handled by Stripe Embedded Components —
		// no custom upload/list/delete endpoints needed.

		// Wallet routes (authenticated, permission-gated)
		if deps.Wallet != nil {
			r.Route("/wallet", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.With(middleware.RequirePermission(organization.PermWalletView)).Get("/", deps.Wallet.GetWallet)
				r.With(middleware.RequirePermission(organization.PermWalletWithdraw)).Post("/payout", deps.Wallet.RequestPayout)
			})
		}

		// Referral (apport d'affaires) routes — authenticated, no per-route
		// permission middleware: ownership (referrer / provider / client
		// party of the referral) is enforced inside the service layer by
		// loadAndAuthorise on every state transition.
		if deps.Referral != nil {
			r.Route("/referrals", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.Post("/", deps.Referral.Create)
				r.Get("/me", deps.Referral.ListMine)
				r.Get("/incoming", deps.Referral.ListIncoming)
				r.Get("/{id}", deps.Referral.Get)
				r.Post("/{id}/respond", deps.Referral.Respond)
				r.Get("/{id}/negotiations", deps.Referral.ListNegotiations)
			})
		}

		// Dispute routes (authenticated, permission-gated)
		if deps.Dispute != nil {
			r.Route("/disputes", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				// Read
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermProposalsView))
					r.Get("/mine", deps.Dispute.ListMyDisputes)
					r.Get("/{id}", deps.Dispute.GetDispute)
				})
				// Write (disputes are proposal-level actions)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(organization.PermProposalsRespond))
					r.Post("/", deps.Dispute.OpenDispute)
					r.Post("/{id}/counter-propose", deps.Dispute.CounterPropose)
					r.Post("/{id}/counter-proposals/{cpId}/respond", deps.Dispute.RespondToCounter)
					r.Post("/{id}/cancel", deps.Dispute.CancelDispute)
					r.Post("/{id}/cancellation/respond", deps.Dispute.RespondToCancellation)
				})
			})
		}

		// Stripe routes
		if deps.Stripe != nil {
			r.Route("/stripe", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.NoCache)
				r.With(middleware.RequirePermission(organization.PermBillingView)).Get("/config", deps.Stripe.GetConfig)
			})
			// Webhook: NO auth — Stripe sends directly, verified by signature
			r.Post("/stripe/webhook", deps.Stripe.HandleWebhook)
		}

		// WebSocket (auth handled inside the handler)
		if deps.WSHandler != nil {
			r.Get("/ws", deps.WSHandler)
		}

		// Admin routes (authenticated + admin only)
		if deps.Admin != nil {
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService, deps.UserRepo))
				r.Use(middleware.RequireAdmin())
				r.Use(middleware.NoCache)
				r.Get("/dashboard/stats", deps.Admin.GetDashboardStats)
				r.Get("/users", deps.Admin.ListUsers)
				r.Get("/users/{id}", deps.Admin.GetUser)
				r.Post("/users/{id}/suspend", deps.Admin.SuspendUser)
				r.Post("/users/{id}/unsuspend", deps.Admin.UnsuspendUser)
				r.Post("/users/{id}/ban", deps.Admin.BanUser)
				r.Post("/users/{id}/unban", deps.Admin.UnbanUser)

				// Conversation moderation endpoints
				r.Get("/conversations", deps.Admin.ListConversations)
				r.Get("/conversations/{id}", deps.Admin.GetConversation)
				r.Get("/conversations/{id}/messages", deps.Admin.GetConversationMessages)
				r.Get("/conversations/{id}/reports", deps.Admin.ListConversationReports)

				// Report management endpoints
				r.Get("/users/{id}/reports", deps.Admin.ListUserReports)
				r.Post("/reports/{id}/resolve", deps.Admin.ResolveReport)

				// Job admin endpoints
				r.Get("/jobs", deps.Admin.ListJobs)
				r.Get("/jobs/{id}", deps.Admin.GetAdminJob)
				r.Get("/jobs/{id}/reports", deps.Admin.ListJobReports)
				r.Delete("/jobs/{id}", deps.Admin.DeleteAdminJob)
				r.Get("/job-applications", deps.Admin.ListJobApplications)
				r.Delete("/job-applications/{id}", deps.Admin.DeleteJobApplication)

				// Message moderation action endpoints
				r.Post("/messages/{id}/approve-moderation", deps.Admin.ApproveMessageModeration)
				r.Post("/messages/{id}/hide", deps.Admin.HideMessage)

				// Review admin endpoints
				r.Get("/reviews", deps.Admin.ListReviews)
				r.Get("/reviews/{id}", deps.Admin.GetReview)
				r.Delete("/reviews/{id}", deps.Admin.DeleteReview)
				r.Get("/reviews/{id}/reports", deps.Admin.ListReviewReports)
				r.Post("/reviews/{id}/approve-moderation", deps.Admin.ApproveReviewModeration)

				// Unified moderation queue
				r.Get("/moderation", deps.Admin.ListModerationItems)
				r.Get("/moderation/count", deps.Admin.ModerationCount)

				// Admin notification counters
				r.Get("/notifications", deps.Admin.GetNotificationCounters)
				r.Post("/notifications/{category}/reset", deps.Admin.ResetNotificationCounter)

				// Media moderation endpoints
				r.Get("/media", deps.Admin.ListMedia)
				r.Get("/media/{id}", deps.Admin.GetMediaDetail)
				r.Post("/media/{id}/approve", deps.Admin.ApproveMedia)
				r.Post("/media/{id}/reject", deps.Admin.RejectMedia)
				r.Delete("/media/{id}", deps.Admin.DeleteMedia)

				// Dispute admin endpoints
				if deps.AdminDispute != nil {
					r.Get("/disputes", deps.AdminDispute.ListDisputes)
					r.Get("/disputes/{id}", deps.AdminDispute.GetAdminDispute)
					r.Post("/disputes/{id}/resolve", deps.AdminDispute.ResolveDispute)
					r.Post("/disputes/{id}/force-escalate", deps.AdminDispute.ForceEscalate)
					r.Post("/disputes/{id}/ai-chat", deps.AdminDispute.AskAI)
					r.Post("/disputes/{id}/ai-budget", deps.AdminDispute.IncreaseAIBudget)
					r.Get("/disputes/count", deps.AdminDispute.CountDisputes)
				}

				// Proposal admin endpoints (force activate for testing)
				if deps.Proposal != nil {
					r.Post("/proposals/{id}/activate", deps.Proposal.AdminActivateProposal)
				}

				// Job credit admin endpoints
				if deps.JobApplication != nil {
					r.Post("/credits/reset", deps.JobApplication.ResetCredits)
					r.Post("/credits/reset/{userId}", deps.JobApplication.ResetCreditsForUser)
				}

				// Team admin endpoints (Phase 6).
				// All five are gated by the same RequireAdmin middleware
				// applied to this whole group. Reads live under /users,
				// mutations live under /organizations.
				r.Get("/users/{id}/organization", deps.Admin.GetUserOrganization)
				r.Post("/organizations/{id}/force-transfer", deps.Admin.ForceTransferOwnership)
				r.Patch("/organizations/{id}/members/{userID}", deps.Admin.ForceUpdateMemberRole)
				r.Delete("/organizations/{id}/members/{userID}", deps.Admin.ForceRemoveMember)
				r.Delete("/organizations/{id}/invitations/{invID}", deps.Admin.ForceCancelInvitation)

				// Credit bonus fraud log endpoints
				if deps.Proposal != nil {
					r.Get("/credits/bonus-log", deps.Proposal.AdminListBonusLog)
					r.Get("/credits/bonus-log/pending", deps.Proposal.AdminListPendingBonusLog)
					r.Post("/credits/bonus-log/{id}/approve", deps.Proposal.AdminApproveBonusEntry)
					r.Post("/credits/bonus-log/{id}/reject", deps.Proposal.AdminRejectBonusEntry)
				}
			})
		}

		// Test routes (debug — backend & DB connectivity)
		r.Route("/test", func(r chi.Router) {
			r.Get("/health-check", deps.Health.HealthCheck)
			r.Get("/words", deps.Health.GetWords)
			r.Post("/words", deps.Health.AddWord)
		})
	})

	return r
}
