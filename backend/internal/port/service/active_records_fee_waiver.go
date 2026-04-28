package service

import (
	"context"

	"github.com/google/uuid"
)

// ActiveRecordsFeeWaiver is the contract the subscription service uses
// to retroactively zero the platform fee on every payment_record of an
// org that is still in flight (not yet transferred). Called when a
// subscription becomes active so missions started BEFORE the upgrade
// stop carrying a fee from the moment the user goes Premium.
//
// Records that have already been transferred to the provider's Stripe
// connected account are intentionally NOT touched: the money has
// already been split, refunding the fee retroactively would require an
// additional Stripe transfer that this V1 doesn't model. Future work
// can add a top-up flow if the product wants to offer a fee credit on
// already-completed milestones.
//
// Implemented by the payment app service. Wired post-construction via
// subscription.Service.SetFeeWaiver because the payment service is
// built before subscription in main.go (subscription depends on the
// payment-issued SubscriptionReader).
type ActiveRecordsFeeWaiver interface {
	WaivePlatformFeeOnActiveRecords(ctx context.Context, providerOrgID uuid.UUID) error
}
