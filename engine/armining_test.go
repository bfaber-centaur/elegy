package engine

import (
	"slices"
	"testing"
)

// hookTerraform is stubTerraform with a hook run at each RemoteMine call.
type hookTerraform struct {
	stubTerraform
	hook func(g *Game, fi int)
}

func (s *hookTerraform) RemoteMine(g *Game, fi int, rng Rand) (Minerals, bool) {
	s.hook(g, fi)
	return s.stubTerraform.RemoteMine(g, fi, rng)
}

// arMineGame is remoteMineGame with player 0 an Alternate Reality race
// and its homeworld (index 0) holding a one-unit Mineral Alchemy queue.
func arMineGame(t *testing.T) (Game, *hookTerraform) {
	t.Helper()
	g, _ := remoteMineGame(t, 0, pgHomeworld().Planets[0].Pos, 1)
	g.Players[0].Race.PRT = PRTAlternateReality
	g.Planets[0].HasQueue = true
	g.Planets[0].Queue = []QueueItem{{Kind: ItemMineralAlchemy, Count: 1}}
	h := &hookTerraform{stubTerraform: stubTerraform{rate: 1}, hook: func(*Game, int) {}}
	g.Terraform = h
	return g, h
}

func arTurn(t *testing.T, g Game) TurnResult {
	t.Helper()
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// KERNEL.md "Remote mining" and "Turn order" step 4.1 (stars-elegy
// 8a3b9b5): an Alternate Reality owner's own miners at its own planet
// mine in the year's mining step, before production (MEASURED, RG-002 Y1
// and Y4 restarts), once, keeping the task. The hook sees the planet's
// queue as production has not yet touched it.
func TestMeasuredAROwnMinersBeforeProduction(t *testing.T) {
	g, h := arMineGame(t)
	var pct []int
	h.hook = func(g *Game, _ int) { pct = append(pct, g.Planets[0].Queue[0].Percent) }
	r := arTurn(t, g)
	if !slices.Equal(h.mined, []int{1}) || !slices.Equal(pct, []int{0}) {
		t.Errorf("mined %v with alchemy at %v%%, want fleet 1 once at 0%%", h.mined, pct)
	}
	if q := r.Game.Planets[0].Queue; len(q) != 1 || q[0].Percent == 0 {
		t.Fatalf("queue %+v: production did not run", q)
	}
	if k := r.Game.Fleets[0].Task.Kind; k != TaskRemoteMine {
		t.Errorf("task %v, want kept", k)
	}
}

// BINARY-ONLY (stars-elegy 8a3b9b5): the step runs after movement on
// the fleet as it stands, with no did-not-move test, so a fleet arriving
// this year with the task and no further waypoint mines that year. One
// still at the planet with a further waypoint (here out of fuel) mines
// neither then nor at step 6c, and keeps its task with no message.
func TestAROwnMinersArrivalAndFurtherWaypoint(t *testing.T) {
	home := pgHomeworld().Planets[0].Pos
	g, h := arMineGame(t)
	g.Fleets[0].Pos = Point{home.X + 10, home.Y}
	g.Fleets[0].Task = Task{}
	g.Fleets[0].Waypoints = []Waypoint{{Pos: home, Warp: 5, Task: Task{Kind: TaskRemoteMine}}}
	r := arTurn(t, g)
	if f := r.Game.Fleets[0]; f.Pos != home || f.Task.Kind != TaskRemoteMine || !slices.Equal(h.mined, []int{1}) {
		t.Errorf("arrival: pos %v task %v mined %v, want arrived and mined", f.Pos, f.Task.Kind, h.mined)
	}

	g, h = arMineGame(t)
	g.Fleets[0].Fuel = 0
	g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{home.X + 30, home.Y}, Warp: 5}}
	r = arTurn(t, g)
	if f := r.Game.Fleets[0]; f.Pos != home || f.Task.Kind != TaskRemoteMine || len(h.mined) != 0 {
		t.Errorf("further waypoint: pos %v task %v mined %v, want stayed, task kept, nothing mined", f.Pos, f.Task.Kind, h.mined)
	}
	for _, e := range r.Events {
		if e.Fleet == 1 && e.Kind >= EventRemoteMineNoModules && e.Kind <= EventRemoteMineDeepSpace {
			t.Errorf("further waypoint: refusal %+v", e)
		}
	}
}

