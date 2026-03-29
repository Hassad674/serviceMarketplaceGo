package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type BusinessPersonRepository struct {
	db *sql.DB
}

func NewBusinessPersonRepository(db *sql.DB) *BusinessPersonRepository {
	return &BusinessPersonRepository{db: db}
}

func (r *BusinessPersonRepository) Create(ctx context.Context, p *payment.BusinessPerson) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO business_persons (id, user_id, role, first_name, last_name, date_of_birth, email, phone, address, city, postal_code, title, stripe_person_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		p.ID, p.UserID, string(p.Role), p.FirstName, p.LastName, p.DateOfBirth,
		p.Email, p.Phone, p.Address, p.City, p.PostalCode, p.Title, p.StripePersonID,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert business person: %w", err)
	}
	return nil
}

func (r *BusinessPersonRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*payment.BusinessPerson, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, role, first_name, last_name, date_of_birth, email, phone, address, city, postal_code, title, COALESCE(stripe_person_id, ''), created_at, updated_at
		FROM business_persons WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list business persons: %w", err)
	}
	defer rows.Close()

	var persons []*payment.BusinessPerson
	for rows.Next() {
		var p payment.BusinessPerson
		var role string
		if err := rows.Scan(&p.ID, &p.UserID, &role, &p.FirstName, &p.LastName, &p.DateOfBirth, &p.Email, &p.Phone, &p.Address, &p.City, &p.PostalCode, &p.Title, &p.StripePersonID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan business person: %w", err)
		}
		p.Role = payment.PersonRole(role)
		persons = append(persons, &p)
	}
	return persons, nil
}

func (r *BusinessPersonRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `DELETE FROM business_persons WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete business persons: %w", err)
	}
	return nil
}
