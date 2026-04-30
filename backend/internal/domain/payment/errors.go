package payment

import "errors"

var (
	ErrNotFound             = errors.New("payment info not found")
	ErrFirstNameRequired    = errors.New("first name is required")
	ErrLastNameRequired     = errors.New("last name is required")
	ErrDateOfBirthRequired  = errors.New("date of birth is required")
	ErrNationalityRequired  = errors.New("nationality is required")
	ErrAddressRequired      = errors.New("address is required")
	ErrCityRequired         = errors.New("city is required")
	ErrPostalCodeRequired   = errors.New("postal code is required")
	ErrAccountHolderRequired = errors.New("account holder is required")
	ErrBankDetailsRequired  = errors.New("IBAN or account number + routing number is required")
	ErrBusinessNameRequired = errors.New("business name is required when is_business is true")
	ErrTaxIDRequired        = errors.New("tax ID is required when is_business is true")

	// Payment record errors
	ErrPaymentRecordNotFound    = errors.New("payment record not found")
	ErrPaymentAlreadyExists     = errors.New("payment already exists for this proposal")
	ErrPaymentNotPending        = errors.New("payment is not in pending state")
	ErrPaymentNotSucceeded      = errors.New("payment has not succeeded")
	// ErrPaymentNotConfirmed fires when MarkPaymentSucceeded is called for a
	// payment record whose Stripe PaymentIntent is NOT in the `succeeded`
	// state. Closes SEC-02: a client cannot trigger /confirm-payment to flip
	// the record locally without an actual Stripe charge having cleared.
	// Mapped to HTTP 402 Payment Required at the handler boundary.
	ErrPaymentNotConfirmed      = errors.New("payment intent is not in succeeded state")
	ErrTransferAlreadyDone      = errors.New("transfer already completed")
	ErrTransferNotRetriable     = errors.New("transfer not retriable")
	ErrStripeAccountNotFound    = errors.New("provider has no Stripe connected account")
	ErrStripeAccountNotVerified = errors.New("provider Stripe account is not verified")
	// ErrProviderPayoutsDisabled fires when the provider has a Stripe Connect
	// account on file BUT payouts_enabled=false (KYC pending, capability
	// disabled, …). Distinct from ErrStripeAccountNotFound because the user
	// already started Stripe onboarding — the UX is "finish onboarding",
	// not "set up payments". Mapped to HTTP 412 (Precondition Failed) at
	// the handler boundary.
	ErrProviderPayoutsDisabled = errors.New("provider Stripe payouts are not enabled")

	// ErrInvalidStateTransition fires when a payment_record domain method is
	// invoked from a state that disallows it (e.g. MarkRefunded on a record
	// already Failed). Closes BUG-02: previously MarkFailed / MarkRefunded /
	// ApplyDisputeResolution had no source-state guards, so a webhook replay
	// or a buggy caller could overwrite ProviderPayout to 0 on an already-
	// transferred record and "lose" the provider's money. The wrapper
	// StateTransitionError carries the expected vs actual states for
	// observability — callers can inspect them via errors.As.
	ErrInvalidStateTransition = errors.New("invalid payment_record state transition")

	// Identity document errors
	ErrInvalidDocumentCategory = errors.New("invalid document category")
	ErrInvalidDocumentType     = errors.New("invalid document type")
	ErrInvalidDocumentSide     = errors.New("invalid document side for this document type")
	ErrDocumentFileKeyRequired = errors.New("file key is required")
	ErrDocumentNotFound        = errors.New("identity document not found")
	ErrDocumentNotPending      = errors.New("document is not in pending state")

	// Business person errors
	ErrInvalidPersonRole  = errors.New("invalid person role")
	ErrPersonNameRequired = errors.New("person first and last name are required")

	// Country errors
	ErrCountryRequired = errors.New("activity country is required")
)
