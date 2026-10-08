package objects

import (
	"errors"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// The space objects inside engine.GenerateTurn (OBJECTS.md "Turn
// placement", KERNEL.md "Turn order").

// zero is an engine.Rand that always draws 0: every minefield draw hits.
type zero struct{}

func (zero) Intn(int) int { return 0 }

// top is an engine.Rand that always draws n − 1: no minefield draw hits,
// no Trader warp rises, no wormhole jumps.
type top struct{}

func (top) Intn(n int) int { return n - 1 }

// leave is top, but draws 0 from rand(2).
type leave struct{}

func (leave) Intn(n int) int {
	if n == 2 {
		return 0
	}
	return n - 1
}

func turn(t *testing.T, g engine.Game, rng engine.Rand) engine.Game {
	t.Helper()
	r, err := engine.GenerateTurn(withRules(g), nil, rng)
	if err != nil {
		t.Fatal(err)
	}
	return r.Game
}

func space(g engine.Game) *Space { return g.Objects.(*Space) }

// A fleet crossing an enemy field at warp 9 stops on the first draw that
// hits, where it entered the field, and takes the hit; the paying field
// loses its mines during movement and then decays (OBJECTS.md "Hits on
// moving fleets", "On a hit"; KERNEL.md "Turn order" 3.3 then 3a, MF-4).
func TestConfirmedMineStopInTurn(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, at(-100, 0), "Tank", 2)
	f := &l.g.Fleets[fi]
	f.Fuel = 500
	f.Waypoints = []engine.Waypoint{{Pos: at(200, 0), Warp: 9}}
	l.g.Objects = &Space{Minefields: []Minefield{{Owner: 1, Kind: Heavy, Pos: origin, Count: 2500}}}

	g := turn(t, *l.g, zero{})
	got := g.Fleets[0]
	if got.Pos != at(-50, 0) {
		t.Errorf("stopped at %v, want the field's edge %v", got.Pos, at(-50, 0))
	}
	if len(got.Waypoints) != 1 {
		t.Errorf("waypoints %v: the stopped fleet did not arrive", got.Waypoints)
	}
	// Heavy, 2 ships of 1 engine: 2·500 = 1000 < 2000, so 2000 in all,
	// shields none; each Tank takes 1000 of 3200 + hull armor.
	if len(got.Stacks) != 1 || got.Stacks[0].Damage.Units == 0 {
		t.Errorf("stacks %+v, want damaged and alive", got.Stacks)
	}
	m := space(g).Minefields
	paid := MinesLost(2500)
	after := 2500 - paid - DecayLoss(Minefield{Kind: Heavy, Count: 2500 - paid}, 0, false)
	if len(m) != 1 || m[0].Count != after {
		t.Errorf("field %+v, want %d after the hit and decay", m, after)
	}
	// The original game is unchanged.
	if l.g.Objects.(*Space).Minefields[0].Count != 2500 || l.g.Fleets[0].Pos != at(-100, 0) {
		t.Error("GenerateTurn changed its input")
	}
}

// With no draw that hits the fleet passes and arrives.
func TestConfirmedMineMissInTurn(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, at(-100, 0), "Tank", 1)
	l.g.Fleets[fi].Fuel = 500
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: at(-20, 0), Warp: 9}}
	l.g.Objects = &Space{Minefields: []Minefield{{Owner: 1, Kind: Heavy, Pos: origin, Count: 2500}}}
	g := turn(t, *l.g, top{})
	if g.Fleets[0].Pos != at(-20, 0) || len(g.Fleets[0].Waypoints) != 0 {
		t.Errorf("fleet at %v, waypoints %v", g.Fleets[0].Pos, g.Fleets[0].Waypoints)
	}
}

// A fleet destroyed by a hit leaves the game, and its minerals become
// salvage at the stop point.
func TestPredictionMineHitDestroysFleet(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, at(-100, 0), "Freighter", 1)
	f := &l.g.Fleets[fi]
	f.Fuel = 450
	f.Cargo.Minerals = engine.Minerals{50, 0, 0}
	f.Waypoints = []engine.Waypoint{{Pos: at(0, 0), Warp: 9}}
	l.g.Objects = &Space{Minefields: []Minefield{{Owner: 1, Kind: Heavy, Pos: origin, Count: 2500}}}
	g := turn(t, *l.g, zero{})
	if len(g.Fleets) != 0 {
		t.Fatalf("fleets %+v, want the freighter destroyed", g.Fleets)
	}
	if len(g.Salvage) != 1 || g.Salvage[0].Pos != at(-50, 0) || g.Salvage[0].Minerals[0] != 50 {
		t.Errorf("salvage %+v", g.Salvage)
	}
}

