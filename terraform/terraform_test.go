package terraform

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// race is the KX-002 vectors' habitat: 50 (15..85) on every axis.
func race() engine.Race {
	r := engine.Race{PRT: engine.PRTJackOfAllTrades}
	for a := range r.Env {
		r.Env[a] = engine.EnvRange{Center: 50, Low: 15, High: 85}
	}
	return r
}

func levels(energy, weapons, propulsion, construction, electronics, biotech int) [engine.NumFields]int {
	var l [engine.NumFields]int
	l[engine.Energy], l[engine.Weapons], l[engine.Propulsion] = energy, weapons, propulsion
	l[engine.Construction], l[engine.Electronics], l[engine.Biotech] = construction, electronics, biotech
	return l
}

func planet(env, orig [3]int) *engine.Planet { return &engine.Planet{Env: env, OrigEnv: orig} }

// Reach per axis (KERNEL.md "Terraforming"; CONFIRMED KX-002 T1–T3,
// KX-005, KB-2C).
func TestConfirmedReach(t *testing.T) {
	r := race()
	// KX-002 T1/T3: tech 2,0,1,0,5,1 gives Gravity and Temp ±3, no
	// radiation part.
	if got := Reach(r, levels(2, 0, 1, 0, 5, 1)); got != [3]int{3, 3, 0} {
		t.Errorf("T1 reach %v", got)
	}
	// KX-002 T2: TT at biotech 0 gives Total Terraform ±3 on every axis.
	tt := r
	tt.LRT.TotalTerraforming = true
	if got := Reach(tt, levels(2, 0, 0, 0, 5, 0)); got != [3]int{3, 3, 3} {
		t.Errorf("T2 reach %v", got)
	}
	// Without TT the Total Terraform parts are not available.
	if got := Reach(r, levels(2, 0, 0, 0, 5, 0)); got != [3]int{} {
		t.Errorf("no TT reach %v", got)
	}
	// KX-005's adjuster example: gravity 11, temperature 7, radiation 3.
	if got := Reach(r, levels(5, 1, 10, 0, 0, 3)); got != [3]int{11, 7, 3} {
		t.Errorf("adjuster reach %v", got)
	}
	// An immune axis has no reach (KB-2C).
	im := r
	im.Env[Gravity] = engine.EnvRange{Immune: true}
	if got := Reach(im, levels(2, 0, 1, 0, 5, 1)); got != [3]int{0, 3, 0} {
		t.Errorf("immune reach %v", got)
	}
}

// Capacity and the order clip (CONFIRMED KX-002 T1–T3, KB-2C).
func TestConfirmedCapacityAndClip(t *testing.T) {
	r := race()
	t1 := Reach(r, levels(2, 0, 1, 0, 5, 1))
	type clipCase struct {
		name      string
		p         *engine.Planet
		race      engine.Race
		reach     [3]int
		order     int
		cap, n    int
		clip, rem bool
	}
	cases := []clipCase{
		{"T1 60/50/50", planet([3]int{60, 50, 50}, [3]int{60, 50, 50}), r, t1, 5, 3, 3, true, false},
		{"T3 58 of orig 60", planet([3]int{58, 50, 50}, [3]int{60, 50, 50}), r, t1, 5, 1, 1, true, false},
		{"T2 60/45/50 TT", planet([3]int{60, 45, 50}, [3]int{60, 45, 50}), r, [3]int{3, 3, 3}, 5, 6, 5, false, false},
	}
	im := r
	im.Env[Gravity] = engine.EnvRange{Immune: true}
	imReach := Reach(im, levels(2, 0, 1, 0, 5, 1))
	cases = append(cases,
		clipCase{"KB-2C 20/47/50", planet([3]int{20, 47, 50}, [3]int{20, 47, 50}), im, imReach, 5, 3, 3, true, false},
		clipCase{"KB-2C 10/50/50", planet([3]int{10, 50, 50}, [3]int{10, 50, 50}), im, imReach, 2, 0, 0, true, true},
	)
	for _, c := range cases {
		cp := Capacity(*c.p, c.race, c.reach)
		n, clip, rem := ClipOrder(c.order, cp)
		if cp != c.cap || n != c.n || clip != c.clip || rem != c.rem {
			t.Errorf("%s: capacity %d, clip ×%d → %d %v %v", c.name, cp, c.order, n, clip, rem)
		}
	}
}

