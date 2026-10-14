package engine

import (
	"errors"
	"reflect"
	"testing"
)

// ftLab is two Medium Freighters with Long Hump 6 (cargo 210 kT, fuel
// 450 mg), X (id 1) and Y (id 2), of player 0 in deep space at (1200,
// 1200) with 200 mg each: the FO-02 setup (stars-elegy PARITY.md "Fleet
// Operations", "Method"). yOwner owns Y. Player 1's relation toward
// player 0 is rel; player 0 is an enemy toward player 1.
func ftLab(t *testing.T, yOwner int, rel Relation) *tkLab {
	l := newTKLab(t, 3)
	fr := l.design("Medium Freighter", SlotFill{0, "Long Hump 6", 1})
	pi := l.planet(NoOwner, 0, 0)
	for range 2 {
		fi := l.fleet(0, pi, 0, Stack{Design: fr, Count: 1})
		l.g.Fleets[fi].Pos = Point{1200, 1200}
		l.g.Fleets[fi].Fuel = 200
	}
	l.g.Fleets[1].Owner = yOwner
	l.g.Players[1].Relations = []Relation{rel, RelationNeutral}
	return l
}

// ftTurn generates the year with X following Y (its waypoint 0 aimed at
// Y) carrying task, and returns X, Y and the events.
func ftTurn(t *testing.T, l *tkLab, task Task) (x, y Fleet, events []Event) {
	t.Helper()
	r := followTurn(t, l, FollowOrder{Fleet: 1, Leader: 2, Task: task})
	return followed(t, r.Game, 1), followed(t, r.Game, 2), r.Events
}

func ftTask(c int, tr Transport) Task {
	task := Task{Kind: TaskTransport}
	task.Transport[c] = tr
	return task
}

func kindsOf(events []Event, kinds ...EventKind) []Event {
	var out []Event
	for _, e := range events {
		for _, k := range kinds {
			if e.Kind == k {
				out = append(out, e)
			}
		}
	}
	return out
}

func TestConfirmedFleetTransportClamps(t *testing.T) {
	// FO-02 L: X (0/0/100, free 110) loads all Ir and all Bo from Y
	// (150/60): 110 Ir, no Bo.
	l := ftLab(t, 0, RelationNeutral)
	l.g.Fleets[0].Cargo.Minerals = Minerals{0, 0, 100}
	l.g.Fleets[1].Cargo.Minerals = Minerals{150, 60, 0}
	task := ftTask(Ironium, Transport{Action: LoadAll})
	task.Transport[Boranium] = Transport{Action: LoadAll}
	x, y, _ := ftTurn(t, l, task)
	if x.Cargo.Minerals != (Minerals{110, 0, 100}) || y.Cargo.Minerals != (Minerals{40, 60, 0}) {
		t.Errorf("FO-02 L: X %v Y %v, want 110/0/100 and 40/60/0", x.Cargo.Minerals, y.Cargo.Minerals)
	}
	// FO-02 M: a full hold still takes fuel, up to the free tank.
	l = ftLab(t, 0, RelationNeutral)
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 210
	l.g.Fleets[1].Cargo.Minerals[Ironium] = 50
	l.g.Fleets[1].Fuel = 450
	task = ftTask(Ironium, Transport{Action: LoadAll})
	task.Transport[CargoFuel] = Transport{Action: LoadAll}
	x, y, _ = ftTurn(t, l, task)
	if x.Cargo.Minerals[Ironium] != 210 || x.Fuel != 450 || y.Cargo.Minerals[Ironium] != 50 || y.Fuel != 200 {
		t.Errorf("FO-02 M: X %d kT %d mg, Y %d kT %d mg, want 210 450 and 50 200", x.Cargo.Minerals[Ironium], x.Fuel, y.Cargo.Minerals[Ironium], y.Fuel)
	}
	// FO-02 N, O: unloads capped by the receiver's free space.
	l = ftLab(t, 0, RelationNeutral)
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 200
	l.g.Fleets[0].Fuel = 400
	l.g.Fleets[1].Cargo.Minerals[Ironium] = 100
	l.g.Fleets[1].Fuel = 300
	task = ftTask(Ironium, Transport{Action: UnloadAll})
	task.Transport[CargoFuel] = Transport{Action: UnloadExactly, Amount: 300}
	x, y, _ = ftTurn(t, l, task)
	if x.Cargo.Minerals[Ironium] != 90 || y.Cargo.Minerals[Ironium] != 210 || x.Fuel != 250 || y.Fuel != 450 {
		t.Errorf("FO-02 N, O: X %d kT %d mg, Y %d kT %d mg, want 90 250 and 210 450", x.Cargo.Minerals[Ironium], x.Fuel, y.Cargo.Minerals[Ironium], y.Fuel)
	}
}

