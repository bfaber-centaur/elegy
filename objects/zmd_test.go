package objects

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Chasers of a fleet that a minefield destroys during movement go to the
// coordinates their waypoint held before movement, the target's start
// position, not where it died (stars-elegy ORDERS.md "Target destroyed
// during movement", CONFIRMED by prediction Z-MD-1, Z-MD-3, stars-elegy
// #198 de1bd6f). The near chaser reaches it and completes its waypoint;
// the far one moves straight toward it, and after the year its waypoint
// is deep space there, with one 0x028 naming the gone target (MEASURED
// Z-MD-1, Z-MD-3). With the target alive, both head for its end position
// (Z-MD-2, control, here without a stop).
func TestConfirmedChaseTargetDestroyedByMines(t *testing.T) {
	setup := func() (*lab, int, int, int) {
		l := newLab(t)
		s := at(100, 200)
		ti := l.fleet(1, s, "Freighter", 1)
		l.g.Fleets[ti].Fuel = 450
		l.g.Fleets[ti].Waypoints = []engine.Waypoint{{Pos: at(181, 200), Warp: 9}}
		near := l.fleet(0, at(100, 228), "Tank", 1)
		far := l.fleet(0, at(100, 260), "Tank", 1)
		for _, c := range []struct{ fi, warp int }{{near, 9}, {far, 5}} {
			f := &l.g.Fleets[c.fi]
			f.Fuel = 500
			f.Waypoints = []engine.Waypoint{{Pos: s, Warp: c.warp, Target: engine.TargetFleet, ID: l.g.Fleets[ti].ID}}
		}
		l.g.Objects = &Space{Minefields: []Minefield{{Owner: 0, Kind: Heavy, Pos: at(200, 200), Count: 2500}}}
		return l, ti, near, far
	}

	t.Run("destroyed", func(t *testing.T) {
		l, ti, near, far := setup()
		target, nearID, farID := l.g.Fleets[ti], l.g.Fleets[near].ID, l.g.Fleets[far].ID
		r, err := engine.GenerateTurn(withRules(*l.g), nil, zero{})
		if err != nil {
			t.Fatal(err)
		}
		g := r.Game
		byID := map[int]engine.Fleet{}
		for _, f := range g.Fleets {
			byID[f.ID] = f
		}
		if _, ok := byID[target.ID]; ok {
			t.Fatal("the target survived the field")
		}
		if n := byID[nearID]; n.Pos != target.Pos || len(n.Waypoints) != 0 {
			t.Errorf("near chaser at %v with %v, want at the start %v, waypoint completed", n.Pos, n.Waypoints, target.Pos)
		}
		f := byID[farID]
		if f.Pos != at(100, 235) {
			t.Errorf("far chaser at %v, want 25 ly toward the start, %v", f.Pos, at(100, 235))
		}
		want := []engine.Waypoint{{Pos: target.Pos, Warp: 5, Target: engine.TargetSpace}}
		if !reflect.DeepEqual(f.Waypoints, want) {
			t.Errorf("far chaser's waypoints %+v, want deep space at the start %+v", f.Waypoints, want)
		}
		var gone []engine.Event
		for _, e := range r.Events {
			if e.Kind == engine.EventTargetGone {
				gone = append(gone, e)
			}
		}
		if len(gone) != 1 || gone[0].Fleet != farID || !reflect.DeepEqual(gone[0].Axes, []int{1, target.Number}) {
			t.Errorf("0x028 events %+v, want one for the far chaser naming player 1's fleet %d", gone, target.Number)
		}
	})

	t.Run("alive", func(t *testing.T) {
		l, ti, near, far := setup()
		target := l.g.Fleets[ti]
		g := turn(t, *l.g, top{})
		byID := map[int]engine.Fleet{}
		for _, f := range g.Fleets {
			byID[f.ID] = f
		}
		tg := byID[target.ID]
		if tg.Pos != at(181, 200) {
			t.Fatalf("target at %v, want its destination", tg.Pos)
		}
		for _, c := range []int{near, far} {
			f := byID[l.g.Fleets[c].ID]
			// Unseen by the chasers' owner, the waypoint becomes space at
			// the target's end position as the files are written (0x02a).
			if len(f.Waypoints) == 0 || f.Waypoints[0].Pos != tg.Pos {
				t.Errorf("chaser %d waypoints %+v, want the target's stop point %v", c, f.Waypoints, tg.Pos)
			}
		}
	})
}
