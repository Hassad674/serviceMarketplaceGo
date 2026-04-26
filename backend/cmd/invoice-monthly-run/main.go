// Package main is the one-shot CLI that runs the monthly invoicing
// consolidation batch outside of the in-process scheduler.
//
// Usage:
//
//	invoice-monthly-run -year=2026 -month=4
//	invoice-monthly-run -year=2026 -month=4 -org=<uuid>
//	invoice-monthly-run -year=2026 -month=4 -dry-run
//
// The CLI is feature-flag-aware: missing INVOICE_ISSUER_* env vars or a
// failing PDF renderer fail fast with a clear error so a misconfigured
// deployment never silently produces nothing. Output on stdout is one
// line per org for grep-friendly piping into log aggregators:
//
//	org=<uuid> records=<n> fee=<cents> result=<issued|skipped|error>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	emailadapter "marketplace-backend/internal/adapter/email"
	pdfadapter "marketplace-backend/internal/adapter/pdf"
	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	resendadapter "marketplace-backend/internal/adapter/resend"
	s3adapter "marketplace-backend/internal/adapter/s3"
	"marketplace-backend/internal/config"
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
	year     int
	month    int
	orgArg   string
	dryRun   bool
}

func main() {
	rc, err := parseFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("invoice-monthly-run: %v", err)
	}

	cfg := config.Load()

	exitCode := run(context.Background(), rc, cfg)
	os.Exit(exitCode)
}

// parseFlags returns a typed runConfig from the raw argv. Returns an
// error rather than calling os.Exit so the unit test can drive it.
func parseFlags(args []string) (runConfig, error) {
	fs := flag.NewFlagSet("invoice-monthly-run", flag.ContinueOnError)
	year := fs.Int("year", 0, "year of the period to consolidate (required)")
	month := fs.Int("month", 0, "month of the period to consolidate, 1-12 (required)")
	orgArg := fs.String("org", "all", "uuid of an organization or 'all'")
	dryRun := fs.Bool("dry-run", false, "list what would be issued without writing")

	if err := fs.Parse(args); err != nil {
		return runConfig{}, err
	}

	if *year < 2000 {
		return runConfig{}, fmt.Errorf("invalid -year=%d (must be >= 2000)", *year)
	}
	if *month < 1 || *month > 12 {
		return runConfig{}, fmt.Errorf("invalid -month=%d (must be 1..12)", *month)
	}
	if *orgArg != "all" {
		if _, err := uuid.Parse(*orgArg); err != nil {
			return runConfig{}, fmt.Errorf("invalid -org=%q (must be 'all' or a uuid): %w", *orgArg, err)
		}
	}
	return runConfig{
		year:   *year,
		month:  *month,
		orgArg: *orgArg,
		dryRun: *dryRun,
	}, nil
}

