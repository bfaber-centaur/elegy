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
	g := &engine.Game{Salvage: []engine.Salvage{
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

// Mine-hit salvage joins the first object at the exact spot, whatever its
// owner, and marks it; elsewhere a new object of the hit fleet's owner
// takes a number from its packet pool (OBJECTS.md "Salvage", "Joining
// existing salvage", "Owner").
func TestPredictionMineSalvage(t *testing.T) {
	g := &engine.Game{Salvage: []engine.Salvage{{Pos: at(5, 5), Owner: 1, Number: 3}, {Pos: at(5, 5), Owner: 0, Number: 7, Minerals: engine.Minerals{10, 0, 0}}}}
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
	g = &engine.Game{}
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
