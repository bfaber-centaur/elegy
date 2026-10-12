package engine

import "testing"

type zeroRand struct{}

func (zeroRand) Intn(int) int { return 0 }

// TestConfirmedBreedInTransit checks KERNEL.md "Inner Strength colonists
// breed in transit" with its OT-5 vectors (growth 15%, CONFIRMED).
func TestConfirmedBreedInTransit(t *testing.T) {
	is := pgRace()
	is.PRT = PRTInnerStrength
	is.GrowthRate = 15
	home, enemy := Point{X: 100, Y: 100}, Point{X: 200, Y: 100}
	for _, c := range []struct {
		name                string
		hold, colonists     int
		pos                 Point
		wantCargo, wantHome int
		wantEnemy           int
	}{
		// A full 2100 kT fleet over its own planet breeds 157, all of it
		// landing on the planet.
		{"full over own planet", 2100, 2100, home, 2100, 1000 + 157, 500},
		// 200 kT in a 210 kT hold in deep space takes 10 and loses 5.
		{"deep space", 210, 200, Point{X: 150, Y: 150}, 210, 1000, 500},
		// A full fleet over an enemy planet gains nothing and the planet
		// gets nothing.
		{"full over enemy planet", 2100, 2100, enemy, 2100, 1000, 500},
	} {
		g := Game{Rules: ElegyRules(),
			Players: []Player{{Race: is}, {Race: pgRace()}},
			Designs: []Design{{CargoCapacity: c.hold}},
			Planets: []Planet{{ID: 1, Pos: home, Owner: 0, Population: 1000}, {ID: 2, Pos: enemy, Owner: 1, Population: 500}},
			Fleets:  []Fleet{{ID: 1, Owner: 0, Pos: c.pos, Stacks: []Stack{{Design: 0, Count: 1}}, Cargo: Cargo{Colonists: c.colonists}}},
		}
		g.breedInTransit(zeroRand{})
		if got := g.Fleets[0].Cargo.Colonists; got != c.wantCargo {
			t.Errorf("%s: fleet colonists %d, want %d", c.name, got, c.wantCargo)
		}
		if got := g.Planets[0].Population; got != c.wantHome {
			t.Errorf("%s: own planet %d, want %d", c.name, got, c.wantHome)
		}
		if got := g.Planets[1].Population; got != c.wantEnemy {
			t.Errorf("%s: enemy planet %d, want %d", c.name, got, c.wantEnemy)
		}
	}
	// RG-001 fleet (3,1): 162 kT bred to 174, 187, 201 (trunc(C·15/200)).
	g := Game{Rules: ElegyRules(), Players: []Player{{Race: is}}, Designs: []Design{{CargoCapacity: 1000}},
		Fleets: []Fleet{{ID: 1, Owner: 0, Pos: Point{X: 5, Y: 5}, Stacks: []Stack{{Design: 0, Count: 1}}, Cargo: Cargo{Colonists: 162}}}}
	for _, want := range []int{174, 187, 201} {
		g.breedInTransit(zeroRand{})
		if got := g.Fleets[0].Cargo.Colonists; got != want {
			t.Errorf("RG-001 fleet: colonists %d, want %d", got, want)
		}
	}
	// Another race's colonists do not breed.
	g.Players[0].Race = pgRace()
	if g.breedInTransit(zeroRand{}); g.Fleets[0].Cargo.Colonists != 201 {
		t.Errorf("non-IS fleet bred to %d", g.Fleets[0].Cargo.Colonists)
	}
}

// TestBreedInTransitBeforeGrowth checks the place of the step (KERNEL.md
// "Turn order" step 3b, CONFIRMED OT-5): the overflow lands on the planet
// before production and growth, and the planet grows on it that year.
func TestBreedInTransitBeforeGrowth(t *testing.T) {
	run := func(fleet bool) Game {
		g := pgHomeworld()
		g.Players[0].Race.PRT = PRTInnerStrength
		g.Players[0].Race.GrowthRate = 15
		if fleet {
			g.Designs = []Design{{CargoCapacity: 2100}}
			g.Fleets = []Fleet{{ID: 1, Owner: 0, Pos: g.Planets[0].Pos, Stacks: []Stack{{Design: 0, Count: 1}}, Cargo: Cargo{Colonists: 2100}}}
		}
		r, err := GenerateTurn(withRules(g), nil, highRand{})
		if err != nil {
			t.Fatal(err)
		}
		return r.Game
	}
	without, with := run(false), run(true)
	if got := with.Fleets[0].Cargo.Colonists; got != 2100 {
		t.Errorf("fleet colonists %d, want 2100", got)
	}
	// 157 landed, and the planet grew on them too.
	if d := with.Planets[0].Population - without.Planets[0].Population; d <= 157 {
		t.Errorf("population gained %d from the fleet, want more than the 157 that landed", d)
	}
}

// TestBreedInTransitZeroDraw checks the BINARY-ONLY draw: a breed of 0
// (here 13·15/200) becomes 1 when rand(3) draws 0, and stays 0 otherwise.
func TestBreedInTransitZeroDraw(t *testing.T) {
	is := pgRace()
	is.PRT = PRTInnerStrength
	is.GrowthRate = 15
	for _, c := range []struct {
		name string
		rng  Rand
		want int
	}{{"draw 0", zeroRand{}, 14}, {"draw 2", highRand{}, 13}} {
		g := Game{Rules: ElegyRules(),
			Players: []Player{{Race: is}},
			Designs: []Design{{CargoCapacity: 100}},
			Fleets:  []Fleet{{ID: 1, Owner: 0, Pos: Point{X: 10, Y: 10}, Stacks: []Stack{{Design: 0, Count: 1}}, Cargo: Cargo{Colonists: 13}}},
		}
		g.breedInTransit(c.rng)
		if got := g.Fleets[0].Cargo.Colonists; got != c.want {
			t.Errorf("%s: colonists %d, want %d", c.name, got, c.want)
		}
	}
}
