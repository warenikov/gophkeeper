// Пакет model содержит доменные типы сервера GophKeeper и sentinel-ошибки,
// общие для слоёв сервиса и хранилища.
package model

import (
	"errors"
	"time"
)

// Доменные sentinel-ошибки. Слой gRPC отображает их в коды ответа.
var (
	// ErrUserExists — пользователь с таким логином уже зарегистрирован.
	ErrUserExists = errors.New("model: user already exists")
	// ErrInvalidCredentials — неверная пара логин/пароль.
	ErrInvalidCredentials = errors.New("model: invalid credentials")
	// ErrNotFound — запрошенная сущность не найдена.
	ErrNotFound = errors.New("model: not found")
	// ErrVersionConflict — версия записи не совпала с ожидаемой (конфликт).
	ErrVersionConflict = errors.New("model: version conflict")
	// ErrValidation — некорректные входные данные запроса.
	ErrValidation = errors.New("model: validation failed")
)

// User — зарегистрированный пользователь. PasswordHash хранит bcrypt-хеш.
type User struct {
	ID           string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}

// SecretType — тип приватной записи.
type SecretType int16

// Поддерживаемые типы записей. Zero-значение — Unspecified.
const (
	SecretTypeUnspecified   SecretType = 0
	SecretTypeLoginPassword SecretType = 1
	SecretTypeText          SecretType = 2
	SecretTypeCard          SecretType = 3
	SecretTypeBinary        SecretType = 4
)

// Secret — приватная запись владельца. EncryptedPayload зашифрован на клиенте;
// сервер хранит его как непрозрачные байты.
type Secret struct {
	ID               string
	OwnerID          string
	Type             SecretType
	Name             string
	Metadata         string
	EncryptedPayload []byte
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	// Deleted — признак тумбстона (мягкое удаление) для синхронизации.
	Deleted bool
	// Revision — монотонный курсор изменений для pull-синхронизации.
	Revision int64
}
