package engine

// Space objects in the turn: the minefields, wormholes and Mystery Traders
// that package objects holds (stars-elegy OBJECTS.md). The objects package
// imports the engine, so the engine reaches it through SpaceObjects and
// calls each step where OBJECTS.md "Turn placement" and KERNEL.md "Turn
// order" put it.

// SpaceObjects is the game's space objects. A nil Game.Objects is a galaxy
// with none of them; the steps then do nothing and draw nothing.
//
// Every method takes the year's game and changes it as its rule says
// (fleets damaged or consumed, designs and fleets created, salvage); the
// engine removes fleets left with no ships.
type SpaceObjects interface {
	// CloneObjects returns an independent copy, for GenerateTurn's copy of
	// the game.
	CloneObjects() SpaceObjects

	// ObjectPos is where a waypoint's object target is now; ok is false
	// when the object is gone (KERNEL.md "Turn order" 1a.3, the waypoint
	// check).
	ObjectPos(kind TargetKind, id int) (pos Point, ok bool)

	// MoveObjects is step 3.2: the Traders move, then the packets in
	// flight move and may hit. Waypoints aimed at a Trader that left
	// become plain positions; the ones aimed at a Trader still here are
	// renumbered with it.
	MoveObjects(g *Game, rng Rand) []Event

	// MineCheck checks one movement step of fleet fi, distance ly from
	// `from` toward `toward`, for a minefield stop (OBJECTS.md "Hits on
	// moving fleets"). stop is the distance from `from` at which the fleet
	// stops; kind identifies the stopping field kind for MineHit.
	MineCheck(g *Game, fi int, from, toward Point, distance int, rng Rand) (stop, kind int, hit bool)
	// MineHit applies a stop of fleet fi, already at its stop point.
	MineHit(g *Game, fi int, kind int, rng Rand) []Event

	// Stargate runs fleet fi's stargate order to dest instead of moving
	// it (OBJECTS.md "Stargates"). jumped reports the fleet at dest; lost
	// a fleet the jump destroyed, which the engine removes.
	Stargate(g *Game, fi int, dest Point, rng Rand) (jumped, lost bool, ev []Event)

	// TransitWormhole takes fleet fi through the wormhole end its reached
	// waypoint targets (OBJECTS.md "Travel").
	TransitWormhole(g *Game, fi int, end int) []Event

	// DecayObjects is step 3a: packet decay, detonations, then
	// minefield decay.
	DecayObjects(g *Game) []Event

	// TraderAppears is the Mystery Trader's appearance at the end of step 4c.
	TraderAppears(g *Game, rng Rand) []Event

	// MoveObjectsAgain is step 5.1: the packets launched this year fly
	// half a year and may hit, then the wormholes jiggle or jump, with
	// the waypoints aimed at a moved end.
	MoveObjectsAgain(g *Game, rng Rand) []Event

	// MeetTraders is step 6b, the Mystery Trader encounters. A gift fleet is
	// reported with EventTraderGift.
	MeetTraders(g *Game, rng Rand) []Event

	// LayMines lays mines for the given fleets (step 6c.2).
	LayMines(g *Game, layers []MineLayer) []Event

	// SweepMines is step 7.1, mine sweeping.
	SweepMines(g *Game) []Event

	// SeeObjects records what each player's scanners see of the objects
	// this year, as knowledge carried to later years (SCANNING.md "When
	// knowledge is computed", "Space objects").
	SeeObjects(g *Game, scanners []ObjectScanner)
}

// ObjectScanner is one of a player's viewing objects, a fleet or a
// planet, with its normal and penetrating ranges in ly.
type ObjectScanner struct {
	Player int
	Pos    Point
	R, P   int
	Fleet  bool
}

// ObjectScanners lists every player's viewing objects (SCANNING.md
// "Scanner ranges").
func (g *Game) ObjectScanners() []ObjectScanner {
	var out []ObjectScanner
	for v := range g.Players {
		for _, s := range g.scanners(v) {
			out = append(out, ObjectScanner{Player: v, Pos: s.pos, R: s.R, P: s.P, Fleet: s.fleet})
		}
	}
	return out
}

// MineLayer is a fleet laying mines this year (OBJECTS.md "Laying"):
// Fleet is its index in Game.Fleets; Half marks the Space Demolition lay
// of a fleet that moved.
type MineLayer struct {
	Fleet int
	Half  bool
}

// Waypoint targets that are space objects. ID is the wormhole end
// (2 × wormhole + end) or the Trader's index in object order.
const (
	TargetWormhole TargetKind = TargetFleet + 1 + iota
	TargetTrader
)

// StargateWarp is a waypoint's warp for a stargate jump (OBJECTS.md
// "Stargates").
const StargateWarp = 11

// TaskLayMines is the "lay mines" waypoint task. Task.Years is its
// duration: 1 for "this year only", k for k years, YearsIndefinitely to
// lay until the task is changed (OBJECTS.md "Laying", "Duration").
const TaskLayMines TaskKind = TaskRoute + 3

// YearsIndefinitely is TaskLayMines' duration "indefinitely".
const YearsIndefinitely = -1

