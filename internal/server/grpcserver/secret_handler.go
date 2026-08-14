package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenikov/gophkeeper/internal/pb"
	"github.com/warenikov/gophkeeper/internal/server/auth"
	"github.com/warenikov/gophkeeper/internal/server/model"
)

// secretService — бизнес-операции над записями, нужные обработчику.
type secretService interface {
	Create(ctx context.Context, ownerID string, secret model.Secret) (model.Secret, error)
	List(ctx context.Context, ownerID string) ([]model.Secret, error)
	Get(ctx context.Context, ownerID, id string) (model.Secret, error)
	Update(ctx context.Context, ownerID string, secret model.Secret, expectedVersion int64) (model.Secret, error)
	Delete(ctx context.Context, ownerID, id string) error
	Sync(ctx context.Context, ownerID string, sinceRevision int64) ([]model.Secret, int64, error)
}

// SecretHandler реализует pb.SecretsServiceServer поверх сервисного слоя.
type SecretHandler struct {
	pb.UnimplementedSecretsServiceServer
	svc secretService
}

// NewSecretHandler создаёт обработчик записей.
func NewSecretHandler(svc secretService) *SecretHandler {
	return &SecretHandler{svc: svc}
}

// Create создаёт новую запись авторизованного пользователя.
func (h *SecretHandler) Create(ctx context.Context, req *pb.CreateSecretRequest) (*pb.Secret, error) {
	ownerID, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	created, err := h.svc.Create(ctx, ownerID, model.Secret{
		Type:             toModelType(req.GetType()),
		Name:             req.GetName(),
		Metadata:         req.GetMetadata(),
		EncryptedPayload: req.GetEncryptedPayload(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoSecret(created), nil
}

// List возвращает все записи авторизованного пользователя.
func (h *SecretHandler) List(ctx context.Context, _ *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	ownerID, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	secrets, err := h.svc.List(ctx, ownerID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &pb.ListSecretsResponse{Secrets: make([]*pb.Secret, 0, len(secrets))}
	for _, s := range secrets {
		resp.Secrets = append(resp.Secrets, toProtoSecret(s))
	}
	return resp, nil
}

// Get возвращает запись по идентификатору.
func (h *SecretHandler) Get(ctx context.Context, req *pb.GetSecretRequest) (*pb.Secret, error) {
	ownerID, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := h.svc.Get(ctx, ownerID, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoSecret(secret), nil
}

// Update обновляет запись при совпадении ожидаемой версии.
func (h *SecretHandler) Update(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.Secret, error) {
	ownerID, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := h.svc.Update(ctx, ownerID, model.Secret{
		ID:               req.GetId(),
		Name:             req.GetName(),
		Metadata:         req.GetMetadata(),
		EncryptedPayload: req.GetEncryptedPayload(),
	}, req.GetVersion())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoSecret(updated), nil
}

// Delete удаляет запись по идентификатору.
func (h *SecretHandler) Delete(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	ownerID, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.svc.Delete(ctx, ownerID, req.GetId()); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteSecretResponse{}, nil
}

// Sync возвращает изменения записей владельца с указанной ревизии.
func (h *SecretHandler) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	ownerID, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	secrets, cursor, err := h.svc.Sync(ctx, ownerID, req.GetSinceRevision())
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &pb.SyncResponse{
		Secrets:  make([]*pb.Secret, 0, len(secrets)),
		Revision: cursor,
	}
	for _, s := range secrets {
		resp.Secrets = append(resp.Secrets, toProtoSecret(s))
	}
	return resp, nil
}

// ownerFromContext извлекает идентификатор владельца, помещённый в контекст
// интерцептором авторизации.
func ownerFromContext(ctx context.Context) (string, error) {
	ownerID, ok := auth.UserIDFromContext(ctx)
	if !ok || ownerID == "" {
		return "", status.Error(codes.Unauthenticated, "missing user identity")
	}
	return ownerID, nil
}
