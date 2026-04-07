package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
)

// AdminModerationRepository implements repository.AdminModerationRepository using PostgreSQL.
type AdminModerationRepository struct {
	db *sql.DB
}

// NewAdminModerationRepository returns a new AdminModerationRepository backed by the given DB.
func NewAdminModerationRepository(db *sql.DB) *AdminModerationRepository {
	return &AdminModerationRepository{db: db}
}

// List returns moderation items matching the given filters.
func (r *AdminModerationRepository) List(ctx context.Context, filters repository.ModerationFilters) ([]repository.ModerationItem, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query, args := buildModerationUnionQuery(filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("admin moderation list: %w", err)
	}
	defer rows.Close()

	var items []repository.ModerationItem
	for rows.Next() {
		item, scanErr := scanModerationItem(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("admin moderation list: scan: %w", scanErr)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin moderation list: rows: %w", err)
	}

	if items == nil {
		items = []repository.ModerationItem{}
	}
	return items, nil
}

// Count returns the total count of moderation items matching the given filters.
func (r *AdminModerationRepository) Count(ctx context.Context, filters repository.ModerationFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query, args := buildModerationCountQuery(filters)

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("admin moderation count: %w", err)
	}
	return count, nil
}

// PendingCount returns the total number of pending moderation items across all sources.
func (r *AdminModerationRepository) PendingCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := buildPendingCountQuery()

	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("admin moderation pending count: %w", err)
	}
	return count, nil
}

func scanModerationItem(rows *sql.Rows) (repository.ModerationItem, error) {
	var item repository.ModerationItem
	var source, contentType string
	var conversationID *uuid.UUID

	err := rows.Scan(
		&item.ID,
		&source,
		&contentType,
		&item.ContentID,
		&item.ContentPreview,
		&item.Status,
		&item.ModerationScore,
		&item.Reason,
		&item.UserInvolvedID,
		&item.UserInvolvedName,
		&item.UserInvolvedRole,
		&conversationID,
		&item.CreatedAt,
	)
	if err != nil {
		return item, err
	}

	item.Source = repository.ModerationSource(source)
	item.ContentType = contentType
	item.ConversationID = conversationID
	return item, nil
}
