package portfolio_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	portfolioapp "marketplace-backend/internal/app/portfolio"
	domain "marketplace-backend/internal/domain/portfolio"
)

func newTestService(repo *mockPortfolioRepo) *portfolioapp.Service {
	return portfolioapp.NewService(portfolioapp.ServiceDeps{
		Portfolios: repo,
	})
}

func existingItem(userID uuid.UUID) *domain.PortfolioItem {
	return &domain.PortfolioItem{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Existing Project",
		Description: "A project",
		LinkURL:     "https://example.com",
		Position:    0,
		Media:       []*domain.PortfolioMedia{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestService_CreateItem_Success(t *testing.T) {
	repo := &mockPortfolioRepo{
		CountByUserFunc: func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil },
		CreateFunc:      func(_ context.Context, _ *domain.PortfolioItem) error { return nil },
	}
	svc := newTestService(repo)

	item, err := svc.CreateItem(context.Background(), portfolioapp.CreateItemInput{
		UserID:   uuid.New(),
		Title:    "My Project",
		Position: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, "My Project", item.Title)
	assert.NotEqual(t, uuid.Nil, item.ID)
}

func TestService_CreateItem_TooManyItems(t *testing.T) {
	repo := &mockPortfolioRepo{
		CountByUserFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
			return domain.MaxItemsPerUser, nil
		},
	}
	svc := newTestService(repo)

	_, err := svc.CreateItem(context.Background(), portfolioapp.CreateItemInput{
		UserID:   uuid.New(),
		Title:    "One Too Many",
		Position: 0,
	})
	assert.ErrorIs(t, err, domain.ErrTooManyItems)
}

func TestService_CreateItem_ValidationError(t *testing.T) {
	repo := &mockPortfolioRepo{
		CountByUserFunc: func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil },
	}
	svc := newTestService(repo)

	_, err := svc.CreateItem(context.Background(), portfolioapp.CreateItemInput{
		UserID:   uuid.New(),
		Title:    "", // empty
		Position: 0,
	})
	assert.ErrorIs(t, err, domain.ErrMissingTitle)
}

func TestService_UpdateItem_Success(t *testing.T) {
	userID := uuid.New()
	item := existingItem(userID)

	repo := &mockPortfolioRepo{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.PortfolioItem, error) {
			return item, nil
		},
		UpdateFunc: func(_ context.Context, _ *domain.PortfolioItem) error { return nil },
	}
	svc := newTestService(repo)

	newTitle := "Updated Title"
	updated, err := svc.UpdateItem(context.Background(), userID, item.ID, portfolioapp.UpdateItemInput{
		Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)
}

func TestService_UpdateItem_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	item := existingItem(ownerID)

	repo := &mockPortfolioRepo{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.PortfolioItem, error) {
			return item, nil
		},
	}
	svc := newTestService(repo)

	_, err := svc.UpdateItem(context.Background(), otherID, item.ID, portfolioapp.UpdateItemInput{})
	assert.ErrorIs(t, err, domain.ErrNotOwner)
}

func TestService_UpdateItem_NotFound(t *testing.T) {
	repo := &mockPortfolioRepo{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.PortfolioItem, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := newTestService(repo)

	_, err := svc.UpdateItem(context.Background(), uuid.New(), uuid.New(), portfolioapp.UpdateItemInput{})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestService_DeleteItem_Success(t *testing.T) {
	userID := uuid.New()
	item := existingItem(userID)

	repo := &mockPortfolioRepo{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.PortfolioItem, error) {
			return item, nil
		},
		DeleteFunc: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(repo)

	err := svc.DeleteItem(context.Background(), userID, item.ID)
	require.NoError(t, err)
}

func TestService_DeleteItem_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	item := existingItem(ownerID)

	repo := &mockPortfolioRepo{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.PortfolioItem, error) {
			return item, nil
		},
	}
	svc := newTestService(repo)

	err := svc.DeleteItem(context.Background(), otherID, item.ID)
	assert.ErrorIs(t, err, domain.ErrNotOwner)
}

func TestService_ListByUser_Success(t *testing.T) {
	items := []*domain.PortfolioItem{existingItem(uuid.New())}

	repo := &mockPortfolioRepo{
		ListByUserFunc: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.PortfolioItem, string, error) {
			return items, "", nil
		},
	}
	svc := newTestService(repo)

	result, cursor, err := svc.ListByUser(context.Background(), uuid.New(), "", 20)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Empty(t, cursor)
}

func TestService_ListByUser_DefaultLimit(t *testing.T) {
	var capturedLimit int
	repo := &mockPortfolioRepo{
		ListByUserFunc: func(_ context.Context, _ uuid.UUID, _ string, limit int) ([]*domain.PortfolioItem, string, error) {
			capturedLimit = limit
			return nil, "", nil
		},
	}
	svc := newTestService(repo)

	_, _, _ = svc.ListByUser(context.Background(), uuid.New(), "", 0)
	assert.Equal(t, 20, capturedLimit)
}

func TestService_ReorderItems_Success(t *testing.T) {
	repo := &mockPortfolioRepo{
		ReorderItemsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
			return nil
		},
	}
	svc := newTestService(repo)

	err := svc.ReorderItems(context.Background(), uuid.New(), []uuid.UUID{uuid.New(), uuid.New()})
	require.NoError(t, err)
}

func TestService_UpdateItem_WithMedia(t *testing.T) {
	userID := uuid.New()
	item := existingItem(userID)

	repo := &mockPortfolioRepo{
		GetByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.PortfolioItem, error) {
			return item, nil
		},
		UpdateFunc:       func(_ context.Context, _ *domain.PortfolioItem) error { return nil },
		ReplaceMediaFunc: func(_ context.Context, _ uuid.UUID, _ []*domain.PortfolioMedia) error { return nil },
	}
	svc := newTestService(repo)

	updated, err := svc.UpdateItem(context.Background(), userID, item.ID, portfolioapp.UpdateItemInput{
		Media: []portfolioapp.MediaInput{
			{MediaURL: "https://r2.example.com/img.jpg", MediaType: "image", Position: 0},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, updated)
}
