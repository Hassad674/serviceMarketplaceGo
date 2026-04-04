package payment

// AccountRequirements holds the full set of Stripe account requirements,
// categorised by urgency level.
type AccountRequirements struct {
	CurrentlyDue        []string
	EventuallyDue       []string
	PastDue             []string
	PendingVerification []string
	CurrentDeadline     int64
	Errors              []RequirementError
}

// RequirementError holds a Stripe requirement validation error.
type RequirementError struct {
	Code        string // e.g. "invalid_phone_number"
	Reason      string // human-readable message from Stripe
	Requirement string // e.g. "company.phone"
}