// BINARY-ONLY (stars-elegy 8a3b9b5): each miner is its own step, in
// fleet order (owner, then number), inside the planet-by-planet mining
// pass: after its planet's own mining and before the next planet's. The
// second planet here, a non-AR colony of player 1 with a higher id, has
// not been mined when the miners run.
func TestAROwnMinersOrder(t *testing.T) {
	g, h := arMineGame(t)
	f2 := g.Fleets[0]
	f2.ID, f2.Number = 2, 1
	g.Fleets[0].Number = 2
	g.Fleets = append(g.Fleets, f2)
	home := g.Planets[0]
	colony := Planet{ID: 9, Owner: 1, Pos: Point{home.Pos.X, home.Pos.Y + 40}, Env: home.Env,
		Population: 486, Mines: 10, Deposits: home.Deposits, Surface: Minerals{1, 1, 1}}
	g.Planets = append(g.Planets, colony)
	var homeSurf, colonySurf []Minerals
	h.hook = func(g *Game, _ int) {
		homeSurf = append(homeSurf, g.Planets[0].Surface)
		colonySurf = append(colonySurf, g.Planets[2].Surface)
	}
	r := arTurn(t, g)
	if !slices.Equal(h.mined, []int{2, 1}) {
		t.Errorf("mined %v, want fleet order 2 (number 1), then 1 (number 2)", h.mined)
	}
	if r.Game.Planets[2].Surface == colony.Surface {
		t.Fatal("the colony was not mined")
	}
	for i := range colonySurf {
		if homeSurf[i] == home.Surface || colonySurf[i] != colony.Surface {
			t.Errorf("call %d: home %v (start %v), colony %v (start %v); want home mined, colony not yet", i, homeSurf[i], home.Surface, colonySurf[i], colony.Surface)
		}
	}
}

// mineTerraform is stubTerraform whose RemoteMine mines as
// terraform.RemoteMine does (MineYear at eff 10 with the stub's rate, the
// homeworld floor at an owned planet), so a whole turn's minerals can be
// followed without the terraform package.
type mineTerraform struct{ stubTerraform }

func (s *mineTerraform) RemoteMine(g *Game, fi int, rng Rand) (Minerals, bool) {
	s.stubTerraform.RemoteMine(g, fi, rng)
	f := &g.Fleets[fi]
	pi := g.planetAt(f.Pos)
	var gain Minerals
	p := &g.Planets[pi]
	for k := range NumMinerals {
		gain[k], p.Deposits[k] = MineYear(p.Deposits[k], s.rate, 10, p.Owner >= 0 && p.Homeworld, rng)
		p.Surface[k] += gain[k]
	}
	return gain, true
}

type fifty struct{}

func (fifty) Intn(n int) int { return min(50, n-1) }

