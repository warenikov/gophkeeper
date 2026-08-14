# Changelog

Все значимые изменения проекта фиксируются в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/),
проект придерживается [семантического версионирования](https://semver.org/lang/ru/).

## [Unreleased]

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
- Каркас проекта: Go-модуль `github.com/warenik/gophkeeper`, раскладка
  `cmd/{server,client}`, `internal/buildmeta`.
- Точки входа сервера (slog + graceful shutdown) и клиента (команда `version`).
- Проброс версии/даты сборки/commit через `-ldflags`.
- `Makefile`, конфиг `golangci-lint` v2, `docker-compose` с PostgreSQL 17.
- Лицензия MIT, CHANGELOG, CONTRIBUTING.
- Юнит-тесты пакета `buildmeta`.
