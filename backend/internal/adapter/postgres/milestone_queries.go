package postgres

// SQL queries for the milestone repository. All queries use parameterised
// placeholders and the standard $N ordering. Status strings and sequences
// flow through the Go domain so the DB is a pure storage layer.

// queryInsertMilestoneColumns is the column list shared by the
// single-row queryInsertMilestone (kept for any single-row use case)
// and the multi-row CreateBatch path that builds N tuples of
// placeholders. Keeping the column list in one place keeps the two
// queries in sync and makes a column add/remove a one-line change.
const queryInsertMilestoneColumns = `(
    id, proposal_id, sequence, title, description, amount, deadline,
    status, version,
    funded_at, submitted_at, approved_at, released_at,
    disputed_at, cancelled_at,
    active_dispute_id, last_dispute_id,
    created_at, updated_at
)`

const queryInsertMilestone = `
INSERT INTO proposal_milestones ` + queryInsertMilestoneColumns + `
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
`

const queryGetMilestoneByID = `
SELECT id, proposal_id, sequence, title, description, amount, deadline,
       status, version,
       funded_at, submitted_at, approved_at, released_at,
       disputed_at, cancelled_at,
       active_dispute_id, last_dispute_id,
       created_at, updated_at
FROM proposal_milestones
WHERE id = $1
`

// queryGetMilestoneByIDForUpdate is no longer used — see BUG-11. The
// previous GetByIDForUpdate adapter opened a transaction, ran SELECT
// FOR UPDATE, and committed immediately, which released the lock at
// commit. Race protection has always come from the optimistic version
// check in queryUpdateMilestone (WHERE id = $1 AND version = $2).
// The renamed GetByIDWithVersion uses a plain SELECT.

const queryListMilestonesByProposal = `
SELECT id, proposal_id, sequence, title, description, amount, deadline,
       status, version,
       funded_at, submitted_at, approved_at, released_at,
       disputed_at, cancelled_at,
       active_dispute_id, last_dispute_id,
       created_at, updated_at
FROM proposal_milestones
WHERE proposal_id = $1
ORDER BY sequence ASC
`

// queryListMilestonesByProposals resolves a batch of proposals in one
// round trip so list endpoints don't generate N+1 queries.
const queryListMilestonesByProposals = `
SELECT id, proposal_id, sequence, title, description, amount, deadline,
       status, version,
       funded_at, submitted_at, approved_at, released_at,
       disputed_at, cancelled_at,
       active_dispute_id, last_dispute_id,
       created_at, updated_at
FROM proposal_milestones
WHERE proposal_id = ANY($1::uuid[])
ORDER BY proposal_id, sequence ASC
`

// queryGetCurrentActiveMilestone returns the lowest-sequence non-terminal
// milestone of a proposal. Terminal statuses are released, cancelled, refunded.
const queryGetCurrentActiveMilestone = `
SELECT id, proposal_id, sequence, title, description, amount, deadline,
       status, version,
       funded_at, submitted_at, approved_at, released_at,
       disputed_at, cancelled_at,
       active_dispute_id, last_dispute_id,
       created_at, updated_at
FROM proposal_milestones
WHERE proposal_id = $1
  AND status NOT IN ('released', 'cancelled', 'refunded')
ORDER BY sequence ASC
LIMIT 1
`

// queryUpdateMilestone enforces optimistic concurrency: the WHERE clause
// matches both the id AND the prior version; on success the version is
// bumped. If zero rows are affected, the adapter returns
// milestone.ErrConcurrentUpdate.
//
// We intentionally do NOT update immutable fields (proposal_id, sequence,
// title, description, amount, deadline, created_at) — those are set at
// insert time and are only changed by future contract-change amendments
// (which will introduce their own migration-aware SQL).
const queryUpdateMilestone = `
UPDATE proposal_milestones
SET status            = $3,
    version           = version + 1,
    funded_at         = $4,
    submitted_at      = $5,
    approved_at       = $6,
    released_at       = $7,
    disputed_at       = $8,
    cancelled_at      = $9,
    active_dispute_id = $10,
    last_dispute_id   = $11,
    updated_at        = $12
WHERE id = $1 AND version = $2
`

const queryInsertMilestoneDeliverable = `
INSERT INTO milestone_deliverables (
    id, milestone_id, filename, url, size, mime_type, uploaded_by, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

const queryListMilestoneDeliverables = `
SELECT id, milestone_id, filename, url, size, mime_type, uploaded_by, created_at
FROM milestone_deliverables
WHERE milestone_id = $1
ORDER BY created_at ASC
`

const queryDeleteMilestoneDeliverable = `
DELETE FROM milestone_deliverables WHERE id = $1
`
