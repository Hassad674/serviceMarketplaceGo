package postgres_test

// Integration tests for InvoiceRepository (migration 121 schema).
// Gated behind MARKETPLACE_TEST_DATABASE_URL via the testDB helper in
// job_credit_repository_test.go — auto-skip when unset.
//
// Run against the local feature DB:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_feat_invoicing?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestInvoiceRepository -count=1

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/invoicing"
)

// invoicingTestProviderUser returns the provider user_id for the given
// org. We need it because payment_records.provider_id references users,
// not organizations — the org link flows back through users.organization_id.
func invoicingTestProviderUser(t *testing.T, db *sql.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	var userID uuid.UUID
	err := db.QueryRow(`SELECT id FROM users WHERE organization_id = $1 LIMIT 1`, orgID).Scan(&userID)
	require.NoError(t, err, "find provider user for org")
	return userID
}

// invoicingTestClient inserts a second user that plays the client role
// on payment_records (FK to users(id)). Cleaned up via t.Cleanup.
func invoicingTestClient(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'x', 'Cli', 'Test', 'Cli Test', 'enterprise')`,
		id, id.String()[:8]+"@cli.local",
	)
	require.NoError(t, err, "insert client user")
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM payment_records WHERE client_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// seedReleasedPaymentRecord plants a payment_records row with the given
// provider/transferred_at, plus a minimal proposal + milestone so the
// FKs are satisfied. Returns the (paymentRecordID, milestoneID) pair.
func seedReleasedPaymentRecord(
	t *testing.T,
	db *sql.DB,
	clientID, providerID uuid.UUID,
	transferredAt time.Time,
	amountCents int64,
	feeCents int64,
) (paymentRecordID, milestoneID uuid.UUID) {
	t.Helper()

	conversationID := uuid.New()
	// Minimal conversation row — only id is required by the schema.
	_, err := db.Exec(`INSERT INTO conversations (id) VALUES ($1)`, conversationID)
	require.NoError(t, err, "insert conversation")

	proposalID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO proposals (
			id, conversation_id, sender_id, recipient_id,
			title, description, amount, status, client_id, provider_id
		) VALUES (
			$1, $2, $3, $4,
			'Mission test', 'Desc', $5, 'accepted', $3, $4
		)`,
		proposalID, conversationID, clientID, providerID, amountCents,
	)
	require.NoError(t, err, "insert proposal")

	milestoneID = uuid.New()
	_, err = db.Exec(`
		INSERT INTO proposal_milestones (
			id, proposal_id, sequence, title, description, amount, status
		) VALUES ($1, $2, 1, 'Step 1', 'desc', $3, 'released')`,
		milestoneID, proposalID, amountCents,
	)
	require.NoError(t, err, "insert milestone")

	paymentRecordID = uuid.New()
	_, err = db.Exec(`
		INSERT INTO payment_records (
			id, proposal_id, milestone_id, client_id, provider_id,
			proposal_amount, stripe_fee_amount, platform_fee_amount,
			client_total_amount, provider_payout,
			currency, status, transfer_status, paid_at, transferred_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, 0, $7,
			$8, $9,
			'eur', 'paid', 'transferred', $10, $10
		)`,
		paymentRecordID, proposalID, milestoneID, clientID, providerID,
		amountCents, feeCents,
		amountCents+feeCents, amountCents-feeCents,
		transferredAt,
	)
	require.NoError(t, err, "insert payment_record")

	t.Cleanup(func() {
		_, _ = db.Exec(`
			DELETE FROM invoice_item WHERE payment_record_id = $1`, paymentRecordID)
		_, _ = db.Exec(`DELETE FROM payment_records WHERE id = $1`, paymentRecordID)
		_, _ = db.Exec(`DELETE FROM proposal_milestones WHERE id = $1`, milestoneID)
		_, _ = db.Exec(`DELETE FROM proposals WHERE id = $1`, proposalID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, conversationID)
	})
	return paymentRecordID, milestoneID
}

// makeFinalizedInvoice builds an invoice via the domain constructor +
// Finalize, ready to hand to CreateInvoice. The invoice has exactly one
// item and a unique stripe_event_id when the caller passes one — empty
// strings flow through as NULL.
func makeFinalizedInvoice(t *testing.T, repo *postgres.InvoiceRepository, orgID uuid.UUID, stripeEventID string) *invoicing.Invoice {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	item := invoicing.InvoiceItem{
		ID:             uuid.New(),
		Description:    "Premium Agence — avril 2026",
		Quantity:       1,
		UnitPriceCents: 5000,
		AmountCents:    5000,
		CreatedAt:      now,
	}
	inv, err := invoicing.NewInvoice(invoicing.NewInvoiceInput{
		RecipientOrganizationID: orgID,
		Recipient: invoicing.RecipientInfo{
			OrganizationID: orgID.String(),
			ProfileType:    "business",
			LegalName:      "Recipient SARL",
			Country:        "FR",
		},
		Issuer: invoicing.IssuerInfo{
			LegalName: "Marketplace SAS",
			SIRET:     "12345678901234",
			Country:   "FR",
		},
		ServicePeriodStart: now.AddDate(0, -1, 0),
		ServicePeriodEnd:   now,
		SourceType:         invoicing.SourceSubscription,
		StripeEventID:      stripeEventID,
		Items:              []invoicing.InvoiceItem{item},
	})
	require.NoError(t, err)

	// Reserve a real number from the counter so the FAC-NNNNNN sequence
	// stays continuous on the test DB.
	seq, err := repo.ReserveNumber(context.Background(), invoicing.ScopeInvoice)
	require.NoError(t, err)
	number := invoicing.FormatInvoiceNumber(seq)
	require.NoError(t, inv.Finalize(number, "r2://invoices/"+number+".pdf"))
	return inv
}

