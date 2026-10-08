package engine

import "testing"

// setLab is a Medium Freighter (210 kT) of player 0 holding cargo, with
// one action on cargo type c, at player 0's planet with surface ironium
// 300 and population 87.
func setLab(t *testing.T, c int, tr Transport, cargo Cargo) *tkLab {
	l := newTKLab(t, 3)
	medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
	pi := l.planet(0, 0, 87)
	l.g.Planets[pi].Surface[Ironium] = 300
	fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
	f := &l.g.Fleets[fi]
	f.Cargo = cargo
	f.Task = Task{Kind: TaskTransport}
	f.Task.Transport[c] = tr
	return l
}

// The load direction of "set amount to v" (v − C) and "set waypoint to
// v" (A − v), in the load pass (TAKEOVER.md "Unload and load amounts",
// CONFIRMED FO-01 G, H). ASSUMPTION T11: a load that falls short keeps
// the action, like "load exactly"; one with nothing to load is satisfied.
func TestSetAmountAndWaypointLoad(t *testing.T) {
	for _, tt := range []struct {
		name      string
		tr        Transport
		held      int
		surface   int
		load      int
		keepsTask bool
	}{
		{"set amount 80", Transport{SetAmount, 80}, 0, 300, 80, false},
		{"set amount 80, 30 aboard", Transport{SetAmount, 80}, 30, 300, 50, false},
		{"set amount 300, hold short", Transport{SetAmount, 300}, 0, 300, 210, true},
		{"set amount 100, surface short", Transport{SetAmount, 100}, 0, 40, 40, true},
		{"set amount 80, 100 aboard", Transport{SetAmount, 80}, 100, 300, 0, false},
		{"set waypoint 100", Transport{SetWaypoint, 100}, 0, 300, 200, false},
		{"set waypoint 50, hold short", Transport{SetWaypoint, 50}, 0, 300, 210, true},
		{"set waypoint 100, exactly met", Transport{SetWaypoint, 100}, 110, 200, 100, false},
		{"set waypoint 500", Transport{SetWaypoint, 500}, 0, 300, 0, false},
	} {
		l := setLab(t, Ironium, tt.tr, Cargo{Minerals: Minerals{tt.held, 0, 0}})
		l.g.Planets[0].Surface[Ironium] = tt.surface
		f := &l.g.Fleets[0]
		l.g.load(f)
		if got := f.Cargo.Minerals[Ironium] - tt.held; got != tt.load || l.g.Planets[0].Surface[Ironium] != tt.surface-tt.load {
			t.Errorf("%s: loaded %d (surface %d), want %d", tt.name, got, l.g.Planets[0].Surface[Ironium], tt.load)
		}
		if kept := f.Task.Kind == TaskTransport && f.Task.Transport[Ironium] == tt.tr; kept != tt.keepsTask {
			t.Errorf("%s: task %+v, want kept %v", tt.name, f.Task, tt.keepsTask)
		}
	}
}

// The unload direction, in the unload phase: "set amount to v" unloads
// C − v; "set waypoint to v" unloads the shortfall v − A, capped by the
// cargo (CONFIRMED TK-114 U4: ironium 100 → 60; U5: colonists 120
// against 87 unload 33). Either action clears once it unloads; one that
// would load is left to the load pass.
func TestSetAmountAndWaypointUnload(t *testing.T) {
	for _, tt := range []struct {
		name     string
		c        int
		tr       Transport
		cargo    Cargo
		unload   int
		leftTask bool
	}{
		{"set amount Ir 60", Ironium, Transport{SetAmount, 60}, Cargo{Minerals: Minerals{100, 0, 0}}, 40, false},
		{"set amount colonists 10", CargoColonists, Transport{SetAmount, 10}, Cargo{Colonists: 50}, 40, false},
		{"set amount Ir 80, a load", Ironium, Transport{SetAmount, 80}, Cargo{Minerals: Minerals{50, 0, 0}}, 0, true},
		{"set waypoint colonists 120", CargoColonists, Transport{SetWaypoint, 120}, Cargo{Colonists: 50}, 33, false},
		{"set waypoint colonists 200, cargo short", CargoColonists, Transport{SetWaypoint, 200}, Cargo{Colonists: 50}, 50, false},
		{"set waypoint Ir 100, a load", Ironium, Transport{SetWaypoint, 100}, Cargo{Minerals: Minerals{50, 0, 0}}, 0, true},
	} {
		l := setLab(t, tt.c, tt.tr, tt.cargo)
		f := &l.g.Fleets[0]
		before := l.g.Planets[0]
		l.g.unloadPhase(l.g.phaseStart())
		p := l.g.Planets[0]
		got := p.Surface[Ironium] - before.Surface[Ironium]
		if tt.c == CargoColonists {
			got = p.Population - before.Population
		}
		if got != tt.unload {
			t.Errorf("%s: unloaded %d, want %d", tt.name, got, tt.unload)
		}
		if left := f.Task.Kind == TaskTransport; left != tt.leftTask {
			t.Errorf("%s: task %+v, want left %v", tt.name, f.Task, tt.leftTask)
		}
	}
}

// Away from a planet, "set waypoint to" measures A at the salvage
// object under the fleet (ASSUMPTION T5), and in deep space A is 0
// (ASSUMPTION T12), so the shortfall is unloaded and destroyed.
func TestSetWaypointAwayFromPlanet(t *testing.T) {
	task := Task{Kind: TaskTransport}
	task.Transport[Ironium] = Transport{SetWaypoint, 50}
	l := salvageLab(t, Salvage{Minerals: Minerals{30, 0, 0}, Steps: 10}, task)
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 40
	l.g.unloadPhase([]bool{})
	if sv, f := l.g.Salvage[0].Minerals[Ironium], l.g.Fleets[0].Cargo.Minerals[Ironium]; sv != 50 || f != 20 {
		t.Errorf("salvage: salvage %d, fleet %d; want 50 and 20", sv, f)
	}
	l = salvageLab(t, Salvage{Minerals: Minerals{30, 0, 0}, Steps: 10}, task)
	l.g.Salvage = nil
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 40
	l.g.unloadPhase([]bool{})
	if f := l.g.Fleets[0].Cargo.Minerals[Ironium]; f != 0 {
		t.Errorf("deep space: fleet %d, want 0 (40 destroyed)", f)
	}
}

// Loading from salvage (ASSUMPTION T5) takes the same amounts, with A the
// salvage object's holding: set amount to 50 with 20 aboard loads 30 and
// clears; set waypoint to 100 from 400 wants 300, the 160 kT of space
// left loads 160, and the 240 left behind keep the action (T11).
func TestSetAmountAndWaypointFromSalvage(t *testing.T) {
	task := Task{Kind: TaskTransport}
	task.Transport[Ironium] = Transport{SetAmount, 50}
	task.Transport[Boranium] = Transport{SetWaypoint, 100}
	l := salvageLab(t, Salvage{Minerals: Minerals{100, 400, 0}, Steps: 50}, task)
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 20
	l.g.loadPass(false)
	f, sv := l.g.Fleets[0], l.g.Salvage[0]
	if f.Cargo.Minerals != (Minerals{50, 160, 0}) || sv.Minerals != (Minerals{70, 240, 0}) {
		t.Errorf("fleet %v, salvage %v; want [50 160 0] and [70 240 0]", f.Cargo.Minerals, sv.Minerals)
	}
	want := [NumCargo]Transport{Boranium: {SetWaypoint, 100}}
	if f.Task.Kind != TaskTransport || f.Task.Transport != want {
		t.Errorf("task %+v, want %+v", f.Task, want)
	}
}
