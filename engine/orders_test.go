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
		PlanetSettingsOrder{Planet: 2, LeftoverOnly: true},
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

// Without space objects there is no minefield for a detonate order to
// name; the space objects' own refusals are tested in package objects.
func TestDetonateWithoutObjects(t *testing.T) {
	g := ordersGame()
	if errs, _ := apply(g, 0, DetonateOrder{Minefield: 1}); !errors.Is(errs[0], ErrNoSuchObject) {
		t.Errorf("detonate: %v", errs[0])
	}
}

func TestConfirmedQueueReplace(t *testing.T) {
	// LIMITS.md "Production-queue replace"
	// (CONFIRMED LQ-1..LQ-4) with its chosen rule: a sent percentage is kept
	// only against an unused old item of the same kind with exactly that
	// percentage. The base queue is Factory ×5 at 49%, Mine ×5 at 30%,
	// Defenses ×5, Factory ×5 at 20%.
	base := []QueueItem{{ItemFactory, 5, 49, 0}, {ItemMine, 5, 30, 0}, {ItemDefenses, 5, 0, 0}, {ItemFactory, 5, 20, 0}}
	for _, tt := range []struct {
		name      string
		sent      []QueueItem
		want      []QueueItem
		wantQueue bool
	}{
		// Moved and recounted items keep progress (LQ-1: Mine ×5 → ×3).
		{"moved", []QueueItem{{ItemFactory, 5, 20, 0}, {ItemMine, 3, 30, 0}, {ItemDefenses, 5, 0, 0}, {ItemFactory, 5, 49, 0}},
			[]QueueItem{{ItemFactory, 5, 20, 0}, {ItemMine, 3, 30, 0}, {ItemDefenses, 5, 0, 0}, {ItemFactory, 5, 49, 0}}, true},
		// LQ-2: the 49% Factory removed, a new Factory at the top sent at
		// 0; the other Factory keeps its 20.
		{"LQ-2", []QueueItem{{ItemFactory, 5, 0, 0}, {ItemMine, 5, 30, 0}, {ItemDefenses, 5, 0, 0}, {ItemFactory, 5, 20, 0}},
			[]QueueItem{{ItemFactory, 5, 0, 0}, {ItemMine, 5, 30, 0}, {ItemDefenses, 5, 0, 0}, {ItemFactory, 5, 20, 0}}, true},
		// LQ-4: an empty list removes the queue.
		{"LQ-4", nil, nil, false},
	} {
		g := ordersGame()
		p := &g.Planets[0]
		p.HasQueue, p.Queue = true, append([]QueueItem(nil), base...)
		if errs, _ := apply(g, 0, QueueOrder{Planet: 1, Queue: tt.sent}); errs[0] != nil {
			t.Fatalf("%s: %v", tt.name, errs[0])
		}
		if p.HasQueue != tt.wantQueue || !reflect.DeepEqual(p.Queue, tt.want) {
			t.Errorf("%s: queue %v %+v, want %+v", tt.name, p.HasQueue, p.Queue, tt.want)
		}
	}
}

func TestPredictionQueueNoNewProgress(t *testing.T) {
	// The chosen rule never creates progress: on the LQ-3 host-edited
	// queue (Factory 0, Mine 0, Factory 20) a client sending 49, 30, 20
	// gets 0, 0, 20 (the original gave the first Factory 49). Bad items
	// are refused (L14).
	g := ordersGame()
	p := &g.Planets[0]
	p.HasQueue, p.Queue = true, []QueueItem{{ItemFactory, 9, 0, 0}, {ItemMine, 5, 0, 0}, {ItemFactory, 9, 20, 0}}
	errs, _ := apply(g, 0,
		QueueOrder{Planet: 1, Queue: []QueueItem{{ItemFactory, 9, 49, 0}, {ItemMine, 5, 30, 0}, {ItemFactory, 9, 20, 0}}},
		QueueOrder{Planet: 1, Queue: []QueueItem{{ItemMine, 0, 0, 0}}},
		QueueOrder{Planet: 1, Queue: []QueueItem{{ItemMine, 1024, 0, 0}}},
	)
	if errs[0] != nil || !errors.Is(errs[1], ErrOutOfRange) || !errors.Is(errs[2], ErrOutOfRange) {
		t.Fatal(errs)
	}
	if want := []QueueItem{{ItemFactory, 9, 0, 0}, {ItemMine, 5, 0, 0}, {ItemFactory, 9, 20, 0}}; !reflect.DeepEqual(p.Queue, want) {
		t.Errorf("queue %+v, want %+v", p.Queue, want)
	}
}

