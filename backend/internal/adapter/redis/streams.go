package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const streamKey = "messaging:events"

type StreamEvent struct {
	Type         string `json:"type"`
	RecipientIDs string `json:"recipient_ids"`
	Payload      string `json:"payload"`
	SourceID     string `json:"source_id"`
}

type StreamBroadcaster struct {
	client   *goredis.Client
	sourceID string
}

func NewStreamBroadcaster(client *goredis.Client, sourceID string) *StreamBroadcaster {
	return &StreamBroadcaster{client: client, sourceID: sourceID}
}

func (b *StreamBroadcaster) BroadcastNewMessage(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "new_message", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastTyping(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "typing", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastStatusUpdate(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "status_update", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastUnreadCount(ctx context.Context, userID uuid.UUID, count int) error {
	payload, _ := json.Marshal(map[string]any{"count": count})
	return b.publish(ctx, "unread_count", []uuid.UUID{userID}, payload)
}

func (b *StreamBroadcaster) BroadcastMessageRead(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "status_update", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastPresence(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "presence", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastNotification(ctx context.Context, userID uuid.UUID, payload []byte) error {
	return b.publish(ctx, "notification", []uuid.UUID{userID}, payload)
}

func (b *StreamBroadcaster) BroadcastCallEvent(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "call_event", recipientIDs, payload)
}

func (b *StreamBroadcaster) publish(ctx context.Context, eventType string, recipientIDs []uuid.UUID, payload []byte) error {
	ids, err := json.Marshal(recipientIDs)
	if err != nil {
		return fmt.Errorf("marshal recipient ids: %w", err)
	}

	err = b.client.XAdd(ctx, &goredis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"type":          eventType,
			"recipient_ids": string(ids),
			"payload":       string(payload),
			"source_id":     b.sourceID,
		},
		MaxLen: 10000,
		Approx: true,
	}).Err()
	if err != nil {
		return fmt.Errorf("publish stream event: %w", err)
	}

	return nil
}

type StreamHandler func(event StreamEvent)

func (b *StreamBroadcaster) Subscribe(ctx context.Context, handler StreamHandler) {
	// Use plain XREAD (not consumer groups). For a single backend instance,
	// consumer groups cause problems: each deploy creates a new consumer name
	// (random UUID), leaving dead consumers with undelivered pending messages.
	// XREAD with "$" reads only new messages, which is exactly what we need.
	lastID := "$"

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := b.client.XRead(ctx, &goredis.XReadArgs{
			Streams: []string{streamKey, lastID},
			Count:   10,
			Block:   5 * time.Second,
		}).Result()

		if err != nil {
			if err == goredis.Nil || ctx.Err() != nil {
				continue
			}
			slog.Error("stream read error", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				event := StreamEvent{
					Type:         msg.Values["type"].(string),
					RecipientIDs: msg.Values["recipient_ids"].(string),
					Payload:      msg.Values["payload"].(string),
					SourceID:     msg.Values["source_id"].(string),
				}

				handler(event)
				lastID = msg.ID
			}
		}
	}
}

