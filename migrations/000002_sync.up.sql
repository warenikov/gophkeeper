-- Синхронизация: тумбстоны (мягкое удаление) и монотонная ревизия-курсор.

-- Последовательность-ревизия увеличивается при каждом создании и обновлении
-- записи, задавая глобальный монотонный порядок изменений для pull-синхронизации.
CREATE SEQUENCE IF NOT EXISTS secrets_revision_seq;

ALTER TABLE secrets
    ADD COLUMN IF NOT EXISTS deleted  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS revision BIGINT  NOT NULL DEFAULT nextval('secrets_revision_seq');

CREATE INDEX IF NOT EXISTS idx_secrets_owner_revision ON secrets (owner_id, revision);
