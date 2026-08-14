// Пакет auth отвечает за хеширование паролей, выпуск/проверку JWT и
// gRPC-интерцептор авторизации сервера GophKeeper.
package auth

import (
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost — стоимость bcrypt. 12 — консервативный выбор для менеджера
// паролей (каждая +1 удваивает время хеширования).
const bcryptCost = 12

// prehash снимает ограничение bcrypt в 72 байта: пароль сворачивается в
// SHA-256 и кодируется base64 (44 байта, всегда < 72). Это устраняет как
// молчаливую обрезку, так и ошибку на длинных паролях.
func prehash(password string) []byte {
	sum := sha256.Sum256([]byte(password))
	encoded := base64.StdEncoding.EncodeToString(sum[:])
	return []byte(encoded)
}

// HashPassword возвращает bcrypt-хеш пароля (соль встроена в результат).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(prehash(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword возвращает nil, если пароль соответствует хешу, и
// ненулевую ошибку в противном случае.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), prehash(password))
}
