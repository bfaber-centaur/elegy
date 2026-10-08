package engine

import (
	"reflect"
	"testing"
)

// Takeover tests follow stars-elegy docs/TAKEOVER.md (with the order and
// amount rules of stars-elegy #34) and the TK-001..TK-007 cases of
// experiments/tk/README.md. TestConfirmed* restate the TK observations;
// TestPrediction* pin BINARY-ONLY rules.

// tkRace is the TK race: JOAT, growth 15%, at 100% habitability on a
// 50/50/50 planet.
func tkRace() Race {
	r := pgRace()
	r.PRT = PRTJackOfAllTrades
	r.GrowthRate = 15
	return r
}

// tkLab is a two-player TK game: player 0 attacks at tech 26 in every
// field, player 1 owns the targets at energy tech `energy`. Both research
// biotechnology, so no level-up changes the defenses during a turn.
type tkLab struct {
	t *testing.T
	g Game
}

func newTKLab(t *testing.T, energy int) *tkLab {
	l := &tkLab{t: t}
	l.g.Rules = ElegyRules()
	for range 2 {
		pl := Player{Race: tkRace(), Research: ResearchState{Current: Biotech, Next: NextSameField}}
		pl.Relations = []Relation{RelationEnemy, RelationEnemy}
		pl.Plans = []BattlePlan{{Attack: AttackEnemies, Tactic: TacticMaximizeDamage}}
		l.g.Players = append(l.g.Players, pl)
	}
	for f := range NumFields {
		l.g.Players[0].Research.Levels[f] = MaxTechLevel
	}
	l.g.Players[1].Research.Levels[Energy] = energy
	l.g.Defenses = Components().Defenses()
	return l
}

// planet adds a planet of owner at x with population pop and returns its
// index.
func (l *tkLab) planet(owner, x, pop int) int {
	l.g.Planets = append(l.g.Planets, Planet{
		ID: len(l.g.Planets), Pos: Point{x, 0}, Owner: owner, Population: pop,
		Env: [3]int{50, 50, 50}, OrigEnv: [3]int{50, 50, 50},
	})
	return len(l.g.Planets) - 1
}

// design adds a catalogue design and returns its index.
func (l *tkLab) design(hull string, fills ...SlotFill) int {
	l.t.Helper()
	d, err := Components().NewDesign(hull, hull, fills)
	if err != nil {
		l.t.Fatal(err)
	}
	l.g.Designs = append(l.g.Designs, d)
	return len(l.g.Designs) - 1
}

// fleet adds a fleet of owner at planet pi with plan and returns its index.
func (l *tkLab) fleet(owner, pi, plan int, stacks ...Stack) int {
	id := len(l.g.Fleets) + 1
	l.g.Fleets = append(l.g.Fleets, Fleet{ID: id, Owner: owner, Pos: l.g.Planets[pi].Pos, Stacks: stacks, Plan: plan})
	return len(l.g.Fleets) - 1
}

// bomber is a B-17 Bomber with one bomb, so a stack of n ships carries n
// bombs.
func (l *tkLab) bomber(bomb string) int {
	return l.design("B-17 Bomber", SlotFill{0, "Quick Jump 5", 2}, SlotFill{1, bomb, 1})
}

func TestConfirmedBombingVectors(t *testing.T) {
	// TK-001 run2, TK-003, TK-005 (T-6..T-18), on P' directly.
	tests := []struct {
		name        string
		bomb        string
		n           int // ships of one bomb each
		energy      int
		pop         int
		mines, facs int
		defs        int
		draws       []int // nil: no draw may be made
		wantPop     int
		wantMines   int
		wantFacs    int
		wantDefs    int
	}{
		{"T-12 10 Cherry", "Cherry Bomb", 10, 3, 920, 0, 0, 0, nil, 690, 0, 0, 0},
		{"T-10 Lady Finger minimum", "Lady Finger Bomb", 1, 3, 10, 0, 0, 0, []int{999}, 7, 0, 0, 0},
		{"T-11 LBU-17 no minimum", "LBU-17 Bomb", 1, 3, 10, 0, 0, 0, []int{999}, 9, 0, 0, 0},
		{"T-14 20 Peerless multiply", "Peerless Bomb", 20, 3, 1150, 0, 0, 0, nil, 412, 0, 0, 0},
		{"T-14 Peerless never empty", "Peerless Bomb", 20, 3, 1, 0, 0, 0, nil, 1, 0, 0, 0},
		{"T-15 LBU-32 installations", "LBU-32 Bomb", 1, 3, 1000, 30, 30, 0, nil, 997, 16, 16, 0},
		{"T-6 20 Cherry, 100 SDI", "Cherry Bomb", 20, 3, 1000, 0, 0, 100, nil, 666, 0, 0, 0},
		{"T-7 only 4 SDI count", "Cherry Bomb", 20, 3, 100, 0, 0, 100, nil, 42, 0, 0, 0},
		{"T-9 20 Smart half coverage", "Smart Bomb", 20, 3, 1000, 0, 0, 100, nil, 812, 0, 0, 100},
		{"T-8 Missile Batteries", "Cherry Bomb", 20, 5, 1000, 0, 0, 100, nil, 777, 0, 0, 0},
		{"T-7 Missile Batteries", "Cherry Bomb", 20, 5, 100, 0, 0, 100, []int{999}, 45, 0, 0, 0},
		{"T-9 Missile Batteries", "Smart Bomb", 20, 5, 1000, 0, 0, 100, nil, 846, 0, 0, 100},
		{"T-17 Orbital Construction Module", "", 1, 3, 100, 0, 0, 0, nil, 80, 0, 0, 0},
		{"T-18 Multi Contained Munition", "Multi Contained Munition", 1, 3, 100, 10, 0, 0, nil, 97, 5, 0, 0},
		// TK-004/TK-007 random cases: both outcomes.
		{"T-11 Hush-a-Boom, draw ≤ 500", "Hush-a-Boom", 1, 3, 50, 0, 0, 0, []int{500}, 48, 0, 0, 0},
		{"T-11 Hush-a-Boom, draw > 500", "Hush-a-Boom", 1, 3, 50, 0, 0, 0, []int{501}, 49, 0, 0, 0},
		{"T-15 LBU-17, factory draw < 10", "LBU-17 Bomb", 1, 3, 1000, 20, 10, 0, []int{9}, 998, 10, 4, 0},
		{"T-15 LBU-17, factory draw ≥ 10", "LBU-17 Bomb", 1, 3, 1000, 20, 10, 0, []int{10}, 998, 9, 5, 0},
	}
	for _, tt := range tests {
		l := newTKLab(t, tt.energy)
		pi := l.planet(1, 0, tt.pop)
		p := &l.g.Planets[pi]
		p.Mines, p.Factories, p.Defenses = tt.mines, tt.facs, tt.defs
		var d int
		switch tt.bomb {
		case "":
			d = l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1}, SlotFill{1, "Orbital Construction Module", 1})
		case "Multi Contained Munition":
			d = l.design("Frigate", SlotFill{0, "Quick Jump 5", 1}, SlotFill{2, "Multi Contained Munition", 1})
		default:
			d = l.bomber(tt.bomb)
		}
		l.fleet(0, pi, 0, Stack{Design: d, Count: tt.n})
		var rng Rand = panicRand{}
		if tt.draws != nil {
			rng = &seqRand{draws: tt.draws}
		}
		bombing(&l.g, rng)
		p = &l.g.Planets[pi]
		got := [4]int{p.Population, p.Mines, p.Factories, p.Defenses}
		if want := [4]int{tt.wantPop, tt.wantMines, tt.wantFacs, tt.wantDefs}; got != want {
			t.Errorf("%s: pop/mines/factories/defenses %v, want %v", tt.name, got, want)
		}
	}
}

