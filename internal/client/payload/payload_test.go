package payload_test

import (
	"testing"

	"github.com/warenik/gophkeeper/internal/client/payload"
)

func TestLoginPasswordRoundtrip(t *testing.T) {
	want := payload.LoginPassword{Login: "alice", Password: "s3cret"}

	data, err := payload.EncodeLoginPassword(want)
	if err != nil {
		t.Fatalf("EncodeLoginPassword: %v", err)
	}

	got, err := payload.DecodeLoginPassword(data)
	if err != nil {
		t.Fatalf("DecodeLoginPassword: %v", err)
	}
	if got != want {
		t.Errorf("после roundtrip = %+v, ожидалось %+v", got, want)
	}
}

func TestDecodeLoginPasswordInvalid(t *testing.T) {
	if _, err := payload.DecodeLoginPassword([]byte("not json")); err == nil {
		t.Error("ожидалась ошибка на некорректном JSON")
	}
}

func TestTextRoundtrip(t *testing.T) {
	const want = "произвольный текст"
	if got := payload.DecodeText(payload.EncodeText(want)); got != want {
		t.Errorf("после roundtrip = %q, ожидалось %q", got, want)
	}
}
