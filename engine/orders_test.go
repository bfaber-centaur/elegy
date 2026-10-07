package engine

import (
	"errors"
	"reflect"
	"testing"
)

// ordersGame: two players with the five starting plans (COMBAT.md
// "Starting plans"), player 0's planet 1 at (1100,1100) with fleets 1 and
// 2 in orbit, player 1's planet 2 and fleet 3 at (1200,1200), and an
// unowned planet 3 at (1300,1300) with player 0's fleet 4 in orbit.
// Design 1 has a 100 kT hold and a 200 mg tank.
func ordersGame() *Game {
	plans := []BattlePlan{
		{Name: "Default", Tactic: TacticMaximizeRatio, Primary: TargetArmed, Secondary: TargetAny, Attack: AttackNeutralsAndEnemies},
		{Name: "Kill Starbase", Tactic: TacticMaximizeRatio, Primary: TargetStarbase, Secondary: TargetArmed, Attack: AttackNeutralsAndEnemies},
		{Name: "Max-Defense", Tactic: TacticMaximizeNet, Primary: TargetArmed, Secondary: TargetBombersFreighters, Attack: AttackNeutralsAndEnemies},
		{Name: "Sniper", Tactic: TacticDisengageIfChallenged, Primary: TargetUnarmed, Attack: AttackNeutralsAndEnemies},
		{Name: "Chicken", Tactic: TacticDisengage, Primary: TargetAny, Attack: AttackNeutralsAndEnemies},
	}
	g := opsGame()
	g.ID, g.Year = 77, 2410
	g.Designs[1].CargoCapacity, g.Designs[1].FuelCapacity = 100, 200
	for i := range g.Players {
		g.Players[i].Plans = append([]BattlePlan(nil), plans...)
	}
	g.Planets = []Planet{
		{ID: 1, Pos: Point{1100, 1100}, Owner: 0, Population: 50, Surface: Minerals{40, 40, 40}},
		{ID: 2, Pos: Point{1200, 1200}, Owner: 1, Population: 80},
		{ID: 3, Pos: Point{1300, 1300}, Owner: NoOwner},
	}
	g.Fleets = []Fleet{
		{ID: 1, Owner: 0, Pos: Point{1100, 1100}, Stacks: []Stack{{Design: 1, Count: 1}}, Fuel: 150, Cargo: Cargo{Minerals: Minerals{30, 0, 0}, Colonists: 10}, Plan: 3},
		{ID: 2, Owner: 0, Pos: Point{1100, 1100}, Stacks: []Stack{{Design: 1, Count: 1}}, Fuel: 20, Plan: 4},
		{ID: 3, Owner: 1, Pos: Point{1200, 1200}, Stacks: []Stack{{Design: 1, Count: 1}}, Fuel: 100},
		{ID: 4, Owner: 0, Pos: Point{1300, 1300}, Stacks: []Stack{{Design: 1, Count: 1}}, Fuel: 100, Cargo: Cargo{Minerals: Minerals{0, 20, 0}, Colonists: 30}},
	}
	return g
}

// apply runs one player's orders and returns the per-order errors.
func apply(g *Game, player int, orders ...Order) ([]error, *Applied) {
	a := ApplyOrders(g, []PlayerOrders{{Player: player, GameID: g.ID, Year: g.Year, Orders: orders}}, []int{player})
	var errs []error
	for _, r := range a.Results {
		errs = append(errs, r.Err)
	}
	return errs, a
}

