package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/report"
)

func (s *Service) loadJobPendingReportCounts(ctx context.Context, jobs []AdminJob) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(jobs))
	if len(jobs) == 0 {
		return counts, nil
	}

	ids := make([]uuid.UUID, len(jobs))
	for i, j := range jobs {
		ids[i] = j.ID
	}

	rows, err := s.db.QueryContext(ctx, queryAdminJobPendingReportCounts, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("load job pending report counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var targetID uuid.UUID
		var count int
		if err := rows.Scan(&targetID, &count); err != nil {
			return nil, fmt.Errorf("scan job report count: %w", err)
		}
		counts[targetID] = count
	}

	return counts, nil
}

func (s *Service) loadApplicationPendingReportCounts(ctx context.Context, apps []AdminJobApplication) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(apps))
	if len(apps) == 0 {
		return counts, nil
	}

	ids := make([]uuid.UUID, len(apps))
	for i, a := range apps {
		ids[i] = a.ID
	}

	rows, err := s.db.QueryContext(ctx, queryAdminApplicationPendingReportCounts, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("load application pending report counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var targetID uuid.UUID
		var count int
		if err := rows.Scan(&targetID, &count); err != nil {
			return nil, fmt.Errorf("scan application report count: %w", err)
		}
		counts[targetID] = count
	}

	return counts, nil
}

// ListJobReports returns all reports targeting a specific job.
func (s *Service) ListJobReports(ctx context.Context, jobID uuid.UUID) ([]*report.Report, error) {
	reports, err := s.reports.ListByTarget(ctx, string(report.TargetJob), jobID)
	if err != nil {
		return nil, fmt.Errorf("list job reports: %w", err)
	}
	return reports, nil
}

const queryAdminJobPendingReportCounts = `
	SELECT r.target_id, COUNT(*)
	FROM reports r
	WHERE r.target_type = 'job'
		AND r.status = 'pending'
		AND r.target_id = ANY($1)
	GROUP BY r.target_id`

const queryAdminApplicationPendingReportCounts = `
	SELECT r.target_id, COUNT(*)
	FROM reports r
	WHERE r.target_type = 'job_application'
		AND r.status = 'pending'
		AND r.target_id = ANY($1)
	GROUP BY r.target_id`
