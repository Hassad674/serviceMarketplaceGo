package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

const (
	// maxRetries bounds the per-job retry budget. After this many
	// failed deliveries the job is dropped to a dead-letter log line.
	maxRetries = 3

	// DefaultWorkerConcurrency is the parallelism used when the
	// caller does not pass an explicit value to Run. Five workers
	// keep p99 burst latency under a second on the typical load
	// (mail / push timeouts dominate). Override per-deployment via
	// WorkerConfig.Concurrency.
	DefaultWorkerConcurrency = 5
)

// WorkerDeps groups dependencies for the notification delivery worker.
type WorkerDeps struct {
	Queue    NotificationQueue
	Presence service.PresenceService
	Push     service.PushService
	Email    service.EmailService
	Users    repository.UserRepository
	Notifs   repository.NotificationRepository
}

// WorkerConfig groups optional runtime tunables for the worker.
// Zero-value is acceptable: every field falls back to the package
// default.
type WorkerConfig struct {
	// Concurrency is the number of parallel processors that share
	// the same Redis consumer group. Each runs its own
	// Dequeue→processJob→Ack loop inside a single Run() call.
	// Zero or negative falls back to DefaultWorkerConcurrency.
	Concurrency int
}

// Worker processes notification delivery jobs from the queue.
//
// BUG-16 fix: a single processing goroutine combined with a blocking
// `time.Sleep(backoff)` made the burst-after-inactivity p99 spike
// past 7 seconds (2s + 4s = 6s of pure sleep before the third
// attempt, plus the per-attempt timeouts). Two changes lift that
// ceiling:
//
//  1. Run() spawns N parallel processors so a slow delivery never
//     blocks queue drainage. Each processor owns the entire
//     Dequeue → processJob → Ack cycle and uses a distinct consumer
//     id so Redis-stream group semantics keep messages disjoint.
//
//  2. processJob no longer sleeps inside the retry path. When a
//     delivery fails, the job is re-enqueued AFTER a non-blocking
//     delay timer fires in a separate goroutine. The main processor
//     immediately ACKs the failed job and grabs the next one.
type Worker struct {
	queue    NotificationQueue
	presence service.PresenceService
	push     service.PushService
	email    service.EmailService
	users    repository.UserRepository
	notifs   repository.NotificationRepository

	cfg WorkerConfig

	// retryWG tracks the deferred re-enqueue goroutines so a
	// graceful shutdown (Run's ctx cancelled) waits for them to
	// finish. Without this they could lose retries on terminate.
	retryWG sync.WaitGroup
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

// WithConfig wires a non-default WorkerConfig and returns the same
// worker for fluent setup. Zero values fall back to package defaults
// so callers can override only the fields they care about.
func (w *Worker) WithConfig(cfg WorkerConfig) *Worker {
	w.cfg = cfg
	return w
}

// Run starts the worker pool and blocks until ctx is cancelled.
// Spawns Concurrency processors that share the queue's consumer
// group; ctx cancellation propagates to every processor and to the
// in-flight retry goroutines, so a graceful shutdown drains all
// pending re-enqueues before returning.
func (w *Worker) Run(ctx context.Context) {
	concurrency := w.cfg.Concurrency
	if concurrency <= 0 {
		concurrency = DefaultWorkerConcurrency
	}
	slog.Info("notification worker pool started", "concurrency", concurrency)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			w.runProcessor(ctx, workerIdx)
		}(i)
	}
	wg.Wait()
	w.retryWG.Wait()
	slog.Info("notification worker pool stopped")
}

// runProcessor is the per-goroutine drain loop. Stays small (under
// 50 lines / 3 nesting levels) so the parallel structure stays
// reviewable. Errors are logged and the loop continues — the only
// way out is ctx cancellation.
func (w *Worker) runProcessor(ctx context.Context, workerIdx int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, msgID, err := w.queue.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("notification worker: dequeue error",
				"worker", workerIdx, "error", err)
			// Don't busy-spin on persistent dequeue failures, but
			// also don't block the *other* processors.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if job == nil {
			continue // timeout, no job available
		}

		w.handleAndAck(ctx, *job, msgID, workerIdx)
	}
}

// handleAndAck runs processJob and acknowledges the message. Split
// out so runProcessor stays under the 50-line cap.
func (w *Worker) handleAndAck(ctx context.Context, job DeliveryJob, msgID string, workerIdx int) {
	if err := w.processJob(ctx, job); err != nil {
		slog.Error("notification worker: process error",
			"worker", workerIdx,
			"notification_id", job.NotificationID,
			"attempt", job.Attempt,
			"error", err,
		)
	}
	if err := w.queue.Ack(ctx, msgID); err != nil {
		slog.Error("notification worker: ack error",
			"worker", workerIdx, "message_id", msgID, "error", err)
	}
}

