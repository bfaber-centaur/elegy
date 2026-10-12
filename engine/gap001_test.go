package engine

import (
	"fmt"
	"testing"
)

// GAP-001 sub-cases S (scrap at a starbase), G (Genesis Device) and T
// (fleet gifts). The predictions were pushed in d2df970 before lane F ran
// the case; the run (stars-elegy PARITY.md "GAP-001", 0d63a84) matched
// C's minerals, S1, S2, T1, T2, T3 and G1, and refuted G2: Elegy did not
// miniaturize the Genesis Device. KERNEL-STATUS.md "GAP-001 results" has
// each row.

// gap001Race is player 0's race: Interstellar Traveler with No Ram Scoop
// Engines, Cheap Engines, Only Basic Remote Mining, Low Starting
// Population and Bleeding Edge Technology, tech 26 in every field.
func gap001Game(t *testing.T) Game {
	t.Helper()
	r0 := pgRace()
	r0.PRT = PRTInterstellarTraveler
	r0.LRT = LRTs{NoRamScoopEngines: true, CheapEngines: true, OnlyBasicRemoteMining: true, LowStartingPopulation: true, BleedingEdgeTech: true}
	r1 := pgRace()
	r1.PRT = PRTClaimAdjuster
	g := Game{Rules: ElegyRules(), Year: 2400}
	g.Players = []Player{
		{Race: r0, Relations: []Relation{RelationFriend, RelationFriend}},
		{Race: r1, Relations: []Relation{RelationNeutral, RelationFriend}},
	}
	for f := range NumFields {
		g.Players[0].Research.Levels[f] = 26
		g.Players[1].Research.Levels[f] = 3
	}
	g.Players[1].Research.Levels[Propulsion] = 10
	g.Players[1].Research.Levels[Biotech] = 6
	for i := range g.Players {
		g.Players[i].Research.Current, g.Players[i].Research.Next = Biotech, NextSameField
		g.Players[i].Plans = []BattlePlan{{Attack: AttackEnemies, Tactic: TacticMaximizeDamage}}
	}
	add := func(hull string, fills ...SlotFill) int {
		d, err := Components().NewDesign(hull, hull, fills)
		if err != nil {
			t.Fatal(err)
		}
		g.Designs = append(g.Designs, d)
		return len(g.Designs) - 1
	}
	hauler := add("Small Freighter", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Rhino Scanner", 1})
	fort := add("Orbital Fort")
	g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: hauler}, {Owner: 0, Starbase: true, Slot: 0, Design: fort}}
	planet := func(id, x, y int) Planet {
		return Planet{ID: id, Pos: Point{x, y}, Owner: 0, Population: 10,
			Env: [3]int{50, 50, 50}, OrigEnv: [3]int{50, 50, 50}}
	}
	p21 := planet(21, 1359, 1121)
	p21.HasStarbase, p21.StarbaseDesign = true, fort
	g.Planets = []Planet{p21, planet(18, 1324, 1192)}
	cargo := Cargo{Minerals: Minerals{50, 20, 10}}
	g.Fleets = []Fleet{
		{ID: 1, Number: 11, Owner: 0, Pos: Point{1359, 1121}, Stacks: []Stack{{Design: hauler, Count: 2}}, Cargo: cargo, Task: Task{Kind: TaskScrap}},
		{ID: 2, Number: 12, Owner: 0, Pos: Point{1324, 1192}, Stacks: []Stack{{Design: hauler, Count: 2}}, Cargo: cargo, Task: Task{Kind: TaskScrap}},
	}
	// Player 1's fleets in the file are 0, 1, 2 and 4 (Elegy's numbers
	// are the file's plus one), away from everything here.
	for k, n := range []int{1, 2, 3, 5} {
		g.Fleets = append(g.Fleets, Fleet{ID: 10 + k, Number: n, Owner: 1, Pos: Point{1700 + 10*k, 1700}, Stacks: []Stack{{Design: hauler, Count: 1}}})
	}
	gift := Task{Kind: TaskTransferFleet, Player: 1}
	g.Fleets = append(g.Fleets,
		Fleet{ID: 21, Number: 21, Owner: 0, Pos: Point{1200, 1000}, Stacks: []Stack{{Design: hauler, Count: 1}}, Fuel: 200, Task: gift},
		Fleet{ID: 22, Number: 22, Owner: 0, Pos: Point{1200, 1010}, Stacks: []Stack{{Design: hauler, Count: 1}}, Fuel: 200, Cargo: Cargo{Colonists: 10}, Task: gift},
		Fleet{ID: 23, Number: 23, Owner: 0, Pos: Point{1200, 1020}, Stacks: []Stack{{Design: hauler, Count: 1}}, Fuel: 200, Cargo: Cargo{Minerals: Minerals{30, 0, 0}}, Task: gift},
	)
	return g
}

func TestConfirmedGAP001HaulerCost(t *testing.T) {
	// C, the Hauler's cost for player 0 (COMPONENTS.md "Cost for an
	// owner"; the minerals CONFIRMED by GAP-001, the resources not
	// observable there and still a prediction): at tech 26 with Bleeding Edge Technology every component is
	// 80% off, halves rounded up, at least 1. Hull 20/12/0/17 → 4/2/0/3,
	// Long Hump 6 6/5/0/1 → 1/1/0/1, Rhino Scanner 3/3/0/2 → 1/1/0/1.
	// Cheap Engines changes no cost here.
	g := gap001Game(t)
	c := designCost(g.Designs[0], g.Players[0].Race, g.Players[0].Research.Levels)
	if want := (Cost{Resources: 6, Minerals: Minerals{4, 0, 5}}); c != want {
		t.Errorf("Hauler cost %+v, want %+v", c, want)
	}
}

