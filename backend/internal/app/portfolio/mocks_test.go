package portfolio_test

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/portfolio"
)

type mockPortfolioRepo struct {
	CreateFunc       func(ctx context.Context, item *portfolio.PortfolioItem) error
	GetByIDFunc      func(ctx context.Context, id uuid.UUID) (*portfolio.PortfolioItem, error)
	ListByUserFunc   func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*portfolio.PortfolioItem, string, error)
	UpdateFunc       func(ctx context.Context, item *portfolio.PortfolioItem) error
	DeleteFunc       func(ctx context.Context, id uuid.UUID) error
	CountByUserFunc  func(ctx context.Context, userID uuid.UUID) (int, error)
	ReorderItemsFunc func(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error
	ReplaceMediaFunc func(ctx context.Context, itemID uuid.UUID, media []*portfolio.PortfolioMedia) error
}

func (m *mockPortfolioRepo) Create(ctx context.Context, item *portfolio.PortfolioItem) error {
	return m.CreateFunc(ctx, item)
}

func (m *mockPortfolioRepo) GetByID(ctx context.Context, id uuid.UUID) (*portfolio.PortfolioItem, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *mockPortfolioRepo) ListByUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*portfolio.PortfolioItem, string, error) {
	return m.ListByUserFunc(ctx, userID, cursor, limit)
}

func (m *mockPortfolioRepo) Update(ctx context.Context, item *portfolio.PortfolioItem) error {
	return m.UpdateFunc(ctx, item)
}

func (m *mockPortfolioRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFunc(ctx, id)
}

func (m *mockPortfolioRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return m.CountByUserFunc(ctx, userID)
}

func (m *mockPortfolioRepo) ReorderItems(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error {
	return m.ReorderItemsFunc(ctx, userID, itemIDs)
}

func (m *mockPortfolioRepo) ReplaceMedia(ctx context.Context, itemID uuid.UUID, media []*portfolio.PortfolioMedia) error {
	return m.ReplaceMediaFunc(ctx, itemID, media)
}
