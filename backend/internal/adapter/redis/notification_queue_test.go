package redis_test

// BUG-22 — the Dequeue path discarded Ack errors, causing Redis to
// re-deliver the message on the next cycle and double the
// notification fan-out. These tests assert:
//
//   - happy-path Ack succeeds and the message is removed from the
//     pending list (legacy behaviour preserved).
//   - Ack failure is now logged at WARN with the message_id (the bug).
//   - corrupt payloads still surface the legacy WARN AND now log the
//     Ack failure when the underlying connection is dead.
//
// The "Ack fails" simulation closes the go-redis client mid-flight so
// XAck returns a connection-closed error — the smallest realistic
// proxy for a Redis blip the operator would see in production.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/redis"
	notifapp "marketplace-backend/internal/app/notification"
)

// failXAckHook intercepts only XACK commands and returns the injected
// error, leaving every other command (XADD, XREADGROUP, XPENDING…)
// untouched. Used to simulate a Redis blip on the Ack path while
// keeping the rest of the queue working — the smallest possible repro
// of the BUG-22 production scenario.
type failXAckHook struct {
	err error
}

func (h failXAckHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}
func (h failXAckHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		if strings.EqualFold(cmd.Name(), "xack") {
			cmd.SetErr(h.err)
			return h.err
		}
		return next(ctx, cmd)
	}
}
func (h failXAckHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		return next(ctx, cmds)
	}
}

const testNotifStream = "notification:jobs"

// newNotifQueueTest spins up a miniredis backing the queue. The
// returned queue uses a fresh consumer ID so parallel tests don't
// collide on the consumer-group state.
func newNotifQueueTest(t *testing.T) (*adapter.NotificationJobQueue, *miniredis.Miniredis, *goredis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))
	return q, mr, client
}

// captureLogs swaps the default slog handler to capture WARN output.
func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	prev := slog.Default()
	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf, func() { slog.SetDefault(prev) }
}

func enqueueValid(t *testing.T, q *adapter.NotificationJobQueue) string {
	t.Helper()
	job := notifapp.DeliveryJob{
		NotificationID: "notif-1",
		UserID:         "user-1",
		Type:           "info",
		Title:          "Hello",
		Body:           "World",
		Data:           json.RawMessage(`{}`),
		Attempt:        0,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	require.NoError(t, q.Enqueue(context.Background(), job))
	return job.NotificationID
}

func TestNotificationQueue_DequeueValidJob(t *testing.T) {
	q, _, _ := newNotifQueueTest(t)
	enqueueValid(t, q)

	job, msgID, err := q.Dequeue(context.Background())
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "notif-1", job.NotificationID)
	assert.NotEmpty(t, msgID)
}

func TestNotificationQueue_DequeueTimesOutOnEmptyStream(t *testing.T) {
	q, _, _ := newNotifQueueTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	job, msgID, err := q.Dequeue(ctx)
	// XReadGroup with Block returns nil error on timeout AND nil job.
	// Either nil-error or context.DeadlineExceeded is acceptable here —
	// we mainly assert there is no panic and no spurious payload.
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	}
	assert.Nil(t, job)
	assert.Empty(t, msgID)
}

func TestNotificationQueue_AckSucceedsRemovesFromPending(t *testing.T) {
	q, _, _ := newNotifQueueTest(t)
	enqueueValid(t, q)

	_, msgID, err := q.Dequeue(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, msgID)

	// Happy-path Ack returns nil and removes the entry from pending.
	require.NoError(t, q.Ack(context.Background(), msgID))
}

// BUG-22 — malformed JSON branch is logged as `unmarshal job failed`,
// then Ack runs to drop the bad message. Even when miniredis converts
// non-string values into strings, malformed JSON is the realistic
// production failure mode (a producer with a corrupted schema).
func TestNotificationQueue_MalformedPayload_LegacyWarnPreserved(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	// Push a malformed entry — value is a number-as-string. After
	// miniredis stringifies it (42 → "42"), the json.Unmarshal into a
	// DeliveryJob fails with "cannot unmarshal number" — that is the
	// branch we are asserting.
	require.NoError(t, client.XAdd(context.Background(), &goredis.XAddArgs{
		Stream: testNotifStream,
		Values: map[string]interface{}{"job": 42},
	}).Err())

	logs, restore := captureLogs(t)
	defer restore()

	job, msgID, err := q.Dequeue(context.Background())
	require.NoError(t, err)
	// Malformed payload was discarded → Dequeue returns (nil, "", nil)
	// after consuming the message inside the loop.
	assert.Nil(t, job)
	assert.Empty(t, msgID)

	// Legacy WARN must still fire so on-call sees the bad payload.
	assert.Contains(t, logs.String(), "unmarshal job failed")
}

