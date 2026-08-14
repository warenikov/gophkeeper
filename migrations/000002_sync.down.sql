-- Откат синхронизации.

DROP INDEX IF EXISTS idx_secrets_owner_revision;

ALTER TABLE secrets
    DROP COLUMN IF EXISTS revision,
    DROP COLUMN IF EXISTS deleted;

DROP SEQUENCE IF EXISTS secrets_revision_seq;
