package service

import (
	"context"
	"fmt"

	"github.com/warenik/gophkeeper/internal/server/model"
)

// secretStore — доступ к записям, необходимый сервису.
type secretStore interface {
	Create(ctx context.Context, s model.Secret) (model.Secret, error)
	ListByOwner(ctx context.Context, ownerID string) ([]model.Secret, error)
	GetByID(ctx context.Context, ownerID, id string) (model.Secret, error)
	Update(ctx context.Context, s model.Secret, expectedVersion int64) (model.Secret, error)
	Delete(ctx context.Context, ownerID, id string) error
	SyncByOwner(ctx context.Context, ownerID string, sinceRevision int64) ([]model.Secret, int64, error)
}

// Secret реализует операции над приватными записями.
type Secret struct {
	store secretStore
}

// NewSecret создаёт сервис записей.
func NewSecret(store secretStore) *Secret {
	return &Secret{store: store}
}

// Create создаёт новую запись владельца. Валидирует тип, имя и наличие
// зашифрованной нагрузки.
func (s *Secret) Create(ctx context.Context, ownerID string, secret model.Secret) (model.Secret, error) {
	if err := validate(secret); err != nil {
		return model.Secret{}, err
	}
	secret.OwnerID = ownerID
	return s.store.Create(ctx, secret)
}

// List возвращает все записи владельца.
func (s *Secret) List(ctx context.Context, ownerID string) ([]model.Secret, error) {
	return s.store.ListByOwner(ctx, ownerID)
}

// Get возвращает запись владельца по идентификатору.
func (s *Secret) Get(ctx context.Context, ownerID, id string) (model.Secret, error) {
	if id == "" {
		return model.Secret{}, fmt.Errorf("%w: id is required", model.ErrValidation)
	}
	return s.store.GetByID(ctx, ownerID, id)
}

// Update обновляет запись владельца при совпадении ожидаемой версии.
func (s *Secret) Update(ctx context.Context, ownerID string, secret model.Secret, expectedVersion int64) (model.Secret, error) {
	if secret.ID == "" {
		return model.Secret{}, fmt.Errorf("%w: id is required", model.ErrValidation)
	}
	// Тип записи при обновлении не меняется, поэтому проверяем только контент.
	if err := validateContent(secret); err != nil {
		return model.Secret{}, err
	}
	secret.OwnerID = ownerID
	return s.store.Update(ctx, secret, expectedVersion)
}

// Delete удаляет запись владельца.
func (s *Secret) Delete(ctx context.Context, ownerID, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", model.ErrValidation)
	}
	return s.store.Delete(ctx, ownerID, id)
}

// Sync возвращает изменения владельца с указанной ревизии (включая тумбстоны) и
// новый курсор для следующей синхронизации.
func (s *Secret) Sync(ctx context.Context, ownerID string, sinceRevision int64) ([]model.Secret, int64, error) {
	return s.store.SyncByOwner(ctx, ownerID, sinceRevision)
}

// validate проверяет тип и содержимое записи (используется при создании).
func validate(secret model.Secret) error {
	switch secret.Type {
	case model.SecretTypeLoginPassword, model.SecretTypeText,
		model.SecretTypeCard, model.SecretTypeBinary:
		// поддерживаемые типы
	default:
		return fmt.Errorf("%w: unsupported secret type", model.ErrValidation)
	}
	return validateContent(secret)
}

// validateContent проверяет обязательные имя и зашифрованную нагрузку записи
// (используется при обновлении, когда тип не меняется).
func validateContent(secret model.Secret) error {
	if secret.Name == "" {
		return fmt.Errorf("%w: name is required", model.ErrValidation)
	}
	if len(secret.EncryptedPayload) == 0 {
		return fmt.Errorf("%w: encrypted payload is required", model.ErrValidation)
	}
	return nil
}
