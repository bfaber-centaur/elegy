package terraform

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// gap001W is GAP-001 sub-case W: player 1, a Claim Adjuster neutral to
// player 0, at propulsion 10 and biotechnology 6 with its other fields at
// 3 (ASSUMPTION: the base's starting levels, as in the OB vectors built
// from the same base), so its reach is gravity ±11, temperature ±3 and
// radiation ±3. Player 0's habitat is 50 (15–85) on every axis.
func gap001W(t *testing.T, starbase bool) (*lab, int) {
	t.Helper()
	l := newLab(t)
	cat := engine.Components()
	d, err := cat.NewDesign("OA1", "Midget Miner", []engine.SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 1, Part: "Orbital Adjuster", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	l.designs["OA1"] = len(l.g.Designs)
	l.g.Designs = append(l.g.Designs, d)
	l.g.Players[1].Race.PRT = engine.PRTClaimAdjuster
	l.g.Players[1].Research.Levels = levels(3, 3, 10, 3, 3, 6)
	l.g.Players[1].Relations[0] = engine.RelationNeutral
	pi := l.planet(0, [3]int{60, 60, 60}, [3]int{})
	l.g.Planets[pi].HasStarbase = starbase
	// Fleet 1/4 (one adjuster) is listed before fleet 1/1 (two), so the
	// fleet-number order is Elegy's, not the slice's.
	l.g.Fleets[l.fleet(1, pi, "OA1", 1)].Number = 4
	l.g.Fleets[l.fleet(1, pi, "Adjuster", 1)].Number = 1
	return l, pi
}

// GAP-001 W1 and W2 (KERNEL.md "Terraforming", Orbital Adjusters, stars-elegy
// 8addf9e): three hostile clicks on an unprotected planet, none at a planet
// with a starbase.
func TestPredictionGAP001Worsen(t *testing.T) {
	if r := Reach(engine.Race{PRT: engine.PRTClaimAdjuster}, levels(3, 3, 10, 3, 3, 6)); r != [3]int{11, 3, 3} {
		t.Fatalf("reach %v, want 11/3/3", r)
	}
	// W1: every click goes to gravity (toward 71, the farther end of
	// 60 ± 11 from the centre 50); the original environment is unchanged.
	// Messages: a pair (fleet owner, planet owner) per fleet whose clicks
	// changed player 0's habitability. Fleet 1/1's two clicks take it from
	// 72% to 70%; fleet 1/4's click (62 → 63) leaves it at 70%, so Elegy
	// sends that fleet nothing (ASSUMPTION: KERNEL.md gives the fleet
	// owner a message per planet changed and does not say per fleet).
	l, pi := gap001W(t, false)
	ev := Rules{}.Adjust(l.g)
	p := l.g.Planets[pi]
	if p.Env != [3]int{63, 60, 60} || p.OrigEnv != [3]int{60, 60, 60} {
		t.Errorf("W1: env %v orig %v, want 63/60/60 and 60/60/60", p.Env, p.OrigEnv)
	}
	if h := [3]int{engine.Habitability(race(), [3]int{60, 60, 60}), engine.Habitability(race(), [3]int{62, 60, 60}), engine.Habitability(race(), [3]int{63, 60, 60})}; h != [3]int{72, 70, 70} {
		t.Fatalf("habitability %v, want 72/70/70", h)
	}
	type msg struct{ kind, player, fleetNumber, hab int }
	var got []msg
	for _, e := range ev {
		n := -1
		for _, f := range l.g.Fleets {
			if f.ID == e.Fleet {
				n = f.Number
			}
		}
		got = append(got, msg{int(e.Kind), e.Player, n, e.Count})
	}
	deg := int(engine.EventPlanetDegraded)
	want := []msg{{deg, 1, 1, 70}, {deg, 0, 1, 70}}
	if len(got) != len(want) {
		t.Fatalf("W1: messages %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("W1: message %d %+v, want %+v", i, got[i], want[i])
		}
	}
	// W2: a starbase blocks every hostile click; no message.
	l, pi = gap001W(t, true)
	if ev := (Rules{}).Adjust(l.g); len(ev) != 0 || l.g.Planets[pi].Env != [3]int{60, 60, 60} {
		t.Errorf("W2: env %v, messages %+v", l.g.Planets[pi].Env, ev)
	}
}
