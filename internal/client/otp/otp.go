// Пакет otp реализует генерацию одноразовых паролей TOTP (RFC 6238) на клиенте.
// Секрет (seed) хранится зашифрованным как обычная запись — код вычисляется
// локально, поэтому сервер seed в открытом виде не видит (zero-knowledge).
package otp

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238/HOTP определён на HMAC-SHA1; это требование стандарта
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// Параметры TOTP по умолчанию (как в Google Authenticator).
const (
	DefaultPeriod = 30 // секунд
	DefaultDigits = 6
)

// GenerateTOTP возвращает TOTP-код для секрета в кодировке base32 на момент t.
// period — длина окна в секундах, digits — число цифр кода.
func GenerateTOTP(base32Secret string, t time.Time, period uint, digits int) (string, error) {
	key, err := decodeSecret(base32Secret)
	if err != nil {
		return "", err
	}
	if period == 0 {
		return "", fmt.Errorf("otp: period must be positive")
	}

	counter := uint64(t.Unix()) / uint64(period) //nolint:gosec // время TOTP положительно
	return hotp(key, counter, digits), nil
}

// Code возвращает текущий TOTP-код и число секунд до его смены (окно 30 с,
// 6 цифр).
func Code(base32Secret string, now time.Time) (code string, remaining int, err error) {
	code, err = GenerateTOTP(base32Secret, now, DefaultPeriod, DefaultDigits)
	if err != nil {
		return "", 0, err
	}
	remaining = DefaultPeriod - int(now.Unix()%int64(DefaultPeriod))
	return code, remaining, nil
}

// decodeSecret декодирует секрет base32, допуская пробелы, нижний регистр и
// отсутствие выравнивания (как в QR-кодах аутентификаторов).
func decodeSecret(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(secret))
	normalized = strings.TrimRight(normalized, "=")
	if normalized == "" {
		return nil, fmt.Errorf("otp: empty secret")
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("otp: decode base32 secret: %w", err)
	}
	return key, nil
}

// hotp вычисляет HOTP-код (RFC 4226) для ключа и счётчика.
func hotp(key []byte, counter uint64, digits int) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	// Динамическое усечение (RFC 4226 §5.3).
	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	mod := uint32(1)
	for range digits {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, binCode%mod)
}
