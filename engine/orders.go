package engine

import (
	"errors"
	"fmt"
)

// The orders layer: what a player may order for a year and the checks
// Elegy applies when it accepts them. The specification is stars-elegy
// docs/ORDERS.md (PR #51 head 14e3c10), with battle plans from COMBAT.md
// (PR #59) and research from KERNEL.md "Research". Rules are tagged as
// there; Elegy's own choices where the specs are silent are marked
// ASSUMPTION Ln and listed in docs/ORDERS-LAYER-STATUS.md.
//
// Elegy defines its own order format (ORDERS.md "Scope and vocabulary"):
// an order is a Go value, not a record of the original's files.

// PlayerOrders is one player's orders for one year, the equivalent of an
// order file.
type PlayerOrders struct {
	Player int
	// GameID and Year stamp the file for one game and year (ORDERS.md
	// "Wrong game or wrong year").
	GameID uint64
	Year   int
	Orders []Order
}

// Order is one order. Each kind validates itself against the game and the
// submitting player and then applies; a rejected order changes nothing.
type Order interface {
	apply(g *Game, player int, a *Applied) error
}

// Order and file rejections. The wording is Elegy's own.
var (
	ErrWrongGame      = errors.New("orders: the file belongs to another game")
	ErrOutOfDate      = errors.New("orders: the file is for an earlier year")
	ErrLaterYear      = errors.New("orders: the file is for a later year")
	ErrNoSuchPlayer   = errors.New("orders: no such player")
	ErrDuplicateFile  = errors.New("orders: a second file for the same player")
	ErrNotYours       = errors.New("orders: the object belongs to another player")
	ErrNoSuchObject   = errors.New("orders: no such object")
	ErrOutOfRange     = errors.New("orders: a value is out of range")
	ErrNotModelled    = errors.New("orders: Elegy does not model this yet")
	ErrAwaitingSpec   = errors.New("orders: placeholder until the stars-elegy spec lands")
	ErrNotTogether    = errors.New("orders: the objects are not at the same place")
	ErrRefusedByOwner = errors.New("orders: the receiver will not accept it")
)

// OrderResult is the outcome of one order: Err is nil when it applied.
type OrderResult struct {
	Player int
	Index  int // index in PlayerOrders.Orders, or -1 for the whole file
	Err    error
}

// Applied is what applying a year's orders produced besides the changes to
// the game: every order's result, the events, and the colonist drops the
// orders made.
type Applied struct {
	Results []OrderResult
	Events  []Event
	// Gifts is cargo given to another owner's planet or fleet, taken from
	// the giver in the first pass and credited in the second.
	Gifts []CargoGift
	// drops is colonists put onto another player's planet, in the order
	// given. They join the before-movement drop resolution (TAKEOVER.md
	// "Manual cargo transfers to other players", CONFIRMED TK-501) at the
	// front of its queue (TAKEOVER.md "Order inside a phase").
	drops []drop
}

// maxNameLength is the longest fleet, design or battle-plan name Elegy
// accepts. For battle plans it is COMBAT.md "Order validation" (stars-elegy
// #72), Elegy's rule: a name over 31 characters is refused, any text and
// the empty name are accepted.
//
// ASSUMPTION L1: fleet and design names follow the same rule; no public
// spec gives their limit.
const maxNameLength = 31

// acceptFile is ORDERS.md "File acceptance" (BINARY-ONLY): a file for
// another game is rejected, one for an earlier year is out of date, one
// for a later year is ignored. The registered-copy gate has no Elegy
// equivalent (ORDERS.md "Registered-copy gate", implementation note).
func (g *Game) acceptFile(o PlayerOrders) error {
	switch {
	case o.GameID != g.ID:
		return ErrWrongGame
	case o.Year < g.Year:
		return ErrOutOfDate
	case o.Year > g.Year:
		return ErrLaterYear
	case o.Player < 0 || o.Player >= len(g.Players):
		return ErrNoSuchPlayer
	}
	return nil
}

