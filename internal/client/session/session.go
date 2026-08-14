// Пакет session хранит локальное состояние клиента GophKeeper (адрес сервера,
// логин и токен доступа) в файле в пользовательском каталоге конфигурации.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoSession — локальная сессия отсутствует (пользователь не вошёл).
var ErrNoSession = errors.New("session: not authenticated, run login first")

// Session — сохранённое состояние авторизованного клиента.
type Session struct {
	ServerAddress string `json:"server_address"`
	Login         string `json:"login"`
	Token         string `json:"token"`
}

// dir возвращает каталог хранения состояния (~/.config/gophkeeper или
// эквивалент ОС).
func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("session: user config dir: %w", err)
	}
	return filepath.Join(base, "gophkeeper"), nil
}

// path возвращает путь к файлу сессии.
func path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "session.json"), nil
}

// Load читает сохранённую сессию. Возвращает ErrNoSession, если файла нет.
func Load() (Session, error) {
	p, err := path()
	if err != nil {
		return Session{}, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, ErrNoSession
		}
		return Session{}, fmt.Errorf("session: read: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, fmt.Errorf("session: unmarshal: %w", err)
	}
	return s, nil
}

// Save сохраняет сессию в файл с правами 0600 (только владелец).
func Save(s Session) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return fmt.Errorf("session: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}

	p := filepath.Join(d, "session.json")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("session: write: %w", err)
	}
	return nil
}

// Clear удаляет сохранённую сессию. Отсутствие файла не считается ошибкой.
func Clear() error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("session: remove: %w", err)
	}
	return nil
}
