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
// loads from the packet, whoever owns it, minerals only, then is told it
// has finished its orders at its last waypoint; there is no 0x11e and the
// emptied packet flies on (ORDERS.md "Salvage and mineral packet
// targets", "Arrival", stars-elegy a1c4c91; PARITY.md "A transport task
// aimed at a packet (ET-2)": MEASURED for an own packet, CONFIRMED by
// prediction for another player's, ET-2c). The same task aimed at salvage
// loads from it, and a fleet with no task or another task that reaches
// the packet is not told anything more.
func TestPacketTransportLoadsInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Planets = []engine.Planet{{ID: 7, Pos: at(900, 0), Owner: engine.NoOwner}}
	l.g.Objects = &Space{Packets: []Packet{
		{Owner: 1, Number: 2, Pos: origin, Target: 7, Warp: 10, Cargo: engine.Minerals{100, 0, 0}},
		{Owner: 0, Number: 3, Pos: at(0, -600), Target: 7, Warp: 10, Cargo: engine.Minerals{0, 100, 50}},
	}}
	l.g.Salvage = []engine.Salvage{{Pos: at(0, 500), Owner: 1, Number: 0, Minerals: engine.Minerals{60, 0, 0}, Steps: 6, Fresh: true}}
	load := engine.Task{Kind: engine.TaskTransport}
	load.Transport[0] = engine.Transport{Action: engine.LoadAll}
	loadBor := engine.Task{Kind: engine.TaskTransport}
	loadBor.Transport[1] = engine.Transport{Action: engine.LoadAll}
	loadBor.Transport[engine.CargoColonists] = engine.Transport{Action: engine.SetWaypoint}
	unload := engine.Task{Kind: engine.TaskTransport}
	unload.Transport[0] = engine.Transport{Action: engine.UnloadAll}
	// Player 1's packet (ET-2c).
	fi := l.fleet(0, at(100, 10), "Freighter", 1)
	l.g.Fleets[fi].Fuel = 500
	l.g.Fleets[fi].Cargo.Minerals = engine.Minerals{50, 0, 0}
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 2, Task: load}}
	fj := l.fleet(0, at(0, 510), "Freighter", 1)
	l.g.Fleets[fj].Fuel = 500
	l.g.Fleets[fj].Waypoints = []engine.Waypoint{{Pos: at(0, 500), Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 0, Task: load}}
	// A fleet with no task reaches the packet too.
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
	// The owner's own packet (ET-2): boranium loads; colonists cannot, so
	// "set waypoint to 0" colonists is met (ASSUMPTION T6), though the
	// packet still holds germanium.
	fo := l.fleet(0, at(90, -540), "Freighter", 1)
	l.g.Fleets[fo].Fuel = 500
	l.g.Fleets[fo].Waypoints = []engine.Waypoint{{Pos: at(0, -600), Warp: 5, Target: engine.TargetPacketObject, Owner: 0, ID: 3, Task: loadBor}}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
	if err != nil {
		t.Fatal(err)
	}
	pk := space(r.Game).Packets
	if len(pk) != 3 || pk[0].Cargo != (engine.Minerals{}) || pk[1].Cargo != (engine.Minerals{0, 0, 50}) || pk[0].Pos == origin {
		t.Fatalf("packets %+v, want both loaded from emptied and still flying", pk)
	}
	p := pk[0].Pos
	if f := r.Game.Fleets[3]; f.Pos != p || f.Task.Kind != engine.TaskPatrol {
		t.Errorf("patrol fleet at %v task %+v, want at the packet %v, still patrolling", f.Pos, f.Task, p)
	}
	if f := r.Game.Fleets[4]; f.Cargo.Minerals[0] != 0 || r.Game.Planets[1].Surface[0] != 40 {
		t.Errorf("planet 9 unload: cargo %v, surface %v, want 40 kT unloaded", f.Cargo.Minerals, r.Game.Planets[1].Surface)
	}
	if r.Game.Fleets[2].Pos != p || r.Game.Fleets[2].Cargo != (engine.Cargo{}) {
		t.Fatalf("taskless fleet at %v cargo %+v, want at the packet %v, empty", r.Game.Fleets[2].Pos, r.Game.Fleets[2].Cargo, p)
	}
	if f := r.Game.Fleets[0]; f.Pos != p || f.Task != (engine.Task{}) || f.Cargo.Minerals != (engine.Minerals{150, 0, 0}) {
		t.Errorf("player 1's packet: fleet at %v task %+v cargo %v; want at the packet %v, no task, 100 kT loaded", f.Pos, f.Task, f.Cargo.Minerals, p)
	}
	if f := r.Game.Fleets[5]; f.Pos != pk[1].Pos || f.Task != (engine.Task{}) || f.Cargo != (engine.Cargo{Minerals: engine.Minerals{0, 100, 0}}) {
		t.Errorf("own packet: fleet at %v task %+v cargo %+v; want at the packet %v, no task, 100 kT boranium", f.Pos, f.Task, f.Cargo, pk[1].Pos)
	}
	// No 0x11e; 0x04e (a) after the load at the last waypoint; the
	// taskless fleet its arrival 0x04e and the patrol fleet none (0x04e
	// (b)).
	for _, c := range []struct {
		fleet int
		want  []engine.EventKind
	}{
		{0, []engine.EventKind{engine.EventFleetArrived}},
		{2, []engine.EventKind{engine.EventFleetArrived}},
		{3, nil},
		{5, []engine.EventKind{engine.EventFleetArrived}},
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
// salvage target with a transport task does not panic and sends no 0x11e.
// (Such a game does not load from salvage or packets; loading needs the
// space objects' rules.)
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

// A fleet that reaches a waypoint aimed at a packet or salvage loads from
// that object only, found by owner and number (ORDERS.md "Salvage and
// mineral packet targets", "Arrival", BINARY-ONLY): not from salvage or
// another packet at the same spot, nor from a planet there (ASSUMPTION
// T5). A waypoint at a plain position on a passing packet does not load
// from it (ASSUMPTION T5). Packet 1/2 flies from the origin to (100, 0)
// before fleets move; the fleet starts at (100, 10).
func TestPacketTransportNamedSource(t *testing.T) {
	task := func(c int, a engine.TransportAction, v int) engine.Task {
		k := engine.Task{Kind: engine.TaskTransport}
		k.Transport[c] = engine.Transport{Action: a, Amount: v}
		return k
	}
	named := func(owner, id int, k engine.Task) engine.Waypoint {
		return engine.Waypoint{Pos: origin, Warp: 5, Target: engine.TargetPacketObject, Owner: owner, ID: id, Task: k}
	}
	salvage := func(n, ironium int) engine.Salvage {
		return engine.Salvage{Pos: at(100, 0), Owner: 1, Number: n, Minerals: engine.Minerals{ironium, 0, 0}, Steps: 6, Fresh: true}
	}
	packet := func(n int, c engine.Minerals) Packet {
		return Packet{Owner: 1, Number: n, Pos: origin, Target: 7, Warp: 10, Cargo: c}
	}
	loadIronium := task(0, engine.LoadAll, 0)
	for _, c := range []struct {
		name     string
		packets  []Packet
		salvage  []engine.Salvage
		planet   bool // player 0's planet 8 at (100, 0) with 500 kT ironium
		wp       engine.Waypoint
		cargo    engine.Minerals
		left     []engine.Minerals // the packets' cargo after
		salvLeft []int             // the salvage's ironium after
		action   engine.TransportAction
	}{
		{"named packet over salvage", []Packet{packet(2, engine.Minerals{100, 0, 0})}, []engine.Salvage{salvage(0, 60)}, false,
			named(1, 2, loadIronium), engine.Minerals{100, 0, 0}, []engine.Minerals{{}}, []int{60}, engine.TransportNone},
		{"named salvage, the second there", []Packet{packet(2, engine.Minerals{100, 0, 0})}, []engine.Salvage{salvage(0, 60), salvage(1, 40)}, false,
			named(1, 1, loadIronium), engine.Minerals{40, 0, 0}, []engine.Minerals{{100, 0, 0}}, []int{60, 0}, engine.TransportNone},
		{"named packet, the second there", []Packet{packet(2, engine.Minerals{100, 0, 0}), packet(3, engine.Minerals{0, 50, 0})}, nil, false,
			named(1, 3, task(1, engine.LoadAll, 0)), engine.Minerals{0, 50, 0}, []engine.Minerals{{100, 0, 0}, {}}, nil, engine.TransportNone},
		{"plain position on the packet", []Packet{packet(2, engine.Minerals{100, 0, 0})}, nil, false,
			engine.Waypoint{Pos: at(100, 0), Warp: 5, Task: loadIronium}, engine.Minerals{}, []engine.Minerals{{100, 0, 0}}, nil, engine.TransportNone},
		{"named packet at the owner's planet", []Packet{packet(2, engine.Minerals{100, 0, 0})}, nil, true,
			named(1, 2, loadIronium), engine.Minerals{100, 0, 0}, []engine.Minerals{{}}, nil, engine.TransportNone},
		// "set amount to 50" from 30 kT: unmet, the task keeps waiting.
		{"set amount, short", []Packet{packet(2, engine.Minerals{30, 0, 0})}, nil, false,
			named(1, 2, task(0, engine.SetAmount, 50)), engine.Minerals{30, 0, 0}, []engine.Minerals{{}}, nil, engine.SetAmount},
		// A packet launched this year waits at the origin until after
		// movement, then flies on: the fleet that reached it there loads
		// nothing.
		{"named packet gone on", []Packet{{Owner: 1, Number: 2, Pos: at(100, 0), Target: 7, Warp: 10, New: true, Cargo: engine.Minerals{100, 0, 0}}}, nil, false,
			engine.Waypoint{Pos: at(100, 0), Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 2, Task: loadIronium}, engine.Minerals{}, []engine.Minerals{{100, 0, 0}}, nil, engine.TransportNone},
		// "set waypoint to 10" from 30 kT: 20 load, the packet holds 10, met.
		{"set waypoint", []Packet{packet(2, engine.Minerals{30, 0, 0})}, nil, false,
			named(1, 2, task(0, engine.SetWaypoint, 10)), engine.Minerals{20, 0, 0}, []engine.Minerals{{10, 0, 0}}, nil, engine.TransportNone},
	} {
		l := newLab(t)
		l.g.Planets = []engine.Planet{{ID: 7, Pos: at(900, 0), Owner: engine.NoOwner}}
		if c.planet {
			l.g.Planets = append(l.g.Planets, engine.Planet{ID: 8, Pos: at(100, 0), Owner: 0, Surface: engine.Minerals{500, 0, 0}})
		}
		l.g.Objects = &Space{Packets: c.packets}
		l.g.Salvage = c.salvage
		fi := l.fleet(0, at(100, 10), "Freighter", 1)
		l.g.Fleets[fi].Fuel = 500
		l.g.Fleets[fi].Waypoints = []engine.Waypoint{c.wp}
		g := turn(t, *l.g, top{})
		f := g.Fleets[0]
		if f.Pos != at(100, 0) || f.Cargo.Minerals != c.cargo || f.Task.Transport[0].Action != c.action {
			t.Errorf("%s: fleet at %v cargo %v action %v; want at (100, 0), %v, %v", c.name, f.Pos, f.Cargo.Minerals, f.Task.Transport[0].Action, c.cargo, c.action)
		}
		for i, p := range space(g).Packets {
			if p.Cargo != c.left[i] {
				t.Errorf("%s: packet %d holds %v, want %v", c.name, p.Number, p.Cargo, c.left[i])
			}
		}
		for i, sv := range g.Salvage {
			if i < len(c.salvLeft) && sv.Minerals[0] != c.salvLeft[i] {
				t.Errorf("%s: salvage %d holds %v, want %d ironium", c.name, sv.Number, sv.Minerals, c.salvLeft[i])
			}
		}
		if c.planet && g.Planets[1].Surface[0] != 500 {
			t.Errorf("%s: planet surface %v, want 500 untouched", c.name, g.Planets[1].Surface)
		}
	}

	s := &Space{Packets: []Packet{{Owner: 1, Number: 1, Pos: origin, Cargo: engine.Minerals{1, 0, 0}}, {Owner: 1, Number: 2, Pos: at(5, 0), Cargo: engine.Minerals{2, 0, 0}}}}
	if c, pos := s.PacketCargo(1, 2); c != &s.Packets[1].Cargo || pos != at(5, 0) {
		t.Errorf("PacketCargo(1, 2) = %v at %v, want packet 2's own cargo at (5, 0)", c, pos)
	}
	if c, _ := s.PacketCargo(0, 2); c != nil {
		t.Errorf("PacketCargo of another owner's number %v, want nil", c)
	}
}
