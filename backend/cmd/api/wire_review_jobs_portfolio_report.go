package main

import (
	"database/sql"

	"marketplace-backend/internal/adapter/postgres"
	jobapp "marketplace-backend/internal/app/job"
	"marketplace-backend/internal/app/messaging"
	notifapp "marketplace-backend/internal/app/notification"
	portfolioapp "marketplace-backend/internal/app/portfolio"
	projecthistoryapp "marketplace-backend/internal/app/projecthistory"
	reportapp "marketplace-backend/internal/app/report"
	reviewapp "marketplace-backend/internal/app/review"
	jobdomain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// jobsWiring carries the products of the job feature initialisation:
// the four job-side repositories (jobs, applications, views, credits),
// the app service, and both job + application HTTP handlers. Returned
// up-front so other wires (proposal service, report, admin) can reach
// into the credit / repo pointers without knowing the inner shape.
type jobsWiring struct {
	JobRepo         *postgres.JobRepository
	JobAppRepo      *postgres.JobApplicationRepository
	JobViewRepo     *postgres.JobViewRepository
	JobCreditRepo   *postgres.JobCreditRepository
	JobSvc          *jobapp.Service
	JobHandler      *handler.JobHandler
	JobAppHandler   *handler.JobApplicationHandler
}

// jobsDeps captures the upstream dependencies the job feature needs:
// the SQL pool plus the cross-feature collaborators that get pulled
// into the job service ServiceDeps (users, organizations, profiles,
// messaging).
type jobsDeps struct {
	DB               *sql.DB
	UserRepo         repository.UserRepository
	OrganizationRepo repository.OrganizationRepository
	ProfileRepo      repository.ProfileRepository
	MessagingSvc     *messaging.Service
}

// wireJobs brings up the job feature: the four job-side repositories,
// the app service, and both HTTP handlers. The credit repository
// drives a lazy weekly refill from its GetOrCreate method — every
// read on an org whose pool has aged past RefillPeriod floor-bumps
// the balance back up to WeeklyQuota atomically. No cron, no
// background worker, self-healing after downtime.
func wireJobs(deps jobsDeps) jobsWiring {
	// Job feature
	jobRepo := postgres.NewJobRepository(deps.DB)
	jobAppRepo := postgres.NewJobApplicationRepository(deps.DB)
	jobViewRepo := postgres.NewJobViewRepository(deps.DB)
	// The credit repository drives a lazy weekly refill from its
	// GetOrCreate method — every read on an org whose pool has aged
	// past RefillPeriod floor-bumps the balance back up to WeeklyQuota
	// atomically. No cron, no background worker, self-healing after
	// downtime.
	jobCreditRepo := postgres.NewJobCreditRepository(deps.DB, jobdomain.WeeklyQuota, jobdomain.RefillPeriod)
	jobSvc := jobapp.NewService(jobapp.ServiceDeps{
		Jobs:          jobRepo,
		Applications:  jobAppRepo,
		Users:         deps.UserRepo,
		Organizations: deps.OrganizationRepo,
		Profiles:      deps.ProfileRepo,
		Messages:      deps.MessagingSvc,
		JobViews:      jobViewRepo,
		Credits:       jobCreditRepo,
	})

	jobHandler := handler.NewJobHandler(jobSvc)
	jobAppHandler := handler.NewJobApplicationHandler(jobSvc)

	return jobsWiring{
		JobRepo:       jobRepo,
		JobAppRepo:    jobAppRepo,
		JobViewRepo:   jobViewRepo,
		JobCreditRepo: jobCreditRepo,
		JobSvc:        jobSvc,
		JobHandler:    jobHandler,
		JobAppHandler: jobAppHandler,
	}
}

// portfolioWiring carries the products of the portfolio feature
// initialisation: the repository (consumed only inside this wire so
// not exposed) and the HTTP handler. The portfolio service is also
// kept private — no other feature reads it directly.
type portfolioWiring struct {
	Handler *handler.PortfolioHandler
}

// wirePortfolio brings up the portfolio feature: a thin repository +
// service + handler chain with no cross-feature dependencies.
func wirePortfolio(db *sql.DB) portfolioWiring {
	// Portfolio feature
	portfolioRepo := postgres.NewPortfolioRepository(db)
	portfolioSvc := portfolioapp.NewService(portfolioapp.ServiceDeps{
		Portfolios: portfolioRepo,
	})
	portfolioHandler := handler.NewPortfolioHandler(portfolioSvc)
	return portfolioWiring{Handler: portfolioHandler}
}

// projectHistoryWiring carries the project history HTTP handler.
// The service composes proposal + review reads for the public
// provider profile page; both repositories are passed in by main.go
// because they are owned by separate wires.
type projectHistoryWiring struct {
	Handler *handler.ProjectHistoryHandler
}

// projectHistoryDeps captures the proposal + review repositories the
// project history feature reads from.
type projectHistoryDeps struct {
	ProposalRepo repository.ProposalRepository
	ReviewRepo   repository.ReviewRepository
}

// wireProjectHistory brings up the project history feature
// (orchestrates proposal + review reads for the public provider
// profile page).
func wireProjectHistory(deps projectHistoryDeps) projectHistoryWiring {
	// Project history feature (orchestrates proposal + review reads for the
	// public provider profile page).
	projectHistorySvc := projecthistoryapp.NewService(projecthistoryapp.ServiceDeps{
		Proposals: deps.ProposalRepo,
		Reviews:   deps.ReviewRepo,
	})
	projectHistoryHandler := handler.NewProjectHistoryHandler(projectHistorySvc)
	return projectHistoryWiring{Handler: projectHistoryHandler}
}

// reviewRepoWiring carries the review repository so other features
// (project history, referrer reputation, client profile read service,
// admin) can reach into it without going through the review service.
// The repo is built early so cross-feature wires that only need
// reads do not have to wait for the review app service.
type reviewRepoWiring struct {
	Repo *postgres.ReviewRepository
}

// wireReviewRepo brings up the review repository in isolation. The
// app service is wired separately by wireReviewService once the
// notification feature exists.
func wireReviewRepo(db *sql.DB) reviewRepoWiring {
	// Review feature
	return reviewRepoWiring{Repo: postgres.NewReviewRepository(db)}
}

// reviewServiceWiring carries the review app service + handler. Runs
// AFTER the notification feature has been wired because the review
// service emits notifications on every review submission.
type reviewServiceWiring struct {
	Svc     *reviewapp.Service
	Handler *handler.ReviewHandler
}

// reviewServiceDeps captures the upstream services + repos the review
// app service depends on.
type reviewServiceDeps struct {
	ReviewRepo    repository.ReviewRepository
	ProposalRepo  repository.ProposalRepository
	UserRepo      repository.UserRepository
	Notifications *notifapp.Service
}

// wireReviewService brings up the review app service and HTTP
// handler. Must run AFTER wireNotificationFeature so the service
// can fire submission events through the same notif pipeline as the
// rest of the app.
func wireReviewService(deps reviewServiceDeps) reviewServiceWiring {
	reviewSvc := reviewapp.NewService(reviewapp.ServiceDeps{
		Reviews:       deps.ReviewRepo,
		Proposals:     deps.ProposalRepo,
		Users:         deps.UserRepo,
		Notifications: deps.Notifications,
	})
	reviewHandler := handler.NewReviewHandler(reviewSvc)
	return reviewServiceWiring{
		Svc:     reviewSvc,
		Handler: reviewHandler,
	}
}

// reportWiring carries the products of the report feature: the
// repository (consumed by admin) and the HTTP handler.
type reportWiring struct {
	Repo    *postgres.ReportRepository
	Svc     *reportapp.Service
	Handler *handler.ReportHandler
}

// reportDeps captures the upstream dependencies the report feature
// needs: the SQL pool plus the user / message / job / application
// repositories the report service reads from.
type reportDeps struct {
	DB              *sql.DB
	UserRepo        repository.UserRepository
	MessageRepo     repository.MessageRepository
	JobRepo         repository.JobRepository
	JobAppRepo      repository.JobApplicationRepository
}

// wireReport brings up the report feature: repository, app service,
// and HTTP handler. Reports cross-cut every other resource type
// (users, messages, jobs, applications) so the deps surface is
// wide.
func wireReport(deps reportDeps) reportWiring {
	// Report feature
	reportRepo := postgres.NewReportRepository(deps.DB)
	reportSvc := reportapp.NewService(reportapp.ServiceDeps{
		Reports:      reportRepo,
		Users:        deps.UserRepo,
		Messages:     deps.MessageRepo,
		Jobs:         deps.JobRepo,
		Applications: deps.JobAppRepo,
	})
	reportHandler := handler.NewReportHandler(reportSvc)
	return reportWiring{
		Repo:    reportRepo,
		Svc:     reportSvc,
		Handler: reportHandler,
	}
}
