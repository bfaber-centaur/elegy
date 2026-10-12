package objects

import (
	"slices"
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

// A fleet that reaches a waypoint aimed at a packet with a transport task
// has the task cancelled, and its owner is told (ORDERS.md "Salvage and
// mineral packet targets", "Arrival", stars-elegy f2f9794, BINARY-ONLY;
// MESSAGES.md 0x11e). The same task aimed at salvage loads from it, and a
// fleet with no task that reaches the packet is not told.
func TestPacketTransportCancelledInTurn(t *testing.T) {
	l := newLab(t)
	// The packet's move ends over planet 8, one of player 0's, which the
	// cancelled unload does not reach.
	l.g.Planets = []engine.Planet{{ID: 7, Pos: at(900, 0), Owner: engine.NoOwner}, {ID: 8, Pos: at(100, 0), Owner: 0}}
	l.g.Objects = &Space{Packets: []Packet{{Owner: 1, Number: 2, Pos: origin, Target: 7, Warp: 10, Cargo: engine.Minerals{1000, 0, 0}}}}
	l.g.Salvage = []engine.Salvage{{Pos: at(0, 500), Owner: 1, Number: 0, Minerals: engine.Minerals{60, 0, 0}, Steps: 6, Fresh: true}}
	load := engine.Task{Kind: engine.TaskTransport}
	load.Transport[0] = engine.Transport{Action: engine.LoadAll}
	unload := engine.Task{Kind: engine.TaskTransport}
	unload.Transport[0] = engine.Transport{Action: engine.UnloadAll}
	fi := l.fleet(0, at(100, 10), "Freighter", 1)
	l.g.Fleets[fi].Fuel = 500
	l.g.Fleets[fi].Cargo.Minerals = engine.Minerals{50, 0, 0}
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 2, Task: unload}}
	fj := l.fleet(0, at(0, 510), "Freighter", 1)
	l.g.Fleets[fj].Fuel = 500
	l.g.Fleets[fj].Waypoints = []engine.Waypoint{{Pos: at(0, 500), Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 0, Task: load}}
	// A fleet with no task reaches the packet too: nothing to cancel.
	fk := l.fleet(0, at(110, 0), "Freighter", 1)
	l.g.Fleets[fk].Fuel = 500
	l.g.Fleets[fk].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 2}}
	// A fleet with another task reaches the packet: the task is kept.
	fp := l.fleet(0, at(110, 10), "Freighter", 1)
	l.g.Fleets[fp].Fuel = 500
	l.g.Fleets[fp].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 2, Task: engine.Task{Kind: engine.TaskPatrol, Range: 50}}}
	// An unload aimed at planet 9, whose id is also the number of a
	// packet of player 0's: a planet target, so the unload runs.
	l.g.Planets = append(l.g.Planets, engine.Planet{ID: 9, Pos: at(0, -300), Owner: 0})
	space(*l.g).Packets = append(space(*l.g).Packets, Packet{Owner: 0, Number: 9, Pos: at(-1000, -1000), Target: 7, Warp: 5, Cargo: engine.Minerals{100, 0, 0}})
	fq := l.fleet(0, at(0, -310), "Freighter", 1)
	l.g.Fleets[fq].Fuel = 500
	l.g.Fleets[fq].Cargo.Minerals = engine.Minerals{40, 0, 0}
	l.g.Fleets[fq].Waypoints = []engine.Waypoint{{Pos: at(0, -300), Warp: 5, Target: engine.TargetPlanet, ID: 9, Task: unload}}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
	if err != nil {
		t.Fatal(err)
	}
	p := space(r.Game).Packets[0].Pos
	if f := r.Game.Fleets[3]; f.Pos != p || f.Task.Kind != engine.TaskPatrol {
		t.Errorf("patrol fleet at %v task %+v, want at the packet %v, still patrolling", f.Pos, f.Task, p)
	}
	if f := r.Game.Fleets[4]; f.Cargo.Minerals[0] != 0 || r.Game.Planets[2].Surface[0] != 40 {
		t.Errorf("planet 9 unload: cargo %v, surface %v, want 40 kT unloaded", f.Cargo.Minerals, r.Game.Planets[2].Surface)
	}
	if r.Game.Fleets[2].Pos != p {
		t.Fatalf("taskless fleet at %v, want at the packet %v", r.Game.Fleets[2].Pos, p)
	}
	f := r.Game.Fleets[0]
	if f.Pos != p || f.Task != (engine.Task{}) || f.Cargo.Minerals != (engine.Minerals{50, 0, 0}) || len(r.Game.Salvage) != 1 || r.Game.Planets[1].Surface[0] != 0 {
		t.Errorf("packet fleet at %v task %+v cargo %v, %d salvage; want at the packet %v, no task, nothing unloaded", f.Pos, f.Task, f.Cargo.Minerals, len(r.Game.Salvage), p)
	}
	told := 0
	for _, e := range r.Events {
		if e.Kind == engine.EventTransportNoHold {
			told++
			if e.Player != 0 || e.Fleet != f.ID {
				t.Errorf("event %+v, want player 0's fleet %d", e, f.ID)
			}
		}
	}
	if told != 1 {
		t.Errorf("%d cancel messages, want 1", told)
	}
	// MESSAGES.md 0x04e (a), BINARY-ONLY: the cancelled fleet is at its
	// last waypoint, so 0x04e follows the 0x11e. The taskless fleet gets
	// its one arrival 0x04e and the patrol fleet none (0x04e (b)).
	for _, c := range []struct {
		fleet int
		want  []engine.EventKind
	}{
		{0, []engine.EventKind{engine.EventTransportNoHold, engine.EventFleetArrived}},
		{2, []engine.EventKind{engine.EventFleetArrived}},
		{3, nil},
	} {
		id := r.Game.Fleets[c.fleet].ID
		var got []engine.EventKind
		for _, e := range r.Events {
			if e.Fleet == id && (e.Kind == engine.EventTransportNoHold || e.Kind == engine.EventFleetArrived) {
				got = append(got, e.Kind)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("fleet %d: events %v, want %v", id, got, c.want)
		}
	}
	if g := r.Game.Fleets[1]; g.Pos != at(0, 500) || g.Cargo.Minerals[0] != 60 {
		t.Errorf("salvage fleet at %v cargo %v, want 60 kT loaded at the salvage", g.Pos, g.Cargo.Minerals)
	}
}

