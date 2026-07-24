// Пакет buildmeta предоставляет метаданные сборки (версию, дату сборки и
// commit VCS), которые внедряются в бинарники через -ldflags на этапе
// компиляции.
//
// Переменные намеренно сделаны пакетного уровня и устанавливаемыми линкером,
// чтобы единый источник истины переиспользовался и сервером, и клиентом. При
// сборке без ldflags (например, под `go run` или `go test`) значения
// откатываются к «dev»-умолчаниям, заданным ниже.
package buildmeta

import "fmt"

// Метаданные сборки. Эти переменные переопределяются на этапе сборки так:
//
//	go build -ldflags "\
//	  -X github.com/warenik/gophkeeper/internal/buildmeta.Version=v1.0.0 \
//	  -X github.com/warenik/gophkeeper/internal/buildmeta.BuildDate=2026-07-24 \
//	  -X github.com/warenik/gophkeeper/internal/buildmeta.Commit=abc1234"
var (
	// Version — семантическая версия сборки (например, "v1.0.0").
	Version = "dev"
	// BuildDate — дата сборки бинарника в UTC (например, "2026-07-24").
	BuildDate = "unknown"
	// Commit — короткая ревизия VCS, из которой собран бинарник.
	Commit = "unknown"
)

// String возвращает одну человекочитаемую строку с описанием сборки, пригодную
// для вывода в ответ на флаг `--version`.
func String() string {
	return fmt.Sprintf("version=%s date=%s commit=%s", Version, BuildDate, Commit)
}
