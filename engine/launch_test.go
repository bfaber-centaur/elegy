package engine

import (
	"reflect"
	"testing"
)

func TestIdealWarpPerEngine(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Ideal warp of the fleet": results for one
	// design (CONFIRMED for Long Hump 6 and Quick Jump 5, SL-04..SL-07;
	// BINARY-ONLY for the others).
	want := map[string]int{
		"Settler's Delight": 6, "Quick Jump 5": 5, "Fuel Mizer": 4, "Long Hump 6": 6,
		"Daddy Long Legs 7": 7, "Alpha Drive 8": 8, "Trans-Galactic Drive": 9, "Interspace-10": 10,
		"Enigma Pulsar": 10, "Trans-Star 10": 10, "Radiating Hydro-Ram Scoop": 6,
		"Sub-Galactic Fuel Scoop": 5, "Trans-Galactic Fuel Scoop": 6, "Trans-Galactic Super Scoop": 7,
		"Trans-Galactic Mizer Scoop": 10, "Galaxy Scoop": 10,
	}
	c := Components()
	for engine, w := range want {
		d, err := c.NewDesign(engine, "Scout", []SlotFill{{Slot: 0, Part: engine, Count: 1}})
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{Designs: []Design{d}}
		if got := g.idealWarp(&Fleet{Stacks: []Stack{{Design: 0, Count: 1}}}); got != w {
			t.Errorf("%s: warp %d, want %d", engine, got, w)
		}
	}
}

// slScouts are the SL vector designs: the Scout (Long Hump 6, Rhino
// Scanner, X-Ray Laser, 23 kT, 50 mg) and the QJ5 Scout (Quick Jump 5,
// Bat Scanner, 14 kT, 50 mg).
func slScouts(t *testing.T) (Design, Design) {
	c := Components()
	scout, err := c.NewDesign("Scout", "Scout", []SlotFill{{0, "Long Hump 6", 1}, {1, "Rhino Scanner", 1}, {2, "X-Ray Laser", 1}})
	if err != nil {
		t.Fatal(err)
	}
	qj5, err := c.NewDesign("QJ5 Scout", "Scout", []SlotFill{{0, "Quick Jump 5", 1}, {1, "Bat Scanner", 1}})
	if err != nil {
		t.Fatal(err)
	}
	if scout.Mass != 23 || qj5.Mass != 14 || scout.FuelCapacity != 50 || qj5.FuelCapacity != 50 {
		t.Fatalf("designs %d kT %d mg, %d kT %d mg; want 23/50 and 14/50", scout.Mass, scout.FuelCapacity, qj5.Mass, qj5.FuelCapacity)
	}
	return scout, qj5
}

func TestConfirmedRouteWarp(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Route warp" vectors (CONFIRMED SL-04..SL-07,
	// both streams). The source planet has a Space Station. Rows with
	// gates at both ends are left out: Elegy has no stargates.
	scout, qj5 := slScouts(t)
	c := Components()
	station, err := c.NewDesign("Station", "Space Station", nil)
	if err != nil {
		t.Fatal(err)
	}
	fort, err := c.NewDesign("Fort", "Orbital Fort", nil)
	if err != nil {
		t.Fatal(err)
	}
	const (
		none = iota
		ownStation
		ownFort
		otherStation
	)
	for _, tt := range []struct {
		dest          int
		d             int
		scout, qj5Got int
	}{
		{none, 12, 4, 4}, {none, 17, 5, 5}, {none, 25, 6, 4}, {none, 41, 5, 5}, {none, 86, 6, 5}, {none, 299, 6, 5},
		{ownStation, 72, 7, 9}, {ownStation, 84, 7, 7},
		{ownStation, 94, 7, 7}, // a gate at the destination only: the dock rule applies
		{ownFort, 77, 6, 5}, {otherStation, 77, 6, 5},
		{ownStation, 0, 2, 2}, // the building planet itself
	} {
		for k, want := range []int{tt.scout, tt.qj5Got} {
			g := &Game{
				Players: make([]Player, 2),
				Designs: []Design{scout, qj5, station, fort},
				Planets: []Planet{
					{ID: 1, Owner: 0, Pos: Point{1000, 1000}, HasStarbase: true, StarbaseDesign: 2, StarbaseHull: 3},
					{ID: 2, Owner: NoOwner, Pos: Point{1000 + tt.d, 1000}},
				},
			}
			dst := 1
			if tt.d == 0 {
				dst = 0
			}
			p := &g.Planets[1]
			switch tt.dest {
			case ownStation:
				p.Owner, p.HasStarbase, p.StarbaseDesign, p.StarbaseHull = 0, true, 2, 3
			case ownFort:
				p.Owner, p.HasStarbase, p.StarbaseDesign, p.StarbaseHull = 0, true, 3, 1
			case otherStation:
				p.Owner, p.HasStarbase, p.StarbaseDesign, p.StarbaseHull = 1, true, 2, 3
			}
			f := Fleet{Owner: 0, Pos: g.Planets[0].Pos, Stacks: []Stack{{Design: k, Count: 1}}, Fuel: 50}
			if got := g.routeWarp(&f, 0, dst); got != want {
				t.Errorf("%s, destination kind %d at %d ly: warp %d, want %d", g.Designs[k].Name, tt.dest, tt.d, got, want)
			}
		}
	}
}

