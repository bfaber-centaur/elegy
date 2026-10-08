package ai

import "testing"

// script is a Rand that returns scripted draws and records each bound.
type script struct {
	t      *testing.T
	draws  []int
	bounds []int
}

func (s *script) Intn(n int) int {
	s.bounds = append(s.bounds, n)
	if len(s.draws) == 0 {
		s.t.Fatalf("unexpected draw Random(%d)", n)
	}
	v := s.draws[0]
	s.draws = s.draws[1:]
	if v < 0 || v >= n {
		s.t.Fatalf("scripted draw %d outside Random(%d)", v, n)
	}
	return v
}
