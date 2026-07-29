# Участие в разработке

Спасибо за интерес к GophKeeper. Ниже — как быстро развернуть окружение и внести
изменения.

## Требования

- Go 1.25+
- Docker + Docker Compose
- `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc` (генерация из `.proto`)
- `golangci-lint` v2, `migrate` (golang-migrate)

## Быстрый старт

```bash
git clone https://github.com/warenik/gophkeeper.git
cd gophkeeper
cp .env.example .env
make up        # поднять инфраструктуру (PostgreSQL)
make build     # собрать бинарники
make test      # прогнать тесты
```

## Перед коммитом

Обязательно прогоните:

```bash
make fmt       # форматирование
make lint      # golangci-lint без замечаний
make cover     # тесты + покрытие (цель ≥ 70–80%)
```

## Стиль и соглашения

- Комментарии в коде — на русском, godoc на каждой экспортируемой сущности.
- Ранние возвраты, обработка ошибок с `%w`, слоистая архитектура
  (handler → service → repository).
- Именование по конвенциям Go (MixedCaps, без статтеринга, `Err`-префикс у
  sentinel-ошибок).

## Сообщения коммитов

Кратко и по делу, в настоящем времени. Например:
`Спринт 1: добавлен gRPC-сервис Auth`.
