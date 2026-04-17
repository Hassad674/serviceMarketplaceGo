-- 112_search_queries_id_and_click.down.sql
-- Rollback for phase 3 analytics column additions.

DROP INDEX IF EXISTS idx_search_queries_clicked_at;
DROP INDEX IF EXISTS idx_search_queries_search_id;

ALTER TABLE search_queries
    DROP COLUMN IF EXISTS clicked_at,
    DROP COLUMN IF EXISTS search_id;
