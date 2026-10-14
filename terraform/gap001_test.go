package terraform

import (
	"slices"
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
	l.g.Planets[pi].ID = 15 // GAP-001's planet 15, not 0, so a message names it
	// Fleet 1/4 (one adjuster) is listed before fleet 1/1 (two), so the
	// fleet-number order is Elegy's, not the slice's.
	l.g.Fleets[l.fleet(1, pi, "OA1", 1)].Number = 4
	l.g.Fleets[l.fleet(1, pi, "Adjuster", 1)].Number = 1
	return l, pi
}

// GAP-001 W1 and W2 (KERNEL.md "Terraforming", Orbital Adjusters): three
// hostile clicks on an unprotected planet, none at a planet with a
// starbase. Predicted at stars-elegy 8addf9e (0aa44f5); the run matched
// except W1's fleet 1/4 message, fixed to the run (stars-elegy 0d63a84).
func TestPredictionGAP001Worsen(t *testing.T) {
	if r := Reach(engine.Race{PRT: engine.PRTClaimAdjuster}, levels(3, 3, 10, 3, 3, 6)); r != [3]int{11, 3, 3} {
		t.Fatalf("reach %v, want 11/3/3", r)
	}
	// W1: every click goes to gravity (toward 71, the farther end of
	// 60 ± 11 from the centre 50); the original environment is unchanged.
	// Messages go per fleet. Fleet 1/1's two clicks take player 0's
	// habitability from 72% to 70%: "degraded" to both players. Fleet
	// 1/4's click (62 → 63) leaves it at 70%: "cannot degrade any
	// further" to player 1 only (CONFIRMED GAP-001 W1; the prediction of
	// no message for fleet 1/4 was REFUTED).
	l, pi := gap001W(t, false)
	ev := Rules{}.Adjust(l.g)
	p := l.g.Planets[pi]
	if p.Env != [3]int{63, 60, 60} || p.OrigEnv != [3]int{60, 60, 60} {
		t.Errorf("W1: env %v orig %v, want 63/60/60 and 60/60/60", p.Env, p.OrigEnv)
	}
	if h := [3]int{engine.Habitability(race(), [3]int{60, 60, 60}), engine.Habitability(race(), [3]int{62, 60, 60}), engine.Habitability(race(), [3]int{63, 60, 60})}; h != [3]int{72, 70, 70} {
		t.Fatalf("habitability %v, want 72/70/70", h)
	}
	type msg struct{ kind, player, planet, fleetNumber, hab, before int }
	var got []msg
	for _, e := range ev {
		n := -1
		for _, f := range l.g.Fleets {
			if f.ID == e.Fleet {
				n = f.Number
			}
		}
		before := -1
		if len(e.Axes) == 1 {
			before = e.Axes[0]
		}
		got = append(got, msg{int(e.Kind), e.Player, e.Planet, n, e.Count, before})
	}
	deg := int(engine.EventPlanetDegraded)
	not := int(engine.EventPlanetNotDegraded)
	id := l.g.Planets[pi].ID
	// GAP-001 W1's 0x15a slots read 72 then 70 (MESSAGES.md 0x15a,
	// "percent, percent"); 0x15b has one percent slot.
	want := []msg{{deg, 1, id, 1, 70, 72}, {deg, 0, id, 1, 70, 72}, {not, 1, id, 4, 70, -1}}
	if len(got) != len(want) {
		t.Fatalf("W1: messages %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("W1: message %d %+v, want %+v", i, got[i], want[i])
		}
	}
	// Not GAP-001: the same fleet 1/4 as a friend, at 63/60/60, improves
	// gravity to 62 and leaves 70%: "cannot improve beyond 70%" to the
	// fleet owner alone (MESSAGES.md 0x12d, BINARY-ONLY).
	l, pi = gap001W(t, false)
	l.g.Players[1].Relations[0] = engine.RelationFriend
	l.g.Planets[pi].Env = [3]int{63, 60, 60}
	l.g.Fleets = l.g.Fleets[:1]
	ev = Rules{}.Adjust(l.g)
	if l.g.Planets[pi].Env != [3]int{62, 60, 60} || len(ev) != 1 ||
		ev[0].Kind != engine.EventPlanetNotImproved || ev[0].Player != 1 || ev[0].Fleet != l.g.Fleets[0].ID || ev[0].Count != 70 || ev[0].Axes != nil {
		t.Errorf("friendly: env %v, messages %+v", l.g.Planets[pi].Env, ev)
	}
	// Not GAP-001: hostile fleets that can make no click (every axis
	// already at the edge of their reach, 71/63/63 from 60/60/60) change
	// nothing; each still gets "cannot degrade any further" at the
	// unchanged 51%, in fleet order (MESSAGES.md "Remote terraforming":
	// every turn, even when nothing changes; BINARY-ONLY).
	l, pi = gap001W(t, false)
	l.g.Planets[pi].Env = [3]int{71, 63, 63}
	ev = Rules{}.Adjust(l.g)
	if h := engine.Habitability(race(), [3]int{71, 63, 63}); h != 51 {
		t.Fatalf("habitability %d, want 51", h)
	}
	if l.g.Planets[pi].Env != [3]int{71, 63, 63} || len(ev) != 2 ||
		ev[0].Kind != engine.EventPlanetNotDegraded || ev[0].Fleet != l.g.Fleets[1].ID || ev[0].Count != 51 ||
		ev[1].Kind != engine.EventPlanetNotDegraded || ev[1].Fleet != l.g.Fleets[0].ID || ev[1].Count != 51 {
		t.Errorf("no click: env %v, messages %+v", l.g.Planets[pi].Env, ev)
	}
	// W2: a starbase blocks every hostile click; no message.
	l, pi = gap001W(t, true)
	if ev := (Rules{}).Adjust(l.g); len(ev) != 0 || l.g.Planets[pi].Env != [3]int{60, 60, 60} {
		t.Errorf("W2: env %v, messages %+v", l.g.Planets[pi].Env, ev)
	}
}

