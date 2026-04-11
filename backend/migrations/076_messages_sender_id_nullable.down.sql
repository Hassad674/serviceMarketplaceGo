BEGIN;

-- Restore NOT NULL on messages.sender_id. This rollback is only safe when
-- no rows with sender_id = NULL exist — such rows are the footprint of a
-- hard-deleted user whose message history was preserved. Forcing NOT NULL
-- back on a table that already contains deleted-user history would either
-- destroy that history or be impossible (Postgres refuses the ALTER
-- because of the existing NULLs). We guard the migration with an explicit
-- check so the failure mode is a clear, actionable error message instead
-- of an opaque NOT NULL violation.
DO $$
DECLARE
    null_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO null_count FROM messages WHERE sender_id IS NULL;
    IF null_count > 0 THEN
        RAISE EXCEPTION
            'cannot restore NOT NULL on messages.sender_id: % rows have sender_id = NULL (preserved history of hard-deleted users). Restore those rows first or accept the data loss before rolling this migration back.',
            null_count;
    END IF;
END $$;

ALTER TABLE messages ALTER COLUMN sender_id SET NOT NULL;

COMMIT;
