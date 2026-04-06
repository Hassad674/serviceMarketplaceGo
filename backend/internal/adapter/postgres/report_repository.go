package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/report"
	"marketplace-backend/pkg/cursor"
)

// ReportRepository implements repository.ReportRepository using PostgreSQL.
type ReportRepository struct {
	db *sql.DB
}

// NewReportRepository creates a new PostgreSQL-backed report repository.
func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, rp *report.Report) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var convID *uuid.UUID
	if rp.ConversationID != uuid.Nil {
		convID = &rp.ConversationID
	}

	_, err := r.db.ExecContext(ctx, queryInsertReport,
		rp.ID, rp.ReporterID, string(rp.TargetType), rp.TargetID, convID,
		string(rp.Reason), rp.Description, string(rp.Status), rp.AdminNote,
		rp.ResolvedAt, rp.ResolvedBy, rp.CreatedAt, rp.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return report.ErrAlreadyReported
		}
		return fmt.Errorf("insert report: %w", err)
	}
	return nil
}

func (r *ReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rp, err := scanReport(r.db.QueryRowContext(ctx, queryGetReportByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, report.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get report by id: %w", err)
	}
	return rp, nil
}

func (r *ReportRepository) ListByStatus(ctx context.Context, status string, cursorStr string, limit int) ([]*report.Report, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListReportsByStatusFirst, status, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListReportsByStatusWithCursor, status, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list reports by status: %w", err)
	}
	defer rows.Close()

	return collectReports(rows, limit)
}

func (r *ReportRepository) ListByReporter(ctx context.Context, reporterID uuid.UUID, cursorStr string, limit int) ([]*report.Report, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListReportsByReporterFirst, reporterID, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListReportsByReporterWithCursor, reporterID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list reports by reporter: %w", err)
	}
	defer rows.Close()

	return collectReports(rows, limit)
}

func (r *ReportRepository) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]*report.Report, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListReportsByTarget, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("list reports by target: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		rp, scanErr := scanReport(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan report: %w", scanErr)
		}
		reports = append(reports, rp)
	}
	return reports, nil
}

func (r *ReportRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, adminNote string, resolvedBy uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryUpdateReportStatus, id, status, adminNote, resolvedBy)
	if err != nil {
		return fmt.Errorf("update report status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return report.ErrNotFound
	}
	return nil
}

func (r *ReportRepository) HasPendingReport(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	err := r.db.QueryRowContext(ctx, queryHasPendingReport, reporterID, targetType, targetID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check pending report: %w", err)
	}
	return exists, nil
}

func (r *ReportRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*report.Report, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListReportsByConversation, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list reports by conversation: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		rp, scanErr := scanReport(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan report: %w", scanErr)
		}
		reports = append(reports, rp)
	}
	return reports, nil
}

func (r *ReportRepository) ListByUserInvolved(ctx context.Context, userID uuid.UUID) ([]*report.Report, []*report.Report, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	againstRows, err := r.db.QueryContext(ctx, queryListReportsAgainstUser, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list reports against user: %w", err)
	}
	defer againstRows.Close()

	var against []*report.Report
	for againstRows.Next() {
		rp, scanErr := scanReport(againstRows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan report: %w", scanErr)
		}
		against = append(against, rp)
	}

	filedRows, err := r.db.QueryContext(ctx, queryListReportsFiledByUser, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list reports filed by user: %w", err)
	}
	defer filedRows.Close()

	var filed []*report.Report
	for filedRows.Next() {
		rp, scanErr := scanReport(filedRows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan report: %w", scanErr)
		}
		filed = append(filed, rp)
	}

	return against, filed, nil
}

func (r *ReportRepository) PendingCountsByTargets(ctx context.Context, targetType string, targetIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	counts := make(map[uuid.UUID]int, len(targetIDs))
	if len(targetIDs) == 0 {
		return counts, nil
	}

	rows, err := r.db.QueryContext(ctx, queryPendingCountsByTargets, targetType, pq.Array(targetIDs))
	if err != nil {
		return nil, fmt.Errorf("pending counts by targets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var targetID uuid.UUID
		var count int
		if err := rows.Scan(&targetID, &count); err != nil {
			return nil, fmt.Errorf("scan pending count: %w", err)
		}
		counts[targetID] = count
	}

	return counts, nil
}

// reportScanner interface satisfied by both *sql.Row and *sql.Rows.
type reportScanner interface {
	Scan(dest ...any) error
}

func scanReport(s reportScanner) (*report.Report, error) {
	var rp report.Report
	var convID *uuid.UUID
	err := s.Scan(
		&rp.ID, &rp.ReporterID, &rp.TargetType, &rp.TargetID, &convID,
		&rp.Reason, &rp.Description, &rp.Status, &rp.AdminNote,
		&rp.ResolvedAt, &rp.ResolvedBy, &rp.CreatedAt, &rp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if convID != nil {
		rp.ConversationID = *convID
	}
	return &rp, nil
}

func collectReports(rows *sql.Rows, limit int) ([]*report.Report, string, error) {
	var reports []*report.Report
	for rows.Next() {
		rp, scanErr := scanReport(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("scan report: %w", scanErr)
		}
		reports = append(reports, rp)
	}

	var nextCursor string
	if len(reports) > limit {
		last := reports[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		reports = reports[:limit]
	}

	return reports, nextCursor, nil
}
