package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
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
) *Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor(log),
			tm.NewUnaryInterceptor(),
		),
	)

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
// длительность.
func loggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Info("gRPC-вызов",
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration", time.Since(start).String(),
		)
		return resp, err
	}
}
