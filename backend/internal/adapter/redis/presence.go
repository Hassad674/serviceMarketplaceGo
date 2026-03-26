package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type PresenceService struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewPresenceService(client *goredis.Client, ttl time.Duration) *PresenceService {
	return &PresenceService{client: client, ttl: ttl}
}

func presenceKey(userID uuid.UUID) string {
	return "presence:" + userID.String()
}

func (s *PresenceService) SetOnline(ctx context.Context, userID uuid.UUID) error {
	if err := s.client.SetEx(ctx, presenceKey(userID), "1", s.ttl).Err(); err != nil {
		return fmt.Errorf("set online: %w", err)
	}
	return nil
}

func (s *PresenceService) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	result, err := s.client.Exists(ctx, presenceKey(userID)).Result()
	if err != nil {
		return false, fmt.Errorf("check online: %w", err)
	}
	return result > 0, nil
}

func (s *PresenceService) BulkIsOnline(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(userIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*goredis.IntCmd, len(userIDs))

	for i, id := range userIDs {
		cmds[i] = pipe.Exists(ctx, presenceKey(id))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("bulk check online: %w", err)
	}

	result := make(map[uuid.UUID]bool, len(userIDs))
	for i, id := range userIDs {
		val, err := cmds[i].Result()
		if err != nil {
			result[id] = false
			continue
		}
		result[id] = val > 0
	}

	return result, nil
}
