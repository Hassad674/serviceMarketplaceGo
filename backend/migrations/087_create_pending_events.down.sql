DROP INDEX IF EXISTS idx_pending_events_type;
DROP INDEX IF EXISTS idx_pending_events_due;
DROP TRIGGER IF EXISTS pending_events_updated_at ON pending_events;
DROP TABLE IF EXISTS pending_events;
