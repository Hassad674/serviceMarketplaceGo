// stripe-backfill-metadata is a one-shot script that brings Stripe
// customers and subscriptions created before the org-scoped migration
// (backend/migrations/119_subscriptions_org_scoped) in line with the
// new metadata contract: every object gets metadata['organization_id']
// populated from the caller's current users.organization_id.
//
// The script is idempotent — objects that already carry
// metadata['organization_id'] are skipped. A --dry-run flag prints what
// would happen without mutating anything.
//
// Usage:
//
//	DATABASE_URL=... STRIPE_SECRET_KEY=sk_... \
//	  go run ./cmd/stripe-backfill-metadata --dry-run
//	DATABASE_URL=... STRIPE_SECRET_KEY=sk_... \
//	  go run ./cmd/stripe-backfill-metadata
//
// The script does NOT remove the legacy metadata['user_id'] — that
// stays as belt-and-suspenders for the dual-read window. A follow-up
// PR drops the dual-read once we've confirmed all webhooks resolve
// via organization_id in production.
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
	"github.com/stripe/stripe-go/v82/customer"
	stripesub "github.com/stripe/stripe-go/v82/subscription"

	"marketplace-backend/internal/adapter/postgres"
)

type stats struct {
	customersScanned   int
	customersUpdated   int
	customersSkipped   int
	subscriptionsScanned int
	subscriptionsUpdated int
	subscriptionsSkipped int
	missingUserRows    int
	userWithoutOrg     int
}

func main() {
	dryRun := flag.Bool("dry-run", false, "print actions without mutating Stripe")
	flag.Parse()

	databaseURL := mustEnv("DATABASE_URL")
	stripeKey := mustEnv("STRIPE_SECRET_KEY")
	stripe.Key = stripeKey

	db, err := postgres.NewConnection(databaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := &stats{}
	backfillCustomers(ctx, db, s, *dryRun)
	backfillSubscriptions(ctx, db, s, *dryRun)

	fmt.Println("---")
	fmt.Printf("customers:      scanned=%d updated=%d skipped=%d\n",
		s.customersScanned, s.customersUpdated, s.customersSkipped)
	fmt.Printf("subscriptions:  scanned=%d updated=%d skipped=%d\n",
		s.subscriptionsScanned, s.subscriptionsUpdated, s.subscriptionsSkipped)
	fmt.Printf("warnings:       missing_user_rows=%d user_without_org=%d\n",
		s.missingUserRows, s.userWithoutOrg)
	if *dryRun {
		fmt.Println("DRY RUN — no Stripe objects were modified.")
	}
}

// backfillCustomers walks every active Stripe customer that still
// carries metadata['user_id'] without metadata['organization_id'] and
// attaches the org id derived from the local users row. Customers with
// neither key are logged and skipped — the script cannot guess the
// owning entity for bare customers.
func backfillCustomers(ctx context.Context, db *sql.DB, s *stats, dryRun bool) {
	iter := customer.List(&stripe.CustomerListParams{
		ListParams: stripe.ListParams{Context: ctx, Limit: stripe.Int64(100)},
	})
	for iter.Next() {
		c := iter.Customer()
		s.customersScanned++
		orgID, skip := deriveOrgForBackfill(db, c.Metadata, s)
		if skip {
			s.customersSkipped++
			continue
		}
		if dryRun {
			log.Printf("DRY: customer %s would gain organization_id=%s", c.ID, orgID)
			s.customersUpdated++
			continue
		}
		params := &stripe.CustomerParams{}
		params.AddMetadata("organization_id", orgID)
		params.Context = ctx
		if _, err := customer.Update(c.ID, params); err != nil {
			log.Printf("WARN: update customer %s: %v", c.ID, err)
			continue
		}
		s.customersUpdated++
	}
	if err := iter.Err(); err != nil {
		log.Printf("WARN: customer list iterator: %v", err)
	}
}

// backfillSubscriptions applies the same backfill to subscription
// metadata. Stripe Subscriptions carry their own metadata map separate
// from the Customer's — the webhook handler reads the Subscription's
// copy when resolving the local row.
func backfillSubscriptions(ctx context.Context, db *sql.DB, s *stats, dryRun bool) {
	iter := stripesub.List(&stripe.SubscriptionListParams{
		ListParams: stripe.ListParams{Context: ctx, Limit: stripe.Int64(100)},
		Status:     stripe.String("all"),
	})
	for iter.Next() {
		sub := iter.Subscription()
		s.subscriptionsScanned++
		orgID, skip := deriveOrgForBackfill(db, sub.Metadata, s)
		if skip {
			s.subscriptionsSkipped++
			continue
		}
		if dryRun {
			log.Printf("DRY: subscription %s would gain organization_id=%s", sub.ID, orgID)
			s.subscriptionsUpdated++
			continue
		}
		params := &stripe.SubscriptionParams{}
		params.AddMetadata("organization_id", orgID)
		params.Context = ctx
		if _, err := stripesub.Update(sub.ID, params); err != nil {
			log.Printf("WARN: update subscription %s: %v", sub.ID, err)
			continue
		}
		s.subscriptionsUpdated++
	}
	if err := iter.Err(); err != nil {
		log.Printf("WARN: subscription list iterator: %v", err)
	}
}

// deriveOrgForBackfill returns the organization id we should set on the
// Stripe object, or skip=true when the object is already fine or
// unresolvable. The caller never rewrites existing organization_id —
// idempotency means the script can safely run more than once.
func deriveOrgForBackfill(db *sql.DB, metadata map[string]string, s *stats) (orgID string, skip bool) {
	if metadata == nil {
		return "", true
	}
	if metadata["organization_id"] != "" {
		return "", true // already backfilled
	}
	rawUser := metadata["user_id"]
	if rawUser == "" {
		return "", true // nothing to resolve from
	}
	userID, err := uuid.Parse(rawUser)
	if err != nil {
		log.Printf("WARN: metadata user_id %q not a uuid — skipping", rawUser)
		return "", true
	}
	var orgUUID uuid.UUID
	err = db.QueryRow(`SELECT organization_id FROM users WHERE id = $1`, userID).Scan(&orgUUID)
	if err == sql.ErrNoRows {
		s.missingUserRows++
		return "", true
	}
	if err != nil {
		log.Printf("WARN: lookup user %s: %v", userID, err)
		return "", true
	}
	if orgUUID == uuid.Nil {
		s.userWithoutOrg++
		return "", true
	}
	return orgUUID.String(), false
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}
