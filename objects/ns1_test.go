package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// NS-1 (stars-elegy a1b3651, PARITY.md "A transport task whose waypoint
// names a packet or salvage (NS-1)"): a transport task loads from the
// object its waypoint names, never from a planet at the same spot. Every
// fleet is player 0's empty freighter with "load all ironium, load all
// boranium"; every named object holds 100 kT boranium and every planet
// 50 kT ironium.
func ns1Lab(t *testing.T, planetOwner int) (*lab, int) {
	l := newLab(t)
	l.g.Planets = []engine.Planet{
		{ID: 7, Pos: at(900, 0), Owner: engine.NoOwner},
		{ID: 8, Pos: at(100, 0), Owner: planetOwner, Surface: engine.Minerals{50, 0, 0}},
	}
	l.g.Objects = &Space{}
	fi := l.fleet(0, at(100, 0), "Freighter", 1)
	l.g.Fleets[fi].Fuel = 500
	return l, fi
}

func ns1Task() engine.Task {
	k := engine.Task{Kind: engine.TaskTransport}
	k.Transport[engine.Ironium] = engine.Transport{Action: engine.LoadAll}
	k.Transport[engine.Boranium] = engine.Transport{Action: engine.LoadAll}
	return k
}

// Waypoint 0 names the object (A3–A6, B1, B2, NS-2 N and F; CONFIRMED by
// prediction):
// before movement the fleet loads the object's 100 kT boranium and no
// ironium from the planet, gets 0x04e and no 0x11e, and stays. The
// object, emptied before movement, is gone at the end of the year
// (MEASURED), and the finished task leaves no named object (MEASURED:
// waypoint 0 then named the planet).
func TestNS1WaypointZeroNamed(t *testing.T) {
	boranium := engine.Minerals{0, 100, 0}
	for _, c := range []struct {
		name        string
		planetOwner int
		salvage     *engine.Salvage
		packet      *Packet
	}{
		{"A3 own salvage over own planet", 0, &engine.Salvage{Pos: at(100, 0), Owner: 0, Number: 0, Minerals: boranium, Steps: 10}, nil},
		{"A4 own salvage over player 1's planet", 1, &engine.Salvage{Pos: at(100, 0), Owner: 0, Number: 0, Minerals: boranium, Steps: 10}, nil},
		{"A5 own packet over own planet", 0, nil, &Packet{Owner: 0, Number: 0, Pos: at(100, 0), Target: 7, Warp: 5, Cargo: boranium}},
		{"A6 player 1's packet over player 1's planet", 1, nil, &Packet{Owner: 1, Number: 0, Pos: at(100, 0), Target: 7, Warp: 5, Cargo: boranium}},
		{"B1 own packet 40 ly out", 0, nil, &Packet{Owner: 0, Number: 0, Pos: at(140, 0), Target: 7, Warp: 5, Cargo: boranium}},
		{"B2 player 1's packet 40 ly out", 1, nil, &Packet{Owner: 1, Number: 0, Pos: at(140, 0), Target: 7, Warp: 5, Cargo: boranium}},
		// NS-2 N and F (stars-elegy f5bffd8, PARITY.md "A named packet at
		// other distances and after movement (NS-2)"; CONFIRMED by
		// prediction).
		{"NS-2 N own packet 10 ly out", 0, nil, &Packet{Owner: 0, Number: 0, Pos: at(110, 0), Target: 7, Warp: 5, Cargo: boranium}},
		{"NS-2 F own packet 150 ly out", 0, nil, &Packet{Owner: 0, Number: 0, Pos: at(250, 0), Target: 7, Warp: 5, Cargo: boranium}},
	} {
		l, fi := ns1Lab(t, c.planetOwner)
		f := &l.g.Fleets[fi]
		f.Task = ns1Task()
		if c.salvage != nil {
			l.g.Salvage = []engine.Salvage{*c.salvage}
			f.Target0 = &engine.ObjectRef{Owner: c.salvage.Owner, Number: c.salvage.Number}
		} else {
			space(*l.g).Packets = []Packet{*c.packet}
			f.Target0 = &engine.ObjectRef{Owner: c.packet.Owner, Number: c.packet.Number}
		}
		r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
		if err != nil {
			t.Fatal(err)
		}
		g := r.Game
		got := g.Fleets[0]
		if got.Pos != at(100, 0) || got.Cargo.Minerals != boranium || got.Task.Kind != engine.TaskNone || got.Target0 != nil {
			t.Errorf("%s: fleet at %v cargo %v task %v target %v; want at the planet, the 100 boranium, no task, no named object", c.name, got.Pos, got.Cargo.Minerals, got.Task.Kind, got.Target0)
		}
		if s := g.Planets[1].Surface; s[engine.Ironium] != 50 {
			t.Errorf("%s: planet surface %v, want its 50 ironium untouched", c.name, s)
		}
		if len(g.Salvage) != 0 || len(space(g).Packets) != 0 {
			t.Errorf("%s: salvage %v packets %v, want the emptied object gone", c.name, g.Salvage, space(g).Packets)
		}
		arrived := 0
		for _, e := range r.Events {
			switch {
			case e.Kind == engine.EventTransportNoHold:
				t.Errorf("%s: 0x11e %+v", c.name, e)
			case e.Kind == engine.EventFleetArrived && e.Fleet == got.ID:
				arrived++
			}
		}
		if arrived != 1 {
			t.Errorf("%s: %d 0x04e, want 1", c.name, arrived)
		}
	}
}