// processJob handles a single delivery job: push + email with retry.
// Stays under the 50-line cap by delegating retry/dead-letter
// decisions to scheduleRetryOrDeadLetter.
func (w *Worker) processJob(ctx context.Context, job DeliveryJob) error {
	userID, err := uuid.Parse(job.UserID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}

	prefs := w.getPrefs(ctx, userID, notif.NotificationType(job.Type))

	var pushErr, emailErr error
	if prefs.Push && w.push != nil {
		pushErr = w.deliverPush(ctx, job, userID)
	}
	if prefs.Email && job.Type != string(notif.TypeNewMessage) && w.email != nil {
		emailErr = w.maybeDeliverEmail(ctx, job, userID)
	}

	if pushErr == nil && emailErr == nil {
		slog.Debug("notification worker: delivered",
			"notification_id", job.NotificationID,
			"user_id", job.UserID,
			"type", job.Type,
			"attempt", job.Attempt,
		)
		return nil
	}

	w.scheduleRetryOrDeadLetter(ctx, job, pushErr, emailErr)
	return nil
}

// maybeDeliverEmail honours the global email-notifications opt-out
// before invoking deliverEmail. Pulled out so processJob's branching
// stays under the 3-nesting cap.
func (w *Worker) maybeDeliverEmail(ctx context.Context, job DeliveryJob, userID uuid.UUID) error {
	u, err := w.users.GetByID(ctx, userID)
	if err == nil && !u.EmailNotificationsEnabled {
		slog.Debug("notification worker: email skipped (globally disabled)",
			"notification_id", job.NotificationID, "user_id", job.UserID,
		)
		return nil
	}
	return w.deliverEmail(ctx, job, userID)
}

// scheduleRetryOrDeadLetter is the BUG-16 hot-path replacement for
// the old in-line `time.Sleep`. When the job has retries left, the
// re-enqueue is deferred to a one-shot goroutine that sleeps
// `delay` and then calls queue.Enqueue — the main processor returns
// immediately and grabs the next job. Goroutines are tracked via
// retryWG so a graceful shutdown waits for them.
func (w *Worker) scheduleRetryOrDeadLetter(ctx context.Context, job DeliveryJob, pushErr, emailErr error) {
	if job.Attempt >= maxRetries-1 {
		slog.Error("notification dead letter: max retries exceeded",
			"notification_id", job.NotificationID,
			"user_id", job.UserID,
			"push_error", pushErr,
			"email_error", emailErr,
		)
		return
	}

	delay := time.Duration(math.Pow(2, float64(job.Attempt))) * time.Second
	job.Attempt++

	w.retryWG.Add(1)
	go func() {
		defer w.retryWG.Done()
		w.deferredRequeue(ctx, job, delay)
	}()
}

// deferredRequeue runs in its own goroutine: waits the backoff
// duration, then re-enqueues the job. Shutdown-aware via ctx.
// Logged-on-failure: a re-enqueue error is non-recoverable for
// this attempt, but the original (already ACK'd) message is gone
// so the user won't get a duplicate.
func (w *Worker) deferredRequeue(ctx context.Context, job DeliveryJob, delay time.Duration) {
	select {
	case <-ctx.Done():
		// Shutdown: best-effort — try to enqueue with a fresh
		// background context so the retry survives a graceful
		// shutdown. Use the same timeout budget as the queue
		// itself (5s) to bound the wait.
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := w.queue.Enqueue(bg, job); err != nil {
			slog.Error("notification worker: shutdown re-enqueue failed",
				"notification_id", job.NotificationID, "error", err)
		}
		return
	case <-time.After(delay):
	}

	if err := w.queue.Enqueue(ctx, job); err != nil {
		slog.Error("notification worker: re-enqueue failed",
			"notification_id", job.NotificationID,
			"attempt", job.Attempt,
			"error", err,
		)
	}
}

func (w *Worker) deliverPush(ctx context.Context, job DeliveryJob, userID uuid.UUID) error {
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

	data := buildPushData(job)
	return w.push.SendPush(ctx, tokenStrings, job.Title, job.Body, data)
}

// buildPushData merges the job's free-form Data field with the
// canonical fields the mobile FCM tap handler expects. Pulled out
// so deliverPush stays under the 50-line cap.
func buildPushData(job DeliveryJob) map[string]string {
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
	return data
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
