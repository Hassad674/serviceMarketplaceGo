package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentInfoRepository struct {
	db *sql.DB
}

func NewPaymentInfoRepository(db *sql.DB) *PaymentInfoRepository {
	return &PaymentInfoRepository{db: db}
}

func (r *PaymentInfoRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, user_id,
			first_name, last_name, date_of_birth, nationality,
			address, city, postal_code,
			is_business, business_name, business_address, business_city,
			business_postal_code, business_country, tax_id, vat_number, role_in_company,
			iban, bic, account_number, routing_number, account_holder, bank_country,
			stripe_account_id, stripe_verified,
			created_at, updated_at
		FROM payment_info
		WHERE user_id = $1`

	p := &payment.PaymentInfo{}
	var (
		businessName    sql.NullString
		businessAddr    sql.NullString
		businessCity    sql.NullString
		businessPostal  sql.NullString
		businessCountry sql.NullString
		taxID           sql.NullString
		vatNumber       sql.NullString
		roleInCompany   sql.NullString
		iban            sql.NullString
		bic             sql.NullString
		accountNumber   sql.NullString
		routingNumber   sql.NullString
		bankCountry     sql.NullString
		stripeAccID     sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&p.ID, &p.UserID,
		&p.FirstName, &p.LastName, &p.DateOfBirth, &p.Nationality,
		&p.Address, &p.City, &p.PostalCode,
		&p.IsBusiness, &businessName, &businessAddr, &businessCity,
		&businessPostal, &businessCountry, &taxID, &vatNumber, &roleInCompany,
		&iban, &bic, &accountNumber, &routingNumber, &p.AccountHolder, &bankCountry,
		&stripeAccID, &p.StripeVerified,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, payment.ErrNotFound
		}
		return nil, fmt.Errorf("get payment info: %w", err)
	}

	p.BusinessName = businessName.String
	p.BusinessAddress = businessAddr.String
	p.BusinessCity = businessCity.String
	p.BusinessPostalCode = businessPostal.String
	p.BusinessCountry = businessCountry.String
	p.TaxID = taxID.String
	p.VATNumber = vatNumber.String
	p.RoleInCompany = roleInCompany.String
	p.IBAN = iban.String
	p.BIC = bic.String
	p.AccountNumber = accountNumber.String
	p.RoutingNumber = routingNumber.String
	p.BankCountry = bankCountry.String
	p.StripeAccountID = stripeAccID.String

	return p, nil
}

func (r *PaymentInfoRepository) Upsert(ctx context.Context, info *payment.PaymentInfo) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO payment_info (
			id, user_id,
			first_name, last_name, date_of_birth, nationality,
			address, city, postal_code,
			is_business, business_name, business_address, business_city,
			business_postal_code, business_country, tax_id, vat_number, role_in_company,
			iban, bic, account_number, routing_number, account_holder, bank_country,
			created_at, updated_at
		) VALUES (
			$1, $2,
			$3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26
		)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			date_of_birth = EXCLUDED.date_of_birth,
			nationality = EXCLUDED.nationality,
			address = EXCLUDED.address,
			city = EXCLUDED.city,
			postal_code = EXCLUDED.postal_code,
			is_business = EXCLUDED.is_business,
			business_name = EXCLUDED.business_name,
			business_address = EXCLUDED.business_address,
			business_city = EXCLUDED.business_city,
			business_postal_code = EXCLUDED.business_postal_code,
			business_country = EXCLUDED.business_country,
			tax_id = EXCLUDED.tax_id,
			vat_number = EXCLUDED.vat_number,
			role_in_company = EXCLUDED.role_in_company,
			iban = EXCLUDED.iban,
			bic = EXCLUDED.bic,
			account_number = EXCLUDED.account_number,
			routing_number = EXCLUDED.routing_number,
			account_holder = EXCLUDED.account_holder,
			bank_country = EXCLUDED.bank_country`

	_, err := r.db.ExecContext(ctx, query,
		info.ID, info.UserID,
		info.FirstName, info.LastName, info.DateOfBirth, info.Nationality,
		info.Address, info.City, info.PostalCode,
		info.IsBusiness, nullString(info.BusinessName), nullString(info.BusinessAddress),
		nullString(info.BusinessCity), nullString(info.BusinessPostalCode),
		nullString(info.BusinessCountry), nullString(info.TaxID),
		nullString(info.VATNumber), nullString(info.RoleInCompany),
		nullString(info.IBAN), nullString(info.BIC),
		nullString(info.AccountNumber), nullString(info.RoutingNumber),
		info.AccountHolder, nullString(info.BankCountry),
		info.CreatedAt, info.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert payment info: %w", err)
	}

	return nil
}

func (r *PaymentInfoRepository) UpdateStripeFields(ctx context.Context, userID uuid.UUID, stripeAccountID string, stripeVerified bool) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE payment_info SET stripe_account_id = $1, stripe_verified = $2 WHERE user_id = $3`,
		stripeAccountID, stripeVerified, userID)
	if err != nil {
		return fmt.Errorf("update stripe fields: %w", err)
	}
	return nil
}

func (r *PaymentInfoRepository) GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p := &payment.PaymentInfo{}
	var (
		businessName, businessAddr, businessCity, businessPostal, businessCountry sql.NullString
		taxID, vatNumber, roleInCompany                                          sql.NullString
		iban, bic, accountNumber, routingNumber, bankCountry, stripeAccID        sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id,
			first_name, last_name, date_of_birth, nationality,
			address, city, postal_code,
			is_business, business_name, business_address, business_city,
			business_postal_code, business_country, tax_id, vat_number, role_in_company,
			iban, bic, account_number, routing_number, account_holder, bank_country,
			stripe_account_id, stripe_verified, created_at, updated_at
		FROM payment_info WHERE stripe_account_id = $1`, stripeAccountID).Scan(
		&p.ID, &p.UserID,
		&p.FirstName, &p.LastName, &p.DateOfBirth, &p.Nationality,
		&p.Address, &p.City, &p.PostalCode,
		&p.IsBusiness, &businessName, &businessAddr, &businessCity,
		&businessPostal, &businessCountry, &taxID, &vatNumber, &roleInCompany,
		&iban, &bic, &accountNumber, &routingNumber, &p.AccountHolder, &bankCountry,
		&stripeAccID, &p.StripeVerified, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, payment.ErrNotFound
		}
		return nil, fmt.Errorf("get payment info by stripe account: %w", err)
	}

	p.BusinessName = businessName.String
	p.BusinessAddress = businessAddr.String
	p.BusinessCity = businessCity.String
	p.BusinessPostalCode = businessPostal.String
	p.BusinessCountry = businessCountry.String
	p.TaxID = taxID.String
	p.VATNumber = vatNumber.String
	p.RoleInCompany = roleInCompany.String
	p.IBAN = iban.String
	p.BIC = bic.String
	p.AccountNumber = accountNumber.String
	p.RoutingNumber = routingNumber.String
	p.BankCountry = bankCountry.String
	p.StripeAccountID = stripeAccID.String

	return p, nil
}

// nullString converts an empty string to a sql.NullString with Valid=false.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
