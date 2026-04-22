package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/subscription"
)

// SubscriptionRepository implements repository.SubscriptionRepository
// against Postgres. All queries are parameterized and wrapped in a
// per-query 5 second timeout to keep a stuck query from pinning a
// handler goroutine.
type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// The column list is kept in one const to avoid drift between the three
// queries that scan a Subscription row.
const subscriptionColumns = `
	id, organization_id, plan, billing_cycle, status,
	stripe_customer_id, stripe_subscription_id, stripe_price_id,
	current_period_start, current_period_end,
	cancel_at_period_end, grace_period_ends_at, canceled_at,
	started_at,
	pending_billing_cycle, pending_cycle_effective_at, stripe_schedule_id,
	created_at, updated_at
`

func (r *SubscriptionRepository) Create(ctx context.Context, s *domain.Subscription) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			id, organization_id, plan, billing_cycle, status,
			stripe_customer_id, stripe_subscription_id, stripe_price_id,
			current_period_start, current_period_end,
			cancel_at_period_end, grace_period_ends_at, canceled_at,
			started_at,
			pending_billing_cycle, pending_cycle_effective_at, stripe_schedule_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19
		)`,
		s.ID, s.OrganizationID, string(s.Plan), string(s.BillingCycle), string(s.Status),
		s.StripeCustomerID, s.StripeSubscriptionID, s.StripePriceID,
		s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.GracePeriodEndsAt, s.CanceledAt,
		s.StartedAt,
		pendingCycleStringOrNil(s.PendingBillingCycle), s.PendingCycleEffectiveAt, s.StripeScheduleID,
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert subscription: %w", err)
	}
	return nil
}

// pendingCycleStringOrNil returns the BillingCycle as a pointer-to-string
// so Postgres receives NULL when the tuple is absent. The `all-or-none`
// CHECK constraint depends on this being NULL, not "".
func pendingCycleStringOrNil(c *domain.BillingCycle) *string {
	if c == nil {
		return nil
	}
	str := string(*c)
	return &str
}

func (r *SubscriptionRepository) FindOpenByOrganization(ctx context.Context, organizationID uuid.UUID) (*domain.Subscription, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+subscriptionColumns+`
		FROM subscriptions
		WHERE organization_id = $1
		  AND status IN ('incomplete', 'active', 'past_due')
		LIMIT 1`, organizationID)

	s, err := scanSubscription(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find open subscription: %w", err)
	}
	return s, nil
}

func (r *SubscriptionRepository) FindByStripeID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+subscriptionColumns+`
		FROM subscriptions
		WHERE stripe_subscription_id = $1
		LIMIT 1`, stripeSubID)

	s, err := scanSubscription(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find subscription by stripe id: %w", err)
	}
	return s, nil
}

func (r *SubscriptionRepository) Update(ctx context.Context, s *domain.Subscription) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// updated_at is driven by the SQL trigger subscriptions_updated_at;
	// we still pass it so the value the app layer computed (now()) lands
	// in the row even if the trigger is ever removed.
	res, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions SET
			plan = $2,
			billing_cycle = $3,
			status = $4,
			stripe_customer_id = $5,
			stripe_subscription_id = $6,
			stripe_price_id = $7,
			current_period_start = $8,
			current_period_end = $9,
			cancel_at_period_end = $10,
			grace_period_ends_at = $11,
			canceled_at = $12,
			started_at = $13,
			pending_billing_cycle = $14,
			pending_cycle_effective_at = $15,
			stripe_schedule_id = $16,
			updated_at = now()
		WHERE id = $1`,
		s.ID, string(s.Plan), string(s.BillingCycle), string(s.Status),
		s.StripeCustomerID, s.StripeSubscriptionID, s.StripePriceID,
		s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.GracePeriodEndsAt, s.CanceledAt,
		s.StartedAt,
		pendingCycleStringOrNil(s.PendingBillingCycle), s.PendingCycleEffectiveAt, s.StripeScheduleID,
	)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// scanSubscription reads a row returned by the SELECT queries into the
// domain type. All nullable columns are handled via sql.Null* then
// normalised to Go native types/pointers.
func scanSubscription(row *sql.Row) (*domain.Subscription, error) {
	var (
		s                  domain.Subscription
		plan               string
		cycle              string
		status             string
		gracePeriod        sql.NullTime
		canceledAt         sql.NullTime
		pendingCycle       sql.NullString
		pendingEffectiveAt sql.NullTime
		scheduleID         sql.NullString
	)
	err := row.Scan(
		&s.ID, &s.OrganizationID, &plan, &cycle, &status,
		&s.StripeCustomerID, &s.StripeSubscriptionID, &s.StripePriceID,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.CancelAtPeriodEnd, &gracePeriod, &canceledAt,
		&s.StartedAt,
		&pendingCycle, &pendingEffectiveAt, &scheduleID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Plan = domain.Plan(plan)
	s.BillingCycle = domain.BillingCycle(cycle)
	s.Status = domain.Status(status)
	if gracePeriod.Valid {
		s.GracePeriodEndsAt = &gracePeriod.Time
	}
	if canceledAt.Valid {
		s.CanceledAt = &canceledAt.Time
	}
	if pendingCycle.Valid {
		c := domain.BillingCycle(pendingCycle.String)
		s.PendingBillingCycle = &c
	}
	if pendingEffectiveAt.Valid {
		t := pendingEffectiveAt.Time
		s.PendingCycleEffectiveAt = &t
	}
	if scheduleID.Valid {
		sid := scheduleID.String
		s.StripeScheduleID = &sid
	}
	return &s, nil
}
