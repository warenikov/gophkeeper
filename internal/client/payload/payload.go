// Пакет payload сериализует полезную нагрузку записей клиента перед шифрованием
// и восстанавливает её после расшифровки.
package payload

import (
	"encoding/json"
	"fmt"
)

// LoginPassword — полезная нагрузка записи типа «логин/пароль».
type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"` //nolint:gosec // это поле полезной нагрузки; шифруется перед отправкой
}

// EncodeLoginPassword сериализует пару логин/пароль в JSON.
func EncodeLoginPassword(lp LoginPassword) ([]byte, error) {
	data, err := json.Marshal(lp)
	if err != nil {
		return nil, fmt.Errorf("payload: marshal login/password: %w", err)
	}
	return data, nil
}

// DecodeLoginPassword восстанавливает пару логин/пароль из JSON.
func DecodeLoginPassword(data []byte) (LoginPassword, error) {
	var lp LoginPassword
	if err := json.Unmarshal(data, &lp); err != nil {
		return LoginPassword{}, fmt.Errorf("payload: unmarshal login/password: %w", err)
	}
	return lp, nil
}

// EncodeText представляет произвольный текст как байты полезной нагрузки.
func EncodeText(text string) []byte {
	return []byte(text)
}

// DecodeText восстанавливает текст из байтов полезной нагрузки.
func DecodeText(data []byte) string {
	return string(data)
}