func TestConfirmedSmartThenNormal(t *testing.T) {
	// T-13: 10 Smart + 10 Cherry on 921 → 606; smart first, then 25% of
	// the rest.
	l := newTKLab(t, 3)
	pi := l.planet(1, 0, 921)
	l.fleet(0, pi, 0, Stack{Design: l.bomber("Smart Bomb"), Count: 10}, Stack{Design: l.bomber("Cherry Bomb"), Count: 10})
	bombing(&l.g, panicRand{})
	if got := l.g.Planets[pi].Population; got != 606 {
		t.Errorf("pop %d, want 606", got)
	}
}

func TestConfirmedRetroBombs(t *testing.T) {
	// T-16: 3 Retro on 55/47/52, original 50/50/50 → 52/50/50, original
	// kept.
	l := newTKLab(t, 3)
	pi := l.planet(1, 0, 100)
	l.g.Planets[pi].Env = [3]int{55, 47, 52}
	l.fleet(0, pi, 0, Stack{Design: l.bomber("Retro Bomb"), Count: 3})
	bombing(&l.g, panicRand{})
	p := l.g.Planets[pi]
	if p.Env != [3]int{52, 50, 50} || p.OrigEnv != [3]int{50, 50, 50} {
		t.Errorf("env %v original %v, want 52/50/50 and 50/50/50", p.Env, p.OrigEnv)
	}
}