func TestPredictionSettingOrders(t *testing.T) {
	// LIMITS.md "Setting orders" (BINARY-ONLY): planet settings on the
	// sender's planet; relations change only the sender's row.
	g := ordersGame()
	errs, _ := apply(g, 0,
		PlanetSettingsOrder{Planet: 1, LeftoverOnly: true, HasRoute: true, RouteTo: 3},
		PlanetSettingsOrder{Planet: 1, HasRoute: true, RouteTo: 99},
		RelationsOrder{Relations: []Relation{RelationEnemy, RelationEnemy}},
		RelationsOrder{Relations: []Relation{RelationEnemy}},
	)
	if errs[0] != nil || !errors.Is(errs[1], ErrNoSuchObject) || errs[2] != nil || !errors.Is(errs[3], ErrOutOfRange) {
		t.Fatal(errs)
	}
	p := g.Planets[0]
	if !p.LeftoverOnly || !p.HasRoute || p.RouteTo != 3 {
		t.Errorf("planet %+v", p)
	}
	if !reflect.DeepEqual(g.Players[0].Relations, []Relation{RelationFriend, RelationEnemy}) || g.Players[1].Relations != nil {
		t.Errorf("relations %v / %v", g.Players[0].Relations, g.Players[1].Relations)
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
		WaypointOrder{Fleet: 2, Waypoints: []Waypoint{{Pos: Point{1200, 1200}, Warp: 12}}},
		WaypointOrder{Fleet: 2, Waypoints: []Waypoint{{Target: TargetPlanet, ID: 9}}},
		WaypointOrder{Fleet: 2, Waypoints: []Waypoint{{Pos: Point{1200, 1200}, Warp: 11}}},
	)
	// ORDERS.md Q13 (chosen rule): warp outside 0..11 and a missing
	// target are rejected; warp 11, the stargate hop (CONFIRMED GT-004),
	// is accepted, the gate conditions checked at the jump (OBJECTS.md
	// "Stargates").
	if errs[0] != nil || !errors.Is(errs[1], ErrOutOfRange) || !errors.Is(errs[2], ErrNoSuchObject) || errs[3] != nil {
		t.Fatalf("errors %v", errs)
	}
	want := []Waypoint{{Pos: Point{1400, 1000}, Warp: 6}, {Pos: Point{1200, 1200}, Warp: 5, Target: TargetPlanet, ID: 2}}
	if !reflect.DeepEqual(g.Fleets[0].Waypoints, want) {
		t.Errorf("waypoints %+v, want %+v", g.Fleets[0].Waypoints, want)
	}
}

