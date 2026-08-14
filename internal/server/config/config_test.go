package config_test

import (
	"testing"
	"time"

	"github.com/warenikov/gophkeeper/internal/server/config"
)

func TestLoadSuccessWithDefaults(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "postgres://localhost/db")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "0123456789abcdef0123456789abcdef")

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
	t.Setenv("GOPHKEEPER_JWT_SECRET", "0123456789abcdef0123456789abcdef")
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
	t.Setenv("GOPHKEEPER_JWT_SECRET", "0123456789abcdef0123456789abcdef")

	if _, err := config.Load(); err == nil {
		t.Error("ожидалась ошибка при отсутствии DATABASE_DSN")
	}
}

func TestTLSEnabled(t *testing.T) {
	if (config.Config{}).TLSEnabled() {
		t.Error("без сертификата TLS должен быть выключен")
	}
	full := config.Config{TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"}
	if !full.TLSEnabled() {
		t.Error("с сертификатом и ключом TLS должен быть включён")
	}
	partial := config.Config{TLSCertFile: "cert.pem"}
	if partial.TLSEnabled() {
		t.Error("только с сертификатом (без ключа) TLS должен быть выключен")
	}
}

func TestLoadShortJWTSecret(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "postgres://localhost/db")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "too-short")

	if _, err := config.Load(); err == nil {
		t.Error("ожидалась ошибка при слишком коротком JWT_SECRET")
	}
}

func TestLoadInvalidTTL(t *testing.T) {
	t.Setenv("GOPHKEEPER_DATABASE_DSN", "postgres://localhost/db")
	t.Setenv("GOPHKEEPER_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("GOPHKEEPER_TOKEN_TTL", "не-длительность")

	if _, err := config.Load(); err == nil {
		t.Error("ожидалась ошибка при некорректном TOKEN_TTL")
	}
}