// Arrival at salvage over a planet (A1, A2; MEASURED): the year's decay
// runs before the load after movement, so the fleet finds 100 − 10 = 90
// kT and loads that, none of the planet's ironium. The emptied salvage
// stays until the next decay.
func TestNS1ArrivalAfterDecay(t *testing.T) {
	for _, owner := range []int{0, 1} {
		l, fi := ns1Lab(t, owner)
		l.g.Salvage = []engine.Salvage{{Pos: at(100, 0), Owner: 0, Number: 0, Minerals: engine.Minerals{0, 100, 0}, Steps: 10}}
		f := &l.g.Fleets[fi]
		f.Pos = at(80, 0)
		f.Waypoints = []engine.Waypoint{{Pos: at(100, 0), Warp: 5, Target: engine.TargetPacketObject, Owner: 0, ID: 0, Task: ns1Task()}}
		g := turn(t, *l.g, top{})
		got := g.Fleets[0]
		if got.Pos != at(100, 0) || got.Cargo.Minerals != (engine.Minerals{0, 90, 0}) || got.Task.Kind != engine.TaskNone {
			t.Errorf("planet owner %d: fleet at %v cargo %v task %v; want at the salvage, 90 boranium, no task", owner, got.Pos, got.Cargo.Minerals, got.Task.Kind)
		}
		if s := g.Planets[1].Surface; s[engine.Ironium] != 50 {
			t.Errorf("planet owner %d: planet surface %v, want its 50 ironium untouched", owner, s)
		}
		if len(g.Salvage) != 1 || g.Salvage[0].Minerals != (engine.Minerals{}) {
			t.Errorf("planet owner %d: salvage %v, want it kept, empty", owner, g.Salvage)
		}
	}
}

