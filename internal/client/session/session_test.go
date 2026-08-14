package session_test

import (
	"errors"
	"testing"

	"github.com/warenik/gophkeeper/internal/client/session"
)

// isolate перенаправляет каталог конфигурации во временную директорию теста.
func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)            // macOS, Linux fallback
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("AppData", dir)         // Windows
}

func TestSaveAndLoad(t *testing.T) {
	isolate(t)

	want := session.Session{ServerAddress: "localhost:8081", Login: "alice", Token: "tok"}
	if err := session.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := session.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("загружено %+v, ожидалось %+v", got, want)
	}
}

func TestLoadNoSession(t *testing.T) {
	isolate(t)
	if _, err := session.Load(); !errors.Is(err, session.ErrNoSession) {
		t.Errorf("ожидалась ErrNoSession, получено %v", err)
	}
}

func TestClear(t *testing.T) {
	isolate(t)

	if err := session.Save(session.Session{Login: "alice"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := session.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := session.Load(); !errors.Is(err, session.ErrNoSession) {
		t.Errorf("после Clear ожидалась ErrNoSession, получено %v", err)
	}
	// Повторный Clear не должен возвращать ошибку.
	if err := session.Clear(); err != nil {
		t.Errorf("повторный Clear вернул ошибку: %v", err)
	}
}