func TestWaypointOrderUpkeepTasks(t *testing.T) {
	// ORDERS.md "Waypoint upkeep and the remaining tasks": route, patrol
	// and transfer-fleet tasks are accepted; a transfer naming an absent
	// player is accepted and refused when the task runs ("Transfer
	// fleet"). ASSUMPTION L18: a negative patrol range is rejected.
	g := ordersGame()
	patrol := Task{Kind: TaskPatrol, Range: 50}
	gift := Task{Kind: TaskTransferFleet, Player: 7}
	errs, _ := apply(g, 0,
		WaypointOrder{Fleet: 1, Task: patrol, Waypoints: []Waypoint{{Pos: Point{1100, 1100}, Task: Task{Kind: TaskRoute}}}},
		WaypointOrder{Fleet: 2, Waypoints: []Waypoint{{Pos: Point{1100, 1100}}, {Pos: Point{1200, 1200}, Warp: 5, Task: gift}}},
		WaypointOrder{Fleet: 2, Task: Task{Kind: TaskPatrol, Range: -1}},
	)
	if errs[0] != nil || errs[1] != nil || !errors.Is(errs[2], ErrOutOfRange) {
		t.Fatalf("errors %v", errs)
	}
	if g.Fleets[0].Task != patrol || g.Fleets[1].Waypoints[1].Task != gift {
		t.Errorf("tasks %+v and %+v", g.Fleets[0].Task, g.Fleets[1].Waypoints[1].Task)
	}
}

func TestWaypointOrderMiningAndLayingTasks(t *testing.T) {
	// KERNEL.md "Remote mining" and OBJECTS.md "Laying": both tasks are
	// accepted and refused, if at all, when they run. ASSUMPTION L24: a
	// lay-mines duration below one year, other than indefinitely, is
	// rejected.
	g := ordersGame()
	lay := Task{Kind: TaskLayMines, Years: 3}
	forever := Task{Kind: TaskLayMines, Years: YearsIndefinitely}
	mine := Task{Kind: TaskRemoteMine}
	errs, _ := apply(g, 0,
		WaypointOrder{Fleet: 1, Task: lay, Waypoints: []Waypoint{{Pos: Point{1100, 1100}, Task: mine}}},
		WaypointOrder{Fleet: 2, Task: forever},
		WaypointOrder{Fleet: 2, Task: Task{Kind: TaskLayMines}},
		WaypointOrder{Fleet: 2, Task: Task{Kind: TaskLayMines, Years: -2}},
	)
	if errs[0] != nil || errs[1] != nil || !errors.Is(errs[2], ErrOutOfRange) || !errors.Is(errs[3], ErrOutOfRange) {
		t.Fatalf("errors %v", errs)
	}
	if g.Fleets[0].Task != lay || g.Fleets[0].Waypoints[0].Task != mine || g.Fleets[1].Task != forever {
		t.Errorf("tasks %+v, %+v and %+v", g.Fleets[0].Task, g.Fleets[0].Waypoints[0].Task, g.Fleets[1].Task)
	}
}

