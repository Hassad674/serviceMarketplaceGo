package redis

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	portservice "marketplace-backend/internal/port/service"
)

const (
	adminNotifKeyPrefix = "admin:notif:"
	adminIDsCacheKey    = "admin:user_ids"
	adminIDsCacheTTL    = 5 * time.Minute
	adminNotifTTL       = 24 * time.Hour
)

// AdminNotifierService implements portservice.AdminNotifierService using Redis
// for per-admin notification counters.
type AdminNotifierService struct {
	client      *goredis.Client
	db          *sql.DB
	broadcaster portservice.MessageBroadcaster
}

// NewAdminNotifierService creates a new admin notifier backed by Redis.
func NewAdminNotifierService(
	client *goredis.Client,
	db *sql.DB,
	broadcaster portservice.MessageBroadcaster,
) *AdminNotifierService {
	return &AdminNotifierService{
		client:      client,
		db:          db,
		broadcaster: broadcaster,
	}
}

// IncrementAll increments a category counter for every admin user.
func (s *AdminNotifierService) IncrementAll(ctx context.Context, category string) error {
	adminIDs, err := s.getAdminIDs(ctx)
	if err != nil {
		return fmt.Errorf("admin notifier: get admin IDs: %w", err)
	}
	if len(adminIDs) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for _, id := range adminIDs {
		key := notifKey(id, category)
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, adminNotifTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("admin notifier: pipeline incr: %w", err)
	}

	s.broadcastUpdate(ctx, adminIDs)
	return nil
}

// GetAll returns all category counters for a specific admin user.
func (s *AdminNotifierService) GetAll(ctx context.Context, adminID uuid.UUID) (map[string]int64, error) {
	categories := portservice.AdminNotifCategories
	keys := make([]string, len(categories))
	for i, cat := range categories {
		keys[i] = notifKey(adminID, cat)
	}

	vals, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("admin notifier: mget: %w", err)
	}

	result := make(map[string]int64, len(categories))
	for i, cat := range categories {
		result[cat] = parseRedisInt(vals[i])
	}
	return result, nil
}

// Reset deletes a category counter for a specific admin user.
func (s *AdminNotifierService) Reset(ctx context.Context, adminID uuid.UUID, category string) error {
	key := notifKey(adminID, category)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("admin notifier: del %s: %w", key, err)
	}
	return nil
}

// getAdminIDs fetches admin user IDs, with a short Redis cache to avoid
// hitting the database on every increment.
func (s *AdminNotifierService) getAdminIDs(ctx context.Context) ([]uuid.UUID, error) {
	cached, err := s.client.Get(ctx, adminIDsCacheKey).Result()
	if err == nil && cached != "" {
		var ids []uuid.UUID
		if jsonErr := json.Unmarshal([]byte(cached), &ids); jsonErr == nil {
			return ids, nil
		}
	}

	ids, err := s.queryAdminIDs(ctx)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(ids)
	_ = s.client.Set(ctx, adminIDsCacheKey, string(data), adminIDsCacheTTL).Err()

	return ids, nil
}

func (s *AdminNotifierService) queryAdminIDs(ctx context.Context) ([]uuid.UUID, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(queryCtx,
		`SELECT id FROM users WHERE is_admin = true`)
	if err != nil {
		return nil, fmt.Errorf("admin notifier: query admin ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("admin notifier: scan admin id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *AdminNotifierService) broadcastUpdate(ctx context.Context, adminIDs []uuid.UUID) {
	if s.broadcaster == nil || len(adminIDs) == 0 {
		return
	}
	if err := s.broadcaster.BroadcastAdminNotification(ctx, adminIDs); err != nil {
		slog.Error("admin notifier: broadcast update", "error", err)
	}
}

func notifKey(adminID uuid.UUID, category string) string {
	return adminNotifKeyPrefix + adminID.String() + ":" + category
}

func parseRedisInt(val interface{}) int64 {
	if val == nil {
		return 0
	}
	s, ok := val.(string)
	if !ok {
		return 0
	}
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
