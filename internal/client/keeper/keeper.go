// Пакет keeper — клиент gRPC-API GophKeeper: устанавливает соединение и
// предоставляет типобезопасные методы над сервисами Auth и Secrets.
package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/warenik/gophkeeper/internal/pb"
)

// Client — подключение к серверу GophKeeper.
type Client struct {
	conn    *grpc.ClientConn
	auth    pb.AuthServiceClient
	secrets pb.SecretsServiceClient
}

// Dial создаёт клиент к серверу по адресу address. Если token не пуст, он
// прикладывается к каждому исходящему вызову в заголовке authorization.
//
// Если caFile не пуст, соединение устанавливается поверх TLS с доверием к
// указанному корневому сертификату; иначе используется незащищённый транспорт
// (локальная разработка).
func Dial(address, token, caFile string) (*Client, error) {
	transportCreds := insecure.NewCredentials()
	if caFile != "" {
		tlsCreds, err := credentials.NewClientTLSFromFile(caFile, "")
		if err != nil {
			return nil, fmt.Errorf("keeper: load CA %s: %w", caFile, err)
		}
		transportCreds = tlsCreds
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
	}
	if token != "" {
		opts = append(opts, grpc.WithUnaryInterceptor(tokenInterceptor(token)))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("keeper: dial %s: %w", address, err)
	}

	return &Client{
		conn:    conn,
		auth:    pb.NewAuthServiceClient(conn),
		secrets: pb.NewSecretsServiceClient(conn),
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Register регистрирует пользователя и возвращает токен доступа.
func (c *Client) Register(ctx context.Context, login, password string) (string, error) {
	resp, err := c.auth.Register(ctx, &pb.RegisterRequest{Login: login, Password: password})
	if err != nil {
		return "", err
	}
	return resp.GetToken(), nil
}

// Login аутентифицирует пользователя и возвращает токен доступа.
func (c *Client) Login(ctx context.Context, login, password string) (string, error) {
	resp, err := c.auth.Login(ctx, &pb.LoginRequest{Login: login, Password: password})
	if err != nil {
		return "", err
	}
	return resp.GetToken(), nil
}

// Create создаёт новую запись с уже зашифрованной нагрузкой.
func (c *Client) Create(ctx context.Context, typ pb.SecretType, name, meta string, encrypted []byte) (*pb.Secret, error) {
	return c.secrets.Create(ctx, &pb.CreateSecretRequest{
		Type:             typ,
		Name:             name,
		Metadata:         meta,
		EncryptedPayload: encrypted,
	})
}

// List возвращает все записи пользователя.
func (c *Client) List(ctx context.Context) ([]*pb.Secret, error) {
	resp, err := c.secrets.List(ctx, &pb.ListSecretsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetSecrets(), nil
}

// Get возвращает запись по идентификатору.
func (c *Client) Get(ctx context.Context, id string) (*pb.Secret, error) {
	return c.secrets.Get(ctx, &pb.GetSecretRequest{Id: id})
}

// Update обновляет запись при совпадении ожидаемой версии.
func (c *Client) Update(ctx context.Context, id, name, meta string, encrypted []byte, version int64) (*pb.Secret, error) {
	return c.secrets.Update(ctx, &pb.UpdateSecretRequest{
		Id:               id,
		Name:             name,
		Metadata:         meta,
		EncryptedPayload: encrypted,
		Version:          version,
	})
}

// Delete удаляет запись по идентификатору.
func (c *Client) Delete(ctx context.Context, id string) error {
	_, err := c.secrets.Delete(ctx, &pb.DeleteSecretRequest{Id: id})
	return err
}

// Sync возвращает изменения (включая тумбстоны) с указанной ревизии и новый
// курсор для следующей синхронизации.
func (c *Client) Sync(ctx context.Context, sinceRevision int64) ([]*pb.Secret, int64, error) {
	resp, err := c.secrets.Sync(ctx, &pb.SyncRequest{SinceRevision: sinceRevision})
	if err != nil {
		return nil, 0, err
	}
	return resp.GetSecrets(), resp.GetRevision(), nil
}

// tokenInterceptor добавляет заголовок authorization с bearer-токеном к каждому
// исходящему unary-вызову.
func tokenInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