func TestRepeatOrder(t *testing.T) {
	// ORDERS.md "Reaching a waypoint": the repeat-orders flag; ownership
	// checked (ORDERS.md "Ownership", chosen rule, against LIMITS.md's
	// LEGACY BUG of no owner check).
	g := ordersGame()
	errs, _ := apply(g, 0, RepeatOrder{Fleet: 1, On: true}, RepeatOrder{Fleet: 3, On: true})
	if errs[0] != nil || !errors.Is(errs[1], ErrNotYours) || !g.Fleets[0].Repeat || g.Fleets[2].Repeat {
		t.Errorf("errors %v, repeat %v %v", errs, g.Fleets[0].Repeat, g.Fleets[2].Repeat)
	}
	apply(g, 0, RepeatOrder{Fleet: 1})
	if g.Fleets[0].Repeat {
		t.Error("repeat not cleared")
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

func TestConfirmedManualTransfersToOthers(t *testing.T) {
	// TAKEOVER.md "Manual cargo transfers to other players" (stars-elegy
	// CONFIRMED TK-501, TK-502; minerals MEASURED TK-405, TK-412):
	// colonists onto another player's planet are a drop, whatever the
	// relations; onto an unowned planet they are lost and the giver is
	// told; minerals join the surface once every file has applied, with no
	// message. Taking from another owner is rejected (chosen rule).
	g := ordersGame()
	g.Fleets[0].Pos = g.Planets[1].Pos
	g.Players[1].Relations = []Relation{RelationFriend, RelationFriend}
	errs, a := apply(g, 0,
		CargoOrder{Fleet: 4, Target: TargetPlanet, ID: 3, Amounts: [NumCargo + 1]int{0, -15, 0, -25}},
		CargoOrder{Fleet: 4, Target: TargetPlanet, ID: 3, Amounts: [NumCargo + 1]int{0, 1, 0, 0}},
		CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 2, Amounts: [NumCargo + 1]int{-100, 0, 0, -10}},
	)
	if errs[0] != nil || !errors.Is(errs[1], ErrNotYours) || errs[2] != nil {
		t.Fatal(errs)
	}
	if f := g.Fleets[3]; f.Cargo != (Cargo{Minerals: Minerals{0, 5, 0}, Colonists: 5}) {
		t.Errorf("giver's cargo %+v", f.Cargo)
	}
	if !reflect.DeepEqual(a.drops, []drop{{planet: 1, player: 0, troops: 10}}) {
		t.Errorf("drops %+v, want 10 onto planet 2", a.drops)
	}
	if len(a.Events) != 1 || a.Events[0].Kind != EventColonistsLostGiven || a.Events[0].Count != 25 {
		t.Errorf("events %+v, want 25 colonists lost on the unowned planet", a.Events)
	}
	if g.Planets[2].Surface != (Minerals{0, 15, 0}) || g.Planets[1].Surface != (Minerals{30, 0, 0}) {
		t.Errorf("surfaces %+v and %+v", g.Planets[2].Surface, g.Planets[1].Surface)
	}
}

func TestGiftCreditedInPlace(t *testing.T) {
	// ORDERS.md "Cross-owner cargo": the gift is credited in place as its
	// order applies, so player 1, replaying after player 0, can unload
	// what it was just given the same year (BINARY-ONLY).
	g := ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	files := []PlayerOrders{
		{Player: 0, GameID: 77, Year: 2410, Orders: []Order{CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30}}}},
		{Player: 1, GameID: 77, Year: 2410, Orders: []Order{CargoOrder{Fleet: 3, Target: TargetPlanet, ID: 2, Amounts: [NumCargo + 1]int{-30}}}},
	}
	ApplyOrders(g, files, []int{0, 1})
	if g.Planets[1].Surface[Ironium] != 30 || g.Fleets[2].Cargo.Minerals[Ironium] != 0 {
		t.Errorf("planet %d Fe, fleet %d Fe; want 30 and 0", g.Planets[1].Surface[Ironium], g.Fleets[2].Cargo.Minerals[Ironium])
	}
}

func TestPredictionCargoToForeignFleet(t *testing.T) {
	// TAKEOVER.md (MEASURED TK-406/407/409, over-full CONFIRMED): no
	// relation check; the receiver takes what fits and the rest is lost,
	// the giver told. Colonists to another player's fleet are rejected
	// (chosen rule; no legal client writes them, TK-408, TK-414).
	g := ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	g.Fleets[2].Cargo.Minerals[Germanium] = 90
	g.Players[1].Relations = []Relation{RelationEnemy, RelationFriend}
	errs, a := apply(g, 0,
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{0, 0, 0, -5}},
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30, 0, 0, 0, -10}},
	)
	if !errors.Is(errs[0], ErrRefusedByOwner) || errs[1] != nil {
		t.Fatal(errs)
	}
	if g.Fleets[2].Cargo.Minerals != (Minerals{10, 0, 90}) || g.Fleets[2].Fuel != 110 || g.Fleets[0].Cargo.Minerals[Ironium] != 0 {
		t.Errorf("receiver %+v %d mg", g.Fleets[2].Cargo, g.Fleets[2].Fuel)
	}
	if len(a.Events) != 1 || a.Events[0].Kind != EventCargoGiftLost || a.Events[0].Count != 20 {
		t.Errorf("events %+v", a.Events)
	}
}

