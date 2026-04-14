package postgres

// SQL for the pending_events table (unified scheduler + outbox queue).
// See migration 087_create_pending_events for the schema.

const queryInsertPendingEvent = `
INSERT INTO pending_events (
    id, event_type, payload, fires_at,
    status, attempts, last_error,
    processed_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

// queryPopDuePendingEvents claims up to N due rows in a single
// statement. The CTE uses FOR UPDATE SKIP LOCKED so concurrent
// workers never see the same row twice; the wrapping UPDATE bumps
// the status to processing, increments attempts, and refreshes
// updated_at — all in one round trip. The RETURNING clause hands
// back the freshly-claimed rows so the worker doesn't need a
// follow-up SELECT.
const queryPopDuePendingEvents = `
WITH due AS (
    SELECT id
    FROM pending_events
    WHERE status IN ('pending', 'failed')
      AND fires_at <= now()
    ORDER BY fires_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE pending_events e
SET    status     = 'processing',
       attempts   = e.attempts + 1,
       updated_at = now()
FROM   due
WHERE  e.id = due.id
RETURNING e.id, e.event_type, e.payload, e.fires_at,
          e.status, e.attempts, e.last_error,
          e.processed_at, e.created_at, e.updated_at
`

const queryMarkPendingEventDone = `
UPDATE pending_events
SET    status       = 'done',
       processed_at = now(),
       last_error   = NULL,
       updated_at   = now()
WHERE  id = $1 AND status = 'processing'
`

const queryMarkPendingEventFailed = `
UPDATE pending_events
SET    status     = 'failed',
       last_error = $2,
       fires_at   = $3,
       updated_at = now()
WHERE  id = $1 AND status = 'processing'
`

const queryGetPendingEventByID = `
SELECT id, event_type, payload, fires_at,
       status, attempts, last_error,
       processed_at, created_at, updated_at
FROM pending_events
WHERE id = $1
`
