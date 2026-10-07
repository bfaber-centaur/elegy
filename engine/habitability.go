package engine

import "math"

// Habitability returns a planet's habitability percentage for a race:
// 1..100 for habitable environments (0 is possible at the edges), negative
// (−1..−45) for hostile ones. env is the planet's current environment on
// the 0–100 internal scale.
//
// KERNEL.md "Habitability": CONFIRMED at the race's center (100, PG) and
// at the six KX-002 points (H1–H6, hostile planets included); other points
// and the cap of 15 per hostile axis are BINARY-ONLY.
func Habitability(race Race, env [3]int) int {
	hostile := 0
	for axis, r := range race.Env {
		if r.Immune {
			continue
		}
		v := env[axis]
		switch {
		case v < r.Low:
			hostile += min(15, r.Low-v)
		case v > r.High:
			hostile += min(15, v-r.High)
		}
	}
	if hostile > 0 {
		return -hostile
	}

	s, m := 0, 10000
	for axis, r := range race.Env {
		if r.Immune {
			s += 10000
			continue
		}
		v := env[axis]
		d := abs(v - r.Center)
		w := r.High - r.Center
		if v < r.Center {
			w = r.Center - r.Low
		}
		e := 100
		if d > 0 {
			e = 100 - 100*d/w
		}
		s += e * e
		if over := 2*d - w; over > 0 {
			m = m * (2*w - over) / (2 * w)
		}
	}
	x := int(math.Sqrt(float64(s)/3) + 0.9)
	return x * m / 10000
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
