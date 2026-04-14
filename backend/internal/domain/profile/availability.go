package profile

// AvailabilityStatus is the frozen enum of "can this org take on new
// work right now" states. Stored on the profiles row as TEXT, driven
// end-to-end by these three values — any write with something else
// is rejected at the domain level before it reaches the repository.
type AvailabilityStatus string

const (
	// AvailabilityNow is the default: the org is accepting work
	// immediately and will respond to messages / proposals.
	AvailabilityNow AvailabilityStatus = "available_now"

	// AvailabilitySoon signals booked-up-but-will-be-free — the org
	// shows up in searches with a warning badge so prospective
	// clients know to plan ahead.
	AvailabilitySoon AvailabilityStatus = "available_soon"

	// AvailabilityNot hides the org from availability-filtered
	// searches. Profile is still public, just flagged as off.
	AvailabilityNot AvailabilityStatus = "not_available"
)

// IsValid reports whether a is one of the three frozen enum values.
// An empty string (zero value) is NOT valid — callers building an
// AvailabilityStatus from raw input must default to AvailabilityNow
// or fail explicitly.
func (a AvailabilityStatus) IsValid() bool {
	switch a {
	case AvailabilityNow, AvailabilitySoon, AvailabilityNot:
		return true
	}
	return false
}

// ParseAvailabilityStatus converts a raw string to the typed enum
// and returns ErrInvalidAvailabilityStatus when the value is not one
// of the three accepted entries. Empty input is rejected — the
// caller is expected to have filled a default before parsing.
func ParseAvailabilityStatus(raw string) (AvailabilityStatus, error) {
	candidate := AvailabilityStatus(raw)
	if !candidate.IsValid() {
		return "", ErrInvalidAvailabilityStatus
	}
	return candidate, nil
}
