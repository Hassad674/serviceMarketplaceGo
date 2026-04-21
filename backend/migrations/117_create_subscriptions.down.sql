-- Down migration: drop the subscriptions table. Rolls back everything the
-- up migration created, including triggers and indexes (indexes are
-- dropped automatically with the table, the trigger too).
DROP TABLE IF EXISTS subscriptions;