// rg002Planet35 is the Macinti AR homeworld of the RG-002 restarts
// (stars-elegy vectors/rg/rg002-y1.json and rg002-y4.json, planet 35):
// a Space Station, the owner's 60-point miner fleet with a standing
// remote-mining task, design 12 (a Destroyer, 89 resources and 32/19/26
// kT) and design 14 (a Miner) in the owner's slots, research budget 15%.
func rg002Planet35(t *testing.T, pop int, conc, frac [3]int, surface Minerals, levels [NumFields]int, queue []QueueItem) (Game, *mineTerraform) {
	t.Helper()
	c := Components()
	station, err := c.NewDesign("Station", "Space Station", nil)
	if err != nil {
		t.Fatal(err)
	}
	destroyer, err := c.NewDesign("Destroyer", "Destroyer", []SlotFill{
		{0, "Radiating Hydro-Ram Scoop", 1}, {1, "X-Ray Laser", 1}, {2, "X-Ray Laser", 1}, {3, "X-Ray Laser", 1},
		{4, "Strobnium", 2}, {5, "Maneuvering Jet", 1}, {6, "Battle Computer", 1}})
	if err != nil {
		t.Fatal(err)
	}
	miner, err := c.NewDesign("Miner", "Miner", []SlotFill{
		{0, "Radiating Hydro-Ram Scoop", 2}, {1, "Maneuvering Jet", 2}, {2, "Robo-Midget Miner", 2},
		{3, "Robo-Midget Miner", 1}, {4, "Robo-Midget Miner", 2}, {5, "Robo-Midget Miner", 1}})
	if err != nil {
		t.Fatal(err)
	}
	race := Race{PRT: PRTAlternateReality, GrowthRate: 20, ColonistsPerResource: 1000}
	for a := range race.Env {
		race.Env[a] = EnvRange{Center: 50, Low: 15, High: 85}
	}
	pl := Player{Race: race, ResearchBudget: 15, Research: ResearchState{Levels: levels, Current: Energy, Next: NextSameField}}
	p := Planet{ID: 36, Owner: 0, Homeworld: true, Pos: Point{1197, 1362}, Env: [3]int{50, 50, 50}, OrigEnv: [3]int{50, 50, 50},
		Population: pop, Surface: surface, HasQueue: true, Queue: queue,
		HasStarbase: true, StarbaseDesign: 0, StarbaseHull: station.Hull.StarbaseNumber}
	for k := range NumMinerals {
		p.Deposits[k] = Deposit{Concentration: conc[k], Fraction: frac[k]}
	}
	g := Game{Rules: ElegyRules(), Year: 2462, Players: []Player{pl}, Planets: []Planet{p},
		Designs:     []Design{station, destroyer, miner},
		DesignSlots: []DesignSlot{{Owner: 0, Starbase: true, Slot: 6, Design: 0}, {Owner: 0, Slot: 12, Design: 1}, {Owner: 0, Slot: 14, Design: 2}},
		Fleets:      []Fleet{{ID: 1, Number: 1, Owner: 0, Pos: p.Pos, Fuel: 100, Stacks: []Stack{{Design: 2, Count: 2}}, Task: Task{Kind: TaskRemoteMine}}}}
	if got := designCost(destroyer, race, levels); got != (Cost{Resources: 89, Minerals: Minerals{32, 19, 26}}) {
		t.Fatalf("design 12 costs %+v, want 89 and 32/19/26 (RG-002 Y1)", got)
	}
	tf := &mineTerraform{stubTerraform{rate: 60}}
	g.Terraform = tf
	return g, tf
}

// MEASURED (stars-elegy 8a3b9b5, PARITY.md "RG-002 restarts: an AR
// owner's own miners"): the RG-002 planet-35 numbers. The miner's output
// reaches production; it is not mined again after production, so the
// year ends at 0 ironium as the oracle's did. Draws: 50, so the planet's
// own +1s are as the remainders give and the miner's ironium +1 hits.
func TestMeasuredRG002Planet35(t *testing.T) {
	ship := func(slot, count, pct int) QueueItem {
		return QueueItem{Kind: ItemShip, Slot: slot, Count: count, Percent: pct}
	}
	built := func(r TurnResult) int {
		n := 0
		for _, f := range r.Game.Fleets {
			if f.ID != 1 {
				for _, s := range f.Stacks {
					if s.Design == 1 {
						n += s.Count
					}
				}
			}
		}
		return n
	}
	// Y1 (2463): 155 kT of ironium less the 81 the three design-7 ships
	// before it took, plus 1 from Mineral Alchemy, is 75 here, with
	// design 12 ×7 alone. Own mines trunc(sqrt(8066)) = 89 at 63: +56;
	// the miner 60 at 63: +37, +1. 169 kT builds 5 and leaves 9 kT:
	// largestPercent(32, 9) = 30%. The oracle: fleet 6/12 of 5, the next
	// unit at 30%, 0 ironium. Mining after production: 131 kT, 4 and 11%.
	g, tf := rg002Planet35(t, 8066, [3]int{63, 55, 55}, [3]int{133, 2, 160}, Minerals{75, 1761, 607},
		levelsOf(13, 8, 8, 10, 9, 8), []QueueItem{ship(12, 7, 0)})
	r, err := GenerateTurn(g, nil, fifty{})
	if err != nil {
		t.Fatal(err)
	}
	p := r.Game.Planets[0]
	if n := built(r); n != 5 || len(p.Queue) != 1 || p.Queue[0] != ship(12, 2, 30) || p.Surface[0] != 0 || len(tf.mined) != 1 {
		t.Errorf("Y1: built %d, queue %+v, ironium %d, mined %v; want 5, design 12 ×2 at 30%%, 0, once", n, p.Queue, p.Surface[0], tf.mined)
	}
	// Y4 (2466): 1 kT of ironium; the Miner at 48%, then design 12 ×4.
	// The oracle builds the Miner and one design 12 and ends at 0
	// ironium; mining after production leaves the planet without the
	// design-12 ship.
	g, tf = rg002Planet35(t, 8283, [3]int{62, 53, 54}, [3]int{28, 244, 141}, Minerals{1, 1812, 528},
		levelsOf(13, 8, 9, 10, 9, 8), []QueueItem{ship(14, 1, 48), ship(12, 4, 0)})
	if r, err = GenerateTurn(g, nil, fifty{}); err != nil {
		t.Fatal(err)
	}
	p = r.Game.Planets[0]
	if n := built(r); n != 1 || p.Surface[0] != 0 || len(tf.mined) != 1 {
		t.Errorf("Y4: built %d, queue %+v, ironium %d, mined %v; want 1 design 12, 0, once", n, p.Queue, p.Surface[0], tf.mined)
	}
}