func TestAssumptionFleetTransportTargetAway(t *testing.T) {
	// ASSUMPTION T16: a target fleet that is gone or not at the fleet's
	// position takes and gives nothing. The unload clears, keeping the
	// cargo; the load waits before movement and is cancelled after it
	// with 0x123, as in deep space.
	l := ftLab(t, 0, RelationNeutral)
	g := withRules(l.g)
	x := &g.Fleets[0]
	x.Cargo.Minerals[Ironium] = 50
	g.Fleets[1].Cargo.Minerals[Boranium] = 50
	g.Fleets[1].Pos = Point{1250, 1200}
	x.Task = ftTask(Ironium, Transport{Action: UnloadAll})
	x.Task.Transport[Boranium] = Transport{Action: LoadExactly, Amount: 30}
	x.Task.ToFleet, x.Task.Fleet = true, 2
	if ev := g.unload(x, nil, nil); len(ev) != 0 || x.Cargo.Minerals[Ironium] != 50 || x.Task.Transport[Ironium].Action != TransportNone {
		t.Errorf("unload to a fleet elsewhere: %v, X %v task %+v", ev, x.Cargo.Minerals, x.Task)
	}
	if ev := g.load(x, false); len(ev) != 0 || x.Task.Transport[Boranium].Action != LoadExactly {
		t.Errorf("load before movement from a fleet elsewhere: %v, task %+v, want it waiting", ev, x.Task)
	}
	want := []Event{{Kind: EventLoadRefused, Player: 0, Planet: -1, Fleet: 1, Axes: []int{Boranium}}}
	if ev := g.load(x, true); !reflect.DeepEqual(ev, want) || x.Task.Kind != TaskNone {
		t.Errorf("load after movement from a fleet elsewhere: %v, task %+v, want %v and no task", ev, x.Task, want)
	}
	// Gone.
	x.Cargo.Minerals[Ironium] = 50
	x.Task = ftTask(Ironium, Transport{Action: UnloadAll})
	x.Task.ToFleet, x.Task.Fleet = true, 99
	if g.unload(x, nil, nil); x.Cargo.Minerals[Ironium] != 50 || x.Task.Kind != TaskNone {
		t.Errorf("unload to a gone fleet: X %v task %+v", x.Cargo.Minerals, x.Task)
	}
}

func TestConfirmedForeignFleetTransport(t *testing.T) {
	// FO-04 and FO-05 A to C: minerals given when the receiver's relation
	// toward the giver is not enemy, nothing otherwise; colonists refused
	// with 0x155; a load from another player's fleet moves nothing and
	// gets 0x120 after movement.
	for _, c := range []struct {
		rel  Relation
		gets int
	}{{RelationNeutral, 50}, {RelationEnemy, 0}} {
		l := ftLab(t, 1, c.rel)
		l.g.Fleets[0].Cargo.Minerals[Ironium] = 50
		x, y, _ := ftTurn(t, l, ftTask(Ironium, Transport{Action: UnloadAll}))
		if y.Cargo.Minerals[Ironium] != c.gets || x.Cargo.Minerals[Ironium] != 50-c.gets {
			t.Errorf("relation %d: X %d Y %d, want Y %d", c.rel, x.Cargo.Minerals[Ironium], y.Cargo.Minerals[Ironium], c.gets)
		}
	}
	l := ftLab(t, 1, RelationNeutral)
	l.g.Fleets[0].Cargo.Colonists = 30
	x, y, ev := ftTurn(t, l, ftTask(CargoColonists, Transport{Action: UnloadAll}))
	want := []Event{{Kind: EventColonistsToForeignFleet, Player: 0, Planet: -1, Fleet: 1, Axes: []int{2}}}
	if got := kindsOf(ev, EventColonistsToForeignFleet); !reflect.DeepEqual(got, want) || x.Cargo.Colonists != 30 || y.Cargo.Colonists != 0 {
		t.Errorf("colonists: %v, X %d Y %d, want %v", got, x.Cargo.Colonists, y.Cargo.Colonists, want)
	}
	l = ftLab(t, 1, RelationNeutral)
	l.g.Fleets[1].Cargo.Minerals[Ironium] = 100
	x, y, ev = ftTurn(t, l, ftTask(Ironium, Transport{Action: LoadAll}))
	want = []Event{{Kind: EventFleetLoadRefused, Player: 0, Planet: -1, Fleet: 1, Axes: []int{Ironium, 2}}}
	if got := kindsOf(ev, EventFleetLoadRefused, EventLoadRefused); !reflect.DeepEqual(got, want) || x.Cargo.Minerals[Ironium] != 0 || y.Cargo.Minerals[Ironium] != 100 || x.Task.Kind != TaskNone {
		t.Errorf("load: %v, X %d Y %d task %+v, want %v", got, x.Cargo.Minerals[Ironium], y.Cargo.Minerals[Ironium], x.Task, want)
	}
}

