package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// FindPlatformFeeByMilestoneID returns the platform_fee invoice emitted
// for the given milestone, or invoicing.ErrNotFound when no row exists.
//
// Used as the idempotence probe by the per-milestone immediate emission
// path (app.invoicing.IssueFromMilestone) AND by the monthly
// safety-net scheduler to skip milestones already invoiced. The query
// constrains source_type='platform_fee' so it never returns a row from
// the legacy subscription / monthly_commission flows even though
// milestone_id is column-shared.
//
// The system actor warning is suppressed: the per-milestone path runs
// on the synchronous milestone-approval flow which has no current_org_id
// yet (the org is on the payment record, not in the JWT context). The
// caller is expected to tag system.WithSystemActor when crossing this
// boundary if RLS is later extended to require it.
func (r *InvoiceRepository) FindPlatformFeeByMilestoneID(ctx context.Context, milestoneID uuid.UUID) (*invoicing.Invoice, error) {
	if milestoneID == uuid.Nil {
		return nil, invoicing.ErrNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT `+invoiceColumns+`
		FROM invoice
		WHERE source_type = 'platform_fee'
		  AND milestone_id = $1`, milestoneID)

	inv, err := scanInvoice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invoicing.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find platform_fee invoice by milestone id: %w", err)
	}

	items, err := r.loadItems(ctx, inv.ID)
	if err != nil {
		return nil, fmt.Errorf("find platform_fee invoice by milestone id: load items: %w", err)
	}
	inv.Items = items
	return inv, nil
}
