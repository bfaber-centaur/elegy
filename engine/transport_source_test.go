package engine

import (
	"reflect"
	"testing"
)

// sourceLab is a Medium Freighter (210 kT) of player 0 with task tr on
// ironium (and boranium, if trB is set): at its own planet ("own"),
// player 1's ("enemy"), or in deep space ("deep"). The planet holds 50
// Fe and no Bo.
func sourceLab(t *testing.T, where string, tr, trB Transport, held int) (*tkLab, int) {
	l := newTKLab(t, 3)
	medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
	owner := 0
	if where == "enemy" {
		owner = 1
	}
	pi := l.planet(owner, 0, 100)
	l.g.Planets[pi].Surface[Ironium] = 50
	fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
	f := &l.g.Fleets[fi]
	if where == "deep" {
		f.Pos = Point{0, 40}
	}
	f.Cargo.Minerals[Ironium] = held
	f.Task = Task{Kind: TaskTransport}
	f.Task.Transport[Ironium], f.Task.Transport[Boranium] = tr, trB
	return l, pi
}

// Loads with no usable source (KERNEL.md "Loads with no usable source",
// BINARY-ONLY): deep space with no salvage and another player's planet.
// A load that wants nothing clears silently. One that wants more waits
// before movement with no message, and after movement is cancelled with
// "could not load" (MESSAGES.md 0x123 deep space, 0x11f planet), a set
// amount more than the target holds first told it waits (0x121).
// ASSUMPTION T13: the first cargo type, in cargo order, that wants more
// is the one refused, and the cancel ends the whole task.
func TestLoadWithNoSource(t *testing.T) {
	refused := func(planet, c int) Event {
		return Event{Kind: EventLoadRefused, Player: 0, Planet: planet, Fleet: 1, Axes: []int{c}}
	}
	waiting := func(planet, c, v int) Event {
		return Event{Kind: EventLoadWaiting, Player: 0, Planet: planet, Fleet: 1, Count: v, Axes: []int{c}}
	}
	for _, tt := range []struct {
		name      string
		where     string
		tr, trB   Transport
		held      int
		waits     bool // before movement
		after     []Event
		planetEvs bool // the events name the planet, not -1
	}{
		{"deep, load all: wants nothing", "deep", Transport{Action: LoadAll}, Transport{}, 0, false, nil, false},
		{"deep, load exactly 10", "deep", Transport{LoadExactly, 10}, Transport{}, 0, true, []Event{refused(-1, Ironium)}, false},
		{"deep, fill to 50%", "deep", Transport{FillTo, 50}, Transport{}, 0, true, []Event{refused(-1, Ironium)}, false},
		{"deep, fill to 50%, already 105", "deep", Transport{FillTo, 50}, Transport{}, 105, false, nil, false},
		{"deep, set amount 30", "deep", Transport{SetAmount, 30}, Transport{}, 0, true, []Event{waiting(-1, Ironium, 30), refused(-1, Ironium)}, false},
		{"enemy, load all Bo: none there", "enemy", Transport{}, Transport{Action: LoadAll}, 0, false, nil, true},
		{"enemy, load all Fe", "enemy", Transport{Action: LoadAll}, Transport{}, 0, true, []Event{refused(0, Ironium)}, true},
		{"enemy, wait for 50%", "enemy", Transport{WaitFor, 50}, Transport{}, 0, true, []Event{refused(0, Ironium)}, true},
		{"enemy, set amount 30 of 50 there", "enemy", Transport{SetAmount, 30}, Transport{}, 0, true, []Event{refused(0, Ironium)}, true},
		{"enemy, set amount 100 of 50 there", "enemy", Transport{SetAmount, 100}, Transport{}, 0, true, []Event{waiting(0, Ironium, 100), refused(0, Ironium)}, true},
		{"enemy, set amount 100 with 50 aboard", "enemy", Transport{SetAmount, 100}, Transport{}, 50, true, []Event{refused(0, Ironium)}, true},
		{"enemy, set waypoint 20 of 50 there", "enemy", Transport{SetWaypoint, 20}, Transport{}, 0, true, []Event{refused(0, Ironium)}, true},
		{"enemy, Fe and Bo both refused: Fe told", "enemy", Transport{Action: LoadAll}, Transport{LoadExactly, 5}, 0, true, []Event{refused(0, Ironium)}, true},
		{"enemy, Fe wants nothing, Bo told", "enemy", Transport{SetAmount, 0}, Transport{LoadExactly, 5}, 0, true, []Event{refused(0, Boranium)}, true},
	} {
		l, _ := sourceLab(t, tt.where, tt.tr, tt.trB, tt.held)
		f := &l.g.Fleets[0]
		if ev := l.g.load(f, false); len(ev) != 0 || (f.Task.Kind == TaskTransport) != tt.waits {
			t.Errorf("%s, before movement: events %+v, task %+v; want none and waiting %v", tt.name, ev, f.Task, tt.waits)
		}
		if f.Task.Kind == TaskTransport && f.Cargo.Minerals != (Minerals{tt.held, 0, 0}) {
			t.Errorf("%s: loaded %v", tt.name, f.Cargo.Minerals)
		}
		l, _ = sourceLab(t, tt.where, tt.tr, tt.trB, tt.held)
		f = &l.g.Fleets[0]
		if ev := l.g.load(f, true); !reflect.DeepEqual(ev, tt.after) || f.Task.Kind != TaskNone {
			t.Errorf("%s, after movement: events %+v, task %+v; want %+v and none", tt.name, ev, f.Task, tt.after)
		}
		if l.g.Planets[0].Surface[Ironium] != 50 || f.Cargo.Minerals != (Minerals{tt.held, 0, 0}) {
			t.Errorf("%s: planet %v, fleet %v; nothing loads", tt.name, l.g.Planets[0].Surface, f.Cargo.Minerals)
		}
	}
}

