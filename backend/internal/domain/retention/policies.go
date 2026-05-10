package retention

import "time"

// DefaultPolicies returns the canonical set of retention policies the
// scheduler enforces in production. Centralising them in the domain
// layer keeps the privacy-policy guarantees auditable in a single
// place — no policy lives in adapter or wire code.
//
// Each entry's MaxAge maps onto the audit table in
// `gdpr-audit.md` Section 7 ("Proposed retention matrix for the
// privacy policy") and the corresponding row in `gdpr-roadmap.md`
// Phase B.1.
//
// The function takes overrides so an operator can tune any single
// MaxAge without forking the function. A zero override means "use
// the default". This is the only configurability surface — tables,
// columns, strategies and archive targets are NEVER overridable
// because changing them silently would change the privacy contract.
type Overrides struct {
	MessagesMaxAge         time.Duration
	NotificationsMaxAge    time.Duration
	DeviceTokensMaxAge     time.Duration
	SearchQueriesMaxAge    time.Duration
	AuditLogsHotMaxAge     time.Duration
	UserSessionsRevokedAge time.Duration
	// AuditLogsColdMaxAge controls how old an audit_logs_archive row
	// must be before the B.2 sweep dumps its payload to R2 and frees
	// the Postgres row. Counted from `archived_at`, so the total
	// retention is hot+cold (defaults: 24mo hot + 24mo cold = 4y).
	AuditLogsColdMaxAge time.Duration
}

// Default values, in one place so the test suite can pin them.
const (
	DefaultMessagesMaxAge         = 3 * 365 * 24 * time.Hour // 3 years
	DefaultNotificationsMaxAge    = 90 * 24 * time.Hour      // 90 days
	DefaultDeviceTokensMaxAge     = 60 * 24 * time.Hour      // 60 days inactivity
	DefaultSearchQueriesMaxAge    = 12 * 30 * 24 * time.Hour // ~12 months
	DefaultAuditLogsHotMaxAge     = 24 * 30 * 24 * time.Hour // ~24 months
	DefaultUserSessionsRevokedAge = 30 * 24 * time.Hour      // 30 days post-revoke
	// DefaultAuditLogsColdMaxAge is the additional time a row sits
	// in audit_logs_archive (Postgres) before the B.2 sweep writes
	// it to R2 and deletes the Postgres copy. Same magnitude as the
	// hot tier — together they preserve the legal/compliance trail
	// for ~4 years of online (Postgres+R2) access.
	DefaultAuditLogsColdMaxAge = 24 * 30 * 24 * time.Hour // ~24 months
)

// DefaultPolicies returns the five Phase B.1 policies as a fresh
// slice. The slice is intentionally not memoized: tests build their
// own scheduler with shorter durations and the call site is a single
// boot path, so the allocation cost is negligible.
func DefaultPolicies(o Overrides) []Policy {
	pick := func(override, fallback time.Duration) time.Duration {
		if override > 0 {
			return override
		}
		return fallback
	}
	return []Policy{
		{
			Name:      "messages_3y",
			Table:     "messages",
			AgeColumn: "created_at",
			MaxAge:    pick(o.MessagesMaxAge, DefaultMessagesMaxAge),
			Strategy:  StrategyDelete,
		},
		{
			Name:      "notifications_90d",
			Table:     "notifications",
			AgeColumn: "created_at",
			MaxAge:    pick(o.NotificationsMaxAge, DefaultNotificationsMaxAge),
			Strategy:  StrategyDelete,
		},
		{
			Name:      "device_tokens_60d_inactive",
			Table:     "device_tokens",
			AgeColumn: "last_seen_at",
			MaxAge:    pick(o.DeviceTokensMaxAge, DefaultDeviceTokensMaxAge),
			Strategy:  StrategyDelete,
		},
		{
			Name:             "search_queries_12mo_anonymize",
			Table:            "search_queries",
			AgeColumn:        "created_at",
			MaxAge:           pick(o.SearchQueriesMaxAge, DefaultSearchQueriesMaxAge),
			Strategy:         StrategyAnonymize,
			AnonymizeColumns: []string{"user_id", "session_id"},
		},
		{
			Name:         "audit_logs_24mo_archive",
			Table:        "audit_logs",
			AgeColumn:    "created_at",
			MaxAge:       pick(o.AuditLogsHotMaxAge, DefaultAuditLogsHotMaxAge),
			Strategy:     StrategyArchive,
			ArchiveTable: "audit_logs_archive",
		},
		{
			// B.4: revoke-and-stale sessions older than 30 days are
			// hard-deleted to keep the table small. Active sessions
			// are kept until expiry. The adapter enforces both
			// revoked_at < cutoff AND expires_at < cutoff.
			Name:      "user_sessions_revoked_30d_delete",
			Table:     "user_sessions",
			AgeColumn: "revoked_at",
			MaxAge:    pick(o.UserSessionsRevokedAge, DefaultUserSessionsRevokedAge),
			Strategy:  StrategyDeleteRevokedSessions,
		},
		{
			// B.2: audit_logs_archive rows whose `archived_at` is
			// older than the cold-tier cutoff are dumped to R2 as
			// gzipped JSONL and then hard-deleted from Postgres.
			// The adapter handles the two-phase split (upload +
			// stamp r2_key on tick N, hard-delete on tick N+1) so
			// a crash mid-flight never loses or duplicates rows.
			Name:      "audit_logs_archive_to_r2_24mo",
			Table:     "audit_logs_archive",
			AgeColumn: "archived_at",
			MaxAge:    pick(o.AuditLogsColdMaxAge, DefaultAuditLogsColdMaxAge),
			Strategy:  StrategyArchiveToR2,
		},
	}
}
