package grpcserver

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenikov/gophkeeper/internal/pb"
	"github.com/warenikov/gophkeeper/internal/server/auth"
	"github.com/warenikov/gophkeeper/internal/server/model"
)

// --- Заглушки сервисов ---

type mockAuthSvc struct {
	registerFn func(ctx context.Context, login, password string) (string, error)
	loginFn    func(ctx context.Context, login, password string) (string, error)
}

func (m mockAuthSvc) Register(ctx context.Context, login, password string) (string, error) {
	return m.registerFn(ctx, login, password)
}

func (m mockAuthSvc) Login(ctx context.Context, login, password string) (string, error) {
	return m.loginFn(ctx, login, password)
}

type mockSecretSvc struct {
	createFn func(ctx context.Context, ownerID string, s model.Secret) (model.Secret, error)
	listFn   func(ctx context.Context, ownerID string) ([]model.Secret, error)
	getFn    func(ctx context.Context, ownerID, id string) (model.Secret, error)
	updateFn func(ctx context.Context, ownerID string, s model.Secret, v int64) (model.Secret, error)
	deleteFn func(ctx context.Context, ownerID, id string) error
	syncFn   func(ctx context.Context, ownerID string, since int64) ([]model.Secret, int64, error)
}

func (m mockSecretSvc) Create(ctx context.Context, ownerID string, s model.Secret) (model.Secret, error) {
	return m.createFn(ctx, ownerID, s)
}

func (m mockSecretSvc) List(ctx context.Context, ownerID string) ([]model.Secret, error) {
	return m.listFn(ctx, ownerID)
}

func (m mockSecretSvc) Get(ctx context.Context, ownerID, id string) (model.Secret, error) {
	return m.getFn(ctx, ownerID, id)
}

func (m mockSecretSvc) Update(ctx context.Context, ownerID string, s model.Secret, v int64) (model.Secret, error) {
	return m.updateFn(ctx, ownerID, s, v)
}

func (m mockSecretSvc) Delete(ctx context.Context, ownerID, id string) error {
	return m.deleteFn(ctx, ownerID, id)
}

func (m mockSecretSvc) Sync(ctx context.Context, ownerID string, since int64) ([]model.Secret, int64, error) {
	return m.syncFn(ctx, ownerID, since)
}

// --- Тесты ---

func TestAuthHandlerRegister(t *testing.T) {
	h := NewAuthHandler(mockAuthSvc{
		registerFn: func(_ context.Context, login, _ string) (string, error) {
			return "token-" + login, nil
		},
	})

	resp, err := h.Register(context.Background(), &pb.RegisterRequest{Login: "alice", Password: "p"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.GetToken() != "token-alice" {
		t.Errorf("token = %q", resp.GetToken())
	}
}

func TestAuthHandlerRegisterMapsError(t *testing.T) {
	h := NewAuthHandler(mockAuthSvc{
		registerFn: func(context.Context, string, string) (string, error) {
			return "", model.ErrUserExists
		},
	})

	_, err := h.Register(context.Background(), &pb.RegisterRequest{Login: "alice"})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("код = %s, ожидался AlreadyExists", status.Code(err))
	}
}

func TestSecretHandlerRequiresAuth(t *testing.T) {
	h := NewSecretHandler(mockSecretSvc{})
	// Контекст без userID → Unauthenticated.
	_, err := h.List(context.Background(), &pb.ListSecretsRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("код = %s, ожидался Unauthenticated", status.Code(err))
	}
}

func TestSecretHandlerCreate(t *testing.T) {
	h := NewSecretHandler(mockSecretSvc{
		createFn: func(_ context.Context, ownerID string, s model.Secret) (model.Secret, error) {
			s.ID = "s1"
			s.OwnerID = ownerID
			return s, nil
		},
	})

	ctx := auth.ContextWithUserID(context.Background(), "owner-1")
	resp, err := h.Create(ctx, &pb.CreateSecretRequest{
		Type:             pb.SecretType_SECRET_TYPE_TEXT,
		Name:             "note",
		EncryptedPayload: []byte("cipher"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.GetId() != "s1" {
		t.Errorf("id = %q, ожидалось s1", resp.GetId())
	}
}

func TestSecretHandlerGetNotFound(t *testing.T) {
	h := NewSecretHandler(mockSecretSvc{
		getFn: func(context.Context, string, string) (model.Secret, error) {
			return model.Secret{}, model.ErrNotFound
		},
	})

	ctx := auth.ContextWithUserID(context.Background(), "owner-1")
	_, err := h.Get(ctx, &pb.GetSecretRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("код = %s, ожидался NotFound", status.Code(err))
	}
}

func TestSecretHandlerUpdate(t *testing.T) {
	h := NewSecretHandler(mockSecretSvc{
		updateFn: func(_ context.Context, ownerID string, s model.Secret, v int64) (model.Secret, error) {
			if ownerID != "owner-1" || v != 2 {
				t.Errorf("ownerID=%q version=%d", ownerID, v)
			}
			s.Version = v + 1
			return s, nil
		},
	})

	ctx := auth.ContextWithUserID(context.Background(), "owner-1")
	resp, err := h.Update(ctx, &pb.UpdateSecretRequest{
		Id:               "s1",
		Name:             "note",
		EncryptedPayload: []byte("cipher"),
		Version:          2,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if resp.GetVersion() != 3 {
		t.Errorf("версия = %d, ожидалось 3", resp.GetVersion())
	}
}

func TestSecretHandlerCreateValidationError(t *testing.T) {
	h := NewSecretHandler(mockSecretSvc{
		createFn: func(context.Context, string, model.Secret) (model.Secret, error) {
			return model.Secret{}, model.ErrValidation
		},
	})

	ctx := auth.ContextWithUserID(context.Background(), "owner-1")
	_, err := h.Create(ctx, &pb.CreateSecretRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("код = %s, ожидался InvalidArgument", status.Code(err))
	}
}

func TestSecretHandlerListAndDelete(t *testing.T) {
	var deleted string
	h := NewSecretHandler(mockSecretSvc{
		listFn: func(context.Context, string) ([]model.Secret, error) {
			return []model.Secret{{ID: "s1"}, {ID: "s2"}}, nil
		},
		deleteFn: func(_ context.Context, _, id string) error {
			deleted = id
			return nil
		},
	})
	ctx := auth.ContextWithUserID(context.Background(), "owner-1")

	list, err := h.List(ctx, &pb.ListSecretsRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.GetSecrets()) != 2 {
		t.Errorf("получено %d записей, ожидалось 2", len(list.GetSecrets()))
	}

	if _, err := h.Delete(ctx, &pb.DeleteSecretRequest{Id: "s1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted != "s1" {
		t.Errorf("удалён %q, ожидалось s1", deleted)
	}
}
