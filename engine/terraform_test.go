package engine

import "testing"

// caGame is one Claim Adjuster planet of pgRace with these environments
// and levels.
func caGame(env, orig [3]int, levels [NumFields]int, tt bool) Game {
	r := pgRace()
	r.PRT = PRTClaimAdjuster
	r.LRT.TotalTerraforming = tt
	g := Game{
		Rules:   ElegyRules(),
		Players: []Player{{Race: r}},
		Planets: []Planet{{ID: 1, Owner: 0, Population: 100, Env: env, OrigEnv: orig}},
	}
	g.Players[0].Research.Levels = levels
	return g
}

// noDrift draws an axis whose original is at the centre, or a failed
// rand(10), so only the year-end step acts.
type noDrift struct{}

func (noDrift) Intn(n int) int { return n - 1 }

// TestPredictionTerraformReach: the largest available part per axis,
// Total Terraform counting for every axis (KERNEL.md "Terraforming",
// COMPONENTS.md terraform rows).
func TestPredictionTerraformReach(t *testing.T) {
	r := pgRace()
	var lv [NumFields]int
	lv[Propulsion], lv[Biotech], lv[Energy] = 10, 3, 5
	if got := terraformReach(r, lv); got != [3]int{11, 7, 0} {
		t.Errorf("reach = %v, want [11 7 0]", got)
	}
	r.LRT.TotalTerraforming = true
	lv = [NumFields]int{}
	lv[Biotech] = 25
	if got := terraformReach(r, lv); got != [3]int{30, 30, 30} {
		t.Errorf("TT reach = %v, want [30 30 30]", got)
	}
	r.Env[1].Immune = true
	if got := terraformReach(r, lv); got != [3]int{30, 0, 30} {
		t.Errorf("immune reach = %v, want [30 0 30]", got)
	}
}

// TestConfirmedClaimAdjusterYearEnd: KERNEL.md "Terraforming" → Claim
// Adjuster worked examples (KX-003 S3/S3L at reach 3; TK-108,
// TK-118..121 after a capture at reach 15 and TT 30).
func TestConfirmedClaimAdjusterYearEnd(t *testing.T) {
	var reach3, reach15, none [NumFields]int
	reach3[Energy], reach3[Weapons], reach3[Propulsion], reach3[Biotech] = 1, 1, 1, 1
	reach15[Energy], reach15[Weapons], reach15[Propulsion], reach15[Biotech] = 16, 16, 16, 4
	var tt30 [NumFields]int
	tt30[Biotech] = 25
	for _, c := range []struct {
		name            string
		env, orig, want [3]int
		levels          [NumFields]int
		tt              bool
	}{
		{"S3 reach 3", [3]int{60, 42, 56}, [3]int{60, 42, 56}, [3]int{57, 45, 53}, reach3, false},
		{"S3L terraformed", [3]int{58, 50, 50}, [3]int{60, 50, 50}, [3]int{57, 50, 50}, reach3, false},
		{"capture ±15 near centre", [3]int{55, 47, 52}, [3]int{55, 47, 52}, [3]int{50, 50, 50}, reach15, false},
		{"capture ±15", [3]int{80, 20, 80}, [3]int{80, 20, 80}, [3]int{65, 35, 65}, reach15, false},
		{"capture TT ±30", [3]int{80, 20, 80}, [3]int{80, 20, 80}, [3]int{50, 50, 50}, tt30, true},
		{"capture no tech", [3]int{80, 20, 80}, [3]int{80, 20, 80}, [3]int{80, 20, 80}, none, false},
	} {
		g := caGame(c.env, c.orig, c.levels, c.tt)
		if ev := g.claimAdjusterYearEnd(noDrift{}); len(ev) != 0 {
			t.Errorf("%s: events %v, want none", c.name, ev)
		}
		if got := g.Planets[0].Env; got != c.want {
			t.Errorf("%s: env = %v, want %v", c.name, got, c.want)
		}
		if got := g.Planets[0].OrigEnv; got != c.orig {
			t.Errorf("%s: original = %v, want unchanged %v", c.name, got, c.orig)
		}
	}
}

// TestConfirmedClaimAdjusterDrift: rand(3) picks the axis, rand(10) must
// be 0, then the drift needs 1000 units or rand(1000) < population
// (KERNEL.md "Terraforming" → Claim Adjuster, KX-005). The year-end step
// then measures the reach from the new original.
func TestConfirmedClaimAdjusterDrift(t *testing.T) {
	var none [NumFields]int
	g := caGame([3]int{50, 70, 50}, [3]int{50, 70, 50}, none, false)
	g.Planets[0].Population = 100
	rng := &seqRand{draws: []int{1, 0, 99}}
	ev := g.claimAdjusterYearEnd(rng)
	if len(ev) != 1 || ev[0].Kind != EventOriginalDrift || ev[0].Axes[0] != 1 {
		t.Fatalf("events = %v, want one drift on temperature", ev)
	}
	if got := g.Planets[0].OrigEnv; got != [3]int{50, 69, 50} {
		t.Errorf("original = %v, want [50 69 50]", got)
	}
	if got := g.Planets[0].Env; got != [3]int{50, 69, 50} {
		t.Errorf("env = %v, want [50 69 50] (reach 0 from the new original)", got)
	}
	if len(rng.draws) != 0 {
		t.Errorf("%d draws left, want all three used", len(rng.draws))
	}

	// rand(1000) not below the population: no drift.
	g = caGame([3]int{50, 70, 50}, [3]int{50, 70, 50}, none, false)
	rng = &seqRand{draws: []int{1, 0, 100}}
	if ev := g.claimAdjusterYearEnd(rng); len(ev) != 0 {
		t.Errorf("events = %v, want none", ev)
	}
	// 1000 units or more: no third draw.
	g = caGame([3]int{50, 70, 50}, [3]int{50, 70, 50}, none, false)
	g.Planets[0].Population = 1000
	rng = &seqRand{draws: []int{1, 0, 999, 7}}
	if ev := g.claimAdjusterYearEnd(rng); len(ev) != 1 || len(rng.draws) != 2 {
		t.Errorf("events = %v, draws left %d; want one drift and two left", ev, len(rng.draws))
	}
	// An axis at the centre draws only rand(3).
	g = caGame([3]int{50, 70, 50}, [3]int{50, 70, 50}, none, false)
	rng = &seqRand{draws: []int{0, 5}}
	if ev := g.claimAdjusterYearEnd(rng); len(ev) != 0 || len(rng.draws) != 1 {
		t.Errorf("events = %v, draws left %d; want none and one left", ev, len(rng.draws))
	}
}
