package receipt

import "errors"

// ErrNotFound is returned when no receipt matches the given id.
var ErrNotFound = errors.New("receipt not found")

// ErrForbidden is returned when the caller's organization is not a
// party (client / provider / referrer) on the receipt. The handler
// maps this to 403.
//
// Why a separate error code (and not merging into ErrNotFound to avoid
// leaking existence): the receipt id is opaque to clients — they only
// ever obtain it through the list endpoint, which already filters by
// org membership. A direct GET for an id the user has never seen
// returning ErrNotFound is the same observable behaviour as ErrForbidden
// but the audit trail is clearer if the layers can distinguish the two.
var ErrForbidden = errors.New("receipt access forbidden")
