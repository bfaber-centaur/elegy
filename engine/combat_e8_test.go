package engine

import (
	"math/rand"
	"testing"
)

func TestConfirmedE8SalvageBesideLyingSalvage(t *testing.T) {
	// E-8 (stars-elegy 96613bb, #190, PARITY.md "Salvage from several
	// battles beside lying salvage (E-8)"), CONFIRMED by prediction in 4
	// of 4 streams. Three deep-space battles in one year: player 0's
	// fleets 1, 2 and 3 (one Heavy Blaster Battleship each) kill player
	// 1's fleets 4, 1 and 2 (two Laser Destroyers each, on a plan that
	// never fires) at X, Z and Y. Unmarked salvage already lies at X
	// (player 1's number 1) and at Y (player 0's number 0); its decay is
	// package objects' (TestConfirmedE8SalvageDecayFloor there), so it
	// lies here as decayed.
	//   P2: no battle joins the lying salvage, whatever its owner.
	//   P3: the new objects take player 1's lowest unused numbers in
	//       battle order by first fleet (X, Z, Y): 0, 2 and 3.
	//   P4: each new object holds 6/4/2, marked.
	//   P5: no other object: the 2 lying and the 3 new.
	bb := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 3, Part: "Heavy Blaster", Count: 6},
		SlotFill{Slot: 4, Part: "Heavy Blaster", Count: 6})
	dd := lateDesign(t, "Destroyer",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 1},
		SlotFill{Slot: 1, Part: "Laser", Count: 1})
	x, z, y := Point{1020, 1230}, Point{1010, 1150}, Point{1000, 1400}
	lying := []Salvage{
		{Pos: y, Owner: 0, Number: 0, Minerals: Minerals{50, 0, 0}, Steps: 5},
		{Pos: x, Owner: 1, Number: 1, Minerals: Minerals{225, 40, 20}, Steps: 27},
	}
	for seed := range int64(8) {
		g := combatLabGame(dd, 1, 2)
		g.Designs = []Design{bb, dd}
		for p := range g.Players {
			for f := range g.Players[p].Research.Levels {
				g.Players[p].Research.Levels[f] = 26
			}
		}
		g.Players[1].Plans = []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Secondary: TargetAny, Attack: AttackNobody}}
		g.Fleets = []Fleet{
			{ID: 1, Owner: 0, Pos: x, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 2, Owner: 0, Pos: z, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 3, Owner: 0, Pos: y, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 1, Owner: 1, Pos: z, Stacks: []Stack{{Design: 1, Count: 2}}},
			{ID: 2, Owner: 1, Pos: y, Stacks: []Stack{{Design: 1, Count: 2}}},
			{ID: 4, Owner: 1, Pos: x, Stacks: []Stack{{Design: 1, Count: 2}}},
		}
		g.Salvage = append([]Salvage(nil), lying...)
		battles(&g, rand.New(rand.NewSource(seed)), map[int]bool{}, nil)
		if len(g.Salvage) != 5 {
			t.Errorf("seed %d: %d objects, want 5 (P5): %+v", seed, len(g.Salvage), g.Salvage)
			continue
		}
		for i, l := range lying {
			if g.Salvage[i] != l {
				t.Errorf("seed %d: lying salvage %+v became %+v (P2)", seed, l, g.Salvage[i])
			}
		}
		for i, want := range []struct {
			pos Point
			n   int
		}{{x, 0}, {z, 2}, {y, 3}} {
			s := g.Salvage[2+i]
			if s.Owner != 1 || s.Pos != want.pos || s.Number != want.n || s.Minerals != (Minerals{6, 4, 2}) || !s.Fresh {
				t.Errorf("seed %d: new salvage %d %+v, want player 1's number %d at %v, 6/4/2, marked (P3, P4)", seed, i, s, want.n, want.pos)
			}
		}
	}
}