// A lay-mines task holds the fleet, lays every year after decay, and a
// two-year duration ends after its second lay; the later waypoint stays
// queued (OBJECTS.md "Laying", CONFIRMED OB-014-D, OB-019, OB-002-F).
func TestConfirmedLayMinesTask(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, origin, "MML", 1)
	f := &l.g.Fleets[fi]
	f.Fuel = 100
	f.Task = engine.Task{Kind: engine.TaskLayMines, Years: 2}
	f.Waypoints = []engine.Waypoint{{Pos: at(25, 0), Warp: 5}}
	l.g.Objects = &Space{}

	g := turn(t, *l.g, top{})
	if g.Fleets[0].Pos != origin || len(g.Fleets[0].Waypoints) != 1 {
		t.Fatalf("year 1: fleet at %v with %v, want held", g.Fleets[0].Pos, g.Fleets[0].Waypoints)
	}
	if m := space(g).Minefields; len(m) != 1 || m[0].Count != 160 || m[0].Owner != 0 {
		t.Fatalf("year 1: %+v, want one 160 field", m)
	}
	g = turn(t, g, top{})
	want := 160 - DecayLoss(Minefield{Kind: Standard, Count: 160}, 0, false) + 160
	if m := space(g).Minefields; len(m) != 1 || m[0].Count != want {
		t.Errorf("year 2: %+v, want %d (decay, then the lay)", m, want)
	}
	if g.Fleets[0].Task.Kind != engine.TaskNone || g.Fleets[0].Pos != origin {
		t.Errorf("year 2: task %+v at %v, want ended in place", g.Fleets[0].Task, g.Fleets[0].Pos)
	}
	g = turn(t, g, top{})
	if g.Fleets[0].Pos == origin {
		t.Error("year 3: the fleet still holds after its task ended")
	}
}

// Indefinitely never ends; a fleet with no dispensers lays nothing.
func TestPredictionLayMinesIndefinitely(t *testing.T) {
	l := newLab(t)
	a := l.fleet(0, origin, "MML", 1)
	b := l.fleet(0, at(300, 0), "Tank", 1)
	for _, fi := range []int{a, b} {
		l.g.Fleets[fi].Task = engine.Task{Kind: engine.TaskLayMines, Years: engine.YearsIndefinitely}
	}
	l.g.Objects = &Space{}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
	if err != nil {
		t.Fatal(err)
	}
	g := turn(t, r.Game, top{})
	if g.Fleets[0].Task.Kind != engine.TaskLayMines || len(space(g).Minefields) != 1 {
		t.Errorf("task %+v, fields %+v", g.Fleets[0].Task, space(g).Minefields)
	}
	no := 0
	for _, e := range r.Events {
		if e.Kind == engine.EventNoDispensers && e.Fleet == l.g.Fleets[b].ID {
			no++
		}
	}
	if no != 1 {
		t.Errorf("%d no-dispenser messages, want 1", no)
	}
}

// Sweeping runs after laying in the same year (KERNEL.md "Turn order"
// 7.1, CONFIRMED OB-007-D): an enemy beam fleet sitting in the new field
// sweeps it at once.
func TestConfirmedSweepAfterLay(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, origin, "MML", 1)
	l.g.Fleets[fi].Task = engine.Task{Kind: engine.TaskLayMines, Years: 1}
	l.fleet(1, at(3, 0), "Laser DD", 1)
	l.g.Objects = &Space{}
	g := turn(t, *l.g, top{})
	m := space(g).Minefields
	if len(m) != 1 || m[0].Count >= 160 {
		t.Errorf("fields %+v, want the 160 field swept the year it was laid", m)
	}
}