func TestPredictionFileAcceptance(t *testing.T) {
	// ORDERS.md "Wrong game or wrong year" (BINARY-ONLY): another game,
	// an earlier year and a later year are each refused whole; the
	// player's research stays as it was.
	for _, tt := range []struct {
		game uint64
		year int
		want error
	}{{78, 2410, ErrWrongGame}, {77, 2409, ErrOutOfDate}, {77, 2411, ErrLaterYear}} {
		g := ordersGame()
		a := ApplyOrders(g, []PlayerOrders{{Player: 0, GameID: tt.game, Year: tt.year, Orders: []Order{ResearchOrder{Budget: 50}}}}, []int{0})
		if len(a.Results) != 1 || !errors.Is(a.Results[0].Err, tt.want) || a.Results[0].Index != -1 {
			t.Errorf("game %d year %d: %+v, want %v", tt.game, tt.year, a.Results, tt.want)
		}
		if g.Players[0].ResearchBudget != 0 {
			t.Errorf("game %d year %d: research applied", tt.game, tt.year)
		}
	}
}

func TestPredictionReplayOrder(t *testing.T) {
	// ORDERS.md "Conflicts between players" (BINARY-ONLY): the later
	// player in the replay order wins. With the chosen ownership rule the
	// only shared objects are a player's own, so the conflict shows as
	// each player's orders running in turn; a player with no file keeps
	// their settings (ORDERS.md "A player who submits nothing").
	g := ordersGame()
	g.Players = append(g.Players, Player{ResearchBudget: 33})
	files := []PlayerOrders{
		{Player: 1, GameID: 77, Year: 2410, Orders: []Order{ResearchOrder{Budget: 10, Field: Weapons, Next: NextSameField}}},
		{Player: 0, GameID: 77, Year: 2410, Orders: []Order{ResearchOrder{Budget: 20, Field: Biotech, Next: NextLowestField}}},
		{Player: 0, GameID: 77, Year: 2410, Orders: []Order{ResearchOrder{Budget: 90}}},
	}
	a := ApplyOrders(g, files, []int{1, 2, 0})
	if g.Players[0].ResearchBudget != 20 || g.Players[1].ResearchBudget != 10 || g.Players[2].ResearchBudget != 33 {
		t.Errorf("budgets %d %d %d, want 20 10 33", g.Players[0].ResearchBudget, g.Players[1].ResearchBudget, g.Players[2].ResearchBudget)
	}
	if len(a.Results) != 3 || !errors.Is(a.Results[0].Err, ErrDuplicateFile) {
		t.Errorf("results %+v, want the second file of player 0 refused first", a.Results)
	}
}

func TestPredictionResearchOrder(t *testing.T) {
	// ORDERS.md "Research allocation" (BINARY-ONLY): a budget in 0..100
	// applies verbatim; a budget, field or next-field selector outside
	// its legal set rejects the whole order.
	g := ordersGame()
	errs, _ := apply(g, 0,
		ResearchOrder{Budget: 100, Field: Propulsion, Next: Biotech},
		ResearchOrder{Budget: 101, Field: Energy, Next: NextSameField},
		ResearchOrder{Budget: 10, Field: NumFields, Next: NextSameField},
		ResearchOrder{Budget: 10, Field: Energy, Next: -3},
		ResearchOrder{Budget: -1, Field: Energy, Next: NextSameField},
	)
	if errs[0] != nil {
		t.Fatal(errs[0])
	}
	for i, err := range errs[1:] {
		if !errors.Is(err, ErrOutOfRange) {
			t.Errorf("order %d: %v, want out of range", i+1, err)
		}
	}
	r := g.Players[0]
	if r.ResearchBudget != 100 || r.Research.Current != Propulsion || r.Research.Next != Biotech {
		t.Errorf("research %d %+v, want the first order", r.ResearchBudget, r.Research)
	}
}

