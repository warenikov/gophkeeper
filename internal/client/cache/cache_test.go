package cache_test

import (
	"testing"
	"time"

	"github.com/warenik/gophkeeper/internal/client/cache"
)

func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
}

func TestSaveAndLoad(t *testing.T) {
	isolate(t)

	want := cache.Cache{
		Login:    "alice",
		Revision: 7,
		Secrets: map[string]cache.Secret{
			"s1": {ID: "s1", Type: 2, Name: "note", EncryptedPayload: []byte("x"), Version: 1},
		},
	}
	if err := cache.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := cache.Load("alice")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Revision != 7 || len(got.Secrets) != 1 || got.Secrets["s1"].Name != "note" {
		t.Errorf("загружен неверный кэш: %+v", got)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	isolate(t)

	got, err := cache.Load("nobody")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Login != "nobody" || len(got.Secrets) != 0 {
		t.Errorf("ожидался пустой кэш, получено %+v", got)
	}
}

func TestSortedByUpdatedAt(t *testing.T) {
	base := time.Now()
	c := cache.Cache{Secrets: map[string]cache.Secret{
		"b": {ID: "b", UpdatedAt: base.Add(time.Hour)},
		"a": {ID: "a", UpdatedAt: base},
	}}

	sorted := c.Sorted()
	if len(sorted) != 2 || sorted[0].ID != "a" || sorted[1].ID != "b" {
		t.Errorf("неверный порядок сортировки: %+v", sorted)
	}
}
