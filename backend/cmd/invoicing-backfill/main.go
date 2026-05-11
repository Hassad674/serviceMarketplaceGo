// Package main is the one-shot CLI that backfills platform_fee
// invoices for historical milestones approved before the per-milestone
// emission path was wired in. Mirrors invoice-monthly-run for the
// per-milestone flow.
//
// Usage:
//
//	invoicing-backfill -since=2026-01-01
//	invoicing-backfill -since=2026-01-01 -dry-run
//
// The CLI is feature-flag-aware: missing INVOICE_ISSUER_* env vars or a
// failing PDF renderer fail fast in non-dry-run mode. Output on stdout
// is one line per eligible milestone for grep-friendly piping into log
// aggregators:
//
//	milestone=<uuid> result=<issued|skipped|error> [number=<FAC-...>] [reason=<...>]
//
// Idempotent: re-runnable safely. The synchronous emission path's
// FindPlatformFeeByMilestoneID probe short-circuits already-invoiced
// milestones; a second run produces only "skipped" lines.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	emailadapter "marketplace-backend/internal/adapter/email"
	pdfadapter "marketplace-backend/internal/adapter/pdf"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/system"
	"marketplace-backend/pkg/confighelpers"
)

const (
	resultIssued  = "issued"
	resultSkipped = "skipped"
	resultError   = "error"
)

// runConfig groups the parsed CLI flags so the rest of the program
// reads from a typed struct instead of dereferencing flag pointers.
type runConfig struct {
	since  time.Time
	dryRun bool
}

func main() {
	rc, err := parseFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("invoicing-backfill: %v", err)
	}
	cfg := config.Load()
	os.Exit(run(context.Background(), rc, cfg))
}

// parseFlags returns a typed runConfig from the raw argv.
func parseFlags(args []string) (runConfig, error) {
	fs := flag.NewFlagSet("invoicing-backfill", flag.ContinueOnError)
	since := fs.String("since", "", "ISO date (YYYY-MM-DD) — only milestones approved after this are backfilled. Required.")
	dryRun := fs.Bool("dry-run", false, "List what would be issued without writing.")
	if err := fs.Parse(args); err != nil {
		return runConfig{}, err
	}
	if *since == "" {
		return runConfig{}, fmt.Errorf("-since=YYYY-MM-DD is required")
	}
	t, err := time.Parse("2006-01-02", *since)
	if err != nil {
		return runConfig{}, fmt.Errorf("invalid -since=%q: %w", *since, err)
	}
	return runConfig{since: t.UTC(), dryRun: *dryRun}, nil
}

// run wires the dependencies and processes the backfill.
func run(ctx context.Context, rc runConfig, cfg *config.Config) int {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	ctx = system.WithSystemActor(ctx)

	deps, err := wireDependencies(cfg, rc.dryRun)
	if err != nil {
		log.Printf("invoicing-backfill: %v", err)
		return 1
	}
	defer deps.close()

	candidates, err := listCandidates(ctx, deps.db, rc.since)
	if err != nil {
		log.Printf("invoicing-backfill: list candidates: %v", err)
		return 1
	}
	fmt.Printf("Found %d candidate milestones (approved since %s)\n", len(candidates), rc.since.Format("2006-01-02"))

	if rc.dryRun {
		var alreadyInvoiced, toBackfill int
		for _, c := range candidates {
			has, probeErr := hasExistingPlatformFee(ctx, deps.db, c.MilestoneID)
			if probeErr != nil {
				log.Printf("milestone=%s result=error reason=probe_failed err=%v", c.MilestoneID, probeErr)
				continue
			}
			if has {
				alreadyInvoiced++
				continue
			}
			toBackfill++
			fmt.Printf("milestone=%s result=would_issue fee_cents=%d provider_org=%s\n",
				c.MilestoneID, c.PlatformFeeCents, c.ProviderOrgID)
		}
		fmt.Printf("\nSummary (dry-run): %d eligible, %d already invoiced, %d to backfill.\n",
			len(candidates), alreadyInvoiced, toBackfill)
		return 0
	}

	exitCode := 0
	var issued, skipped, errored int
	for _, c := range candidates {
		result, line := processCandidate(ctx, deps, c)
		fmt.Println(line)
		switch result {
		case resultIssued:
			issued++
		case resultSkipped:
			skipped++
		default:
			errored++
			exitCode = 1
		}
	}
	fmt.Printf("\nSummary: %d eligible, %d issued, %d skipped, %d errored.\n",
		len(candidates), issued, skipped, errored)
	return exitCode
}

