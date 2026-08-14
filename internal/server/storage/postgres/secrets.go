package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/warenik/gophkeeper/internal/server/model"
)

// SecretRepository — доступ к таблице приватных записей.
type SecretRepository struct {
	pool *pgxpool.Pool
}

// NewSecretRepository создаёт репозиторий записей.
func NewSecretRepository(pool *pgxpool.Pool) *SecretRepository {
	return &SecretRepository{pool: pool}
}

// Create добавляет запись и возвращает её с проставленными сервером id,
// version и временными метками.
func (r *SecretRepository) Create(ctx context.Context, s model.Secret) (model.Secret, error) {
	const q = `
		INSERT INTO secrets (owner_id, type, name, metadata, encrypted_payload)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING id::text, version, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		s.OwnerID, int16(s.Type), s.Name, s.Metadata, s.EncryptedPayload,
	).Scan(&s.ID, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return model.Secret{}, fmt.Errorf("insert secret: %w", err)
	}

	return s, nil
}

// ListByOwner возвращает все записи владельца, отсортированные по времени
// создания.
func (r *SecretRepository) ListByOwner(ctx context.Context, ownerID string) ([]model.Secret, error) {
	const q = `
		SELECT id::text, owner_id::text, type, name, metadata, encrypted_payload,
		       version, created_at, updated_at
		FROM secrets
		WHERE owner_id = $1::uuid
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query secrets: %w", err)
	}
	defer rows.Close()

	secrets := []model.Secret{}
	for rows.Next() {
		s, scanErr := scanSecret(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan secret: %w", scanErr)
		}
		secrets = append(secrets, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}

	return secrets, nil
}

// GetByID возвращает запись владельца по идентификатору или model.ErrNotFound.
func (r *SecretRepository) GetByID(ctx context.Context, ownerID, id string) (model.Secret, error) {
	const q = `
		SELECT id::text, owner_id::text, type, name, metadata, encrypted_payload,
		       version, created_at, updated_at
		FROM secrets
		WHERE id = $1::uuid AND owner_id = $2::uuid`

	s, err := scanSecret(r.pool.QueryRow(ctx, q, id, ownerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Secret{}, model.ErrNotFound
		}
		return model.Secret{}, fmt.Errorf("get secret: %w", err)
	}

	return s, nil
}

// Update обновляет запись при совпадении ожидаемой версии (оптимистичная
// блокировка). Возвращает model.ErrVersionConflict при рассогласовании версий
// и model.ErrNotFound, если запись отсутствует.
func (r *SecretRepository) Update(ctx context.Context, s model.Secret, expectedVersion int64) (model.Secret, error) {
	const q = `
		UPDATE secrets
		SET name = $1, metadata = $2, encrypted_payload = $3,
		    version = version + 1, updated_at = now()
		WHERE id = $4::uuid AND owner_id = $5::uuid AND version = $6
		RETURNING id::text, owner_id::text, type, name, metadata, encrypted_payload,
		          version, created_at, updated_at`

	updated, err := scanSecret(r.pool.QueryRow(ctx, q,
		s.Name, s.Metadata, s.EncryptedPayload, s.ID, s.OwnerID, expectedVersion,
	))
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.Secret{}, fmt.Errorf("update secret: %w", err)
		}
		// Строк не затронуто: либо запись отсутствует, либо конфликт версий.
		exists, existsErr := r.exists(ctx, s.OwnerID, s.ID)
		if existsErr != nil {
			return model.Secret{}, existsErr
		}
		if exists {
			return model.Secret{}, model.ErrVersionConflict
		}
		return model.Secret{}, model.ErrNotFound
	}

	return updated, nil
}

// Delete удаляет запись владельца. Возвращает model.ErrNotFound, если записи
// нет.
func (r *SecretRepository) Delete(ctx context.Context, ownerID, id string) error {
	const q = `DELETE FROM secrets WHERE id = $1::uuid AND owner_id = $2::uuid`

	tag, err := r.pool.Exec(ctx, q, id, ownerID)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	return nil
}

// exists сообщает, есть ли запись с данным id у владельца.
func (r *SecretRepository) exists(ctx context.Context, ownerID, id string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM secrets WHERE id = $1::uuid AND owner_id = $2::uuid)`

	var ok bool
	if err := r.pool.QueryRow(ctx, q, id, ownerID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check secret exists: %w", err)
	}
	return ok, nil
}

// scanRow — общий интерфейс pgx.Row и pgx.Rows для сканирования одной строки.
type scanRow interface {
	Scan(dest ...any) error
}

// scanSecret считывает одну запись из строки результата.
func scanSecret(row scanRow) (model.Secret, error) {
	var (
		s   model.Secret
		typ int16
	)
	err := row.Scan(
		&s.ID, &s.OwnerID, &typ, &s.Name, &s.Metadata, &s.EncryptedPayload,
		&s.Version, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return model.Secret{}, err
	}
	s.Type = model.SecretType(typ)
	return s, nil
}
