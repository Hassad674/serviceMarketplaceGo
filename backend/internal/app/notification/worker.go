package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

const maxRetries = 3

// WorkerDeps groups dependencies for the notification delivery worker.
type WorkerDeps struct {
	Queue    NotificationQueue
	Presence service.PresenceService
	Push     service.PushService
	Email    service.EmailService
	Users    repository.UserRepository
	Notifs   repository.NotificationRepository
}

// Worker processes notification delivery jobs from the queue.
type Worker struct {
	queue    NotificationQueue
	presence service.PresenceService
	push     service.PushService
	email    service.EmailService
	users    repository.UserRepository
	notifs   repository.NotificationRepository
}

// NewWorker creates a new notification delivery worker.
func NewWorker(deps WorkerDeps) *Worker {
	return &Worker{
		queue:    deps.Queue,
		presence: deps.Presence,
		push:     deps.Push,
		email:    deps.Email,
		users:    deps.Users,
		notifs:   deps.Notifs,
	}
}

// Run starts the worker loop. It blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("notification worker started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("notification worker stopped")
			return
		default:
		}

		job, msgID, err := w.queue.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("notification worker: dequeue error", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if job == nil {
			continue // timeout, no job available
		}

		if err := w.processJob(ctx, *job); err != nil {
			slog.Error("notification worker: process error",
				"notification_id", job.NotificationID,
				"attempt", job.Attempt,
				"error", err,
			)
		}

		if err := w.queue.Ack(ctx, msgID); err != nil {
			slog.Error("notification worker: ack error", "message_id", msgID, "error", err)
		}
	}
}

// processJob handles a single delivery job: push + email with retry.
func (w *Worker) processJob(ctx context.Context, job DeliveryJob) error {
	userID, err := uuid.Parse(job.UserID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}

	// Load user preferences
	prefs := w.getPrefs(ctx, userID, notif.NotificationType(job.Type))

	var pushErr, emailErr error

	// Push delivery
	if prefs.Push && w.push != nil {
		pushErr = w.deliverPush(ctx, job, userID)
	}

	// Email delivery (never for new_message)
	if prefs.Email && job.Type != string(notif.TypeNewMessage) && w.email != nil {
		emailErr = w.deliverEmail(ctx, job, userID)
	}

	// Handle failures with retry
	if pushErr != nil || emailErr != nil {
		if job.Attempt >= maxRetries-1 {
			slog.Error("notification dead letter: max retries exceeded",
				"notification_id", job.NotificationID,
				"user_id", job.UserID,
				"push_error", pushErr,
				"email_error", emailErr,
			)
			return nil // don't re-enqueue, it's dead
		}

		// Exponential backoff before retry
		delay := time.Duration(math.Pow(2, float64(job.Attempt))) * time.Second
		time.Sleep(delay)

		job.Attempt++
		if enqErr := w.queue.Enqueue(ctx, job); enqErr != nil {
			slog.Error("notification worker: re-enqueue failed",
				"notification_id", job.NotificationID,
				"error", enqErr,
			)
		}
		return nil
	}

	slog.Debug("notification worker: delivered",
		"notification_id", job.NotificationID,
		"user_id", job.UserID,
		"type", job.Type,
		"attempt", job.Attempt,
	)

	return nil
}

func (w *Worker) deliverPush(ctx context.Context, job DeliveryJob, userID uuid.UUID) error {
	// Skip if user is online (they already got the WS notification)
	online, err := w.presence.IsOnline(ctx, userID)
	if err != nil {
		slog.Warn("notification worker: presence check failed", "error", err)
		// Continue anyway — better to send a duplicate push than miss it
	}
	if online {
		return nil
	}

	tokens, err := w.notifs.ListDeviceTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return nil // no tokens, nothing to do
	}

	tokenStrings := make([]string, 0, len(tokens))
	for _, dt := range tokens {
		tokenStrings = append(tokenStrings, dt.Token)
	}

	data := make(map[string]string)
	if job.Data != nil {
		var m map[string]any
		if json.Unmarshal(job.Data, &m) == nil {
			for k, v := range m {
				data[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	data["notification_type"] = job.Type
	data["notification_id"] = job.NotificationID

	return w.push.SendPush(ctx, tokenStrings, job.Title, job.Body, data)
}

func (w *Worker) deliverEmail(ctx context.Context, job DeliveryJob, userID uuid.UUID) error {
	u, err := w.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("load user: %w", err)
	}

	subject := job.Title + " — Marketplace Service"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #F43F5E;">%s</h2>
			<p>%s</p>
			<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
			<p style="color: #64748B; font-size: 12px;">Marketplace Service</p>
		</div>
	`, job.Title, job.Body)

	return w.email.SendNotification(ctx, u.Email, subject, html)
}

func (w *Worker) getPrefs(ctx context.Context, userID uuid.UUID, nType notif.NotificationType) *notif.Preferences {
	saved, err := w.notifs.GetPreferences(ctx, userID)
	if err != nil {
		return notif.DefaultPreferences(userID, nType)
	}
	for _, p := range saved {
		if p.NotificationType == nType {
			return p
		}
	}
	return notif.DefaultPreferences(userID, nType)
}
