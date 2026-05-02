package pendingevent

import "errors"

var (
	// ErrEventNotFound is returned when a lookup by ID yields no result.
	ErrEventNotFound = errors.New("pending event not found")

	// ErrInvalidEventType is returned when an event type is not one of
	// the recognised values defined in entity.go.
	ErrInvalidEventType = errors.New("invalid pending event type")

	// ErrEmptyPayload is returned when an event is created without a
	// payload. The worker needs the payload to decode the typed
	// arguments for the handler.
	ErrEmptyPayload = errors.New("pending event payload cannot be empty")

	// ErrZeroFiresAt is returned when fires_at is the zero time. Every
	// event must carry an explicit fire moment so the worker query
	// stays predictable.
	ErrZeroFiresAt = errors.New("pending event fires_at must be set")

	// ErrInvalidStatus is returned when a transition is attempted from
	// an incompatible current status.
	ErrInvalidStatus = errors.New("invalid pending event status for this operation")

	// ErrMissingStripeEventID is returned when a TypeStripeWebhook
	// event is built without its Stripe `evt_*` identifier. The id
	// is mandatory because it is the deduplication key for retried
	// deliveries — without it the partial unique index on
	// pending_events.stripe_event_id is bypassed and Stripe
	// re-deliveries would produce duplicate worker dispatches.
	ErrMissingStripeEventID = errors.New("pending event of type stripe.webhook requires StripeEventID")
)