// candidate is the slim projection the backfill loop iterates over.
type candidate struct {
	MilestoneID      uuid.UUID
	PaymentRecordID  uuid.UUID
	ProviderUserID   uuid.UUID
	ProviderOrgID    uuid.UUID // may be uuid.Nil — fallback resolves via users.organization_id
	PlatformFeeCents int64
	ApprovedAt       time.Time
}

// listCandidates returns the milestones eligible for backfill: an
// approved (status='approved' OR 'released') proposal_milestone with a
// succeeded payment_record on or after `since` and platform_fee > 0.
//
// We deliberately query the payment_records table for the fee + currency
// + provider id (the source of truth) and join proposal_milestones on
// milestone_id for the approval timestamp. Rows whose payment record is
// not Succeeded are skipped — there is nothing to bill.
func listCandidates(ctx context.Context, db *sql.DB, since time.Time) ([]candidate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT pr.milestone_id, pr.id, pr.provider_id,
		       COALESCE(pr.provider_organization_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       pr.platform_fee_amount,
		       COALESCE(pm.approved_at, pr.paid_at, pr.created_at)
		FROM payment_records pr
		JOIN proposal_milestones pm ON pm.id = pr.milestone_id
		WHERE pr.status = 'succeeded'
		  AND pr.platform_fee_amount > 0
		  AND pm.status IN ('approved', 'released')
		  AND COALESCE(pm.approved_at, pr.paid_at, pr.created_at) >= $1
		ORDER BY COALESCE(pm.approved_at, pr.paid_at, pr.created_at) ASC, pr.id ASC`, since)
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()

	out := make([]candidate, 0)
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.MilestoneID, &c.PaymentRecordID, &c.ProviderUserID, &c.ProviderOrgID, &c.PlatformFeeCents, &c.ApprovedAt); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// hasExistingPlatformFee returns true when the milestone already has a
// platform_fee invoice — the dedup probe used by the dry-run path.
func hasExistingPlatformFee(ctx context.Context, db *sql.DB, milestoneID uuid.UUID) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM invoice
		WHERE source_type = 'platform_fee' AND milestone_id = $1`, milestoneID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// processCandidate emits the invoice for one candidate row and returns
// the result tag and the stdout line. Idempotent via the service-layer
// FindPlatformFeeByMilestoneID probe.
func processCandidate(ctx context.Context, deps *cliDeps, c candidate) (string, string) {
	orgID, resolveLine := resolveOrgForCandidate(ctx, deps.db, c)
	if resolveLine != "" {
		return resultError, resolveLine
	}

	// We need a payment.PaymentRecord aggregate for the service call;
	// the dedicated repo method handles the heavy lifting (RLS, scan).
	rec, err := deps.payments.GetByMilestoneID(ctx, c.MilestoneID)
	if err != nil {
		return resultError, fmt.Sprintf("milestone=%s result=%s reason=payment_record_lookup_failed err=%v", c.MilestoneID, resultError, err)
	}

	inv, err := deps.invoicingSvc.IssueFromMilestone(ctx, invoicingapp.IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: orgID,
		ApprovedAt:             c.ApprovedAt,
	})
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			return resultError, fmt.Sprintf("milestone=%s result=%s reason=billing_profile_missing", c.MilestoneID, resultError)
		}
		return resultError, fmt.Sprintf("milestone=%s result=%s reason=issue_failed err=%v", c.MilestoneID, resultError, err)
	}
	if inv == nil {
		return resultSkipped, fmt.Sprintf("milestone=%s result=%s reason=skipped_premium_or_existing", c.MilestoneID, resultSkipped)
	}
	return resultIssued, fmt.Sprintf("milestone=%s result=%s number=%s amount_cents=%d", c.MilestoneID, resultIssued, inv.Number, inv.AmountInclTaxCents)
}

