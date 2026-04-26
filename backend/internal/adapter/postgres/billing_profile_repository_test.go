package postgres_test

// Integration tests for BillingProfileRepository (migration 121 schema).
// Gated behind MARKETPLACE_TEST_DATABASE_URL via the testDB helper in
// job_credit_repository_test.go — auto-skip when unset.
//
// Run against the local feature DB:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_feat_invoicing?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestBillingProfileRepository -count=1

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/invoicing"
)

// invoicingTestOrg seeds a user + organization the same way subTestUser
// does in subscription_repository_test.go. Returns the org id which is
// what billing_profile / invoice key on. Cleanup is registered via
// t.Cleanup so reruns stay isolated.
func invoicingTestOrg(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	orgID := uuid.New()
	email := userID.String()[:8] + "@inv.local"
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'x', 'Inv', 'Test', 'Inv Test', 'provider')`,
		userID, email,
	)
	require.NoError(t, err, "insert user")
	_, err = db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'provider_personal', 'Inv Test Org')`,
		orgID, userID,
	)
	require.NoError(t, err, "insert organization")
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID)
	require.NoError(t, err, "link user to organization")

	t.Cleanup(func() {
		// Order matters: invoice_item -> invoice -> billing_profile -> users -> organizations.
		_, _ = db.Exec(`
			DELETE FROM invoice_item
			WHERE invoice_id IN (SELECT id FROM invoice WHERE recipient_organization_id = $1)`,
			orgID)
		_, _ = db.Exec(`DELETE FROM credit_note WHERE recipient_organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM invoice WHERE recipient_organization_id = $1`, orgID)
		_, _ = db.Exec(`DELETE FROM billing_profile WHERE organization_id = $1`, orgID)
		_, _ = db.Exec(`UPDATE users SET organization_id = NULL WHERE id = $1`, userID)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, userID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE id = $1`, orgID)
	})
	return orgID
}

func makeBillingProfile(orgID uuid.UUID) *invoicing.BillingProfile {
	now := time.Now().UTC().Truncate(time.Second)
	return &invoicing.BillingProfile{
		OrganizationID: orgID,
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "ACME SARL",
		TradingName:    "ACME",
		LegalForm:      "SARL",
		TaxID:          "12345678901234",
		VATNumber:      "FR12345678901",
		AddressLine1:   "1 rue du test",
		AddressLine2:   "BP 42",
		PostalCode:     "75001",
		City:           "Paris",
		Country:        "FR",
		InvoicingEmail: "billing@acme.test",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestBillingProfileRepository_UpsertInsertAndFind(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewBillingProfileRepository(db)
	orgID := invoicingTestOrg(t, db)

	want := makeBillingProfile(orgID)
	require.NoError(t, repo.Upsert(context.Background(), want))

	got, err := repo.FindByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, want.OrganizationID, got.OrganizationID)
	assert.Equal(t, want.ProfileType, got.ProfileType)
	assert.Equal(t, want.LegalName, got.LegalName)
	assert.Equal(t, want.TaxID, got.TaxID)
	assert.Equal(t, want.VATNumber, got.VATNumber)
	assert.Equal(t, want.AddressLine1, got.AddressLine1)
	assert.Equal(t, want.City, got.City)
	assert.Equal(t, want.Country, got.Country)
	assert.Equal(t, want.InvoicingEmail, got.InvoicingEmail)
	assert.Nil(t, got.VATValidatedAt)
	assert.Nil(t, got.SyncedFromKYCAt)
	assert.Nil(t, got.VATValidationPayload)
}

func TestBillingProfileRepository_UpsertUpdatesMutableFields(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewBillingProfileRepository(db)
	orgID := invoicingTestOrg(t, db)

	first := makeBillingProfile(orgID)
	require.NoError(t, repo.Upsert(context.Background(), first))

	createdAt, err := repo.FindByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	originalCreatedAt := createdAt.CreatedAt

	// Same org, mutated fields. We expect the row to be updated in place.
	updated := makeBillingProfile(orgID)
	updated.LegalName = "ACME 2 SAS"
	updated.LegalForm = "SAS"
	updated.City = "Lyon"
	updated.PostalCode = "69000"
	updated.InvoicingEmail = "billing-v2@acme.test"

	// Sleep long enough for now() in the SQL trigger to differ.
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, repo.Upsert(context.Background(), updated))

	got, err := repo.FindByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "ACME 2 SAS", got.LegalName)
	assert.Equal(t, "SAS", got.LegalForm)
	assert.Equal(t, "Lyon", got.City)
	assert.Equal(t, "69000", got.PostalCode)
	assert.Equal(t, "billing-v2@acme.test", got.InvoicingEmail)
	assert.True(t, got.UpdatedAt.After(originalCreatedAt),
		"updated_at must move forward on Upsert update")
	assert.WithinDuration(t, originalCreatedAt, got.CreatedAt, time.Second,
		"created_at must NOT change on Upsert update")
}

func TestBillingProfileRepository_FindByOrganizationNotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewBillingProfileRepository(db)

	_, err := repo.FindByOrganization(context.Background(), uuid.New())

	assert.ErrorIs(t, err, invoicing.ErrNotFound)
}

func TestBillingProfileRepository_VATValidationRoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewBillingProfileRepository(db)
	orgID := invoicingTestOrg(t, db)

	validatedAt := time.Now().UTC().Truncate(time.Second)
	payload, err := json.Marshal(map[string]any{
		"valid":        true,
		"countryCode":  "FR",
		"vatNumber":    "12345678901",
		"name":         "ACME SARL",
		"address":      "1 rue du test, 75001 Paris",
		"requestDate":  validatedAt.Format(time.RFC3339),
	})
	require.NoError(t, err)

	syncedAt := validatedAt.Add(-1 * time.Hour)
	p := makeBillingProfile(orgID)
	p.VATValidatedAt = &validatedAt
	p.VATValidationPayload = payload
	p.SyncedFromKYCAt = &syncedAt

	require.NoError(t, repo.Upsert(context.Background(), p))

	got, err := repo.FindByOrganization(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got.VATValidatedAt)
	assert.WithinDuration(t, validatedAt, *got.VATValidatedAt, time.Second)
	require.NotNil(t, got.SyncedFromKYCAt)
	assert.WithinDuration(t, syncedAt, *got.SyncedFromKYCAt, time.Second)

	// JSONB round-trip — Postgres normalises whitespace, so we compare
	// the parsed map rather than the raw bytes.
	require.NotNil(t, got.VATValidationPayload)
	var roundTripped map[string]any
	require.NoError(t, json.Unmarshal(got.VATValidationPayload, &roundTripped))
	assert.Equal(t, true, roundTripped["valid"])
	assert.Equal(t, "FR", roundTripped["countryCode"])
	assert.Equal(t, "ACME SARL", roundTripped["name"])
}
