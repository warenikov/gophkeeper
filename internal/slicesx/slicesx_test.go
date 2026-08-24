package slicesx_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/warenikov/gophkeeper/internal/slicesx"
)

func TestMap(t *testing.T) {
	got := slices.Collect(slicesx.Map(slices.Values([]int{1, 2, 3}), strconv.Itoa))
	want := []string{"1", "2", "3"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, ожидалось %v", got, want)
	}
}

func TestMapEmpty(t *testing.T) {
	if got := slices.Collect(slicesx.Map(slices.Values([]int{}), strconv.Itoa)); len(got) != 0 {
		t.Errorf("для пустой последовательности ожидался пустой результат, получено %v", got)
	}
}

// TestMapLazy проверяет ленивость: при досрочной остановке потребления f
// вычисляется только для потреблённых элементов, промежуточный срез не строится.
func TestMapLazy(t *testing.T) {
	calls := 0
	seq := slicesx.Map(slices.Values([]int{1, 2, 3, 4, 5}), func(n int) int {
		calls++
		return n * 10
	})

	for v := range seq {
		if v == 20 { // останавливаемся после второго элемента
			break
		}
	}
	if calls != 2 {
		t.Errorf("f вызвана %d раз, ожидалось 2 (ленивое вычисление)", calls)
	}
}