// The named object is kept across years and changed only by arriving,
// setting off or a waypoint order (ASSUMPTION T14): a fleet still
// waiting on a packet loads from it the next year, before movement.
func TestNS1NamedObjectKept(t *testing.T) {
	l, fi := ns1Lab(t, 0)
	space(*l.g).Packets = []Packet{{Owner: 1, Number: 3, Pos: at(100, 0), Target: 7, Warp: 5, Cargo: engine.Minerals{0, 30, 0}}}
	f := &l.g.Fleets[fi]
	wait := engine.Task{Kind: engine.TaskTransport}
	wait.Transport[engine.Boranium] = engine.Transport{Action: engine.SetAmount, Amount: 60}
	f.Waypoints = []engine.Waypoint{{Pos: at(100, 0), Warp: 5, Target: engine.TargetPacketObject, Owner: 1, ID: 3, Task: wait}}
	g := turn(t, *l.g, top{})
	got := g.Fleets[0]
	if got.Target0 == nil || *got.Target0 != (engine.ObjectRef{Owner: 1, Number: 3}) || got.Task.Kind != engine.TaskTransport {
		t.Fatalf("after arriving: target %v task %v, want packet 1/3 kept with the task waiting", got.Target0, got.Task)
	}
	// The packet flew 25 ly toward planet 7 before fleets moved, and the
	// fleet followed it there at warp 5 and loaded what it held.
	loaded := got.Cargo.Minerals[engine.Boranium]
	if got.Pos != at(125, 0) || loaded == 0 {
		t.Fatalf("after arriving: fleet at %v with %d boranium, want at the packet with some loaded", got.Pos, loaded)
	}
	// Next year the waiting fleet stays, the packet holds 40 kT more, and
	// the load before movement takes from it.
	space(g).Packets[0].Cargo[engine.Boranium] += 40
	g = turn(t, g, top{})
	got = g.Fleets[0]
	if got.Pos != at(125, 0) || got.Cargo.Minerals[engine.Boranium] <= loaded {
		t.Errorf("next year: fleet at %v with %d boranium (had %d); want more boranium from the packet", got.Pos, got.Cargo.Minerals[engine.Boranium], loaded)
	}

	// Reaching a waypoint that names anything else clears it, here the
	// planet under the fleet at warp 0, reached without setting off.
	l, fi = ns1Lab(t, 0)
	l.g.Fleets[fi].Target0 = &engine.ObjectRef{Owner: 1, Number: 3}
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: at(100, 0), Target: engine.TargetPlanet, ID: 8}}
	g = turn(t, *l.g, top{})
	if got := g.Fleets[0]; got.Pos != at(100, 0) || len(got.Waypoints) != 0 || got.Target0 != nil {
		t.Errorf("reaching the planet: fleet at %v waypoints %v target %v, want at the planet with no named object", got.Pos, got.Waypoints, got.Target0)
	}

	// Setting off clears it.
	l, fi = ns1Lab(t, 0)
	l.g.Fleets[fi].Target0 = &engine.ObjectRef{Owner: 1, Number: 3}
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: at(100, 300), Warp: 5}}
	g = turn(t, *l.g, top{})
	if got := g.Fleets[0]; got.Pos == at(100, 0) || got.Target0 != nil {
		t.Errorf("setting off: fleet at %v target %v, want moved with no named object", got.Pos, got.Target0)
	}
}

// Waypoint 1 names a packet 20 ly ahead (NS-2 P1 and P1-F, stars-elegy
// f5bffd8; MEASURED): the packet moves 25 ly before the fleets and the
// waypoint follows it. At warp 5 the fleet falls short and loads nothing
// (P1); at warp 9 it reaches the packet's new position and loads its 100
// kT boranium after movement, and the emptied packet stays (P1-F).
func TestNS2AfterMovement(t *testing.T) {
	for _, c := range []struct {
		name  string
		warp  int
		pos   engine.Point
		cargo engine.Minerals
		left  engine.Minerals
	}{
		{"P1 short", 5, at(125, 0), engine.Minerals{}, engine.Minerals{0, 100, 0}},
		{"P1-F reached", 9, at(145, 0), engine.Minerals{0, 100, 0}, engine.Minerals{}},
	} {
		l, fi := ns1Lab(t, 0)
		space(*l.g).Packets = []Packet{{Owner: 0, Number: 0, Pos: at(120, 0), Target: 7, Warp: 5, Cargo: engine.Minerals{0, 100, 0}}}
		l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: at(120, 0), Warp: c.warp, Target: engine.TargetPacketObject, Owner: 0, ID: 0, Task: ns1Task()}}
		g := turn(t, *l.g, top{})
		got := g.Fleets[0]
		if got.Pos != c.pos || got.Cargo.Minerals != c.cargo {
			t.Errorf("%s: fleet at %v cargo %v, want at %v with %v", c.name, got.Pos, got.Cargo.Minerals, c.pos, c.cargo)
		}
		if ps := space(g).Packets; len(ps) != 1 || ps[0].Pos != at(145, 0) || ps[0].Cargo != c.left {
			t.Errorf("%s: packets %v, want one at (145, 0) holding %v", c.name, ps, c.left)
		}
	}
}
