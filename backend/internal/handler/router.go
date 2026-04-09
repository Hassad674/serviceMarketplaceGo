package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
)

type RouterDeps struct {
	Auth           *AuthHandler
	Profile        *ProfileHandler
	Upload         *UploadHandler
	Health         *HealthHandler
	Messaging      *MessagingHandler
	Proposal       *ProposalHandler
	Job            *JobHandler
	JobApplication *JobApplicationHandler
	Review         *ReviewHandler
	Call           *CallHandler
	SocialLink     *SocialLinkHandler
	Embedded       *EmbeddedHandler
	Notification   *NotificationHandler
	Stripe         *StripeHandler
	Report         *ReportHandler
	Wallet         *WalletHandler
	Admin          *AdminHandler
	Portfolio      *PortfolioHandler
	WSHandler      http.HandlerFunc
	Config         *config.Config
	TokenService   service.TokenService
	SessionService service.SessionService
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
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Get("/me", deps.Auth.Me)
				r.Get("/ws-token", deps.Auth.WSToken)
				r.Post("/logout", deps.Auth.Logout)
				r.Put("/referrer-enable", deps.Auth.EnableReferrer)
			})
		})

		// Profile routes (authenticated)
		r.Route("/profile", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
			r.Use(middleware.NoCache)
			r.Get("/", deps.Profile.GetMyProfile)
			r.Put("/", deps.Profile.UpdateMyProfile)
		})

		// Upload routes (authenticated)
		r.Route("/upload", func(r chi.Router) {
			r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
			r.Use(middleware.NoCache)
			r.Post("/photo", deps.Upload.UploadPhoto)
			r.Post("/video", deps.Upload.UploadVideo)
			r.Delete("/video", deps.Upload.DeleteVideo)
			r.Post("/referrer-video", deps.Upload.UploadReferrerVideo)
			r.Delete("/referrer-video", deps.Upload.DeleteReferrerVideo)
			r.Post("/review-video", deps.Upload.UploadReviewVideo)
			r.Post("/portfolio-image", deps.Upload.UploadPortfolioImage)
			r.Post("/portfolio-video", deps.Upload.UploadPortfolioVideo)
		})

		// Public profiles
		r.Get("/profiles/search", deps.Profile.SearchProfiles)
		r.Get("/profiles/{userId}", deps.Profile.GetPublicProfile)

		// Messaging routes (authenticated)
		if deps.Messaging != nil {
			r.Route("/messaging", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Post("/conversations", deps.Messaging.StartConversation)
				r.Get("/conversations", deps.Messaging.ListConversations)
				r.Get("/conversations/{id}/messages", deps.Messaging.ListMessages)
				r.Post("/conversations/{id}/messages", deps.Messaging.SendMessage)
				r.Post("/conversations/{id}/read", deps.Messaging.MarkAsRead)
				r.Put("/messages/{id}", deps.Messaging.EditMessage)
				r.Delete("/messages/{id}", deps.Messaging.DeleteMessage)
				r.Post("/upload-url", deps.Messaging.GetPresignedURL)
				r.Get("/unread-count", deps.Messaging.GetTotalUnread)
			})
		}

		// Proposal routes (authenticated)
		if deps.Proposal != nil {
			r.Route("/proposals", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Post("/", deps.Proposal.CreateProposal)
				r.Get("/{id}", deps.Proposal.GetProposal)
				r.Post("/{id}/accept", deps.Proposal.AcceptProposal)
				r.Post("/{id}/decline", deps.Proposal.DeclineProposal)
				r.Post("/{id}/modify", deps.Proposal.ModifyProposal)
				r.Post("/{id}/pay", deps.Proposal.PayProposal)
				r.Post("/{id}/confirm-payment", deps.Proposal.ConfirmPayment)
				r.Post("/{id}/request-completion", deps.Proposal.RequestCompletion)
				r.Post("/{id}/complete", deps.Proposal.CompleteProposal)
				r.Post("/{id}/reject-completion", deps.Proposal.RejectCompletion)
			})
			r.Route("/projects", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Get("/", deps.Proposal.ListActiveProjects)
			})
		}

		// Job routes (authenticated)
		if deps.Job != nil {
			r.Route("/jobs", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)

				// Static routes first (before {id} wildcard)
				r.Post("/", deps.Job.CreateJob)
				r.Get("/mine", deps.Job.ListMyJobs)

				if deps.JobApplication != nil {
					r.Get("/open", deps.JobApplication.ListOpenJobs)
					r.Get("/credits", deps.JobApplication.GetCredits)
					r.Get("/applications/mine", deps.JobApplication.ListMyApplications)
					r.Delete("/applications/{applicationId}", deps.JobApplication.WithdrawApplication)
				}

				// Parameterized routes
				r.Get("/{id}", deps.Job.GetJob)
				r.Put("/{id}", deps.Job.UpdateJob)
				r.Post("/{id}/close", deps.Job.CloseJob)
				r.Post("/{id}/reopen", deps.Job.ReopenJob)
				r.Delete("/{id}", deps.Job.DeleteJob)
				r.Post("/{id}/mark-viewed", deps.Job.MarkApplicationsViewed)

				if deps.JobApplication != nil {
					r.Post("/{id}/apply", deps.JobApplication.ApplyToJob)
					r.Get("/{id}/applications", deps.JobApplication.ListJobApplications)
					r.Get("/{id}/has-applied", deps.JobApplication.HasApplied)
					r.Post("/{id}/applications/{applicantId}/contact", deps.JobApplication.ContactApplicant)
				}
			})
		}

		// Review routes (mixed: public reads, authenticated writes)
		if deps.Review != nil {
			r.Route("/reviews", func(r chi.Router) {
				// Public: read reviews and average ratings
				r.Get("/user/{userId}", deps.Review.ListByUser)
				r.Get("/average/{userId}", deps.Review.GetAverageRating)

				// Authenticated: create reviews and check eligibility
				r.Group(func(r chi.Router) {
					r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
					r.Use(middleware.NoCache)
					r.Post("/", deps.Review.CreateReview)
					r.Get("/can-review/{proposalId}", deps.Review.CanReview)
				})
			})
		}

		// Report routes (authenticated)
		if deps.Report != nil {
			r.Route("/reports", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Post("/", deps.Report.CreateReport)
				r.Get("/mine", deps.Report.ListMyReports)
			})
		}

		// Social link routes
		if deps.SocialLink != nil {
			// Public: read social links
			r.Get("/profiles/{userId}/social-links", deps.SocialLink.ListPublicSocialLinks)

			// Authenticated: manage own social links
			r.Route("/profile/social-links", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Get("/", deps.SocialLink.ListMySocialLinks)
				r.Put("/", deps.SocialLink.UpsertSocialLink)
				r.Delete("/{platform}", deps.SocialLink.DeleteSocialLink)
			})
		}

		// Portfolio routes (mixed: public reads, authenticated writes)
		if deps.Portfolio != nil {
			// Public: read portfolio
			r.Get("/portfolio/user/{userId}", deps.Portfolio.ListPortfolioByUser)
			r.Get("/portfolio/{id}", deps.Portfolio.GetPortfolioItem)

			// Authenticated: manage own portfolio
			r.Route("/portfolio", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Post("/", deps.Portfolio.CreatePortfolioItem)
				r.Put("/reorder", deps.Portfolio.ReorderPortfolio)
				r.Put("/{id}", deps.Portfolio.UpdatePortfolioItem)
				r.Delete("/{id}", deps.Portfolio.DeletePortfolioItem)
			})
		}

		// Call routes (authenticated)
		if deps.Call != nil {
			r.Route("/calls", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Post("/initiate", deps.Call.InitiateCall)
				r.Post("/{id}/accept", deps.Call.AcceptCall)
				r.Post("/{id}/decline", deps.Call.DeclineCall)
				r.Post("/{id}/end", deps.Call.EndCall)
			})
		}

		// Payment info routes — all served by Embedded Components now.
		if deps.Embedded != nil {
			r.Route("/payment-info", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Post("/account-session", deps.Embedded.CreateAccountSession)
				r.Delete("/account-session", deps.Embedded.ResetAccount)
				r.Get("/account-status", deps.Embedded.GetAccountStatus)
			})
		}

		// Notification routes (authenticated)
		if deps.Notification != nil {
			r.Route("/notifications", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Get("/", deps.Notification.ListNotifications)
				r.Get("/unread-count", deps.Notification.GetUnreadCount)
				r.Post("/{id}/read", deps.Notification.MarkAsRead)
				r.Post("/read-all", deps.Notification.MarkAllAsRead)
				r.Delete("/{id}", deps.Notification.DeleteNotification)
				r.Get("/preferences", deps.Notification.GetPreferences)
				r.Put("/preferences", deps.Notification.UpdatePreferences)
				r.Post("/device-token", deps.Notification.RegisterDeviceToken)
			})
		}

		// Identity documents are now handled by Stripe Embedded Components —
		// no custom upload/list/delete endpoints needed.

		// Wallet routes (authenticated)
		if deps.Wallet != nil {
			r.Route("/wallet", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Get("/", deps.Wallet.GetWallet)
				r.Post("/payout", deps.Wallet.RequestPayout)
			})
		}

		// Stripe routes
		if deps.Stripe != nil {
			r.Route("/stripe", func(r chi.Router) {
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
				r.Use(middleware.NoCache)
				r.Get("/config", deps.Stripe.GetConfig)
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
				r.Use(middleware.Auth(deps.TokenService, deps.SessionService))
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

				// Media moderation endpoints
				r.Get("/media", deps.Admin.ListMedia)
				r.Get("/media/{id}", deps.Admin.GetMediaDetail)
				r.Post("/media/{id}/approve", deps.Admin.ApproveMedia)
				r.Post("/media/{id}/reject", deps.Admin.RejectMedia)
				r.Delete("/media/{id}", deps.Admin.DeleteMedia)

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