// A fleet whose waypoint is a wormhole end transits on arrival to the
// partner's position from before the wormholes move, and both ends become
// known to its owner (OBJECTS.md "Travel", CONFIRMED OB-005 C, D).
func TestConfirmedWormholeTransitInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Size = 1
	in, out := at(100, 100), at(600, 500)
	fi := l.fleet(0, at(60, 100), "Tank", 1)
	l.g.Fleets[fi].Fuel = 500
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: in, Warp: 9, Target: engine.TargetWormhole, ID: WormholeEndID(0, 0)}}
	l.g.Objects = &Space{Wormholes: []Wormhole{{Ends: [2]WormholeEnd{{Pos: in}, {Pos: out}}}}}
	g := turn(t, *l.g, top{})
	if g.Fleets[0].Pos != out {
		t.Errorf("fleet at %v, want the exit's start-of-year position %v", g.Fleets[0].Pos, out)
	}
	w := space(g).Wormholes[0]
	if !w.Ends[0].KnownBy(0) || !w.Ends[1].KnownBy(0) || !w.Ends[0].DestinationKnownBy(0) {
		t.Errorf("knowledge %+v", w)
	}
	if w.Ends[1].Pos == out {
		t.Error("the exit did not move after the fleets")
	}
	// A plain position on the end is no transit.
	l.g.Fleets[fi].Waypoints[0].Target, l.g.Fleets[fi].Waypoints[0].ID = engine.TargetSpace, 0
	if g := turn(t, *l.g, top{}); g.Fleets[0].Pos != in {
		t.Errorf("plain waypoint: fleet at %v, want %v", g.Fleets[0].Pos, in)
	}
}

// A waypoint on an end its owner does not know becomes a plain position
// at the end's old position after the wormhole moves; one on a known end
// follows it (CONFIRMED OB-025-F, OB-027).
func TestConfirmedWormholeWaypointKnowledge(t *testing.T) {
	l := newLab(t)
	l.g.Size = 1
	end := at(400, 400)
	for owner := range 2 {
		fi := l.fleet(owner, at(0, 400), "Tank", 1)
		l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: end, Warp: 0, Target: engine.TargetWormhole, ID: WormholeEndID(0, 0)}}
	}
	l.g.Objects = &Space{Wormholes: []Wormhole{{Ends: [2]WormholeEnd{{Pos: end, Known: []bool{true}}, {Pos: at(700, 700)}}}}}
	g := turn(t, *l.g, top{})
	moved := space(g).Wormholes[0].Ends[0].Pos
	if moved == end {
		t.Fatal("the end did not move")
	}
	if wp := g.Fleets[0].Waypoints[0]; wp.Target != engine.TargetWormhole || wp.Pos != moved {
		t.Errorf("known: %+v, want following %v", wp, moved)
	}
	if wp := g.Fleets[1].Waypoints[0]; wp.Target != engine.TargetSpace || wp.Pos != end {
		t.Errorf("unknown: %+v, want plain %v", wp, end)
	}
}

// The Trader moves before fleets, and a waypoint on it reads its new
// position (KERNEL.md "Turn order" 3.2, CONFIRMED WT-001 F1, WT-005).
func TestConfirmedTraderWaypointInTurn(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemResearch})
	fi := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fi].Fuel = 500
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 9, Target: engine.TargetTrader, ID: 0}}
	l.g.Objects = s
	g := turn(t, *l.g, top{})
	tr := space(g).Traders[0]
	if tr.Pos != at(81, 0) {
		t.Fatalf("Trader at %v, want 81 ly east", tr.Pos)
	}
	if wp := g.Fleets[0].Waypoints[0]; wp.Pos != tr.Pos || wp.Target != engine.TargetTrader {
		t.Errorf("waypoint %+v, want on the Trader at %v", wp, tr.Pos)
	}
}