func TestAssumptionForeignFleetFuelAndOrder(t *testing.T) {
	// ASSUMPTION T17: fuel to another player's fleet follows the
	// minerals' relation rule.
	for _, c := range []struct {
		rel  Relation
		gets int
	}{{RelationNeutral, 100}, {RelationEnemy, 0}} {
		l := ftLab(t, 1, c.rel)
		x, y, _ := ftTurn(t, l, ftTask(CargoFuel, Transport{Action: UnloadExactly, Amount: 100}))
		// The year's fuel use is none: neither fleet moves.
		if y.Fuel != 200+c.gets || x.Fuel != 200-c.gets {
			t.Errorf("fuel, relation %d: X %d Y %d, want Y %d", c.rel, x.Fuel, y.Fuel, 200+c.gets)
		}
	}
	// ASSUMPTION T18: the colonists' refusal cancels the task where it is
	// reached: the ironium before it moves, the fuel after it does not.
	l := ftLab(t, 1, RelationNeutral)
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 20
	l.g.Fleets[0].Cargo.Colonists = 10
	task := ftTask(Ironium, Transport{Action: UnloadAll})
	task.Transport[CargoColonists] = Transport{Action: UnloadAll}
	task.Transport[CargoFuel] = Transport{Action: UnloadAll}
	x, y, _ := ftTurn(t, l, task)
	if y.Cargo.Minerals[Ironium] != 20 || x.Cargo.Colonists != 10 || x.Fuel != 200 || y.Fuel != 200 {
		t.Errorf("X %v %d mg, Y %v %d mg, want Y 20 Ir, X 10 colonists, fuel unchanged", x.Cargo, x.Fuel, y.Cargo, y.Fuel)
	}
}

