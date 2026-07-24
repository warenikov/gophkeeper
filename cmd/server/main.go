// Команда server — сервер GophKeeper.
//
// Он предоставляет gRPC-API для регистрации/аутентификации пользователей и для
// хранения их приватных данных (зашифрованных на клиенте) в PostgreSQL. Сервер
// работает по модели zero-knowledge: он никогда не видит данные в открытом
// виде.
//
// Пока файл содержит точку входа и инициализацию структурированного
// логирования; gRPC-сервер, хранилище и аутентификация появятся в Спринте 1.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/warenik/gophkeeper/internal/buildmeta"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("запуск сервера gophkeeper",
		"version", buildmeta.Version,
		"build_date", buildmeta.BuildDate,
		"commit", buildmeta.Commit,
	)

	// Ожидаем сигнал завершения. Собственно gRPC-сервер подключается в
	// Спринте 1; пока это проверяет каркас процесса/логирования/сигналов.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("получен сигнал завершения, выходим")
}
