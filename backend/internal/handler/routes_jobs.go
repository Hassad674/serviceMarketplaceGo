package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountJobRoutes wires the /jobs surface — jobs CRUD + the application
// counterpart endpoints that share the same {id} prefix. Application
// routes are nested here so chi resolves the {id}/applications path
// before the static /applications/mine route below.
func mountJobRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Job == nil {
		return
	}
	r.Route("/jobs", func(r chi.Router) {
		r.Use(auth)
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
