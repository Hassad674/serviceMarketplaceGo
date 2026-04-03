package payment

// AccountRequirements holds the full set of Stripe account requirements,
// categorised by urgency level.
type AccountRequirements struct {
	CurrentlyDue        []string
	EventuallyDue       []string
	PastDue             []string
	PendingVerification []string
	CurrentDeadline     int64
}
