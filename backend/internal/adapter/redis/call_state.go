package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/domain/call"
)

const callTTL = 1 * time.Hour

type CallStateService struct {
	client *goredis.Client
}

func NewCallStateService(client *goredis.Client) *CallStateService {
	return &CallStateService{client: client}
}

func callKey(id uuid.UUID) string    { return "call:" + id.String() }
func callUserKey(id uuid.UUID) string { return "call:user:" + id.String() }

func (s *CallStateService) SaveActiveCall(ctx context.Context, c *call.Call) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal call: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, callKey(c.ID), data, callTTL)
	pipe.Set(ctx, callUserKey(c.InitiatorID), c.ID.String(), callTTL)
	pipe.Set(ctx, callUserKey(c.RecipientID), c.ID.String(), callTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("save active call: %w", err)
	}
	return nil
}

func (s *CallStateService) GetActiveCall(ctx context.Context, callID uuid.UUID) (*call.Call, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	data, err := s.client.Get(ctx, callKey(callID)).Bytes()
	if err == goredis.Nil {
		return nil, call.ErrCallNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active call: %w", err)
	}

	var c call.Call
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal call: %w", err)
	}
	return &c, nil
}

func (s *CallStateService) GetActiveCallByUser(ctx context.Context, userID uuid.UUID) (*call.Call, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	callIDStr, err := s.client.Get(ctx, callUserKey(userID)).Result()
	if err == goredis.Nil {
		return nil, call.ErrCallNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get call by user: %w", err)
	}

	callID, err := uuid.Parse(callIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse call id: %w", err)
	}

	return s.GetActiveCall(ctx, callID)
}

func (s *CallStateService) RemoveActiveCall(ctx context.Context, callID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get the call first to find participant IDs
	c, err := s.GetActiveCall(ctx, callID)
	if err != nil {
		return nil // Already removed or not found
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, callKey(callID))
	pipe.Del(ctx, callUserKey(c.InitiatorID))
	pipe.Del(ctx, callUserKey(c.RecipientID))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("remove active call: %w", err)
	}
	return nil
}