func TestConfirmedWhoBombs(t *testing.T) {
	// T-3, T-19, TK-005, TK-006, TK-003: 10 Cherry on 920 bomb to 690
	// exactly when a plan covers the owner and there is no starbase.
	tests := []struct {
		name     string
		relation Relation
		plan     BattlePlan
		fort     bool
		want     int
	}{
		{"enemies, nobody", RelationEnemy, BattlePlan{Attack: AttackNobody}, false, 920},
		{"enemies, attacker itself only", RelationEnemy, BattlePlan{Attack: AttackPlayer, Player: 0}, false, 920},
		{"enemies, player 1 only", RelationEnemy, BattlePlan{Attack: AttackPlayer, Player: 1}, false, 690},
		{"enemies, everyone", RelationEnemy, BattlePlan{Attack: AttackEveryone}, false, 690},
		{"enemies, enemies", RelationEnemy, BattlePlan{Attack: AttackEnemies}, false, 690},
		{"neutral, enemies", RelationNeutral, BattlePlan{Attack: AttackEnemies}, false, 920},
		{"neutral, player 1 only", RelationNeutral, BattlePlan{Attack: AttackPlayer, Player: 1}, false, 690},
		{"neutral, everyone", RelationNeutral, BattlePlan{Attack: AttackEveryone}, false, 690},
		{"friends, everyone", RelationFriend, BattlePlan{Attack: AttackEveryone}, false, 690},
		{"T-3 bare Orbital Fort", RelationEnemy, BattlePlan{Attack: AttackEnemies}, true, 920},
	}
	for _, tt := range tests {
		l := newTKLab(t, 3)
		l.g.Players[0].Relations = []Relation{RelationFriend, tt.relation}
		l.g.Players[0].Plans = []BattlePlan{tt.plan}
		pi := l.planet(1, 0, 920)
		if tt.fort {
			l.g.Planets[pi].StarbaseHull = 1
		}
		l.fleet(0, pi, 0, Stack{Design: l.bomber("Cherry Bomb"), Count: 10})
		bombing(&l.g, panicRand{})
		if got := l.g.Planets[pi].Population; got != tt.want {
			t.Errorf("%s: pop %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestConfirmedFleetsBombAsOne(t *testing.T) {
	// T-20, TK-005: every fleet of the player at the planet joins one
	// pass once one fleet's plan covers the owner, whatever the order and
	// even if that fleet has no bombs.
	tests := []struct {
		name  string
		first int // plan of the first fleet; the second has the other
		laser bool
	}{
		{"first attacks, second nobody", 1, false},
		{"first nobody, second attacks", 0, false},
		{"Laser Frigate attacks, bombers nobody", 1, true},
	}
	for _, tt := range tests {
		l := newTKLab(t, 3)
		l.g.Players[0].Plans = []BattlePlan{{Attack: AttackNobody}, {Attack: AttackEnemies}}
		pi := l.planet(1, 0, 920)
		cherry := l.bomber("Cherry Bomb")
		if tt.laser {
			frig := l.design("Frigate", SlotFill{0, "Quick Jump 5", 1}, SlotFill{2, "Laser", 1})
			l.fleet(0, pi, 1, Stack{Design: frig, Count: 1})
			l.fleet(0, pi, 0, Stack{Design: cherry, Count: 10})
		} else {
			l.fleet(0, pi, tt.first, Stack{Design: cherry, Count: 6})
			l.fleet(0, pi, 1-tt.first, Stack{Design: cherry, Count: 4})
		}
		bombing(&l.g, panicRand{})
		if got := l.g.Planets[pi].Population; got != 690 {
			t.Errorf("%s: pop %d, want 690 (one pass of 10 Cherry)", tt.name, got)
		}
	}
}

// groundLab is a planet of player 1 with population pop after growth
// and defs defenses, attacked by drops.
func groundLab(t *testing.T, energy, pop, defs int) (*tkLab, int) {
	l := newTKLab(t, energy)
	pi := l.planet(1, 0, pop)
	p := &l.g.Planets[pi]
	p.Defenses, p.Mines, p.Factories, p.GrowthCarry, p.HasScanner = defs, 20, 15, 42, true
	return l, pi
}

func TestConfirmedGroundCombat(t *testing.T) {
	// TK-002 run2 and TK-003 (T-21..T-26): owner and population after the
	// drops; a capture keeps mines, factories and carry and loses
	// defenses and the scanner.
	tests := []struct {
		name               string
		energy, pop, defs  int
		troops             int
		wantOwner, wantPop int
	}{
		{"T-21 100 vs 100", 3, 100, 0, 100, 0, 9},
		{"T-22 tie goes to the attacker", 3, 110, 0, 100, 0, 1},
		{"T-23 100 vs 200", 3, 200, 0, 100, 1, 90},
		{"T-25 600 vs 500, 20 SDI", 3, 500, 20, 600, 0, 72},
		{"T-26 300 vs 200, 10 SDI", 3, 200, 10, 300, 0, 106},
		{"T-25 600 vs 500, 20 Missile Batteries", 5, 500, 20, 600, 1, 5},
	}
	for _, tt := range tests {
		l, pi := groundLab(t, tt.energy, tt.pop, tt.defs)
		l.g.resolveQueue([]drop{{pi, 0, tt.troops}}, &seqRand{}, map[int]bool{})
		p := l.g.Planets[pi]
		if p.Owner != tt.wantOwner || p.Population != tt.wantPop {
			t.Errorf("%s: owner %d pop %d, want %d and %d", tt.name, p.Owner, p.Population, tt.wantOwner, tt.wantPop)
		}
		if p.Owner == 0 {
			if p.Mines != 20 || p.Factories != 15 || p.GrowthCarry != 42 || p.Defenses != 0 || p.HasScanner {
				t.Errorf("%s: captured planet mines %d factories %d carry %d defenses %d scanner %v, want 20, 15, 42, 0, none",
					tt.name, p.Mines, p.Factories, p.GrowthCarry, p.Defenses, p.HasScanner)
			}
		}
	}
}

func TestConfirmedSeveralPlayersDrop(t *testing.T) {
	// T-32 (LEGACY BUG): colony ships of 25 and 12 units on an unowned
	// planet. A lower-index winner is not reduced; a higher-index winner
	// is; a tie lands nobody.
	tests := []struct {
		name               string
		p0, p1             int
		wantOwner, wantPop int
	}{
		{"25 vs 12", 25, 12, 0, 25},
		{"12 vs 25", 12, 25, 1, 12},
		{"25 vs 25", 25, 25, NoOwner, 0},
	}
	for _, tt := range tests {
		l := newTKLab(t, 3)
		pi := l.planet(NoOwner, 0, 0)
		l.g.resolveQueue([]drop{{pi, 0, tt.p0}, {pi, 1, tt.p1}}, panicRand{}, map[int]bool{})
		p := l.g.Planets[pi]
		if p.Owner != tt.wantOwner || p.Population != tt.wantPop {
			t.Errorf("%s: owner %d pop %d, want %d and %d", tt.name, p.Owner, p.Population, tt.wantOwner, tt.wantPop)
		}
	}
}

// colonyShip is the TK colony ship: Colony Ship hull, Long Hump 6 and a
// Colonization Module.
func (l *tkLab) colonyShip() int {
	return l.design("Colony Ship", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Colonization Module", 1})
}

func TestConfirmedColonyMinerals(t *testing.T) {
	// T-30: the colony ship leaves ⌊3/4⌋ of its mineral cost for the
	// owner: 18/6/17 kT at tech 3 and 4/1/5 at tech 26. T-32 25 vs 25:
	// both ships' minerals are delivered though nobody lands.
	for _, tt := range []struct {
		tech int
		want Minerals
	}{{3, Minerals{18, 6, 17}}, {26, Minerals{4, 1, 5}}} {
		l := newTKLab(t, 3)
		for f := range NumFields {
			l.g.Players[0].Research.Levels[f] = tt.tech
		}
		pi := l.planet(NoOwner, 0, 0)
		fi := l.fleet(0, pi, 0, Stack{Design: l.colonyShip(), Count: 1})
		l.g.Fleets[fi].Cargo.Colonists = 25
		l.g.Fleets[fi].Task = Task{Kind: TaskColonize}
		queue, _ := l.g.unloadPhase(l.g.phaseStart())
		l.g.resolveQueue(queue, panicRand{}, map[int]bool{})
		p := l.g.Planets[pi]
		if p.Surface != tt.want || p.Owner != 0 || p.Population != 25 || len(l.g.Fleets) != 0 {
			t.Errorf("tech %d: surface %v owner %d pop %d fleets %d, want %v, 0, 25, consumed",
				tt.tech, p.Surface, p.Owner, p.Population, len(l.g.Fleets), tt.want)
		}
	}

	l := newTKLab(t, 3)
	for f := range NumFields {
		l.g.Players[1].Research.Levels[f] = 3
	}
	pi := l.planet(NoOwner, 0, 0)
	for p := range 2 {
		fi := l.fleet(p, pi, 0, Stack{Design: l.colonyShip(), Count: 1})
		l.g.Fleets[fi].Cargo.Colonists = 25
		l.g.Fleets[fi].Task = Task{Kind: TaskColonize}
	}
	queue, _ := l.g.unloadPhase(l.g.phaseStart())
	l.g.resolveQueue(queue, panicRand{}, map[int]bool{})
	if p := l.g.Planets[pi]; p.Surface != (Minerals{22, 7, 22}) || p.Owner != NoOwner || len(l.g.Fleets) != 0 {
		t.Errorf("25 vs 25: surface %v owner %d fleets %d, want 22/7/22, unowned, both consumed", p.Surface, p.Owner, len(l.g.Fleets))
	}
}

// turnLab makes l's game a whole-turn game: every planet gets deposits,
// and fleets moving to a planet use a fuel-free engine.
func (l *tkLab) turn(rng Rand) TurnResult {
	l.t.Helper()
	r, err := GenerateTurn(withRules(l.g), nil, rng)
	if err != nil {
		l.t.Fatal(err)
	}
	return r
}

// arriving adds a fleet of owner 20 ly from planet pi, moving to it at
// warp 6 with task, carrying colonists.
func (l *tkLab) arriving(owner, pi, design, colonists int, task Task) int {
	p := l.g.Planets[pi]
	id := len(l.g.Fleets) + 1
	l.g.Fleets = append(l.g.Fleets, Fleet{
		ID: id, Owner: owner, Pos: Point{p.Pos.X, p.Pos.Y + 20}, Fuel: 1000,
		Stacks:    []Stack{{Design: design, Count: 1}},
		Cargo:     Cargo{Colonists: colonists},
		Waypoints: []Waypoint{{Pos: p.Pos, Warp: 6, Target: TargetPlanet, ID: p.ID, Task: task}},
	})
	return id
}

var (
	unloadColonists = Task{Kind: TaskTransport, Transport: [NumCargo]Transport{CargoColonists: {Action: UnloadAll}}}
	colonizeTask    = Task{Kind: TaskColonize}
)

func fleetByID(g *Game, id int) *Fleet {
	if i := g.fleetIndex(id); i >= 0 {
		return &g.Fleets[i]
	}
	return nil
}

func TestConfirmedTakeoverTiming(t *testing.T) {
	// TK-002: whole turns. Growth of 87 with carry 37 is 100 with carry 42.
	l := newTKLab(t, 3)
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	colony := l.colonyShip()
	cherry := l.bomber("Cherry Bomb")

	control := l.planet(1, 0, 87)
	l.g.Planets[control].GrowthCarry = 37

	// T-5: 100 in orbit unload before growth: 100 vs 87 → 20, which grow
	// as player 0's colony to 23.
	t5 := l.planet(1, 100, 87)
	f5 := l.fleet(0, t5, 0, Stack{Design: freighter, Count: 1})
	l.g.Fleets[f5].Cargo.Colonists = 100
	l.g.Fleets[f5].Task = unloadColonists

	// T-21: 100 arriving after growth: 100 vs 100 → 9.
	t21 := l.planet(1, 200, 87)
	l.g.Planets[t21].GrowthCarry = 37
	l.arriving(0, t21, freighter, 100, unloadColonists)

	// T-1: a colony ship in orbit colonizes before growth (25 → 28); one
	// arriving colonizes after growth and keeps exactly 25.
	t1 := l.planet(NoOwner, 300, 0)
	f1 := l.fleet(0, t1, 0, Stack{Design: colony, Count: 1})
	l.g.Fleets[f1].Cargo.Colonists = 25
	l.g.Fleets[f1].Task = colonizeTask
	t1b := l.planet(NoOwner, 400, 0)
	l.arriving(0, t1b, colony, 25, colonizeTask)

	// T-4: 4 Cherry in orbit bomb 9 → 10 empty (minimum 12 units), then
	// an arriving transport colonizes it with all 50.
	t4 := l.planet(1, 500, 9)
	l.g.Planets[t4].Mines, l.g.Planets[t4].Factories = 20, 15
	l.fleet(0, t4, 0, Stack{Design: cherry, Count: 4})
	l.arriving(0, t4, freighter, 50, unloadColonists)

	// T-4 control: a planet unowned at the start refuses; the freighter
	// keeps 50.
	t4c := l.planet(NoOwner, 600, 0)
	f4c := l.arriving(0, t4c, freighter, 50, unloadColonists)

	// T-28: a bare Orbital Fort refuses the drop.
	t28 := l.planet(1, 700, 87)
	l.g.Planets[t28].StarbaseHull = 1
	f28 := l.arriving(0, t28, freighter, 100, unloadColonists)

	r := l.turn(&seqRand{})
	g := r.Game
	check := func(name string, pi, owner, pop int) {
		t.Helper()
		if p := g.Planets[pi]; p.Owner != owner || p.Population != pop {
			t.Errorf("%s: owner %d pop %d, want %d and %d", name, p.Owner, p.Population, owner, pop)
		}
	}
	check("control", control, 1, 100)
	if c := g.Planets[control].GrowthCarry; c != 42 {
		t.Errorf("control carry %d, want 42", c)
	}
	check("T-5 in orbit", t5, 0, 23)
	check("T-21 arriving", t21, 0, 9)
	check("T-1 in orbit", t1, 0, 28)
	check("T-1 arriving", t1b, 0, 25)
	check("T-4 bombed then colonized", t4, 0, 50)
	if p := g.Planets[t4]; p.Mines != 0 || p.Factories != 0 {
		t.Errorf("T-4: mines %d factories %d, want 0 and 0 (4 Cherry destroy 40)", p.Mines, p.Factories)
	}
	check("T-4 control", t4c, NoOwner, 0)
	if f := fleetByID(&g, f4c); f == nil || f.Cargo.Colonists != 50 {
		t.Errorf("T-4 control: freighter should keep 50")
	}
	check("T-28 fort", t28, 1, 100)
	if f := fleetByID(&g, f28); f == nil || f.Cargo.Colonists != 100 {
		t.Errorf("T-28: freighter should keep 100")
	}
}

func TestConfirmedFriendInvaded(t *testing.T) {
	// T-29: unloading on a friend's planet invades it like an enemy's.
	l, pi := groundLab(t, 3, 100, 0)
	l.g.Players[0].Relations = []Relation{RelationFriend, RelationFriend}
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	fi := l.fleet(0, pi, 0, Stack{Design: freighter, Count: 1})
	l.g.Fleets[fi].Cargo.Colonists = 100
	l.g.Fleets[fi].Task = unloadColonists
	queue, _ := l.g.unloadPhase(l.g.phaseStart())
	l.g.resolveQueue(queue, &seqRand{}, map[int]bool{})
	if p := l.g.Planets[pi]; p.Owner != 0 || p.Population != 9 {
		t.Errorf("owner %d pop %d, want 0 and 9", p.Owner, p.Population)
	}
}

func TestPredictionBombingOrder(t *testing.T) {
	// #34 "Bombing order": fleet order (owner, then fleet number), one
	// pass per player and planet at its first qualifying fleet, lower
	// players first. Player 2's later fleet comes before player 1's
	// passes in fleet-id order but bombs after them.
	l := newTKLab(t, 3)
	l.g.Players = append(l.g.Players, l.g.Players[0])
	for p := range l.g.Players {
		l.g.Players[p].Relations = []Relation{RelationEnemy, RelationEnemy, RelationEnemy}
	}
	a := l.planet(0, 0, 920)
	b := l.planet(0, 10, 920)
	cherry := l.bomber("Cherry Bomb")
	l.fleet(2, a, 0, Stack{Design: cherry, Count: 10})
	l.fleet(1, b, 0, Stack{Design: cherry, Count: 10})
	l.fleet(1, a, 0, Stack{Design: cherry, Count: 10})
	ev := bombing(&l.g, &seqRand{draws: []int{999}})
	var got [][2]int
	for _, e := range ev {
		got = append(got, [2]int{e.Player, e.Planet})
	}
	want := [][2]int{{1, 1}, {1, 0}, {2, 0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bombing order %v, want %v", got, want)
	}
	// Planet a: 920 → 690 by player 1, then 25% of 690 (172.5, the
	// draw rounding down) by player 2.
	if pop := l.g.Planets[a].Population; pop != 518 {
		t.Errorf("planet a pop %d, want 518", pop)
	}
}

func TestPredictionPhaseStartOwnership(t *testing.T) {
	// #34 "At the start of this phase": a planet colonized by player 1
	// before movement and bombed empty the same year counts as owned for
	// the after-movement drop, so player 0's arriving freighter colonizes
	// it.
	l := newTKLab(t, 3)
	l.g.Players[1].Research.Levels = [NumFields]int{26, 26, 26, 26, 26, 26}
	pi := l.planet(NoOwner, 0, 0)
	ci := l.fleet(1, pi, 0, Stack{Design: l.colonyShip(), Count: 1})
	l.g.Fleets[ci].Cargo.Colonists = 25
	l.g.Fleets[ci].Task = colonizeTask
	// Two Orbital Construction Modules kill at least 40 units.
	ocm := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1}, SlotFill{1, "Orbital Construction Module", 1})
	l.fleet(0, pi, 0, Stack{Design: ocm, Count: 2})
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	l.arriving(0, pi, freighter, 50, unloadColonists)

	g := l.turn(&seqRand{}).Game
	if p := g.Planets[pi]; p.Owner != 0 || p.Population != 50 {
		t.Errorf("owner %d pop %d, want 0 and 50", p.Owner, p.Population)
	}

	// A planet unowned when the after-movement phase began refuses.
	l = newTKLab(t, 3)
	pi = l.planet(NoOwner, 0, 0)
	freighter = l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	fi := l.arriving(0, pi, freighter, 50, unloadColonists)
	g = l.turn(&seqRand{}).Game
	if f := fleetByID(&g, fi); g.Planets[pi].Owner != NoOwner || f == nil || f.Cargo.Colonists != 50 {
		t.Errorf("unowned planet should refuse the unload")
	}
}

func TestPredictionUnloadAmounts(t *testing.T) {
	// #34 "Unload and load amounts": unload exactly v moves min(v, C),
	// once; on the owner's own planet colonists join the population at
	// once with no cap, before growth when before movement.
	l := newTKLab(t, 3)
	pi := l.planet(0, 0, 100)
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	fi := l.fleet(0, pi, 0, Stack{Design: freighter, Count: 1})
	f := &l.g.Fleets[fi]
	f.Cargo = Cargo{Minerals: Minerals{30, 0, 5}, Colonists: 40}
	f.Task = Task{Kind: TaskTransport, Transport: [NumCargo]Transport{
		Ironium:        {Action: UnloadExactly, Amount: 10},
		Germanium:      {Action: UnloadExactly, Amount: 10},
		CargoColonists: {Action: UnloadAll},
	}}
	l.g.unloadPhase(l.g.phaseStart())
	p, f := l.g.Planets[pi], &l.g.Fleets[fi]
	if p.Population != 140 || p.Surface != (Minerals{10, 0, 5}) || f.Cargo != (Cargo{Minerals: Minerals{20, 0, 0}}) {
		t.Errorf("pop %d surface %v cargo %+v, want 140, 10/0/5, 20/0/0 aboard", p.Population, p.Surface, f.Cargo)
	}
	if f.Task.Transport != ([NumCargo]Transport{}) {
		t.Errorf("unload actions should clear after running: %+v", f.Task.Transport)
	}

	// Whole turn: an own-planet unload before movement grows this year
	// (87 + 13 → 100 + 15 = 115); an arriving one does not.
	l = newTKLab(t, 3)
	a := l.planet(0, 0, 87)
	b := l.planet(0, 100, 100)
	freighter = l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	fa := l.fleet(0, a, 0, Stack{Design: freighter, Count: 1})
	l.g.Fleets[fa].Cargo.Colonists = 13
	l.g.Fleets[fa].Task = unloadColonists
	l.arriving(0, b, freighter, 13, unloadColonists)
	g := l.turn(&seqRand{}).Game
	if pa, pb := g.Planets[a].Population, g.Planets[b].Population; pa != 115 || pb != 128 {
		t.Errorf("before movement %d, after movement %d; want 115 and 115+13=128", pa, pb)
	}
}

func TestPredictionGroundStrengthTraits(t *testing.T) {
	// War Monger k = 165, Inner Strength defender D = 2P, Alternate
	// Reality drops refused (TAKEOVER.md, BINARY-ONLY).
	l, pi := groundLab(t, 3, 100, 0)
	l.g.Players[0].Race.PRT = PRTWarMonger
	l.g.resolveQueue([]drop{{pi, 0, 100}}, &seqRand{}, map[int]bool{})
	// Strength 165 vs 100: lands 100·65/165 = 39.
	if p := l.g.Planets[pi]; p.Owner != 0 || p.Population != 39 {
		t.Errorf("War Monger: owner %d pop %d, want 0 and 39", p.Owner, p.Population)
	}

	l, pi = groundLab(t, 3, 100, 0)
	l.g.Players[1].Race.PRT = PRTInnerStrength
	l.g.resolveQueue([]drop{{pi, 0, 100}}, &seqRand{}, map[int]bool{})
	// D = 200 > 110: the planet loses ⌊100·110/200⌋ = 55.
	if p := l.g.Planets[pi]; p.Owner != 1 || p.Population != 45 {
		t.Errorf("Inner Strength: owner %d pop %d, want 1 and 45", p.Owner, p.Population)
	}

	l, pi = groundLab(t, 3, 100, 0)
	l.g.Players[0].Race.PRT = PRTAlternateReality
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	fi := l.fleet(0, pi, 0, Stack{Design: freighter, Count: 1})
	l.g.Fleets[fi].Cargo.Colonists = 100
	l.g.Fleets[fi].Task = unloadColonists
	queue, ev := l.g.unloadPhase(l.g.phaseStart())
	if len(queue) != 0 || len(ev) != 1 || ev[0].Kind != EventDropRefused || l.g.Fleets[fi].Cargo.Colonists != 100 {
		t.Errorf("Alternate Reality drop should be refused: queue %v events %v", queue, ev)
	}
}

func TestPredictionCaptureTech(t *testing.T) {
	// #34 "Capture": the winner of an owned planet makes one tech attempt
	// against the old owner's levels; a colonization of an unowned
	// planet makes none, and a player that already gained makes no draws.
	l, pi := groundLab(t, 3, 100, 0)
	l.g.Players[0].Research.Levels = [NumFields]int{}
	l.g.Players[1].Research.Levels[Weapons] = 4
	draws := []int{50} // the 50% roll passes
	draws = append(draws, make([]int, 13)...)
	draws = append(draws, Energy, Weapons) // energy: 0 < 3, gained
	rng := &seqRand{draws: draws}
	gained := map[int]bool{}
	ev := l.g.resolveQueue([]drop{{pi, 0, 100}}, rng, gained)
	if !gained[0] || len(rng.draws) != 1 || l.g.Players[0].Research.Accumulated[Energy] == 0 {
		t.Errorf("capture should gain energy research: gained %v, draws left %v, events %v", gained, rng.draws, ev)
	}

	l, pi = groundLab(t, 3, 100, 0)
	l.g.resolveQueue([]drop{{pi, 0, 100}}, panicRand{}, map[int]bool{0: true})
	if l.g.Planets[pi].Owner != 0 {
		t.Errorf("capture failed")
	}

	l = newTKLab(t, 3)
	pi = l.planet(NoOwner, 0, 0)
	l.g.resolveQueue([]drop{{pi, 0, 25}}, panicRand{}, map[int]bool{})
	if l.g.Planets[pi].Owner != 0 {
		t.Errorf("colonization failed")
	}
}

func TestConfirmedColonizeTriedOnce(t *testing.T) {
	// TK-113 (stars-elegy #37): players 0 and 2 drop 150 each on a planet
	// of 100. That is a tie, so the planet is emptied and nobody lands.
	// Player 0's colony ship there has already failed ("planet owned"):
	// it keeps its 25 colonists and its task is cleared, whether it was
	// in orbit or arrived. The planet stays unowned with no minerals.
	for _, arrive := range []bool{false, true} {
		l := newTKLab(t, 3)
		l.g.Players = append(l.g.Players, l.g.Players[0])
		for p := range l.g.Players {
			l.g.Players[p].Relations = []Relation{RelationEnemy, RelationEnemy, RelationEnemy}
		}
		pi := l.planet(1, 0, 100)
		freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
		colony := l.colonyShip()
		var ci int
		if arrive {
			ci = l.arriving(0, pi, colony, 25, colonizeTask)
		} else {
			fi := l.fleet(0, pi, 0, Stack{Design: colony, Count: 1})
			l.g.Fleets[fi].Cargo.Colonists = 25
			l.g.Fleets[fi].Task = colonizeTask
			ci = l.g.Fleets[fi].ID
		}
		for _, p := range []int{0, 2} {
			if arrive {
				l.arriving(p, pi, freighter, 150, unloadColonists)
			} else {
				fi := l.fleet(p, pi, 0, Stack{Design: freighter, Count: 1})
				l.g.Fleets[fi].Cargo.Colonists = 150
				l.g.Fleets[fi].Task = unloadColonists
			}
		}
		g := l.turn(&seqRand{}).Game
		p := g.Planets[pi]
		f := fleetByID(&g, ci)
		if p.Owner != NoOwner || p.Population != 0 || p.Surface != (Minerals{}) {
			t.Errorf("arrive=%v: owner %d pop %d surface %v, want unowned, empty, no minerals", arrive, p.Owner, p.Population, p.Surface)
		}
		if f == nil || f.Cargo.Colonists != 25 || f.Task != (Task{}) {
			t.Errorf("arrive=%v: colony ship %+v, want kept with 25 colonists and no task", arrive, f)
		}
	}
}

func TestPredictionEmptiedPlanet(t *testing.T) {
	// TAKEOVER.md "Capture": starvation empties a planet; a Claim
	// Adjuster's lost planet returns to its original environment
	// (CONFIRMED, TK-116); a new colony gets the default queue less AR's first
	// three or CA's fifth and sixth items, and the default leftover
	// setting.
	l := newTKLab(t, 3)
	pi := l.planet(1, 0, 1)
	l.g.Planets[pi].Env = [3]int{95, 95, 95} // hostile
	l.g.Planets[pi].Defenses = 5
	g := l.turn(&seqRand{}).Game
	if p := g.Planets[pi]; p.Owner != NoOwner || p.Defenses != 0 {
		t.Errorf("starved planet: owner %d defenses %d, want unowned and 0", p.Owner, p.Defenses)
	}

	l = newTKLab(t, 3)
	l.g.Players[1].Race.PRT = PRTClaimAdjuster
	pi = l.planet(1, 0, 10)
	l.g.Planets[pi].Env = [3]int{60, 60, 60}
	l.g.emptyPlanet(pi)
	if env := l.g.Planets[pi].Env; env != [3]int{50, 50, 50} {
		t.Errorf("CA lost planet env %v, want the original 50/50/50", env)
	}

	queue := []QueueItem{q(ItemMine, 1, 0), q(ItemFactory, 2, 0), q(ItemDefenses, 3, 0), q(ItemMine, 4, 0), q(ItemFactory, 5, 0), q(ItemDefenses, 6, 0), q(ItemMine, 7, 0)}
	counts := func(items []QueueItem) []int {
		var c []int
		for _, it := range items {
			c = append(c, it.Count)
		}
		return c
	}
	for _, tt := range []struct {
		prt  PRT
		want []int
	}{
		{PRTJackOfAllTrades, []int{1, 2, 3, 4, 5, 6, 7}},
		{PRTAlternateReality, []int{4, 5, 6, 7}},
		{PRTClaimAdjuster, []int{1, 2, 3, 4, 7}},
	} {
		l = newTKLab(t, 3)
		l.g.Players[0].Race.PRT = tt.prt
		l.g.Players[0].DefaultQueue = queue
		l.g.Players[0].DefaultLeftoverOnly = true
		pi = l.planet(NoOwner, 0, 0)
		l.g.newColony(pi, 0, 25)
		p := l.g.Planets[pi]
		if got := counts(p.Queue); !reflect.DeepEqual(got, tt.want) || !p.HasQueue || !p.LeftoverOnly {
			t.Errorf("PRT %v: queue %v leftover %v, want %v and true", tt.prt, got, p.LeftoverOnly, tt.want)
		}
	}
}

func TestMeasuredEmptyDefaultQueueNoQueue(t *testing.T) {
	// WU-CAP (MEASURED): a capture by a player with no default queue left
	// the planet without a queue, so its resources all went to research.
	// An Alternate Reality default of three items is skipped whole, the
	// same way (inferred: KERNEL.md says a zero-item queue does not arise
	// in play).
	for _, tt := range []struct {
		prt   PRT
		queue []QueueItem
	}{
		{PRTJackOfAllTrades, nil},
		{PRTAlternateReality, []QueueItem{{Kind: ItemMine, Count: 1}, {Kind: ItemFactory, Count: 2}, {Kind: ItemDefenses, Count: 3}}},
	} {
		l := newTKLab(t, 3)
		l.g.Players[0].Race.PRT = tt.prt
		l.g.Players[0].DefaultQueue = tt.queue
		pi := l.planet(NoOwner, 0, 0)
		l.g.newColony(pi, 0, 25)
		if p := l.g.Planets[pi]; p.HasQueue || len(p.Queue) != 0 {
			t.Errorf("PRT %v: queue %v %v, want none", tt.prt, p.HasQueue, p.Queue)
		}
		res := 100
		if r, _ := l.g.PlanetProduction(pi, ProductionInput{Resources: res, ResearchBudget: 15}); r != res {
			t.Errorf("PRT %v: research %d, want all %d", tt.prt, r, res)
		}
	}
}

func TestPredictionMineLossClamp(t *testing.T) {
	// #34: I = 1 on one factory, one defense and one mine. Both
	// draws round up, so 2 installations die, and the rest (−1) takes no
	// mine.
	l := newTKLab(t, 3)
	pi := l.planet(1, 0, 10)
	p := &l.g.Planets[pi]
	p.Factories, p.Defenses, p.Mines = 1, 1, 1
	pass := bombPass{I: 1, pi: 1}
	l.g.bombPlanet(pi, 0, pass, &seqRand{draws: []int{0, 0}})
	p = &l.g.Planets[pi]
	if p.Factories != 0 || p.Defenses != 0 || p.Mines != 1 {
		t.Errorf("I = 1: factories %d defenses %d mines %d, want 0, 0, 1", p.Factories, p.Defenses, p.Mines)
	}
}

func TestPredictionForeignMinerals(t *testing.T) {
	// #34: minerals unloaded on another player's planet or an unowned one
	// join its surface and leave the fleet, whatever the relation.
	l := newTKLab(t, 3)
	enemy := l.planet(1, 0, 100)
	unowned := l.planet(NoOwner, 10, 0)
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	all := Task{Kind: TaskTransport, Transport: [NumCargo]Transport{Ironium: {Action: UnloadAll}, Boranium: {Action: UnloadAll}}}
	for _, pi := range []int{enemy, unowned} {
		fi := l.fleet(0, pi, 0, Stack{Design: freighter, Count: 1})
		l.g.Fleets[fi].Cargo.Minerals = Minerals{7, 3, 0}
		l.g.Fleets[fi].Task = all
	}
	l.g.unloadPhase(l.g.phaseStart())
	for i, pi := range []int{enemy, unowned} {
		if s, c := l.g.Planets[pi].Surface, l.g.Fleets[i].Cargo.Minerals; s != (Minerals{7, 3, 0}) || c != (Minerals{}) {
			t.Errorf("planet %d: surface %v, fleet keeps %v; want 7/3/0 delivered", pi, s, c)
		}
	}
}

func TestPredictionDeepSpaceUnload(t *testing.T) {
	// #37: in deep space minerals are destroyed with no salvage, and
	// colonists are refused and stay aboard.
	l := newTKLab(t, 3)
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	l.g.Fleets = append(l.g.Fleets, Fleet{ID: 1, Owner: 0, Pos: Point{50, 50}, Stacks: []Stack{{Design: freighter, Count: 1}},
		Cargo: Cargo{Minerals: Minerals{7, 3, 0}, Colonists: 20},
		Task:  Task{Kind: TaskTransport, Transport: [NumCargo]Transport{Ironium: {Action: UnloadAll}, CargoColonists: {Action: UnloadAll}}}})
	_, ev := l.g.unloadPhase(l.g.phaseStart())
	f := l.g.Fleets[0]
	if f.Cargo != (Cargo{Minerals: Minerals{0, 3, 0}, Colonists: 20}) || len(l.g.Salvage) != 0 || len(ev) != 1 || ev[0].Kind != EventDropRefused {
		t.Errorf("cargo %+v salvage %v events %v; want ironium destroyed, 20 colonists kept, no salvage, one refusal", f.Cargo, l.g.Salvage, ev)
	}
}

func TestConfirmedAlternateRealityColonyStarbase(t *testing.T) {
	// TAKEOVER.md "Colonization" (CONFIRMED T-26, T-33): an Alternate
	// Reality colony gets a starbase of the owner's first starbase design,
	// here the Space Station in slot 1 rather than the Orbital Fort in
	// slot 3 (lowest occupied slot, ASSUMPTION T3). With no starbase
	// design, none (UNRESOLVED: TAKEOVER.md does not say).
	l := newTKLab(t, 3)
	l.g.Players[0].Race.PRT = PRTAlternateReality
	fort := l.design("Orbital Fort")
	station := l.design("Space Station")
	l.g.DesignSlots = append(l.g.DesignSlots,
		DesignSlot{Owner: 0, Starbase: true, Slot: 3, Design: fort},
		DesignSlot{Owner: 1, Starbase: true, Slot: 0, Design: fort},
		DesignSlot{Owner: 0, Starbase: true, Slot: 1, Design: station})
	pi := l.planet(NoOwner, 0, 0)
	l.g.newColony(pi, 0, 25)
	p := l.g.Planets[pi]
	if !p.HasStarbase || p.StarbaseDesign != station || p.StarbaseHull != l.g.Designs[station].Hull.StarbaseNumber || p.StarbaseDamage != 0 {
		t.Errorf("starbase %v design %d hull %d damage %d, want the Space Station", p.HasStarbase, p.StarbaseDesign, p.StarbaseHull, p.StarbaseDamage)
	}
	if col := NewColony(&p, &l.g.Players[0]); col.MaxPop == 0 {
		t.Error("the colony's maximum population is 0")
	}

	l = newTKLab(t, 3)
	l.g.Players[0].Race.PRT = PRTAlternateReality
	pi = l.planet(NoOwner, 0, 0)
	l.g.newColony(pi, 0, 25)
	if p := l.g.Planets[pi]; p.HasStarbase || p.StarbaseHull != 0 {
		t.Errorf("no design: starbase %v hull %d, want none", p.HasStarbase, p.StarbaseHull)
	}
}

func TestAlternateRealityColonyGeneratesYears(t *testing.T) {
	// The lane D audit's halt: an Alternate Reality colony ship colonizes
	// a habitable planet, and the following years must generate under
	// every ruleset, also the one that stops on a maximum population of 0
	// (RULESET.md). The colony gets its starbase (CONFIRMED T-26, T-33),
	// so its maximum population is not 0 and it keeps its population.
	for _, rules := range Rulesets() {
		l := newTKLab(t, 3)
		l.g.Players[0].Race.PRT = PRTAlternateReality
		fort := l.design("Orbital Fort")
		l.g.DesignSlots = append(l.g.DesignSlots, DesignSlot{Owner: 0, Starbase: true, Slot: 0, Design: fort})
		pi := l.planet(NoOwner, 0, 0)
		fi := l.fleet(0, pi, 0, Stack{Design: l.colonyShip(), Count: 1})
		l.g.Fleets[fi].Cargo.Colonists = 25
		l.g.Fleets[fi].Task = colonizeTask
		g := l.g
		g.Rules = rules
		for y := 1; y <= 4; y++ {
			r, err := GenerateTurn(g, nil, &seqRand{})
			if err != nil {
				t.Fatalf("%s v%d year %d: %v", rules.ID, rules.Version, y, err)
			}
			g = r.Game
		}
		if p := g.Planets[pi]; p.Owner != 0 || p.Population <= 0 || !p.HasStarbase {
			t.Errorf("%s v%d: owner %d population %d starbase %v, want a living colony with its starbase", rules.ID, rules.Version, p.Owner, p.Population, p.HasStarbase)
		}
	}
}

func TestConfirmedLoadAmounts(t *testing.T) {
	// TAKEOVER.md "Unload and load amounts" (CONFIRMED TK-301, TK-302):
	// load exactly 30 colonists from 100 leaves 70; 40 ironium asked with
	// 25 on the surface loads 25; 300 asked with 500 on the surface and a
	// 210 kT hold loads 210 and leaves 290. ASSUMPTION T4: what load
	// exactly could not move stays as the action; load all clears.
	l := newTKLab(t, 3)
	medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
	load := func(c int, a TransportAction, v int) Task {
		t := Task{Kind: TaskTransport}
		t.Transport[c] = Transport{Action: a, Amount: v}
		return t
	}
	cases := []struct {
		task          Task
		surface, pop  int
		wantHeld      int
		wantLeft      int
		wantRemaining int
	}{
		{load(CargoColonists, LoadExactly, 30), 0, 100, 30, 70, 0},
		{load(0, LoadExactly, 40), 25, 10, 25, 0, 15},
		{load(0, LoadExactly, 300), 500, 10, 210, 290, 90},
		{load(0, LoadAll, 0), 500, 10, 210, 290, 0},
	}
	for k, c := range cases {
		l.g.Planets, l.g.Fleets = nil, nil
		pi := l.planet(0, 0, c.pop)
		l.g.Planets[pi].Surface[0] = c.surface
		fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
		l.g.Fleets[fi].Task = c.task
		l.g.loadPass(false)
		f, p := l.g.Fleets[0], l.g.Planets[pi]
		held, left := f.Cargo.Minerals[0], p.Surface[0]
		if c.task.Transport[CargoColonists].Action != TransportNone {
			held, left = f.Cargo.Colonists, p.Population
		}
		remaining := 0
		if f.Task.Kind == TaskTransport {
			for _, tr := range f.Task.Transport {
				remaining += tr.Amount
			}
		}
		if held != c.wantHeld || left != c.wantLeft || remaining != c.wantRemaining {
			t.Errorf("case %d: held %d, left %d, action keeps %d; want %d, %d, %d", k, held, left, remaining, c.wantHeld, c.wantLeft, c.wantRemaining)
		}
	}
	// Another player's planet gives nothing, and the action waits.
	l.g.Planets, l.g.Fleets = nil, nil
	pi := l.planet(1, 0, 100)
	fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
	l.g.Fleets[fi].Task = load(CargoColonists, LoadAll, 0)
	l.g.loadPass(false)
	if f := l.g.Fleets[0]; f.Cargo.Colonists != 0 || f.Task.Transport[CargoColonists].Action != LoadAll {
		t.Errorf("foreign planet: %+v", f)
	}
}
