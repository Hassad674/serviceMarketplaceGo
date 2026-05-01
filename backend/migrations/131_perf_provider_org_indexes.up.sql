-- 131_perf_provider_org_indexes.up.sql
--
-- PERF-B-08: drop two BitmapOr + nested-loop plans by adding
-- provider_organization_id to payment_records (mirroring the column
-- already present on proposals since migration 115) plus the
-- composite partial indexes that back the new "WHERE org_id OR
-- provider_org_id" predicates used by:
--
--   - ListActiveProjectsByOrg / ListCompletedByOrg / IsOrgAuthorized
--     (proposal_queries.go) — old plan: JOIN users + BitmapOr.
--     New plan: Index Scan on idx_proposals_provider_org_status_created
--     for the provider side, Index Scan on idx_proposals_org_status
--     for the client side, Append + Sort.
--
--   - PaymentRecordRepository.ListByOrganization
--     (payment_record_repository.go) — old plan: LEFT JOIN users +
--     BitmapOr. New plan: Index Scan on
--     idx_payment_records_provider_org_created for the provider side,
--     idx_payment_records_org_created (already present, migration 064)
--     for the client side, Append.
--
-- Audit trace: auditperf.md PERF-B-08 / index-audit row #1 / row #2.
--
-- Rollout: indexes are partial + use IF NOT EXISTS so the migration is
-- idempotent. The provider_organization_id column is nullable + ON
-- DELETE SET NULL so historical records keep working when an
-- organization is later removed. The backfill is deterministic and
-- safe to re-run.
--
-- Production operator note: payment_records and proposals are listed
-- in migrations/README.md (convention #4) as growth tables for which
-- index creation should ideally be run with CREATE INDEX
-- CONCURRENTLY to avoid holding ACCESS EXCLUSIVE while the index
-- builds. The CONCURRENTLY clause is incompatible with the
-- transactional wrapper golang-migrate runs every migration in, so
-- if your prod payment_records is large enough that a blocking
-- index build matters (~> 1M rows), apply the indexes manually with
-- CONCURRENTLY before running this migration:
--
--   psql $DATABASE_URL -c "CREATE INDEX CONCURRENTLY IF NOT EXISTS \
--     idx_payment_records_provider_org_created \
--     ON payment_records (provider_organization_id, created_at DESC, id DESC) \
--     WHERE provider_organization_id IS NOT NULL;"
--   psql $DATABASE_URL -c "CREATE INDEX CONCURRENTLY IF NOT EXISTS \
--     idx_proposals_provider_org_status_created \
--     ON proposals (provider_organization_id, status, created_at DESC, id DESC) \
--     WHERE provider_organization_id IS NOT NULL;"
--   psql $DATABASE_URL -c "CREATE INDEX CONCURRENTLY IF NOT EXISTS \
--     idx_proposals_provider_org_completed \
--     ON proposals (provider_organization_id, completed_at DESC, id DESC) \
--     WHERE status = 'completed' AND provider_organization_id IS NOT NULL;"
--
-- Then run `make migrate-up` — the IF NOT EXISTS clauses in the
-- migration's bodies turn the in-tx CREATE INDEX into a no-op. The
-- ALTER TABLE ADD COLUMN below remains as a fast metadata-only
-- operation (Postgres 11+ doesn't rewrite the table for nullable
-- columns without a default).

BEGIN;

-- ---------------------------------------------------------------------
-- 1. payment_records.provider_organization_id — denormalize the
--    provider's users.organization_id at INSERT time so the wallet
--    list view never has to JOIN users on provider_id again.
-- ---------------------------------------------------------------------

ALTER TABLE payment_records
    ADD COLUMN IF NOT EXISTS provider_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

UPDATE payment_records pr
SET    provider_organization_id = u.organization_id
FROM   users u
WHERE  u.id = pr.provider_id
  AND  pr.provider_organization_id IS NULL
  AND  u.organization_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payment_records_provider_org_created
    ON payment_records (provider_organization_id, created_at DESC, id DESC)
    WHERE provider_organization_id IS NOT NULL;

-- ---------------------------------------------------------------------
-- 2. proposals.provider_organization_id composite indexes — the
--    column itself was already added by migration 115 with a
--    backfill, so we only need the read-path indexes that match the
--    refactored queries.
-- ---------------------------------------------------------------------

-- (provider_org_id, status, created_at DESC, id DESC) — backs
-- ListActiveProjectsByOrg's provider-side filter on the in-flight
-- statuses (paid / active / completion_requested / completed /
-- disputed). Partial index keeps the footprint to current/recent
-- projects only.
CREATE INDEX IF NOT EXISTS idx_proposals_provider_org_status_created
    ON proposals (provider_organization_id, status, created_at DESC, id DESC)
    WHERE provider_organization_id IS NOT NULL;

-- (provider_org_id, completed_at DESC, id DESC) — dedicated index for
-- the project-history view ordered by completion. Symmetric with the
-- client-side idx_proposals_client_org_completed (migration 115).
CREATE INDEX IF NOT EXISTS idx_proposals_provider_org_completed
    ON proposals (provider_organization_id, completed_at DESC, id DESC)
    WHERE status = 'completed' AND provider_organization_id IS NOT NULL;

COMMIT;