// ApplyOrders applies a year's order files, KERNEL.md "Turn order" step 1:
// one player at a time, in the replay order given (player indices). Cargo
// given to another owner is taken from the giver as each order applies,
// and credited only after every file has been replayed (TAKEOVER.md
// "Manual cargo transfers to other players", stars-elegy #69: replay is
// two passes, debits then credits; creditGifts). Each
// order is validated on its own; a rejected order is dropped and the rest
// of the file still applies (ORDERS.md "Per-order validation"). When two
// players' orders act on one object, the later one in the replay order
// wins (ORDERS.md "Conflicts between players", BINARY-ONLY). A player with
// no accepted file keeps their standing orders (ORDERS.md "A player who
// submits nothing", BINARY-ONLY): nothing is applied for them.
//
// The replay order is the caller's; YearOrders draws it as the original
// does.
//
// ASSUMPTION L2: a player's second file in the same year is rejected
// whole; the first is used. ORDERS.md has one file per player.
func ApplyOrders(g *Game, files []PlayerOrders, replay []int) *Applied {
	a := &Applied{}
	byPlayer := map[int]int{}
	for i, f := range files {
		err := g.acceptFile(f)
		if err == nil {
			if _, dup := byPlayer[f.Player]; dup {
				err = ErrDuplicateFile
			} else {
				byPlayer[f.Player] = i
			}
		}
		if err != nil {
			a.Results = append(a.Results, OrderResult{Player: f.Player, Index: -1, Err: err})
		}
	}
	for _, p := range replay {
		i, ok := byPlayer[p]
		if !ok {
			continue
		}
		for k, o := range files[i].Orders {
			err := o.apply(g, p, a)
			a.Results = append(a.Results, OrderResult{Player: p, Index: k, Err: err})
		}
	}
	a.Events = append(a.Events, g.creditGifts(a.Gifts)...)
	return a
}

// ShufflePlayers is the year's player replay order (KERNEL.md "Turn
// order", 1. Orders, step 2, stars-elegy #53): a forward shuffle of the n
// players, where position i swaps with position i + Random(n − i). One
// draw per player, the last always 0; they are the first draws of the
// year (CONFIRMED as a draw count, KX-004; the permutation BINARY-ONLY).
// With orders that pass Elegy's ownership rule the permutation has no
// observable effect, but the draws are made.
func ShufflePlayers(n int, rng Rand) []int {
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	for i := range n {
		j := i + rng.Intn(n-i)
		perm[i], perm[j] = perm[j], perm[i]
	}
	return perm
}

// YearOrders applies a year's order files in the replay order drawn by
// ShufflePlayers, KERNEL.md step 1. It is the call GenerateTurn makes
// first.
func YearOrders(g *Game, files []PlayerOrders, rng Rand) *Applied {
	return ApplyOrders(g, files, ShufflePlayers(len(g.Players), rng))
}

// ownFleet returns the index of fleet id when player owns it. ORDERS.md
// "Ownership" chosen rule: ownership is checked on every order kind.
func (g *Game) ownFleet(player, id int) (int, error) {
	i := g.fleetIndex(id)
	switch {
	case i < 0:
		return -1, fmt.Errorf("fleet %d: %w", id, ErrNoSuchObject)
	case g.Fleets[i].Owner != player:
		return -1, fmt.Errorf("fleet %d: %w", id, ErrNotYours)
	}
	return i, nil
}

// planetIndex returns the index of planet id, or -1.
func (g *Game) planetIndex(id int) int {
	for i := range g.Planets {
		if g.Planets[i].ID == id {
			return i
		}
	}
	return -1
}

// ownPlanet returns the index of planet id when player owns it.
func (g *Game) ownPlanet(player, id int) (int, error) {
	i := g.planetIndex(id)
	switch {
	case i < 0:
		return -1, fmt.Errorf("planet %d: %w", id, ErrNoSuchObject)
	case g.Planets[i].Owner != player:
		return -1, fmt.Errorf("planet %d: %w", id, ErrNotYours)
	}
	return i, nil
}

