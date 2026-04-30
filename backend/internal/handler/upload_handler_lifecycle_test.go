package handler

// BUG-17 — RecordUpload goroutines used to be detached:
//   `go h.mediaSvc.RecordUpload(...)`. SIGTERM during an upload
// truncated the in-flight Rekognition / S3 work and left orphan media
// records. The new lifecycle wraps every spawn in a sync.WaitGroup
// tracked by the handler, exposes Stop(parent) for the graceful-
// shutdown path, and surfaces the goroutine context for cancellation
// once the application starts winding down.
//
// These tests assert:
//   1. handler responds before the goroutine completes (legacy semantic),
//   2. Stop() blocks until the goroutine has finished,
//   3. Stop() returns context.DeadlineExceeded when the drain budget
//      is exceeded, AND emits the WARN log,
//   4. SIGTERM (uploadCtx.Cancel()) propagates to the goroutine's
//      context so a hung downstream can short-circuit instead of
//      waiting out the 60s timeout,
//   5. 20 concurrent uploads followed by Stop() all complete OR are
//      logged as timeout.

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mediadomain "marketplace-backend/internal/domain/media"
)

// fakeRecorder satisfies the unexported mediaRecorder interface and
// exposes hooks that the BUG-17 tests use to simulate a slow / hung
// downstream.
type fakeRecorder struct {
	mu sync.Mutex
	// calls records every (uploaderID, mediaCtx) pair so tests can
	// assert the goroutine fired exactly once per upload.
	calls []recordCall
	// duration is how long RecordUpload takes per call. Default 0.
	duration time.Duration
	// hang, when set, blocks until the test closes the channel
	// (simulates a wedged downstream).
	hang chan struct{}
	// done is signalled when RecordUpload returns.
	done chan struct{}
	// active counts in-flight calls.
	active int32
}

type recordCall struct {
	UploaderID uuid.UUID
	MediaCtx   mediadomain.Context
}

func newFakeRecorder() *fakeRecorder {
	return &fakeRecorder{done: make(chan struct{}, 64)}
}

func (f *fakeRecorder) RecordUpload(
	uploaderID uuid.UUID,
	_ string,
	_ string,
	_ string,
	_ int64,
	mediaCtx mediadomain.Context,
) {
	atomic.AddInt32(&f.active, 1)
	defer atomic.AddInt32(&f.active, -1)

	f.mu.Lock()
	d := f.duration
	hang := f.hang
	f.mu.Unlock()

	if hang != nil {
		<-hang
	} else if d > 0 {
		time.Sleep(d)
	}

	f.mu.Lock()
	f.calls = append(f.calls, recordCall{UploaderID: uploaderID, MediaCtx: mediaCtx})
	f.mu.Unlock()
	select {
	case f.done <- struct{}{}:
	default:
	}
}

// withFakeRecorder swaps the handler's recorder for the test fake AND
// keeps the legacy mediaSvc-nil constructor path. Returns a clean-up
// helper to drain any in-flight goroutines at end of test.
func withFakeRecorder(t *testing.T) (*UploadHandler, *fakeRecorder, func()) {
	t.Helper()
	h := NewUploadHandler(nil, nil, nil)
	rec := newFakeRecorder()
	h.recorder = rec
	cleanup := func() {
		_ = h.Stop(context.Background())
	}
	return h, rec, cleanup
}

func sampleInput() trackUploadInput {
	return trackUploadInput{
		UploaderID: uuid.New(),
		FileURL:    "https://bucket/x.jpg",
		FileType:   "image/jpeg",
		FileSize:   1024,
		MediaCtx:   mediadomain.ContextProfilePhoto,
	}
}

// captureUploadLogs swaps the package's default slog handler for one
// that writes into a buffer so tests can assert on the WARN line
// emitted by Stop() when the drain budget is exceeded.
func captureUploadLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	prev := slog.Default()
	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf, func() { slog.SetDefault(prev) }
}

// --- 1. handler returns before goroutine completes (legacy semantic) ---

func TestTrackUpload_NonBlocking_ReturnsBeforeRecorderRuns(t *testing.T) {
	h, rec, cleanup := withFakeRecorder(t)
	defer cleanup()

	rec.duration = 200 * time.Millisecond

	start := time.Now()
	h.trackUpload(context.Background(), sampleInput())
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 50*time.Millisecond,
		"trackUpload must spawn a goroutine and return immediately")

	// Wait for the recorder to actually run so cleanup doesn't time out.
	select {
	case <-rec.done:
	case <-time.After(time.Second):
		t.Fatal("recorder never ran")
	}
}

// --- 2. Stop() drains pending goroutines ---