// A Trader that leaves turns waypoints on it into plain positions where
// it left, and their owner is told.
func TestPredictionTraderLeftInTurn(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemResearch})
	s.Traders[0].Dest = at(40, 0)
	fi := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fi].Waypoints = []engine.Waypoint{{Pos: origin, Warp: 0, Target: engine.TargetTrader, ID: 0}}
	l.g.Objects = s
	// rand(2) = 0 on arrival removes the Trader.
	r, err := engine.GenerateTurn(withRules(*l.g), nil, leave{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(space(r.Game).Traders); n != 0 {
		t.Fatalf("%d Traders, want it gone", n)
	}
	wp := r.Game.Fleets[0].Waypoints[0]
	if wp.Target != engine.TargetSpace {
		t.Errorf("waypoint %+v, want plain space", wp)
	}
	told := false
	for _, e := range r.Events {
		told = told || e.Kind == engine.EventTraderLeft && e.Player == 0
	}
	if !told {
		t.Error("no message")
	}
}

// The Trader meets fleets after battles in the turn; a 5,000 kT fleet at
// its position is consumed (OBJECTS.md "Encounters", CONFIRMED OB-004).
func TestConfirmedTraderMeetInTurn(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemResearch})
	s.Traders[0].Dest = at(0, 0)
	s.Traders[0].Pos = at(0, 0)
	fi := l.fleet(0, origin, "Freighter", 1)
	cargo(l, fi, 5000)
	l.g.Objects = s
	g := turn(t, *l.g, top{})
	if len(g.Fleets) != 0 {
		t.Errorf("fleets %+v, want the freighter consumed", g.Fleets)
	}
}

// With random events on, the Trader's appearance draws come after the
// other random events, and an appearing Trader is announced to every
// player (KERNEL.md "Turn order" 4c, OBJECTS.md "Appearance").
func TestPredictionTraderAppearsInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Year = 2400 + 42
	l.g.RandomEvents = true
	l.g.Objects = &Space{}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, zero{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(space(r.Game).Traders); n != 1 {
		t.Fatalf("%d Traders, want 1", n)
	}
	n := 0
	for _, e := range r.Events {
		if e.Kind == engine.EventTraderAppeared {
			n++
		}
	}
	if n != len(l.g.Players) {
		t.Errorf("%d announcements, want %d", n, len(l.g.Players))
	}
}

// A packet in flight moves before fleets and hits its target planet
// (OBJECTS.md "Turn placement" step 3, "Impact").
func TestConfirmedPacketInTurn(t *testing.T) {
	l := newLab(t)
	l.g.Planets = []engine.Planet{{ID: 7, Pos: at(50, 0), Owner: engine.NoOwner}}
	l.g.Objects = &Space{Packets: []Packet{{Owner: 0, Pos: origin, Target: 7, Warp: 10, Class: 0, Cargo: engine.Minerals{1000, 0, 0}}}}
	g := turn(t, *l.g, top{})
	if n := len(space(g).Packets); n != 0 {
		t.Errorf("%d packets, want the packet arrived", n)
	}
	if s := g.Planets[0].Surface[0]; s != 111 {
		t.Errorf("surface %d, want 111 (uncaught, a ninth)", s)
	}
}

// The production queue's launch through the engine interface (OBJECTS.md
// "Launch", CONFIRMED OB-028, OB-028-F).
func TestConfirmedLaunchPacketInterface(t *testing.T) {
	l := newLab(t)
	sb := station(t, l, "Mass Driver 7", "")
	l.g.Planets = []engine.Planet{{ID: 3, Pos: origin, Owner: 0}, {ID: 7, Pos: at(50, 0), Owner: engine.NoOwner}}
	var o engine.SpaceObjects = &Space{}
	built, _, ev := o.LaunchPacket(l.g, 0, 7, 0, engine.Ironium, 1)
	if built || len(ev) != 1 || ev[0].Kind != engine.EventPacketNoDriver || ev[0].Planet != 3 {
		t.Errorf("no starbase: built %v, events %v", built, ev)
	}
	l.g.Planets[0].HasStarbase, l.g.Planets[0].StarbaseDesign = true, sb
	built, spend, ev := o.LaunchPacket(l.g, 0, 7, 0, engine.Ironium, 1)
	if !built || len(ev) != 0 || spend != (engine.Minerals{110, 0, 0}) {
		t.Errorf("mass driver: built %v, spend %v, events %v", built, spend, ev)
	}
	if p := o.(*Space).Packets; len(p) != 1 || p[0].Cargo != (engine.Minerals{100, 0, 0}) || p[0].Warp != 7 {
		t.Errorf("packets %+v", p)
	}
	if _, spend, _ := o.LaunchPacket(l.g, 0, 7, 0, engine.PacketMixed, 1); spend != (engine.Minerals{44, 44, 44}) {
		t.Errorf("mixed spend %v", spend)
	}
}

