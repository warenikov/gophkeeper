// Пакет cache хранит локальную копию записей пользователя для синхронизации и
// работы в офлайн-режиме (read-only). Полезная нагрузка остаётся зашифрованной.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Secret — запись в локальном кэше (нагрузка хранится зашифрованной).
type Secret struct {
	ID               string    `json:"id"`
	Type             int32     `json:"type"`
	Name             string    `json:"name"`
	Metadata         string    `json:"metadata"`
	EncryptedPayload []byte    `json:"encrypted_payload"`
	Version          int64     `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Cache — локальный кэш записей одного пользователя с курсором синхронизации.
type Cache struct {
	Login    string            `json:"login"`
	Revision int64             `json:"revision"`
	Secrets  map[string]Secret `json:"secrets"`
}

// dir возвращает каталог хранения кэша (тот же, что и у сессии).
func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cache: user config dir: %w", err)
	}
	return filepath.Join(base, "gophkeeper"), nil
}

// filePath возвращает путь к файлу кэша конкретного пользователя.
func filePath(login string) (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cache_"+login+".json"), nil
}

// Load читает кэш пользователя. Если файла нет, возвращает пустой кэш.
func Load(login string) (Cache, error) {
	p, err := filePath(login)
	if err != nil {
		return Cache{}, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Cache{Login: login, Secrets: map[string]Secret{}}, nil
		}
		return Cache{}, fmt.Errorf("cache: read: %w", err)
	}

	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return Cache{}, fmt.Errorf("cache: unmarshal: %w", err)
	}
	if c.Secrets == nil {
		c.Secrets = map[string]Secret{}
	}
	return c, nil
}

// Save сохраняет кэш пользователя с правами 0600.
func Save(c Cache) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return fmt.Errorf("cache: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}

	p, err := filePath(c.Login)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("cache: write: %w", err)
	}
	return nil
}

// Sorted возвращает записи кэша, отсортированные по времени обновления.
func (c Cache) Sorted() []Secret {
	out := make([]Secret, 0, len(c.Secrets))
	for _, s := range c.Secrets {
		out = append(out, s)
	}
	// Простая сортировка по UpdatedAt (пузырьком не нужно — используем slices).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].UpdatedAt.After(out[j].UpdatedAt); j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
