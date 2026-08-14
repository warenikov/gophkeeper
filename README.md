# GophKeeper

Клиент-серверный менеджер приватных данных: надёжное и безопасное хранение
логинов/паролей, произвольного текста, бинарных данных и банковских карт с
синхронизацией между несколькими устройствами одного владельца.

Выпускной проект курса Яндекс.Практикума по разработке на Go.

> **Статус:** Спринт 2 — все типы данных (логин/пароль, текст, карты, бинарные),
> команды add/update/get/delete, синхронизация между клиентами и офлайн-режим
> (read-only из кэша). Далее: envelope-шифрование, версионирование, релизы.

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

# синхронизация с другого устройства: тот же логин/пароль/мастер-пароль
./bin/gophkeeper-client login --login alice --password 'pass123'
./bin/gophkeeper-client list
```

Мастер-пароль можно не передавать через окружение — тогда клиент запросит его
интерактивно (без эха). Адрес сервера задаётся флагом `--address` или
переменной `GOPHKEEPER_ADDRESS` (по умолчанию `localhost:8081`).

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