func TestInvoiceRepository_ReserveNumber_Sequential(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)

	first, err := repo.ReserveNumber(context.Background(), invoicing.ScopeInvoice)
	require.NoError(t, err)
	second, err := repo.ReserveNumber(context.Background(), invoicing.ScopeInvoice)
	require.NoError(t, err)
	third, err := repo.ReserveNumber(context.Background(), invoicing.ScopeInvoice)
	require.NoError(t, err)

	assert.Equal(t, first+1, second, "second draw must be exactly first+1")
	assert.Equal(t, second+1, third, "third draw must be exactly second+1")
}

func TestInvoiceRepository_ReserveNumber_ConcurrentNoDuplicates(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)

	const racers = 50
	results := make([]int64, racers)
	errs := make([]error, racers)

	var wg sync.WaitGroup
	wg.Add(racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx], errs[idx] = repo.ReserveNumber(context.Background(), invoicing.ScopeInvoice)
		}(i)
	}
	close(start)
	wg.Wait()

	seen := make(map[int64]struct{}, racers)
	for i, err := range errs {
		require.NoError(t, err, "racer %d", i)
		_, dup := seen[results[i]]
		require.False(t, dup, "duplicate sequence value %d at racer %d", results[i], i)
		seen[results[i]] = struct{}{}
	}
	assert.Len(t, seen, racers, "must have exactly %d unique values", racers)
}

func TestInvoiceRepository_CreateAndFindByID(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)
	orgID := invoicingTestOrg(t, db)

	inv := makeFinalizedInvoice(t, repo, orgID, "evt_test_"+uuid.New().String()[:8])
	require.NoError(t, repo.CreateInvoice(context.Background(), inv))

	got, err := repo.FindInvoiceByID(context.Background(), inv.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, inv.ID, got.ID)
	assert.Equal(t, inv.Number, got.Number)
	assert.Equal(t, inv.RecipientOrganizationID, got.RecipientOrganizationID)
	assert.Equal(t, inv.AmountInclTaxCents, got.AmountInclTaxCents)
	assert.Equal(t, invoicing.StatusIssued, got.Status)
	require.Len(t, got.Items, 1)
	assert.Equal(t, inv.Items[0].Description, got.Items[0].Description)
	assert.Equal(t, inv.Items[0].AmountCents, got.Items[0].AmountCents)
	// Snapshot JSONB round-trip
	assert.Equal(t, inv.RecipientSnapshot.LegalName, got.RecipientSnapshot.LegalName)
	assert.Equal(t, inv.IssuerSnapshot.LegalName, got.IssuerSnapshot.LegalName)
	// Mentions array round-trip
	assert.Equal(t, inv.MentionsRendered, got.MentionsRendered)
}

func TestInvoiceRepository_FindInvoiceByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)

	_, err := repo.FindInvoiceByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, invoicing.ErrNotFound)
}

func TestInvoiceRepository_FindInvoiceByStripeEventID(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)
	orgID := invoicingTestOrg(t, db)

	eventID := "evt_test_" + uuid.New().String()[:8]
	inv := makeFinalizedInvoice(t, repo, orgID, eventID)
	require.NoError(t, repo.CreateInvoice(context.Background(), inv))

	got, err := repo.FindInvoiceByStripeEventID(context.Background(), eventID)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, got.ID)
	require.Len(t, got.Items, 1)

	_, err = repo.FindInvoiceByStripeEventID(context.Background(), "evt_never_exists")
	assert.ErrorIs(t, err, invoicing.ErrNotFound)
}

func TestInvoiceRepository_HasInvoiceItemForPaymentRecord(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)
	orgID := invoicingTestOrg(t, db)

	providerID := invoicingTestProviderUser(t, db, orgID)
	clientID := invoicingTestClient(t, db)
	transferredAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	prID, milestoneID := seedReleasedPaymentRecord(t, db, clientID, providerID, transferredAt, 10000, 1000)

	// No invoice yet → returns false.
	exists, err := repo.HasInvoiceItemForPaymentRecord(context.Background(), prID)
	require.NoError(t, err)
	assert.False(t, exists, "no invoice item yet")

	// Build an invoice with one item referencing this payment_record.
	inv := makeFinalizedInvoice(t, repo, orgID, "evt_test_"+uuid.New().String()[:8])
	pr := prID
	mid := milestoneID
	inv.Items[0].PaymentRecordID = &pr
	inv.Items[0].MilestoneID = &mid
	require.NoError(t, repo.CreateInvoice(context.Background(), inv))

	exists, err = repo.HasInvoiceItemForPaymentRecord(context.Background(), prID)
	require.NoError(t, err)
	assert.True(t, exists, "invoice item now references this payment_record")

	// Random other id stays false.
	exists, err = repo.HasInvoiceItemForPaymentRecord(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, exists, "random id never returns true")
}

