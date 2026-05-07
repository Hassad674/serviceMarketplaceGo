package postgres

// Unit tests for the receipt repository's pure helpers — cursor
// encoding/decoding, snapshot JSON hydration, party authorization.
// SQL-path tests live with the rest of the integration suite gated
// by MARKETPLACE_TEST_DATABASE_URL (see job_credit_repository_test.go);
// here we focus on the bits that do not need a real database.

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/receipt"
)

func TestReceiptCursor_EncodeDecode_RoundTrip(t *testing.T) {
	in := receiptCursor{CreatedAt: time.Now().UTC().Truncate(time.Second), ID: uuid.New()}
	enc, err := encodeReceiptCursor(in)
	require.NoError(t, err)
	assert.NotEmpty(t, enc)

	out, err := decodeReceiptCursor(enc)
	require.NoError(t, err)
	assert.Equal(t, in.ID, out.ID)
	assert.True(t, in.CreatedAt.Equal(out.CreatedAt))
}

func TestReceiptCursor_DecodeInvalid(t *testing.T) {
	_, err := decodeReceiptCursor("not-base64!!!")
	assert.Error(t, err)
}

func TestHydrateSnapshot_NullColumn_LeavesAvailableFalse(t *testing.T) {
	rec := &receipt.Receipt{}
	hydrateSnapshot(rec, sql.NullString{Valid: false})
	assert.False(t, rec.SnapshotAvailable)
	assert.Nil(t, rec.Client)
	assert.Nil(t, rec.Provider)
	assert.Nil(t, rec.Referrer)
}

func TestHydrateSnapshot_EmptyColumn_LeavesAvailableFalse(t *testing.T) {
	rec := &receipt.Receipt{}
	hydrateSnapshot(rec, sql.NullString{Valid: true, String: "  "})
	assert.False(t, rec.SnapshotAvailable)
}

func TestHydrateSnapshot_GarbageJSON_LeavesAvailableFalse(t *testing.T) {
	rec := &receipt.Receipt{}
	hydrateSnapshot(rec, sql.NullString{Valid: true, String: "{not valid"})
	assert.False(t, rec.SnapshotAvailable)
}

func TestHydrateSnapshot_HappyPath_PopulatesAllFields(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	referrerOrg := uuid.New()

	raw := `{
		"client": {
			"organization_id": "` + clientOrg.String() + `",
			"name": "Client SAS",
			"siret": "12345678900012",
			"vat": "FR12345678901",
			"address_line1": "1 rue de Paris",
			"address_line2": "Bât. B",
			"city": "Paris",
			"postal_code": "75001",
			"country": "FR"
		},
		"provider": {
			"organization_id": "` + providerOrg.String() + `",
			"name": "Provider SARL"
		},
		"referrer": {
			"organization_id": "` + referrerOrg.String() + `",
			"name": "Apporteur SAS"
		},
		"referrer_commission_amount_cents": 5000
	}`

	rec := &receipt.Receipt{}
	hydrateSnapshot(rec, sql.NullString{Valid: true, String: raw})

	assert.True(t, rec.SnapshotAvailable)
	require.NotNil(t, rec.Client)
	assert.Equal(t, clientOrg, rec.Client.OrganizationID)
	assert.Equal(t, "Client SAS", rec.Client.Name)
	assert.Equal(t, "12345678900012", rec.Client.SIRET)
	assert.Equal(t, "FR12345678901", rec.Client.VAT)
	assert.Equal(t, "1 rue de Paris", rec.Client.AddressLine1)

	require.NotNil(t, rec.Provider)
	assert.Equal(t, providerOrg, rec.Provider.OrganizationID)

	require.NotNil(t, rec.Referrer)
	assert.Equal(t, referrerOrg, rec.Referrer.OrganizationID)
	assert.Equal(t, int64(5000), rec.ReferrerCommissionAmountCents)
}

func TestHydrateSnapshot_NoReferrerInJSON_LeavesNil(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	raw := `{
		"client": {"organization_id": "` + clientOrg.String() + `", "name": "C"},
		"provider": {"organization_id": "` + providerOrg.String() + `", "name": "P"},
		"referrer": null,
		"referrer_commission_amount_cents": 0
	}`
	rec := &receipt.Receipt{}
	hydrateSnapshot(rec, sql.NullString{Valid: true, String: raw})

	assert.True(t, rec.SnapshotAvailable)
	assert.NotNil(t, rec.Client)
	assert.NotNil(t, rec.Provider)
	assert.Nil(t, rec.Referrer)
	assert.Equal(t, int64(0), rec.ReferrerCommissionAmountCents)
}

func TestPartyJSON_ToDomain_EmptyReturnsNil(t *testing.T) {
	p := partyJSON{}
	assert.Nil(t, p.toDomain())
}

func TestPartyJSON_ToDomain_OrgIDOnlyKept(t *testing.T) {
	id := uuid.New()
	p := partyJSON{OrganizationID: id.String(), Name: "X"}
	out := p.toDomain()
	require.NotNil(t, out)
	assert.Equal(t, id, out.OrganizationID)
	assert.Equal(t, "X", out.Name)
}
