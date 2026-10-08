package engine

import (
	"errors"
	"testing"
)

// salvageOrder is a cargo order of fleet with player 1's salvage object 0.
func salvageOrder(fleet int, amounts ...int) CargoOrder {
	o := CargoOrder{Fleet: fleet, Target: TargetSalvage, Owner: 1, ID: 0}
	copy(o.Amounts[:], amounts)
	return o
}

// A manual load from salvage (OBJECTS.md "Salvage", "Loading"): minerals
// only, whoever owns the object, capped by what it holds and by the
// hold; players' loads are taken in replay order, and a load the object
// could not fill is told so (MESSAGES.md 0x0db, 0x0dc; ASSUMPTION L30).
func TestSalvageCargoOrderLoad(t *testing.T) {
	l := salvageLab(t, Salvage{Minerals: Minerals{120, 500, 40}, Steps: 66}, Task{})
	f := l.g.Fleets[0]
	f.ID, f.Owner, f.Stacks = 2, 1, append([]Stack(nil), f.Stacks...)
	l.g.Fleets = append(l.g.Fleets, f)
	// Player 0's object 0 elsewhere: the order names player 1's.
	l.g.Salvage = append(l.g.Salvage, Salvage{Pos: Point{900, 900}, Owner: 0, Number: 0, Minerals: Minerals{5, 5, 5}})
	a := ApplyOrders(&l.g, []PlayerOrders{
		{Player: 0, Orders: []Order{salvageOrder(1, 50, 100, 10)}},
		{Player: 1, Orders: []Order{salvageOrder(2, 100, 0, 40)}},
	}, []int{1, 0})
	for _, r := range a.Results {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
	}
	// Player 1 first: 100 Fe and all 40 Ge. Player 0 then gets the 20 Fe
	// left, its 100 Bo and no Ge.
	if got := l.g.Fleets[1].Cargo.Minerals; got != (Minerals{100, 0, 40}) {
		t.Errorf("player 1's fleet %v, want [100 0 40]", got)
	}
	if got := l.g.Fleets[0].Cargo.Minerals; got != (Minerals{20, 100, 0}) {
		t.Errorf("player 0's fleet %v, want [20 100 0]", got)
	}
	if got := l.g.Salvage[0].Minerals; got != (Minerals{0, 400, 0}) {
		t.Errorf("salvage %v, want [0 400 0]", got)
	}
	want := []Event{
		{Kind: EventSalvageEmptiedFirst, Player: 0, Planet: -1, Fleet: 1, Count: 20, Axes: []int{Ironium}},
		{Kind: EventSalvageEmptiedFirst, Player: 0, Planet: -1, Fleet: 1, Count: 0, Axes: []int{Germanium}},
	}
	if len(a.Events) != len(want) {
		t.Fatalf("events %+v, want %+v", a.Events, want)
	}
	for k := range want {
		if e := a.Events[k]; e.Kind != want[k].Kind || e.Player != 0 || e.Fleet != 1 || e.Count != want[k].Count || len(e.Axes) != 1 || e.Axes[0] != want[k].Axes[0] {
			t.Errorf("event %d %+v, want %+v", k, e, want[k])
		}
	}
	// A load the hold limits is not told: 300 Bo asked, 90 kT of room.
	errs, a2 := apply(&l.g, 0, salvageOrder(1, 0, 300))
	if errs[0] != nil || l.g.Fleets[0].Cargo.Minerals[Boranium] != 190 || len(a2.Events) != 0 {
		t.Errorf("hold-limited load: %v, fleet %v, events %+v; want 190 Bo and no event", errs[0], l.g.Fleets[0].Cargo.Minerals, a2.Events)
	}
}

// Minerals given to salvage go in up to its room, the rest staying
// aboard (ASSUMPTION T7): 37 kT in 6 steps leaves 23, so 10 Fe and then
// 13 of 30 go in; germanium the fleet does not carry gives nothing. Colonists or fuel,
// an object elsewhere or none at all reject the order (ASSUMPTION L30,
// L7).
func TestSalvageCargoOrderGiveAndRefusals(t *testing.T) {
	l := salvageLab(t, Salvage{Minerals: Minerals{30, 0, 7}, Steps: 6}, Task{})
	l.g.Fleets[0].Cargo = Cargo{Minerals: Minerals{40, 0, 0}, Colonists: 10}
	l.g.Fleets[0].Fuel = 100
	errs, _ := apply(&l.g, 0,
		salvageOrder(1, -10, 0, -10),
		salvageOrder(1, -30),
		salvageOrder(1, 0, 0, 0, 5),
		salvageOrder(1, 0, 0, 0, -5),
		salvageOrder(1, 0, 0, 0, 0, -10),
	)
	if errs[0] != nil || errs[1] != nil {
		t.Fatal(errs[:2])
	}
	if sv, f := l.g.Salvage[0].Minerals, l.g.Fleets[0].Cargo.Minerals; sv != (Minerals{53, 0, 7}) || f != (Minerals{17, 0, 0}) {
		t.Errorf("salvage %v, fleet %v; want [53 0 7] and [17 0 0]", sv, f)
	}
	for k, err := range errs[2:] {
		if !errors.Is(err, ErrOutOfRange) {
			t.Errorf("order %d: %v, want out of range", k+2, err)
		}
	}
	if l.g.Fleets[0].Cargo.Colonists != 10 || l.g.Fleets[0].Fuel != 100 {
		t.Errorf("fleet %+v fuel %d: a refused order moved cargo", l.g.Fleets[0].Cargo, l.g.Fleets[0].Fuel)
	}
	l.g.Fleets[0].Pos = Point{600, 600}
	if errs, _ := apply(&l.g, 0, salvageOrder(1, 5)); !errors.Is(errs[0], ErrNotTogether) {
		t.Errorf("elsewhere: %v", errs[0])
	}
	l.g.Fleets[0].Pos = l.g.Salvage[0].Pos
	o := salvageOrder(1, 5)
	o.ID = 3
	if errs, _ := apply(&l.g, 0, o); !errors.Is(errs[0], ErrNoSuchObject) {
		t.Errorf("no object 3: %v", errs[0])
	}
	l.g.Objects = nil
	if errs, _ := apply(&l.g, 0, salvageOrder(1, 5)); !errors.Is(errs[0], ErrNoSuchObject) {
		t.Errorf("no space objects: %v", errs[0])
	}
}

// One order giving two minerals: each takes the room the earlier left.
// Room 10: 7 Fe, then 3 of the 7 Ge, the rest aboard (ASSUMPTION T7).
func TestSalvageCargoOrderGiveTwoMinerals(t *testing.T) {
	l := salvageLab(t, Salvage{Steps: 1}, Task{})
	l.g.Fleets[0].Cargo.Minerals = Minerals{7, 0, 7}
	if errs, _ := apply(&l.g, 0, salvageOrder(1, -7, 0, -7)); errs[0] != nil {
		t.Fatal(errs[0])
	}
	if sv, f := l.g.Salvage[0].Minerals, l.g.Fleets[0].Cargo.Minerals; sv != (Minerals{7, 0, 3}) || f != (Minerals{0, 0, 4}) {
		t.Errorf("salvage %v, fleet %v; want [7 0 3] and [0 0 4]", sv, f)
	}
}