// run wires the dependencies and dispatches the batch. Returns the
// process exit code (0 on full success, 1 on any per-org error or
// init failure). Extracted from main so tests can inject custom
// arguments and inspect stdout deterministically.
func run(ctx context.Context, rc runConfig, cfg *config.Config) int {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	deps, err := wireDependencies(cfg, rc.dryRun)
	if err != nil {
		log.Printf("invoice-monthly-run: %v", err)
		return 1
	}
	defer deps.close()

	orgIDs, err := resolveOrgs(ctx, deps.orgs, rc.orgArg)
	if err != nil {
		log.Printf("invoice-monthly-run: resolve orgs: %v", err)
		return 1
	}
	if len(orgIDs) == 0 {
		fmt.Println("invoice-monthly-run: no organizations matched, nothing to do")
		return 0
	}

	periodStart := time.Date(rc.year, time.Month(rc.month), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	exitCode := 0
	for _, orgID := range orgIDs {
		line := processOrg(ctx, deps, rc, orgID, periodStart, periodEnd)
		fmt.Println(line.String())
		if line.result == resultError {
			exitCode = 1
		}
	}
	return exitCode
}

// processOrgResult is the per-org grep-friendly summary of one batch
// iteration. Stringified for stdout via String().
type processOrgResult struct {
	orgID   uuid.UUID
	records int
	feeCents int64
	result  string
	errMsg  string
}

func (r processOrgResult) String() string {
	base := fmt.Sprintf("org=%s records=%d fee=%d result=%s",
		r.orgID, r.records, r.feeCents, r.result)
	if r.errMsg != "" {
		return base + " error=" + r.errMsg
	}
	return base
}

// processOrg runs a single org through the batch. In dry-run mode it
// only queries ListReleasedPaymentRecordsForOrg (no writes); in normal
// mode it calls IssueMonthlyConsolidated and reports the outcome.
func processOrg(ctx context.Context, deps *cliDeps, rc runConfig, orgID uuid.UUID, periodStart, periodEnd time.Time) processOrgResult {
	out := processOrgResult{orgID: orgID, result: resultSkipped}

	if rc.dryRun {
		records, err := deps.invoiceRepo.ListReleasedPaymentRecordsForOrg(ctx, orgID, periodStart, periodEnd)
		if err != nil {
			out.result = resultError
			out.errMsg = err.Error()
			return out
		}
		out.records = len(records)
		for _, rec := range records {
			out.feeCents += rec.PlatformFeeCents
		}
		return out
	}

	inv, err := deps.invoicingSvc.IssueMonthlyConsolidated(ctx, invoicingapp.IssueMonthlyConsolidatedInput{
		OrganizationID: orgID,
		Year:           rc.year,
		Month:          rc.month,
	})
	if err != nil {
		out.result = resultError
		out.errMsg = err.Error()
		return out
	}
	if inv == nil {
		// No released milestones in the period — service returned (nil, nil).
		return out
	}
	out.result = resultIssued
	out.records = len(inv.Items)
	out.feeCents = inv.AmountInclTaxCents
	return out
}

// resolveOrgs turns the -org flag into a concrete list of uuids.
func resolveOrgs(ctx context.Context, orgs *postgres.OrganizationRepository, orgArg string) ([]uuid.UUID, error) {
	if orgArg == "all" {
		return orgs.ListWithStripeAccount(ctx)
	}
	id, err := uuid.Parse(orgArg)
	if err != nil {
		return nil, err
	}
	return []uuid.UUID{id}, nil
}

// cliDeps carries the wired-up dependencies. Created per-run so the
// CLI does not leak DB / Redis connections.
type cliDeps struct {
	db           closer
	redisClose   closer
	orgs         *postgres.OrganizationRepository
	invoiceRepo  *postgres.InvoiceRepository
	invoicingSvc *invoicingapp.Service
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
// service. Mirrors the order in cmd/api/main.go so a future config
// change there propagates to the CLI naturally.
func wireDependencies(cfg *config.Config, dryRun bool) (*cliDeps, error) {
	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	redisClient, err := redisadapter.NewClient(cfg.RedisURL)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	orgRepo := postgres.NewOrganizationRepository(db, 0)
	invoiceRepo := postgres.NewInvoiceRepository(db)

	deps := &cliDeps{
		db:          db,
		redisClose:  redisClient,
		orgs:        orgRepo,
		invoiceRepo: invoiceRepo,
	}

	// Dry-run does not need PDF / email / issuer config — it only
	// reads the released-records list. Bail out early so missing
	// INVOICE_ISSUER_* env vars don't block a dev-mode preview.
	if dryRun {
		return deps, nil
	}

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
	emailSvc := resendadapter.NewEmailService(cfg.ResendAPIKey, cfg.ResendDevRedirectTo)
	storageSvc := s3adapter.NewStorageService(
		cfg.StorageEndpoint,
		cfg.StorageAccessKey,
		cfg.StorageSecretKey,
		cfg.StorageBucket,
		cfg.StoragePublicURL,
		cfg.StorageUseSSL,
	)
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