func TestGiftReceiverRemovedEarlier(t *testing.T) {
	// ORDERS.md "Cross-owner cargo", "Missing endpoint" (stars-elegy #87,
	// BINARY-ONLY for the same-turn removal): player 1's replay runs first
	// and merges the receiver away, so player 0's gift is skipped whole and
	// the giver keeps the cargo.
	g := ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	g.Fleets = append(g.Fleets, Fleet{ID: 5, Owner: 1, Pos: g.Fleets[2].Pos, Stacks: []Stack{{Design: 1, Count: 1}}})
	files := []PlayerOrders{
		{Player: 0, GameID: 77, Year: 2410, Orders: []Order{CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30}}}},
		{Player: 1, GameID: 77, Year: 2410, Orders: []Order{MergeOrder{Into: 5, From: []int{3}}}},
	}
	a := ApplyOrders(g, files, []int{1, 0})
	if g.Fleets[0].Cargo.Minerals[Ironium] != 30 || len(a.Gifts) != 0 || len(a.Events) != 0 {
		t.Errorf("giver Fe %d, gifts %+v, events %+v; want 30, none, none", g.Fleets[0].Cargo.Minerals[Ironium], a.Gifts, a.Events)
	}
}

func TestGiftToFleetMergedAway(t *testing.T) {
	// ORDERS.md "Receiver removed after the credit, same turn"
	// (BINARY-ONLY): the credited gift is the receiver's own cargo, so a
	// later merge pools it into the surviving fleet; nothing goes back to
	// the giver.
	g := ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	g.Fleets = append(g.Fleets, Fleet{ID: 5, Owner: 1, Pos: g.Fleets[2].Pos, Stacks: []Stack{{Design: 1, Count: 1}}})
	files := []PlayerOrders{
		{Player: 0, GameID: 77, Year: 2410, Orders: []Order{CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30}}}},
		{Player: 1, GameID: 77, Year: 2410, Orders: []Order{MergeOrder{Into: 5, From: []int{3}}}},
	}
	a := ApplyOrders(g, files, []int{0, 1})
	if g.fleetIndex(3) >= 0 {
		t.Fatal("receiver not merged away")
	}
	if giver, into := g.Fleets[0].Cargo.Minerals[Ironium], g.Fleets[g.fleetIndex(5)].Cargo.Minerals[Ironium]; giver != 0 || into != 30 || len(a.Events) != 0 {
		t.Errorf("giver Fe %d, surviving fleet Fe %d, events %+v; want 0, 30, none", giver, into, a.Events)
	}
}

func TestMeasuredGiftLostWithDeletedDesign(t *testing.T) {
	// ORDERS.md "Receiver removed after the credit, same turn": a later
	// design delete shares the gifted cargo out as any cargo (MEASURED
	// CO-07c), so a receiver whose only ships are deleted loses it all.
	g := ordersGame()
	g.Fleets[0].Pos = g.Fleets[2].Pos
	g.DesignSlots = append(g.DesignSlots, DesignSlot{Owner: 1, Slot: 0, Design: 1})
	files := []PlayerOrders{
		{Player: 0, GameID: 77, Year: 2410, Orders: []Order{CargoOrder{Fleet: 1, Target: TargetFleet, ID: 3, Amounts: [NumCargo + 1]int{-30}}}},
		{Player: 1, GameID: 77, Year: 2410, Orders: []Order{DeleteDesignOrder{Slot: 0}}},
	}
	a := ApplyOrders(g, files, []int{0, 1})
	if a.Results[len(a.Results)-1].Err != nil {
		t.Fatal(a.Results)
	}
	if g.fleetIndex(3) >= 0 || g.Fleets[0].Cargo.Minerals[Ironium] != 0 || len(a.Events) != 0 {
		t.Errorf("receiver index %d, giver Fe %d, events %+v; want gone, 0, none", g.fleetIndex(3), g.Fleets[0].Cargo.Minerals[Ironium], a.Events)
	}
}