func TestConfirmedRouteWarpTankScout(t *testing.T) {
	// PRODUCTION-LAUNCH.md, seen in the SL tooling check: two Scouts with
	// a Fuel Tank instead of the laser (25 kT, 600 mg between them) to an unowned
	// planet at 133 ly, warp 6; one Scout at 41 ly, warp 5.
	c := Components()
	tank, err := c.NewDesign("Tank Scout", "Scout", []SlotFill{{0, "Long Hump 6", 1}, {1, "Rhino Scanner", 1}, {2, "Fuel Tank", 1}})
	if err != nil {
		t.Fatal(err)
	}
	scout, _ := slScouts(t)
	g := &Game{Players: make([]Player, 1), Designs: []Design{tank, scout},
		Planets: []Planet{{ID: 1, Owner: 0, Pos: Point{1000, 1000}, HasStarbase: true, StarbaseHull: 3}, {ID: 2, Owner: NoOwner, Pos: Point{1133, 1000}}, {ID: 3, Owner: NoOwner, Pos: Point{1000, 1041}}}}
	f := Fleet{Owner: 0, Pos: Point{1000, 1000}, Stacks: []Stack{{Design: 0, Count: 2}}}
	f.Fuel = g.tankCapacity(&f)
	if tank.Mass != 25 || f.Fuel != 600 {
		t.Fatalf("tank scout %d kT, fleet %d mg", tank.Mass, f.Fuel)
	}
	if got := g.routeWarp(&f, 0, 1); got != 6 {
		t.Errorf("two tank scouts at 133 ly: warp %d, want 6", got)
	}
	one := Fleet{Owner: 0, Pos: Point{1000, 1000}, Stacks: []Stack{{Design: 1, Count: 1}}, Fuel: 50}
	if got := g.routeWarp(&one, 0, 2); got != 5 {
		t.Errorf("scout at 41 ly: warp %d, want 5", got)
	}
}

// launchGame: player 0's planet 1 with a Space Station, routed to its
// planet 2 at 72 ly with a Station; design 0 is the Scout.
func launchGame(t *testing.T) *Game {
	scout, qj5 := slScouts(t)
	station, err := Components().NewDesign("Station", "Space Station", nil)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		Players: make([]Player, 2),
		Designs: []Design{scout, qj5, station},
		Planets: []Planet{
			{ID: 1, Owner: 0, Pos: Point{1000, 1000}, HasStarbase: true, StarbaseDesign: 2, StarbaseHull: 3},
			{ID: 2, Owner: 0, Pos: Point{1072, 1000}, HasStarbase: true, StarbaseDesign: 2, StarbaseHull: 3},
		},
	}
	for f := range NumFields {
		g.Players[0].Research.Levels[f] = 26
	}
	return g
}

func TestConfirmedLaunchNewFleets(t *testing.T) {
	// PRODUCTION-LAUNCH.md "The new fleet" (CONFIRMED SL-01, SL-02): one
	// fleet per build event, the lowest unused number (with #1, #2 and #4
	// the next is #3, then #5), full tanks, no cargo, no task, plan 0, no
	// name.
	g := launchGame(t)
	// Player 1's fleets do not take player 0's numbers.
	g.Fleets = []Fleet{{ID: 1, Number: 1, Owner: 0}, {ID: 2, Number: 2, Owner: 0}, {ID: 3, Number: 4, Owner: 0}, {ID: 4, Number: 3, Owner: 1}}
	var numbers []int
	for _, n := range []int{2, 1} {
		_, id := g.Launch(0, 0, n)
		numbers = append(numbers, g.Fleets[g.fleetIndex(id)].Number)
	}
	if numbers[0] != 3 || numbers[1] != 5 {
		t.Fatalf("new fleet numbers %v, want 3 and 5", numbers)
	}
	f := g.Fleets[len(g.Fleets)-2]
	if f.Pos != g.Planets[0].Pos || f.Fuel != 100 || f.Cargo != (Cargo{}) || f.Plan != 0 || f.Name != "" ||
		f.Task.Kind != TaskNone || len(f.Waypoints) != 0 || f.Stacks[0] != (Stack{Design: 0, Count: 2}) {
		t.Errorf("new fleet %+v", f)
	}
}