// withRules is g under the Elegy ruleset.
func withRules(g engine.Game) engine.Game {
	g.Rules = engine.ElegyRules()
	return g
}

// A detonate order reaches the minefields through the engine's order
// layer: the owner's Space Demolition order on its standard field is
// applied, and each refusal comes back as the order's error and changes
// nothing (OBJECTS.md "The detonate setting").
func TestDetonateOrderThroughEngine(t *testing.T) {
	g := &engine.Game{Rules: engine.ElegyRules(), Players: []engine.Player{
		{Race: engine.Race{PRT: engine.PRTSpaceDemolition}},
		{Race: engine.Race{PRT: engine.PRTJackOfAllTrades}},
	}}
	s := &Space{Minefields: []Minefield{
		{Owner: 0, Number: 0, Kind: Standard, Count: 1000},
		{Owner: 0, Number: 1, Kind: Heavy, Count: 1000},
		{Owner: 1, Number: 0, Kind: Standard, Count: 1000},
	}}
	g.Objects = s
	files := []engine.PlayerOrders{
		{Player: 0, GameID: g.ID, Year: g.Year, Orders: []engine.Order{
			engine.DetonateOrder{Minefield: 0, On: true},
			engine.DetonateOrder{Minefield: 1, On: true},
			engine.DetonateOrder{Minefield: 5, On: true},
		}},
		{Player: 1, GameID: g.ID, Year: g.Year, Orders: []engine.Order{
			engine.DetonateOrder{Minefield: 0, On: true},
		}},
	}
	a := engine.ApplyOrders(g, files, []int{0, 1})
	want := map[[2]int]error{{0, 0}: nil, {0, 1}: ErrNotStandardMine, {0, 2}: ErrNoField, {1, 0}: ErrNotDemolition}
	seen := 0
	for _, r := range a.Results {
		w, ok := want[[2]int{r.Player, r.Index}]
		if !ok {
			continue
		}
		seen++
		if (w == nil) != (r.Err == nil) || (w != nil && !errors.Is(r.Err, w)) {
			t.Errorf("player %d order %d: %v, want %v", r.Player, r.Index, r.Err, w)
		}
	}
	if seen != len(want) {
		t.Errorf("%d of %d order results", seen, len(want))
	}
	if !s.Minefields[0].Detonate || s.Minefields[1].Detonate || s.Minefields[2].Detonate {
		t.Errorf("detonate settings %v %v %v", s.Minefields[0].Detonate, s.Minefields[1].Detonate, s.Minefields[2].Detonate)
	}
}

