-- PERF-E: rollback for 152_perf_indexes_hot_queries.

BEGIN;

DROP INDEX IF EXISTS idx_pve_org_time_came_from;
DROP INDEX IF EXISTS idx_audit_logs_user_time_series;
DROP INDEX IF EXISTS idx_user_sessions_active_by_user;

COMMIT;