func TestAssumptionForeignFleetLoadSilent(t *testing.T) {
	// ASSUMPTION T19: a load of colonists or fuel from another player's
	// fleet is cancelled after movement with no message; minerals get
	// 0x120 (TestConfirmedForeignFleetTransport).
	for _, c := range []int{CargoColonists, CargoFuel} {
		l := ftLab(t, 1, RelationNeutral)
		l.g.Fleets[1].Cargo.Colonists = 50
		x, y, ev := ftTurn(t, l, ftTask(c, Transport{Action: LoadAll}))
		if got := kindsOf(ev, EventFleetLoadRefused, EventLoadRefused); len(got) != 0 || x.Task.Kind != TaskNone || *held(&x, c) != map[int]int{CargoColonists: 0, CargoFuel: 200}[c] || y.Cargo.Colonists != 50 {
			t.Errorf("cargo %d: %v, X task %+v holds %d", c, got, x.Task, *held(&x, c))
		}
	}
	// ASSUMPTION T10 for fleets: a fleet that can steal from fleets keeps
	// waiting; stealing is not modelled.
	l := ftLab(t, 1, RelationNeutral)
	l.g.Fleets[0].Stacks[0].Design = l.design("Medium Freighter", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Pick Pocket Scanner", 1})
	l.g.Fleets[1].Cargo.Minerals[Ironium] = 100
	x, _, ev := ftTurn(t, l, ftTask(Ironium, Transport{Action: LoadAll}))
	if got := kindsOf(ev, EventFleetLoadRefused, EventLoadRefused); len(got) != 0 || x.Task.Transport[Ironium].Action != LoadAll || x.Cargo.Minerals[Ironium] != 0 {
		t.Errorf("stealer: %v, task %+v, Ir %d, want waiting with nothing loaded", got, x.Task, x.Cargo.Minerals[Ironium])
	}
}

func TestConfirmedTransportFuelIgnoredAtPlanet(t *testing.T) {
	// FM-102-N (MEASURED; KERNEL.md, CONFIRMED FM-101..105): "unload all"
	// fuel at the owner's own planet keeps the fuel and clears. FO-01 E:
	// "load all" fuel there loads none, and the ironium still loads.
	l := ftLab(t, 0, RelationNeutral)
	pi := l.planet(0, 50, 1000)
	l.g.Planets[pi].Surface[Ironium] = 100
	g := withRules(l.g)
	x := &g.Fleets[0]
	x.Pos = g.Planets[pi].Pos
	x.Task = ftTask(CargoFuel, Transport{Action: UnloadAll})
	g.unload(x, nil, nil)
	if x.Fuel != 200 || x.Task.Kind != TaskNone {
		t.Errorf("unload all fuel: %d mg task %+v, want 200 and no task", x.Fuel, x.Task)
	}
	x.Task = ftTask(CargoFuel, Transport{Action: LoadAll})
	x.Task.Transport[Ironium] = Transport{Action: LoadAll}
	g.load(x, false)
	if x.Fuel != 200 || x.Cargo.Minerals[Ironium] != 100 || x.Task.Kind != TaskNone {
		t.Errorf("load all fuel and Ir: %d mg %d kT task %+v, want 200, 100 and no task", x.Fuel, x.Cargo.Minerals[Ironium], x.Task)
	}
}

func TestAssumptionTransportFuelLoadClears(t *testing.T) {
	// ASSUMPTION T20: a fuel load in deep space, as at a planet, clears
	// as a load that wants nothing; a "wait for" fuel does not hold the
	// fleet.
	l := ftLab(t, 0, RelationNeutral)
	g := withRules(l.g)
	x := &g.Fleets[0]
	x.Task = ftTask(CargoFuel, Transport{Action: WaitFor, Amount: 100})
	if ev := g.load(x, false); len(ev) != 0 || x.Task.Kind != TaskNone {
		t.Errorf("wait for fuel in deep space: %v, task %+v, want no task", ev, x.Task)
	}
}

func TestConfirmedTransportOnArrivalAtFleet(t *testing.T) {
	// FO-02 S, U: X arrives at stationary Y, its waypoint aimed at Y, and
	// unloads (or loads) after movement.
	for _, load := range []bool{false, true} {
		l := ftLab(t, 0, RelationNeutral)
		l.g.Fleets[0].Pos = Point{1170, 1200}
		tr := Transport{Action: UnloadAll}
		from, to := 0, 1
		if load {
			tr, from, to = Transport{Action: LoadAll}, 1, 0
		}
		l.g.Fleets[from].Cargo.Minerals[Ironium] = 100
		l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1200, 1200}, Warp: 6, Target: TargetFleet, ID: 2, Task: ftTask(Ironium, tr)}}
		r := followTurn(t, l)
		x, y := followed(t, r.Game, 1), followed(t, r.Game, 2)
		fl := []Fleet{x, y}
		if x.Pos != y.Pos || fl[to].Cargo.Minerals[Ironium] != 100 || fl[from].Cargo.Minerals[Ironium] != 0 {
			t.Errorf("load %v: X at %v with %d, Y at %v with %d", load, x.Pos, x.Cargo.Minerals[Ironium], y.Pos, y.Cargo.Minerals[Ironium])
		}
	}
}

func TestFleetTargetDroppedAtYearEnd(t *testing.T) {
	// ORDERS.md "Waypoint 0 aimed at a fleet" (BINARY-ONLY): a waypoint 0
	// still aimed at a fleet is the planet or deep space there when the
	// files are written, so a waiting transport task (here "wait for"
	// 100% of ironium from Y, which holds 50) is left aimed at deep space.
	l := ftLab(t, 0, RelationNeutral)
	l.g.Fleets[1].Cargo.Minerals[Ironium] = 50
	x, y, _ := ftTurn(t, l, ftTask(Ironium, Transport{Action: WaitFor, Amount: 100}))
	if x.Cargo.Minerals[Ironium] != 50 || y.Cargo.Minerals[Ironium] != 0 {
		t.Errorf("X %d Y %d, want 50 and 0", x.Cargo.Minerals[Ironium], y.Cargo.Minerals[Ironium])
	}
	if x.Task.Kind != TaskTransport || x.Task.ToFleet || x.Task.Fleet != 0 || x.Task.Transport[Ironium].Action != WaitFor {
		t.Errorf("X task %+v, want the wait for, no longer aimed at a fleet", x.Task)
	}
}

