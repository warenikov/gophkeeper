# Changelog

Все значимые изменения проекта фиксируются в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/),
проект придерживается [семантического версионирования](https://semver.org/lang/ru/).

## [Unreleased]

### Added (Спринт 3 — углубление)
- TUI (BubbleTea): интерактивный интерфейс `tui` — ввод мастер-пароля, список
  записей, просмотр с расшифровкой, удаление и обновление. Обёртка над клиентом.
- OTP/TOTP (RFC 6238): тип данных и клиентская генерация одноразовых кодов
  (`add otp`, код показывается в `get`); секрет хранится зашифрованным.
- Опциональный TLS: сервер (`GOPHKEEPER_TLS_CERT_FILE`/`_KEY_FILE`), клиент
  (`--tls-ca`).
- CI (GitHub Actions): линтинг, `go vet`, проверка `go mod tidy`, тесты с
  race/shuffle и сервисом PostgreSQL; матрица версий Go.
- Релизы: GoReleaser + workflow — кросс-сборка под Linux/macOS/Windows,
  публикация в GitHub Releases по тегу. Dependabot для gomod и actions.

### Fixed (по код-ревью)
- Клиент печатает ошибки команд в stderr (ранее терялись из-за `SilenceErrors`).
- Recovery-интерцептор сервера: паника в обработчике не роняет процесс.
- Логирование внутренних ошибок сервера (код + текст).
- Пароли/CVV читаются интерактивно без эха (не через флаги).
- Единый тримминг мастер-пароля (env и интерактив давали разные ключи).

### Added (Спринт 2 — полнота данных и синхронизация)
- Типы данных «банковская карта» и «бинарные данные» (из файла, с лимитом
  размера); команды `add card`, `add binary`, `update` (все типы), `get --out`.
- Синхронизация: тумбстоны (мягкое удаление) + монотонная ревизия, gRPC-метод
  `Sync`, локальный кэш клиента, команда `sync`.
- Офлайн-режим (read-only): `list` и `get` читают из кэша при недоступности
  сервера.
- Интеграционные тесты БД-слоя против реального PostgreSQL (пропуск без
  `GOPHKEEPER_TEST_DSN`); `make test-integration` / `cover-integration`.

### Added (Спринт 1 — MVP)
- gRPC-контракты Auth (Register/Login) и Secrets (CRUD) — proto как документация.
- Сервер: PostgreSQL (pgx v5), миграции при старте (golang-migrate + embed),
  JWT-аутентификация (HS256, защита от alg-confusion), bcrypt, gRPC-интерцептор
  авторизации; слои handler → service → repository.
- Клиент (Cobra): `register`, `login`, `logout`, `add login-password`,
  `add text`, `list`, `get`, `delete`, `version`.
- Клиентское шифрование zero-knowledge: Argon2id + AES-256-GCM; сервер хранит
  только шифртекст.
- Локальная сессия клиента; синхронизация между устройствами.
- `docker compose up` разворачивает PostgreSQL и сервер (проект `gophkeeper`).
- Тесты, включая end-to-end через bufconn; покрытие бизнес-логики ~81%.

### Added (Спринт 0 — фундамент)
- Каркас проекта: Go-модуль `github.com/warenikov/gophkeeper`, раскладка
  `cmd/{server,client}`, `internal/buildmeta`.
- Точки входа сервера (slog + graceful shutdown) и клиента (команда `version`).
- Проброс версии/даты сборки/commit через `-ldflags`.
- `Makefile`, конфиг `golangci-lint` v2, `docker-compose` с PostgreSQL 17.
- Лицензия MIT, CHANGELOG, CONTRIBUTING.
- Юнит-тесты пакета `buildmeta`.
