-- Roll back SEC-SESSIONS device + geo enrichment columns.

ALTER TABLE user_sessions
    DROP COLUMN IF EXISTS country_code,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS os,
    DROP COLUMN IF EXISTS browser,
    DROP COLUMN IF EXISTS device_label;