// Another player's planet is a source for a fleet that can steal cargo
// from planets (KERNEL.md "Loads with no usable source"; COMPONENTS.md
// steals_cargo "fleets_and_planets", the Robber Baron Scanner). Stealing
// is not modelled; ASSUMPTION T10: its loads there wait, after movement
// too, with no message.
func TestStealerLoadWaits(t *testing.T) {
	l, _ := sourceLab(t, "enemy", Transport{Action: LoadAll}, Transport{}, 0)
	d := &l.g.Designs[l.g.Fleets[0].Stacks[0].Design]
	d.Slots = append(d.Slots, Slot{Part: Part{Name: "Robber Baron Scanner", DetailedPlanetScan: true}, Count: 1})
	f := &l.g.Fleets[0]
	if ev := l.g.load(f, true); len(ev) != 0 || f.Task.Transport[Ironium] != (Transport{Action: LoadAll}) {
		t.Errorf("events %+v, task %+v; want none and load all kept", ev, f.Task)
	}
	l, _ = sourceLab(t, "enemy", Transport{Action: LoadAll}, Transport{}, 0)
	d = &l.g.Designs[l.g.Fleets[0].Stacks[0].Design]
	d.Slots = append(d.Slots, Slot{Part: Part{Name: "Robber Baron Scanner", DetailedPlanetScan: true}, Count: 0})
	if ev := l.g.load(&l.g.Fleets[0], true); len(ev) != 1 {
		t.Errorf("an empty scanner slot steals nothing: events %+v, want one refusal", ev)
	}
}

// At a source, a "set amount to" the target cannot meet waits and, after
// movement, is told so each year (KERNEL.md "Which loads are unmet";
// MESSAGES.md 0x121, 0x122 for colonists, BINARY-ONLY). One held back by
// the fleet's hold, not the target, is released instead (T14).
func TestSetAmountWaitingMessage(t *testing.T) {
	for _, tt := range []struct {
		name  string
		tr    Transport
		after []Event
	}{
		{"100 from 50", Transport{SetAmount, 100}, []Event{{Kind: EventLoadWaiting, Player: 0, Planet: 0, Fleet: 1, Count: 100, Axes: []int{Ironium}}}},
		{"50 from 50", Transport{SetAmount, 50}, nil},
	} {
		for _, after := range []bool{false, true} {
			l, _ := sourceLab(t, "own", tt.tr, Transport{}, 0)
			f := &l.g.Fleets[0]
			ev := l.g.loadPass(after)
			want := tt.after
			if !after {
				want = nil
			}
			if !reflect.DeepEqual(ev, want) || f.Cargo.Minerals[Ironium] != 50 {
				t.Errorf("%s, after %v: events %+v, holds %d; want %+v and 50", tt.name, after, ev, f.Cargo.Minerals[Ironium], want)
			}
		}
	}
	// Held back by the hold: 300 wanted, 1000 there, 210 kT of space. The
	// full hold releases the action, as for "wait for" (ASSUMPTION T14),
	// with no message, before or after movement.
	for _, after := range []bool{false, true} {
		l, pi := sourceLab(t, "own", Transport{SetAmount, 300}, Transport{}, 0)
		l.g.Planets[pi].Surface[Ironium] = 1000
		f := &l.g.Fleets[0]
		if ev := l.g.load(f, after); len(ev) != 0 || f.Cargo.Minerals[Ironium] != 210 || f.Task.Kind != TaskNone {
			t.Errorf("hold full, after %v: events %+v, holds %d, task %+v; want none, 210 and released", after, ev, f.Cargo.Minerals[Ironium], f.Task)
		}
	}
	// Colonists at salvage, which holds none (ASSUMPTION T6).
	task := Task{Kind: TaskTransport}
	task.Transport[CargoColonists] = Transport{SetAmount, 20}
	sl := salvageLab(t, Salvage{Minerals: Minerals{50, 0, 0}, Steps: 10}, task)
	sl.g.Fleets[0].Cargo.Colonists = 10
	want := []Event{{Kind: EventLoadWaiting, Player: 0, Planet: -1, Fleet: 1, Count: 20, Axes: []int{CargoColonists}}}
	if ev := sl.g.load(&sl.g.Fleets[0], true); !reflect.DeepEqual(ev, want) {
		t.Errorf("colonists at salvage: events %+v, want %+v", ev, want)
	}
	// A colonist "set waypoint to 0" there is met: the salvage holds none.
	task.Transport[CargoColonists] = Transport{SetWaypoint, 0}
	sl = salvageLab(t, Salvage{Minerals: Minerals{50, 0, 0}, Steps: 10}, task)
	if ev := sl.g.load(&sl.g.Fleets[0], true); len(ev) != 0 || sl.g.Fleets[0].Task.Kind != TaskNone {
		t.Errorf("colonist set waypoint 0 at salvage: events %+v, task %+v; want none", ev, sl.g.Fleets[0].Task)
	}
}
