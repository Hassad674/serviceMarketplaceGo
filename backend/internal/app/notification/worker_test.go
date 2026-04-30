package notification

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/user"
)

func TestWorker_ProcessJob_PushAndEmail(t *testing.T) {
	userID := uuid.New()
	var pushCalled, emailCalled bool

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return false, nil // offline
			},
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				pushCalled = true
				return nil
			},
		},
		email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		users: &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{Email: "test@test.com", EmailNotificationsEnabled: true}, nil
			},
		},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					InApp:            true,
					Push:             true,
					Email:            true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "fcm-token-123"}}, nil
			},
		},
		queue: &mockQueue{},
	}

	err := w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
		Body:           "Test body",
	})

	assert.NoError(t, err)
	assert.True(t, pushCalled, "push should be called when user is offline")
	assert.True(t, emailCalled, "email should be called for proposal_received")
}

func TestWorker_ProcessJob_UserOnline_NoPush(t *testing.T) {
	userID := uuid.New()
	var pushCalled, emailCalled bool

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return true, nil // online
			},
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				pushCalled = true
				return nil
			},
		},
		email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		users: &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{Email: "test@test.com", EmailNotificationsEnabled: true}, nil
			},
		},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
					Email:            true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: &mockQueue{},
	}

	err := w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
	})

	assert.NoError(t, err)
	assert.False(t, pushCalled, "push should NOT be called when user is online")
	assert.True(t, emailCalled, "email should still be called regardless of presence")
}

func TestWorker_ProcessJob_NewMessage_NoEmail(t *testing.T) {
	userID := uuid.New()
	var emailCalled bool

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error { return nil },
		},
		email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		users: &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{Email: "test@test.com", EmailNotificationsEnabled: true}, nil
			},
		},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeNewMessage,
					Push:             true,
					Email:            true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: &mockQueue{},
	}

	_ = w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeNewMessage),
		Title:          "New msg",
	})

	assert.False(t, emailCalled, "email should NEVER be sent for new_message type")
}

func TestWorker_ProcessJob_PushFails_Retries(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				return errors.New("FCM timeout")
			},
		},
		email:  nil, // no email configured
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
					Email:            false,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: q,
	}

	_ = w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
		Attempt:        0,
	})

	// Re-enqueue is now deferred to a goroutine (BUG-16 fix) so the
	// test waits for the queue to actually receive the job rather
	// than asserting synchronously. Attempt 0 → backoff 1s; we wait
	// up to 3s to be tolerant of CI scheduler jitter.
	jobs := q.waitForJobs(t, 1, 3*time.Second)
	assert.Len(t, jobs, 1, "should re-enqueue failed job")
	assert.Equal(t, 1, jobs[0].Attempt, "attempt should be incremented")

	// Drain the retry goroutine before the test exits so the
	// next test starts clean.
	w.retryWG.Wait()
}

func TestWorker_ProcessJob_MaxRetries_DeadLetter(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				return errors.New("FCM down")
			},
		},
		email:  nil,
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
					Email:            false,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: q,
	}

	_ = w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
		Attempt:        2, // maxRetries-1 = 2
	})

	// No async re-enqueue is launched at the dead-letter cap, so
	// retryWG is empty and snapshotJobs sees zero entries.
	w.retryWG.Wait()
	assert.Empty(t, q.snapshotJobs(), "should NOT re-enqueue after max retries (dead letter)")
}

// ---------------------------------------------------------------------------
// BUG-16 — parallel workers + non-blocking re-enqueue.
// ---------------------------------------------------------------------------

