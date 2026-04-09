package portfolio

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

const (
	MaxItemsPerUser    = 30
	MaxMediaPerItem    = 8
	MaxTitleLength     = 200
	MaxDescriptionLen  = 2000
	MaxLinkURLLength   = 500
)

// MediaType distinguishes images from videos in the gallery.
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

func (t MediaType) IsValid() bool {
	return t == MediaTypeImage || t == MediaTypeVideo
}

// PortfolioItem represents a single portfolio project entry.
type PortfolioItem struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
	LinkURL     string
	Position    int
	Media       []*PortfolioMedia
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PortfolioMedia is a single image or video in a portfolio item gallery.
type PortfolioMedia struct {
	ID              uuid.UUID
	PortfolioItemID uuid.UUID
	MediaURL        string
	MediaType       MediaType
	ThumbnailURL    string // Optional custom thumbnail (videos only)
	Position        int
	CreatedAt       time.Time
}

// firstMedia returns the media at position 0 (or first in list if none match).
func (p *PortfolioItem) firstMedia() *PortfolioMedia {
	for _, m := range p.Media {
		if m.Position == 0 {
			return m
		}
	}
	if len(p.Media) > 0 {
		return p.Media[0]
	}
	return nil
}

// CoverURL returns the best image URL to display as cover.
//
// For images: media_url. For videos: thumbnail_url if set, otherwise empty
// (the frontend falls back to extracting the first video frame).
func (p *PortfolioItem) CoverURL() string {
	cover := p.firstMedia()
	if cover == nil {
		return ""
	}
	if cover.MediaType == MediaTypeImage {
		return cover.MediaURL
	}
	// Video — return custom thumbnail if set, empty otherwise
	return cover.ThumbnailURL
}

// NewItemInput contains all data needed to create a portfolio item.
type NewItemInput struct {
	UserID      uuid.UUID
	Title       string
	Description string
	LinkURL     string
	Position    int
	Media       []NewMediaInput
}

// NewMediaInput describes a single media to attach.
type NewMediaInput struct {
	MediaURL     string
	MediaType    MediaType
	ThumbnailURL string // Optional custom thumbnail (videos only)
	Position     int
}

// NewPortfolioItem creates and validates a new PortfolioItem.
func NewPortfolioItem(in NewItemInput) (*PortfolioItem, error) {
	if in.UserID == uuid.Nil {
		return nil, ErrMissingUserID
	}
	if in.Title == "" {
		return nil, ErrMissingTitle
	}
	if len([]rune(in.Title)) > MaxTitleLength {
		return nil, ErrTitleTooLong
	}
	if len([]rune(in.Description)) > MaxDescriptionLen {
		return nil, ErrDescriptionTooLong
	}
	if err := validateLinkURL(in.LinkURL); err != nil {
		return nil, err
	}
	if in.Position < 0 {
		return nil, ErrInvalidPosition
	}
	if len(in.Media) > MaxMediaPerItem {
		return nil, ErrTooManyMedia
	}

	now := time.Now()
	itemID := uuid.New()

	media := make([]*PortfolioMedia, 0, len(in.Media))
	for _, m := range in.Media {
		if m.MediaURL == "" {
			return nil, ErrMissingMediaURL
		}
		if !m.MediaType.IsValid() {
			return nil, ErrInvalidMediaType
		}
		if m.Position < 0 {
			return nil, ErrInvalidPosition
		}
		media = append(media, &PortfolioMedia{
			ID:              uuid.New(),
			PortfolioItemID: itemID,
			MediaURL:        m.MediaURL,
			MediaType:       m.MediaType,
			ThumbnailURL:    m.ThumbnailURL,
			Position:        m.Position,
			CreatedAt:       now,
		})
	}

	return &PortfolioItem{
		ID:          itemID,
		UserID:      in.UserID,
		Title:       in.Title,
		Description: in.Description,
		LinkURL:     in.LinkURL,
		Position:    in.Position,
		Media:       media,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// UpdateItem updates the item's mutable fields after validation.
func (p *PortfolioItem) UpdateItem(title, description, linkURL string) error {
	if title == "" {
		return ErrMissingTitle
	}
	if len([]rune(title)) > MaxTitleLength {
		return ErrTitleTooLong
	}
	if len([]rune(description)) > MaxDescriptionLen {
		return ErrDescriptionTooLong
	}
	if err := validateLinkURL(linkURL); err != nil {
		return err
	}
	p.Title = title
	p.Description = description
	p.LinkURL = linkURL
	p.UpdatedAt = time.Now()
	return nil
}

// SetMedia replaces all media on this item.
func (p *PortfolioItem) SetMedia(media []*PortfolioMedia) {
	p.Media = media
}

func validateLinkURL(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > MaxLinkURLLength {
		return ErrLinkURLTooLong
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ErrInvalidLinkURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidLinkURL
	}
	if u.Host == "" {
		return ErrInvalidLinkURL
	}
	return nil
}
