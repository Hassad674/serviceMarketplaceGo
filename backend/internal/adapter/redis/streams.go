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

const (
	streamKey     = "messaging:events"
	consumerGroup = "messaging-group"
)

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

func (b *StreamBroadcaster) publish(ctx context.Context, eventType string, recipientIDs []uuid.UUID, payload []byte) error {
	ids, _ := json.Marshal(recipientIDs)

	err := b.client.XAdd(ctx, &goredis.XAddArgs{
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

func (b *StreamBroadcaster) EnsureConsumerGroup(ctx context.Context) error {
	err := b.client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

type StreamHandler func(event StreamEvent)

func (b *StreamBroadcaster) Subscribe(ctx context.Context, handler StreamHandler) {
	consumerName := b.sourceID

	if err := b.EnsureConsumerGroup(ctx); err != nil {
		slog.Error("failed to ensure consumer group", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := b.client.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{streamKey, ">"},
			Count:    10,
			Block:    5 * time.Second,
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
				b.ackMessage(ctx, msg.ID)
			}
		}
	}
}

func (b *StreamBroadcaster) ackMessage(ctx context.Context, messageID string) {
	if err := b.client.XAck(ctx, streamKey, consumerGroup, messageID).Err(); err != nil {
		slog.Error("failed to ack stream message", "error", err, "message_id", messageID)
	}
}