// ResearchOrder sets the research budget, the current field and the next
// field (KERNEL.md "Research"; ORDERS.md "Research allocation",
// BINARY-ONLY): the budget is a percentage in 0..100, Field a field index,
// Next a field index, NextSameField or NextLowestField. Any value outside
// its legal set rejects the whole order and the previous settings stand.
type ResearchOrder struct {
	Budget, Field, Next int
}

func (o ResearchOrder) apply(g *Game, player int, _ *Applied) error {
	legalNext := o.Next == NextSameField || o.Next == NextLowestField || (o.Next >= 0 && o.Next < NumFields)
	if o.Budget < 0 || o.Budget > 100 || o.Field < 0 || o.Field >= NumFields || !legalNext {
		return fmt.Errorf("research %+v: %w", o, ErrOutOfRange)
	}
	pl := &g.Players[player]
	pl.ResearchBudget = o.Budget
	pl.Research.Current = o.Field
	pl.Research.Next = o.Next
	return nil
}

// maxBattlePlans is the battle-plan limit (COMBAT.md "Order validation",
// stars-elegy #72, BINARY-ONLY): Elegy enforces the host's 16; the
// client's 15 is a client limit (MEASURED, BP-L).
const maxBattlePlans = 16

// validPlan checks a battle plan's fields (ORDERS.md "Battle-plan
// fields" and COMBAT.md "Order validation", stars-elegy #72, Elegy's
// rules; the original lets a tactic of 6, a target of 8 and any
// attack-who through, BINARY-ONLY): tactic and targets in their legal
// sets, and a plan that attacks a player must name another player in the
// game.
//
// ASSUMPTION L4: an attack-who value outside Elegy's five is rejected;
// COMBAT.md describes the stored values, Elegy's AttackWho has no others.
func (g *Game) validPlan(player int, p BattlePlan) error {
	switch {
	case p.Tactic < TacticDisengage || p.Tactic > TacticMaximizeDamage,
		p.Primary < TargetNone || p.Primary > TargetFreighters,
		p.Secondary < TargetNone || p.Secondary > TargetFreighters,
		p.Attack < AttackNobody || p.Attack > AttackPlayer,
		p.Attack == AttackPlayer && (p.Player < 0 || p.Player >= len(g.Players) || p.Player == player),
		len(p.Name) > maxNameLength:
		return fmt.Errorf("battle plan %q: %w", p.Name, ErrOutOfRange)
	}
	return nil
}

// BattlePlanOrder defines battle plan Index (COMBAT.md "Adding, replacing
// and deleting" and "Order validation", stars-elegy #72, BINARY-ONLY):
// Index below the count replaces, Index equal to the count adds, anything
// else, and a 17th plan, is refused.
type BattlePlanOrder struct {
	Index int
	Plan  BattlePlan
}

func (o BattlePlanOrder) apply(g *Game, player int, _ *Applied) error {
	pl := &g.Players[player]
	if o.Index < 0 || o.Index > len(pl.Plans) || o.Index >= maxBattlePlans {
		return fmt.Errorf("battle plan %d: %w", o.Index, ErrOutOfRange)
	}
	if err := g.validPlan(player, o.Plan); err != nil {
		return err
	}
	pl.Plans = append([]BattlePlan(nil), pl.Plans...)
	if o.Index == len(pl.Plans) {
		pl.Plans = append(pl.Plans, o.Plan)
	} else {
		pl.Plans[o.Index] = o.Plan
	}
	return nil
}

// DeletePlanOrder deletes battle plan Index (COMBAT.md "Adding, replacing
// and deleting", CONFIRMED BP-1): later plans move down one number, and
// every one of the player's fleets on plan Index or higher has its number
// lowered by one, so a fleet on the deleted plan takes the plan before
// it. Plan 0 is never deleted (ORDERS.md "Battle-plan fields", Elegy's
// chosen rule; the original lets a crafted order delete it, BINARY-ONLY).
type DeletePlanOrder struct {
	Index int
}

