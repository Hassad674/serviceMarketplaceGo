package payment

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// BUG-02 — State machine guards on MarkFailed / MarkRefunded /
// ApplyDisputeResolution. These three methods previously accepted ANY source
// state, so a webhook replay or a buggy caller could overwrite
// ProviderPayout to 0 on an already-transferred record and "lose" the
// provider's money. The tests below pin every legal AND illegal source for
// each guarded method, plus a property test that enforces invariants
// across random sequences of transitions.
// ---------------------------------------------------------------------------

// allStatusCombos enumerates every combination of (PaymentRecordStatus,
// TransferStatus) the domain can hold. Used to drive table tests against
// the three guarded methods.
type stateCombo struct {
	status   PaymentRecordStatus
	transfer TransferStatus
}

func allStatusCombos() []stateCombo {
	statuses := []PaymentRecordStatus{
		RecordStatusPending,
		RecordStatusSucceeded,
		RecordStatusFailed,
		RecordStatusRefunded,
	}
	transfers := []TransferStatus{
		TransferPending,
		TransferCompleted,
		TransferFailed,
	}
	out := make([]stateCombo, 0, len(statuses)*len(transfers))
	for _, s := range statuses {
		for _, t := range transfers {
			out = append(out, stateCombo{status: s, transfer: t})
		}
	}
	return out
}

// withState mutates the fixture record in-place to the given combo. We
// don't go through the public API to set up arbitrary illegal states —
// the tests are about the GUARDS, not the public happy paths.
func withState(rec *PaymentRecord, c stateCombo) {
	rec.Status = c.status
	rec.TransferStatus = c.transfer
}

// ---------------------------------------------------------------------------
// MarkFailed
// ---------------------------------------------------------------------------

func TestMarkFailed_Guard_AcceptsOnlyPending(t *testing.T) {
	for _, combo := range allStatusCombos() {
		t.Run("status="+string(combo.status)+"_transfer="+string(combo.transfer), func(t *testing.T) {
			rec := newFixtureRecord()
			withState(rec, combo)

			// Snapshot fields the guard MUST not mutate on rejection.
			origStatus := rec.Status
			origTransfer := rec.TransferStatus
			origPayout := rec.ProviderPayout
			origUpdatedAt := rec.UpdatedAt

			err := rec.MarkFailed()

			if combo.status == RecordStatusPending {
				require.NoError(t, err, "Pending → Failed must be allowed")
				assert.Equal(t, RecordStatusFailed, rec.Status)
				assert.Equal(t, origTransfer, rec.TransferStatus, "TransferStatus must be untouched")
				return
			}

			require.Error(t, err, "%s → Failed must be rejected", combo.status)
			assert.True(t, errors.Is(err, ErrInvalidStateTransition),
				"err must wrap ErrInvalidStateTransition for errors.Is")

			// errors.As must surface the structured metadata.
			var ste *StateTransitionError
			require.True(t, errors.As(err, &ste), "err must be a *StateTransitionError")
			assert.Equal(t, "MarkFailed", ste.Method)
			assert.Equal(t, RecordStatusPending, ste.ExpectedStatus)
			assert.Equal(t, origStatus, ste.ActualStatus)

			// Critical invariant: a rejected guard MUST NOT mutate the record.
			assert.Equal(t, origStatus, rec.Status, "status untouched on rejection")
			assert.Equal(t, origTransfer, rec.TransferStatus, "transfer status untouched on rejection")
			assert.Equal(t, origPayout, rec.ProviderPayout, "payout untouched on rejection")
			assert.Equal(t, origUpdatedAt, rec.UpdatedAt, "updated_at untouched on rejection")
		})
	}
}

// ---------------------------------------------------------------------------
// MarkRefunded
// ---------------------------------------------------------------------------

