package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/gdpr"
)

// publicURLPrefix is the configured STORAGE_PUBLIC_URL with a trailing
// slash. Set via NewGDPRRepositoryWithStoragePrefix at construction.
// When empty (legacy constructor) the adapter treats every URL as
// already a key — which is wrong in production but keeps the
// pre-existing tests green until they migrate.
//
// We keep the field on the receiver rather than passing it around so
// the GDPRRepository's port surface stays the same and the storage
// concern is encapsulated.
func (r *GDPRRepository) storagePrefix() string {
	return r.storagePublicURL
}

// urlToKey converts a public URL stored in a TEXT column back to the
// underlying bucket key. Empty input returns empty (the caller skips).
// URLs that do not start with the configured prefix are returned
// unchanged: legacy data may already hold raw keys and we don't want
// to silently drop them from the audit.
func (r *GDPRRepository) urlToKey(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	prefix := r.storagePrefix()
	if prefix == "" {
		return url
	}
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix)
	}
	return url
}

// ListUserStorageKeys gathers every R2/MinIO object key tied to the
// user across the tables that hold uploaded media. Sources, in
// order, mirror the audit's "what gets deleted" matrix:
//
//	1. organizations.photo_url            (org avatar — for orgs the user OWNS)
//	2. profiles.photo_url + presentation_video_url + referrer_video_url
//	3. freelance_profiles.video_url
//	4. referrer_profiles.video_url
//	5. portfolio_media.media_url + thumbnail_url
//	   (joined to portfolio_items joined to organizations the user owns)
//	6. reviews.video_url       (reviews authored by the user)
//	7. identity_documents.file_key
//	8. jobs.video_url          (jobs created by the user)
//	9. job_applications.video_url
//	10. messages.metadata->>'url' (attachments authored by the user)
//
// The slice is deduped before return — duplicates are common because
// the org photo is mirrored on both organizations and profiles.
func (r *GDPRRepository) ListUserStorageKeys(ctx context.Context, userID uuid.UUID) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	keys := make(map[string]struct{}, 16)

	gather := func(rows *sql.Rows, err error, isFileKey bool) error {
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v sql.NullString
			if err := rows.Scan(&v); err != nil {
				return err
			}
			if !v.Valid {
				continue
			}
			raw := v.String
			if isFileKey {
				// identity_documents.file_key is already a key,
				// not a URL — never strip the public prefix.
				raw = strings.TrimSpace(raw)
				if raw == "" {
					continue
				}
				keys[raw] = struct{}{}
				continue
			}
			k := r.urlToKey(raw)
			if k == "" {
				continue
			}
			keys[k] = struct{}{}
		}
		return rows.Err()
	}

	// 1 + 5: organization-owned media (photo, portfolio).
	rows, err := r.db.QueryContext(ctx, `
		SELECT photo_url FROM organizations
		WHERE owner_user_id = $1 AND photo_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: org photos: %w", err)
	}

	// 2: legacy profiles row (photo + two video columns).
	rows, err = r.db.QueryContext(ctx, `
		SELECT photo_url               FROM profiles WHERE user_id = $1 AND photo_url <> ''
		UNION ALL
		SELECT presentation_video_url  FROM profiles WHERE user_id = $1 AND presentation_video_url <> ''
		UNION ALL
		SELECT referrer_video_url      FROM profiles WHERE user_id = $1 AND referrer_video_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: profiles: %w", err)
	}

	// 3: freelance video. The table is org-scoped per migration 097
	// but the joins below stay on owner_user_id so a user that owns
	// multiple personas (freelance + referrer) sees both purged.
	rows, err = r.db.QueryContext(ctx, `
		SELECT fp.video_url
		FROM freelance_profiles fp
		JOIN organizations o ON o.id = fp.organization_id
		WHERE o.owner_user_id = $1 AND fp.video_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: freelance video: %w", err)
	}

	// 4: referrer video.
	rows, err = r.db.QueryContext(ctx, `
		SELECT rp.video_url
		FROM referrer_profiles rp
		JOIN organizations o ON o.id = rp.organization_id
		WHERE o.owner_user_id = $1 AND rp.video_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: referrer video: %w", err)
	}

	// 5: portfolio media + thumbnails.
	rows, err = r.db.QueryContext(ctx, `
		SELECT pm.media_url
		FROM portfolio_media pm
		JOIN portfolio_items pi ON pi.id = pm.portfolio_item_id
		JOIN organizations   o  ON o.id = pi.organization_id
		WHERE o.owner_user_id = $1 AND pm.media_url <> ''
		UNION ALL
		SELECT pm.thumbnail_url
		FROM portfolio_media pm
		JOIN portfolio_items pi ON pi.id = pm.portfolio_item_id
		JOIN organizations   o  ON o.id = pi.organization_id
		WHERE o.owner_user_id = $1 AND pm.thumbnail_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: portfolio: %w", err)
	}

	// 6: review videos authored by the user.
	rows, err = r.db.QueryContext(ctx, `
		SELECT video_url FROM reviews
		WHERE reviewer_id = $1 AND video_url IS NOT NULL AND video_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: reviews: %w", err)
	}

	// 7: KYC identity documents — file_key is the bucket key, not a URL.
	rows, err = r.db.QueryContext(ctx, `
		SELECT file_key FROM identity_documents
		WHERE user_id = $1 AND file_key <> ''`, userID)
	if err := gather(rows, err, true); err != nil {
		return nil, fmt.Errorf("list keys: identity docs: %w", err)
	}

	// 8: jobs published by the user.
	rows, err = r.db.QueryContext(ctx, `
		SELECT video_url FROM jobs
		WHERE user_id = $1 AND video_url IS NOT NULL AND video_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: jobs: %w", err)
	}

	// 9: job applications authored by the user.
	rows, err = r.db.QueryContext(ctx, `
		SELECT video_url FROM job_applications
		WHERE applicant_user_id = $1 AND video_url IS NOT NULL AND video_url <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: job apps: %w", err)
	}

	// 10: message attachments. metadata is JSONB with an optional
	// "url" key per messages/service_test.go:1315. Empty url skipped.
	rows, err = r.db.QueryContext(ctx, `
		SELECT metadata->>'url'
		FROM messages
		WHERE sender_id = $1
		  AND metadata IS NOT NULL
		  AND metadata ? 'url'
		  AND COALESCE(metadata->>'url', '') <> ''`, userID)
	if err := gather(rows, err, false); err != nil {
		return nil, fmt.Errorf("list keys: messages: %w", err)
	}

	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	return out, nil
}