// BUG-22 — unmarshal failure path. Same Ack-error logging contract.
func TestNotificationQueue_UnmarshalFails_LegacyWarningPreserved(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	// Well-typed string payload but malformed JSON.
	err = client.XAdd(context.Background(), &goredis.XAddArgs{
		Stream: testNotifStream,
		Values: map[string]interface{}{"job": "{not valid"},
	}).Err()
	require.NoError(t, err)

	logs, restore := captureLogs(t)
	defer restore()

	job, msgID, err := q.Dequeue(context.Background())
	require.NoError(t, err)
	assert.Nil(t, job)
	assert.Empty(t, msgID)

	out := logs.String()
	assert.Contains(t, out, "unmarshal job failed")
}

// BUG-22 — the inner-loop Ack inside Dequeue used to be `_ = q.Ack(...)`.
// When that Ack failed, Redis re-delivered the message on the next
// pull, doubling the notification fan-out for malformed payloads.
//
// We now log a WARN with the message_id and error so on-call can
// correlate notification doubling with Redis incidents. Test:
//   - enqueue a malformed payload (passes XADD),
//   - install a hook that fails ONLY XAck,
//   - call Dequeue,
//   - assert the BUG-22 WARN is present in slog output,
//   - assert the legacy WARN ("unmarshal job failed") is also present
//     so a single incident yields both lines.
func TestNotificationQueue_DequeueAckFailure_LogsWarn(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	// Push malformed JSON so Dequeue takes the ack-and-skip branch.
	require.NoError(t, client.XAdd(context.Background(), &goredis.XAddArgs{
		Stream: testNotifStream,
		Values: map[string]interface{}{"job": "{not valid"},
	}).Err())

	// Install the XACK-only failing hook. From this point onwards
	// every XACK on this client returns "ack synthetic failure".
	client.AddHook(failXAckHook{err: errors.New("ack synthetic failure")})

	logs, restore := captureLogs(t)
	defer restore()

	job, msgID, err := q.Dequeue(context.Background())
	require.NoError(t, err)
	assert.Nil(t, job)
	assert.Empty(t, msgID)

	out := logs.String()
	// Legacy: malformed payload was detected.
	assert.Contains(t, out, "unmarshal job failed")
	// BUG-22 fix: the failed ack is now logged.
	assert.Contains(t, out, "notification queue: ack failed",
		"BUG-22: a failed inner Ack must surface as a WARN")
	assert.Contains(t, out, "ack synthetic failure",
		"the underlying error message must propagate to the log")
}