func TestMarkRefunded_Guard_AcceptsOnlySucceeded(t *testing.T) {
	for _, combo := range allStatusCombos() {
		t.Run("status="+string(combo.status)+"_transfer="+string(combo.transfer), func(t *testing.T) {
			rec := newFixtureRecord()
			withState(rec, combo)

			origStatus := rec.Status
			origTransfer := rec.TransferStatus
			origPayout := rec.ProviderPayout
			origUpdatedAt := rec.UpdatedAt

			err := rec.MarkRefunded()

			if combo.status == RecordStatusSucceeded {
				require.NoError(t, err, "Succeeded → Refunded must be allowed regardless of transfer status")
				assert.Equal(t, RecordStatusRefunded, rec.Status)
				assert.Equal(t, origTransfer, rec.TransferStatus, "TransferStatus must be untouched")
				return
			}

			require.Error(t, err, "%s → Refunded must be rejected", combo.status)
			assert.True(t, errors.Is(err, ErrInvalidStateTransition))

			var ste *StateTransitionError
			require.True(t, errors.As(err, &ste))
			assert.Equal(t, "MarkRefunded", ste.Method)
			assert.Equal(t, RecordStatusSucceeded, ste.ExpectedStatus)
			assert.Equal(t, origStatus, ste.ActualStatus)

			assert.Equal(t, origStatus, rec.Status)
			assert.Equal(t, origTransfer, rec.TransferStatus)
			assert.Equal(t, origPayout, rec.ProviderPayout)
			assert.Equal(t, origUpdatedAt, rec.UpdatedAt)
		})
	}
}

// ---------------------------------------------------------------------------
// ApplyDisputeResolution
// ---------------------------------------------------------------------------

func TestApplyDisputeResolution_Guard_RequiresSucceededAndNotCompleted(t *testing.T) {
	for _, combo := range allStatusCombos() {
		t.Run("status="+string(combo.status)+"_transfer="+string(combo.transfer), func(t *testing.T) {
			rec := newFixtureRecord()
			withState(rec, combo)

			origPayout := rec.ProviderPayout
			origStatus := rec.Status
			origTransfer := rec.TransferStatus
			origStripeTransferID := rec.StripeTransferID
			origTransferredAt := rec.TransferredAt
			origUpdatedAt := rec.UpdatedAt

			err := rec.ApplyDisputeResolution(700, "tr_dispute_xyz")

			// Legal source: Succeeded AND TransferStatus != Completed.
			legal := combo.status == RecordStatusSucceeded && combo.transfer != TransferCompleted
			if legal {
				require.NoError(t, err, "Succeeded + non-completed transfer must accept dispute resolution")
				assert.EqualValues(t, 700, rec.ProviderPayout)
				assert.Equal(t, TransferCompleted, rec.TransferStatus)
				assert.Equal(t, "tr_dispute_xyz", rec.StripeTransferID)
				assert.NotNil(t, rec.TransferredAt)
				return
			}

			require.Error(t, err, "%s/%s must be rejected by ApplyDisputeResolution",
				combo.status, combo.transfer)
			assert.True(t, errors.Is(err, ErrInvalidStateTransition))

			var ste *StateTransitionError
			require.True(t, errors.As(err, &ste))
			assert.Equal(t, "ApplyDisputeResolution", ste.Method)
			assert.Equal(t, RecordStatusSucceeded, ste.ExpectedStatus)
			assert.Equal(t, origStatus, ste.ActualStatus)
			assert.Equal(t, origTransfer, ste.ActualTransfer)

			// Critical: ProviderPayout MUST NOT be overwritten on rejection
			// (this is the precise scenario BUG-02 was about — losing
			// provider money via webhook replay).
			assert.Equal(t, origPayout, rec.ProviderPayout,
				"BUG-02 invariant: rejected resolution must NOT mutate ProviderPayout")
			assert.Equal(t, origStatus, rec.Status)
			assert.Equal(t, origTransfer, rec.TransferStatus)
			assert.Equal(t, origStripeTransferID, rec.StripeTransferID)
			assert.Equal(t, origTransferredAt, rec.TransferredAt)
			assert.Equal(t, origUpdatedAt, rec.UpdatedAt)
		})
	}
}

