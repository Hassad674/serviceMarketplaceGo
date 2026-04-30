package admin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	organizationapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// DashboardStats holds aggregated statistics for the admin dashboard.
type DashboardStats struct {
	TotalUsers         int
	UsersByRole        map[string]int
	ActiveUsers        int
	SuspendedUsers     int
	BannedUsers        int
	TotalProposals     int
	ActiveProposals    int
	TotalJobs          int
	OpenJobs           int
	TotalOrganizations int
	PendingInvitations int
	RecentSignups      []*user.User
}

// ServiceDeps groups dependencies for the admin Service.
type ServiceDeps struct {
	Users              repository.UserRepository
	Reports            repository.ReportRepository
	Reviews            repository.ReviewRepository
	Jobs               repository.JobRepository
	Applications       repository.JobApplicationRepository
	Proposals          repository.ProposalRepository
	AdminConversations repository.AdminConversationRepository
	MediaRepo          repository.MediaRepository
	ModerationRepo     repository.AdminModerationRepository
	ModerationResults  repository.ModerationResultsRepository
	Audit              repository.AuditRepository
	StorageSvc         portservice.StorageService
	SessionSvc         portservice.SessionService
	Broadcaster        portservice.MessageBroadcaster
	AdminNotifier      portservice.AdminNotifierService

	// Organization team management (Phase 6). All four are optional
	// at build time but must be set together for the admin team
	// endpoints to work; otherwise the handlers return a 500.
	Orgs           repository.OrganizationRepository
	OrgMembers     repository.OrganizationMemberRepository
	OrgInvitations repository.OrganizationInvitationRepository
	Membership     *organizationapp.MembershipService
	Invitation     *organizationapp.InvitationService
}

type Service struct {
	users              repository.UserRepository
	reports            repository.ReportRepository
	reviews            repository.ReviewRepository
	jobs               repository.JobRepository
	applications       repository.JobApplicationRepository
	proposals          repository.ProposalRepository
	adminConversations repository.AdminConversationRepository
	mediaRepo          repository.MediaRepository
	moderationRepo     repository.AdminModerationRepository
	moderationResults  repository.ModerationResultsRepository
	audit              repository.AuditRepository
	storageSvc         portservice.StorageService
	sessionSvc         portservice.SessionService
	broadcaster        portservice.MessageBroadcaster
	adminNotifier      portservice.AdminNotifierService

	orgs           repository.OrganizationRepository
	orgMembers     repository.OrganizationMemberRepository
	orgInvitations repository.OrganizationInvitationRepository
	membership     *organizationapp.MembershipService
	invitation     *organizationapp.InvitationService
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		users:              deps.Users,
		reports:            deps.Reports,
		reviews:            deps.Reviews,
		jobs:               deps.Jobs,
		applications:       deps.Applications,
		proposals:          deps.Proposals,
		adminConversations: deps.AdminConversations,
		mediaRepo:          deps.MediaRepo,
		moderationRepo:     deps.ModerationRepo,
		moderationResults:  deps.ModerationResults,
		audit:              deps.Audit,
		storageSvc:         deps.StorageSvc,
		sessionSvc:         deps.SessionSvc,
		broadcaster:        deps.Broadcaster,
		adminNotifier:      deps.AdminNotifier,
		orgs:               deps.Orgs,
		orgMembers:         deps.OrgMembers,
		orgInvitations:     deps.OrgInvitations,
		membership:         deps.Membership,
		invitation:         deps.Invitation,
	}
}

// GetNotificationCounters returns all notification counters for the given admin.
func (s *Service) GetNotificationCounters(ctx context.Context, adminID uuid.UUID) (map[string]int64, error) {
	if s.adminNotifier == nil {
		return make(map[string]int64), nil
	}
	counters, err := s.adminNotifier.GetAll(ctx, adminID)
	if err != nil {
		return nil, fmt.Errorf("get admin notification counters: %w", err)
	}
	return counters, nil
}

