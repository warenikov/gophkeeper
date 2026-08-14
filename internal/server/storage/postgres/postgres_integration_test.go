package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/warenik/gophkeeper/internal/server/model"
	"github.com/warenik/gophkeeper/internal/server/storage/postgres"
)

// testPool применяет миграции, открывает пул и очищает таблицы. Тест
// пропускается, если не задан GOPHKEEPER_TEST_DSN (например, при обычном
// `go test` без запущенной БД).
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("GOPHKEEPER_TEST_DSN")
	if dsn == "" {
		t.Skip("GOPHKEEPER_TEST_DSN не задан — интеграционные тесты БД пропущены")
	}

	if err := postgres.Migrate(dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	pool, err := postgres.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(context.Background(), "TRUNCATE users, secrets CASCADE"); err != nil {
		t.Fatalf("TRUNCATE: %v", err)
	}
	return pool
}

func TestUserRepository(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	user, err := repo.Create(ctx, "alice", "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == "" || user.CreatedAt.IsZero() {
		t.Errorf("сервер не проставил id/created_at: %+v", user)
	}

	// Повторный логин → ErrUserExists.
	if _, err := repo.Create(ctx, "alice", "hash"); !errors.Is(err, model.ErrUserExists) {
		t.Errorf("ожидалась ErrUserExists, получено %v", err)
	}

	got, err := repo.GetByLogin(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if got.ID != user.ID || got.PasswordHash != "hash" {
		t.Errorf("получен неверный пользователь: %+v", got)
	}

	if _, err := repo.GetByLogin(ctx, "ghost"); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено %v", err)
	}
}

func TestSecretRepository(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := postgres.NewUserRepository(pool)
	secrets := postgres.NewSecretRepository(pool)

	owner, err := users.Create(ctx, "bob", "hash")
	if err != nil {
		t.Fatalf("создание владельца: %v", err)
	}

	created, err := secrets.Create(ctx, model.Secret{
		OwnerID:          owner.ID,
		Type:             model.SecretTypeText,
		Name:             "note",
		Metadata:         "meta",
		EncryptedPayload: []byte("cipher"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.Version != 1 {
		t.Errorf("ожидались id и version=1, получено %+v", created)
	}

	// GetByID.
	got, err := secrets.GetByID(ctx, owner.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Type != model.SecretTypeText || string(got.EncryptedPayload) != "cipher" {
		t.Errorf("получена неверная запись: %+v", got)
	}

	// Чужой владелец не видит запись.
	if _, err := secrets.GetByID(ctx, "00000000-0000-0000-0000-000000000000", created.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("чужой владелец: ожидалась ErrNotFound, получено %v", err)
	}

	// ListByOwner.
	list, err := secrets.ListByOwner(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListByOwner: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("в списке %d записей, ожидалась 1", len(list))
	}

	// Update с верной версией.
	created.Name = "note2"
	created.EncryptedPayload = []byte("cipher2")
	updated, err := secrets.Update(ctx, created, 1)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Version != 2 || updated.Name != "note2" {
		t.Errorf("после обновления: %+v", updated)
	}

	// Update с устаревшей версией → конфликт.
	if _, err := secrets.Update(ctx, created, 1); !errors.Is(err, model.ErrVersionConflict) {
		t.Errorf("ожидалась ErrVersionConflict, получено %v", err)
	}

	// Update несуществующей записи → ErrNotFound.
	missing := model.Secret{ID: "11111111-1111-1111-1111-111111111111", OwnerID: owner.ID, Name: "x", EncryptedPayload: []byte("y")}
	if _, err := secrets.Update(ctx, missing, 1); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено %v", err)
	}

	// Delete.
	if err := secrets.Delete(ctx, owner.ID, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := secrets.Delete(ctx, owner.ID, created.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("повторное удаление: ожидалась ErrNotFound, получено %v", err)
	}
}