func TestConfirmedOwnFleetTransferExplicit(t *testing.T) {
	// ORDERS.md "Transfer between the player's own fleets" (CONFIRMED
	// CO-04): the amount given moves, not a capacity rebalance (which
	// would even two equal holds), and an amount over what is aboard
	// moves what is aboard.
	g := ordersGame()
	errs, _ := apply(g, 0,
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 2, Amounts: [NumCargo + 1]int{-20}},
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 2, Amounts: [NumCargo + 1]int{0, 0, 0, -300}},
	)
	if errs[0] != nil || errs[1] != nil {
		t.Fatal(errs)
	}
	if a, b := g.Fleets[0].Cargo, g.Fleets[1].Cargo; a.Minerals[Ironium] != 10 || b.Minerals[Ironium] != 20 || a.Colonists != 0 || b.Colonists != 10 {
		t.Errorf("fleets %+v and %+v; want Fe 10/20, colonists 0/10", a, b)
	}
}

func TestCargoChecks(t *testing.T) {
	// ORDERS.md Q4 (chosen rules): the fleet must be at the target;
	// fuel to a planet is dropped from the order and its minerals still
	// move.
	g := ordersGame()
	errs, _ := apply(g, 0,
		CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 2, Amounts: [NumCargo + 1]int{-1}},
		CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 1, Amounts: [NumCargo + 1]int{-5, 0, 0, 0, -1}},
		CargoOrder{Fleet: 1, Target: TargetFleet, ID: 1},
	)
	for i, want := range []error{ErrNotTogether, nil, ErrNoSuchObject} {
		if !errors.Is(errs[i], want) {
			t.Errorf("order %d: %v, want %v", i, errs[i], want)
		}
	}
	if f := g.Fleets[0]; f.Fuel != 150 || f.Cargo.Minerals[Ironium] != 25 || g.Planets[0].Surface[Ironium] != 45 {
		t.Errorf("fuel %d, fleet Fe %d, planet Fe %d; want 150, 25, 45", f.Fuel, f.Cargo.Minerals[Ironium], g.Planets[0].Surface[Ironium])
	}
}