// Not GAP-001: two hostile fleets that each change player 0's
// habitability report their own before and after values (ASSUMPTION T3:
// fleets act in turn, each on the state the previous left). Fleet 1/1's
// two clicks take gravity 60 → 62 (72% → 70%), then fleet 1/2's two
// clicks 62 → 64.
func TestAdjustersReportEachFleetsChange(t *testing.T) {
	l, pi := gap001W(t, false)
	l.g.Fleets[0].Number = 2
	l.g.Fleets[0].Stacks[0].Design = l.designs["Adjuster"]
	h62, h64 := engine.Habitability(race(), [3]int{62, 60, 60}), engine.Habitability(race(), [3]int{64, 60, 60})
	if h62 != 70 || h64 == h62 {
		t.Fatalf("habitability %d, %d", h62, h64)
	}
	ev := Rules{}.Adjust(l.g)
	if p := l.g.Planets[pi]; p.Env != [3]int{64, 60, 60} {
		t.Fatalf("env %v", p.Env)
	}
	type msg struct{ player, fleetNumber, before, after int }
	var got []msg
	for _, e := range ev {
		n := -1
		for _, f := range l.g.Fleets {
			if f.ID == e.Fleet {
				n = f.Number
			}
		}
		before := -1
		if len(e.Axes) == 1 {
			before = e.Axes[0]
		}
		if e.Kind != engine.EventPlanetDegraded {
			t.Errorf("kind %v", e.Kind)
		}
		got = append(got, msg{e.Player, n, before, e.Count})
	}
	want := []msg{{1, 1, 72, 70}, {0, 1, 72, 70}, {1, 2, 70, h64}, {0, 2, 70, h64}}
	if !slices.Equal(got, want) {
		t.Errorf("messages %+v, want %+v", got, want)
	}
}

// Not GAP-001 (MESSAGES.md 0x12c and 0x12d, BINARY-ONLY): a friend's fleet
// that improves player 0's habitability tells both players, with the
// value before and after; one that changes nothing, even with no click to
// make, tells its owner alone.
func TestAdjustersFriendlyMessages(t *testing.T) {
	l, pi := gap001W(t, false)
	l.g.Players[1].Relations[0] = engine.RelationFriend
	l.g.Planets[pi].Env = [3]int{66, 60, 60}
	l.g.Fleets = l.g.Fleets[1:] // fleet 1/1, two clicks: gravity 66 → 64
	h66, h64 := engine.Habitability(race(), [3]int{66, 60, 60}), engine.Habitability(race(), [3]int{64, 60, 60})
	if h66 == h64 {
		t.Fatalf("habitability %d, %d", h66, h64)
	}
	ev := Rules{}.Adjust(l.g)
	type msg struct {
		kind                  engine.EventKind
		player, before, after int
	}
	var got []msg
	for _, e := range ev {
		before := -1
		if len(e.Axes) == 1 {
			before = e.Axes[0]
		}
		if e.Fleet != l.g.Fleets[0].ID || e.Planet != l.g.Planets[pi].ID {
			t.Errorf("event %+v", e)
		}
		got = append(got, msg{e.Kind, e.Player, before, e.Count})
	}
	imp := engine.EventPlanetImproved
	if want := []msg{{imp, 1, h66, h64}, {imp, 0, h66, h64}}; l.g.Planets[pi].Env != [3]int{64, 60, 60} || !slices.Equal(got, want) {
		t.Errorf("improved: env %v, messages %+v, want %+v", l.g.Planets[pi].Env, got, want)
	}
	// At 50/57/57 every axis is as near the centre 50 as the reach (±11,
	// ±3, ±3 from 60) allows: no click, and "cannot improve beyond" the
	// unchanged value to the fleet owner alone.
	l, pi = gap001W(t, false)
	l.g.Players[1].Relations[0] = engine.RelationFriend
	l.g.Planets[pi].Env = [3]int{50, 57, 57}
	l.g.Fleets = l.g.Fleets[1:]
	best := engine.Habitability(race(), [3]int{50, 57, 57})
	ev = Rules{}.Adjust(l.g)
	if l.g.Planets[pi].Env != [3]int{50, 57, 57} || len(ev) != 1 || ev[0].Kind != engine.EventPlanetNotImproved || ev[0].Player != 1 || ev[0].Count != best || ev[0].Axes != nil {
		t.Errorf("nothing to improve: env %v, messages %+v", l.g.Planets[pi].Env, ev)
	}
}
