package slicesx_test

import (
	"strconv"
	"testing"

	"github.com/warenikov/gophkeeper/internal/slicesx"
)

func TestMap(t *testing.T) {
	got := slicesx.Map([]int{1, 2, 3}, strconv.Itoa)
	want := []string{"1", "2", "3"}
	if len(got) != len(want) {
		t.Fatalf("длина = %d, ожидалось %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, ожидалось %q", i, got[i], want[i])
		}
	}
}

func TestMapEmpty(t *testing.T) {
	if got := slicesx.Map([]int{}, strconv.Itoa); len(got) != 0 {
		t.Errorf("для пустого среза ожидался пустой результат, получено %v", got)
	}
}
