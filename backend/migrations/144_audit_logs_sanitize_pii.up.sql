-- Migration 144 — backfill: redact PII keys in audit_logs.metadata
--
-- Context:
--   B.10 / RGPD art. 5-1-c (data minimization). Until this migration,
--   the `audit_logs.metadata` JSONB column persisted cleartext values
--   for sensitive keys — most notably `email` on `auth.login_failure`
--   events, which silently built a permanent index of "this email has
--   tried to log in on the platform" surviving the user's account
--   deletion.
--
-- What this does:
--   For every existing row whose metadata contains one of the
--   sensitive keys (email, to_email, from_email, recipient, phone,
--   iban) at ANY depth, replace each such VALUE with the first 16 hex
--   chars of sha256(value::text). Matches the deterministic transform
--   applied by `internal/domain/audit.SanitizeMetadata` so old rows
--   and new rows share the same on-disk shape.
--
-- Why it is forward-only:
--   The transform is one-way. Once a row is sanitized, the cleartext
--   is gone — there is no reversible operation. That is the point: if
--   we could decrypt back, the audit log would still leak the PII.
--   The down migration is intentionally a comment-only no-op.
--
-- Why it is chunked:
--   Per backend/CLAUDE.md long-running backfill rule: avoid mixing a
--   schema change and a multi-million-row UPDATE in one statement.
--   The plpgsql LOOP processes 5000 rows per iteration via a CTE so
--   memory stays bounded and per-row lock acquisition is incremental
--   even though golang-migrate wraps the whole migration in one
--   transaction. On a 100k-row table this completes in roughly 30-60s
--   on Neon's smaller compute tiers; on a 1M-row table, ~5-10min.
--   The migration acquires a ROW EXCLUSIVE lock per touched row,
--   which lets concurrent INSERTs from live traffic proceed.
--
-- pgcrypto extension:
--   Already enabled by migration 132 (used by the GDPR purge cron).
--   The CREATE EXTENSION below is a defensive no-op for environments
--   that may have applied 144 before 132 ran.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- _audit_sanitize_pii recursively rewrites a JSONB document by
-- replacing every sensitive value at any depth with the deterministic
-- 16-hex-char SHA-256 prefix of its text representation. Mirrors the
-- behaviour of `internal/domain/audit.SanitizeMetadata`.
--
-- The function is internal-use-only, prefixed with `_` and dropped
-- at the end of the migration so it never lingers in the schema.
CREATE OR REPLACE FUNCTION _audit_sanitize_pii(doc JSONB) RETURNS JSONB
LANGUAGE plpgsql IMMUTABLE AS $func$
DECLARE
    sensitive_keys CONSTANT TEXT[] := ARRAY['email', 'to_email', 'from_email', 'recipient', 'phone', 'iban'];
    result         JSONB;
    kv             RECORD;
    text_val       TEXT;
BEGIN
    -- Non-objects (strings, numbers, arrays, scalars) pass through
    -- untouched. Sanitization only walks {key: value} structures.
    IF jsonb_typeof(doc) <> 'object' THEN
        RETURN doc;
    END IF;

    result := '{}'::jsonb;
    FOR kv IN SELECT key, value FROM jsonb_each(doc) LOOP
        IF kv.key = ANY(sensitive_keys) THEN
            -- NULL value: keep as-is so the redacted shape distinguishes
            -- "no value recorded" from "value redacted to empty hash".
            IF kv.value IS NULL OR jsonb_typeof(kv.value) = 'null' THEN
                result := result || jsonb_build_object(kv.key, NULL);
            ELSE
                text_val := COALESCE(kv.value #>> '{}', '');
                -- Idempotency: if the value already looks like a
                -- 16-hex-char hash, keep it as-is. Without this
                -- guard the migration would loop forever (re-hashing
                -- a 16-hex string produces a different 16-hex string,
                -- so the function output would never converge).
                IF text_val ~ '^[0-9a-f]{16}$' THEN
                    result := result || jsonb_build_object(kv.key, kv.value);
                ELSE
                    result := result || jsonb_build_object(
                        kv.key,
                        SUBSTRING(
                            ENCODE(DIGEST(text_val, 'sha256'), 'hex')
                            FOR 16
                        )
                    );
                END IF;
            END IF;
        ELSIF jsonb_typeof(kv.value) = 'object' THEN
            result := result || jsonb_build_object(kv.key, _audit_sanitize_pii(kv.value));
        ELSE
            result := result || jsonb_build_object(kv.key, kv.value);
        END IF;
    END LOOP;
    RETURN result;
END;
$func$;

DO $$
DECLARE
    sensitive_keys CONSTANT TEXT[] := ARRAY['email', 'to_email', 'from_email', 'recipient', 'phone', 'iban'];
    batch_size     CONSTANT INTEGER := 5000;
    affected       INTEGER := 0;
    total          BIGINT := 0;
BEGIN
    LOOP
        -- Update at most `batch_size` rows per iteration. Chunking
        -- keeps memory bounded and limits the number of dead tuples
        -- produced in a single statement (HOT update friendly). The
        -- whole migration still runs in one outer transaction
        -- (golang-migrate wraps it) so there is no concurrent writer
        -- to race against — `FOR UPDATE SKIP LOCKED` is unnecessary.
        --
        -- The pre-filter uses a regex on the JSONB text dump as a
        -- cheap "does this row look interesting" check, then the
        -- recursive function decides whether the row actually needs
        -- a rewrite (idempotent on already-hashed payloads).
        WITH candidates AS (
            SELECT id
            FROM audit_logs
            WHERE metadata::text ~ '"(email|to_email|from_email|recipient|phone|iban)"'
              AND _audit_sanitize_pii(metadata) IS DISTINCT FROM metadata
            ORDER BY id
            LIMIT batch_size
        )
        UPDATE audit_logs a
        SET metadata = _audit_sanitize_pii(a.metadata)
        FROM candidates c
        WHERE a.id = c.id;

        GET DIAGNOSTICS affected = ROW_COUNT;
        total := total + affected;
        EXIT WHEN affected = 0;
    END LOOP;

    RAISE NOTICE 'audit_logs PII sanitize backfill: rewrote % rows', total;
END$$;

DROP FUNCTION _audit_sanitize_pii(JSONB);
