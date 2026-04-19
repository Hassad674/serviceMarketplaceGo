-- 113_search_queries_ltr_features.down.sql
--
-- Rolls back 113 by dropping the LTR feature-vector columns + the
-- partial index. Safe to run on an empty table or after a partial
-- failure of the up migration (IF EXISTS guards every statement).

DROP INDEX IF EXISTS idx_search_queries_result_vector_sha;

ALTER TABLE search_queries
    DROP COLUMN IF EXISTS result_features_json,
    DROP COLUMN IF EXISTS result_vector_sha;