// Units built one click at a time (CONFIRMED KX-002 T1–T3, KX-005,
// KB-2C).
func TestConfirmedImprove(t *testing.T) {
	r := race()
	t1 := Reach(r, levels(2, 0, 1, 0, 5, 1))
	click := func(p *engine.Planet, race engine.Race, reach [3]int, n int) []int {
		var axes []int
		for range n {
			axes = append(axes, Improve(p, race, reach))
		}
		return axes
	}
	// T1: two units built (200 resources) move gravity 60 → 58.
	p := planet([3]int{60, 50, 50}, [3]int{60, 50, 50})
	click(p, r, t1, 2)
	if p.Env != [3]int{58, 50, 50} {
		t.Errorf("T1: %v", p.Env)
	}
	// T3: 58 of 60 reaches 57, then no room.
	p = planet([3]int{58, 50, 50}, [3]int{60, 50, 50})
	if axes := click(p, r, t1, 2); p.Env != [3]int{57, 50, 50} || axes[1] != -1 {
		t.Errorf("T3: %v %v", p.Env, axes)
	}
	// T2: temperature scores 101 against gravity's 67; both units go to
	// temperature.
	p = planet([3]int{60, 45, 50}, [3]int{60, 45, 50})
	if s := [2]int{score(r, p.Env, Gravity, 57), score(r, p.Env, Temperature, 48)}; s != [2]int{67, 101} {
		t.Errorf("T2 scores %v", s)
	}
	click(p, r, [3]int{3, 3, 3}, 2)
	if p.Env != [3]int{60, 47, 50} {
		t.Errorf("T2: %v", p.Env)
	}
	// KX-005: gravity at the centre, temperature and radiation tied; the
	// first axis wins.
	p = planet([3]int{50, 60, 60}, [3]int{50, 60, 60})
	click(p, r, [3]int{3, 3, 3}, 1)
	if p.Env != [3]int{50, 59, 60} {
		t.Errorf("KX-005 tie: %v", p.Env)
	}
	// KB-2C: a gravity-immune race reaches 20/50/50 from 20/47/50.
	im := r
	im.Env[Gravity] = engine.EnvRange{Immune: true}
	p = planet([3]int{20, 47, 50}, [3]int{20, 47, 50})
	click(p, im, Reach(im, levels(2, 0, 1, 0, 5, 1)), 3)
	if p.Env != [3]int{20, 50, 50} {
		t.Errorf("KB-2C: %v", p.Env)
	}
}

// Unit cost (CONFIRMED KX-002 T1, T2, KX-005; CA with TT ASSUMPTION T2).
func TestConfirmedUnitCost(t *testing.T) {
	r := race()
	tt := r
	tt.LRT.TotalTerraforming = true
	ca := r
	ca.PRT = engine.PRTClaimAdjuster
	catt := ca
	catt.LRT.TotalTerraforming = true
	if got := [4]int{UnitCost(r), UnitCost(tt), UnitCost(ca), UnitCost(catt)}; got != [4]int{100, 70, 50, 35} {
		t.Errorf("costs %v", got)
	}
}

// Auto Max and Auto Min Terraform (CONFIRMED KX-005).
func TestConfirmedAutoUnits(t *testing.T) {
	cases := []struct {
		count, capacity int
		minOnly         bool
		popChange, hab  int
		want            int
	}{
		{9, 6, false, 100, 50, 6}, // Auto Max ×9 with capacity 6 built 6
		{9, 6, true, 100, 50, 0},  // Auto Min on a growing planet
		{9, 1, true, 10, -1, 1},   // Auto Min at −1% habitability
		{9, 6, true, -40, 83, 6},  // Auto Min on an overcrowded planet
		{2, 6, false, 0, 50, 2},   // the count is a per-year limit
	}
	for _, c := range cases {
		if got := AutoUnits(c.count, c.capacity, c.minOnly, c.popChange, c.hab); got != c.want {
			t.Errorf("%+v: %d", c, got)
		}
	}
}
