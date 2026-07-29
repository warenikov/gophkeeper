# Changelog

Все значимые изменения проекта фиксируются в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/),
проект придерживается [семантического версионирования](https://semver.org/lang/ru/).

## [Unreleased]

### Added
- Каркас проекта: Go-модуль `github.com/warenik/gophkeeper`, раскладка
  `cmd/{server,client}`, `internal/buildmeta`.
- Точки входа сервера (slog + graceful shutdown) и клиента (команда `version`).
- Проброс версии/даты сборки/commit через `-ldflags`.
- `Makefile`, конфиг `golangci-lint` v2, `docker-compose` с PostgreSQL 17.
- Лицензия MIT, CHANGELOG, CONTRIBUTING.
- Юнит-тесты пакета `buildmeta`.
