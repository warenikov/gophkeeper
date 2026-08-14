// Пакет service содержит бизнес-логику сервера GophKeeper, независимую от
// транспорта (gRPC) и конкретного хранилища.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/warenikov/gophkeeper/internal/server/auth"
	"github.com/warenikov/gophkeeper/internal/server/model"
)

// userStore — доступ к пользователям, необходимый сервису аутентификации.
type userStore interface {
	Create(ctx context.Context, login, passwordHash string) (model.User, error)
	GetByLogin(ctx context.Context, login string) (model.User, error)
}

// tokenIssuer выпускает токен доступа для пользователя.
type tokenIssuer interface {
	Issue(userID string) (string, error)
}

// Auth реализует регистрацию и вход пользователей.
type Auth struct {
	users  userStore
	tokens tokenIssuer
}

// NewAuth создаёт сервис аутентификации.
func NewAuth(users userStore, tokens tokenIssuer) *Auth {
	return &Auth{users: users, tokens: tokens}
}

// Register регистрирует нового пользователя и возвращает токен доступа.
// Возвращает model.ErrValidation при пустых полях и model.ErrUserExists, если
// логин занят.
func (a *Auth) Register(ctx context.Context, login, password string) (string, error) {
	if login == "" || password == "" {
		return "", fmt.Errorf("%w: login and password are required", model.ErrValidation)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	user, err := a.users.Create(ctx, login, hash)
	if err != nil {
		return "", err
	}

	return a.tokens.Issue(user.ID)
}

// Login аутентифицирует пользователя и возвращает токен доступа. При неверном
// логине или пароле возвращает model.ErrInvalidCredentials (не раскрывая, что
// именно неверно).
func (a *Auth) Login(ctx context.Context, login, password string) (string, error) {
	if login == "" || password == "" {
		return "", fmt.Errorf("%w: login and password are required", model.ErrValidation)
	}

	user, err := a.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return "", model.ErrInvalidCredentials
		}
		return "", err
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return "", model.ErrInvalidCredentials
	}

	return a.tokens.Issue(user.ID)
}