// A fleet removed by the unload pass after movement (an arriving colony
// ship that colonizes) sits before a mine layer in Game.Fleets, with
// another fleet after it: the layer, not that fleet, lays its field and
// its duration counts down (KERNEL.md
// "Turn order" step 6c.2; the layers are taken after that pass).
func TestLayMinesAfterColonizerRemoved(t *testing.T) {
	l := newLab(t)
	d, err := engine.Components().NewDesign("Colony", "Colony Ship", []engine.SlotFill{
		{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 1, Part: "Colonization Module", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	l.designs["Colony"] = len(l.g.Designs)
	l.g.Designs = append(l.g.Designs, d)
	target := at(400, 0)
	l.g.Planets = []engine.Planet{{ID: 1, Owner: engine.NoOwner, Pos: target, Env: [3]int{50, 50, 50}, OrigEnv: [3]int{50, 50, 50}}}
	ci := l.fleet(0, at(390, 0), "Colony", 1)
	c := &l.g.Fleets[ci]
	c.Fuel, c.Cargo.Colonists = 200, 25
	c.Waypoints = []engine.Waypoint{{Pos: target, Warp: 5, Target: engine.TargetPlanet, ID: 1, Task: engine.Task{Kind: engine.TaskColonize}}}
	mi := l.fleet(0, origin, "MML", 1)
	layer := l.g.Fleets[mi].ID
	l.g.Fleets[mi].Task = engine.Task{Kind: engine.TaskLayMines, Years: 2}
	// A fleet after the layer, which a stale index would name instead.
	l.fleet(0, at(300, 300), "Tank", 1)
	r := &l.g.Players[0].Race
	r.GrowthRate, r.ColonistsPerResource, r.FactoryOutput, r.FactoryCost, r.FactoriesOperated = 15, 1000, 10, 10, 10
	r.MineOutput, r.MineCost, r.MinesOperated = 10, 5, 10
	for a := range r.Env {
		r.Env[a] = engine.EnvRange{Center: 50, Low: 15, High: 85}
	}
	l.g.Objects = &Space{}

	g := turn(t, *l.g, top{})
	if len(g.Fleets) != 2 || g.Fleets[0].ID != layer {
		t.Fatalf("fleets %+v, want the layer and the tank (the colony ship consumed)", g.Fleets)
	}
	if m := space(g).Minefields; len(m) != 1 || m[0].Count != 160 || m[0].Owner != 0 || m[0].Pos != origin {
		t.Errorf("fields %+v, want one 160 field of player 0 at the layer", m)
	}
	if tk := g.Fleets[0].Task; tk.Kind != engine.TaskLayMines || tk.Years != 1 {
		t.Errorf("layer's task %+v, want lay mines with 1 year left", tk)
	}
}

// MeetTraders reads a computer player's level from engine.Player.Level
// (OBJECTS.md "Computer players' planets", CONFIRMED O-53): a Harder
// planet with 3,600 kT trades and pays 3,500 kT; a Standard one does not.
func TestMeetTradersComputerLevel(t *testing.T) {
	for _, c := range []struct {
		level int
		trade bool
	}{{0, false}, {1, false}, {2, true}} {
		l, s := traderLab(t, TraderItem{Kind: ItemResearch})
		l.g.Players[1].Computer, l.g.Players[1].Level = true, c.level
		l.g.Planets = []engine.Planet{{ID: 7, Pos: at(50, 0), Owner: 1, HasStarbase: true, Surface: engine.Minerals{0, 0, 3600}}}
		ev := s.MeetTraders(l.g, &script{})
		traded := len(ev) == 1 && ev[0].Kind == engine.EventTraderPlanetTrade && ev[0].Player == 1 && ev[0].Planet == 7
		if traded != c.trade || (c.trade && l.g.Planets[0].Surface != (engine.Minerals{0, 0, 100})) {
			t.Errorf("level %d: events %+v surface %v, want trade %v", c.level, ev, l.g.Planets[0].Surface, c.trade)
		}
		if !c.trade && len(ev) != 0 {
			t.Errorf("level %d: events %+v, want none", c.level, ev)
		}
	}
}

// A design order reads the player's Trader parts (ORDERS.md "Design
// legality"; OBJECTS.md "Encounters"): under the Elegy rules a Multi Cargo
// Pod is kept when the player owns it and dropped when it does not.
func TestDesignOrderTraderParts(t *testing.T) {
	cargo := func(owns bool) int {
		s := &Space{}
		if owns {
			s.TraderParts.give(0, BitMultiCargoPod)
		}
		g := &engine.Game{Rules: engine.ElegyRules(), Year: 2410, Objects: s, Players: []engine.Player{{
			Race:     engine.Race{PRT: engine.PRTJackOfAllTrades},
			Research: engine.ResearchState{Levels: [engine.NumFields]int{11, 11, 11, 11, 11, 11}},
		}}}
		a := engine.ApplyOrders(g, []engine.PlayerOrders{{Player: 0, GameID: g.ID, Year: g.Year, Orders: []engine.Order{engine.DesignOrder{
			Slot: 0, Name: "Pod", Hull: "Small Freighter",
			Fills: []engine.SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}, {Slot: 1, Part: "Multi Cargo Pod", Count: 1}},
		}}}}, []int{0})
		if a.Results[0].Err != nil {
			t.Fatal(a.Results[0].Err)
		}
		return g.Designs[g.DesignSlots[0].Design].CargoCapacity
	}
	if with, without := cargo(true), cargo(false); with-without != 250 {
		t.Errorf("cargo %d with the pod owned, %d without; want 250 more", with, without)
	}
}