func levelsOf(energy, weapons, propulsion, construction, electronics, biotech int) [NumFields]int {
	var l [NumFields]int
	l[Energy], l[Weapons], l[Propulsion] = energy, weapons, propulsion
	l[Construction], l[Electronics], l[Biotech] = construction, electronics, biotech
	return l
}

// KERNEL.md "Remote mining" (BINARY-ONLY, stars-elegy 8a3b9b5): only the
// remote-mining task mines. A fleet with miners on another task at its
// own AR planet, with no further waypoint, does not.
func TestAROwnMinersOnlyTheTask(t *testing.T) {
	wait := Task{Kind: TaskTransport}
	wait.Transport[Ironium] = Transport{Action: SetAmount, Amount: 1 << 20}
	for _, task := range []Task{wait, {Kind: TaskPatrol, Range: 50}, {Kind: TaskLayMines, Years: YearsIndefinitely}} {
		g, h := arMineGame(t)
		g.Fleets[0].Task = task
		arTurn(t, g)
		if len(h.mined) != 0 {
			t.Errorf("task %v: mined %v, want none", task.Kind, h.mined)
		}
	}
}

// drawTerraform records each RemoteMine call's fleet and its first draw.
type drawTerraform struct {
	stubTerraform
	draws []int
}

func (s *drawTerraform) RemoteMine(g *Game, fi int, rng Rand) (Minerals, bool) {
	s.draws = append(s.draws, rng.Intn(1<<20))
	return s.stubTerraform.RemoteMine(g, fi, rng)
}

// stepRand draws 0, 1, 2, … in turn.
type stepRand struct{ n int }

func (c *stepRand) Intn(n int) int { c.n++; return (c.n - 1) % n }

// KERNEL.md "Remote mining" (BINARY-ONLY, stars-elegy 8a3b9b5): the
// miners go in fleet order, each with its own draws there. Fleets 1, 2
// and 3 are numbered 2, 3 and 1, so fleet order (3, 1, 2) is neither the
// slice order nor its reverse.
func TestAROwnMinersFleetOrderDraws(t *testing.T) {
	g, _ := arMineGame(t)
	g.Fleets[0].Number = 2
	for _, n := range []int{3, 1} {
		f := g.Fleets[0]
		f.ID, f.Number = len(g.Fleets)+1, n
		g.Fleets = append(g.Fleets, f)
	}
	d := &drawTerraform{stubTerraform: stubTerraform{rate: 1}}
	g.Terraform = d
	if _, err := GenerateTurn(withRules(g), nil, &stepRand{}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(d.mined, []int{3, 1, 2}) {
		t.Errorf("mined %v, want fleet order 3, 1, 2", d.mined)
	}
	if len(d.draws) != 3 || !(d.draws[0] < d.draws[1] && d.draws[1] < d.draws[2]) {
		t.Errorf("draws %v, want each later in the stream than the one before", d.draws)
	}
}
