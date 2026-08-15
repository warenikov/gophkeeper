# GophKeeper

Клиент-серверный менеджер приватных данных: надёжное и безопасное хранение
логинов/паролей, произвольного текста, бинарных данных и банковских карт с
синхронизацией между несколькими устройствами одного владельца.

Выпускной проект курса Яндекс.Практикума по разработке на Go.

> **Статус:** Спринт 3 — все типы данных (логин/пароль, текст, карты, бинарные,
> OTP/TOTP), синхронизация и офлайн-режим, опциональный TLS, CI и релизы под
> Linux/macOS/Windows. Пароли вводятся интерактивно без эха.

## Ключевые свойства

- **Zero-knowledge.** Данные шифруются на клиенте (AES-256-GCM, ключ выводится
  из мастер-пароля через Argon2id). Сервер хранит только шифртекст и никогда не
  видит открытые данные.
- **gRPC** между клиентом и сервером; `.proto` — контракт и документация
  протокола.
- **PostgreSQL** как хранилище на сервере.
- **CLI-клиент** под Linux, macOS и Windows с выводом версии и даты сборки.
- Синхронизация между клиентами, метаданные к каждой записи.

## Технологический стек

Go · gRPC · PostgreSQL · AES-256 · JWT · bcrypt/Argon2id · Docker Compose

## Требования для разработки

- Go 1.25+
- Docker + Docker Compose
- `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc` (для генерации из `.proto`)
- `golangci-lint` v2, `migrate` (golang-migrate)

## Быстрый старт

Развернуть сервер и БД одной командой:

```bash
cp .env.example .env
make up          # PostgreSQL + сервер (миграции применяются автоматически)
make build       # собрать CLI-клиент в ./bin
```

Базовый сценарий работы с клиентом:

```bash
# регистрация нового пользователя (создаёт локальную сессию)
./bin/gophkeeper-client register --login alice --password 'pass123'

# добавить пару логин/пароль (мастер-пароль шифрует данные на клиенте)
export GOPHKEEPER_MASTER_PASSWORD='мой-мастер-пароль'
./bin/gophkeeper-client add login-password \
  --name "GitHub" --login alice --password 's3cret' --meta "github.com"

# добавить произвольный текст
./bin/gophkeeper-client add text --name "Заметка" --text "секретный текст"

# TOTP-секрет (одноразовые пароли); get покажет текущий код и остаток времени
./bin/gophkeeper-client add otp --name "GitHub 2FA" --secret JBSWY3DPEHPK3PXP --issuer GitHub

# банковская карта и бинарные данные из файла
./bin/gophkeeper-client add card --name "Visa" --number 4111111111111111 \
  --holder "ALICE" --expiry 12/29 --cvv 123 --meta "Bank"
./bin/gophkeeper-client add binary --name "Ключ" --file ./id_rsa

# список записей (без расшифровки)
./bin/gophkeeper-client list

# получить и расшифровать запись (бинарные — с сохранением в файл)
./bin/gophkeeper-client get <id>
./bin/gophkeeper-client get <id> --out ./restored

# обновить запись (версия подставляется автоматически)
./bin/gophkeeper-client update text --id <id> --name "Заметка" --text "новый текст"

# синхронизировать локальный кэш (нужно для офлайн-чтения)
./bin/gophkeeper-client sync

# интерактивный терминальный интерфейс (список, просмотр, удаление)
./bin/gophkeeper-client tui

# синхронизация с другого устройства: тот же логин/пароль/мастер-пароль
./bin/gophkeeper-client login --login alice --password 'pass123'
./bin/gophkeeper-client list
```

Пароли, CVV и мастер-пароль можно не передавать флагами — тогда клиент запросит
их интерактивно (без эха), чтобы они не попали в history/`ps`. Адрес сервера —
флаг `--address` или `GOPHKEEPER_ADDRESS` (по умолчанию `localhost:8081`).

**TLS (опционально).** Сервер включает TLS при заданных
`GOPHKEEPER_TLS_CERT_FILE` и `GOPHKEEPER_TLS_KEY_FILE`; клиент доверяет
корневому сертификату через `--tls-ca` (или `GOPHKEEPER_TLS_CA`). Без них
используется незащищённое соединение (локальная разработка).

**Релизы.** Готовые бинарники под Linux/macOS/Windows публикуются в GitHub
Releases (GoReleaser) по тегу `vX.Y.Z`.

## Основные команды Make

| Команда         | Действие                                        |
|-----------------|-------------------------------------------------|
| `make build`    | Сборка сервера и клиента в `./bin`              |
| `make test`     | Юнит-тесты с race-детектором                    |
| `make cover`    | Тесты + отчёт о покрытии                         |
| `make lint`     | Запуск golangci-lint                            |
| `make up` / `make down` | Поднять/остановить локальный стек       |
| `make help`     | Полный список целей                             |

## Структура проекта

```
cmd/            точки входа: server, client
internal/       внутренние пакеты (buildmeta, далее — server/client/…)
proto/          gRPC-контракты (.proto) — Спринт 1
migrations/     SQL-миграции БД — Спринт 1
deployments/    docker-compose и связанное окружение
```
