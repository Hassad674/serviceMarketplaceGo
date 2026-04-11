package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	notifapp "marketplace-backend/internal/app/notification"
)

const (
	notifJobStream = "notification:jobs"
	notifJobGroup  = "notification-workers"
)

// NotificationJobQueue implements notifapp.NotificationQueue using Redis Streams.
type NotificationJobQueue struct {
	client     *goredis.Client
	consumerID string
}

// NewNotificationJobQueue creates a new queue backed by Redis Streams.
func NewNotificationJobQueue(client *goredis.Client, consumerID string) *NotificationJobQueue {
	return &NotificationJobQueue{client: client, consumerID: consumerID}
}

// EnsureGroup creates the consumer group if it doesn't exist.
//
// Returns nil when the group was freshly created AND when Redis reports
// BUSYGROUP (the group already exists from a previous boot). Both cases
// are logged at INFO so startup logs stay clean.
func (q *NotificationJobQueue) EnsureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, notifJobStream, notifJobGroup, "$").Err()
	if err == nil {
		slog.Info("redis consumer group created",
			"stream", notifJobStream, "group", notifJobGroup)
		return nil
	}
	if isBusyGroupErr(err) {
		slog.Info("redis consumer group already exists",
			"stream", notifJobStream, "group", notifJobGroup)
		return nil
	}
	return fmt.Errorf("create notification job group: %w", err)
}

// Enqueue adds a delivery job to the Redis Stream.
func (q *NotificationJobQueue) Enqueue(ctx context.Context, job notifapp.DeliveryJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	err = q.client.XAdd(ctx, &goredis.XAddArgs{
		Stream: notifJobStream,
		Values: map[string]interface{}{
			"job": string(data),
		},
		MaxLen: 50000,
		Approx: true,
	}).Err()
	if err != nil {
		return fmt.Errorf("enqueue notification job: %w", err)
	}

	return nil
}

// Dequeue reads the next job from the stream. Blocks up to 5 seconds.
// Returns nil job if no messages are available (timeout).
func (q *NotificationJobQueue) Dequeue(ctx context.Context) (*notifapp.DeliveryJob, string, error) {
	streams, err := q.client.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    notifJobGroup,
		Consumer: q.consumerID,
		Streams:  []string{notifJobStream, ">"},
		Count:    1,
		Block:    5 * time.Second,
	}).Result()

	if err != nil {
		if err == goredis.Nil {
			return nil, "", nil // timeout, no messages
		}
		return nil, "", fmt.Errorf("dequeue notification job: %w", err)
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			jobStr, ok := msg.Values["job"].(string)
			if !ok {
				slog.Warn("invalid job payload in stream", "message_id", msg.ID)
				_ = q.Ack(ctx, msg.ID)
				continue
			}

			var job notifapp.DeliveryJob
			if err := json.Unmarshal([]byte(jobStr), &job); err != nil {
				slog.Warn("unmarshal job failed", "message_id", msg.ID, "error", err)
				_ = q.Ack(ctx, msg.ID)
				continue
			}

			return &job, msg.ID, nil
		}
	}

	return nil, "", nil
}

// Ack acknowledges a processed message, removing it from the pending list.
func (q *NotificationJobQueue) Ack(ctx context.Context, messageID string) error {
	return q.client.XAck(ctx, notifJobStream, notifJobGroup, messageID).Err()
}