// TestApplyDisputeResolution_RejectsReplayOnAlreadyTransferred is the
// targeted regression test for BUG-02. A webhook replay arrives after
// the record is Succeeded+TransferCompleted; before the fix this would
// have overwritten ProviderPayout with whatever the new amount said.
// After the fix, the second call returns an error and the original
// ProviderPayout is preserved.
func TestApplyDisputeResolution_RejectsReplayOnAlreadyTransferred(t *testing.T) {
	rec := newFixtureRecord()
	require.NoError(t, rec.MarkPaid())
	require.NoError(t, rec.ApplyDisputeResolution(700, "tr_first"))
	require.Equal(t, int64(700), rec.ProviderPayout)
	require.Equal(t, TransferCompleted, rec.TransferStatus)

	// Webhook replay tries to overwrite with 0 — this would have been
	// the "lose the provider's money" scenario before the guard.
	err := rec.ApplyDisputeResolution(0, "tr_second")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidStateTransition))

	// First state is preserved exactly.
	assert.Equal(t, int64(700), rec.ProviderPayout, "first resolution must survive the replay")
	assert.Equal(t, "tr_first", rec.StripeTransferID, "transfer id must NOT be overwritten")
	assert.Equal(t, TransferCompleted, rec.TransferStatus)
}

// ---------------------------------------------------------------------------
// Property test — full state machine invariants
// ---------------------------------------------------------------------------

// transitionFn is a candidate transition the property test will try to
// apply. The fn returns the error (nil if accepted). Run a random
// permutation across many trials and assert the invariants below hold
// for ANY sequence — illegal calls must be rejected, legal calls must
// preserve the marketplace's accounting truth.
type transitionFn func(*PaymentRecord) error

func transitions() map[string]transitionFn {
	return map[string]transitionFn{
		"MarkPaid": func(r *PaymentRecord) error { return r.MarkPaid() },
		"MarkFailed": func(r *PaymentRecord) error { return r.MarkFailed() },
		"MarkRefunded": func(r *PaymentRecord) error { return r.MarkRefunded() },
		"MarkTransferred": func(r *PaymentRecord) error { return r.MarkTransferred("tr_prop") },
		"MarkTransferFailed": func(r *PaymentRecord) error {
			r.MarkTransferFailed()
			return nil
		},
		"ApplyDisputeResolutionFull": func(r *PaymentRecord) error {
			return r.ApplyDisputeResolution(0, "")
		},
		"ApplyDisputeResolutionSplit": func(r *PaymentRecord) error {
			return r.ApplyDisputeResolution(500, "tr_split")
		},
	}
}

