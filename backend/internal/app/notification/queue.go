package notification

import (
	"context"
	"encoding/json"
)

// DeliveryJob represents a notification delivery task to be processed by the worker.
type DeliveryJob struct {
	NotificationID string          `json:"notification_id"`
	UserID         string          `json:"user_id"`
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	Body           string          `json:"body"`
	Data           json.RawMessage `json:"data"`
	Attempt        int             `json:"attempt"`
	CreatedAt      string          `json:"created_at"`
}

// NotificationQueue abstracts the job queue for testability.
type NotificationQueue interface {
	Enqueue(ctx context.Context, job DeliveryJob) error
	Dequeue(ctx context.Context) (*DeliveryJob, string, error) // returns job + stream message ID
	Ack(ctx context.Context, messageID string) error
}
