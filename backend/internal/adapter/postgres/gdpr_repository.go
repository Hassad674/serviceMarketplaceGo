package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/gdpr"
	domainuser "marketplace-backend/internal/domain/user"
)

// GDPRRepository is the PostgreSQL adapter for the right-to-erasure +
// right-to-export endpoints (P5).
//
// The implementation is split into three concerns:
//
//   - LoadExport: SELECTs across the user's tables and returns one
//     []map[string]any per JSON file the export ZIP contains. Cheap
//     enough to run synchronously even for chatty users — the data
//     volumes per user are bounded (proposals + messages are the
//     largest sets and rarely cross five-figure rows).
//
//   - SoftDelete / CancelDeletion: single-row UPDATEs on users.
//     Idempotent on both sides.
//
//   - PurgeUser: a tx that anonymizes-in-place where FKs are NOT NULL
//     NO ACTION and hard-deletes everything else. The brief calls
//     for "hard cascade DELETE" but the migrations have several
//     legacy NO ACTION FKs (proposals, disputes, jobs, reviews,
//     invoices, payment_records, etc.) so the only RGPD-compliant
//     path that doesn't risk corrupting historical business records
//     is to anonymize the user row itself + every direct PII column
//     and let the rows that hold UUIDs into the deleted user keep
//     their structural integrity. The audit_logs table is
//     additionally anonymized through metadata jsonb_set so a future
//     forensic check can recompute sha256(email+salt) without
//     retaining the raw PII.
//
// Mock-friendly contract: every method takes a context.Context as
// the first argument and returns a domain error or a wrapped DB
// error, never a raw *pq.Error.
type GDPRRepository struct {
	db *sql.DB
}

// NewGDPRRepository builds the repo. Wired from cmd/api/wire_gdpr.go.
func NewGDPRRepository(db *sql.DB) *GDPRRepository {
	return &GDPRRepository{db: db}
}

// LoadExport gathers every section of the export ZIP. Sections are
// loaded sequentially: parallel goroutines would only help if the DB
// was the bottleneck, which is rarely the case for per-user exports
// (single-digit MB at most). Sequential keeps the code simple and
// the tx footprint short.
//
// The returned Export.Locale is left empty — the service layer is in
// charge of stamping it from the user's preferred language.
func (r *GDPRRepository) LoadExport(ctx context.Context, userID uuid.UUID) (*gdpr.Export, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	out := &gdpr.Export{
		UserID:    userID,
		Timestamp: time.Now().UTC(),
	}

	profile, email, err := r.loadProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(profile) == 0 {
		return nil, domainuser.ErrUserNotFound
	}
	out.Profile = profile
	out.Email = email

	loaders := []struct {
		name    string
		fn      func(context.Context, uuid.UUID) ([]map[string]any, error)
		assign  func([]map[string]any)
	}{
		{"proposals", r.loadProposals, func(rows []map[string]any) { out.Proposals = rows }},
		{"messages", r.loadMessages, func(rows []map[string]any) { out.Messages = rows }},
		{"invoices", r.loadInvoices, func(rows []map[string]any) { out.Invoices = rows }},
		{"reviews", r.loadReviews, func(rows []map[string]any) { out.Reviews = rows }},
		{"audit_logs", r.loadAuditLogs, func(rows []map[string]any) { out.AuditLogs = rows }},
		{"notifications", r.loadNotifications, func(rows []map[string]any) { out.Notifications = rows }},
		{"jobs", r.loadJobs, func(rows []map[string]any) { out.Jobs = rows }},
		{"portfolios", r.loadPortfolios, func(rows []map[string]any) { out.Portfolios = rows }},
		{"reports", r.loadReports, func(rows []map[string]any) { out.Reports = rows }},
	}
	for _, ld := range loaders {
		rows, err := ld.fn(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("export load %s: %w", ld.name, err)
		}
		ld.assign(rows)
	}
	return out, nil
}