func TestPredictionDesignOrders(t *testing.T) {
	// ORDERS.md "Design legality" (chosen rule, through ReadDesign): a
	// part above the owner's tech is dropped and the design stored; a
	// hull the owner may not build rejects it. Deleting a design removes
	// its starbase (KERNEL.md, BINARY-ONLY) and its ships (CO-07).
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
	// In use by ships and a starbase: a change to slot 0 is refused
	// (ORDERS.md "Design change into an occupied slot", Elegy's rule).
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

func TestDesignMalformedFills(t *testing.T) {
	// ORDERS.md "Design read, four malformed cases": a part the slot
	// does not take is dropped, a count over capacity is cut, and an
	// emptied engine slot is back-filled; the design is kept.
	g := ordersGame()
	errs, _ := apply(g, 0, DesignOrder{Slot: 0, Name: "S", Hull: "Scout", Fills: []SlotFill{
		{Slot: 0, Part: "Beta Torpedo", Count: 1},
		{Slot: 2, Part: "Fuel Tank", Count: 9},
		{Slot: 7, Part: "Fuel Tank", Count: 1},
	}})
	if errs[0] != nil {
		t.Fatal(errs[0])
	}
	di, _ := g.PlayerDesign(0, false, 0)
	d := g.Designs[di]
	if d.Engine.Name != basicEngine || len(d.Slots) != 2 || d.Slots[1].Count != 1 {
		t.Errorf("design %+v", d)
	}
}

func TestMeasuredDesignDeleteInPlace(t *testing.T) {
	// ORDERS.md "Design delete effect" (MEASURED CO-07): the slot is
	// cleared in place and later slots keep their numbers.
	g := ordersGame()
	fill := []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}
	apply(g, 0,
		DesignOrder{Slot: 0, Name: "A", Hull: "Scout", Fills: fill},
		DesignOrder{Slot: 1, Name: "B", Hull: "Scout", Fills: fill},
		DesignOrder{Slot: 2, Name: "C", Hull: "Scout", Fills: fill},
	)
	c, _ := g.PlayerDesign(0, false, 2)
	if errs, _ := apply(g, 0, DeleteDesignOrder{Slot: 1}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	if _, ok := g.PlayerDesign(0, false, 1); ok {
		t.Error("slot 1 still filled")
	}
	if got, ok := g.PlayerDesign(0, false, 2); !ok || got != c {
		t.Errorf("slot 2 holds %d (%v), want %d", got, ok, c)
	}
}

func TestMeasuredDesignDeleteSharesFuel(t *testing.T) {
	// ORDERS.md "Design delete effect" (MEASURED CO-07c): a 500 mg fleet
	// of tank 950 losing a 900 mg design keeps 500 − ⌊500·900÷950⌋ = 27
	// mg; cargo is shared by hold the same way, so a hold that was all the
	// deleted design's leaves no iron.
	g := ordersGame()
	g.Designs = append(g.Designs,
		Design{Name: "Keep", FuelCapacity: 50},
		Design{Name: "Gone", FuelCapacity: 900, CargoCapacity: 900})
	keep, gone := len(g.Designs)-2, len(g.Designs)-1
	g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: keep}, {Owner: 0, Slot: 1, Design: gone}}
	f := &g.Fleets[0]
	f.Stacks = []Stack{{Design: keep, Count: 1}, {Design: gone, Count: 1}}
	f.Fuel, f.Cargo = 500, Cargo{Minerals: Minerals{100, 0, 0}}
	if errs, _ := apply(g, 0, DeleteDesignOrder{Slot: 1}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	f = &g.Fleets[g.fleetIndex(1)]
	if f.Fuel != 27 || f.Cargo.Minerals[Ironium] != 0 || len(f.Stacks) != 1 {
		t.Errorf("fleet %+v, want 27 mg, no iron, one stack", *f)
	}
}

func TestMeasuredDesignEditQueueOnly(t *testing.T) {
	// ORDERS.md "Design change into an occupied slot" (MEASURED CO-08): a
	// design with nothing in play is overwritten in place, so what refers
	// to it gets the edited design.
	g := ordersGame()
	fill := []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}
	apply(g, 0, DesignOrder{Slot: 0, Name: "A", Hull: "Scout", Fills: fill})
	before, _ := g.PlayerDesign(0, false, 0)
	if errs, _ := apply(g, 0, DesignOrder{Slot: 0, Name: "A2", Hull: "Scout", Fills: fill}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	if after, _ := g.PlayerDesign(0, false, 0); after != before || g.Designs[before].Name != "A2" {
		t.Errorf("slot 0 now %d (%q), want %d edited in place", after, g.Designs[after].Name, before)
	}
}

func TestPredictionPlayerShuffle(t *testing.T) {
	// KERNEL.md "Turn order", 1. Orders, step 2: for
	// i = 0..n−1 swap position i with i + Random(n − i); one draw per
	// player, the last always 0. Draws 2, 0, 1, 0 on four players:
	// [0 1 2 3] → [2 1 0 3] → [2 1 0 3] → [2 1 3 0] → [2 1 3 0].
	r := &seqRand{draws: []int{2, 0, 1, 0}}
	if got := ShufflePlayers(4, r); !reflect.DeepEqual(got, []int{2, 1, 3, 0}) || len(r.draws) != 0 {
		t.Errorf("order %v, %d draws left", got, len(r.draws))
	}
}

