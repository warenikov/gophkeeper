package keeper

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/warenik/gophkeeper/internal/pb"
	"github.com/warenik/gophkeeper/internal/server/auth"
	"github.com/warenik/gophkeeper/internal/server/grpcserver"
	"github.com/warenik/gophkeeper/internal/server/model"
	"github.com/warenik/gophkeeper/internal/server/service"
)

// fakeUsers — потокобезопасное in-memory хранилище пользователей для теста.
type fakeUsers struct {
	mu      sync.Mutex
	byLogin map[string]model.User
	seq     int
}

func (f *fakeUsers) Create(_ context.Context, login, hash string) (model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byLogin[login]; ok {
		return model.User{}, model.ErrUserExists
	}
	f.seq++
	u := model.User{ID: string(rune('A' + f.seq)), Login: login, PasswordHash: hash}
	f.byLogin[login] = u
	return u, nil
}

func (f *fakeUsers) GetByLogin(_ context.Context, login string) (model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byLogin[login]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return u, nil
}

// fakeSecrets — потокобезопасное in-memory хранилище записей для теста.
type fakeSecrets struct {
	mu   sync.Mutex
	byID map[string]model.Secret
	seq  int
}

func (f *fakeSecrets) Create(_ context.Context, s model.Secret) (model.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	s.ID = string(rune('a' + f.seq))
	s.Version = 1
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	f.byID[s.ID] = s
	return s, nil
}

func (f *fakeSecrets) ListByOwner(_ context.Context, ownerID string) ([]model.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []model.Secret{}
	for _, s := range f.byID {
		if s.OwnerID == ownerID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSecrets) GetByID(_ context.Context, ownerID, id string) (model.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[id]
	if !ok || s.OwnerID != ownerID {
		return model.Secret{}, model.ErrNotFound
	}
	return s, nil
}

func (f *fakeSecrets) Update(_ context.Context, s model.Secret, expected int64) (model.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[s.ID]
	if !ok || cur.OwnerID != s.OwnerID {
		return model.Secret{}, model.ErrNotFound
	}
	if cur.Version != expected {
		return model.Secret{}, model.ErrVersionConflict
	}
	cur.Name = s.Name
	cur.Metadata = s.Metadata
	cur.EncryptedPayload = s.EncryptedPayload
	cur.Version++
	cur.UpdatedAt = time.Now()
	f.byID[s.ID] = cur
	return cur, nil
}

func (f *fakeSecrets) Delete(_ context.Context, ownerID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[id]
	if !ok || s.OwnerID != ownerID {
		return model.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

// startServer поднимает gRPC-сервер с реальными хендлерами и in-memory
// хранилищами на bufconn и возвращает диалер.
func startServer(t *testing.T) func(context.Context, string) (net.Conn, error) {
	t.Helper()

	tokens := auth.NewTokenManager("test-secret", time.Hour)
	authSvc := service.NewAuth(&fakeUsers{byLogin: map[string]model.User{}}, tokens)
	secretSvc := service.NewSecret(&fakeSecrets{byID: map[string]model.Secret{}})

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(tokens.NewUnaryInterceptor()))
	pb.RegisterAuthServiceServer(srv, grpcserver.NewAuthHandler(authSvc))
	pb.RegisterSecretsServiceServer(srv, grpcserver.NewSecretHandler(secretSvc))

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
}

// newKeeper создаёт keeper.Client поверх bufconn с указанным токеном.
func newKeeper(t *testing.T, dialer func(context.Context, string) (net.Conn, error), token string) *Client {
	t.Helper()

	opts := []grpc.DialOption{
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if token != "" {
		opts = append(opts, grpc.WithUnaryInterceptor(tokenInterceptor(token)))
	}

	conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &Client{conn: conn, auth: pb.NewAuthServiceClient(conn), secrets: pb.NewSecretsServiceClient(conn)}
}

// TestEndToEnd проходит полный сценарий: регистрация → создание → список →
// получение → обновление → удаление через реальный gRPC-стек.
func TestEndToEnd(t *testing.T) {
	dialer := startServer(t)
	ctx := context.Background()

	// Регистрация без токена.
	anon := newKeeper(t, dialer, "")
	token, err := anon.Register(ctx, "alice", "password")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if token == "" {
		t.Fatal("пустой токен после регистрации")
	}

	// Дальнейшие вызовы — с токеном.
	c := newKeeper(t, dialer, token)

	created, err := c.Create(ctx, pb.SecretType_SECRET_TYPE_TEXT, "note", "meta", []byte("cipher"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := c.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("в списке %d записей, ожидалась 1", len(list))
	}

	got, err := c.Get(ctx, created.GetId())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.GetEncryptedPayload()) != "cipher" {
		t.Errorf("нагрузка = %q", got.GetEncryptedPayload())
	}

	updated, err := c.Update(ctx, created.GetId(), "note2", "meta2", []byte("cipher2"), got.GetVersion())
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.GetVersion() != 2 {
		t.Errorf("версия после обновления = %d, ожидалось 2", updated.GetVersion())
	}

	if err := c.Delete(ctx, created.GetId()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// TestAllSecretTypes проверяет создание и получение всех типов записей через
// реальный gRPC-стек.
func TestAllSecretTypes(t *testing.T) {
	dialer := startServer(t)
	ctx := context.Background()

	token, err := newKeeper(t, dialer, "").Register(ctx, "carol", "pw")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	c := newKeeper(t, dialer, token)

	types := []pb.SecretType{
		pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD,
		pb.SecretType_SECRET_TYPE_TEXT,
		pb.SecretType_SECRET_TYPE_CARD,
		pb.SecretType_SECRET_TYPE_BINARY,
	}
	for _, typ := range types {
		created, err := c.Create(ctx, typ, "name", "meta", []byte("payload"))
		if err != nil {
			t.Fatalf("Create(%v): %v", typ, err)
		}
		got, err := c.Get(ctx, created.GetId())
		if err != nil {
			t.Fatalf("Get(%v): %v", typ, err)
		}
		if got.GetType() != typ {
			t.Errorf("тип = %v, ожидался %v", got.GetType(), typ)
		}
	}

	list, err := c.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != len(types) {
		t.Errorf("в списке %d записей, ожидалось %d", len(list), len(types))
	}
}

// TestDialAndClose проверяет создание клиента (grpc.NewClient ленивый и не
// требует живого сервера) и закрытие соединения — с токеном и без.
func TestDialAndClose(t *testing.T) {
	for _, token := range []string{"tok", ""} {
		c, err := Dial("localhost:12345", token)
		if err != nil {
			t.Fatalf("Dial(token=%q): %v", token, err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("Close(token=%q): %v", token, err)
		}
	}
}

// TestUnauthenticated проверяет, что без токена защищённый метод отклоняется.
func TestUnauthenticated(t *testing.T) {
	dialer := startServer(t)
	c := newKeeper(t, dialer, "")

	if _, err := c.List(context.Background()); err == nil {
		t.Error("ожидалась ошибка при вызове без токена")
	}
}

// TestLoginFlow проверяет вход существующего пользователя.
func TestLoginFlow(t *testing.T) {
	dialer := startServer(t)
	ctx := context.Background()

	anon := newKeeper(t, dialer, "")
	if _, err := anon.Register(ctx, "bob", "secret"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	token, err := anon.Login(ctx, "bob", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Error("пустой токен после входа")
	}

	if _, err := anon.Login(ctx, "bob", "wrong"); err == nil {
		t.Error("ожидалась ошибка при неверном пароле")
	}
}
