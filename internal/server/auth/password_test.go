package auth_test

import (
	"strings"
	"testing"

	"github.com/warenik/gophkeeper/internal/server/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == password {
		t.Fatal("пароль не должен храниться в открытом виде")
	}

	if err := auth.CheckPassword(hash, password); err != nil {
		t.Errorf("верный пароль отклонён: %v", err)
	}
	if err := auth.CheckPassword(hash, "wrong"); err == nil {
		t.Error("неверный пароль принят")
	}
}

// TestHashLongPassword проверяет, что пароль длиннее 72 байт (лимит bcrypt)
// корректно обрабатывается за счёт предварительного хеширования.
func TestHashLongPassword(t *testing.T) {
	long := strings.Repeat("a", 100)

	hash, err := auth.HashPassword(long)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := auth.CheckPassword(hash, long); err != nil {
		t.Errorf("длинный пароль отклонён: %v", err)
	}
	// Пароль, совпадающий по первым 72 байтам, но отличающийся дальше, должен
	// быть отклонён (предхеш охватывает всю длину).
	if err := auth.CheckPassword(hash, strings.Repeat("a", 72)+"different"); err == nil {
		t.Error("пароль, отличающийся после 72 байт, ошибочно принят")
	}
}
