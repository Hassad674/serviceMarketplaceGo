package portfolio

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput() NewItemInput {
	return NewItemInput{
		OrganizationID: uuid.New(),
		Title:       "E-commerce Redesign",
		Description: "Full redesign for a fashion brand.",
		LinkURL:     "https://example.com/project",
		Position:    0,
	}
}

func TestNewPortfolioItem_Valid(t *testing.T) {
	in := validInput()
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	assert.Equal(t, in.Title, item.Title)
	assert.Equal(t, in.Description, item.Description)
	assert.Equal(t, in.LinkURL, item.LinkURL)
	assert.Equal(t, in.OrganizationID, item.OrganizationID)
	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.Empty(t, item.Media)
}

func TestNewPortfolioItem_ValidWithMedia(t *testing.T) {
	in := validInput()
	in.Media = []NewMediaInput{
		{MediaURL: "https://r2.example.com/img1.jpg", MediaType: MediaTypeImage, Position: 0},
		{MediaURL: "https://r2.example.com/vid1.mp4", MediaType: MediaTypeVideo, Position: 1},
	}
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	assert.Len(t, item.Media, 2)
	assert.Equal(t, MediaTypeImage, item.Media[0].MediaType)
	assert.Equal(t, MediaTypeVideo, item.Media[1].MediaType)
	assert.Equal(t, item.ID, item.Media[0].PortfolioItemID)
}

func TestNewPortfolioItem_MissingOrganizationID(t *testing.T) {
	in := validInput()
	in.OrganizationID = uuid.Nil
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrMissingOrganizationID)
}

func TestNewPortfolioItem_MissingTitle(t *testing.T) {
	in := validInput()
	in.Title = ""
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrMissingTitle)
}

func TestNewPortfolioItem_TitleTooLong(t *testing.T) {
	in := validInput()
	in.Title = strings.Repeat("a", MaxTitleLength+1)
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrTitleTooLong)
}

func TestNewPortfolioItem_DescriptionTooLong(t *testing.T) {
	in := validInput()
	in.Description = strings.Repeat("x", MaxDescriptionLen+1)
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

func TestNewPortfolioItem_InvalidLinkURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"no scheme", "example.com"},
		{"ftp scheme", "ftp://example.com"},
		{"javascript", "javascript:alert(1)"},
		{"too long", "https://example.com/" + strings.Repeat("a", 500)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			in.LinkURL = tc.url
			_, err := NewPortfolioItem(in)
			assert.Error(t, err)
		})
	}
}

func TestNewPortfolioItem_EmptyLinkURL_Allowed(t *testing.T) {
	in := validInput()
	in.LinkURL = ""
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	assert.Empty(t, item.LinkURL)
}

func TestNewPortfolioItem_NegativePosition(t *testing.T) {
	in := validInput()
	in.Position = -1
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrInvalidPosition)
}

func TestNewPortfolioItem_TooManyMedia(t *testing.T) {
	in := validInput()
	for i := 0; i <= MaxMediaPerItem; i++ {
		in.Media = append(in.Media, NewMediaInput{
			MediaURL:  "https://r2.example.com/img.jpg",
			MediaType: MediaTypeImage,
			Position:  i,
		})
	}
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrTooManyMedia)
}

func TestNewPortfolioItem_InvalidMediaType(t *testing.T) {
	in := validInput()
	in.Media = []NewMediaInput{
		{MediaURL: "https://r2.example.com/file.pdf", MediaType: "pdf", Position: 0},
	}
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrInvalidMediaType)
}

func TestNewPortfolioItem_MissingMediaURL(t *testing.T) {
	in := validInput()
	in.Media = []NewMediaInput{
		{MediaURL: "", MediaType: MediaTypeImage, Position: 0},
	}
	_, err := NewPortfolioItem(in)
	assert.ErrorIs(t, err, ErrMissingMediaURL)
}

func TestCoverURL_ImageCover(t *testing.T) {
	in := validInput()
	in.Media = []NewMediaInput{
		{MediaURL: "https://r2.example.com/img1.jpg", MediaType: MediaTypeImage, Position: 1},
		{MediaURL: "https://r2.example.com/cover.jpg", MediaType: MediaTypeImage, Position: 0},
	}
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	assert.Equal(t, "https://r2.example.com/cover.jpg", item.CoverURL())
}

func TestCoverURL_VideoCoverWithThumbnail(t *testing.T) {
	in := validInput()
	in.Media = []NewMediaInput{
		{
			MediaURL:     "https://r2.example.com/video.mp4",
			MediaType:    MediaTypeVideo,
			ThumbnailURL: "https://r2.example.com/custom-thumb.jpg",
			Position:     0,
		},
	}
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	assert.Equal(t, "https://r2.example.com/custom-thumb.jpg", item.CoverURL())
}

func TestCoverURL_VideoCoverWithoutThumbnail(t *testing.T) {
	in := validInput()
	in.Media = []NewMediaInput{
		{MediaURL: "https://r2.example.com/video.mp4", MediaType: MediaTypeVideo, Position: 0},
	}
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	// Empty: frontend falls back to first-frame extraction
	assert.Empty(t, item.CoverURL())
}

func TestCoverURL_NoMedia(t *testing.T) {
	in := validInput()
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)
	assert.Empty(t, item.CoverURL())
}

func TestUpdateItem_Valid(t *testing.T) {
	in := validInput()
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)

	err = item.UpdateItem("New Title", "New desc", "https://new.example.com")
	require.NoError(t, err)
	assert.Equal(t, "New Title", item.Title)
	assert.Equal(t, "New desc", item.Description)
	assert.Equal(t, "https://new.example.com", item.LinkURL)
}

func TestUpdateItem_EmptyTitle(t *testing.T) {
	in := validInput()
	item, err := NewPortfolioItem(in)
	require.NoError(t, err)

	err = item.UpdateItem("", "desc", "")
	assert.ErrorIs(t, err, ErrMissingTitle)
}

func TestMediaType_IsValid(t *testing.T) {
	assert.True(t, MediaTypeImage.IsValid())
	assert.True(t, MediaTypeVideo.IsValid())
	assert.False(t, MediaType("pdf").IsValid())
	assert.False(t, MediaType("").IsValid())
}
