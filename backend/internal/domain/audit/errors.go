package audit

import "errors"

// Domain sentinels. Wrapped at the app layer, matched at the handler
// layer via errors.Is.
var (
	ErrActionRequired = errors.New("audit: action is required")
)
