package auth_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/warenik/gophkeeper/internal/server/auth"
)

func TestContextUserIDRoundtrip(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), "u1")
	id, ok := auth.UserIDFromContext(ctx)
	if !ok || id != "u1" {
		t.Errorf("получено (%q, %v), ожидалось (u1, true)", id, ok)
	}

	if _, ok := auth.UserIDFromContext(context.Background()); ok {
		t.Error("для пустого контекста ожидалось ok=false")
	}
}

func TestInterceptorPublicMethodSkipsAuth(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	interceptor := tm.NewUnaryInterceptor()

	called := false
	handler := func(context.Context, any) (any, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.AuthService/Login"}
	if _, err := interceptor(context.Background(), nil, info, handler); err != nil {
		t.Fatalf("публичный метод вернул ошибку: %v", err)
	}
	if !called {
		t.Error("обработчик публичного метода не вызван")
	}
}

func TestInterceptorRejectsMissingToken(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	interceptor := tm.NewUnaryInterceptor()

	handler := func(context.Context, any) (any, error) { return nil, nil }
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretsService/List"}

	_, err := interceptor(context.Background(), nil, info, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("код = %s, ожидался Unauthenticated", status.Code(err))
	}
}

func TestInterceptorAcceptsValidToken(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	token, _ := tm.Issue("u1")
	interceptor := tm.NewUnaryInterceptor()

	var gotUserID string
	handler := func(ctx context.Context, _ any) (any, error) {
		gotUserID, _ = auth.UserIDFromContext(ctx)
		return "ok", nil
	}

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretsService/List"}

	if _, err := interceptor(ctx, nil, info, handler); err != nil {
		t.Fatalf("валидный токен отклонён: %v", err)
	}
	if gotUserID != "u1" {
		t.Errorf("userID в контексте = %q, ожидалось u1", gotUserID)
	}
}

func TestInterceptorRejectsInvalidToken(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	interceptor := tm.NewUnaryInterceptor()

	handler := func(context.Context, any) (any, error) { return nil, nil }
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer garbage"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretsService/List"}

	_, err := interceptor(ctx, nil, info, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("код = %s, ожидался Unauthenticated", status.Code(err))
	}
}
