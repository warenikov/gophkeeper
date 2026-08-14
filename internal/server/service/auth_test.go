package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/warenik/gophkeeper/internal/server/auth"
	"github.com/warenik/gophkeeper/internal/server/model"
	"github.com/warenik/gophkeeper/internal/server/service"
)

// mockUserStore — конфигурируемая заглушка хранилища пользователей.
type mockUserStore struct {
	createFn func(ctx context.Context, login, hash string) (model.User, error)
	getFn    func(ctx context.Context, login string) (model.User, error)
}

func (m mockUserStore) Create(ctx context.Context, login, hash string) (model.User, error) {
	return m.createFn(ctx, login, hash)
}

func (m mockUserStore) GetByLogin(ctx context.Context, login string) (model.User, error) {
	return m.getFn(ctx, login)
}

// stubTokens — заглушка выпуска токенов.
type stubTokens struct{}

func (stubTokens) Issue(userID string) (string, error) { return "token-" + userID, nil }

func TestAuthRegister(t *testing.T) {
	users := mockUserStore{
		createFn: func(_ context.Context, login, hash string) (model.User, error) {
			if hash == "" {
				t.Error("пароль должен быть захеширован")
			}
			return model.User{ID: "u1", Login: login}, nil
		},
	}
	svc := service.NewAuth(users, stubTokens{})

	token, err := svc.Register(context.Background(), "alice", "pass")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if token != "token-u1" {
		t.Errorf("token = %q, ожидалось token-u1", token)
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	svc := service.NewAuth(mockUserStore{}, stubTokens{})
	if _, err := svc.Register(context.Background(), "", "pass"); !errors.Is(err, model.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено %v", err)
	}
}

func TestAuthRegisterUserExists(t *testing.T) {
	users := mockUserStore{
		createFn: func(context.Context, string, string) (model.User, error) {
			return model.User{}, model.ErrUserExists
		},
	}
	svc := service.NewAuth(users, stubTokens{})
	if _, err := svc.Register(context.Background(), "alice", "pass"); !errors.Is(err, model.ErrUserExists) {
		t.Errorf("ожидалась ErrUserExists, получено %v", err)
	}
}

func TestAuthLogin(t *testing.T) {
	hash, _ := auth.HashPassword("pass")
	users := mockUserStore{
		getFn: func(_ context.Context, login string) (model.User, error) {
			return model.User{ID: "u1", Login: login, PasswordHash: hash}, nil
		},
	}
	svc := service.NewAuth(users, stubTokens{})

	token, err := svc.Login(context.Background(), "alice", "pass")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token != "token-u1" {
		t.Errorf("token = %q, ожидалось token-u1", token)
	}
}

func TestAuthLoginWrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("pass")
	users := mockUserStore{
		getFn: func(_ context.Context, login string) (model.User, error) {
			return model.User{ID: "u1", Login: login, PasswordHash: hash}, nil
		},
	}
	svc := service.NewAuth(users, stubTokens{})
	if _, err := svc.Login(context.Background(), "alice", "wrong"); !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("ожидалась ErrInvalidCredentials, получено %v", err)
	}
}

func TestAuthLoginUnknownUser(t *testing.T) {
	users := mockUserStore{
		getFn: func(context.Context, string) (model.User, error) {
			return model.User{}, model.ErrNotFound
		},
	}
	svc := service.NewAuth(users, stubTokens{})
	// Неизвестный пользователь маскируется под неверные учётные данные.
	if _, err := svc.Login(context.Background(), "ghost", "pass"); !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("ожидалась ErrInvalidCredentials, получено %v", err)
	}
}

func TestAuthLoginValidation(t *testing.T) {
	svc := service.NewAuth(mockUserStore{}, stubTokens{})
	if _, err := svc.Login(context.Background(), "", ""); !errors.Is(err, model.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено %v", err)
	}
}
