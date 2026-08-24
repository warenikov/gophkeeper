// Пакет slicesx содержит небольшие обобщённые (generic) помощники для
// последовательностей.
package slicesx

import "iter"

// Map возвращает ленивый итератор, применяющий f к каждому элементу seq.
//
// В отличие от варианта, возвращающего срез, промежуточный срез не аллоцируется:
// элементы вычисляются по мере потребления. Это выгодно в цепочках
// трансформаций (Map→Filter→…); материализовать результат в срез можно через
// slices.Collect.
func Map[T, U any](seq iter.Seq[T], f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}