func TestConfirmedLaunchRouted(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Routing" (CONFIRMED SL-04..SL-07): a planet
	// with a route destination gives each new fleet a waypoint there with
	// the route task; the route setting is unchanged. Own Station at 72
	// ly: Scout warp 7.
	g := launchGame(t)
	g.Planets[0].HasRoute, g.Planets[0].RouteTo = true, 2
	ev, id := g.Launch(0, 0, 1)
	f := g.Fleets[g.fleetIndex(id)]
	want := Waypoint{Pos: Point{1072, 1000}, Warp: 7, Target: TargetPlanet, ID: 2, Task: Task{Kind: TaskRoute}}
	if len(f.Waypoints) != 1 || f.Waypoints[0] != want || !g.Planets[0].HasRoute || g.Planets[0].RouteTo != 2 {
		t.Errorf("waypoints %+v, want %+v", f.Waypoints, want)
	}
	if len(ev) != 2 || ev[1].Kind != EventFleetRouted || ev[1].Count != 7 {
		t.Errorf("events %+v", ev)
	}
}

func TestPredictionLaunchNeedsStarbaseAndTech(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Can the planet build it" (BINARY-ONLY): no
	// starbase, nothing built; no tech, nothing built and the plans are
	// lost.
	g := launchGame(t)
	g.Players[0].Research.Levels = [NumFields]int{}
	if ev, id := g.Launch(0, 0, 1); id != -1 || len(ev) != 1 || ev[0].Kind != EventPlansLost {
		t.Errorf("no tech: %+v %d", ev, id)
	}
	g = launchGame(t)
	g.Planets[0].HasStarbase, g.Planets[0].StarbaseHull = false, 0
	if ev, id := g.Launch(0, 0, 1); id != -1 || len(ev) != 0 || len(g.Fleets) != 0 {
		t.Errorf("no starbase: %+v %d", ev, id)
	}
}

func TestConfirmedFleetLimit(t *testing.T) {
	// PRODUCTION-LAUNCH.md "The 512-fleet limit" (CONFIRMED SL-08..SL-10):
	// with 511 fleets and two items the first makes the 512th fleet and
	// the second joins it; a stack at 32765 is passed over; with no fleet
	// that qualifies the ships are lost.
	g := launchGame(t)
	for i := range maxFleets - 1 {
		g.Fleets = append(g.Fleets, Fleet{ID: i + 1, Number: i + 1, Owner: 0, Pos: Point{1500, 1500}})
	}
	_, first := g.Launch(0, 0, 2)
	ev, second := g.Launch(0, 0, 1)
	if g.Fleets[g.fleetIndex(first)].Number != maxFleets || second != first || ev[0].Kind != EventShipsJoinedFleet {
		t.Fatalf("first %d, second %d, events %+v", first, second, ev)
	}
	if s := g.Fleets[g.fleetIndex(first)].Stacks; len(s) != 1 || s[0].Count != 3 {
		t.Errorf("joined stacks %+v", s)
	}
	g.Fleets[g.fleetIndex(first)].Stacks[0].Count = joinLimit
	if ev, id := g.Launch(0, 0, 1); id != -1 || ev[0].Kind != EventShipsLostFleetLimit || ev[0].Count != 1 {
		t.Errorf("full stack: %+v %d, want the ships lost", ev, id)
	}
}

