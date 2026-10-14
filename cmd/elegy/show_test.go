package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// showReport is a hand-made report that reaches every branch of show.
func showReport() game.Report {
	race := engine.Race{Env: [3]engine.EnvRange{{Center: 50, Low: 15, High: 85}, {Center: 50, Low: 15, High: 85}, {Center: 50, Low: 15, High: 85}}}
	scout := engine.Design{Name: "Scout", Mass: 25, Hull: engine.Hull{Name: "Scout"}, Engines: 1, Engine: engine.Engine{Name: "Long Hump 6"}, FuelCapacity: 300}
	hauler := engine.Design{Name: "Hauler", Mass: 130, Hull: engine.Hull{Name: "Medium Freighter"}, CargoCapacity: 210}
	other := engine.Design{Name: "Spy", Mass: 20}
	return game.Report{
		GameID: 9, Year: 2405, Player: 1,
		Rules: engine.Ruleset{ID: "elegy", Version: 6},
		Self: engine.Player{Race: race, ResearchBudget: 15, Plans: []engine.BattlePlan{
			{Name: "Default", Tactic: engine.TacticMaximizeDamage, Primary: engine.TargetArmed, Secondary: engine.TargetAny, Attack: engine.AttackEnemies},
			{Name: "Hunt", Tactic: engine.TacticDisengage, Primary: engine.TargetFreighters, Attack: engine.AttackPlayer, Player: 2, DumpCargo: true}}, HasHomeworld: true, Homeworld: 0, Dead: true,
			Research: engine.ResearchState{Levels: [engine.NumFields]int{3, 9, 4, 3, 3, 10}, Accumulated: [engine.NumFields]int{7, 0, 0, 0, 0, 5}, Current: engine.Biotech, Next: engine.NextLowestField, MaxLevel: 10}},
		Planets: []engine.Planet{
			{ID: 0, Pos: engine.Point{X: 1000, Y: 1000}, Homeworld: true, Env: [3]int{50, 50, 50}, Population: 250, Mines: 10, Factories: 12, Defenses: 3,
				HasStarbase: true, StarbaseDesign: 2, Surface: engine.Minerals{1, 2, 3},
				Deposits: [engine.NumMinerals]engine.Deposit{{Concentration: 40}, {Concentration: 50}, {Concentration: 60}},
				Queue:    []engine.QueueItem{{Kind: engine.ItemFactory, Count: 5, Percent: 30}, {Kind: engine.ItemShip, Count: 1, Slot: 1}, {Kind: engine.ItemStarbase, Count: 1, Slot: 0}}},
			{ID: 1, Pos: engine.Point{X: 1030, Y: 1040}, Env: [3]int{0, 50, 50}, Population: 5, LeftoverOnly: true, HasRoute: true, RouteTo: 0},
		},
		Fleets: []engine.Fleet{
			{ID: 0, Number: 1, Pos: engine.Point{X: 1000, Y: 1000}, Stacks: []engine.Stack{{Design: 1, Count: 2}, {Design: 4, Count: 1}}, Fuel: 90,
				Cargo: engine.Cargo{Minerals: engine.Minerals{4, 5, 6}, Colonists: 7}, Task: engine.Task{Kind: engine.TaskTransport},
				Waypoints: []engine.Waypoint{{Pos: engine.Point{X: 1030, Y: 1040}, Target: engine.TargetPlanet, ID: 1, Warp: 5, Task: engine.Task{Kind: engine.TaskColonize}},
					{Pos: engine.Point{X: 1060, Y: 1080}, Target: engine.TargetFleet, ID: 9, Warp: 6}, {Pos: engine.Point{X: 1060, Y: 1081}, Warp: 7}}, Repeat: true},
			{ID: 5, Number: 2, Name: "Home Guard", Plan: 1, Pos: engine.Point{X: 1001, Y: 1000}, Stacks: []engine.Stack{{Design: 1, Count: 1}}},
			{ID: 6, Number: 3, Pos: engine.Point{X: 1001, Y: 1000}},
			{ID: 7, Number: 4, Pos: engine.Point{X: 1001, Y: 1000}, Stacks: []engine.Stack{{Design: 8, Count: 1}}},
		},
		Designs: []game.OwnDesign{
			{Index: 1, Slot: engine.DesignSlot{Slot: 0}, Design: scout},
			{Index: 2, Slot: engine.DesignSlot{Slot: 0, Starbase: true}, Design: engine.Design{Name: "Fort", Hull: engine.Hull{Name: "Orbital Fort"}}},
			{Index: 4, Slot: engine.DesignSlot{Slot: 1}, Design: hauler},
		},
		View: engine.PlayerView{Fleets: []engine.FleetSighting{{Owner: 0, Pos: engine.Point{X: 1030, Y: 1040}, Stacks: []engine.Stack{{Design: 3, Count: 4}, {Design: 5, Count: 1}}}}},
		Events: []engine.Event{
			{Kind: engine.EventBuilt, Planet: 0, Fleet: -1, Item: engine.ItemFactory, Count: 3},
			{Kind: engine.EventShipsBuilt, Planet: 0, Fleet: 0, Count: 1},
			{Kind: engine.EventLevelUp, Planet: -1, Fleet: -1},
			{Kind: engine.EventKind(999), Planet: 7, Fleet: 2},
		},
		Orders: []game.OrderOutcome{{Index: 0}, {Index: 1, Message: "fleet 9: no such object"}, {Index: -1, Message: "late"}},
		History: []game.PlanetRecord{
			{Year: 2405, Report: engine.PlanetReport{Planet: 0, Level: engine.ReportOwn, Pos: engine.Point{X: 1000, Y: 1000}}},
			{Year: 2403, Report: engine.PlanetReport{Planet: 2, Level: engine.ReportNormal, Pos: engine.Point{X: 1300, Y: 1400}, Owner: engine.NoOwner, Env: [3]int{50, 50, 90}, Concentrations: [engine.NumMinerals]int{1, 2, 3}}},
			{Year: 2404, Report: engine.PlanetReport{Planet: 3, Level: engine.ReportPosition, Pos: engine.Point{X: 1003, Y: 1004}, Owner: 0}},
		},
		Universe: []game.UniversePlanet{
			{ID: 0, Pos: engine.Point{X: 1000, Y: 1000}, NameIndex: 10},
			{ID: 1, Pos: engine.Point{X: 1030, Y: 1040}, NameIndex: 11},
			{ID: 2, Pos: engine.Point{X: 1300, Y: 1400}, NameIndex: 12},
			{ID: 3, Pos: engine.Point{X: 1003, Y: 1004}, NameIndex: 13},
			{ID: 4, Pos: engine.Point{X: 1100, Y: 1000}, NameIndex: 14},
		},
		KnownDesigns: []game.KnownDesign{
			{Index: 3, Year: 2404, Hull: "Scout", Mass: 20, Full: true, Design: &other},
			{Index: 5, Year: 2402, Hull: "Destroyer", Mass: 180},
		},
	}
}