// resolveOrgForCandidate returns the provider's org id, falling back to
// users.organization_id when the payment_records column is missing.
// Returns the org id + empty string on success, uuid.Nil + a formatted
// error line on failure so the caller can echo it without nesting.
func resolveOrgForCandidate(ctx context.Context, db *sql.DB, c candidate) (uuid.UUID, string) {
	if c.ProviderOrgID != uuid.Nil {
		return c.ProviderOrgID, ""
	}
	var resolved sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT organization_id::text FROM users WHERE id = $1`, c.ProviderUserID).Scan(&resolved); err != nil {
		return uuid.Nil, fmt.Sprintf("milestone=%s result=%s reason=org_lookup_failed err=%v", c.MilestoneID, resultError, err)
	}
	if !resolved.Valid || resolved.String == "" {
		return uuid.Nil, fmt.Sprintf("milestone=%s result=%s reason=provider_has_no_org provider_user=%s", c.MilestoneID, resultError, c.ProviderUserID)
	}
	parsed, perr := uuid.Parse(resolved.String)
	if perr != nil {
		return uuid.Nil, fmt.Sprintf("milestone=%s result=%s reason=org_parse_failed err=%v", c.MilestoneID, resultError, perr)
	}
	return parsed, ""
}

// cliDeps carries the wired-up dependencies.
type cliDeps struct {
	db           *sql.DB
	redisClose   closer
	invoicingSvc *invoicingapp.Service
	payments     *postgres.PaymentRecordRepository
}

type closer interface{ Close() error }

func (d *cliDeps) close() {
	if d == nil {
		return
	}
	if d.redisClose != nil {
		_ = d.redisClose.Close()
	}
	if d.db != nil {
		_ = d.db.Close()
	}
}

// wireDependencies stands up just enough of the backend to issue
// invoices: postgres, redis, storage, email, pdf, the invoicing
// service. Mirrors invoice-monthly-run/wireDependencies to keep config
// drift in one place.
func wireDependencies(cfg *config.Config, dryRun bool) (*cliDeps, error) {
	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	deps := &cliDeps{db: db}

	deps.payments = postgres.NewPaymentRecordRepository(db)

	if dryRun {
		// Dry-run only inspects the schema — bail out before requiring
		// Redis / Stripe / issuer config.
		return deps, nil
	}

	redisClient, err := redisadapter.NewClient(cfg.RedisURL)
	if err != nil {
		deps.close()
		return nil, fmt.Errorf("redis connect: %w", err)
	}
	deps.redisClose = redisClient

	issuer, err := confighelpers.LoadInvoiceIssuer()
	if err != nil {
		deps.close()
		return nil, fmt.Errorf("load invoice issuer config: %w", err)
	}
	pdfRenderer, err := pdfadapter.New()
	if err != nil {
		deps.close()
		return nil, fmt.Errorf("init pdf renderer: %w", err)
	}
	emailSvc := resendadapter.NewEmailService(cfg.ResendAPIKey, cfg.EmailFrom, cfg.ResendDevRedirectTo)
	storageSvc := s3adapter.NewStorageService(
		cfg.StorageEndpoint,
		cfg.StorageAccessKey,
		cfg.StorageSecretKey,
		cfg.StorageBucket,
		cfg.StoragePublicURL,
		cfg.StorageUseSSL,
	)
	invoiceRepo := postgres.NewInvoiceRepository(db)
	billingProfileRepo := postgres.NewBillingProfileRepository(db)
	deliverer := emailadapter.NewDeliverer(emailSvc)
	idempotency := redisadapter.NewWebhookIdempotencyStore(redisClient, redisadapter.DefaultWebhookIdempotencyTTL)

	invoicingSvc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    invoiceRepo,
		Profiles:    billingProfileRepo,
		PDF:         pdfRenderer,
		Storage:     storageSvc,
		Deliverer:   deliverer,
		Issuer:      issuer,
		Idempotency: idempotency,
	})
	deps.invoicingSvc = invoicingSvc
	return deps, nil
}