func TestConfirmedDeletePlanRenumbers(t *testing.T) {
	// COMBAT.md "Adding, replacing and deleting" (CONFIRMED BP-1): with
	// plans 0..6 and fleets on 3, 5, 2 and 6, deleting plan 3 left six
	// plans and the fleets on 2, 4, 2 and 5, the later plans keeping
	// their fields.
	g := ordersGame()
	pl := &g.Players[0]
	pl.Plans = append(pl.Plans, BattlePlan{Name: "Five", Tactic: TacticMaximizeDamage}, BattlePlan{Name: "Six", Attack: AttackEveryone})
	g.Fleets = []Fleet{{ID: 1, Owner: 0, Plan: 3}, {ID: 2, Owner: 0, Plan: 5}, {ID: 3, Owner: 0, Plan: 2}, {ID: 4, Owner: 0, Plan: 6}, {ID: 5, Owner: 1, Plan: 4}}
	before := append([]BattlePlan(nil), pl.Plans...)
	errs, _ := apply(g, 0, DeletePlanOrder{Index: 3})
	if errs[0] != nil {
		t.Fatal(errs[0])
	}
	var got []int
	for _, f := range g.Fleets {
		got = append(got, f.Plan)
	}
	if !reflect.DeepEqual(got, []int{2, 4, 2, 5, 4}) {
		t.Errorf("fleet plans %v, want 2 4 2 5 and player 1's 4 untouched", got)
	}
	want := append(append([]BattlePlan(nil), before[:3]...), before[4:]...)
	if !reflect.DeepEqual(g.Players[0].Plans, want) {
		t.Errorf("plans %+v, want %+v", g.Players[0].Plans, want)
	}
}

func TestPredictionBattlePlanOrders(t *testing.T) {
	// COMBAT.md "Adding, replacing and deleting" (BINARY-ONLY) and
	// ORDERS.md "Battle-plan fields" (Elegy's chosen rule): a definition
	// replaces or adds the next plan, up to 16; an out-of-range tactic
	// (6) or target (8) is rejected; plan 0 is never deleted; a fleet's
	// plan must be one its owner has.
	g := ordersGame()
	p := BattlePlan{Name: "Raid", Tactic: TacticMaximizeDamage, Primary: TargetFreighters, Attack: AttackPlayer, Player: 1}
	errs, _ := apply(g, 0,
		BattlePlanOrder{Index: 5, Plan: p},                                // adds plan 5
		BattlePlanOrder{Index: 2, Plan: p},                                // replaces plan 2
		BattlePlanOrder{Index: 7, Plan: p},                                // beyond the next number
		BattlePlanOrder{Index: 6, Plan: BattlePlan{Tactic: 6}},            // tactic 6
		BattlePlanOrder{Index: 6, Plan: BattlePlan{Primary: 8}},           // target 8
		BattlePlanOrder{Index: 6, Plan: BattlePlan{Attack: AttackPlayer}}, // attacks itself
		DeletePlanOrder{Index: 0},                                         // plan 0
		FleetPlanOrder{Fleet: 1, Plan: 5},                                 // ok
		FleetPlanOrder{Fleet: 2, Plan: 6},                                 // no plan 6
		FleetPlanOrder{Fleet: 3, Plan: 0},                                 // player 1's fleet
	)
	want := []error{nil, nil, ErrOutOfRange, ErrOutOfRange, ErrOutOfRange, ErrOutOfRange, ErrOutOfRange, nil, ErrOutOfRange, ErrNotYours}
	for i := range want {
		if !errors.Is(errs[i], want[i]) && !(want[i] == nil && errs[i] == nil) {
			t.Errorf("order %d: %v, want %v", i, errs[i], want[i])
		}
	}
	pl := g.Players[0]
	if len(pl.Plans) != 6 || pl.Plans[2] != p || pl.Plans[5] != p || g.Fleets[0].Plan != 5 || g.Fleets[1].Plan != 4 {
		t.Errorf("plans %d, plan 2 %+v, fleets on %d and %d", len(pl.Plans), pl.Plans[2], g.Fleets[0].Plan, g.Fleets[1].Plan)
	}
	for len(g.Players[0].Plans) < maxBattlePlans {
		if errs, _ := apply(g, 0, BattlePlanOrder{Index: len(g.Players[0].Plans), Plan: p}); errs[0] != nil {
			t.Fatal(errs[0])
		}
	}
	if errs, _ := apply(g, 0, BattlePlanOrder{Index: maxBattlePlans, Plan: p}); !errors.Is(errs[0], ErrOutOfRange) {
		t.Errorf("17th plan: %v, want refused", errs[0])
	}
}