func TestPacketSettingsOrder(t *testing.T) {
	// OBJECTS.md "The settings" (BINARY-ONLY): the packet destination and
	// speed are stored as sent, the destination unchecked; a speed outside
	// 0 or 4..19 is refused (ASSUMPTION L19). Packet items are accepted in
	// a queue.
	g := ordersGame()
	errs, _ := apply(g, 0,
		PlanetSettingsOrder{Planet: 1, HasPacketDest: true, PacketDest: 99, PacketSpeed: 13},
		PlanetSettingsOrder{Planet: 1, HasPacketDest: true, PacketDest: 3, PacketSpeed: 3},
		PlanetSettingsOrder{Planet: 1, HasPacketDest: true, PacketDest: 3, PacketSpeed: 20},
		QueueOrder{Planet: 1, Queue: []QueueItem{{Kind: ItemIroniumPacket, Count: 2}, {Kind: ItemAutoPackets, Count: 1000}}},
	)
	if errs[0] != nil || !errors.Is(errs[1], ErrOutOfRange) || !errors.Is(errs[2], ErrOutOfRange) || errs[3] != nil {
		t.Fatal(errs)
	}
	if p := g.Planets[0]; !p.HasPacketDest || p.PacketDest != 99 || p.PacketSpeed != 13 || len(p.Queue) != 2 {
		t.Errorf("planet %+v", p)
	}
	if errs, _ := apply(g, 0, PlanetSettingsOrder{Planet: 1}); errs[0] != nil || g.Planets[0].HasPacketDest || g.Planets[0].PacketDest != 0 || g.Planets[0].PacketSpeed != 0 {
		t.Errorf("cleared: %v %+v", errs, g.Planets[0])
	}
}

func TestDesignOrderCreatedAndPicture(t *testing.T) {
	// AI.md "Storing a design": the stored design's creation year is the
	// current year, and it has one of the hull's four pictures. ASSUMPTION
	// L25: a picture outside 0..3 is rejected; L26: an edit in place
	// restamps the year and picture.
	g := ordersGame()
	g.Year = 2410
	scout := []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}
	errs, _ := apply(g, 0,
		DesignOrder{Slot: 3, Name: "S", Hull: "Scout", Fills: scout, Picture: 2},
		DesignOrder{Starbase: true, Slot: 1, Name: "Fort", Hull: "Orbital Fort", Picture: 3},
		DesignOrder{Slot: 4, Name: "X", Hull: "Scout", Fills: scout, Picture: 4},
		DesignOrder{Slot: 4, Name: "Y", Hull: "Scout", Fills: scout, Picture: -1},
	)
	if errs[0] != nil || errs[1] != nil || !errors.Is(errs[2], ErrOutOfRange) || !errors.Is(errs[3], ErrOutOfRange) {
		t.Fatalf("errors %v", errs)
	}
	slot := func(starbase bool, n int) DesignSlot {
		for _, s := range g.DesignSlots {
			if s.Owner == 0 && s.Starbase == starbase && s.Slot == n {
				return s
			}
		}
		t.Fatalf("no slot %v %d", starbase, n)
		return DesignSlot{}
	}
	if s := slot(false, 3); s.Created != 2410 || s.Picture != 2 {
		t.Errorf("ship slot 3: created %d picture %d, want 2410 and 2", s.Created, s.Picture)
	}
	if s := slot(true, 1); s.Created != 2410 || s.Picture != 3 {
		t.Errorf("starbase slot 1: created %d picture %d, want 2410 and 3", s.Created, s.Picture)
	}
	g.Year = 2415
	for i := range g.DesignSlots {
		if !g.DesignSlots[i].Starbase && g.DesignSlots[i].Slot == 3 {
			g.DesignSlots[i].Built = 7
		}
	}
	if errs, _ := apply(g, 0, DesignOrder{Slot: 3, Name: "S2", Hull: "Scout", Fills: scout, Picture: 1}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	// ASSUMPTION L29: the edit also starts the built count again.
	if s := slot(false, 3); s.Created != 2415 || s.Picture != 1 || s.Built != 0 {
		t.Errorf("edited slot 3: created %d picture %d built %d, want 2415, 1 and 0", s.Created, s.Picture, s.Built)
	}
}
