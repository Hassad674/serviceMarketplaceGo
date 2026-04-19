-- 113_search_queries_ltr_features.up.sql
--
-- Phase 6G of the Ranking V1 rollout (docs/ranking-v1.md §9.1).
--
-- Extends search_queries with the data needed to train a Learning-
-- to-Rank model once we have 3-6 months of real-user traffic:
--
--   result_features_json  — JSONB payload holding the full feature
--                           vector + final score for each of the 20
--                           ranked documents. Shape:
--                             [
--                               {"doc_id": "...", "rank_position": 1,
--                                "final_score": 87.3, "features": {...}},
--                               ...
--                             ]
--
--   result_vector_sha     — SHA-256 of the canonicalised payload.
--                           Lets the capture path be idempotent:
--                           writing the same result set twice for
--                           the same search_id produces the same
--                           hash, and the INSERT deduplicates.
--
-- Storage back-of-the-envelope: 20 docs × ~12 numbers per doc ≈ 3
-- KB per row. At 10 k queries/day that's ≈ 30 MB/day → ~11 GB/year.
-- Negligible vs. our PG footprint today.
--
-- The hash column is indexed (partial: WHERE NOT NULL) so admin
-- tooling can dedupe exports by ordering fingerprint without a full
-- table scan.

ALTER TABLE search_queries
    ADD COLUMN IF NOT EXISTS result_features_json JSONB,
    ADD COLUMN IF NOT EXISTS result_vector_sha    TEXT;

CREATE INDEX IF NOT EXISTS idx_search_queries_result_vector_sha
    ON search_queries(result_vector_sha)
 WHERE result_vector_sha IS NOT NULL;