const showWant = `Game 9, year 2405, player 1, ruleset elegy v6
This player has been eliminated.
Planets: 2, population 25500
Homeworld: Planet 11 (#0)
Fleets: 4, ships 5
Research: 15% of resources to biotechnology (level 10), then lowest field
Messages: 4; last year's orders: 3, 2 rejected

Planets (2)
Planet 11 (#0) at (1000, 1000), homeworld
  population 25000, mines 10, factories 12, defenses 3
  starbase Fort
  surface minerals: ironium 1, boranium 2, germanium 3 kT
  concentrations: 40, 50, 60
  environment 50/50/50, habitability 100%
  queue:
    1. factories ×5 (30% done)
    2. ship Hauler ×1
    3. starbase Fort ×1
Planet 12 (#1) at (1030, 1040)
  population 500, mines 0, factories 0, defenses 0
  surface minerals: ironium 0, boranium 0, germanium 0 kT
  concentrations: 0, 0, 0
  environment 0/50/50, habitability -15%
  queue: empty
  only leftover resources go to research
  route to Planet 11 (#0)

Other planets (3)
Planet 14 (#3) at (1003, 1004), 5.0 ly, seen 2404, owned by player 0
Planet 15 (#4) at (1100, 1000), 100.0 ly, never seen
Planet 13 (#2) at (1300, 1400), 500.0 ly, seen 2403, unowned, habitability -5%, concentrations 1/2/3
Other players' fleets seen (1)
player 0 fleet at Planet 12 (#1): 4 Spy, 1 unknown Destroyer

Fleets (4)
#0 Scout #1, at Planet 11 (#0)
  ships: 2 Scout, 1 Hauler
  fuel 90 mg; cargo ironium 4, boranium 5, germanium 6 kT, colonists 700
  task here: transport
  waypoint 1: Planet 12 (#1), 50.0 ly, warp 5, then colonize
  waypoint 2: fleet #9, 50.0 ly, warp 6
  waypoint 3: (1060, 1081), 1.0 ly, warp 7
  repeats its orders
#5 Home Guard, in space at (1001, 1000)
  ships: 1 Scout
  fuel 0 mg; cargo ironium 0, boranium 0, germanium 0 kT, colonists 0
  battle plan 1
#6 Fleet #3, in space at (1001, 1000)
  ships: none
  fuel 0 mg; cargo ironium 0, boranium 0, germanium 0 kT, colonists 0
#7 design #8 #4, in space at (1001, 1000)
  ships: 1 design #8
  fuel 0 mg; cargo ironium 0, boranium 0, germanium 0 kT, colonists 0

Designs (3)
ship slot 0: Scout (Scout hull), 25 kT, 1 Long Hump 6, fuel 300 mg
starbase slot 0: Fort (Orbital Fort hull), 0 kT
ship slot 1: Hauler (Medium Freighter hull), 130 kT, cargo 210 kT
Other players' designs known (2)
design #3: Spy, Scout hull, 20 kT, seen 2404
design #5: unknown Destroyer, Destroyer hull, 180 kT, seen 2402

Battle plans (2)
0 Default: maximize-damage; primary armed, secondary any; attack enemies
1 Hunt: disengage; primary freighters, secondary none; attack player 2; dumps cargo

Research: 15% of resources, current field biotechnology, then lowest field
  energy        level  3, 14 of 1060 toward level 4
  weapons       level  9, 0 of 8180 toward level 10
  propulsion    level  4, 0 of 1320 toward level 5
  construction  level  3, 0 of 1060 toward level 4
  electronics   level  3, 0 of 1060 toward level 4
  biotechnology level 10  <- researching

Messages for 2404 (4)
built factories: Planet 11 (#0), count 3
ships built: Planet 11 (#0), fleet #0, count 1
level up
event 999: planet #7, fleet #2
order 1 rejected: fleet 9: no such object
order file refused: late
`

