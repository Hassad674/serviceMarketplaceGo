package admin

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// DashboardStats holds aggregated statistics for the admin dashboard.
type DashboardStats struct {
	TotalUsers      int
	UsersByRole     map[string]int
	ActiveUsers     int
	SuspendedUsers  int
	BannedUsers     int
	TotalProposals  int
	ActiveProposals int
	TotalJobs       int
	OpenJobs        int
	RecentSignups   []*user.User
}

type Service struct {
	users   repository.UserRepository
	reports repository.ReportRepository
	db      *sql.DB
}

func NewService(users repository.UserRepository, reports repository.ReportRepository, db *sql.DB) *Service {
	return &Service{users: users, reports: reports, db: db}
}

func (s *Service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	roleCount, err := s.users.CountByRole(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: count by role: %w", err)
	}

	statusCount, err := s.users.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: count by status: %w", err)
	}

	recent, err := s.users.RecentSignups(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: recent signups: %w", err)
	}

	totalProposals, activeProposals, err := s.countProposals(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: proposals: %w", err)
	}

	totalJobs, openJobs, err := s.countJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: jobs: %w", err)
	}

	totalUsers := 0
	for _, c := range roleCount {
		totalUsers += c
	}

	return &DashboardStats{
		TotalUsers:      totalUsers,
		UsersByRole:     roleCount,
		ActiveUsers:     statusCount["active"],
		SuspendedUsers:  statusCount["suspended"],
		BannedUsers:     statusCount["banned"],
		TotalProposals:  totalProposals,
		ActiveProposals: activeProposals,
		TotalJobs:       totalJobs,
		OpenJobs:        openJobs,
		RecentSignups:   recent,
	}, nil
}

func (s *Service) countProposals(ctx context.Context) (total int, active int, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM proposals").Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("count total proposals: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM proposals WHERE status IN ('paid', 'active', 'completion_requested')",
	).Scan(&active)
	if err != nil {
		return 0, 0, fmt.Errorf("count active proposals: %w", err)
	}
	return total, active, nil
}

func (s *Service) countJobs(ctx context.Context) (total int, open int, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs").Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("count total jobs: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM jobs WHERE status = 'open'",
	).Scan(&open)
	if err != nil {
		return 0, 0, fmt.Errorf("count open jobs: %w", err)
	}
	return total, open, nil
}

func (s *Service) ListUsers(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, int, error) {
	users, nextCursor, err := s.users.ListAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list admin users: %w", err)
	}

	count, err := s.users.CountAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("count admin users: %w", err)
	}

	return users, nextCursor, count, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}
	return u, nil
}

func (s *Service) SuspendUser(ctx context.Context, userID uuid.UUID, reason string, expiresAt *time.Time) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("suspend user: %w", err)
	}

	u.Suspend(reason, expiresAt)

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("suspend user: save: %w", err)
	}
	return nil
}

func (s *Service) UnsuspendUser(ctx context.Context, userID uuid.UUID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("unsuspend user: %w", err)
	}

	u.Unsuspend()

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("unsuspend user: save: %w", err)
	}
	return nil
}

func (s *Service) BanUser(ctx context.Context, userID uuid.UUID, reason string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}

	u.Ban(reason)

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("ban user: save: %w", err)
	}
	return nil
}

func (s *Service) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("unban user: %w", err)
	}

	u.Unban()

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("unban user: save: %w", err)
	}
	return nil
}

func (s *Service) ListConversationReports(ctx context.Context, conversationID uuid.UUID) ([]*report.Report, error) {
	reports, err := s.reports.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list conversation reports: %w", err)
	}
	return reports, nil
}

func (s *Service) ListUserReports(ctx context.Context, userID uuid.UUID) ([]*report.Report, []*report.Report, error) {
	against, filed, err := s.reports.ListByUserInvolved(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list user reports: %w", err)
	}
	return against, filed, nil
}

func (s *Service) ResolveReport(ctx context.Context, reportID uuid.UUID, status string, adminNote string, resolvedBy uuid.UUID) error {
	if status != string(report.StatusResolved) && status != string(report.StatusDismissed) {
		return fmt.Errorf("resolve report: invalid status %q", status)
	}
	if err := s.reports.UpdateStatus(ctx, reportID, status, adminNote, resolvedBy); err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}
	return nil
}
