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
// CONFIRMED FO-01 G, H). A set amount is unmet while the target holds
// less than v − cargo, whatever the free hold (KERNEL.md "Which loads are
// unmet", BINARY-ONLY); ASSUMPTION T11: a short set waypoint keeps the
// action. One with nothing to load is satisfied.
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
		{"set amount 300 of 300 there, hold short: satisfied", Transport{SetAmount, 300}, 0, 300, 210, false},
		{"set amount 300 of 250 there, hold short: unmet", Transport{SetAmount, 300}, 0, 250, 210, true},
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
		l.g.load(f, false)
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

// A set action with nothing to move is satisfied wherever the fleet is,
// so it moves on (ASSUMPTION T11; KERNEL.md "Loads with no usable source":
// a load that wants nothing is satisfied), even where the load pass loads
// nothing: another player's planet or deep space. A real load at another
// player's planet waits before movement, holding the fleet that year, and
// is cancelled after movement. The freighter heads 30 ly away at warp 6.
func TestSetActionNothingToMove(t *testing.T) {
	for _, tt := range []struct {
		name    string
		where   string // "own", "enemy" or "deep"
		tr      Transport
		held    int
		move    bool
		cleared bool
	}{
		{"own planet, set amount 50 with 50", "own", Transport{SetAmount, 50}, 50, true, true},
		{"enemy planet, set amount 50 with 50", "enemy", Transport{SetAmount, 50}, 50, true, true},
		{"deep space, set amount 50 with 50", "deep", Transport{SetAmount, 50}, 50, true, true},
		{"deep space, set waypoint 50, empty", "deep", Transport{SetWaypoint, 50}, 0, true, true},
		{"enemy planet, set waypoint 0, empty: a load", "enemy", Transport{SetWaypoint, 0}, 0, false, true},
	} {
		l := newTKLab(t, 3)
		medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
		owner := 0
		if tt.where == "enemy" {
			owner = 1
		}
		pi := l.planet(owner, 0, 100)
		dst := l.planet(0, 30, 100)
		l.g.Planets[pi].Surface[Ironium] = 50
		fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
		f := &l.g.Fleets[fi]
		if tt.where == "deep" {
			f.Pos = Point{0, 40}
		}
		start := f.Pos
		f.Fuel = 500
		f.Cargo.Minerals[Ironium] = tt.held
		f.Task = Task{Kind: TaskTransport}
		f.Task.Transport[Ironium] = tt.tr
		f.Waypoints = []Waypoint{{Pos: l.g.Planets[dst].Pos, Warp: 6, Target: TargetPlanet, ID: l.g.Planets[dst].ID}}
		g := l.turn(&seqRand{}).Game
		if moved := g.Fleets[0].Pos != start; moved != tt.move {
			t.Errorf("%s: at %v, moved %v, want %v (task %+v)", tt.name, g.Fleets[0].Pos, moved, tt.move, g.Fleets[0].Task)
		}
		if cleared := g.Fleets[0].Task.Kind == TaskNone; cleared != tt.cleared {
			t.Errorf("%s: task %+v, want cleared %v", tt.name, g.Fleets[0].Task, tt.cleared)
		}
	}
}

// The load pass settles a set action after every unload of the phase, so
// the result does not depend on fleet order (ASSUMPTION T11): at its own
// planet holding 100 Fe, fleet A sets the waypoint to 100 with an empty
// hold and fleet B unloads all its 50 Fe. Either way round, B's 50 lands
// and A loads it back, leaving the planet at 100.
func TestSetWaypointAfterOtherUnloads(t *testing.T) {
	for _, aFirst := range []bool{true, false} {
		l := newTKLab(t, 3)
		medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
		pi := l.planet(0, 0, 100)
		l.g.Planets[pi].Surface[Ironium] = 100
		ai := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
		bi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
		l.g.Fleets[ai].Number, l.g.Fleets[bi].Number = 1, 2
		if !aFirst {
			l.g.Fleets[ai].Number, l.g.Fleets[bi].Number = 2, 1
		}
		a, b := &l.g.Fleets[ai], &l.g.Fleets[bi]
		a.Task = Task{Kind: TaskTransport}
		a.Task.Transport[Ironium] = Transport{SetWaypoint, 100}
		b.Cargo.Minerals[Ironium] = 50
		b.Task = Task{Kind: TaskTransport}
		b.Task.Transport[Ironium] = Transport{Action: UnloadAll}
		l.g.unloadPhase(l.g.phaseStart())
		l.g.loadPass(false)
		a, b = &l.g.Fleets[ai], &l.g.Fleets[bi]
		if a.Cargo.Minerals[Ironium] != 50 || b.Cargo.Minerals[Ironium] != 0 || l.g.Planets[pi].Surface[Ironium] != 100 {
			t.Errorf("A first %v: A holds %d, B %d, planet %d; want 50, 0 and 100", aFirst, a.Cargo.Minerals[Ironium], b.Cargo.Minerals[Ironium], l.g.Planets[pi].Surface[Ironium])
		}
		if a.Task.Kind != TaskNone || b.Task.Kind != TaskNone {
			t.Errorf("A first %v: tasks %+v and %+v, want none", aFirst, a.Task, b.Task)
		}
	}
}

// Colonists at salvage: a set amount already met clears in the load
// pass, though nothing loads colonists there; one that would load waits
// (ASSUMPTION T6, T11; KERNEL.md "Which loads are unmet").
func TestSetColonistsAtSalvage(t *testing.T) {
	for _, tt := range []struct {
		v, held int
		left    bool
	}{{10, 10, false}, {20, 10, true}} {
		task := Task{Kind: TaskTransport}
		task.Transport[CargoColonists] = Transport{SetAmount, tt.v}
		l := salvageLab(t, Salvage{Minerals: Minerals{50, 0, 0}, Steps: 10}, task)
		l.g.Fleets[0].Cargo.Colonists = tt.held
		l.g.unloadPhase([]bool{})
		l.g.loadPass(false)
		f := l.g.Fleets[0]
		if left := f.Task.Kind == TaskTransport; left != tt.left || f.Cargo.Colonists != tt.held {
			t.Errorf("set amount %d with %d: task %+v, colonists %d; want left %v, %d", tt.v, tt.held, f.Task, f.Cargo.Colonists, tt.left, tt.held)
		}
	}
}