// ResetNotificationCounter resets a single notification counter for the given admin.
func (s *Service) ResetNotificationCounter(ctx context.Context, adminID uuid.UUID, category string) error {
	if s.adminNotifier == nil {
		return nil
	}
	if err := s.adminNotifier.Reset(ctx, adminID, category); err != nil {
		return fmt.Errorf("reset admin notification counter: %w", err)
	}
	return nil
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

	totalProposals, activeProposals, err := s.proposals.CountAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: proposals: %w", err)
	}

	totalJobs, openJobs, err := s.jobs.CountAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: jobs: %w", err)
	}

	// Phase 6 — team stats. Both counts are optional at the wiring
	// level (s.orgs / s.orgInvitations may be nil on a minimal
	// deployment), so we degrade gracefully to zero rather than
	// failing the whole dashboard when the feature is not wired.
	var totalOrgs, pendingInvites int
	if s.orgs != nil {
		totalOrgs, err = s.orgs.CountAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("dashboard stats: organizations: %w", err)
		}
	}
	if s.orgInvitations != nil {
		pendingInvites, err = s.orgInvitations.CountPending(ctx)
		if err != nil {
			return nil, fmt.Errorf("dashboard stats: pending invitations: %w", err)
		}
	}

	totalUsers := 0
	for _, c := range roleCount {
		totalUsers += c
	}

	return &DashboardStats{
		TotalUsers:         totalUsers,
		UsersByRole:        roleCount,
		ActiveUsers:        statusCount["active"],
		SuspendedUsers:     statusCount["suspended"],
		BannedUsers:        statusCount["banned"],
		TotalProposals:     totalProposals,
		ActiveProposals:    activeProposals,
		TotalJobs:          totalJobs,
		OpenJobs:           openJobs,
		TotalOrganizations: totalOrgs,
		PendingInvitations: pendingInvites,
		RecentSignups:      recent,
	}, nil
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

	s.invalidateAndNotify(ctx, userID, reason)

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &userID,
		Action:       audit.ActionAdminUserSuspend,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &userID,
		Metadata: map[string]any{
			"reason":     reason,
			"expires_at": stringifyTime(expiresAt),
		},
	})
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

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &userID,
		Action:       audit.ActionAdminUserUnsuspend,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &userID,
		Metadata:     map[string]any{},
	})
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

	s.invalidateAndNotify(ctx, userID, reason)

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &userID,
		Action:       audit.ActionAdminUserBan,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &userID,
		Metadata: map[string]any{
			"reason": reason,
		},
	})
	return nil
}

// invalidateAndNotify deletes the user's session, bumps their session
// version, and broadcasts a WS event so the frontend disconnects
// immediately after a suspension or ban.
//
// The session_version bump is the SEC-05 fix: without it, mobile JWTs
// stay valid for up to 15 minutes after a ban (the JWT TTL) because
// only the cookie session in Redis was being purged. Bumping
// session_version invalidates every previously-issued JWT on the next
// request via the auth middleware's version check.
func (s *Service) invalidateAndNotify(ctx context.Context, userID uuid.UUID, reason string) {
	// SEC-05: bump the user's session_version BEFORE wiping sessions.
	// Order matters subtly: a concurrent request that reads the old
	// version + reaches the auth middleware before the bump persists
	// is still rejected because the version it reads from the JWT is
	// strictly less than the new DB value. Best-effort failure is
	// acceptable — both backends MUST already pass for the user to be
	// truly locked out, but a transient bump failure leaves the cookie
	// path still purged, which is strictly better than the pre-fix
	// state.
	if _, err := s.users.BumpSessionVersion(ctx, userID); err != nil {
		slog.Error("admin: bump session_version after suspend/ban",
			"error", err, "user_id", userID)
	}
	if s.sessionSvc != nil {
		if err := s.sessionSvc.DeleteByUserID(ctx, userID); err != nil {
			slog.Error("admin: delete sessions after suspension",
				"error", err, "user_id", userID)
		}
	}
	if s.broadcaster != nil {
		if err := s.broadcaster.BroadcastAccountSuspended(ctx, userID, reason); err != nil {
			slog.Error("admin: broadcast account_suspended",
				"error", err, "user_id", userID)
		}
	}
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

	s.logAudit(ctx, audit.NewEntryInput{
		UserID:       &userID,
		Action:       audit.ActionAdminUserUnban,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &userID,
		Metadata:     map[string]any{},
	})
	return nil
}

// logAudit writes one append-only audit row for the action just taken.
// Mirrors the auth.Service helper — failures are logged via slog and
// never returned. Audit emission is best-effort by policy.
func (s *Service) logAudit(ctx context.Context, in audit.NewEntryInput) {
	if s.audit == nil {
		return
	}
	entry, err := audit.NewEntry(in)
	if err != nil {
		slog.Warn("audit: build entry failed", "action", in.Action, "error", err)
		return
	}
	if err := s.audit.Log(ctx, entry); err != nil {
		slog.Warn("audit: insert failed", "action", in.Action, "error", err)
	}
}

// stringifyTime turns a *time.Time into its RFC3339 representation, or
// the empty string when nil. Used in audit metadata so the JSON value
// is always a primitive (jsonb-friendly).
func stringifyTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
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