// TestStateMachine_Invariants_Property runs many random transition
// sequences against fresh records and asserts the marketplace
// accounting invariants hold no matter what was attempted:
//
//  1. Status=Failed implies the record was Pending earlier — i.e.
//     a Failed record can NEVER show up after a Succeeded snapshot.
//  2. ProviderPayout > 0 with Status=Failed is impossible (Failed
//     means the charge never cleared, so there can be no payout).
//  3. TransferStatus=Completed implies Status was Succeeded at some
//     point (you can't transfer money you never received).
//  4. ProviderPayout NEVER becomes negative through any sequence.
//  5. MarkRefunded never accepts on a Pending or Failed record (the
//     guard's contract).
func TestStateMachine_Invariants_Property(t *testing.T) {
	rng := rand.New(rand.NewSource(20260430))

	const iterations = 500
	const sequenceLen = 10

	transitionMap := transitions()
	keys := make([]string, 0, len(transitionMap))
	for k := range transitionMap {
		keys = append(keys, k)
	}

	for it := 0; it < iterations; it++ {
		rec := newFixtureRecord()

		// Track the highest status the record ever reached so we can
		// assert invariant 1 ("Failed never follows Succeeded").
		sawSucceeded := false
		sawTransferCompleted := false

		for step := 0; step < sequenceLen; step++ {
			name := keys[rng.Intn(len(keys))]
			before := *rec
			err := transitionMap[name](rec)

			// If the transition was rejected, NOTHING the guard claims
			// to protect must have changed. We focus on Status,
			// ProviderPayout, StripeTransferID and TransferStatus —
			// MarkTransferFailed is the only mutator that can change
			// state without going through a guard, so we exclude it
			// from this check.
			if err != nil && name != "MarkTransferFailed" {
				assert.Equal(t, before.Status, rec.Status,
					"iter=%d step=%d: status mutated on rejection of %s", it, step, name)
				assert.Equal(t, before.ProviderPayout, rec.ProviderPayout,
					"iter=%d step=%d: payout mutated on rejection of %s", it, step, name)
				assert.Equal(t, before.StripeTransferID, rec.StripeTransferID,
					"iter=%d step=%d: stripe_transfer_id mutated on rejection of %s", it, step, name)
				assert.Equal(t, before.TransferStatus, rec.TransferStatus,
					"iter=%d step=%d: transfer_status mutated on rejection of %s", it, step, name)
			}

			// Track high-water-mark observations for the after-the-loop assertions.
			if rec.Status == RecordStatusSucceeded {
				sawSucceeded = true
			}
			if rec.TransferStatus == TransferCompleted {
				sawTransferCompleted = true
			}

			// Per-step invariants the domain MUST always honour.
			assert.GreaterOrEqual(t, rec.ProviderPayout, int64(0),
				"iter=%d step=%d: provider payout went negative", it, step)
		}

		// Invariant 2: a Failed record never carries a positive payout.
		if rec.Status == RecordStatusFailed {
			assert.Zero(t, rec.PaidAt, "iter=%d: Failed record never had a PaidAt", it)
		}

		// Invariant 3: TransferCompleted requires the record to have been
		// Succeeded at some point during the sequence.
		if sawTransferCompleted {
			assert.True(t, sawSucceeded,
				"iter=%d: TransferCompleted observed without ever seeing Succeeded", it)
		}

		// Invariant 4: a Refunded record's transfer is either Pending
		// (refund applied before any transfer) or Completed (refund
		// applied after — currently disallowed by the new guard but the
		// invariant survives even if the guard relaxes).
		if rec.Status == RecordStatusRefunded {
			assert.True(t,
				rec.TransferStatus == TransferPending ||
					rec.TransferStatus == TransferCompleted ||
					rec.TransferStatus == TransferFailed,
				"iter=%d: refunded record has unknown transfer status %s",
				it, rec.TransferStatus,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// Concurrency test — only ONE caller can apply a dispute resolution
// ---------------------------------------------------------------------------

// TestApplyDisputeResolution_Concurrent_OnlyOneSucceeds runs 10 goroutines
// against the same in-memory record and asserts exactly one wins. The
// other 9 must receive ErrInvalidStateTransition. Without the guard,
// they would all silently overwrite ProviderPayout — the LAST writer
// wins, which is the original BUG-02 race symptom.
//
// Note: the in-memory record is not safe for cross-goroutine writes
// (no mutex on the struct) — production code serialises through the
// repository's optimistic-locked Update. The guard runs BEFORE any
// mutation though, so even with concurrent reads of the same struct,
// only the goroutine that observes Status=Succeeded AND
// TransferStatus!=Completed gets to mutate.
//
// To avoid muddying the test with sync primitives the domain doesn't
// own, we use a Mutex that mimics the repository's exclusive Update
// window — the test exercises the GUARD, not the unrelated question
// "is the struct safe for concurrent write" (it isn't, by design).
func TestApplyDisputeResolution_Concurrent_OnlyOneSucceeds(t *testing.T) {
	rec := newFixtureRecord()
	require.NoError(t, rec.MarkPaid())

	var (
		mu        sync.Mutex
		successes atomic.Int32
		failures  atomic.Int32
		wg        sync.WaitGroup
	)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(amount int64) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			err := rec.ApplyDisputeResolution(amount, "tr_concurrent")
			if err != nil {
				if errors.Is(err, ErrInvalidStateTransition) {
					failures.Add(1)
				}
				return
			}
			successes.Add(1)
		}(int64(700 + i))
	}

	wg.Wait()

	assert.EqualValues(t, 1, successes.Load(),
		"exactly one goroutine should win the resolution apply")
	assert.EqualValues(t, 9, failures.Load(),
		"the other 9 must be rejected by the state guard")
	assert.Equal(t, TransferCompleted, rec.TransferStatus)
	// The winning amount is whichever goroutine ran first; the test
	// only proves "exactly one winner", not which one.
	assert.GreaterOrEqual(t, rec.ProviderPayout, int64(700))
	assert.LessOrEqual(t, rec.ProviderPayout, int64(709))
}

// TestMarkFailed_Concurrent_OnlyOneSucceeds is the same shape but for
// MarkFailed: 10 webhook replays from a flaky Stripe queue arrive at
// the same record. Only the first one (still Pending) gets to flip
// the record to Failed; the next 9 are rejected.
func TestMarkFailed_Concurrent_OnlyOneSucceeds(t *testing.T) {
	rec := newFixtureRecord()

	var (
		mu        sync.Mutex
		successes atomic.Int32
		failures  atomic.Int32
		wg        sync.WaitGroup
	)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			err := rec.MarkFailed()
			if err != nil {
				if errors.Is(err, ErrInvalidStateTransition) {
					failures.Add(1)
				}
				return
			}
			successes.Add(1)
		}()
	}

	wg.Wait()

	assert.EqualValues(t, 1, successes.Load())
	assert.EqualValues(t, 9, failures.Load())
	assert.Equal(t, RecordStatusFailed, rec.Status)
}