func TestConfirmedJoinDamage(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Damage of the receiving stack" (CONFIRMED
	// SL-10): 10 Scouts (armor 20) at 50%/100 units plus 1 built → 45%/125;
	// at 50%/300 → 45%/375. An undamaged stack stays undamaged.
	for _, tt := range []struct{ units, want int }{{100, 125}, {300, 375}} {
		if got := joinDamage(Damage{Pct: 50, Units: tt.units}, 10, 1, 20); got != (Damage{Pct: 45, Units: tt.want}) {
			t.Errorf("50%%/%d: %+v, want 45%%/%d", tt.units, got, tt.want)
		}
	}
	if got := joinDamage(Damage{}, 10, 1, 20); got != (Damage{}) {
		t.Errorf("undamaged: %+v", got)
	}
	scout, _ := slScouts(t)
	if a := designArmor(scout); a != 20 {
		t.Errorf("Scout armor %d, want 20", a)
	}
}

func TestDockAllows(t *testing.T) {
	// ORDERS.md "Production queue (starbase dock)", Elegy's chosen rule:
	// no ship at an Orbital Fort, a Space Dock up to 200 kT of hull, a
	// Space Station anything.
	c := Components()
	var designs []Design
	for _, h := range []string{"Orbital Fort", "Space Dock", "Space Station"} {
		d, err := c.NewDesign(h, h, nil)
		if err != nil {
			t.Fatal(err)
		}
		designs = append(designs, d)
	}
	scout, _ := slScouts(t)
	heavy := scout
	heavy.Hull.Mass = 201
	designs = append(designs, scout, heavy)
	g := &Game{Designs: designs, Planets: []Planet{{HasStarbase: true}}}
	for sb, want := range [][2]bool{{false, false}, {true, false}, {true, true}} {
		g.Planets[0].StarbaseDesign = sb
		for k, w := range want {
			if got := g.DockAllows(0, 3+k); got != w {
				t.Errorf("%s, design %d: %v, want %v", designs[sb].Name, 3+k, got, w)
			}
		}
	}
}

func TestMeasuredStarbaseReplacementCost(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Cost of a replacement" (MEASURED SL-12): a JOAT
	// + ISB Space Dock replaced by a Space Station design of 92/72/157/364
	// costs 35/29/61 kT and 136 resources (fresh: 37/29/63 and 146). The
	// old design's cost is not given; 10/0/10/50 is one the measurement
	// allows.
	isb := Race{LRT: LRTs{ImprovedStarbases: true}}
	c := Cost{Resources: 364, Minerals: Minerals{92, 72, 157}}
	got := starbaseCharge(replacementBase(c, Cost{Resources: 50, Minerals: Minerals{10, 0, 10}}), isb)
	if got != (Cost{Resources: 136, Minerals: Minerals{35, 29, 61}}) {
		t.Errorf("replacement %+v, want 35/29/61 and 136", got)
	}
	// The fresh resources are 146 (the spec's 149 was a typo, corrected on
	// PRODUCTION-LAUNCH.md).
	if fresh := starbaseCharge(c, isb); fresh != (Cost{Resources: 146, Minerals: Minerals{37, 29, 63}}) {
		t.Errorf("fresh %+v", fresh)
	}
}

func TestConfirmedStarbaseKeepsDamage(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Starbases" (CONFIRMED SL-12): the planet's
	// starbase damage units are kept on the new starbase.
	g := launchGame(t)
	dock, err := Components().NewDesign("Dock", "Space Dock", nil)
	if err != nil {
		t.Fatal(err)
	}
	g.Designs = append(g.Designs, dock)
	g.Planets[0].StarbaseDamage = 200
	ev := g.BuildStarbase(0, 3)
	p := g.Planets[0]
	if p.StarbaseDesign != 3 || p.StarbaseHull != 2 || !p.StarbaseDock || p.StarbaseDamage != 200 || len(ev) != 1 || ev[0].Count != 200 {
		t.Errorf("planet %+v, events %+v", p, ev)
	}
}

func TestPredictionJoinTakesSlotPlace(t *testing.T) {
	// PRODUCTION-LAUNCH.md (BINARY-ONLY): at the fleet limit, ships of
	// a design the fleet lacks take their design slot's place.
	g := launchGame(t)
	g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: 0}, {Owner: 0, Slot: 1, Design: 1}, {Owner: 0, Slot: 2, Design: 2}}
	f := Fleet{Owner: 0, Stacks: []Stack{{Design: 0, Count: 1}, {Design: 2, Count: 1}}}
	g.insertStack(&f, Stack{Design: 1, Count: 4})
	if want := []Stack{{Design: 0, Count: 1}, {Design: 1, Count: 4}, {Design: 2, Count: 1}}; !reflect.DeepEqual(f.Stacks, want) {
		t.Errorf("stacks %+v, want %+v", f.Stacks, want)
	}
}
