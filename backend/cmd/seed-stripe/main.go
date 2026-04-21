// seed-stripe is an idempotent seed script that ensures the four Premium
// Price objects exist in the connected Stripe account. Prices are keyed
// by stable lookup_keys so application code never hardcodes price IDs —
// the same seeded key works in test and prod.
//
// Usage:
//
//	STRIPE_SECRET_KEY=sk_test_... go run ./cmd/seed-stripe
//	# or: make seed-stripe
//
// Safe to rerun: existing prices are left alone; missing ones are created.
package main

import (
	"fmt"
	"log"
	"os"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// planSpec bundles everything needed to create one (product, price) pair.
type planSpec struct {
	LookupKey     string
	ProductName   string
	AmountCents   int64
	IntervalCount int64
	Interval      string // "month" or "year"
}

func main() {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		log.Fatal("STRIPE_SECRET_KEY must be set (same key as the backend uses)")
	}
	stripe.Key = key

	specs := []planSpec{
		{LookupKey: "premium_freelance_monthly", ProductName: "Premium Freelance — Monthly", AmountCents: 1900, IntervalCount: 1, Interval: "month"},
		{LookupKey: "premium_freelance_annual", ProductName: "Premium Freelance — Annual", AmountCents: 18000, IntervalCount: 1, Interval: "year"},
		{LookupKey: "premium_agency_monthly", ProductName: "Premium Agency — Monthly", AmountCents: 4900, IntervalCount: 1, Interval: "month"},
		{LookupKey: "premium_agency_annual", ProductName: "Premium Agency — Annual", AmountCents: 46800, IntervalCount: 1, Interval: "year"},
	}

	for _, spec := range specs {
		if err := ensurePrice(spec); err != nil {
			log.Fatalf("seed %s: %v", spec.LookupKey, err)
		}
	}
	fmt.Println("stripe seed complete — all four Premium prices are in place")
}

// ensurePrice creates the product + price pair if no active price with
// the given lookup_key exists. When one does, prints the existing id and
// exits cleanly. Never updates or archives an existing price — changes
// to amount or interval MUST be done via a new price + a new lookup_key
// so historical subscriptions stay linked to their original pricing.
func ensurePrice(spec planSpec) error {
	// 1. Existing price?
	iter := price.List(&stripe.PriceListParams{
		LookupKeys: stripe.StringSlice([]string{spec.LookupKey}),
		Active:     stripe.Bool(true),
	})
	for iter.Next() {
		p := iter.Price()
		if p.LookupKey == spec.LookupKey {
			fmt.Printf("  %-35s  %s  (already exists, %d %s)\n",
				spec.LookupKey, p.ID, p.UnitAmount, p.Currency)
			return nil
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("list existing prices: %w", err)
	}

	// 2. Ensure a Product exists (match by name — idempotent enough for
	//    a seed script; manual re-runs of the seed won't orphan prices).
	var productID string
	searchIter := product.Search(&stripe.ProductSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf("name:'%s' AND active:'true'", spec.ProductName),
			Limit: stripe.Int64(1),
		},
	})
	if searchIter.Next() {
		productID = searchIter.Product().ID
	} else {
		if err := searchIter.Err(); err != nil {
			// Stripe sometimes returns "resource_missing" on a cold
			// search index — fall through to product creation.
			log.Printf("  product search for %q returned %v, creating anew", spec.ProductName, err)
		}
		p, err := product.New(&stripe.ProductParams{
			Name: stripe.String(spec.ProductName),
		})
		if err != nil {
			return fmt.Errorf("create product: %w", err)
		}
		productID = p.ID
	}

	// 3. Create the Price with a stable lookup_key.
	newPrice, err := price.New(&stripe.PriceParams{
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(spec.AmountCents),
		Currency:   stripe.String(string(stripe.CurrencyEUR)),
		LookupKey:  stripe.String(spec.LookupKey),
		Recurring: &stripe.PriceRecurringParams{
			Interval:      stripe.String(spec.Interval),
			IntervalCount: stripe.Int64(spec.IntervalCount),
		},
	})
	if err != nil {
		return fmt.Errorf("create price: %w", err)
	}
	fmt.Printf("  %-35s  %s  (created)\n", spec.LookupKey, newPrice.ID)
	return nil
}