// RecordStoragePurgeAudit appends an audit row capturing the cron's
// best-effort R2 cleanup attempt. Called by the GDPR service AFTER
// BulkDelete returns and BEFORE PurgeUser anonymizes the user row.
//
// The row survives the user being purged because user_id FK is ON
// DELETE SET NULL (migration 144). Compliance evidence for art. 17
// erasure requests does not need the original user_id once the user
// itself is gone — the timestamp + keys_count + key arrays are what
// auditors look at.
func (r *GDPRRepository) RecordStoragePurgeAudit(ctx context.Context, m gdpr.StoragePurgeManifest) error {
	if err := m.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	purged := m.PurgedKeys
	if purged == nil {
		purged = []string{}
	}
	failed := m.FailedKeys
	if failed == nil {
		failed = []string{}
	}

	var orgID interface{}
	if m.OrganizationID != nil {
		orgID = m.OrganizationID.String()
	}

	purgedAt := m.PurgedAt
	if purgedAt.IsZero() {
		purgedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO storage_purge_audits (
			user_id, organization_id, keys_count,
			purged_keys, failed_keys, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		m.UserID,
		orgID,
		m.KeysCount(),
		pq.Array(purged),
		pq.Array(failed),
		purgedAt,
	)
	if err != nil {
		return fmt.Errorf("record storage purge audit: %w", err)
	}
	return nil
}
