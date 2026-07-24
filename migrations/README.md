# migrations

SQL-миграции схемы PostgreSQL (формат golang-migrate: пары файлов
`NNNNNN_name.up.sql` / `NNNNNN_name.down.sql`).

Добавляются в Спринте 1: таблицы пользователей и записей (secrets) с полями для
версионирования и синхронизации.
