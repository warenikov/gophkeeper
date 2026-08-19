// Пакет slicesx содержит небольшие обобщённые (generic) помощники для срезов.
package slicesx

// Map применяет f к каждому элементу s и возвращает срез результатов.
func Map[T, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}
