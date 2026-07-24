# proto

gRPC-контракты (`.proto`) для GophKeeper. Служат одновременно документацией
бинарного протокола взаимодействия клиента и сервера.

Добавляются в Спринте 1: сервисы `Auth` (Register/Login) и `Secrets`
(Create/List/Get/Update/Delete). Генерация Go-кода — через `protoc` с плагинами
`protoc-gen-go` и `protoc-gen-go-grpc`.