func TestConfirmedGAP001Scrap(t *testing.T) {
	// S1, planet 21 with a starbase: two Haulers, C = 4/0/5 each, so the
	// fleet's 8/0/10 gives 4C/5 = 6/0/8 plus the 50/20/10 cargo. S2, planet
	// 18 without one: C/3 = 2/0/3 plus the cargo (TAKEOVER.md "Other
	// waypoint tasks", CONFIRMED T-34). Player 0 is at tech 26, so the tech
	// attempt at the starbase gains nothing. CONFIRMED by GAP-001 S1, S2.
	g := gap001Game(t)
	r, err := GenerateTurn(g, nil, &seqRand{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]Minerals{21: {56, 20, 18}, 18: {52, 20, 13}}
	for _, p := range r.Game.Planets {
		if p.Surface != want[p.ID] {
			t.Errorf("planet %d surface %v, want %v", p.ID, p.Surface, want[p.ID])
		}
	}
	if r.Game.Players[0].Research.Levels != g.Players[0].Research.Levels {
		t.Errorf("player 0 tech %v, want unchanged", r.Game.Players[0].Research.Levels)
	}
	scrapped := map[int]bool{}
	for _, e := range r.Events {
		if e.Kind == EventFleetScrapped {
			scrapped[e.Planet] = true
		}
		if e.Kind == EventBattleTech {
			t.Errorf("tech gained: %+v", e)
		}
	}
	if !scrapped[21] || !scrapped[18] {
		t.Errorf("scrap messages at %v, want 21 and 18", scrapped)
	}
}

func TestConfirmedGAP001Gifts(t *testing.T) {
	// T1: the empty Hauler becomes player 1's, at the same place with its
	// 200 mg and no waypoints, numbered with player 1's lowest unused
	// number: file number 3 (CONFIRMED rule, WU-B2). T2: 10 colonists
	// aboard, refused; it stays player 0's (CONFIRMED). T3: 30 kT of
	// ironium aboard: Elegy gives the fleet with its cargo, as file number
	// 5. CONFIRMED by GAP-001 T1–T3.
	g := gap001Game(t)
	r, err := GenerateTurn(g, nil, &seqRand{})
	if err != nil {
		t.Fatal(err)
	}
	type got struct {
		owner, number, fuel int
		pos                 Point
		cargo               Cargo
		waypoints           int
	}
	at := map[Point]got{}
	for _, f := range r.Game.Fleets {
		if f.Pos.X == 1200 {
			at[f.Pos] = got{f.Owner, f.Number, f.Fuel, f.Pos, f.Cargo, len(f.Waypoints)}
		}
	}
	want := map[Point]got{
		{1200, 1000}: {1, 4, 200, Point{1200, 1000}, Cargo{}, 0},
		{1200, 1010}: {0, 22, 200, Point{1200, 1010}, Cargo{Colonists: 10}, 0},
		{1200, 1020}: {1, 6, 200, Point{1200, 1020}, Cargo{Minerals: Minerals{30, 0, 0}}, 0},
	}
	if fmt.Sprint(at) != fmt.Sprint(want) {
		t.Errorf("gift fleets %+v, want %+v", at, want)
	}
	kinds := map[EventKind][]int{}
	for _, e := range r.Events {
		switch e.Kind {
		case EventFleetGiven, EventFleetReceived, EventFleetGiftRefused:
			kinds[e.Kind] = append(kinds[e.Kind], e.Player)
		}
	}
	if fmt.Sprint(kinds[EventFleetGiven]) != "[0 0]" || fmt.Sprint(kinds[EventFleetReceived]) != "[1 1]" || fmt.Sprint(kinds[EventFleetGiftRefused]) != "[0]" {
		t.Errorf("gift messages %v, want two given to player 0, two received by player 1, one refusal to player 0", kinds)
	}
}

func TestMeasuredGAP001Genesis(t *testing.T) {
	// G: the Genesis Device's base cost is 5,000 resources, miniaturized
	// like a ship part (COMPONENTS.md "Cost for an owner"): player 0 has
	// BET and m = 26 − 20 = 6, so d = 30% and 3,500 (the CS-001 readout for
	// BET at tech 26). GAP-001 G2–G4 bound it to 3,445–3,567 (MEASURED).
	// The queue arithmetic is KERNEL.md "Production" (PQ-001). G1: 99% done
	// leaves 3500 − trunc(3500·99/100) = 35 to pay, so 60 resources
	// complete it (CONFIRMED). G2–G4: 37, 500 and 999 resources from 0%
	// reach 1%, 14% and 28% (MEASURED). d2df970 predicted 5,000 and 0% for
	// G2; that was refuted.
	c, ok := Components().Lookup("Genesis Device")
	if !ok {
		t.Fatal("no Genesis Device")
	}
	g := gap001Game(t)
	cost := c.OwnerCost(g.Players[0].Race, g.Players[0].Research.Levels)
	if cost != (Cost{Resources: 3500}) {
		t.Fatalf("cost %+v, want 3500 resources", cost)
	}
	if left := cost.Resources - spent(cost, 99).Resources; left != 35 {
		t.Errorf("G1: %d to pay at 99%%, want 35", left)
	}
	for _, c := range []struct{ avail, pct int }{{37, 1}, {500, 14}, {999, 28}} {
		if p := largestPercent(cost.Resources, c.avail); p != c.pct {
			t.Errorf("%d resources from 0%%: %d%%, want %d%%", c.avail, p, c.pct)
		}
	}
}