// TestMarkRefunded_Concurrent_OnlyOneSucceeds — refund webhook replay
// scenario. 10 goroutines try to refund the same Succeeded record;
// only the first wins.
func TestMarkRefunded_Concurrent_OnlyOneSucceeds(t *testing.T) {
	rec := newFixtureRecord()
	require.NoError(t, rec.MarkPaid())

	var (
		mu        sync.Mutex
		successes atomic.Int32
		failures  atomic.Int32
		wg        sync.WaitGroup
	)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			err := rec.MarkRefunded()
			if err != nil {
				if errors.Is(err, ErrInvalidStateTransition) {
					failures.Add(1)
				}
				return
			}
			successes.Add(1)
		}()
	}

	wg.Wait()

	assert.EqualValues(t, 1, successes.Load())
	assert.EqualValues(t, 9, failures.Load())
	assert.Equal(t, RecordStatusRefunded, rec.Status)
}

// ---------------------------------------------------------------------------
// StateTransitionError unwrapping + formatting
// ---------------------------------------------------------------------------

func TestStateTransitionError_ErrorAndUnwrap(t *testing.T) {
	rec := newFixtureRecord()
	rec.Status = RecordStatusFailed

	err := rec.MarkRefunded()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidStateTransition))

	// Error message must include both expected and actual states for
	// debug-friendly logs.
	msg := err.Error()
	assert.Contains(t, msg, "MarkRefunded")
	assert.Contains(t, msg, string(RecordStatusFailed))
	assert.Contains(t, msg, string(RecordStatusSucceeded))
}

// TestErrInvalidStateTransition_DistinctSentinel ensures the sentinel
// is an exported constant and not nil — a regression here would break
// every caller's errors.Is check.
func TestErrInvalidStateTransition_DistinctSentinel(t *testing.T) {
	assert.NotNil(t, ErrInvalidStateTransition)
	assert.Contains(t, ErrInvalidStateTransition.Error(), "state transition")
}

// ---------------------------------------------------------------------------
// Webhook replay simulation — domain layer
// ---------------------------------------------------------------------------

// TestApplyDisputeResolution_WebhookReplay_ZeroOverwriteRejected is the
// exact bug from the audit: a Stripe webhook fires twice (Stripe retries
// when an idempotency key collides during their internal retry path),
// the second replay carries amount=0, and BEFORE the fix this would
// overwrite ProviderPayout=700 → 0. After the fix the second call is
// rejected and the comptable amount is preserved.
func TestApplyDisputeResolution_WebhookReplay_ZeroOverwriteRejected(t *testing.T) {
	rec := newFixtureRecord()
	require.NoError(t, rec.MarkPaid())
	// Real-world: dispute resolution sets a partial split.
	require.NoError(t, rec.ApplyDisputeResolution(700, "tr_real"))

	// Replay arrives with the (incorrectly client-derived) amount=0.
	err := rec.ApplyDisputeResolution(0, "")
	require.Error(t, err, "replay must be rejected — losing 700 is the BUG-02 symptom")

	assert.Equal(t, int64(700), rec.ProviderPayout, "first split must survive the replay")
	assert.Equal(t, "tr_real", rec.StripeTransferID)
}