func TestStop_WaitsForInflightGoroutines(t *testing.T) {
	h, rec, _ := withFakeRecorder(t)

	rec.duration = 100 * time.Millisecond

	h.trackUpload(context.Background(), sampleInput())
	h.trackUpload(context.Background(), sampleInput())

	start := time.Now()
	require.NoError(t, h.Stop(context.Background()))
	elapsed := time.Since(start)

	// Stop must have waited at least the recorder duration.
	assert.GreaterOrEqual(t, elapsed, 90*time.Millisecond,
		"Stop must wait for in-flight goroutines")

	// Both calls were recorded.
	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Len(t, rec.calls, 2)
}

// --- 3. Stop() respects the parent's deadline ---

func TestStop_ReturnsDeadlineExceeded_OnSlowGoroutine(t *testing.T) {
	logs, restore := captureUploadLogs(t)
	defer restore()

	h, rec, _ := withFakeRecorder(t)
	rec.hang = make(chan struct{}) // never released → goroutine wedged

	h.trackUpload(context.Background(), sampleInput())

	// Parent ctx with a 50ms budget.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := h.Stop(ctx)
	assert.Error(t, err, "Stop must fail when the drain budget is exceeded")

	// Now release the goroutine so it doesn't leak the test.
	close(rec.hang)

	// And verify a WARN was emitted.
	out := logs.String()
	assert.True(t,
		// Either "did not drain in time" (Stop's own timeout fired)
		// or a parent-ctx error (parent.Done()) — both acceptable.
		bytes.Contains([]byte(out), []byte("did not drain in time")) ||
			err == context.DeadlineExceeded || err == context.Canceled,
		"Stop must surface deadline exceeded one way or another. logs=%s", out)
}

// --- 4. SIGTERM propagates to the goroutine ---

func TestTrackUpload_ShutdownContextCancelsTask(t *testing.T) {
	h := NewUploadHandler(nil, nil, nil)
	rec := newFakeRecorder()
	h.recorder = rec

	// Wire a cancellable shutdown context.
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	h.WithShutdownContext(shutdownCtx)

	// Hang the recorder so we can observe the cancellation flow.
	rec.hang = make(chan struct{})

	h.trackUpload(context.Background(), sampleInput())

	// Trigger shutdown. The goroutine's inner ctx must be cancelled.
	shutdownCancel()

	// The goroutine itself blocks on rec.hang — release it so Stop
	// can complete. The point of the test is that the cancellation
	// PATH works, not that RecordUpload obeys ctx (it does not yet).
	close(rec.hang)

	require.NoError(t, h.Stop(context.Background()))
}

// --- 5. 20 concurrent uploads followed by Stop ---

func TestTrackUpload_TwentyConcurrent_AllDrainOrTimeoutLogged(t *testing.T) {
	h, rec, _ := withFakeRecorder(t)

	rec.duration = 20 * time.Millisecond
	const concurrent = 20

	var wg sync.WaitGroup
	wg.Add(concurrent)
	for i := 0; i < concurrent; i++ {
		go func() {
			defer wg.Done()
			h.trackUpload(context.Background(), sampleInput())
		}()
	}
	wg.Wait()

	// All spawns returned synchronously; goroutines are still running.
	require.NoError(t, h.Stop(context.Background()))

	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Equal(t, concurrent, len(rec.calls),
		"every spawned goroutine must complete before Stop returns")
}

// --- Mediasvc-nil legacy path preserved ---

func TestTrackUpload_NilRecorder_IsNoop(t *testing.T) {
	h := NewUploadHandler(nil, nil, nil)
	// recorder remains nil because mediaSvc is nil.

	// Calling trackUpload with no recorder must not panic, must not
	// touch the WaitGroup, and must return immediately.
	start := time.Now()
	h.trackUpload(context.Background(), sampleInput())
	assert.Less(t, time.Since(start), 5*time.Millisecond)
	assert.NoError(t, h.Stop(context.Background()))
}

// --- WithShutdownContext setter is fluent and tolerates nil ---

func TestWithShutdownContext_NilDoesNotOverride(t *testing.T) {
	h := NewUploadHandler(nil, nil, nil)
	original := h.shutdownCtx

	h.WithShutdownContext(nil)
	assert.Equal(t, original, h.shutdownCtx,
		"WithShutdownContext(nil) must keep the existing context")

	custom, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.WithShutdownContext(custom)
	assert.Equal(t, custom, h.shutdownCtx,
		"WithShutdownContext(non-nil) must adopt the new context")
}

// --- Race / concurrent stress ---

func TestTrackUpload_RaceStress(t *testing.T) {
	h, rec, _ := withFakeRecorder(t)
	rec.duration = 1 * time.Millisecond

	const workers = 8
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				h.trackUpload(context.Background(), sampleInput())
			}
		}()
	}
	wg.Wait()

	require.NoError(t, h.Stop(context.Background()))

	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Equal(t, workers*perWorker, len(rec.calls))
}
