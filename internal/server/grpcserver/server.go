package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/warenik/gophkeeper/internal/pb"
	"github.com/warenik/gophkeeper/internal/server/auth"
)

// Server оборачивает gRPC-сервер GophKeeper с его слушателем и логированием.
type Server struct {
	grpc *grpc.Server
	addr string
	log  *slog.Logger
}

// New создаёт gRPC-сервер, настраивает цепочку интерцепторов (логирование →
// авторизация) и регистрирует обработчики Auth и Secrets.
func New(
	addr string,
	tm *auth.TokenManager,
	authHandler *AuthHandler,
	secretHandler *SecretHandler,
	log *slog.Logger,
	creds credentials.TransportCredentials,
) *Server {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log), // внешний: ловит панику любого следующего слоя
			loggingInterceptor(log),
			tm.NewUnaryInterceptor(),
		),
	}
	if creds != nil {
		opts = append(opts, grpc.Creds(creds))
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterAuthServiceServer(srv, authHandler)
	pb.RegisterSecretsServiceServer(srv, secretHandler)

	return &Server{grpc: srv, addr: addr, log: log}
}

// Run запускает сервер и блокируется до отмены контекста, после чего выполняет
// graceful shutdown. Возвращает nil при штатном завершении.
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}

	go func() {
		<-ctx.Done()
		s.log.Info("остановка gRPC-сервера")
		s.grpc.GracefulStop()
	}()

	s.log.Info("gRPC-сервер слушает", "address", s.addr)
	if err := s.grpc.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// loggingInterceptor логирует каждый unary-вызов: метод, код ответа и
// длительность. Внутренние ошибки (Internal/Unknown) логируются на уровне Error
// вместе с текстом ошибки, чтобы причина сбоя была видна (клиенту при этом
// возвращается только код).
func loggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := status.Code(err)
		attrs := []any{
			"method", info.FullMethod,
			"code", code.String(),
			"duration", time.Since(start).String(),
		}
		if code == codes.Internal || code == codes.Unknown {
			log.Error("gRPC-вызов: внутренняя ошибка", append(attrs, "err", err)...)
		} else {
			log.Info("gRPC-вызов", attrs...)
		}
		return resp, err
	}
}

// recoveryInterceptor перехватывает панику в обработчике, логирует её со стеком
// и возвращает клиенту codes.Internal, не давая процессу сервера упасть.
func recoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("паника в обработчике gRPC",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}
