// stripe-payout-schedule-backfill brings every existing connected
// account in line with the manual-only payout policy: it walks
// organizations.stripe_account_id and calls Stripe's account update
// API with payout_schedule.interval = "manual" so Stripe stops paying
// the connected account out automatically.
//
// Why this exists: Stripe Connect Custom accounts default to a daily
// auto-payout schedule in many countries (notably FR). The product
// rule is the opposite — funds only leave Stripe when the user
// clicks "Retirer" in the wallet UI. Code created after the manual
// schedule landed (cf. adapter/stripe/account.go) is already correct;
// this script is for accounts created before the fix shipped.
//
// The script is idempotent — re-running it is safe. It only ever sets
// the interval to "manual"; if Stripe already reports the account on
// the manual schedule, the API call is a cheap no-op.
//
// Usage:
//
//	# show what would change without mutating Stripe:
//	DATABASE_URL=... STRIPE_SECRET_KEY=sk_... \
//	  go run ./cmd/stripe-payout-schedule-backfill --org=all --dry-run
//
//	# apply for real:
//	DATABASE_URL=... STRIPE_SECRET_KEY=sk_... \
//	  go run ./cmd/stripe-payout-schedule-backfill --org=all
//
//	# scope to a single organization:
//	DATABASE_URL=... STRIPE_SECRET_KEY=sk_... \
//	  go run ./cmd/stripe-payout-schedule-backfill --org=<uuid> --dry-run
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"

	"marketplace-backend/internal/adapter/postgres"
)

type stats struct {
	scanned int
	updated int
	skipped int
	errored int
}

func main() {
	dryRun := flag.Bool("dry-run", false, "print actions without mutating Stripe")
	orgFlag := flag.String("org", "all", "organization id to backfill, or \"all\" for every connected account")
	flag.Parse()

	databaseURL := mustEnv("DATABASE_URL")
	stripeKey := mustEnv("STRIPE_SECRET_KEY")
	stripe.Key = stripeKey

	db, err := postgres.NewConnection(databaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	rows, err := selectAccounts(ctx, db, *orgFlag)
	if err != nil {
		log.Fatalf("list connected accounts: %v", err)
	}

	s := &stats{}
	for _, row := range rows {
		s.scanned++
		if row.AccountID == "" {
			s.skipped++
			continue
		}
		if *dryRun {
			log.Printf("DRY: account %s (org %s) would be updated to manual payout schedule",
				row.AccountID, row.OrgID)
			s.updated++
			continue
		}
		if err := setManual(ctx, row.AccountID); err != nil {
			log.Printf("ERROR: account %s (org %s): %v", row.AccountID, row.OrgID, err)
			s.errored++
			continue
		}
		log.Printf("OK: account %s (org %s) set to manual payout schedule",
			row.AccountID, row.OrgID)
		s.updated++
	}

	fmt.Println("---")
	fmt.Printf("scanned=%d updated=%d skipped=%d errored=%d\n",
		s.scanned, s.updated, s.skipped, s.errored)
	if *dryRun {
		fmt.Println("DRY RUN — no Stripe objects were modified.")
	}
}

// connectedAccountRow is a row of (organization_id, stripe_account_id)
// pulled from the organizations table.
type connectedAccountRow struct {
	OrgID     uuid.UUID
	AccountID string
}

// selectAccounts loads every organization whose Stripe Connect account
// id is non-null. When --org=<uuid> is passed, the result is restricted
// to that single org (and the script errors out if the org has no
// connected account, so a typo doesn't silently no-op).
func selectAccounts(ctx context.Context, db *sql.DB, orgFlag string) ([]connectedAccountRow, error) {
	if orgFlag == "" || orgFlag == "all" {
		rows, err := db.QueryContext(ctx,
			`SELECT id, stripe_account_id FROM organizations
			 WHERE stripe_account_id IS NOT NULL AND stripe_account_id <> ''
			 ORDER BY id`)
		if err != nil {
			return nil, fmt.Errorf("query organizations: %w", err)
		}
		defer rows.Close()
		var out []connectedAccountRow
		for rows.Next() {
			var r connectedAccountRow
			if err := rows.Scan(&r.OrgID, &r.AccountID); err != nil {
				return nil, fmt.Errorf("scan row: %w", err)
			}
			out = append(out, r)
		}
		return out, rows.Err()
	}

	orgID, err := uuid.Parse(orgFlag)
	if err != nil {
		return nil, fmt.Errorf("invalid --org value %q (expected uuid or \"all\"): %w", orgFlag, err)
	}
	var acct sql.NullString
	err = db.QueryRowContext(ctx,
		`SELECT stripe_account_id FROM organizations WHERE id = $1`, orgID).Scan(&acct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization %s not found", orgID)
	}
	if err != nil {
		return nil, fmt.Errorf("lookup organization %s: %w", orgID, err)
	}
	if !acct.Valid || acct.String == "" {
		return nil, fmt.Errorf("organization %s has no connected stripe account", orgID)
	}
	return []connectedAccountRow{{OrgID: orgID, AccountID: acct.String}}, nil
}

// setManual issues the actual Stripe API call. Kept tiny so it's
// trivial to unit-test against an httptest backend.
func setManual(ctx context.Context, accountID string) error {
	params := &stripe.AccountParams{
		Settings: &stripe.AccountSettingsParams{
			Payouts: &stripe.AccountSettingsPayoutsParams{
				Schedule: &stripe.AccountSettingsPayoutsScheduleParams{
					Interval: stripe.String("manual"),
				},
			},
		},
	}
	params.Context = ctx
	if _, err := account.Update(accountID, params); err != nil {
		return fmt.Errorf("update account %s: %w", accountID, err)
	}
	return nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}