func TestPredictionOwnershipEveryOrder(t *testing.T) {
	// ORDERS.md "Ownership" (chosen rule): an order naming another
	// player's fleet or planet is rejected, for the kinds the original
	// re-checks and for those it does not. Nothing changes.
	g := ordersGame()
	before := ordersGame()
	errs, _ := apply(g, 0,
		FleetPlanOrder{Fleet: 3, Plan: 1},
		RenameOrder{Fleet: 3, Name: "Mine now"},
		WaypointOrder{Fleet: 3, Waypoints: []Waypoint{{Pos: Point{1500, 1500}, Warp: 5}}},
		CargoOrder{Fleet: 3, Target: TargetPlanet, ID: 2, Amounts: [NumCargo + 1]int{-1}},
		MergeOrder{Into: 1, From: []int{3}},
		QueueOrder{Planet: 2},
		PlanetFlagsOrder{Planet: 2, LeftoverOnly: true},
	)
	for i, err := range errs {
		if err == nil || (i != 4 && !errors.Is(err, ErrNotYours)) {
			t.Errorf("order %d: %v, want refused as another player's", i, err)
		}
	}
	if !reflect.DeepEqual(g, before) {
		t.Error("a refused order changed the game")
	}
}

func TestPlaceholderOrders(t *testing.T) {
	// The production queue, planet flags and settings orders wait on
	// their stars-elegy spec; minefields are not modelled.
	g := ordersGame()
	errs, _ := apply(g, 0, QueueOrder{Planet: 1}, PlanetFlagsOrder{Planet: 1}, SettingsOrder{}, DetonateOrder{Minefield: 1})
	for i, want := range []error{ErrAwaitingSpec, ErrAwaitingSpec, ErrAwaitingSpec, ErrNotModelled} {
		if !errors.Is(errs[i], want) {
			t.Errorf("order %d: %v, want %v", i, errs[i], want)
		}
	}
}

func TestPredictionWaypointClamp(t *testing.T) {
	// ORDERS.md "Waypoint coordinates" (BINARY-ONLY): a waypoint outside
	// the galaxy is clamped to it, not rejected. A tiny galaxy runs from
	// 1000 to 1400 (UNIVERSE.md). A planet target takes the planet's
	// position.
	g := ordersGame()
	errs, _ := apply(g, 0,
		WaypointOrder{Fleet: 1, Waypoints: []Waypoint{{Pos: Point{2000, 900}, Warp: 6}, {Target: TargetPlanet, ID: 2, Warp: 5}}},
		WaypointOrder{Fleet: 2, Waypoints: []Waypoint{{Pos: Point{1200, 1200}, Warp: 11}}},
		WaypointOrder{Fleet: 2, Waypoints: []Waypoint{{Target: TargetPlanet, ID: 9}}},
	)
	if errs[0] != nil || !errors.Is(errs[1], ErrOutOfRange) || !errors.Is(errs[2], ErrNoSuchObject) {
		t.Fatalf("errors %v", errs)
	}
	want := []Waypoint{{Pos: Point{1400, 1000}, Warp: 6}, {Pos: Point{1200, 1200}, Warp: 5, Target: TargetPlanet, ID: 2}}
	if !reflect.DeepEqual(g.Fleets[0].Waypoints, want) {
		t.Errorf("waypoints %+v, want %+v", g.Fleets[0].Waypoints, want)
	}
}

