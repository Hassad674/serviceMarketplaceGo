-- 080_create_org_expertise_domains.down.sql
--
-- Reverts migration 080: drops the organization_expertise_domains
-- table and its indexes. Any declared expertise is lost — there is
-- no other place where this data lives.

DROP INDEX IF EXISTS idx_org_expertise_domain;
DROP INDEX IF EXISTS idx_org_expertise_position;
DROP TABLE IF EXISTS organization_expertise_domains;
