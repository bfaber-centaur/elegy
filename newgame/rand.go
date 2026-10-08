package newgame

import "math/bits"

// Source is Elegy's seeded random generator for new games: SplitMix64
// with Lemire's unbiased bounded draw. It satisfies engine.Rand.
//
// ELEGY CHOICE: UNIVERSE.md "Randomness and seeds" leaves the original's
// random stream and draw order out of the spec. The original makes a
// seeded game a pure function of its definition file; Elegy keeps that
// property (the same Settings and seed give the same game) with its own
// generator and its own draw order. Only the rules and distributions are
// the original's.
type Source struct{ s uint64 }

// NewRand returns a generator seeded with seed.
func NewRand(seed uint64) *Source { return &Source{s: seed} }

func (r *Source) next() uint64 {
	r.s += 0x9e3779b97f4a7c15
	z := r.s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// Intn returns a uniform integer in 0..n−1. It panics when n ≤ 0.
func (r *Source) Intn(n int) int {
	if n <= 0 {
		panic("newgame: Intn of a non-positive bound")
	}
	bound := uint64(n)
	threshold := -bound % bound
	for {
		hi, lo := bits.Mul64(r.next(), bound)
		if lo >= threshold {
			return int(hi)
		}
	}
}

// State is the generator's whole state: NewRand(r.State()) continues the
// stream exactly where r is, so a saved game can resume its draws.
func (r *Source) State() uint64 { return r.s }