func (o DeletePlanOrder) apply(g *Game, player int, _ *Applied) error {
	pl := &g.Players[player]
	if o.Index < 1 || o.Index >= len(pl.Plans) {
		return fmt.Errorf("delete battle plan %d: %w", o.Index, ErrOutOfRange)
	}
	plans := append([]BattlePlan(nil), pl.Plans[:o.Index]...)
	pl.Plans = append(plans, pl.Plans[o.Index+1:]...)
	for i := range g.Fleets {
		if f := &g.Fleets[i]; f.Owner == player && f.Plan >= o.Index {
			f.Plan--
		}
	}
	return nil
}

// FleetPlanOrder puts a fleet on one of its owner's plans. The fleet must
// be the player's, and the plan one the player has (ORDERS.md "Ownership"
// and "Battle-plan fields", Elegy's chosen rules; the original checks
// neither, BINARY-ONLY).
type FleetPlanOrder struct {
	Fleet, Plan int
}

func (o FleetPlanOrder) apply(g *Game, player int, _ *Applied) error {
	i, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	if o.Plan < 0 || o.Plan >= len(g.Players[player].Plans) {
		return fmt.Errorf("fleet %d plan %d: %w", o.Fleet, o.Plan, ErrOutOfRange)
	}
	g.Fleets[i].Plan = o.Plan
	return nil
}

// RenameOrder names one of the player's fleets; an empty name clears it
// (PRODUCTION-LAUNCH.md "Fleet names": a fleet has no stored name until
// the player renames it). Ownership is checked (ORDERS.md "Ownership",
// chosen rule; the original does not check, BINARY-ONLY).
type RenameOrder struct {
	Fleet int
	Name  string
}

func (o RenameOrder) apply(g *Game, player int, _ *Applied) error {
	i, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	if len(o.Name) > maxNameLength {
		return fmt.Errorf("fleet %d name: %w", o.Fleet, ErrOutOfRange)
	}
	g.Fleets[i].Name = o.Name
	return nil
}

// MergeOrder is the direct merge order (ORDERS.md "Merge"), carried out by
// Game.MergeFleets, which checks ownership and co-location.
type MergeOrder struct {
	Into int
	From []int
}

func (o MergeOrder) apply(g *Game, player int, _ *Applied) error {
	return g.MergeFleets(player, o.Into, o.From)
}

// WaypointOrder replaces a fleet's orders: the task at its current
// location and its waypoint list (ORDERS.md "waypoint edit"; Elegy's
// order carries the whole list rather than one edit).
//
// Checks: the fleet is the player's (ORDERS.md "Ownership", chosen rule);
// a waypoint's coordinates are clamped to the galaxy, not rejected
// (ORDERS.md "Waypoint coordinates", BINARY-ONLY; the box is UNIVERSE.md's
// 1000 .. 1000 + W). A planet or fleet target takes that object's
// position.
//
// ASSUMPTION L6: a warp outside 0..10, a negative transport amount, a
// task Elegy does not model, or a target planet or fleet that does not
// exist rejects the order. Stargate jumps are not modelled. ORDERS.md
// gives no other waypoint checks.
type WaypointOrder struct {
	Fleet     int
	Task      Task
	Waypoints []Waypoint
}

