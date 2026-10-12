package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// A waypoint aimed at a packet reads the packet's position after the
// packets move, before fleets move (ORDERS.md "Ordering with moving
// objects"; ORDERS.md "Salvage and mineral packet targets", stars-elegy
// f2f9794, BINARY-ONLY), and still names it after the year, whether or
// not the owner sees it (the same, BINARY-ONLY; MEASURED in RG-002).
// Another owner's object of the same number is not it: that waypoint
// becomes a plain go-to.
func TestPacketObjectWaypointInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Planets = []engine.Planet{{ID: 7, Pos: at(900, 0), Owner: engine.NoOwner}}
	l.g.Objects = &Space{Packets: []Packet{{Owner: 1, Number: 2, Pos: origin, Target: 7, Warp: 10, Cargo: engine.Minerals{1000, 0, 0}}}}
	fi := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fi].Fuel = 500
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 9, Target: engine.TargetPacketObject, Owner: 1, ID: 2}}
	fj := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fj].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 0, Target: engine.TargetPacketObject, Owner: 0, ID: 2}}
	g := turn(t, *l.g, top{})
	p := space(g).Packets[0].Pos
	if p == origin {
		t.Fatal("the packet did not move")
	}
	if wp := g.Fleets[0].Waypoints[0]; wp != (engine.Waypoint{Pos: p, Warp: 9, Target: engine.TargetPacketObject, Owner: 1, ID: 2}) {
		t.Errorf("waypoint %+v, want on the packet at %v", wp, p)
	}
	if wp := g.Fleets[1].Waypoints[0]; wp != (engine.Waypoint{Pos: origin}) {
		t.Errorf("other owner's number: %+v, want plain %v", wp, origin)
	}
}

// A waypoint on a packet that arrives this year becomes a plain go-to at
// the packet's last coordinates, and its owner is not told (ORDERS.md
// "Salvage and mineral packet targets", stars-elegy f2f9794, BINARY-ONLY).
func TestPacketObjectGoneInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Planets = []engine.Planet{{ID: 7, Pos: at(50, 0), Owner: engine.NoOwner}}
	l.g.Objects = &Space{Packets: []Packet{{Owner: 1, Pos: origin, Target: 7, Warp: 10, Cargo: engine.Minerals{1000, 0, 0}}}}
	fi := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 0, Target: engine.TargetPacketObject, Owner: 1, ID: 0}}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(space(r.Game).Packets); n != 0 {
		t.Fatalf("%d packets, want it arrived", n)
	}
	if wp := r.Game.Fleets[0].Waypoints[0]; wp != (engine.Waypoint{Pos: origin}) {
		t.Errorf("waypoint %+v, want plain %v", wp, origin)
	}
	for _, e := range r.Events {
		if e.Player == 0 && e.Fleet == r.Game.Fleets[0].ID {
			t.Errorf("event %+v, want no message", e)
		}
	}
}

// A waypoint on salvage reads the object's position and still names it
// after the year (ORDERS.md "Salvage and mineral packet targets",
// BINARY-ONLY; MEASURED in RG-002).
func TestSalvageWaypointInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Salvage = []engine.Salvage{{Pos: at(100, 0), Owner: 1, Number: 4, Minerals: engine.Minerals{500, 0, 0}, Steps: 50, Fresh: true}}
	l.g.Objects = &Space{}
	fi := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 0, Target: engine.TargetPacketObject, Owner: 1, ID: 4}}
	g := turn(t, *l.g, top{})
	if wp := g.Fleets[0].Waypoints[0]; wp != (engine.Waypoint{Pos: at(100, 0), Target: engine.TargetPacketObject, Owner: 1, ID: 4}) {
		t.Errorf("waypoint %+v, want on the salvage", wp)
	}
}