func TestRenameOrder(t *testing.T) {
	g := ordersGame()
	errs, _ := apply(g, 0, RenameOrder{Fleet: 1, Name: "Armada"}, RenameOrder{Fleet: 2, Name: "this name is far longer than thirty-one"})
	if errs[0] != nil || !errors.Is(errs[1], ErrOutOfRange) || g.Fleets[0].Name != "Armada" || g.Fleets[1].Name != "" {
		t.Errorf("errors %v, names %q %q", errs, g.Fleets[0].Name, g.Fleets[1].Name)
	}
}

func TestConfirmedCargoClamps(t *testing.T) {
	// ORDERS.md "Cargo amounts and clamps" (CONFIRMED FO-01..07): an
	// amount is the minimum of what the source has, what is asked and the
	// destination's free space; hold and tank are independent.
	g := ordersGame()
	// Fleet 1 (30 Fe and 10 colonists in a 100 kT hold, 150 of 200 mg)
	// gives fleet 2 (empty hold, 20 mg) 50 Fe, of which it has 30, and
	// 100 mg, which fit.
	errs, _ := apply(g, 0, CargoOrder{Fleet: 1, Target: TargetFleet, ID: 2, Amounts: [NumCargo + 1]int{-50, 0, 0, 0, -100}})
	if errs[0] != nil {
		t.Fatal(errs[0])
	}
	f1, f2 := g.Fleets[0], g.Fleets[1]
	if f1.Cargo.Minerals[Ironium] != 0 || f2.Cargo.Minerals[Ironium] != 30 || f1.Fuel != 50 || f2.Fuel != 120 {
		t.Errorf("after giving: f1 %+v %d mg, f2 %+v %d mg", f1.Cargo, f1.Fuel, f2.Cargo, f2.Fuel)
	}
	// Fleet 2 then loads 90 Bo (the planet has 40) and 70 Ge (30 fit the
	// hold), and with its hold full still takes fuel: 500 mg asked, 50 left
	// in fleet 1, 80 free.
	errs, _ = apply(g, 0,
		CargoOrder{Fleet: 2, Target: TargetPlanet, ID: 1, Amounts: [NumCargo + 1]int{0, 90, 70, 0, 0}},
		CargoOrder{Fleet: 2, Target: TargetFleet, ID: 1, Amounts: [NumCargo + 1]int{0, 0, 0, 0, 500}},
	)
	if errs[0] != nil || errs[1] != nil {
		t.Fatal(errs)
	}
	f1, f2 = g.Fleets[0], g.Fleets[1]
	if f2.Cargo.Minerals != (Minerals{30, 40, 30}) || g.Planets[0].Surface != (Minerals{40, 0, 10}) || f2.Fuel != 170 || f1.Fuel != 0 {
		t.Errorf("after loading: f2 %+v %d mg, planet %+v, f1 %d mg", f2.Cargo, f2.Fuel, g.Planets[0].Surface, f1.Fuel)
	}
}

func TestConfirmedCargoOwnPlanet(t *testing.T) {
	// TAKEOVER.md "Unload and load amounts" (CONFIRMED TK-201): on the
	// owner's planet unloaded colonists join the population at once, and
	// a load may take every colonist.
	g := ordersGame()
	errs, _ := apply(g, 0,
		CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 1, Amounts: [NumCargo + 1]int{0, 0, 0, -10}},
		CargoOrder{Fleet: 2, Target: TargetPlanet, ID: 1, Amounts: [NumCargo + 1]int{0, 0, 0, 1000}},
	)
	if errs[0] != nil || errs[1] != nil {
		t.Fatal(errs)
	}
	if g.Planets[0].Population != 0 || g.Fleets[1].Cargo.Colonists != 60 || g.Fleets[0].Cargo.Colonists != 0 {
		t.Errorf("population %d, fleet 2 colonists %d", g.Planets[0].Population, g.Fleets[1].Cargo.Colonists)
	}
}

