package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/receipt"
)

// ReceiptRepository is the postgres implementation of the receipt
// read port. Receipts are a presentation projection over the
// payment_records table — there is no dedicated `receipts` table.
//
// Access control:
//   - The SQL filter restricts every read to rows where the caller's
//     org appears as client (organization_id) or provider
//     (provider_organization_id) of the underlying payment_record.
//   - Referrer-side access is layered on top via the
//     billing_snapshot.referrer.organization_id JSONB filter — kept
//     loose here because referrer attribution is itself snapshotted,
//     so the JSONB column is the single source of truth for "is this
//     org a party on this receipt".
//   - The handler layer applies a final domain.IsParty check (defense
//     in depth).
type ReceiptRepository struct {
	db *sql.DB
}

// NewReceiptRepository wires the receipt repository over the shared
// SQL pool. No tx runner — the receipt read paths are pure SELECTs
// and do not need RLS context (the SQL filter already pins the
// caller's org explicitly via the WHERE clause).
func NewReceiptRepository(db *sql.DB) *ReceiptRepository {
	return &ReceiptRepository{db: db}
}

// receiptCursor is the opaque pagination cursor encoded into the
// next_cursor field of the list response. We pin (created_at, id)
// so two rows with the same created_at can never be returned twice
// or skipped between pages.
type receiptCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

func encodeReceiptCursor(c receiptCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeReceiptCursor(s string) (receiptCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return receiptCursor{}, fmt.Errorf("decode cursor: %w", err)
	}
	var c receiptCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return receiptCursor{}, fmt.Errorf("unmarshal cursor: %w", err)
	}
	return c, nil
}

// ListForOrganization returns receipts where the caller's org is the
// client, the provider, or the referrer (extracted from the JSONB
// snapshot). Ordered by created_at DESC, paginated via opaque cursor.
func (r *ReceiptRepository) ListForOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*receipt.Receipt, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	const baseSelect = `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
			COALESCE(proposal_amount, 0) AS amount_cents,
			currency, created_at, billing_snapshot
		FROM payment_records
		WHERE (
			organization_id = $1
			OR provider_organization_id = $1
			OR (billing_snapshot IS NOT NULL
				AND (billing_snapshot->'referrer'->>'organization_id')::uuid = $1)
		)`

	var rows *sql.Rows
	var err error
	if cursor == "" {
		query := baseSelect + ` ORDER BY created_at DESC, id DESC LIMIT $2`
		rows, err = r.db.QueryContext(ctx, query, orgID, limit+1)
	} else {
		c, decodeErr := decodeReceiptCursor(cursor)
		if decodeErr != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", decodeErr)
		}
		query := baseSelect + ` AND (created_at, id) < ($2, $3) ORDER BY created_at DESC, id DESC LIMIT $4`
		rows, err = r.db.QueryContext(ctx, query, orgID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list receipts: %w", err)
	}
	defer rows.Close()

	out := make([]*receipt.Receipt, 0, limit)
	for rows.Next() {
		rec, scanErr := scanReceiptRow(rows)
		if scanErr != nil {
			return nil, "", scanErr
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate receipts: %w", err)
	}

	nextCursor := ""
	if len(out) > limit {
		last := out[limit-1]
		out = out[:limit]
		c, encErr := encodeReceiptCursor(receiptCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		if encErr != nil {
			return nil, "", encErr
		}
		nextCursor = c
	}
	return out, nextCursor, nil
}

// GetForOrganization returns one receipt by id. Splits into two
// errors so the audit layer can distinguish "row does not exist"
// from "row exists but caller is not a party" — both surface as the
// same observable behaviour to the client (404), but the difference
// is logged.
func (r *ReceiptRepository) GetForOrganization(ctx context.Context, receiptID, orgID uuid.UUID) (*receipt.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
		SELECT id, proposal_id, milestone_id, client_id, provider_id,
			COALESCE(proposal_amount, 0) AS amount_cents,
			currency, created_at, billing_snapshot,
			organization_id, provider_organization_id
		FROM payment_records
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, receiptID)
	rec, ownerOrg, providerOrg, err := scanReceiptRowWithOrgs(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, receipt.ErrNotFound
		}
		return nil, err
	}

	// Authorization check: caller must be one of client (organization_id),
	// provider (provider_organization_id), or referrer (snapshot).
	switch {
	case ownerOrg == orgID:
		return rec, nil
	case providerOrg == orgID:
		return rec, nil
	case rec.Referrer != nil && rec.Referrer.OrganizationID == orgID:
		return rec, nil
	}
	return nil, receipt.ErrForbidden
}

