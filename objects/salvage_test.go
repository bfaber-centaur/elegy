package objects

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Decay (OBJECTS.md "Salvage", "Decay"): a fresh object only loses its
// mark; otherwise max(10, ⌊m/10⌋) per non-empty mineral, not below 0, and
// an empty object goes.
func TestPredictionSalvageDecay(t *testing.T) {
	g := &engine.Game{Rules: engine.ElegyRules(), Salvage: []engine.Salvage{
		{Minerals: engine.Minerals{60, 20, 70}, Fresh: true},
		{Minerals: engine.Minerals{250, 5, 0}},
		{Minerals: engine.Minerals{8, 0, 0}},
	}}
	DecaySalvage(g)
	if len(g.Salvage) != 2 || g.Salvage[0].Minerals != (engine.Minerals{60, 20, 70}) || g.Salvage[0].Fresh ||
		g.Salvage[1].Minerals != (engine.Minerals{225, 0, 0}) || g.Salvage[1].Steps != 23 {
		t.Errorf("after one decay: %+v", g.Salvage)
	}
	DecaySalvage(g)
	if g.Salvage[0].Minerals != (engine.Minerals{50, 10, 60}) {
		t.Errorf("after two: %+v", g.Salvage[0])
	}
}

// E-8 (stars-elegy 96613bb, #190): unmarked salvage loses max(10, ⌊m/10⌋)
// per non-empty mineral, CONFIRMED by prediction in 4 of 4 streams:
// 250/50/30 became 225/40/20 and 60/0/0 became 50/0/0 (plain ⌊m/10⌋ would
// give 225/45/27 and 54/0/0). The battle half is engine's
// TestConfirmedE8SalvageBesideLyingSalvage.
func TestConfirmedE8SalvageDecayFloor(t *testing.T) {
	g := &engine.Game{Rules: engine.ElegyRules(), Salvage: []engine.Salvage{
		{Owner: 0, Number: 0, Minerals: engine.Minerals{60, 0, 0}},
		{Owner: 1, Number: 1, Minerals: engine.Minerals{250, 50, 30}},
	}}
	DecaySalvage(g)
	if len(g.Salvage) != 2 || g.Salvage[0].Minerals != (engine.Minerals{50, 0, 0}) || g.Salvage[0].Fresh ||
		g.Salvage[1].Minerals != (engine.Minerals{225, 40, 20}) || g.Salvage[1].Fresh {
		t.Errorf("after decay: %+v, want 50/0/0 and 225/40/20, unmarked", g.Salvage)
	}
}

// E-8 P3 through the production numbering (Space.SalvageNumber, the
// packet pool): with player 1's number 1 in use, three new salvage
// objects take 0, 2 and 3 (CONFIRMED by prediction, stars-elegy 96613bb;
// the battle half is engine's TestConfirmedE8SalvageBesideLyingSalvage).
func TestConfirmedE8SalvageNumbersWithAGap(t *testing.T) {
	g := &engine.Game{Rules: engine.ElegyRules(), Salvage: []engine.Salvage{
		{Owner: 0, Number: 0, Minerals: engine.Minerals{50, 0, 0}},
		{Owner: 1, Number: 1, Minerals: engine.Minerals{225, 40, 20}},
	}}
	s := &Space{}
	var got []int
	for range 3 {
		n, ok := s.SalvageNumber(g, 1)
		if !ok {
			t.Fatalf("no room for salvage %d", len(got))
		}
		got = append(got, n)
		g.Salvage = append(g.Salvage, engine.Salvage{Owner: 1, Number: n, Minerals: engine.Minerals{6, 4, 2}})
	}
	if !reflect.DeepEqual(got, []int{0, 2, 3}) {
		t.Errorf("numbers %v, want [0 2 3]", got)
	}
}

// Mine-hit salvage joins the first object at the exact spot, whatever its
// owner, and marks it; elsewhere a new object of the hit fleet's owner
// takes a number from its packet pool (OBJECTS.md "Salvage", "Joining
// existing salvage", "Owner").
func TestPredictionMineSalvage(t *testing.T) {
	g := &engine.Game{Rules: engine.ElegyRules(), Salvage: []engine.Salvage{{Pos: at(5, 5), Owner: 1, Number: 3}, {Pos: at(5, 5), Owner: 0, Number: 7, Minerals: engine.Minerals{10, 0, 0}}}}
	s := &Space{Packets: []Packet{{Owner: 2, Number: 0}}}
	s.AddMineSalvage(g, 2, at(5, 5), engine.Minerals{5, 5, 5})
	if j := g.Salvage[1]; j.Owner != 0 || j.Minerals != (engine.Minerals{15, 5, 5}) || !j.Fresh || j.Steps != 4 || len(g.Salvage) != 2 {
		t.Errorf("join: %+v", g.Salvage)
	}
	s.AddMineSalvage(g, 2, at(9, 9), engine.Minerals{1, 0, 0})
	if n := g.Salvage[2]; n.Owner != 2 || n.Number != 1 || !n.Fresh || n.Pos != at(9, 9) {
		t.Errorf("new: %+v", n)
	}
	// The 30,000 kT limit: overflow goes to a new object of the adder at
	// the same spot, not fresh (COMBAT.md "Salvage", CB-040).
	g = &engine.Game{Rules: engine.ElegyRules()}
	s = &Space{}
	s.NewSalvage(g, 0, origin, engine.Minerals{36098, 0, 50})
	if len(g.Salvage) != 2 || g.Salvage[0].Minerals != (engine.Minerals{30000, 0, 0}) || g.Salvage[1].Minerals != (engine.Minerals{6098, 0, 50}) ||
		!g.Salvage[0].Fresh || g.Salvage[1].Fresh || g.Salvage[1].Number != 1 {
		t.Errorf("overflow: %+v", g.Salvage)
	}
}

// Loading and unloading limits, and sight (OBJECTS.md "Salvage",
// "Loading", "Visibility").
func TestPredictionSalvageLoadAndSight(t *testing.T) {
	sv := engine.Salvage{Minerals: engine.Minerals{47, 0, 3}, Steps: 6}
	if SalvageLoad(sv, engine.Ironium, 100) != 47 || SalvageLoad(sv, 3, 5) != 0 || SalvageRoom(sv) != 10 {
		t.Errorf("load %d, room %d", SalvageLoad(sv, engine.Ironium, 100), SalvageRoom(sv))
	}
	l := newLab(t)
	l.g.Salvage = []engine.Salvage{{Pos: at(48, 0), Owner: 1}, {Pos: at(53, 0), Owner: 0}}
	sp := &Space{}
	if got := sp.Scan(l.g, 0, []Scanner{rhino(origin)}); !reflect.DeepEqual(got.Salvage, []int{0}) || !reflect.DeepEqual(got.Owners, []int{1}) {
		t.Errorf("sight: %+v", got)
	}
	l.g.Players[0].Race.PRT = engine.PRTPacketPhysics
	if got := sp.Scan(l.g, 0, nil); !reflect.DeepEqual(got.Salvage, []int{0, 1}) {
		t.Errorf("PP sight: %+v", got)
	}
}
