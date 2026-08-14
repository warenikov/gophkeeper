package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/warenikov/gophkeeper/internal/server/auth"
)

func TestIssueAndParse(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)

	token, err := tm.Issue("user-42")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	userID, err := tm.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if userID != "user-42" {
		t.Errorf("userID = %q, ожидалось %q", userID, "user-42")
	}
}

func TestParseExpired(t *testing.T) {
	tm := auth.NewTokenManager("secret", -time.Minute) // уже истёк

	token, err := tm.Issue("user-42")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := tm.Parse(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("ожидалась ErrInvalidToken для истёкшего токена, получено %v", err)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, err := auth.NewTokenManager("secret", time.Hour).Issue("user-42")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	other := auth.NewTokenManager("другой-секрет", time.Hour)
	if _, err := other.Parse(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("ожидалась ErrInvalidToken при неверном секрете, получено %v", err)
	}
}

func TestParseMalformed(t *testing.T) {
	tm := auth.NewTokenManager("secret", time.Hour)
	if _, err := tm.Parse("не.токен.вовсе"); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("ожидалась ErrInvalidToken для мусора, получено %v", err)
	}
}