// TestShow prints every section of a hand-made report under slower tech.
func TestShow(t *testing.T) {
	var out bytes.Buffer
	if err := show(&out, newReportView(showReport(), true), showSections); err != nil {
		t.Fatal(err)
	}
	if out.String() != showWant {
		got, want := strings.Split(out.String(), "\n"), strings.Split(showWant, "\n")
		for i := range min(len(got), len(want)) {
			if got[i] != want[i] {
				t.Errorf("line %d:\n got %q\nwant %q", i+1, got[i], want[i])
			}
		}
		t.Fatalf("show printed %d lines, want %d:\n%s", len(got), len(want), out.String())
	}
}

// TestShowResearchNormal checks research without slower tech, with no
// level cap and the next level's cost from the race's cost setting.
func TestShowResearchNormal(t *testing.T) {
	r := showReport()
	r.Self.Research.MaxLevel = 0
	r.Self.Research.Levels[engine.Weapons] = engine.MaxTechLevel
	r.Self.Race.ResearchCosts[engine.Energy] = engine.ResearchExpensive
	var out bytes.Buffer
	if err := show(&out, newReportView(r, false), []string{"research"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"  energy        level  3, 7 of 1225 toward level 4\n",
		"  weapons       level 26\n",
		"  biotechnology level 10, 5 of 6590 toward level 11  <- researching\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("research lacks %q:\n%s", want, out.String())
		}
	}
}