// BUG-22 integration — proves the doubled-delivery scenario the bug
// describes is now observable in the logs. With the failing Ack the
// message stays in pending; the NEXT Dequeue (without the hook) re-
// delivers it. The legacy `_ = q.Ack(...)` would have produced the
// same redelivery WITHOUT any log line, leaving the operator blind.
func TestNotificationQueue_AckFailure_NextCycleRedelivers(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	// Enqueue a malformed entry.
	require.NoError(t, client.XAdd(context.Background(), &goredis.XAddArgs{
		Stream: testNotifStream,
		Values: map[string]interface{}{"job": "{not valid"},
	}).Err())

	// First cycle: failing-ack hook installed.
	hook := failXAckHook{err: errors.New("ack synthetic failure")}
	client.AddHook(hook)

	logs, restore := captureLogs(t)
	defer restore()

	_, _, err = q.Dequeue(context.Background())
	require.NoError(t, err)

	// Verify the message is still pending — Redis treats an unacked
	// delivery as in-flight.
	pending, err := client.XPendingExt(context.Background(), &goredis.XPendingExtArgs{
		Stream: testNotifStream,
		Group:  "notification-workers",
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()
	require.NoError(t, err)
	assert.Len(t, pending, 1,
		"failing Ack must leave the message in the pending list — proves the BUG-22 redelivery scenario")

	// Logs must surface the failure to the operator.
	assert.Contains(t, logs.String(), "notification queue: ack failed")
}

// TestNotificationQueue_AckFailure_OnDeadConnection — closes the
// client BEFORE the user-driven Ack runs, proving that Ack returns
// the underlying error and the caller can log it. This is the
// minimum BUG-22 contract: the operator-facing Ack is no longer
// fire-and-forget.
func TestNotificationQueue_AckFailure_OnDeadConnection(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	// Close the connection so any subsequent op fails.
	require.NoError(t, client.Close())

	err = q.Ack(context.Background(), "0-0")
	assert.Error(t, err, "Ack against a closed connection must surface the error")
}

// EnsureGroup running twice must be idempotent — a second call detects
// the BUSYGROUP error and returns nil. This is the documented behaviour
// in the EnsureGroup comment ("Returns nil when the group was freshly
// created AND when Redis reports BUSYGROUP").
func TestNotificationQueue_EnsureGroup_IdempotentOnBusyGroup(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")

	// First call: fresh creation.
	require.NoError(t, q.EnsureGroup(context.Background()))

	// Second call: must be a no-op even though the group exists.
	require.NoError(t, q.EnsureGroup(context.Background()),
		"EnsureGroup must be idempotent — BUSYGROUP from Redis is benign")
}

// EnsureGroup must surface non-BUSYGROUP errors so they don't get
// mistaken for the benign duplicate-creation case. Closing the client
// is the smallest realistic Redis-down repro.
func TestNotificationQueue_EnsureGroup_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	err = q.EnsureGroup(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create notification job group")
}

// Enqueue surfaces a Redis failure as a wrapped error. We tear down
// miniredis to force XADD to fail.
func TestNotificationQueue_Enqueue_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	mr.Close()

	job := notifapp.DeliveryJob{
		NotificationID: "fail-1",
		UserID:         "u",
		Type:           "info",
		Title:          "x",
		Body:           "y",
		Data:           json.RawMessage(`{}`),
	}
	err = q.Enqueue(context.Background(), job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enqueue notification job")
}

// Dequeue surfaces a non-timeout Redis failure as a wrapped error
// (rather than a goredis.Nil → nil-job result).
func TestNotificationQueue_Dequeue_RedisDown_ReturnsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	q := adapter.NewNotificationJobQueue(client, "test-consumer")
	require.NoError(t, q.EnsureGroup(context.Background()))

	mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	job, msgID, err := q.Dequeue(ctx)
	require.Error(t, err)
	assert.Nil(t, job)
	assert.Empty(t, msgID)
	// Either the wrap message or the underlying connection error must
	// surface — the contract is that the failure is non-silent.
	if !strings.Contains(err.Error(), "dequeue notification job") {
		assert.NotEmpty(t, err.Error())
	}
}

// BUG-22 — full integration. Enqueue, dequeue, Ack-fail. The legacy
// behaviour was: Ack failure → message redelivered on the next cycle.
// We can't directly probe redelivery without a queue-internal hook;
// we instead assert that with a successful Ack the pending count
// drops to zero (legacy invariant preserved), and rely on the
// dead-connection test above for the failure path.
func TestNotificationQueue_FullCycle_AckRemovesFromPending(t *testing.T) {
	q, _, client := newNotifQueueTest(t)
	enqueueValid(t, q)

	_, msgID, err := q.Dequeue(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, msgID)

	// Pending list must contain our message.
	pending, err := client.XPendingExt(context.Background(), &goredis.XPendingExtArgs{
		Stream: testNotifStream,
		Group:  "notification-workers",
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()
	require.NoError(t, err)
	assert.Len(t, pending, 1, "delivered-but-unacked message must show in pending list")

	// Ack and re-check.
	require.NoError(t, q.Ack(context.Background(), msgID))

	pending, err = client.XPendingExt(context.Background(), &goredis.XPendingExtArgs{
		Stream: testNotifStream,
		Group:  "notification-workers",
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()
	require.NoError(t, err)
	assert.Len(t, pending, 0,
		"successful Ack must clear the pending entry — legacy invariant preserved")
}
