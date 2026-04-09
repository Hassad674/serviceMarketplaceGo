package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/portfolio"
)

// PortfolioRepository implements repository.PortfolioRepository using PostgreSQL.
type PortfolioRepository struct {
	db *sql.DB
}

// NewPortfolioRepository creates a new PostgreSQL-backed portfolio repository.
func NewPortfolioRepository(db *sql.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// Create inserts a portfolio item with all its media in a single transaction.
func (r *PortfolioRepository) Create(ctx context.Context, item *portfolio.PortfolioItem) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, queryInsertPortfolioItem,
		item.ID, item.UserID, item.Title, item.Description,
		item.LinkURL, item.Position, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert portfolio item: %w", err)
	}

	for _, m := range item.Media {
		_, err = tx.ExecContext(ctx, queryInsertPortfolioMedia,
			m.ID, m.PortfolioItemID, m.MediaURL, string(m.MediaType),
			m.ThumbnailURL, m.Position, m.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert portfolio media: %w", err)
		}
	}

	return tx.Commit()
}

// GetByID returns a single item with all its media.
func (r *PortfolioRepository) GetByID(ctx context.Context, id uuid.UUID) (*portfolio.PortfolioItem, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	item, err := scanPortfolioItem(r.db.QueryRowContext(ctx, queryGetPortfolioItemByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, portfolio.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get portfolio item: %w", err)
	}

	media, err := r.loadMediaForItem(ctx, id)
	if err != nil {
		return nil, err
	}
	item.Media = media

	return item, nil
}

// ListByUser returns items ordered by position ASC with cursor pagination.
func (r *PortfolioRepository) ListByUser(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*portfolio.PortfolioItem, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListPortfolioByUserFirst, userID, limit+1)
	} else {
		pos, cID, decErr := decodePositionCursor(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListPortfolioByUserWithCursor, userID, pos, cID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list portfolio items: %w", err)
	}
	defer rows.Close()

	var items []*portfolio.PortfolioItem
	for rows.Next() {
		item, scanErr := scanPortfolioItem(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("scan portfolio item: %w", scanErr)
		}
		items = append(items, item)
	}

	var nextCursor string
	if len(items) > limit {
		last := items[limit-1]
		nextCursor = encodePositionCursor(last.Position, last.ID)
		items = items[:limit]
	}

	// Batch-load media for all items (avoids N+1).
	if len(items) > 0 {
		if err := r.batchLoadMedia(ctx, items); err != nil {
			return nil, "", err
		}
	}

	return items, nextCursor, nil
}

// Update updates the item's mutable fields.
func (r *PortfolioRepository) Update(ctx context.Context, item *portfolio.PortfolioItem) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpdatePortfolioItem,
		item.ID, item.Title, item.Description, item.LinkURL, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update portfolio item: %w", err)
	}
	return nil
}

// Delete removes a portfolio item. CASCADE handles media deletion.
func (r *PortfolioRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryDeletePortfolioItem, id)
	if err != nil {
		return fmt.Errorf("delete portfolio item: %w", err)
	}
	return nil
}

// CountByUser returns the total number of portfolio items for a user.
func (r *PortfolioRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx, queryCountPortfolioByUser, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count portfolio items: %w", err)
	}
	return count, nil
}

// ReorderItems batch-updates positions in a transaction.
func (r *PortfolioRepository) ReorderItems(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for i, id := range itemIDs {
		_, err := tx.ExecContext(ctx, queryUpdatePortfolioPosition, id, i, userID)
		if err != nil {
			return fmt.Errorf("update position %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// ReplaceMedia deletes all existing media and inserts new ones in a transaction.
func (r *PortfolioRepository) ReplaceMedia(ctx context.Context, itemID uuid.UUID, media []*portfolio.PortfolioMedia) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, queryDeleteMediaByItemID, itemID)
	if err != nil {
		return fmt.Errorf("delete old media: %w", err)
	}

	for _, m := range media {
		_, err = tx.ExecContext(ctx, queryInsertPortfolioMedia,
			m.ID, m.PortfolioItemID, m.MediaURL, string(m.MediaType),
			m.ThumbnailURL, m.Position, m.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert portfolio media: %w", err)
		}
	}

	return tx.Commit()
}

// --- helpers ---

func (r *PortfolioRepository) loadMediaForItem(ctx context.Context, itemID uuid.UUID) ([]*portfolio.PortfolioMedia, error) {
	rows, err := r.db.QueryContext(ctx, queryListPortfolioMediaByItemID, itemID)
	if err != nil {
		return nil, fmt.Errorf("load media: %w", err)
	}
	defer rows.Close()

	var media []*portfolio.PortfolioMedia
	for rows.Next() {
		m, scanErr := scanPortfolioMedia(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan media: %w", scanErr)
		}
		media = append(media, m)
	}
	return media, nil
}

func (r *PortfolioRepository) batchLoadMedia(ctx context.Context, items []*portfolio.PortfolioItem) error {
	ids := make([]uuid.UUID, len(items))
	lookup := make(map[uuid.UUID]*portfolio.PortfolioItem, len(items))
	for i, item := range items {
		ids[i] = item.ID
		lookup[item.ID] = item
		item.Media = []*portfolio.PortfolioMedia{} // initialize empty
	}

	rows, err := r.db.QueryContext(ctx, queryListMediaByItemIDs, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("batch load media: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		m, scanErr := scanPortfolioMedia(rows)
		if scanErr != nil {
			return fmt.Errorf("scan media: %w", scanErr)
		}
		if item, ok := lookup[m.PortfolioItemID]; ok {
			item.Media = append(item.Media, m)
		}
	}
	return nil
}

// --- scanners ---

type portfolioScanner interface {
	Scan(dest ...any) error
}

func scanPortfolioItem(s portfolioScanner) (*portfolio.PortfolioItem, error) {
	var item portfolio.PortfolioItem
	err := s.Scan(
		&item.ID, &item.UserID, &item.Title, &item.Description,
		&item.LinkURL, &item.Position, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func scanPortfolioMedia(s portfolioScanner) (*portfolio.PortfolioMedia, error) {
	var m portfolio.PortfolioMedia
	var mediaType string
	err := s.Scan(
		&m.ID, &m.PortfolioItemID, &m.MediaURL, &mediaType,
		&m.ThumbnailURL, &m.Position, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.MediaType = portfolio.MediaType(mediaType)
	return &m, nil
}

// --- position-based cursor (different from time-based cursor in pkg/cursor) ---

type positionCursor struct {
	Position int       `json:"p"`
	ID       uuid.UUID `json:"id"`
}

func encodePositionCursor(position int, id uuid.UUID) string {
	c := positionCursor{Position: position, ID: id}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

func decodePositionCursor(encoded string) (int, uuid.UUID, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, uuid.Nil, fmt.Errorf("invalid base64: %w", err)
	}
	var c positionCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return 0, uuid.Nil, fmt.Errorf("invalid json: %w", err)
	}
	return c.Position, c.ID, nil
}
