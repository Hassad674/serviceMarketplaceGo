-- 111_create_search_queries.down.sql
DROP INDEX IF EXISTS idx_search_queries_query_gin;
DROP INDEX IF EXISTS idx_search_queries_zero_results;
DROP INDEX IF EXISTS idx_search_queries_user_id;
DROP INDEX IF EXISTS idx_search_queries_created_at;
DROP TABLE IF EXISTS search_queries;
