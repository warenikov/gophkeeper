package grpcserver

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenik/gophkeeper/internal/server/auth"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

var testInfo = &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretsService/List"}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := recoveryInterceptor(discardLogger())

	// Паника перехватывается и превращается в codes.Internal.
	_, err := interceptor(context.Background(), nil, testInfo,
		func(context.Context, any) (any, error) { panic("boom") })
	if status.Code(err) != codes.Internal {
		t.Errorf("код при панике = %s, ожидался Internal", status.Code(err))
	}

	// Обычный вызов проходит без изменений.
	resp, err := interceptor(context.Background(), nil, testInfo,
		func(context.Context, any) (any, error) { return "ok", nil })
	if err != nil || resp != "ok" {
		t.Errorf("обычный вызов: resp=%v err=%v", resp, err)
	}
}

func TestLoggingInterceptor(t *testing.T) {
	interceptor := loggingInterceptor(discardLogger())

	cases := []error{
		nil,
		status.Error(codes.NotFound, "клиентская ошибка"),
		status.Error(codes.Internal, "внутренняя ошибка"),
	}
	for _, want := range cases {
		_, err := interceptor(context.Background(), nil, testInfo,
			func(context.Context, any) (any, error) { return "ok", want })
		if status.Code(err) != status.Code(want) {
			t.Errorf("код = %s, ожидался %s", status.Code(err), status.Code(want))
		}
	}
}

func TestNew(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	srv := New(":0", tm, NewAuthHandler(nil), NewSecretHandler(nil), discardLogger(), nil)
	if srv == nil {
		t.Fatal("New вернул nil")
	}
}

func TestRunGracefulShutdown(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	srv := New("127.0.0.1:0", tm, NewAuthHandler(nil), NewSecretHandler(nil), discardLogger(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()

	// Даём серверу подняться, затем инициируем graceful shutdown.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run вернул ошибку: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run не завершился после отмены контекста")
	}
}
