package buildmeta_test

import (
	"strings"
	"testing"

	"github.com/warenikov/gophkeeper/internal/buildmeta"
)

// TestStringContainsAllFields проверяет, что String возвращает строку со всеми
// тремя полями метаданных сборки в формате key=value.
func TestStringContainsAllFields(t *testing.T) {
	got := buildmeta.String()

	for _, part := range []string{"version=", "date=", "commit="} {
		if !strings.Contains(got, part) {
			t.Errorf("String() = %q, не содержит %q", got, part)
		}
	}
}

// TestStringReflectsValues проверяет, что String подставляет актуальные значения
// переменных сборки.
func TestStringReflectsValues(t *testing.T) {
	// Сохраняем и восстанавливаем исходные значения, чтобы не влиять на другие
	// тесты пакета.
	origVersion, origDate, origCommit := buildmeta.Version, buildmeta.BuildDate, buildmeta.Commit
	t.Cleanup(func() {
		buildmeta.Version = origVersion
		buildmeta.BuildDate = origDate
		buildmeta.Commit = origCommit
	})

	buildmeta.Version = "v1.2.3"
	buildmeta.BuildDate = "2026-07-29"
	buildmeta.Commit = "abc1234"

	want := "version=v1.2.3 date=2026-07-29 commit=abc1234"
	if got := buildmeta.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
