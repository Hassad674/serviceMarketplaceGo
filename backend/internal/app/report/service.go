package report

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ServiceDeps groups the dependencies for the report service.
//
// Messages is narrowed to MessageReader — the report service only
// validates the target message exists (GetMessage); every mutation
// path is owned by the messaging app service.
type ServiceDeps struct {
	Reports      repository.ReportRepository
	Users        repository.UserRepository
	Messages     repository.MessageReader
	Jobs         repository.JobRepository
	Applications repository.JobApplicationRepository
}

// Service orchestrates report use cases.
type Service struct {
	reports       repository.ReportRepository
	users         repository.UserRepository
	messages      repository.MessageReader
	jobs          repository.JobRepository
	applications  repository.JobApplicationRepository
	adminNotifier service.AdminNotifierService
}

// SetAdminNotifier sets the admin notifier after construction.
func (s *Service) SetAdminNotifier(n service.AdminNotifierService) {
	s.adminNotifier = n
}

// NewService creates a new report service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		reports:      deps.Reports,
		users:        deps.Users,
		messages:     deps.Messages,
		jobs:         deps.Jobs,
		applications: deps.Applications,
	}
}

// CreateReportInput contains the data needed to create a report.
type CreateReportInput struct {
	ReporterID     uuid.UUID
	TargetType     string
	TargetID       uuid.UUID
	ConversationID uuid.UUID
	Reason         string
	Description    string
}

// CreateReport validates the target and persists a new report.
func (s *Service) CreateReport(ctx context.Context, in CreateReportInput) (*domain.Report, error) {
	targetType := domain.TargetType(in.TargetType)

	// Validate target exists
	switch targetType {
	case domain.TargetUser:
		if _, err := s.users.GetByID(ctx, in.TargetID); err != nil {
			return nil, fmt.Errorf("get target user: %w", err)
		}
	case domain.TargetMessage:
		if _, err := s.messages.GetMessage(ctx, in.TargetID); err != nil {
			return nil, fmt.Errorf("get target message: %w", err)
		}
	case domain.TargetJob:
		if _, err := s.jobs.GetByID(ctx, in.TargetID); err != nil {
			return nil, fmt.Errorf("get target job: %w", err)
		}
	case domain.TargetApplication:
		if _, err := s.applications.GetByID(ctx, in.TargetID); err != nil {
			return nil, fmt.Errorf("get target application: %w", err)
		}
	}

	// Check for existing pending report
	hasPending, err := s.reports.HasPendingReport(ctx, in.ReporterID, in.TargetType, in.TargetID)
	if err != nil {
		return nil, fmt.Errorf("check pending report: %w", err)
	}
	if hasPending {
		return nil, domain.ErrAlreadyReported
	}

	// Create domain entity (validates business rules)
	r, err := domain.NewReport(domain.NewReportInput{
		ReporterID:     in.ReporterID,
		TargetType:     targetType,
		TargetID:       in.TargetID,
		ConversationID: in.ConversationID,
		Reason:         domain.Reason(in.Reason),
		Description:    in.Description,
	})
	if err != nil {
		return nil, err
	}

	if err := s.reports.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("persist report: %w", err)
	}

	if s.adminNotifier != nil {
		// Detach from the request context so cancellation does not
		// propagate (admin notification must land even if the user
		// disconnects after a successful POST), but keep the trace
		// identifiers so the increment can be correlated. Closes
		// gosec G118: parent is request-scoped + WithoutCancel.
		bg := context.WithoutCancel(ctx)
		go func() {
			nCtx, cancel := context.WithTimeout(bg, 5*time.Second)
			defer cancel()
			if err := s.adminNotifier.IncrementAll(nCtx, service.AdminNotifReports); err != nil {
				slog.Error("admin notifier: increment reports", "error", err)
			}
		}()
	}

	return r, nil
}

// ListMyReports returns reports filed by the given user.
func (s *Service) ListMyReports(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*domain.Report, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.reports.ListByReporter(ctx, userID, cursorStr, limit)
}

// ListPendingReports returns pending reports for admin review.
func (s *Service) ListPendingReports(ctx context.Context, cursorStr string, limit int) ([]*domain.Report, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.reports.ListByStatus(ctx, string(domain.StatusPending), cursorStr, limit)
}

// ResolveReport updates a report's status (for admin use).
func (s *Service) ResolveReport(ctx context.Context, id uuid.UUID, status string, adminNote string, resolvedBy uuid.UUID) error {
	return s.reports.UpdateStatus(ctx, id, status, adminNote, resolvedBy)
}
