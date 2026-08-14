package cryptobox_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/warenik/gophkeeper/internal/client/cryptobox"
)

func TestDeriveKey(t *testing.T) {
	key := cryptobox.DeriveKey("master", "alice")
	if len(key) != 32 {
		t.Fatalf("длина ключа = %d, ожидалось 32", len(key))
	}

	// Детерминированность: те же вход → тот же ключ.
	if !bytes.Equal(key, cryptobox.DeriveKey("master", "alice")) {
		t.Error("DeriveKey недетерминирован для одинаковых входных данных")
	}
	// Разный логин → разный ключ.
	if bytes.Equal(key, cryptobox.DeriveKey("master", "bob")) {
		t.Error("разные логины дали одинаковый ключ")
	}
	// Разный пароль → разный ключ.
	if bytes.Equal(key, cryptobox.DeriveKey("other", "alice")) {
		t.Error("разные пароли дали одинаковый ключ")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := cryptobox.DeriveKey("master", "alice")
	cases := map[string][]byte{
		"пустые данные": {},
		"короткие":      []byte("secret"),
		"юникод":        []byte("пароль123"),
	}

	for name, plaintext := range cases {
		t.Run(name, func(t *testing.T) {
			ciphertext, err := cryptobox.Encrypt(key, plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			got, err := cryptobox.Decrypt(key, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !bytes.Equal(got, plaintext) {
				t.Errorf("после roundtrip = %q, ожидалось %q", got, plaintext)
			}
		})
	}
}

func TestEncryptUsesUniqueNonce(t *testing.T) {
	key := cryptobox.DeriveKey("master", "alice")
	a, _ := cryptobox.Encrypt(key, []byte("same"))
	b, _ := cryptobox.Encrypt(key, []byte("same"))
	if bytes.Equal(a, b) {
		t.Error("два шифрования одного текста совпали — nonce не уникален")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	ciphertext, _ := cryptobox.Encrypt(cryptobox.DeriveKey("master", "alice"), []byte("secret"))
	if _, err := cryptobox.Decrypt(cryptobox.DeriveKey("wrong", "alice"), ciphertext); !errors.Is(err, cryptobox.ErrCiphertext) {
		t.Errorf("ожидалась ErrCiphertext, получено %v", err)
	}
}

func TestDecryptTampered(t *testing.T) {
	key := cryptobox.DeriveKey("master", "alice")
	ciphertext, _ := cryptobox.Encrypt(key, []byte("secret"))
	ciphertext[len(ciphertext)-1] ^= 0xff // портим тег аутентификации

	if _, err := cryptobox.Decrypt(key, ciphertext); !errors.Is(err, cryptobox.ErrCiphertext) {
		t.Errorf("ожидалась ErrCiphertext при подделке, получено %v", err)
	}
}

func TestKeySizeValidation(t *testing.T) {
	if _, err := cryptobox.Encrypt([]byte("short"), []byte("x")); !errors.Is(err, cryptobox.ErrKeySize) {
		t.Errorf("Encrypt: ожидалась ErrKeySize, получено %v", err)
	}
	if _, err := cryptobox.Decrypt([]byte("short"), []byte("x")); !errors.Is(err, cryptobox.ErrKeySize) {
		t.Errorf("Decrypt: ожидалась ErrKeySize, получено %v", err)
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := cryptobox.DeriveKey("master", "alice")
	if _, err := cryptobox.Decrypt(key, []byte("tiny")); !errors.Is(err, cryptobox.ErrCiphertext) {
		t.Errorf("ожидалась ErrCiphertext для короткого шифртекста, получено %v", err)
	}
}
