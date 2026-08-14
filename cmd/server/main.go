// Команда server — сервер GophKeeper.
//
// Он предоставляет gRPC-API для регистрации/аутентификации пользователей и для
// хранения их приватных данных (зашифрованных на клиенте) в PostgreSQL. Сервер
// работает по модели zero-knowledge: он никогда не видит данные в открытом
// виде.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc/credentials"

	"github.com/warenik/gophkeeper/internal/buildmeta"
	"github.com/warenik/gophkeeper/internal/server/auth"
	"github.com/warenik/gophkeeper/internal/server/config"
	"github.com/warenik/gophkeeper/internal/server/grpcserver"
	"github.com/warenik/gophkeeper/internal/server/service"
	"github.com/warenik/gophkeeper/internal/server/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("сервер завершился с ошибкой", "err", err)
		os.Exit(1)
	}
}

// run загружает конфигурацию, применяет миграции, собирает зависимости
// (ручной constructor injection) и запускает gRPC-сервер до сигнала завершения.
func run(logger *slog.Logger) error {
	logger.Info("запуск сервера gophkeeper",
		"version", buildmeta.Version,
		"build_date", buildmeta.BuildDate,
		"commit", buildmeta.Commit,
	)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := postgres.Migrate(cfg.DatabaseDSN); err != nil {
		return err
	}
	logger.Info("миграции применены")

	pool, err := postgres.NewPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	secretRepo := postgres.NewSecretRepository(pool)

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.TokenTTL)
	authSvc := service.NewAuth(userRepo, tokens)
	secretSvc := service.NewSecret(secretRepo)

	authHandler := grpcserver.NewAuthHandler(authSvc)
	secretHandler := grpcserver.NewSecretHandler(secretSvc)

	var creds credentials.TransportCredentials
	if cfg.TLSEnabled() {
		creds, err = credentials.NewServerTLSFromFile(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return err
		}
		logger.Info("TLS включён")
	} else {
		logger.Warn("TLS выключен — используется незащищённое соединение (только для локальной разработки)")
	}

	srv := grpcserver.New(cfg.GRPCAddress, tokens, authHandler, secretHandler, logger, creds)
	return srv.Run(ctx)
}