// TestShowCommand checks show's sections from the command line.
func TestShowCommand(t *testing.T) {
	save := filepath.Join(t.TempDir(), "game.json")
	cli(t, "new", "-seed", "5", "-players", "2", "-size", "tiny", "-o", save)
	sum := cli(t, "show", "-game", save, "-player", "1")
	if !strings.HasPrefix(sum, "Game ") || !strings.Contains(sum, "player 1,") || strings.Contains(sum, "Planets (") {
		t.Fatalf("default section:\n%s", sum)
	}
	all := cli(t, "show", "-game", save, "all")
	parts := cli(t, "show", "-game", save, "summary", "planets", "galaxy", "fleets", "designs", "plans", "research", "messages")
	if all != parts || !strings.Contains(all, "\n\nPlanets (1)\n") {
		t.Fatalf("all:\n%s\nsections:\n%s", all, parts)
	}
	for _, want := range []string{"  parts: 32 Laser, 32 Mole-skin Shield\n", "  parts: 1 Long Hump 6, 1 Rhino Scanner, 1 X-Ray Laser\n", "  slots: 0:1:Long Hump 6, 1:1:Rhino Scanner, 2:1:X-Ray Laser\n", "Battle plans (5)\n0 Default: maximize-ratio; "} {
		if !strings.Contains(all, want) {
			t.Errorf("show lacks %q", want)
		}
	}
	if got := cli(t, "show", "-game", save, "research"); !strings.Contains(got, "0 of 390 toward level 4") {
		t.Fatalf("research:\n%s", got)
	}
	// show reads the game's slower-tech setting.
	g, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	g.State.SlowerTech = true
	if err := writeGame(save, g); err != nil {
		t.Fatal(err)
	}
	if got := cli(t, "show", "-game", save, "research"); !strings.Contains(got, "0 of 780 toward level 4") {
		t.Fatalf("research under slower tech:\n%s", got)
	}
	if got := cli(t, "show", "-game", save, "FLEETS"); !strings.HasPrefix(got, "Fleets (") {
		t.Fatalf("FLEETS:\n%s", got)
	}
	for _, args := range [][]string{
		{"show", "-game", save, "planet"},
		{"show", "-game", save, "all", "planets"},
		{"show", "-game", save, "-player", "2"},
		{"show", "-game", filepath.Join(t.TempDir(), "missing.json")},
		{"show", "-bogus"},
	} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Errorf("elegy %v: no error", args)
		}
	}
	if err := show(&bytes.Buffer{}, reportView{}, []string{"nothing"}); err == nil {
		t.Error("show of an unknown section")
	}
}

// TestShowNoHomeworld checks a player without a homeworld: other planets
// in universe order with no distances.
func TestShowNoHomeworld(t *testing.T) {
	r := showReport()
	r.Self.HasHomeworld = false
	r.Self.Dead = false
	var out bytes.Buffer
	r.KnownDesigns = r.KnownDesigns[1:]
	r.Designs = nil
	if err := show(&out, newReportView(r, false), []string{"summary", "galaxy", "designs"}); err != nil {
		t.Fatal(err)
	}
	want := `Game 9, year 2405, player 1, ruleset elegy v6
Planets: 2, population 25500
Fleets: 4, ships 5
Research: 15% of resources to biotechnology (level 10), then lowest field
Messages: 4; last year's orders: 3, 2 rejected

Other planets (3)
Planet 13 (#2) at (1300, 1400), seen 2403, unowned, habitability -5%, concentrations 1/2/3
Planet 14 (#3) at (1003, 1004), seen 2404, owned by player 0
Planet 15 (#4) at (1100, 1000), never seen
Other players' fleets seen (1)
player 0 fleet at Planet 12 (#1): 4 design #3, 1 unknown Destroyer

Designs (0)
Other players' designs known (1)
design #5: unknown Destroyer, Destroyer hull, 180 kT, seen 2402
`
	if out.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out.String(), want)
	}
	// A homeworld missing from the universe has no position.
	r.Self.HasHomeworld, r.Self.Homeworld = true, 99
	if _, ok := newReportView(r, false).homeworld(); ok {
		t.Fatal("homeworld 99 found")
	}
}

// TestShowDesignNames names designs: the player's own, another player's
// known in full or only by hull, and one not known at all.
func TestShowDesignNames(t *testing.T) {
	spy := engine.Design{Name: "Spy"}
	v := newReportView(game.Report{
		Designs:      []game.OwnDesign{{Index: 1, Design: engine.Design{Name: "Mine"}}},
		KnownDesigns: []game.KnownDesign{{Index: 3, Hull: "Scout", Full: true, Design: &spy}},
	}, false)
	for i, want := range map[int]string{1: "Mine", 3: "Spy", 4: "design #4"} {
		if got := v.designName(i); got != want {
			t.Errorf("design %d: %q, want %q", i, got, want)
		}
	}
}
