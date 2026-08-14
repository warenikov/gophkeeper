package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/warenik/gophkeeper/internal/server/model"
)

// uniqueViolation — SQLSTATE-код нарушения уникальности PostgreSQL.
const uniqueViolation = "23505"

// UserRepository — доступ к таблице пользователей.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создаёт репозиторий пользователей.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create добавляет пользователя. Возвращает model.ErrUserExists, если логин уже
// занят.
func (r *UserRepository) Create(ctx context.Context, login, passwordHash string) (model.User, error) {
	const q = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, created_at`

	user := model.User{Login: login, PasswordHash: passwordHash}
	err := r.pool.QueryRow(ctx, q, login, passwordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return model.User{}, model.ErrUserExists
		}
		return model.User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

// GetByLogin возвращает пользователя по логину или model.ErrNotFound.
func (r *UserRepository) GetByLogin(ctx context.Context, login string) (model.User, error) {
	const q = `
		SELECT id::text, login, password_hash, created_at
		FROM users
		WHERE login = $1`

	var user model.User
	err := r.pool.QueryRow(ctx, q, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}
