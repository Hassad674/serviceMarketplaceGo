package proposal

import "errors"

var (
	ErrProposalNotFound       = errors.New("proposal not found")
	ErrInvalidStatus          = errors.New("invalid proposal status for this operation")
	ErrEmptyTitle             = errors.New("proposal title cannot be empty")
	ErrEmptyDescription       = errors.New("proposal description cannot be empty")
	ErrInvalidAmount          = errors.New("proposal amount must be greater than zero")
	ErrSameUser               = errors.New("cannot create proposal with yourself")
	ErrNotAuthorized          = errors.New("not authorized to perform this action")
	ErrInvalidRoleCombination = errors.New("invalid role combination for proposal")
	ErrCannotModify           = errors.New("only the recipient can modify a pending proposal")
	ErrAlreadyAccepted        = errors.New("proposal is already accepted")
	ErrAlreadyDeclined        = errors.New("proposal is already declined")
)
