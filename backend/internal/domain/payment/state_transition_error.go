package payment

import "fmt"

// StateTransitionError carries enough context to debug an illegal state
// transition without making the caller re-derive what was expected.
//
// Used by MarkFailed / MarkRefunded / ApplyDisputeResolution when a guard
// rejects the call. The wrapped sentinel is always ErrInvalidStateTransition
// so callers can use errors.Is to detect the class of failure, AND can use
// errors.As to inspect the metadata if they need to surface a precise
// observability signal (BUG-02 webhook replay diagnosis).
//
// Example:
//
//	if err := record.MarkRefunded(); err != nil {
//	    var ste *StateTransitionError
//	    if errors.As(err, &ste) {
//	        slog.Error("state guard rejected",
//	            "method", ste.Method,
//	            "want_status", ste.ExpectedStatus,
//	            "got_status", ste.ActualStatus,
//	        )
//	    }
//	    return err
//	}
type StateTransitionError struct {
	// Method is the domain method that rejected the transition
	// (e.g. "MarkRefunded"). Provided so logs/audits can be filtered
	// per-call-site without parsing the message.
	Method string

	// ExpectedStatus is the PaymentRecordStatus the guard required.
	// Empty when the guard only checks TransferStatus.
	ExpectedStatus PaymentRecordStatus

	// ActualStatus is the PaymentRecordStatus the record currently has.
	ActualStatus PaymentRecordStatus

	// ExpectedTransfer is the TransferStatus the guard required (when
	// the guard mixes status + transfer status, e.g. ApplyDisputeResolution
	// requires Status=Succeeded AND TransferStatus!=Completed). Empty when
	// the guard does not constrain the transfer status.
	ExpectedTransfer TransferStatus

	// ActualTransfer is the TransferStatus the record currently has.
	ActualTransfer TransferStatus
}

// Error implements the error interface with a structured message that is
// safe to surface in logs but stays opaque enough not to leak business
// logic to API consumers (the handler still maps the error to a generic
// 409 / 422 response).
func (e *StateTransitionError) Error() string {
	return fmt.Sprintf(
		"%s: invalid state transition (status=%s, transfer_status=%s; want status=%s, transfer!=%s)",
		e.Method, e.ActualStatus, e.ActualTransfer, e.ExpectedStatus, e.ExpectedTransfer,
	)
}

// Unwrap returns ErrInvalidStateTransition so errors.Is(err,
// ErrInvalidStateTransition) succeeds for any guard rejection across
// the three guarded methods.
func (e *StateTransitionError) Unwrap() error {
	return ErrInvalidStateTransition
}
