package report

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/port/repository"
)

// ServiceDeps groups the dependencies for the report service.
type ServiceDeps struct {
	Reports  repository.ReportRepository
	Users    repository.UserRepository
	Messages repository.MessageRepository
}

// Service orchestrates report use cases.
type Service struct {
	reports  repository.ReportRepository
	users    repository.UserRepository
	messages repository.MessageRepository
}

// NewService creates a new report service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		reports:  deps.Reports,
		users:    deps.Users,
		messages: deps.Messages,
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
