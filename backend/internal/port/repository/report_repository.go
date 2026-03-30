package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/report"
)

// ReportRepository defines persistence operations for reports.
type ReportRepository interface {
	Create(ctx context.Context, r *report.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error)
	ListByStatus(ctx context.Context, status string, cursor string, limit int) ([]*report.Report, string, error)
	ListByReporter(ctx context.Context, reporterID uuid.UUID, cursor string, limit int) ([]*report.Report, string, error)
	ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]*report.Report, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, adminNote string, resolvedBy uuid.UUID) error
	HasPendingReport(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error)
}
