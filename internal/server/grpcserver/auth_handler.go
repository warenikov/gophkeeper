package grpcserver

import (
	"context"

	"github.com/warenik/gophkeeper/internal/pb"
)

// authService — бизнес-операции аутентификации, нужные обработчику.
type authService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

// AuthHandler реализует pb.AuthServiceServer поверх сервисного слоя.
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	svc authService
}

// NewAuthHandler создаёт обработчик аутентификации.
func NewAuthHandler(svc authService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register регистрирует пользователя и возвращает токен доступа.
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	token, err := h.svc.Register(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.AuthResponse{Token: token}, nil
}

// Login аутентифицирует пользователя и возвращает токен доступа.
func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	token, err := h.svc.Login(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.AuthResponse{Token: token}, nil
}
