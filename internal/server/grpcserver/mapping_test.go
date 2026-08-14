package grpcserver

import (
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenik/gophkeeper/internal/pb"
	"github.com/warenik/gophkeeper/internal/server/model"
)

func TestToGRPCError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"validation", model.ErrValidation, codes.InvalidArgument},
		{"user exists", model.ErrUserExists, codes.AlreadyExists},
		{"credentials", model.ErrInvalidCredentials, codes.Unauthenticated},
		{"not found", model.ErrNotFound, codes.NotFound},
		{"conflict", model.ErrVersionConflict, codes.Aborted},
		{"unknown", errors.New("boom"), codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := status.Code(toGRPCError(tc.err)); got != tc.want {
				t.Errorf("код = %s, ожидалось %s", got, tc.want)
			}
		})
	}
}

func TestTypeMappingRoundtrip(t *testing.T) {
	types := []model.SecretType{model.SecretTypeLoginPassword, model.SecretTypeText}
	for _, mt := range types {
		if got := toModelType(toProtoType(mt)); got != mt {
			t.Errorf("roundtrip типа: получено %d, ожидалось %d", got, mt)
		}
	}

	// Неизвестный тип отображается в Unspecified.
	if got := toModelType(pb.SecretType_SECRET_TYPE_UNSPECIFIED); got != model.SecretTypeUnspecified {
		t.Errorf("неизвестный тип = %d, ожидался Unspecified", got)
	}
}

func TestToProtoSecret(t *testing.T) {
	now := time.Now()
	s := model.Secret{
		ID:               "s1",
		Type:             model.SecretTypeText,
		Name:             "note",
		Metadata:         "meta",
		EncryptedPayload: []byte("cipher"),
		Version:          2,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	got := toProtoSecret(s)
	if got.GetId() != "s1" || got.GetName() != "note" || got.GetMetadata() != "meta" {
		t.Errorf("поля отображены неверно: %+v", got)
	}
	if got.GetType() != pb.SecretType_SECRET_TYPE_TEXT {
		t.Errorf("тип = %v, ожидался TEXT", got.GetType())
	}
	if got.GetVersion() != 2 {
		t.Errorf("версия = %d, ожидалось 2", got.GetVersion())
	}
	if string(got.GetEncryptedPayload()) != "cipher" {
		t.Errorf("нагрузка = %q, ожидалось cipher", got.GetEncryptedPayload())
	}
}