func TestAssumptionFollowRetargetsTransport(t *testing.T) {
	// ASSUMPTION T21: a follower whose leader is elsewhere follows the
	// first fleet in fleet order of the leader's owner at its position
	// (followLeader), and its transport task is aimed there too. Y (id 2)
	// follows a fleet 100 ly away; X (id 1) is the first fleet at Y's
	// position, so Y loads from X.
	l := ftLab(t, 0, RelationNeutral)
	fi := l.fleet(0, 0, 0, l.g.Fleets[1].Stacks...)
	l.g.Fleets[fi].Pos, l.g.Fleets[fi].Fuel = Point{1300, 1200}, 200
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 40
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: l.g.Fleets[fi].ID, Task: ftTask(Ironium, Transport{Action: LoadAll})})
	if x, y := followed(t, r.Game, 1), followed(t, r.Game, 2); x.Cargo.Minerals[Ironium] != 0 || y.Cargo.Minerals[Ironium] != 40 {
		t.Errorf("X %d Y %d, want Y to take X's 40", x.Cargo.Minerals[Ironium], y.Cargo.Minerals[Ironium])
	}
}

func TestTransportToFleetNotAnOrder(t *testing.T) {
	// Task.ToFleet is the engine's: an order carrying it is refused.
	task := ftTask(Ironium, Transport{Action: LoadAll})
	task.ToFleet = true
	if err := validTask(task); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("validTask: %v, want ErrOutOfRange", err)
	}
}

func TestAssumptionFuelPercentOfTank(t *testing.T) {
	// ASSUMPTION T22: "fill to 100%" fuel fills the 450 mg tank, not
	// 100% of the 210 kT hold.
	l := ftLab(t, 0, RelationNeutral)
	l.g.Fleets[1].Fuel = 450
	x, y, _ := ftTurn(t, l, ftTask(CargoFuel, Transport{Action: FillTo, Amount: 100}))
	if x.Fuel != 450 || y.Fuel != 200 {
		t.Errorf("X %d mg Y %d mg, want 450 and 200", x.Fuel, y.Fuel)
	}
}

func TestAssumptionTransportAimedAtItself(t *testing.T) {
	// ASSUMPTION T16: a follower left following itself (its leader 100 ly
	// away, no fleet of the leader's owner at its position but itself)
	// has no target: its "wait for" waits before movement and is
	// cancelled after it with 0x123, as in deep space.
	l := ftLab(t, 0, RelationNeutral)
	l.g.Fleets[1].Pos = Point{1300, 1200}
	l.g.Fleets[0].Cargo.Minerals[Ironium] = 50
	r := followTurn(t, l, FollowOrder{Fleet: 1, Leader: 2, Task: ftTask(Ironium, Transport{Action: WaitFor, Amount: 100})})
	want := []Event{{Kind: EventLoadRefused, Player: 0, Planet: -1, Fleet: 1, Axes: []int{Ironium}}}
	if got := kindsOf(r.Events, EventLoadRefused, EventFleetLoadRefused); !reflect.DeepEqual(got, want) || followed(t, r.Game, 1).Task.Kind != TaskNone {
		t.Errorf("events %v task %+v, want %v and no task", got, followed(t, r.Game, 1).Task, want)
	}
}

func TestAssumptionFuelWantWithNoFleetSource(t *testing.T) {
	// ASSUMPTION T16, T22: with no usable fleet source, "fill to 50%"
	// fuel still wants against the 450 mg tank, not the 210 kT hold: at
	// 150 mg it wants 75 mg (half the hold, 105, it already exceeds).
	// So it waits before movement, whether the target is gone, elsewhere
	// or another player's, and after movement is cancelled, with 0x123
	// for a gone target.
	for _, c := range []struct {
		name string
		edit func(g *Game)
	}{
		{"gone", func(g *Game) { g.Fleets[0].Task.Fleet = 99 }},
		{"elsewhere", func(g *Game) { g.Fleets[1].Pos = Point{1250, 1200} }},
		{"foreign", func(g *Game) { g.Fleets[1].Owner = 1 }},
	} {
		l := ftLab(t, 0, RelationNeutral)
		g := withRules(l.g)
		x := &g.Fleets[0]
		x.Fuel = 150
		x.Task = ftTask(CargoFuel, Transport{Action: FillTo, Amount: 50})
		x.Task.ToFleet, x.Task.Fleet = true, 2
		c.edit(&g)
		if ev := g.load(x, false); len(ev) != 0 || x.Task.Transport[CargoFuel].Action != FillTo {
			t.Errorf("%s, before movement: %v, task %+v, want it waiting", c.name, ev, x.Task)
		}
		if c.name != "gone" {
			continue
		}
		want := []Event{{Kind: EventLoadRefused, Player: 0, Planet: -1, Fleet: 1, Axes: []int{CargoFuel}}}
		if ev := g.load(x, true); !reflect.DeepEqual(ev, want) || x.Task.Kind != TaskNone || x.Fuel != 150 {
			t.Errorf("%s, after movement: %v, task %+v, fuel %d; want %v and no task", c.name, ev, x.Task, x.Fuel, want)
		}
	}
}
