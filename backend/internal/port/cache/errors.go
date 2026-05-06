// Package cache holds port-level types shared between the cache
// adapters (currently the Redis fast-path) and the application
// services that consume them.
//
// V8 NEW-2: the webhook idempotency claimer used to type-assert
// `*adapter/redis.CacheError` to detect cache outages. That broke
// the hexagonal layer rule (`app/` must depend on `port/`, never on
// `adapter/`). Hosting the error type here lets the app layer branch
// on cache failures without importing the redis package; the redis
// adapter still emits the same concrete value, just defined one
// directory over.
package cache

// Error is returned by cache adapters when a transport-level fault
// (network, dial, command timeout) prevents the cache from servicing
// a request. Callers MUST treat this as "cache unavailable, fall
// through to the durable layer" — never as "cache says no".
//
// The wrapped error is preserved so logs can still report the root
// cause, and Unwrap is implemented so `errors.Is` / `errors.As`
// climbs the chain naturally.
type Error struct {
	Err error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return "cache error"
	}
	return "cache: " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
