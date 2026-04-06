package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/report"
)

// ListJobReports returns all reports targeting a specific job.
func (s *Service) ListJobReports(ctx context.Context, jobID uuid.UUID) ([]*report.Report, error) {
	reports, err := s.reports.ListByTarget(ctx, string(report.TargetJob), jobID)
	if err != nil {
		return nil, fmt.Errorf("list job reports: %w", err)
	}
	return reports, nil
}
