// Package oraclerng tells the parity harness where an oracle run's random
// stream started, so Elegy can draw the same numbers the original drew.
//
// The original seeds its generator once, at startup, from the Windows tick
// count. The oracle runs each generated year in a fresh DOSBox process with
// fixed cycles, which pins the startup tick, and the vectors name each
// stream by that cycles value ("cycles 20000"). This package says where a
// stream started (Start) and maps a stream label to its startup tick.
//
// The generator itself is not yet a public rule (stars-elegy KERNEL.md
// d4eafd9 specifies only rand(n), a uniform draw in 0..n−1, and the order
// of the year's draws). Until it is, callers have no Seeder and every
// random outcome compares only as a sample.
package oraclerng

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Rand is a random source with the engine's draw signature (engine.Rand):
// Intn returns a draw in 0..n−1. It is repeated here so the engine's own
// tests can use this package.
type Rand interface {
	Intn(n int) int
}

// Start is where one oracle year's random stream started: the startup
// tick that seeded the generator, the generator's state before the year's
// first draw, or both. The zero Start records nothing.
type Start struct {
	// Tick is the startup tick in milliseconds, when HasTick.
	Tick    int
	HasTick bool
	// State is the generator's state before the year's first draw, in the
	// terms of the published generator; nil when not recorded.
	State json.RawMessage
}

// Recorded reports whether s records anything to start a stream from.
func (s Start) Recorded() bool {
	return s.HasTick || len(s.State) > 0 && string(s.State) != "null"
}

// Seeder returns the original's random stream positioned at s, or false
// when it cannot start from what s records.
type Seeder func(s Start) (Rand, bool)

// Cycles returns the DOSBox cycles value of a vector stream label. Every
// stream label in the stars-elegy vectors at d4eafd9 has the form
// "cycles 20000"; vectors/README.md "Randomness" does not fix the format.
func Cycles(label string) (int, bool) {
	s, ok := strings.CutPrefix(label, "cycles ")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// ticks maps a fixed DOSBox cycles value to the startup tick it gave, as
// identified by replaying the oracle's random outcomes. Source: stars-elegy
// docs/ORACLE.md "Pinned battle RNG", "Which stream a cycles value reaches"
// at d4eafd9 (KX-004 random events and TK-307 scrap-tech replays),
// MEASURED. Each tick is trunc(k·54.925) for a small integer k. Only the
// listed values are included; for the range "1985–1955 → 2032" only the
// two ends.
//
// ASSUMPTION: a cycles value gives the same tick for every start file. The
// table pools runs from different starts (KX-004, TK-307), and no cycles
// value was measured on two starts with different results, but no value
// was measured on both either.
var ticks = map[int]int{
	// KX-004.
	35000: 109, 45000: 109,
	11500: 329,
	10500: 384, 9800: 384,
	6000: 659,
	5200: 768,
	3700: 1098,
	2260: 1812, 2190: 1812,
	1985: 2032, 1955: 2032,
	1750: 2306, 1710: 2306,
	1490: 2691,
	1210: 3295,
	1190: 3405, 1170: 3405, 1160: 3405,
	1165: 3460, 1155: 3460,
	1135: 3570, 1130: 3570, 1090: 3570,
	930: 4284,
	890: 4613,
	880: 4503,
	// TK-307.
	30000: 109, 50000: 109,
	20000: 164,
	17000: 219,
	15000: 274,
	12000: 329,
	10000: 384,
	8800:  439,
	7100:  604, 6500: 604,
	5500: 714,
	4300: 933,
}

// Tick returns the startup tick a cycles value is known to give.
func Tick(cycles int) (int, bool) {
	t, ok := ticks[cycles]
	return t, ok
}

// StreamTick returns the startup tick of a vector stream label.
func StreamTick(label string) (int, bool) {
	c, ok := Cycles(label)
	if !ok {
		return 0, false
	}
	return Tick(c)
}

// Recorder passes draws through to R and records each draw's bound n, so a
// test can compare the order of a year's draws with KERNEL.md "Random
// draws".
type Recorder struct {
	R      Rand
	Bounds []int
}

// Intn draws from R and records n.
func (r *Recorder) Intn(n int) int {
	r.Bounds = append(r.Bounds, n)
	return r.R.Intn(n)
}