// scanReceiptRow scans the columns produced by ListForOrganization.
// Column order: id, proposal_id, milestone_id, client_id, provider_id,
// amount_cents, currency, created_at, billing_snapshot.
func scanReceiptRow(row *sql.Rows) (*receipt.Receipt, error) {
	var (
		id, proposalID, clientID, providerID uuid.UUID
		milestoneID                          uuid.NullUUID
		amountCents                          int64
		currency                             string
		createdAt                            time.Time
		snapshotRaw                          sql.NullString
	)
	if err := row.Scan(&id, &proposalID, &milestoneID, &clientID, &providerID,
		&amountCents, &currency, &createdAt, &snapshotRaw); err != nil {
		return nil, fmt.Errorf("scan receipt: %w", err)
	}

	out := &receipt.Receipt{
		ID:              id,
		PaymentRecordID: id,
		ProposalID:      proposalID,
		AmountCents:     amountCents,
		Currency:        strings.ToUpper(currency),
		CreatedAt:       createdAt,
	}
	if milestoneID.Valid {
		out.MilestoneID = milestoneID.UUID
	}
	hydrateSnapshot(out, snapshotRaw)
	return out, nil
}

// scanReceiptRowWithOrgs is the GET-path variant: alongside the
// receipt fields it also returns the row's owning organization_id +
// provider_organization_id columns so the caller can run the
// authorization check without re-querying.
func scanReceiptRowWithOrgs(row *sql.Row) (*receipt.Receipt, uuid.UUID, uuid.UUID, error) {
	var (
		id, proposalID, clientID, providerID uuid.UUID
		milestoneID                          uuid.NullUUID
		amountCents                          int64
		currency                             string
		createdAt                            time.Time
		snapshotRaw                          sql.NullString
		ownerOrg, providerOrg                uuid.NullUUID
	)
	if err := row.Scan(&id, &proposalID, &milestoneID, &clientID, &providerID,
		&amountCents, &currency, &createdAt, &snapshotRaw,
		&ownerOrg, &providerOrg); err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}

	rec := &receipt.Receipt{
		ID:              id,
		PaymentRecordID: id,
		ProposalID:      proposalID,
		AmountCents:     amountCents,
		Currency:        strings.ToUpper(currency),
		CreatedAt:       createdAt,
	}
	if milestoneID.Valid {
		rec.MilestoneID = milestoneID.UUID
	}
	hydrateSnapshot(rec, snapshotRaw)

	owner, provider := uuid.Nil, uuid.Nil
	if ownerOrg.Valid {
		owner = ownerOrg.UUID
	}
	if providerOrg.Valid {
		provider = providerOrg.UUID
	}
	return rec, owner, provider, nil
}

// hydrateSnapshot deserialises the JSONB billing_snapshot column into
// the typed PartyBilling fields on the receipt. A NULL or unparseable
// snapshot leaves SnapshotAvailable=false so the UI can render the
// "data unavailable" marker for legacy rows.
func hydrateSnapshot(out *receipt.Receipt, snapshotRaw sql.NullString) {
	if !snapshotRaw.Valid || strings.TrimSpace(snapshotRaw.String) == "" {
		return
	}
	var snap snapshotJSON
	if err := json.Unmarshal([]byte(snapshotRaw.String), &snap); err != nil {
		return
	}
	out.SnapshotAvailable = true
	out.Client = snap.Client.toDomain()
	out.Provider = snap.Provider.toDomain()
	if snap.Referrer != nil {
		out.Referrer = snap.Referrer.toDomain()
	}
	out.ReferrerCommissionAmountCents = snap.ReferrerCommissionAmountCents
}

// snapshotJSON mirrors the shape stored under the billing_snapshot
// column. Field tags freeze the wire shape so a future renaming can
// only happen via a new migration.
type snapshotJSON struct {
	Client                        partyJSON  `json:"client"`
	Provider                      partyJSON  `json:"provider"`
	Referrer                      *partyJSON `json:"referrer"`
	ReferrerCommissionAmountCents int64      `json:"referrer_commission_amount_cents"`
}

type partyJSON struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	SIRET          string `json:"siret"`
	VAT            string `json:"vat"`
	AddressLine1   string `json:"address_line1"`
	AddressLine2   string `json:"address_line2"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	Country        string `json:"country"`
}

func (p partyJSON) toDomain() *receipt.PartyBilling {
	if p.OrganizationID == "" && p.Name == "" {
		return nil
	}
	id, _ := uuid.Parse(p.OrganizationID) // empty/invalid → uuid.Nil
	return &receipt.PartyBilling{
		OrganizationID: id,
		Name:           p.Name,
		SIRET:          p.SIRET,
		VAT:            p.VAT,
		AddressLine1:   p.AddressLine1,
		AddressLine2:   p.AddressLine2,
		City:           p.City,
		PostalCode:     p.PostalCode,
		Country:        p.Country,
	}
}
