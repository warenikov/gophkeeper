// Пакет config загружает конфигурацию сервера GophKeeper из переменных
// окружения (12-factor).
package config

import (
	"errors"
	"os"
	"time"
)

// Префикс переменных окружения сервера.
const envPrefix = "GOPHKEEPER_"

// Config — конфигурация сервера.
type Config struct {
	// GRPCAddress — адрес, на котором слушает gRPC-сервер (например, ":8081").
	GRPCAddress string
	// DatabaseDSN — строка подключения к PostgreSQL.
	DatabaseDSN string
	// JWTSecret — секрет для подписи JWT.
	JWTSecret string //nolint:gosec // конфиг намеренно содержит секрет подписи JWT
	// TokenTTL — время жизни токена доступа.
	TokenTTL time.Duration
}

// Load читает конфигурацию из переменных окружения и валидирует обязательные
// поля (DATABASE_DSN и JWT_SECRET). Возвращает ошибку, если они не заданы.
func Load() (Config, error) {
	cfg := Config{
		GRPCAddress: env("GRPC_ADDRESS", ":8081"),
		DatabaseDSN: env("DATABASE_DSN", ""),
		JWTSecret:   env("JWT_SECRET", ""),
		TokenTTL:    24 * time.Hour,
	}

	if raw := env("TOKEN_TTL", ""); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, errors.New("config: invalid " + envPrefix + "TOKEN_TTL")
		}
		cfg.TokenTTL = ttl
	}

	if cfg.DatabaseDSN == "" {
		return Config{}, errors.New("config: " + envPrefix + "DATABASE_DSN is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("config: " + envPrefix + "JWT_SECRET is required")
	}

	return cfg, nil
}

// env возвращает значение переменной окружения с префиксом или значение по
// умолчанию, если она не задана.
func env(key, def string) string {
	if v, ok := os.LookupEnv(envPrefix + key); ok {
		return v
	}
	return def
}