func (o WaypointOrder) apply(g *Game, player int, _ *Applied) error {
	i, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	if err := validTask(o.Task); err != nil {
		return err
	}
	wps := make([]Waypoint, len(o.Waypoints))
	for k, wp := range o.Waypoints {
		if wp.Warp < 0 || wp.Warp > 10 {
			return fmt.Errorf("waypoint %d warp %d: %w", k, wp.Warp, ErrOutOfRange)
		}
		if err := validTask(wp.Task); err != nil {
			return err
		}
		switch wp.Target {
		case TargetPlanet:
			pos, ok := g.planetPos(wp.ID)
			if !ok {
				return fmt.Errorf("waypoint %d planet %d: %w", k, wp.ID, ErrNoSuchObject)
			}
			wp.Pos = pos
		case TargetFleet:
			t := g.fleetIndex(wp.ID)
			if t < 0 {
				return fmt.Errorf("waypoint %d fleet %d: %w", k, wp.ID, ErrNoSuchObject)
			}
			wp.Pos = g.Fleets[t].Pos
		case TargetSpace:
			wp.Pos = g.clampToGalaxy(wp.Pos)
		default:
			return fmt.Errorf("waypoint %d target: %w", k, ErrOutOfRange)
		}
		wps[k] = wp
	}
	f := &g.Fleets[i]
	f.Task = o.Task
	f.Waypoints = wps
	return nil
}

func validTask(t Task) error {
	switch t.Kind {
	case TaskNone, TaskColonize, TaskMerge:
	case TaskTransport:
		for _, tr := range t.Transport {
			if tr.Action < TransportNone || tr.Action > UnloadExactly || tr.Amount < 0 {
				return fmt.Errorf("transport task: %w", ErrOutOfRange)
			}
		}
	default:
		return fmt.Errorf("task %d: %w", t.Kind, ErrNotModelled)
	}
	return nil
}

// clampToGalaxy clamps a point to UNIVERSE.md's coordinate box: 1000 to
// 1000 + W on both axes, W = (size + 1) · 400.
func (g *Game) clampToGalaxy(p Point) Point {
	w := (g.Size + 1) * 400
	c := func(v int) int { return min(max(v, 1000), 1000+w) }
	return Point{c(p.X), c(p.Y)}
}

// DetonateOrder sets a minefield's detonate setting. ORDERS.md's chosen
// rule accepts it only on the player's own minefield of a kind that can
// detonate. Elegy has no minefields yet, so every such order is rejected
// with ErrNotModelled.
type DetonateOrder struct {
	Minefield int
	On        bool
}

func (o DetonateOrder) apply(*Game, int, *Applied) error {
	return fmt.Errorf("minefield %d: %w", o.Minefield, ErrNotModelled)
}

// QueueOrder replaces a planet's production queue. PLACEHOLDER: the
// production-queue spec is being written in stars-elegy (ORDERS.md text
// for queue replace). Elegy checks ownership, which ORDERS.md "Ownership"
// says the original re-checks too, and then rejects the order with
// ErrAwaitingSpec. The starbase-dock check that will apply to ship items
// is DockAllows.
type QueueOrder struct {
	Planet int
	Queue  []QueueItem
}

func (o QueueOrder) apply(g *Game, player int, _ *Applied) error {
	if _, err := g.ownPlanet(player, o.Planet); err != nil {
		return err
	}
	return fmt.Errorf("production queue: %w", ErrAwaitingSpec)
}

// PlanetFlagsOrder sets a planet's production flags (ORDERS.md
// "planet-production flags"). PLACEHOLDER with QueueOrder: ownership is
// checked (re-checked by the original too), then ErrAwaitingSpec.
type PlanetFlagsOrder struct {
	Planet       int
	LeftoverOnly bool
}

func (o PlanetFlagsOrder) apply(g *Game, player int, _ *Applied) error {
	if _, err := g.ownPlanet(player, o.Planet); err != nil {
		return err
	}
	return fmt.Errorf("planet flags: %w", ErrAwaitingSpec)
}

// SettingsOrder is a player-settings or relations order (ORDERS.md
// "housekeeping orders"). PLACEHOLDER: the settings-orders spec is being
// written in stars-elegy; every such order is rejected with
// ErrAwaitingSpec.
type SettingsOrder struct {
	Relations []Relation
}

func (o SettingsOrder) apply(*Game, int, *Applied) error {
	return fmt.Errorf("settings: %w", ErrAwaitingSpec)
}
