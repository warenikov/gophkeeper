// Пакет cryptobox реализует клиентское шифрование GophKeeper (модель
// zero-knowledge): ключ выводится из мастер-пароля пользователя через Argon2id,
// данные шифруются AES-256-GCM. Сервер получает только шифртекст.
//
// Пакет назван cryptobox, а не crypto, чтобы не конфликтовать со стандартным
// пакетом crypto.
package cryptobox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// Параметры Argon2id (согласованы с рекомендациями OWASP): t=3, m=64 MiB, p=1.
const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // в KiB → 64 MiB
	argonThreads uint8  = 1
	keyLen       uint32 = 32 // ключ для AES-256
)

// keySize — длина ключа AES-256 в байтах.
const keySize = 32

// ErrKeySize — ключ имеет неверную длину для AES-256.
var ErrKeySize = errors.New("cryptobox: key must be 32 bytes")

// ErrCiphertext — шифртекст повреждён или слишком короткий.
var ErrCiphertext = errors.New("cryptobox: invalid ciphertext")

// DeriveKey выводит 32-байтовый ключ шифрования из мастер-пароля и логина.
//
// Соль вычисляется детерминированно из логина, чтобы один и тот же ключ
// восстанавливался на любом устройстве пользователя (нужно для синхронизации
// без хранения соли на сервере). Энтропию обеспечивает мастер-пароль; логин
// лишь разделяет пространство ключей между пользователями. Это осознанный
// tradeoff: возможное усиление — случайная соль на пользователя, хранимая на
// сервере и синхронизируемая между устройствами.
func DeriveKey(masterPassword, login string) []byte {
	salt := sha256.Sum256([]byte("gophkeeper:v1:" + login))
	return argon2.IDKey([]byte(masterPassword), salt[:], argonTime, argonMemory, argonThreads, keyLen)
}

// Encrypt шифрует plaintext ключом AES-256-GCM. Результат имеет вид
// nonce || ciphertext || tag. Для каждого вызова используется уникальный
// случайный nonce из crypto/rand.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("cryptobox: read nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt расшифровывает данные вида nonce || ciphertext || tag. Возвращает
// ошибку, если тег аутентификации не сходится (неверный ключ или подделка).
func Decrypt(key, data []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrCiphertext
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCiphertext, err)
	}
	return plaintext, nil
}

// newGCM создаёт AES-256-GCM для ключа длиной 32 байта.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != keySize {
		return nil, ErrKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cryptobox: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cryptobox: new gcm: %w", err)
	}
	return gcm, nil
}