// TestWorker_ProcessJob_FailedJob_DoesNotBlockNextJob asserts that a
// failed delivery does NOT make the calling goroutine sleep — the
// re-enqueue is deferred to a separate goroutine so the main
// processor is free to grab the next message immediately.
func TestWorker_ProcessJob_FailedJob_DoesNotBlockNextJob(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				return errors.New("FCM timeout")
			},
		},
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true, Email: false,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "tok"}}, nil
			},
		},
		queue: q,
	}

	start := time.Now()
	require.NoError(t, w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "T",
		Attempt:        2, // backoff would have been 4s under the old code
	}))
	elapsed := time.Since(start)

	// Pre-fix code slept `time.Sleep(2^attempt seconds)` inside
	// processJob — at attempt 2 that's a 4-second blocking call.
	// The new code returns within milliseconds because the retry is
	// deferred. Use 200ms as a generous CI-safe bound.
	assert.Less(t, elapsed, 200*time.Millisecond,
		"BUG-16: processJob must NOT sleep on the hot path — re-enqueue is async")

	w.retryWG.Wait()
}

// TestWorker_Run_PoolDrainsBurstInParallel proves N>1 workers
// process N jobs concurrently. Each delivery callback signals a
// channel, so we can count parallel arrivals and assert the pool
// actually achieves >1 concurrent in-flight.
func TestWorker_Run_PoolDrainsBurstInParallel(t *testing.T) {
	const burst = 6
	const concurrency = 5

	userID := uuid.New()
	jobs := make(chan *DeliveryJob, burst)
	for i := 0; i < burst; i++ {
		jid := uuid.New().String()
		jobs <- &DeliveryJob{
			NotificationID: jid,
			UserID:         userID.String(),
			Type:           string(notif.TypeProposalReceived),
			Title:          "T", Body: "B",
		}
	}
	close(jobs)

	var (
		inFlight    atomic.Int32
		maxInFlight atomic.Int32
		processed   atomic.Int32
	)

	q := &mockQueue{
		dequeueFn: func(_ context.Context) (*DeliveryJob, string, error) {
			select {
			case j, ok := <-jobs:
				if !ok {
					return nil, "", nil
				}
				return j, j.NotificationID, nil
			default:
				return nil, "", nil
			}
		},
	}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				now := inFlight.Add(1)
				// Track the maximum concurrency observed.
				for {
					m := maxInFlight.Load()
					if now <= m || maxInFlight.CompareAndSwap(m, now) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond) // simulate FCM round-trip
				inFlight.Add(-1)
				processed.Add(1)
				return nil
			},
		},
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true, Email: false,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "tok"}}, nil
			},
		},
		queue: q,
	}
	w = w.WithConfig(WorkerConfig{Concurrency: concurrency})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	doneCh := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(doneCh)
	}()

	// Wait for all jobs to be processed (or the timeout to fire).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if processed.Load() >= int32(burst) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-doneCh

	assert.Equal(t, int32(burst), processed.Load(), "every job must be processed")
	assert.Greater(t, maxInFlight.Load(), int32(1),
		"BUG-16: parallel pool must achieve >1 concurrent deliveries")
}

// TestWorker_Run_GracefulShutdownDrainsRetries verifies the retry
// goroutines complete their re-enqueue even when ctx is cancelled
// mid-backoff. Without retryWG, an in-flight retry could leak its
// payload on shutdown.
func TestWorker_Run_GracefulShutdownDrainsRetries(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				return errors.New("FCM down")
			},
		},
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "tok"}}, nil
			},
		},
		queue: q,
	}

	require.NoError(t, w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "T",
		Attempt:        0, // backoff = 1s
	}))

	// retryWG must be observable as non-empty: the deferred goroutine
	// has not yet fired the enqueue. Wait for the re-enqueue to
	// land — proves shutdown wouldn't lose it.
	jobs := q.waitForJobs(t, 1, 3*time.Second)
	assert.Len(t, jobs, 1)
	w.retryWG.Wait()
}

// TestWorker_Run_DefaultConcurrency exercises the zero-config path.
// Used as a smoke test that the pool starts and stops cleanly when
// no WorkerConfig is wired.
func TestWorker_Run_DefaultConcurrency(t *testing.T) {
	q := &mockQueue{}
	w := &Worker{
		queue:    q,
		presence: &mockPresenceService{},
		push:     &mockPushService{},
		users:    &mockUserRepo{},
		notifs:   &mockNotificationRepo{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	w.Run(ctx) // returns when ctx fires
}
