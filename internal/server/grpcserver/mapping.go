// Пакет grpcserver реализует gRPC-обработчики сервера GophKeeper: преобразует
// запросы в вызовы сервисного слоя и доменные ошибки — в коды gRPC.
package grpcserver

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/warenik/gophkeeper/internal/pb"
	"github.com/warenik/gophkeeper/internal/server/model"
)

// toProtoSecret преобразует доменную запись в protobuf-представление.
func toProtoSecret(s model.Secret) *pb.Secret {
	return &pb.Secret{
		Id:               s.ID,
		Type:             toProtoType(s.Type),
		Name:             s.Name,
		Metadata:         s.Metadata,
		EncryptedPayload: s.EncryptedPayload,
		Version:          s.Version,
		CreatedAt:        timestamppb.New(s.CreatedAt),
		UpdatedAt:        timestamppb.New(s.UpdatedAt),
		Deleted:          s.Deleted,
		Revision:         s.Revision,
	}
}

// toProtoType преобразует доменный тип записи в protobuf-тип.
func toProtoType(t model.SecretType) pb.SecretType {
	switch t {
	case model.SecretTypeLoginPassword:
		return pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD
	case model.SecretTypeText:
		return pb.SecretType_SECRET_TYPE_TEXT
	case model.SecretTypeCard:
		return pb.SecretType_SECRET_TYPE_CARD
	case model.SecretTypeBinary:
		return pb.SecretType_SECRET_TYPE_BINARY
	default:
		return pb.SecretType_SECRET_TYPE_UNSPECIFIED
	}
}

// toModelType преобразует protobuf-тип записи в доменный тип.
func toModelType(t pb.SecretType) model.SecretType {
	switch t {
	case pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD:
		return model.SecretTypeLoginPassword
	case pb.SecretType_SECRET_TYPE_TEXT:
		return model.SecretTypeText
	case pb.SecretType_SECRET_TYPE_CARD:
		return model.SecretTypeCard
	case pb.SecretType_SECRET_TYPE_BINARY:
		return model.SecretTypeBinary
	default:
		return model.SecretTypeUnspecified
	}
}

// toGRPCError отображает доменные sentinel-ошибки в коды gRPC. Неизвестные
// ошибки становятся codes.Internal без раскрытия деталей клиенту.
func toGRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, model.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, model.ErrUserExists):
		return status.Error(codes.AlreadyExists, "user already exists")
	case errors.Is(err, model.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid login or password")
	case errors.Is(err, model.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, model.ErrVersionConflict):
		return status.Error(codes.Aborted, "version conflict")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
