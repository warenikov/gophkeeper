package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ctxKey — неэкспортируемый тип ключа контекста, чтобы избежать коллизий.
type ctxKey string

// userIDKey — ключ, под которым идентификатор пользователя кладётся в контекст.
const userIDKey ctxKey = "userID"

// publicMethods — методы, не требующие авторизации.
var publicMethods = map[string]bool{
	"/gophkeeper.v1.AuthService/Register": true,
	"/gophkeeper.v1.AuthService/Login":    true,
}

// ContextWithUserID возвращает контекст с записанным идентификатором
// пользователя. Используется интерцептором после проверки токена (и в тестах).
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext возвращает идентификатор авторизованного пользователя,
// помещённый в контекст интерцептором, и признак его наличия.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// NewUnaryInterceptor возвращает unary-интерцептор, который для непубличных
// методов извлекает bearer-токен из метаданных, проверяет его и кладёт userID
// в контекст. Для публичных методов авторизация пропускается.
func (m *TokenManager) NewUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		token, err := bearerFromContext(ctx)
		if err != nil {
			return nil, err
		}

		userID, err := m.Parse(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(ContextWithUserID(ctx, userID), req)
	}
}

// bearerFromContext извлекает токен из заголовка authorization формата
// "Bearer <token>".
func bearerFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(values[0], prefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization scheme")
	}

	return strings.TrimPrefix(values[0], prefix), nil
}
