package otp_test

import (
	"testing"
	"time"

	"github.com/warenikov/gophkeeper/internal/client/otp"
)

// secretRFC — base32 от ASCII-строки "12345678901234567890" (seed SHA1 из
// тестовых векторов RFC 6238, Приложение B).
const secretRFC = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// TestGenerateTOTP_RFC6238 сверяет реализацию с эталонными векторами RFC 6238
// (SHA1, 8 цифр, окно 30 с).
func TestGenerateTOTP_RFC6238(t *testing.T) {
	cases := []struct {
		unix int64
		want string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}

	for _, tc := range cases {
		got, err := otp.GenerateTOTP(secretRFC, time.Unix(tc.unix, 0).UTC(), 30, 8)
		if err != nil {
			t.Fatalf("GenerateTOTP(%d): %v", tc.unix, err)
		}
		if got != tc.want {
			t.Errorf("TOTP(T=%d) = %s, ожидалось %s", tc.unix, got, tc.want)
		}
	}
}

func TestCode(t *testing.T) {
	// В начале окна (unix кратен 30) остаётся полные 30 секунд.
	code, remaining, err := otp.Code(secretRFC, time.Unix(30, 0).UTC())
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if len(code) != otp.DefaultDigits {
		t.Errorf("длина кода = %d, ожидалось %d", len(code), otp.DefaultDigits)
	}
	if remaining != otp.DefaultPeriod {
		t.Errorf("remaining = %d, ожидалось %d", remaining, otp.DefaultPeriod)
	}

	// В середине окна остаётся меньше периода.
	_, remaining2, _ := otp.Code(secretRFC, time.Unix(45, 0).UTC())
	if remaining2 != 15 {
		t.Errorf("remaining = %d, ожидалось 15", remaining2)
	}
}

func TestGenerateTOTPInvalidSecret(t *testing.T) {
	if _, err := otp.GenerateTOTP("не-base32-!!!", time.Now(), 30, 6); err == nil {
		t.Error("ожидалась ошибка на некорректном секрете")
	}
	if _, err := otp.GenerateTOTP("", time.Now(), 30, 6); err == nil {
		t.Error("ожидалась ошибка на пустом секрете")
	}
}

func TestNormalizationInsensitive(t *testing.T) {
	// Нижний регистр и пробелы должны давать тот же код.
	upper, _ := otp.GenerateTOTP(secretRFC, time.Unix(59, 0).UTC(), 30, 8)
	lower, err := otp.GenerateTOTP("gezd gnbv gy3t qojq gezd gnbv gy3t qojq", time.Unix(59, 0).UTC(), 30, 8)
	if err != nil {
		t.Fatalf("GenerateTOTP(lower): %v", err)
	}
	if upper != lower {
		t.Errorf("код зависит от регистра/пробелов: %s != %s", upper, lower)
	}
}
