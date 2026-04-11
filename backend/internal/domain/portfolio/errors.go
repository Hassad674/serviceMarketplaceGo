package portfolio

import "errors"

var (
	ErrNotFound              = errors.New("portfolio item not found")
	ErrNotOwner              = errors.New("your organization does not own this portfolio item")
	ErrMissingOrganizationID = errors.New("organization ID is required")
	ErrMissingTitle          = errors.New("title is required")
	ErrTitleTooLong          = errors.New("title exceeds 200 characters")
	ErrDescriptionTooLong    = errors.New("description exceeds 2000 characters")
	ErrLinkURLTooLong        = errors.New("link URL exceeds 500 characters")
	ErrInvalidLinkURL        = errors.New("link URL must be a valid HTTP or HTTPS URL")
	ErrInvalidPosition       = errors.New("position must be non-negative")
	ErrTooManyItems          = errors.New("maximum 30 portfolio items per organization")
	ErrTooManyMedia          = errors.New("maximum 8 media per portfolio item")
	ErrInvalidMediaType      = errors.New("media type must be 'image' or 'video'")
	ErrMissingMediaURL       = errors.New("media URL is required")
)