func TestPredictionCrossOwnerCargo(t *testing.T) {
	// ORDERS.md "Cross-owner cargo" (BINARY-ONLY): colonists given to a
	// planet the giver does not own become a drop; minerals given to
	// another owner leave the giver now and are credited later; taking
	// from another owner is rejected (chosen rule).
	g := ordersGame()
	errs, a := apply(g, 0,
		CargoOrder{Fleet: 4, Target: TargetPlanet, ID: 3, Amounts: [NumCargo + 1]int{0, -15, 0, -25}},
		CargoOrder{Fleet: 4, Target: TargetPlanet, ID: 3, Amounts: [NumCargo + 1]int{0, 1, 0, 0}},
	)
	if errs[0] != nil || !errors.Is(errs[1], ErrNotYours) {
		t.Fatal(errs)
	}
	f := g.Fleets[3]
	if f.Cargo != (Cargo{Minerals: Minerals{0, 5, 0}, Colonists: 5}) {
		t.Errorf("giver's cargo %+v", f.Cargo)
	}
	if !reflect.DeepEqual(a.drops, []drop{{planet: 2, player: 0, troops: 25}}) {
		t.Errorf("drops %+v", a.drops)
	}
	if len(a.Gifts) != 1 || a.Gifts[0].Amounts != [NumCargo + 1]int{0, 15, 0, 0, 0} || g.Planets[2].Surface != (Minerals{}) {
		t.Fatalf("gifts %+v, planet %+v", a.Gifts, g.Planets[2].Surface)
	}
	g.DeliverGifts(a.Gifts)
	if g.Planets[2].Surface != (Minerals{0, 15, 0}) {
		t.Errorf("planet after delivery %+v", g.Planets[2].Surface)
	}
}

func TestPredictionCargoToForeignFleet(t *testing.T) {
	// TAKEOVER.md "Other waypoint tasks": colonists are never given to
	// another player's fleet, and nothing moves to a receiver that
	// regards the giver as an enemy. A gift that does not fit the
	// receiver's hold is lost on delivery (ORDERS.md "Cross-owner
	// cargo").
	g := ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	g.Fleets[2].Cargo.Minerals[Germanium] = 90
	errs, a := apply(g, 0,
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{0, 0, 0, -5}},
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30, 0, 0, 0, -10}},
	)
	if !errors.Is(errs[0], ErrRefusedByOwner) || errs[1] != nil {
		t.Fatal(errs)
	}
	ev := g.DeliverGifts(a.Gifts)
	if g.Fleets[2].Cargo.Minerals != (Minerals{10, 0, 90}) || g.Fleets[2].Fuel != 110 || g.Fleets[0].Cargo.Minerals[Ironium] != 0 {
		t.Errorf("receiver %+v %d mg", g.Fleets[2].Cargo, g.Fleets[2].Fuel)
	}
	if len(ev) != 2 || ev[0].Kind != EventCargoGiven || ev[0].Count != 20 || ev[1].Kind != EventCargoGiftLost || ev[1].Count != 20 {
		t.Errorf("events %+v", ev)
	}
	g = ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	g.Players[1].Relations = []Relation{RelationEnemy, RelationFriend}
	if errs, _ := apply(g, 0, CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30}}); !errors.Is(errs[0], ErrRefusedByOwner) || g.Fleets[0].Cargo.Minerals[Ironium] != 30 {
		t.Errorf("enemy receiver: %v, giver kept %d", errs[0], g.Fleets[0].Cargo.Minerals[Ironium])
	}
}

func TestCargoChecks(t *testing.T) {
	// ASSUMPTION L7: the fleet must be at the target; fuel and planets do
	// not mix.
	g := ordersGame()
	errs, _ := apply(g, 0,
		CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 2, Amounts: [NumCargo + 1]int{-1}},
		CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 1, Amounts: [NumCargo + 1]int{0, 0, 0, 0, -1}},
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 1},
	)
	for i, want := range []error{ErrNotTogether, ErrOutOfRange, ErrNoSuchObject} {
		if !errors.Is(errs[i], want) {
			t.Errorf("order %d: %v, want %v", i, errs[i], want)
		}
	}
}

