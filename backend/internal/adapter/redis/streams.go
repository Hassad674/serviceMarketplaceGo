package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// isBusyGroupErr reports whether err is Redis's benign "consumer group
// already exists" response to XGROUP CREATE. Redis has used several
// wordings across versions ("already used", "already exists"), so we
// match on the BUSYGROUP prefix rather than the exact sentence.
func isBusyGroupErr(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "BUSYGROUP")
}

const (
	streamKey     = "messaging:events"
	consumerGroup = "messaging-consumers"
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

func (b *StreamBroadcaster) BroadcastNotification(ctx context.Context, userID uuid.UUID, payload []byte) error {
	return b.publish(ctx, "notification", []uuid.UUID{userID}, payload)
}

func (b *StreamBroadcaster) BroadcastMessageEdited(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "message_edited", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastMessageDeleted(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "message_deleted", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastCallEvent(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	return b.publish(ctx, "call_event", recipientIDs, payload)
}

func (b *StreamBroadcaster) BroadcastAdminNotification(ctx context.Context, adminIDs []uuid.UUID) error {
	payload, _ := json.Marshal(map[string]string{"event": "counters_updated"})
	return b.publish(ctx, "admin_notification_update", adminIDs, payload)
}

func (b *StreamBroadcaster) BroadcastAccountSuspended(ctx context.Context, userID uuid.UUID, reason string) error {
	payload, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		return fmt.Errorf("marshal account_suspended payload: %w", err)
	}
	return b.publish(ctx, "account_suspended", []uuid.UUID{userID}, payload)
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

// EnsureConsumerGroup creates the consumer group if it does not already exist.
// Uses MKSTREAM so the stream is created automatically if missing.
//
// On every backend restart this call will predictably return BUSYGROUP —
// the group was created on the first boot and Redis keeps it until the
// stream is deleted. That path is benign and is logged at INFO. A genuine
// failure (connectivity, permissions) is still logged at ERROR.
func (b *StreamBroadcaster) EnsureConsumerGroup(ctx context.Context) {
	err := b.client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err()
	if err == nil {
		slog.Info("redis consumer group created",
			"stream", streamKey, "group", consumerGroup)
		return
	}
	if isBusyGroupErr(err) {
		slog.Info("redis consumer group already exists",
			"stream", streamKey, "group", consumerGroup)
		return
	}
	slog.Error("failed to create consumer group",
		"stream", streamKey, "group", consumerGroup, "error", err)
}

// ackMessage acknowledges a message so it is removed from the pending list.
func (b *StreamBroadcaster) ackMessage(ctx context.Context, messageID string) {
	if err := b.client.XAck(ctx, streamKey, consumerGroup, messageID).Err(); err != nil {
		slog.Error("failed to ack stream message", "error", err, "message_id", messageID)
	}
}

type StreamHandler func(event StreamEvent)

// Subscribe reads from the Redis stream using consumer groups with a fixed
// consumer name (sourceID). This supports horizontal scaling: each instance
// gets unique messages, and the fixed name prevents dead consumer accumulation
// across redeploys.
func (b *StreamBroadcaster) Subscribe(ctx context.Context, handler StreamHandler) {
	b.EnsureConsumerGroup(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := b.client.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: b.sourceID,
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