// Without space objects (Game.Objects nil, as in engine-only games) a
// salvage target with a transport task is not looked up among packets:
// there is no 0x11e. (Such a game does not load from salvage either;
// loading needs the space objects' rules.)
func TestSalvageTransportWithoutObjects(t *testing.T) {
	l := newLab(t)
	l.g.Objects = nil
	l.g.Salvage = []engine.Salvage{{Pos: at(0, 500), Owner: 1, Number: 0, Minerals: engine.Minerals{60, 0, 0}, Steps: 6, Fresh: true}}
	load := engine.Task{Kind: engine.TaskTransport}
	load.Transport[0] = engine.Transport{Action: engine.LoadAll}
	fj := l.fleet(0, at(0, 510), "Freighter", 1)
	l.g.Fleets[fj].Fuel = 500
	l.g.Fleets[fj].Waypoints = []engine.Waypoint{{Pos: at(0, 500), Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 0, Task: load}}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
	if err != nil {
		t.Fatal(err)
	}
	if g := r.Game.Fleets[0]; g.Pos != at(0, 500) {
		t.Errorf("salvage fleet at %v, want at the salvage", g.Pos)
	}
	for _, e := range r.Events {
		if e.Kind == engine.EventTransportNoHold {
			t.Errorf("cancel message %+v", e)
		}
	}
}