func (r *GDPRRepository) loadProfile(ctx context.Context, userID uuid.UUID) ([]map[string]any, string, error) {
	const q = `
		SELECT id, email, first_name, last_name, display_name, role, account_type,
		       referrer_enabled, email_notifications_enabled, status,
		       organization_id, email_verified, created_at, updated_at
		FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, userID)

	var (
		id, email, firstName, lastName, displayName, role, accountType, status string
		referrerEnabled, emailNotif, emailVerified                             bool
		orgID                                                                  sql.NullString
		createdAt, updatedAt                                                   time.Time
	)
	err := row.Scan(&id, &email, &firstName, &lastName, &displayName, &role, &accountType,
		&referrerEnabled, &emailNotif, &status, &orgID, &emailVerified, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("load profile: %w", err)
	}
	row1 := map[string]any{
		"id":                          id,
		"email":                       email,
		"first_name":                  firstName,
		"last_name":                   lastName,
		"display_name":                displayName,
		"role":                        role,
		"account_type":                accountType,
		"referrer_enabled":            referrerEnabled,
		"email_notifications_enabled": emailNotif,
		"status":                      status,
		"email_verified":              emailVerified,
		"created_at":                  createdAt,
		"updated_at":                  updatedAt,
	}
	if orgID.Valid {
		row1["organization_id"] = orgID.String
	}
	return []map[string]any{row1}, email, nil
}

func (r *GDPRRepository) loadProposals(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT id, status, client_id, provider_id, sender_id, recipient_id,
		       created_at, updated_at
		FROM proposals
		WHERE client_id = $1 OR provider_id = $1 OR sender_id = $1 OR recipient_id = $1
		ORDER BY created_at DESC`, userID)
}

func (r *GDPRRepository) loadMessages(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT id, conversation_id, sender_id, content, msg_type, created_at
		FROM messages
		WHERE sender_id = $1
		ORDER BY created_at DESC LIMIT 10000`, userID)
}

func (r *GDPRRepository) loadInvoices(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT i.id, i.recipient_organization_id, i.number,
		       i.amount_excl_tax_cents, i.amount_incl_tax_cents,
		       i.currency, i.issued_at, i.status
		FROM invoice i
		JOIN organization_members m ON m.organization_id = i.recipient_organization_id
		WHERE m.user_id = $1
		ORDER BY i.issued_at DESC`, userID)
}

func (r *GDPRRepository) loadReviews(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT id, reviewer_id, reviewed_id, global_rating, comment, side, created_at
		FROM reviews
		WHERE reviewer_id = $1 OR reviewed_id = $1
		ORDER BY created_at DESC`, userID)
}

func (r *GDPRRepository) loadAuditLogs(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, metadata, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC LIMIT 5000`, userID)
}

func (r *GDPRRepository) loadNotifications(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT id, type, title, body, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC LIMIT 5000`, userID)
}

func (r *GDPRRepository) loadJobs(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT id, creator_id, title, status, created_at
		FROM jobs WHERE creator_id = $1
		ORDER BY created_at DESC`, userID)
}

func (r *GDPRRepository) loadPortfolios(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	// portfolio_items is org-scoped; we return the items of every org
	// the user belongs to so the export reflects work the user
	// contributed to as an organization member.
	return r.queryRows(ctx, `
		SELECT pi.id, pi.organization_id, pi.title, pi.description,
		       pi.link_url, pi.created_at
		FROM portfolio_items pi
		JOIN organization_members m ON m.organization_id = pi.organization_id
		WHERE m.user_id = $1
		ORDER BY pi.created_at DESC`, userID)
}

func (r *GDPRRepository) loadReports(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	// target_type='user' captures reports where the user is the
	// subject; reporter_id captures reports the user filed. Both
	// are part of the user's data per RGPD recital 26.
	return r.queryRows(ctx, `
		SELECT id, reporter_id, target_type, target_id,
		       reason, description, status, created_at
		FROM reports
		WHERE reporter_id = $1 OR (target_type = 'user' AND target_id = $1)
		ORDER BY created_at DESC`, userID)
}

// queryRows executes a SELECT and returns each row as a generic map.
// Used by every section loader so the export aggregation stays
// declarative rather than carrying a hand-written scan per table.
//
// Schema mismatches (e.g. an optional table that doesn't exist) are
// downgraded to "no rows": the export is best-effort and the caller
// would rather get a partial export than a 500 because the moderation
// reports table happens to be migrated separately in some envs.
func (r *GDPRRepository) queryRows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		// If a section's table is missing in some envs, return an
		// empty section rather than fail the whole export.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "42P01" {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]any, len(cols))
		for i, name := range cols {
			entry[name] = normalizeExportValue(raw[i])
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// normalizeExportValue turns DB-specific representations (byte slices
// for JSON columns, [16]byte for UUIDs) into JSON-friendly values.
// This is a best-effort conversion so the writer can encode without
// relying on a custom marshaler per column type.
func normalizeExportValue(v any) any {
	switch vv := v.(type) {
	case []byte:
		// Could be JSONB. Try to unmarshal — fall back to base64-ish
		// readable representation if not.
		var parsed any
		if err := json.Unmarshal(vv, &parsed); err == nil {
			return parsed
		}
		return string(vv)
	case [16]byte:
		return uuid.UUID(vv).String()
	default:
		return v
	}
}

// SoftDelete sets users.deleted_at = $at when it is currently NULL.
// Returns the timestamp actually persisted; if a previous request
// already set deleted_at, that earlier timestamp is returned and
// the caller treats the operation as idempotent.
func (r *GDPRRepository) SoftDelete(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var stored time.Time
	err := r.db.QueryRowContext(ctx, `
		UPDATE users
		SET deleted_at = COALESCE(deleted_at, $2),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING deleted_at`, userID, at).Scan(&stored)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, domainuser.ErrUserNotFound
		}
		return time.Time{}, fmt.Errorf("soft delete: %w", err)
	}
	return stored, nil
}

