-- Откат начальной схемы GophKeeper.

DROP INDEX IF EXISTS idx_secrets_owner;
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS users;