// Space object messages. Elegy's wording; Player is who is told.
const (
	EventMineHit           EventKind = iota + EventFleetGiftNoRoom + 1 // Player = victim, Fleet, Count = mines the field lost
	EventMineShipsLost                                                 // Player = victim, Fleet, Count = ships destroyed by a hit or detonation
	EventMineDetonation                                                // Player = victim, Fleet: hit by a detonating field
	EventMinesLaid                                                     // Player, Fleet, Count = mines
	EventMinesNotLaid                                                  // Player, Fleet: no room for a new field
	EventNoDispensers                                                  // Player, Fleet: a lay-mines task with no dispensers
	EventMinesSwept                                                    // Player = sweeper, Fleet (or -1 for a starbase), Planet (or -1), Count = mines
	EventWormholeTransit                                               // Player, Fleet, Count = the end passed through
	EventTraderAppeared                                                // Player (each), Count = the Trader's index
	EventTraderLeft                                                    // Player, Fleet: its waypoint on a Trader that left is now space
	EventTraderTrade                                                   // Player, Fleet (the consumed fleet's id), Count = the item kind
	EventTraderRefused                                                 // Player, Fleet, Count = 1 when already served, 0 for too little cargo
	EventTraderPlanetTrade                                             // Player, Planet: a computer player's planet traded
	EventTraderGift                                                    // Player, Fleet = the gift fleet, Count = ships
	EventTraderNoRoom                                                  // Player, Fleet = the traded fleet: no slot or fleet number for a gift
	EventPacketImpact                                                  // Player = the planet's owner (or the packet's when unowned), Planet, Count = colonists killed (units)
	EventWormholeMoved                                                 // Player = -1, Count = the end; a jump or jiggle
	EventGateRefused                                                   // Player, Fleet, Count = the refusal (objects.GateRefusal)
	EventGateUnloaded                                                  // Player = the source planet's owner, Planet, Fleet: cargo put down before a jump
	EventGateShipsLost                                                 // Player, Fleet, Count = ships destroyed by a jump
	EventGateFleetLost                                                 // Player, Fleet: the jump destroyed the fleet
)

// loseFollowers turns other players' waypoints aimed at fleet f into
// plain positions at `at`: they lose a fleet that went through a
// wormhole or a stargate (OBJECTS.md "Travel", "Stargates").
func (g *Game) loseFollowers(f *Fleet, at Point) {
	for j := range g.Fleets {
		if g.Fleets[j].Owner == f.Owner {
			continue
		}
		for k := range g.Fleets[j].Waypoints {
			if wp := &g.Fleets[j].Waypoints[k]; wp.Target == TargetFleet && wp.ID == f.ID {
				wp.Target, wp.ID, wp.Pos = TargetSpace, 0, at
			}
		}
	}
}

// BombSurvival is the share of a normal bomb's kill that gets through
// planet pi's defenses (TAKEOVER.md "Planetary defenses against bombs"),
// which a packet's damage also uses (OBJECTS.md "Impact" step 6).
func (g *Game) BombSurvival(pi int) float64 {
	s, _ := g.survival(&g.Planets[pi])
	return float64(s)
}

// EmptyPlanet empties planet pi as after bombing (TAKEOVER.md
// "Capture"), for a packet impact that leaves it uninhabited.
func (g *Game) EmptyPlanet(pi int) Event { return g.emptyPlanet(pi) }

// layers is the year's mine layers (OBJECTS.md "Laying", CONFIRMED
// OB-014-C/D, OB-019): a fleet whose current task is "lay mines" and that
// did not move lays in full; a Space Demolition fleet that moved toward or
// onto a "lay mines" waypoint lays half.
//
// ASSUMPTION O9: a fleet that arrived this year onto a "lay mines"
// waypoint moved, so only a Space Demolition fleet lays (half) that year.
func (g *Game) layers(moved map[int]bool) []MineLayer {
	var out []MineLayer
	for i := range g.Fleets {
		f := &g.Fleets[i]
		sd := f.Owner >= 0 && f.Owner < len(g.Players) && g.Players[f.Owner].Race.PRT == PRTSpaceDemolition
		switch {
		case !moved[f.ID] && f.Task.Kind == TaskLayMines:
			out = append(out, MineLayer{Fleet: i})
		case moved[f.ID] && sd && (f.Task.Kind == TaskLayMines || len(f.Waypoints) > 0 && f.Waypoints[0].Task.Kind == TaskLayMines):
			out = append(out, MineLayer{Fleet: i, Half: true})
		}
	}
	return out
}

// endLayYear counts down the lay-mines tasks of the fleets that laid in
// full (OBJECTS.md "Laying", "Duration", CONFIRMED OB-002-N, OB-019,
// OB-025).
//
// ASSUMPTION O10: a Space Demolition half lay does not count toward the
// duration.
func (g *Game) endLayYear(layers []MineLayer) {
	for _, l := range layers {
		f := &g.Fleets[l.Fleet]
		if l.Half || f.Task.Kind != TaskLayMines || f.Task.Years == YearsIndefinitely {
			continue
		}
		f.Task.Years--
		if f.Task.Years <= 0 {
			f.Task = Task{}
		}
	}
}

// dropShiplessFleets removes fleets left with no ships by a space object
// (a mine hit, a detonation, a Trader).
func (g *Game) dropShiplessFleets() {
	gone := map[int]bool{}
	for _, f := range g.Fleets {
		n := 0
		for _, s := range f.Stacks {
			n += s.Count
		}
		if n == 0 {
			gone[f.ID] = true
		}
	}
	if len(gone) > 0 {
		g.removeFleets(gone)
	}
}
