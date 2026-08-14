package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/warenikov/gophkeeper/internal/server/model"
	"github.com/warenikov/gophkeeper/internal/server/service"
)

// mockSecretStore — конфигурируемая заглушка хранилища записей.
type mockSecretStore struct {
	createFn func(ctx context.Context, s model.Secret) (model.Secret, error)
	listFn   func(ctx context.Context, ownerID string) ([]model.Secret, error)
	getFn    func(ctx context.Context, ownerID, id string) (model.Secret, error)
	updateFn func(ctx context.Context, s model.Secret, expected int64) (model.Secret, error)
	deleteFn func(ctx context.Context, ownerID, id string) error
	syncFn   func(ctx context.Context, ownerID string, since int64) ([]model.Secret, int64, error)
}

func (m mockSecretStore) Create(ctx context.Context, s model.Secret) (model.Secret, error) {
	return m.createFn(ctx, s)
}

func (m mockSecretStore) ListByOwner(ctx context.Context, ownerID string) ([]model.Secret, error) {
	return m.listFn(ctx, ownerID)
}

func (m mockSecretStore) GetByID(ctx context.Context, ownerID, id string) (model.Secret, error) {
	return m.getFn(ctx, ownerID, id)
}

func (m mockSecretStore) Update(ctx context.Context, s model.Secret, expected int64) (model.Secret, error) {
	return m.updateFn(ctx, s, expected)
}

func (m mockSecretStore) Delete(ctx context.Context, ownerID, id string) error {
	return m.deleteFn(ctx, ownerID, id)
}

func (m mockSecretStore) SyncByOwner(ctx context.Context, ownerID string, since int64) ([]model.Secret, int64, error) {
	return m.syncFn(ctx, ownerID, since)
}

// validSecret возвращает корректную запись для тестов создания.
func validSecret() model.Secret {
	return model.Secret{
		Type:             model.SecretTypeText,
		Name:             "note",
		EncryptedPayload: []byte("cipher"),
	}
}

func TestSecretCreate(t *testing.T) {
	store := mockSecretStore{
		createFn: func(_ context.Context, s model.Secret) (model.Secret, error) {
			if s.OwnerID != "owner-1" {
				t.Errorf("ownerID = %q, ожидалось owner-1", s.OwnerID)
			}
			s.ID = "s1"
			return s, nil
		},
	}
	svc := service.NewSecret(store)

	got, err := svc.Create(context.Background(), "owner-1", validSecret())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "s1" {
		t.Errorf("ID = %q, ожидалось s1", got.ID)
	}
}

func TestSecretCreateValidation(t *testing.T) {
	svc := service.NewSecret(mockSecretStore{})

	cases := map[string]model.Secret{
		"неизвестный тип": {Type: model.SecretTypeUnspecified, Name: "n", EncryptedPayload: []byte("x")},
		"пустое имя":      {Type: model.SecretTypeText, Name: "", EncryptedPayload: []byte("x")},
		"пустая нагрузка": {Type: model.SecretTypeText, Name: "n", EncryptedPayload: nil},
	}
	for name, secret := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Create(context.Background(), "owner-1", secret); !errors.Is(err, model.ErrValidation) {
				t.Errorf("ожидалась ErrValidation, получено %v", err)
			}
		})
	}
}

func TestSecretList(t *testing.T) {
	store := mockSecretStore{
		listFn: func(context.Context, string) ([]model.Secret, error) {
			return []model.Secret{{ID: "s1"}, {ID: "s2"}}, nil
		},
	}
	got, err := service.NewSecret(store).List(context.Background(), "owner-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("получено %d записей, ожидалось 2", len(got))
	}
}

func TestSecretGet(t *testing.T) {
	store := mockSecretStore{
		getFn: func(_ context.Context, _, id string) (model.Secret, error) {
			return model.Secret{ID: id}, nil
		},
	}
	svc := service.NewSecret(store)

	if _, err := svc.Get(context.Background(), "owner-1", ""); !errors.Is(err, model.ErrValidation) {
		t.Errorf("пустой id: ожидалась ErrValidation, получено %v", err)
	}

	got, err := svc.Get(context.Background(), "owner-1", "s1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "s1" {
		t.Errorf("ID = %q, ожидалось s1", got.ID)
	}
}

func TestSecretUpdate(t *testing.T) {
	store := mockSecretStore{
		updateFn: func(_ context.Context, s model.Secret, expected int64) (model.Secret, error) {
			if expected != 3 {
				t.Errorf("expected version = %d, ожидалось 3", expected)
			}
			s.Version = expected + 1
			return s, nil
		},
	}
	svc := service.NewSecret(store)

	secret := validSecret()
	secret.ID = "s1"
	got, err := svc.Update(context.Background(), "owner-1", secret, 3)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Version != 4 {
		t.Errorf("version = %d, ожидалось 4", got.Version)
	}
}

func TestSecretUpdateValidation(t *testing.T) {
	svc := service.NewSecret(mockSecretStore{})

	// Пустой id.
	if _, err := svc.Update(context.Background(), "owner-1", validSecret(), 1); !errors.Is(err, model.ErrValidation) {
		t.Errorf("пустой id: ожидалась ErrValidation, получено %v", err)
	}
	// Пустая нагрузка при заданном id.
	if _, err := svc.Update(context.Background(), "owner-1", model.Secret{ID: "s1", Name: "n"}, 1); !errors.Is(err, model.ErrValidation) {
		t.Errorf("пустая нагрузка: ожидалась ErrValidation, получено %v", err)
	}
}

func TestSecretDelete(t *testing.T) {
	var called bool
	store := mockSecretStore{
		deleteFn: func(context.Context, string, string) error {
			called = true
			return nil
		},
	}
	svc := service.NewSecret(store)

	if err := svc.Delete(context.Background(), "owner-1", ""); !errors.Is(err, model.ErrValidation) {
		t.Errorf("пустой id: ожидалась ErrValidation, получено %v", err)
	}
	if err := svc.Delete(context.Background(), "owner-1", "s1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !called {
		t.Error("хранилище Delete не вызвано")
	}
}
