package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/port/repository"
)

// MediaRepository implements repository.MediaRepository using PostgreSQL.
type MediaRepository struct {
	db *sql.DB
}

// NewMediaRepository creates a new PostgreSQL-backed media repository.
func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(ctx context.Context, m *media.Media) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	labelsJSON, err := marshalLabels(m.ModerationLabels)
	if err != nil {
		return fmt.Errorf("marshal moderation labels: %w", err)
	}

	_, err = r.db.ExecContext(ctx, queryInsertMedia,
		m.ID, m.UploaderID, m.FileURL, m.FileName, m.FileType, m.FileSize,
		string(m.Context), m.ContextID, string(m.ModerationStatus), labelsJSON,
		m.ModerationScore, m.RekognitionJobID, m.ReviewedAt, m.ReviewedBy, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert media: %w", err)
	}
	return nil
}

func (r *MediaRepository) GetByJobID(ctx context.Context, jobID string) (*media.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	m, err := scanMedia(r.db.QueryRowContext(ctx, queryGetMediaByJobID, jobID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, media.ErrMediaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get media by job id: %w", err)
	}
	return m, nil
}

func (r *MediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*media.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	m, err := scanMedia(r.db.QueryRowContext(ctx, queryGetMediaByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, media.ErrMediaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get media by id: %w", err)
	}
	return m, nil
}

func (r *MediaRepository) Update(ctx context.Context, m *media.Media) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	labelsJSON, err := marshalLabels(m.ModerationLabels)
	if err != nil {
		return fmt.Errorf("marshal moderation labels: %w", err)
	}

	_, err = r.db.ExecContext(ctx, queryUpdateMedia,
		m.ID, string(m.ModerationStatus), labelsJSON,
		m.ModerationScore, m.RekognitionJobID, m.ReviewedAt, m.ReviewedBy, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update media: %w", err)
	}
	return nil
}

func (r *MediaRepository) GetAdminByID(ctx context.Context, id uuid.UUID) (*repository.AdminMediaItem, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT m.id, m.uploader_id, m.file_url, m.file_name, m.file_type, m.file_size,
		m.context, m.context_id, m.moderation_status, m.moderation_labels,
		m.moderation_score, m.reviewed_at, m.reviewed_by, m.created_at, m.updated_at,
		u.display_name, u.email, u.role
		FROM media m
		JOIN users u ON u.id = m.uploader_id
		WHERE m.id = $1`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("get admin media: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, media.ErrMediaNotFound
	}

	item, err := scanAdminMediaRow(rows)
	if err != nil {
		return nil, fmt.Errorf("scan admin media: %w", err)
	}
	return &item, nil
}

func (r *MediaRepository) CountRejectedByUploader(ctx context.Context, uploaderID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM media WHERE uploader_id = $1 AND moderation_status = 'rejected'",
		uploaderID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count rejected media: %w", err)
	}
	return count, nil
}

// ClearSource removes the URL reference from the source table when a media is auto-rejected.
func (r *MediaRepository) ClearSource(ctx context.Context, mediaContext string, contextID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var query string
	switch mediaContext {
	case "profile_photo":
		// TODO(phase5-bug): profiles.user_id was dropped in m.067. This branch
		// is dead until ClearSource is updated to receive an organization_id
		// for profile contexts. Flagged in auditqualite.md.
		query = "UPDATE profiles SET photo_url = '' WHERE user_id = $1" // authorship-by-user-ok
	case "profile_video":
		query = "UPDATE profiles SET presentation_video_url = '' WHERE user_id = $1" // authorship-by-user-ok
	case "referrer_video":
		query = "UPDATE profiles SET referrer_video_url = '' WHERE user_id = $1" // authorship-by-user-ok
	case "review_video":
		query = "UPDATE reviews SET video_url = '' WHERE id = $1"
	case "job_video":
		query = "UPDATE jobs SET video_url = '' WHERE id = $1"
	default:
		return nil // message_attachment, identity_document — no source URL to clear
	}

	_, err := r.db.ExecContext(ctx, query, contextID)
	if err != nil {
		return fmt.Errorf("clear source %s: %w", mediaContext, err)
	}
	return nil
}

func (r *MediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryDeleteMedia, id)
	if err != nil {
		return fmt.Errorf("delete media: %w", err)
	}
	return nil
}

func (r *MediaRepository) ListAdmin(
	ctx context.Context,
	filters repository.AdminMediaFilters,
) ([]repository.AdminMediaItem, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query, args := buildMediaListQuery(filters, false)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin media: %w", err)
	}
	defer rows.Close()

	var items []repository.AdminMediaItem
	for rows.Next() {
		item, err := scanAdminMediaRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan admin media row: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []repository.AdminMediaItem{}
	}
	return items, rows.Err()
}

func (r *MediaRepository) CountAdmin(
	ctx context.Context,
	filters repository.AdminMediaFilters,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query, args := buildMediaListQuery(filters, true)

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count admin media: %w", err)
	}
	return count, nil
}

// buildMediaListQuery builds the SQL query with dynamic WHERE clauses.
func buildMediaListQuery(
	filters repository.AdminMediaFilters,
	countOnly bool,
) (string, []any) {
	var sb strings.Builder
	var args []any
	argIdx := 1

	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM media m JOIN users u ON u.id = m.uploader_id WHERE 1=1")
	} else {
		sb.WriteString(`SELECT m.id, m.uploader_id, m.file_url, m.file_name, m.file_type, m.file_size,
			m.context, m.context_id, m.moderation_status, m.moderation_labels,
			m.moderation_score, m.reviewed_at, m.reviewed_by, m.created_at, m.updated_at,
			u.display_name, u.email, u.role
		FROM media m
		JOIN users u ON u.id = m.uploader_id
		WHERE 1=1`)
	}

	if filters.Status != "" {
		sb.WriteString(fmt.Sprintf(" AND m.moderation_status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Type != "" {
		sb.WriteString(fmt.Sprintf(" AND m.file_type LIKE $%d", argIdx))
		args = append(args, filters.Type+"%")
		argIdx++
	}
	if filters.Context != "" {
		sb.WriteString(fmt.Sprintf(" AND m.context = $%d", argIdx))
		args = append(args, filters.Context)
		argIdx++
	}
	if filters.Search != "" {
		sb.WriteString(fmt.Sprintf(" AND (m.file_name ILIKE $%d OR u.display_name ILIKE $%d OR u.email ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filters.Search+"%")
		argIdx++
	}

	if !countOnly {
		switch filters.Sort {
		case "oldest":
			sb.WriteString(" ORDER BY m.created_at ASC, m.id ASC")
		case "score":
			sb.WriteString(" ORDER BY m.moderation_score DESC, m.created_at DESC")
		default:
			sb.WriteString(" ORDER BY m.created_at DESC, m.id DESC")
		}
		limit := filters.Limit
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		offset := 0
		if filters.Page > 1 {
			offset = (filters.Page - 1) * limit
		}
		sb.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1))
		args = append(args, limit, offset)
	}

	return sb.String(), args
}

func scanMedia(row *sql.Row) (*media.Media, error) {
	var m media.Media
	var ctxStr, statusStr string
	var labelsJSON []byte
	var contextID *uuid.UUID
	var jobID *string

	err := row.Scan(
		&m.ID, &m.UploaderID, &m.FileURL, &m.FileName, &m.FileType, &m.FileSize,
		&ctxStr, &contextID, &statusStr, &labelsJSON,
		&m.ModerationScore, &jobID, &m.ReviewedAt, &m.ReviewedBy, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	m.Context = media.Context(ctxStr)
	m.ModerationStatus = media.ModerationStatus(statusStr)
	m.ContextID = contextID
	m.RekognitionJobID = jobID
	m.ModerationLabels = unmarshalLabels(labelsJSON)
	return &m, nil
}

func scanAdminMediaRow(rows *sql.Rows) (repository.AdminMediaItem, error) {
	var item repository.AdminMediaItem
	var ctxStr, statusStr string
	var labelsJSON []byte
	var contextID *uuid.UUID

	err := rows.Scan(
		&item.ID, &item.UploaderID, &item.FileURL, &item.FileName,
		&item.FileType, &item.FileSize, &ctxStr, &contextID,
		&statusStr, &labelsJSON, &item.ModerationScore,
		&item.ReviewedAt, &item.ReviewedBy, &item.CreatedAt, &item.UpdatedAt,
		&item.UploaderDisplayName, &item.UploaderEmail, &item.UploaderRole,
	)
	if err != nil {
		return item, err
	}

	item.Context = media.Context(ctxStr)
	item.ModerationStatus = media.ModerationStatus(statusStr)
	item.ContextID = contextID
	item.ModerationLabels = unmarshalLabels(labelsJSON)
	return item, nil
}

func marshalLabels(labels []media.ModerationLabel) ([]byte, error) {
	if labels == nil {
		return []byte("null"), nil
	}
	return json.Marshal(labels)
}

func unmarshalLabels(data []byte) []media.ModerationLabel {
	if len(data) == 0 {
		return nil
	}
	var labels []media.ModerationLabel
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil
	}
	return labels
}