func TestPredictionDesignOrders(t *testing.T) {
	// ORDERS.md "Design legality" (chosen rule, through ReadDesign): a
	// part above the owner's tech is dropped and the design stored; a
	// hull the owner may not build rejects it. Deleting a design removes
	// its starbase (KERNEL.md, BINARY-ONLY) and its ships (L12).
	g := ordersGame()
	g.Players[0].Research.Levels[Construction] = 3
	errs, _ := apply(g, 0,
		DesignOrder{Slot: 0, Name: "D", Hull: "Destroyer", Fills: []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}, {Slot: 1, Part: "Beta Torpedo", Count: 1}}},
		DesignOrder{Slot: 1, Name: "BB", Hull: "Battleship", Fills: []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 4}}},
		DesignOrder{Slot: 16, Name: "X", Hull: "Scout", Fills: []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}},
		DesignOrder{Slot: 2, Name: "F", Hull: "Orbital Fort"},
		DesignOrder{Starbase: true, Slot: 0, Name: "Fort", Hull: "Orbital Fort"},
	)
	if errs[0] != nil || errs[1] == nil || !errors.Is(errs[2], ErrOutOfRange) || !errors.Is(errs[3], ErrOutOfRange) || errs[4] != nil {
		t.Fatalf("errors %v", errs)
	}
	di, ok := g.PlayerDesign(0, false, 0)
	if !ok || len(g.Designs[di].Slots) != 1 {
		t.Fatalf("slot 0: %v %+v", ok, g.Designs)
	}
	sb, _ := g.PlayerDesign(0, true, 0)
	g.Planets[0].HasStarbase, g.Planets[0].StarbaseDesign, g.Planets[0].StarbaseHull = true, sb, 1
	g.Fleets[1].Stacks = []Stack{{Design: di, Count: 2}}
	g.Fleets[0].Stacks = append(g.Fleets[0].Stacks, Stack{Design: di, Count: 1})
	// In use: a new design for slot 0 is refused (L11).
	if errs, _ := apply(g, 0, DesignOrder{Slot: 0, Name: "D2", Hull: "Scout", Fills: []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}}); errs[0] == nil {
		t.Error("replacing a design in use: want refused")
	}
	errs, _ = apply(g, 0, DeleteDesignOrder{Slot: 0}, DeleteDesignOrder{Starbase: true, Slot: 0}, DeleteDesignOrder{Slot: 5})
	if errs[0] != nil || errs[1] != nil || !errors.Is(errs[2], ErrNoSuchObject) {
		t.Fatalf("errors %v", errs)
	}
	if _, ok := g.PlayerDesign(0, false, 0); ok || g.Planets[0].HasStarbase || g.Planets[0].Population != 50 {
		t.Errorf("after delete: slot kept %v, starbase %v, population %d", ok, g.Planets[0].HasStarbase, g.Planets[0].Population)
	}
	if g.fleetIndex(2) >= 0 || len(g.Fleets[0].Stacks) != 1 {
		t.Errorf("fleets after delete %+v", g.Fleets)
	}
}

func TestPredictionPlayerShuffle(t *testing.T) {
	// KERNEL.md "Turn order", 1. Orders, step 2 (stars-elegy #53): for
	// i = 0..n−1 swap position i with i + Random(n − i); one draw per
	// player, the last always 0. Draws 2, 0, 1, 0 on four players:
	// [0 1 2 3] → [2 1 0 3] → [2 1 0 3] → [2 1 3 0] → [2 1 3 0].
	r := &seqRand{draws: []int{2, 0, 1, 0}}
	if got := ShufflePlayers(4, r); !reflect.DeepEqual(got, []int{2, 1, 3, 0}) || len(r.draws) != 0 {
		t.Errorf("order %v, %d draws left", got, len(r.draws))
	}
}
