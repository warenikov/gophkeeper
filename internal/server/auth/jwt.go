package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// issuer — значение claim `iss` в выпускаемых токенах.
const issuer = "gophkeeper"

// ErrInvalidToken — токен недействителен (истёк, подделан или некорректен).
var ErrInvalidToken = errors.New("auth: invalid token")

// accessClaims — набор claim'ов токена доступа. Встраивание RegisteredClaims
// (не указателем) делает тип совместимым с интерфейсом jwt.Claims.
type accessClaims struct {
	jwt.RegisteredClaims
}

// TokenManager выпускает и проверяет JWT-токены доступа (HS256).
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager создаёт TokenManager с заданным секретом и временем жизни
// токена.
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

// Issue выпускает подписанный токен доступа, subject которого равен userID.
func (m *TokenManager) Issue(userID string) (string, error) {
	now := time.Now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse проверяет подпись и стандартные claim'ы токена и возвращает userID
// (subject). При любой ошибке валидации возвращает ErrInvalidToken.
func (m *TokenManager) Parse(tokenString string) (string, error) {
	claims := &accessClaims{}

	keyFunc := func(t *jwt.Token) (any, error) {
		// Защита от alg-confusion: помимо WithValidMethods фиксируем именно
		// HMAC, чтобы нельзя было подменить алгоритм на RS256/none.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}

	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		keyFunc,
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(issuer),
	)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	return claims.Subject, nil
}