func TestInvoiceRepository_ListInvoicesByOrganization_CursorPagination(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)
	orgA := invoicingTestOrg(t, db)
	orgB := invoicingTestOrg(t, db)

	// Seed 5 invoices for orgA + 2 for orgB. The Number sequence is global
	// so each call moves the counter — that is fine.
	const totalA = 5
	idsA := make([]uuid.UUID, 0, totalA)
	for i := 0; i < totalA; i++ {
		inv := makeFinalizedInvoice(t, repo, orgA, "evt_a_"+uuid.New().String()[:8])
		// Force a tiny delay so issued_at strictly increases between rows
		// — we order DESC on (issued_at, id) and want a deterministic
		// reverse order in the cursor walk.
		time.Sleep(2 * time.Millisecond)
		require.NoError(t, repo.CreateInvoice(context.Background(), inv))
		idsA = append(idsA, inv.ID)
	}
	for i := 0; i < 2; i++ {
		inv := makeFinalizedInvoice(t, repo, orgB, "evt_b_"+uuid.New().String()[:8])
		time.Sleep(2 * time.Millisecond)
		require.NoError(t, repo.CreateInvoice(context.Background(), inv))
	}

	// Walk orgA with limit=2.
	const pageLimit = 2
	collected := make([]uuid.UUID, 0, totalA)
	cursor := ""
	pages := 0
	for {
		page, next, err := repo.ListInvoicesByOrganization(context.Background(), orgA, cursor, pageLimit)
		require.NoError(t, err)
		pages++
		for _, inv := range page {
			collected = append(collected, inv.ID)
			// Leakage check: every returned invoice must belong to orgA.
			assert.Equal(t, orgA, inv.RecipientOrganizationID, "leak from other org")
		}
		if next == "" {
			break
		}
		cursor = next
		require.Less(t, pages, 10, "pagination must not loop forever")
	}
	require.Len(t, collected, totalA, "must return exactly the seeded count")

	// DESC order: the last seeded invoice (most recent issued_at) is first.
	assert.Equal(t, idsA[totalA-1], collected[0], "newest first")
	assert.Equal(t, idsA[0], collected[totalA-1], "oldest last")
}

func TestInvoiceRepository_ListReleasedPaymentRecordsForOrg(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewInvoiceRepository(db)
	orgID := invoicingTestOrg(t, db)
	providerID := invoicingTestProviderUser(t, db, orgID)
	clientID := invoicingTestClient(t, db)

	march2026 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	april2026 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	may2026 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// 3 payments transferred in March 2026 belonging to orgID.
	pr1, _ := seedReleasedPaymentRecord(t, db, clientID, providerID,
		time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC), 10000, 1000)
	_, _ = seedReleasedPaymentRecord(t, db, clientID, providerID,
		time.Date(2026, 3, 18, 14, 0, 0, 0, time.UTC), 20000, 2000)
	_, _ = seedReleasedPaymentRecord(t, db, clientID, providerID,
		time.Date(2026, 3, 28, 9, 0, 0, 0, time.UTC), 5000, 500)

	// 1 payment transferred in April 2026 — must NOT show up in the March
	// query.
	_, _ = seedReleasedPaymentRecord(t, db, clientID, providerID,
		time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC), 7500, 750)

	got, err := repo.ListReleasedPaymentRecordsForOrg(context.Background(), orgID, march2026, april2026)
	require.NoError(t, err)
	assert.Len(t, got, 3, "exactly 3 payment_records in the March window")

	// Now seed an invoice that covers pr1 via an invoice_item — the
	// query must drop it on the next call.
	inv := makeFinalizedInvoice(t, repo, orgID, "evt_inv_"+uuid.New().String()[:8])
	pr1Copy := pr1
	inv.Items[0].PaymentRecordID = &pr1Copy
	require.NoError(t, repo.CreateInvoice(context.Background(), inv))

	got, err = repo.ListReleasedPaymentRecordsForOrg(context.Background(), orgID, march2026, april2026)
	require.NoError(t, err)
	assert.Len(t, got, 2, "the invoiced payment_record must be filtered out")
	for _, rec := range got {
		assert.NotEqual(t, pr1, rec.ID, "pr1 must no longer appear")
	}

	// April query → returns the single April payment_record.
	got, err = repo.ListReleasedPaymentRecordsForOrg(context.Background(), orgID, april2026, may2026)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}
