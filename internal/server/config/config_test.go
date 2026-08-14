package config_test

import (
	"testing"
	"time"

	"github.com/warenik/gophkeeper/internal/server/config"
)

func TestLoadSuccessWithDefaults(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "postgres://localhost/db")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GRPCAddress != ":8081" {
		t.Errorf("GRPCAddress = %q, ожидалось :8081", cfg.GRPCAddress)
	}
	if cfg.TokenTTL != 24*time.Hour {
		t.Errorf("TokenTTL = %s, ожидалось 24h", cfg.TokenTTL)
	}
}

func TestLoadCustomValues(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "postgres://localhost/db")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "secret")
	t.Setenv("GOPHKEEPER_GRPC_ADDRESS", ":9000")
	t.Setenv("GOPHKEEPER_TOKEN_TTL", "1h")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GRPCAddress != ":9000" {
		t.Errorf("GRPCAddress = %q, ожидалось :9000", cfg.GRPCAddress)
	}
	if cfg.TokenTTL != time.Hour {
		t.Errorf("TokenTTL = %s, ожидалось 1h", cfg.TokenTTL)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "secret")

	if _, err := config.Load(); err == nil {
		t.Error("ожидалась ошибка при отсутствии DATABASE_DSN")
	}
}

func TestLoadInvalidTTL(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "postgres://localhost/db")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "secret")
	t.Setenv("GOPHKEEPER_TOKEN_TTL", "не-длительность")

	if _, err := config.Load(); err == nil {
		t.Error("ожидалась ошибка при некорректном TOKEN_TTL")
	}
}
