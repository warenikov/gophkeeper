package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/warenik/gophkeeper/internal/server/model"
)

// secretColumns — список колонок записи в порядке, ожидаемом scanSecret.
//
//nolint:gosec // G101: это список колонок SQL, а не учётные данные
const secretColumns = `id::text, owner_id::text, type, name, metadata,
	encrypted_payload, version, created_at, updated_at, deleted, revision`

// SecretRepository — доступ к таблице приватных записей.
type SecretRepository struct {
	pool *pgxpool.Pool
}

// NewSecretRepository создаёт репозиторий записей.
func NewSecretRepository(pool *pgxpool.Pool) *SecretRepository {
	return &SecretRepository{pool: pool}
}

// Create добавляет запись и возвращает её с проставленными сервером id,
// version, revision и временными метками.
func (r *SecretRepository) Create(ctx context.Context, s model.Secret) (model.Secret, error) {
	const q = `
		INSERT INTO secrets (owner_id, type, name, metadata, encrypted_payload)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING id::text, version, revision, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		s.OwnerID, int16(s.Type), s.Name, s.Metadata, s.EncryptedPayload,
	).Scan(&s.ID, &s.Version, &s.Revision, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return model.Secret{}, fmt.Errorf("insert secret: %w", err)
	}

	return s, nil
}

// ListByOwner возвращает активные (не удалённые) записи владельца.
func (r *SecretRepository) ListByOwner(ctx context.Context, ownerID string) ([]model.Secret, error) {
	q := `SELECT ` + secretColumns + `
		FROM secrets
		WHERE owner_id = $1::uuid AND deleted = false
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query secrets: %w", err)
	}
	defer rows.Close()

	secrets, err := collectSecrets(rows)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// GetByID возвращает активную запись владельца или model.ErrNotFound.
func (r *SecretRepository) GetByID(ctx context.Context, ownerID, id string) (model.Secret, error) {
	q := `SELECT ` + secretColumns + `
		FROM secrets
		WHERE id = $1::uuid AND owner_id = $2::uuid AND deleted = false`

	s, err := scanSecret(r.pool.QueryRow(ctx, q, id, ownerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Secret{}, model.ErrNotFound
		}
		return model.Secret{}, fmt.Errorf("get secret: %w", err)
	}
	return s, nil
}

// Update обновляет активную запись при совпадении ожидаемой версии. Возвращает
// model.ErrVersionConflict при рассогласовании версий и model.ErrNotFound, если
// записи нет или она удалена.
func (r *SecretRepository) Update(ctx context.Context, s model.Secret, expectedVersion int64) (model.Secret, error) {
	q := `
		UPDATE secrets
		SET name = $1, metadata = $2, encrypted_payload = $3,
		    version = version + 1, revision = nextval('secrets_revision_seq'),
		    updated_at = now()
		WHERE id = $4::uuid AND owner_id = $5::uuid AND version = $6 AND deleted = false
		RETURNING ` + secretColumns

	updated, err := scanSecret(r.pool.QueryRow(ctx, q,
		s.Name, s.Metadata, s.EncryptedPayload, s.ID, s.OwnerID, expectedVersion,
	))
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.Secret{}, fmt.Errorf("update secret: %w", err)
		}
		// Строк не затронуто: либо записи нет, либо конфликт версий.
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

// Delete помечает запись владельца тумбстоном (мягкое удаление) и повышает
// ревизию, чтобы удаление синхронизировалось с другими клиентами. Возвращает
// model.ErrNotFound, если активной записи нет.
func (r *SecretRepository) Delete(ctx context.Context, ownerID, id string) error {
	const q = `
		UPDATE secrets
		SET deleted = true, version = version + 1,
		    revision = nextval('secrets_revision_seq'), updated_at = now()
		WHERE id = $1::uuid AND owner_id = $2::uuid AND deleted = false`

	tag, err := r.pool.Exec(ctx, q, id, ownerID)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	return nil
}

// SyncByOwner возвращает все записи владельца (включая тумбстоны), у которых
// revision больше sinceRevision, в порядке возрастания ревизии, а также новый
// курсор (максимальную ревизию среди возвращённых или sinceRevision, если
// изменений нет).
func (r *SecretRepository) SyncByOwner(ctx context.Context, ownerID string, sinceRevision int64) ([]model.Secret, int64, error) {
	q := `SELECT ` + secretColumns + `
		FROM secrets
		WHERE owner_id = $1::uuid AND revision > $2
		ORDER BY revision`

	rows, err := r.pool.Query(ctx, q, ownerID, sinceRevision)
	if err != nil {
		return nil, 0, fmt.Errorf("sync secrets: %w", err)
	}
	defer rows.Close()

	secrets, err := collectSecrets(rows)
	if err != nil {
		return nil, 0, err
	}

	cursor := sinceRevision
	for _, s := range secrets {
		if s.Revision > cursor {
			cursor = s.Revision
		}
	}
	return secrets, cursor, nil
}

// exists сообщает, есть ли активная запись с данным id у владельца.
func (r *SecretRepository) exists(ctx context.Context, ownerID, id string) (bool, error) {
	const q = `SELECT EXISTS(
		SELECT 1 FROM secrets
		WHERE id = $1::uuid AND owner_id = $2::uuid AND deleted = false)`

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
		&s.Version, &s.CreatedAt, &s.UpdatedAt, &s.Deleted, &s.Revision,
	)
	if err != nil {
		return model.Secret{}, err
	}
	s.Type = model.SecretType(typ)
	return s, nil
}

// collectSecrets считывает все строки результата в срез записей.
func collectSecrets(rows pgx.Rows) ([]model.Secret, error) {
	secrets := []model.Secret{}
	for rows.Next() {
		s, err := scanSecret(rows)
		if err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		secrets = append(secrets, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}
	return secrets, nil
}
