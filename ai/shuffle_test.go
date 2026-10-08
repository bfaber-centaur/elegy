package ai

import (
	"slices"
	"testing"
)

func TestShufflePlanets(t *testing.T) {
	r := &script{t: t, draws: []int{2, 0}}
	got := ShufflePlanets([]int{30, 10, 20}, r)
	// Sorted 10 20 30; swap 0 with 0+2 → 30 20 10; swap 1 with 1+0.
	if !slices.Equal(got, []int{30, 20, 10}) {
		t.Errorf("order %v", got)
	}
	if !slices.Equal(r.bounds, []int{3, 2}) {
		t.Errorf("draws Random%v, want Random(3), Random(2)", r.bounds)
	}
	if got := ShufflePlanets([]int{7}, &script{t: t}); !slices.Equal(got, []int{7}) {
		t.Errorf("one planet: %v", got)
	}
}
