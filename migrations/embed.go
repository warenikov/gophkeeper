// Пакет migrations встраивает SQL-файлы миграций в бинарник, чтобы сервер мог
// применять схему БД при старте без внешних файлов.
package migrations

import "embed"

// FS содержит встроенные файлы миграций (формат golang-migrate).
//
//go:embed *.sql
var FS embed.FS