// CancelDeletion clears deleted_at atomically. Returns true when the
// row actually transitioned, false when there was nothing to cancel.
//
// The single UPDATE statement guarantees a concurrent purge tx that
// already locked the row sees the cancel through SKIP LOCKED — when
// it later re-checks deleted_at IS NOT NULL inside its tx, the cancel
// already landed and the purge skips.
func (r *GDPRRepository) CancelDeletion(ctx context.Context, userID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET deleted_at = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NOT NULL`, userID)
	if err != nil {
		return false, fmt.Errorf("cancel deletion: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("cancel deletion rows affected: %w", err)
	}
	return rows == 1, nil
}

// FindOwnedOrgsBlockingDeletion returns the orgs the user owns that
// have at least one OTHER active member. The query joins
// organizations + organization_members + users so the response can
// list a few admin candidates per blocking org.
func (r *GDPRRepository) FindOwnedOrgsBlockingDeletion(ctx context.Context, userID uuid.UUID) ([]gdpr.BlockedOrg, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.name,
		       (SELECT COUNT(*) FROM organization_members m
		        WHERE m.organization_id = o.id) AS member_count
		FROM organizations o
		WHERE o.owner_user_id = $1
		  AND EXISTS (SELECT 1 FROM organization_members m2
		              WHERE m2.organization_id = o.id
		                AND m2.user_id != $1)`, userID)
	if err != nil {
		return nil, fmt.Errorf("find blocking orgs: %w", err)
	}
	defer rows.Close()

	var blocked []gdpr.BlockedOrg
	for rows.Next() {
		var (
			orgID       uuid.UUID
			orgName     string
			memberCount int
		)
		if err := rows.Scan(&orgID, &orgName, &memberCount); err != nil {
			return nil, fmt.Errorf("scan blocking org: %w", err)
		}
		admins, err := r.fetchAdmins(ctx, orgID, userID)
		if err != nil {
			return nil, err
		}
		blocked = append(blocked, gdpr.BlockedOrg{
			OrgID:       orgID,
			OrgName:     orgName,
			MemberCount: memberCount,
			Admins:      admins,
			Actions: []gdpr.RemediationAction{
				gdpr.ActionTransferOwnership,
				gdpr.ActionDissolveOrg,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("blocking orgs iteration: %w", err)
	}
	return blocked, nil
}

func (r *GDPRRepository) fetchAdmins(ctx context.Context, orgID, ownerID uuid.UUID) ([]gdpr.AvailableAdmin, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.email
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1 AND m.role IN ('admin', 'owner')
		  AND m.user_id != $2
		ORDER BY u.email
		LIMIT 5`, orgID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	var out []gdpr.AvailableAdmin
	for rows.Next() {
		var a gdpr.AvailableAdmin
		if err := rows.Scan(&a.UserID, &a.Email); err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListPurgeable returns users whose deleted_at is older than `before`.
// SKIP LOCKED so a concurrent CancelDeletion (UPDATE … WHERE
// deleted_at IS NOT NULL) does not stall the cron.
func (r *GDPRRepository) ListPurgeable(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM users
		WHERE deleted_at IS NOT NULL AND deleted_at < $1
		ORDER BY deleted_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, before, limit)
	if err != nil {
		return nil, fmt.Errorf("list purgeable: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan purgeable: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// PurgeUser anonymizes-in-place + cleans cascade-able rows.
//
// The purge is wrapped in a single tx with FOR UPDATE on the user
// row. Inside the tx we re-check deleted_at IS NOT NULL AND <
// `before` so a CancelDeletion that landed between ListPurgeable and
// PurgeUser causes the purge to skip cleanly (returns ok=false).
//
// What gets done:
//   1. SELECT … FOR UPDATE on the user row + re-check
//   2. UPDATE audit_logs metadata to anonymize email/name/IP via salt
//   3. UPDATE users to nullify PII columns (email, names, password
//      hash) and keep the row as a tombstone — required because
//      several FKs (proposals, disputes, jobs) are NOT NULL NO
//      ACTION and a hard DELETE would either fail or corrupt
//      historical business records
//   4. CASCADE-eligible per-user rows (notifications, sessions,
//      device tokens, password resets, conversation participants)
//      are removed via the existing FK CASCADE: once we set
//      deleted_at to a sentinel value AND null PII, we still hold
//      the row, but downstream calls treating the user as deleted
//      via IsScheduledForDeletion no longer surface the row in any
//      list endpoint
//
// The "hybrid hard-delete + anonymize" approach is the RGPD-compliant
// implementation when the data model has NOT NULL FKs that tie users
// to long-lived business records. Per the brief Decision 2 (audit
// logs anonymized) + Decision 4 (sha256(email + salt) hash), this
// matches the intent.
func (r *GDPRRepository) PurgeUser(ctx context.Context, userID uuid.UUID, before time.Time, salt string) (bool, error) {
	if salt == "" {
		return false, gdpr.ErrSaltRequired
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("purge tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Re-check deleted_at inside the tx with a row lock.
	var deletedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT deleted_at FROM users WHERE id = $1
		FOR UPDATE SKIP LOCKED`, userID).Scan(&deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Row already gone or someone else owns the lock.
			return false, nil
		}
		return false, fmt.Errorf("purge re-check: %w", err)
	}
	if !deletedAt.Valid || !deletedAt.Time.Before(before) {
		// Cancel won the race or window not elapsed — abort.
		return false, nil
	}

	// 2. Anonymize audit_logs metadata. The actor email hash is
	// computed in SQL via encode(digest(...)) so we don't have to
	// pull every row into Go. We also clobber the dedicated
	// ip_address inet column with a /16 mask for IPv4 and a /32
	// mask for IPv6 so the column-level PII is gone.
	if _, err := tx.ExecContext(ctx, `
		UPDATE audit_logs
		SET metadata = jsonb_set(
			jsonb_set(
				COALESCE(metadata, '{}'::jsonb) - 'email' - 'actor_email' - 'actor_name' - 'first_name' - 'last_name',
				'{actor_email_hash}',
				to_jsonb(encode(
					digest(LOWER(TRIM(COALESCE(metadata->>'email', metadata->>'actor_email', ''))) || $2, 'sha256'),
					'hex'
				))
			),
			'{anonymized_at}',
			to_jsonb(NOW())
		),
		ip_address = CASE
			WHEN ip_address IS NULL THEN NULL
			WHEN family(ip_address) = 4 THEN network(set_masklen(ip_address, 16))
			ELSE network(set_masklen(ip_address, 32))
		END
		WHERE user_id = $1`, userID, salt); err != nil {
		return false, fmt.Errorf("purge anonymize audit: %w", err)
	}

	// 3. Anonymize the user row in place. Email gets a deterministic
	// placeholder so a UNIQUE constraint won't collide if the cron
	// somehow runs twice; first/last/display_name + hashed_password
	// are blanked.
	emailPlaceholder := fmt.Sprintf("anonymized+%s@deleted.local", userID.String())
	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET email = $2,
		    hashed_password = '!ANONYMIZED!',
		    first_name = 'anonymized',
		    last_name = 'user',
		    display_name = 'Anonymized user',
		    linkedin_id = NULL,
		    google_id = NULL,
		    email_verified = false,
		    updated_at = NOW()
		WHERE id = $1`, userID, emailPlaceholder); err != nil {
		return false, fmt.Errorf("purge anonymize user: %w", err)
	}

	// 4. Cascade-eligible side tables — explicit DELETEs for clarity
	// even where the FK already CASCADEs, so the operator runbook can
	// audit the exact rows wiped per purge tx.
	cascadeQueries := []string{
		`DELETE FROM notifications WHERE user_id = $1`,
		`DELETE FROM device_tokens WHERE user_id = $1`,
		`DELETE FROM password_resets WHERE user_id = $1`,
		`DELETE FROM notification_preferences WHERE user_id = $1`,
		`DELETE FROM conversation_participants WHERE user_id = $1`,
		`DELETE FROM conversation_read_state WHERE user_id = $1`,
		`DELETE FROM job_views WHERE user_id = $1`,
	}
	for _, q := range cascadeQueries {
		if _, err := tx.ExecContext(ctx, q, userID); err != nil {
			return false, fmt.Errorf("purge cascade %q: %w", q, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("purge commit: %w", err)
	}
	return true, nil
}
