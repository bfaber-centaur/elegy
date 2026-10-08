// Package objects implements the space objects of stars-elegy
// docs/OBJECTS.md: minefields, and later packets, wormholes, stargates and
// the Mystery Trader.
//
// The package reads the engine's exported state and exposes step and
// query functions. It does not call itself from engine.GenerateTurn: the
// turn engine decides where each step runs (OBJECTS.md "Turn placement").
// Every rule cites its OBJECTS.md section and status; what the spec
// leaves open is marked ASSUMPTION and listed in docs/OBJECTS-STATUS.md.
package objects

import (
	"math"
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// MineKind is a minefield's kind.
type MineKind int

const (
	Standard MineKind = iota
	Heavy
	SpeedBump
	NumMineKinds
)

// Minefield is one minefield. Its size is its mine count; a point is
// inside when d² ≤ Count (OBJECTS.md "Conventions", CONFIRMED OB-002,
// OB-018-C).
type Minefield struct {
	Owner    int
	Number   int
	Kind     MineKind
	Pos      engine.Point
	Count    int
	Detonate bool
	// Known marks the players who know the field: seen, hit by it or
	// swept it (SCANNING.md "Space objects"). Its owner always knows it.
	Known []bool
}

// Contains reports whether p is inside the field.
func (m Minefield) Contains(p engine.Point) bool { return d2(m.Pos, p) <= m.Count }

// MaxFields is the minefields one player may own (OBJECTS.md "Laying",
// MEASURED MF-11, Elegy's chosen rule).
const MaxFields = 512

// MaxObjects is the space objects of all kinds the universe holds
// (OBJECTS.md "Laying", BINARY-ONLY).
const MaxObjects = 4050

// mergeCap is the count above which a field takes no more mines
// (OBJECTS.md "Laying", CONFIRMED MF-10).
const mergeCap = 999_999

// Space holds the space objects this package models, in object order
// (OBJECTS.md "Conventions": by kind, then owner, then number).
type Space struct {
	Minefields []Minefield
	Packets    []Packet
	Wormholes  []Wormhole
	Traders    []Trader
	// TraderParts is each player's Trader part word.
	TraderParts TraderParts
	// GiftDesigns marks the designs (indices into Game.Designs) that a
	// Trader ship gift created; only these can match a later gift
	// (OBJECTS.md "Encounters", matching). The turn engine drops an
	// index when that design is deleted.
	GiftDesigns []int
	// OtherObjects counts the space objects of kinds this package does
	// not hold yet (salvage). Like wormhole ends and Traders,
	// they sort after every minefield, count toward
	// MaxObjects and, with Legacy.FieldLimit511, hold every player to 511
	// fields.
	OtherObjects int
}

// otherObjects counts the space objects that are not minefields; each
// wormhole end is one object.
func (s *Space) otherObjects() int {
	return s.OtherObjects + len(s.Packets) + 2*len(s.Wormholes) + len(s.Traders)
}

// SortMinefields puts the minefields in object order: owner, then number.
func (s *Space) SortMinefields() {
	sort.SliceStable(s.Minefields, func(i, j int) bool {
		a, b := s.Minefields[i], s.Minefields[j]
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		return a.Number < b.Number
	})
}

func d2(a, b engine.Point) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

// relation is player a's view of player b: a player is its own friend and
// a missing entry is neutral (engine.Player.Relations).
func relation(g *engine.Game, a, b int) engine.Relation {
	if a == b {
		return engine.RelationFriend
	}
	if a < 0 || a >= len(g.Players) {
		return engine.RelationNeutral
	}
	if r := g.Players[a].Relations; b >= 0 && b < len(r) {
		return r[b]
	}
	return engine.RelationNeutral
}

func prt(g *engine.Game, player int) engine.PRT {
	if player < 0 || player >= len(g.Players) {
		return engine.PRTOther
	}
	return g.Players[player].Race.PRT
}

// --- Decay ---

// DecayLoss is the mines a field loses to decay with n planets inside it
// (OBJECTS.md "Decay", CONFIRMED OB-002, OB-014-A, OB-015, OB-016):
// p = min(50, 4n + 2), or min(50, n + 2) for a Space Demolition owner;
// +25 when detonating; loss = max(p, ⌊N·p/100⌋), at least 10 for standard
// and heavy fields.
func DecayLoss(m Minefield, n int, ownerSD bool) int {
	p := min(50, 4*n+2)
	if ownerSD {
		p = min(50, n+2)
	}
	if m.Detonate {
		p += 25
	}
	loss := max(p, m.Count*p/100)
	if m.Kind != SpeedBump {
		loss = max(loss, 10)
	}
	return loss
}

// Decay decays every minefield, counting the planets inside each as it
// stands now (after this year's stops, CONFIRMED MF-4), and removes the
// fields whose loss reaches their count. Planet owners do not matter.
func (s *Space) Decay(g *engine.Game) {
	kept := s.Minefields[:0]
	for _, m := range s.Minefields {
		n := 0
		for _, p := range g.Planets {
			if m.Contains(p.Pos) {
				n++
			}
		}
		loss := DecayLoss(m, n, prt(g, m.Owner) == engine.PRTSpaceDemolition)
		if loss >= m.Count {
			continue
		}
		m.Count -= loss
		kept = append(kept, m)
	}
	s.Minefields = kept
}

// --- Laying ---

// LayAmounts is what one fleet lays in a year, per kind (OBJECTS.md
// "Laying", CONFIRMED OB-002, OB-024): for each ship, the sum of its
// dispensers of that kind (count × rating) times the hull's mine-layer
// multiplier (2 on the Mini Mine Layer and Super Mine Layer,
// COMPONENTS.md), plus 40 standard mines per Multi Contained Munition;
// summed over ships.
func LayAmounts(g *engine.Game, f *engine.Fleet) [NumMineKinds]int {
	cat := engine.Components()
	var out [NumMineKinds]int
	for _, st := range f.Stacks {
		if st.Design < 0 || st.Design >= len(g.Designs) {
			continue
		}
		d := g.Designs[st.Design]
		mult := 1
		if hc, ok := cat.Lookup(d.Hull.Name); ok {
			if v, ok := hc.Stats["mine_layer_multiplier"].(float64); ok {
				mult = int(v)
			}
		}
		var per [NumMineKinds]int
		for _, sl := range d.Slots {
			c, ok := cat.Lookup(sl.Part.Name)
			if !ok {
				continue
			}
			if v, ok := c.Stats["mines_per_year"].(float64); ok {
				k, ok := kindOf(c.Stats["field"])
				if ok {
					per[k] += sl.Count * int(v) * mult
				}
			}
			if v, ok := c.Stats["mines_laid"].(float64); ok {
				per[Standard] += sl.Count * int(v)
			}
		}
		for k := range per {
			out[k] += per[k] * st.Count
		}
	}
	return out
}

func kindOf(v any) (MineKind, bool) {
	switch v {
	case "standard":
		return Standard, true
	case "heavy":
		return Heavy, true
	case "speed_bump":
		return SpeedBump, true
	}
	return 0, false
}

// LayResult is what one lay of one kind did.
type LayResult struct {
	Kind   MineKind
	Amount int
	// Field is the index in Space.Minefields of the field that took the
	// mines, or -1 when they were lost for want of room (the owner gets
	// the "failed to lay mines" message).
	Field int
	New   bool
}

// Layer is a fleet that lays mines this year. Who lays is the turn
// engine's: a fleet whose current task is "lay mines" and that did not
// move lays the full amount; a Space Demolition fleet that moved toward a
// "lay mines" waypoint lays half (OBJECTS.md "Laying", CONFIRMED OB-014-C,
// OB-019).
type Layer struct {
	Fleet int // index into Game.Fleets
	Half  bool
}

// Lay lays mines for the given fleets, in fleet order (owner, then fleet
// number), each fleet standard, then heavy, then speed bump, each kind
// merged before the next (OBJECTS.md "Laying", "Order", CONFIRMED MF-12).
// A fleet with no dispensers lays nothing (the caller sends its message).
//
// A half lay halves each kind's amount on its own; amounts are multiples
// of 10, so the half is exact (OBJECTS.md "Laying", BINARY-ONLY).
func (s *Space) Lay(g *engine.Game, layers []Layer) []LayResult {
	order := append([]Layer(nil), layers...)
	sort.SliceStable(order, func(i, j int) bool {
		a, b := &g.Fleets[order[i].Fleet], &g.Fleets[order[j].Fleet]
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		return a.ID < b.ID
	})
	var out []LayResult
	for _, l := range order {
		f := &g.Fleets[l.Fleet]
		amounts := LayAmounts(g, f)
		for k := range NumMineKinds {
			a := amounts[k]
			if l.Half {
				a /= 2
			}
			if a <= 0 {
				continue
			}
			out = append(out, s.lay(f.Owner, f.Pos, k, a, g.Rules.Legacy.FieldLimit511))
		}
	}
	return out
}

// lay merges amount mines of one kind into the owner's nearest field of
// that kind containing pos, unless that field already holds more than
// 999,999; the centre moves to ⌊(x·N + x_f·a)/(N + a)⌋ per axis.
// Otherwise a new field is made at pos, if there is room.
//
// Equally near fields: the first in object order, the lower field number
// (OBJECTS.md "Laying", BINARY-ONLY). A new field takes the lowest number
// unused among the owner's fields of all three kinds (OBJECTS.md
// "Limits", BINARY-ONLY; MF-11, MF-13 measured only the last number).
func (s *Space) lay(owner int, pos engine.Point, k MineKind, amount int, limit511 bool) LayResult {
	best, bestD := -1, 0
	for i, m := range s.Minefields {
		if m.Owner != owner || m.Kind != k || !m.Contains(pos) {
			continue
		}
		if dd := d2(m.Pos, pos); best < 0 || dd < bestD {
			best, bestD = i, dd
		}
	}
	if best >= 0 && s.Minefields[best].Count <= mergeCap {
		m := &s.Minefields[best]
		n := m.Count
		m.Pos = engine.Point{
			X: (m.Pos.X*n + pos.X*amount) / (n + amount),
			Y: (m.Pos.Y*n + pos.Y*amount) / (n + amount),
		}
		m.Count = n + amount
		return LayResult{Kind: k, Amount: amount, Field: best}
	}
	num, ok := s.freeNumber(owner, limit511)
	if !ok || len(s.Minefields)+s.otherObjects() >= MaxObjects {
		return LayResult{Kind: k, Amount: amount, Field: -1}
	}
	s.Minefields = append(s.Minefields, Minefield{Owner: owner, Number: num, Kind: k, Pos: pos, Count: amount})
	s.SortMinefields()
	for i, m := range s.Minefields {
		if m.Owner == owner && m.Number == num {
			return LayResult{Kind: k, Amount: amount, Field: i, New: true}
		}
	}
	return LayResult{Kind: k, Amount: amount, Field: -1}
}

// freeNumber is the owner's lowest unused field number below the limit.
// freeNumber is the lowest field number free for owner. limit511
// reproduces the original's LEGACY BUG that a player's last field number,
// 511, is given only when no other space object sorts after that player's
// minefields (Legacy.FieldLimit511; OBJECTS.md "Laying", MEASURED MF-13).
// Without it (the Elegy ruleset) the limit is a plain 512.
func (s *Space) freeNumber(owner int, limit511 bool) (int, bool) {
	used := map[int]bool{}
	later := s.otherObjects() > 0
	for _, m := range s.Minefields {
		if m.Owner == owner {
			used[m.Number] = true
		} else if m.Owner > owner {
			later = true
		}
	}
	limit := MaxFields
	if limit511 && later {
		limit = MaxFields - 1
	}
	for n := range limit {
		if !used[n] {
			return n, true
		}
	}
	return 0, false
}

// --- Sweeping ---

// SweepRating is a design's sweep rating (OBJECTS.md "Sweeping",
// CONFIRMED OB-001, OB-007): Σ over its beam weapons of count × damage ×
// range², with range 4 for gatling beams and range + 1 on a starbase hull.
// Sappers, torpedoes, missiles and bombs sweep nothing.
func SweepRating(d engine.Design) int {
	s := 0
	for _, sl := range d.Slots {
		p := sl.Part
		if p.Kind != engine.PartBeam || p.Sapper {
			continue
		}
		r := p.Range
		if p.Gatling {
			r = 4
		}
		if d.Hull.Starbase {
			r++
		}
		s += sl.Count * p.Damage * r * r
	}
	return s
}

// attacks reports whether a battle plan would attack player b, for
// sweeping (OBJECTS.md "Sweeping", CONFIRMED OB-001-E, OB-007, OB-008):
// "everyone" and "player N only" ignore relations.
func attacks(g *engine.Game, owner int, plan engine.BattlePlan, b int) bool {
	switch plan.Attack {
	case engine.AttackEnemies:
		return relation(g, owner, b) == engine.RelationEnemy
	case engine.AttackNeutralsAndEnemies:
		return relation(g, owner, b) != engine.RelationFriend
	case engine.AttackEveryone:
		return true
	case engine.AttackPlayer:
		return plan.Player == b
	}
	return false
}

func fleetPlan(g *engine.Game, f *engine.Fleet) engine.BattlePlan {
	if f.Owner >= 0 && f.Owner < len(g.Players) {
		if p := g.Players[f.Owner].Plans; f.Plan >= 0 && f.Plan < len(p) {
			return p[f.Plan]
		}
	}
	return engine.BattlePlan{}
}

// Swept is one sweep: who swept which field and how many mines it lost.
// The sweeper's owner learns the field (Minefield.Known; SCANNING.md
// "Space objects", BINARY-ONLY).
type Swept struct {
	Sweeper int // owner of the sweeping fleet or starbase
	Fleet   int // index into Game.Fleets, or -1 for a starbase
	Planet  int // index into Game.Planets for a starbase, else -1
	Field   Minefield
	Mines   int
}

// Sweep runs a year's sweeping (OBJECTS.md "Sweeping", CONFIRMED OB-001,
// OB-007, OB-008, OB-010-S): every fleet in fleet order, then every
// starbase in planet order, sweeps each field containing it with its full
// rating. A fleet sweeps another player's field whose owner its battle
// plan would attack; a starbase sweeps every field containing its planet
// whose owner is not the planet owner's friend. The amount is the rating
// (a third for speed bumps), at least 2, but never past the sweeper: if
// N − a < d² − 1 then a = N − d² + 1. A field at 0 disappears.
func (s *Space) Sweep(g *engine.Game) []Swept {
	var out []Swept
	sweep := func(owner, fi, pi int, pos engine.Point, rating int, may func(fieldOwner int) bool) {
		for i := range s.Minefields {
			m := &s.Minefields[i]
			if m.Count <= 0 || m.Owner == owner || !m.Contains(pos) || !may(m.Owner) {
				continue
			}
			a := rating
			if m.Kind == SpeedBump {
				a /= 3
			}
			a = max(a, 2)
			if dd := d2(m.Pos, pos); m.Count-a < dd-1 {
				a = m.Count - dd + 1
			}
			a = min(a, m.Count)
			m.Count -= a
			m.learn(owner)
			out = append(out, Swept{Sweeper: owner, Fleet: fi, Planet: pi, Field: *m, Mines: a})
		}
	}
	for _, fi := range fleetOrder(g) {
		f := &g.Fleets[fi]
		rating := 0
		for _, st := range f.Stacks {
			if st.Design >= 0 && st.Design < len(g.Designs) {
				rating += st.Count * SweepRating(g.Designs[st.Design])
			}
		}
		if rating == 0 {
			continue
		}
		plan := fleetPlan(g, f)
		sweep(f.Owner, fi, -1, f.Pos, rating, func(o int) bool { return attacks(g, f.Owner, plan, o) })
	}
	for pi := range g.Planets {
		p := &g.Planets[pi]
		if !p.HasStarbase || p.Owner < 0 || p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs) {
			continue
		}
		rating := SweepRating(g.Designs[p.StarbaseDesign])
		if rating == 0 {
			continue
		}
		owner := p.Owner
		sweep(owner, -1, pi, p.Pos, rating, func(o int) bool { return relation(g, owner, o) != engine.RelationFriend })
	}
	kept := s.Minefields[:0]
	for _, m := range s.Minefields {
		if m.Count > 0 {
			kept = append(kept, m)
		}
	}
	s.Minefields = kept
	return out
}

// fleetOrder is the fleet indices by owner, then number, then id.
func fleetOrder(g *engine.Game) []int {
	order := make([]int, len(g.Fleets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := &g.Fleets[order[i]], &g.Fleets[order[j]]
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		return a.ID < b.ID
	})
	return order
}

// --- Hits on moving fleets ---

// safeWarp is the warp at or below which a kind never stops a fleet
// (OBJECTS.md "Hits on moving fleets", BINARY-ONLY bonuses): standard 4,
// heavy 6, speed bump 5; +1 for a Super Stealth fleet owner, +2 for Space
// Demolition.
func safeWarp(k MineKind, fleetPRT engine.PRT) int {
	w := [NumMineKinds]int{4, 6, 5}[k]
	switch fleetPRT {
	case engine.PRTSuperStealth:
		w++
	case engine.PRTSpaceDemolition:
		w += 2
	}
	return w
}

// hitPerMille is the stop odds per ly per warp above safe (CONFIRMED
// MF-1 as rates).
var hitPerMille = [NumMineKinds]int{3, 10, 35}

// EffectiveWarp is the warp a movement step of D ly counts as: the
// smallest of 3..10 with e² ≥ D − 1 (OBJECTS.md "Hits on moving fleets",
// CONFIRMED OB-010, MF-3).
func EffectiveWarp(distance int) int {
	for e := 3; e < 10; e++ {
		if e*e >= distance-1 {
			return e
		}
	}
	return 10
}

// stopsFleet reports whether field m can stop a fleet of owner victim:
// the field owner is not the victim and does not treat it as a friend
// (CONFIRMED MF-5, MF-6).
func stopsFleet(g *engine.Game, m Minefield, victim int) bool {
	return m.Owner != victim && relation(g, m.Owner, victim) != engine.RelationFriend
}

// stretch is a whole-ly stretch [entry, exit) of the path inside fields
// of one kind.
type stretch struct {
	kind        MineKind
	entry, exit int
}

// floatFoot is the size of |dx·dy|, dx² or dy² from which the original
// computes the foot's x in floating point (OBJECTS.md "Path cut" step 1).
const floatFoot = 500001

// dueNS reproduces the original's LEGACY BUG on legs with no east-west
// component (Legacy.DueNorthSouthCut; OBJECTS.md "Arithmetic details",
// "Due-north and due-south legs", MEASURED MF-15): the foot of the
// perpendicular is taken at the fleet's start, so a fleet that starts
// outside a field and flies due north or south into it is never checked,
// and one that starts inside is checked from its start for
// trunc(√(N − d²)) ly whichever way it flies. Without it the exact foot is
// used (Elegy's choice, not the original's).
// cut is the whole-ly stretch [entry, exit) of a leg from `from` toward
// `toward`, travelling l ly this step, inside a field of centre c and
// count n (OBJECTS.md "Arithmetic details", "Path cut", BINARY-ONLY; east
// legs CONFIRMED as rates, MF-1, MF-3). Divisions truncate toward zero.
func cut(from, toward engine.Point, l int, c engine.Point, n int, dueNS bool) (int, int, bool) {
	sx, sy := from.X, from.Y
	dx, dy := toward.X-sx, toward.Y-sy
	fx, fy := sx, sy
	switch {
	case dx != 0:
		den := dx*dx + dy*dy
		if abs(dx*dy) >= floatFoot || dx*dx >= floatFoot || dy*dy >= floatFoot {
			fx = int((float64(c.X)*float64(dx*dx) + float64(sx)*float64(dy*dy) + float64(c.Y-sy)*float64(dx)*float64(dy)) / float64(den))
		} else {
			fx = (c.X*dx*dx + sx*dy*dy + (c.Y-sy)*dx*dy) / den
		}
		fy = sy + (fx-sx)*dy/dx
	case !dueNS:
		fy = c.Y
	}
	p2 := (fx-c.X)*(fx-c.X) + (fy-c.Y)*(fy-c.Y)
	if p2 >= n {
		return 0, 0, false
	}
	h := isqrt((fx-sx)*(fx-sx) + (fy-sy)*(fy-sy))
	if (dx != 0 && (fx-sx)*dx < 0) || (dx == 0 && (fy-sy)*dy < 0) {
		h = -h
	}
	w := isqrt(n - p2)
	entry, exit := max(0, h-w), min(l, h+w)
	if exit <= 0 || entry >= l {
		return 0, 0, false
	}
	return entry, exit, true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// isqrt is ⌊√x⌋ for x ≥ 0.
func isqrt(x int) int {
	r := int(math.Sqrt(float64(x)))
	for r*r > x {
		r--
	}
	for (r+1)*(r+1) <= x {
		r++
	}
	return r
}

// maxStretches is the stretches kept per kind (OBJECTS.md "Stretches").
const maxStretches = 8

// addStretch adds [a, b) to one kind's stretches, kept sorted by entry
// (OBJECTS.md "Stretches", BINARY-ONLY): it merges with an existing
// stretch it overlaps or touches, or that starts exactly 1 ly after it
// ends; otherwise it is inserted, and dropped when the kind already holds
// eight.
//
// ASSUMPTION O9: the new stretch merges into the first such stretch in
// entry order, and the merged stretch is not merged again with others.
func addStretch(iv [][2]int, a, b int) [][2]int {
	for i, x := range iv {
		if a <= x[1] && b+1 >= x[0] {
			iv[i] = [2]int{min(a, x[0]), max(b, x[1])}
			sort.Slice(iv, func(i, j int) bool { return iv[i][0] < iv[j][0] })
			return iv
		}
	}
	if len(iv) >= maxStretches {
		return iv
	}
	iv = append(iv, [2]int{a, b})
	sort.Slice(iv, func(i, j int) bool { return iv[i][0] < iv[j][0] })
	return iv
}

// StopPoint is where a fleet stopped s ly into a leg from `from` toward
// `toward` lands (OBJECTS.md "Arithmetic details", "Stop point",
// BINARY-ONLY; exact on east legs): each coordinate is the start plus its
// offset times s divided by the leg length rounded to the nearest ly, the
// result rounded to the nearest ly. The salvage and the paying field use
// this point.
func StopPoint(from, toward engine.Point, s int) engine.Point {
	dx, dy := toward.X-from.X, toward.Y-from.Y
	l := int(math.Round(math.Hypot(float64(dx), float64(dy))))
	if l == 0 {
		return from
	}
	step := func(o int) int { return int(math.Round(float64(o*s) / float64(l))) }
	return engine.Point{X: from.X + step(dx), Y: from.Y + step(dy)}
}

// Stop is the outcome of checking one movement step.
type Stop struct {
	Hit bool
	// Distance is how far from the step's start the fleet stops.
	Distance int
	Kind     MineKind
}

// CheckStep checks one movement step of fleet f for minefield stops
// (OBJECTS.md "Hits on moving fleets", CONFIRMED OB-010, OB-024,
// MF-1..MF-9): the fleet moves `distance` ly from `from` toward `toward`
// (its next waypoint). Stationary fleets, fleets already at their
// waypoint and steps through a stargate are the caller's to skip, as are
// later steps once a fleet has been stopped.
//
// Each field that can stop the fleet cuts the path into a stretch (cut),
// in object order; each kind keeps up to eight merged stretches
// (addStretch). Stretches are visited by entry, equal entries standard,
// heavy, speed bump; a kind whose safe warp is at or above e makes no
// draws. A stretch [a, b) draws k = 0 .. b − a − 1, rand(1000) <
// (e − safe)·h, and a hit at k stops the fleet a + k ly from its start
// (OBJECTS.md "Stretches", BINARY-ONLY; stop offsets MEASURED OB-010-S,
// MF-15). The caller places the fleet with StopPoint. Cloak plays no part
// (CONFIRMED MF-1).
func CheckStep(g *engine.Game, s *Space, f *engine.Fleet, from, toward engine.Point, distance int, rng engine.Rand) Stop {
	if distance <= 0 || from == toward {
		return Stop{}
	}
	e := EffectiveWarp(distance)
	fp := prt(g, f.Owner)

	var byKind [NumMineKinds][][2]int
	for _, m := range s.Minefields {
		if m.Count <= 0 || !stopsFleet(g, m, f.Owner) || e <= safeWarp(m.Kind, fp) {
			continue
		}
		if a, b, ok := cut(from, toward, distance, m.Pos, m.Count, g.Rules.Legacy.DueNorthSouthCut); ok {
			byKind[m.Kind] = addStretch(byKind[m.Kind], a, b)
		}
	}
	var all []stretch
	for k := range NumMineKinds {
		for _, x := range byKind[k] {
			all = append(all, stretch{k, x[0], x[1]})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].entry < all[j].entry })
	for _, st := range all {
		odds := (e - safeWarp(st.kind, fp)) * hitPerMille[st.kind]
		for d := st.entry; d < st.exit; d++ {
			if rng.Intn(1000) < odds {
				return Stop{Hit: true, Distance: d, Kind: st.kind}
			}
		}
	}
	return Stop{}
}

// --- Damage ---

// DesignHit is the mine damage one design group of a fleet took.
type DesignHit struct {
	Design int // index into Game.Designs
	Ships  int
	// Damage is the damage before shields, as the hit message reports
	// it; Absorbed is what shields took.
	Damage, Absorbed int
	// Destroyed is set when the group's average damage exceeded its
	// armor; Destroyed ships are removed from the fleet.
	Destroyed bool
}

// noFuelAtWarp4 reports a design whose engine burns no fuel at warp 4.
func noFuelAtWarp4(d engine.Design) bool {
	return d.Engines > 0 && d.Engine.Fuel[4] == 0
}

func designArmor(d engine.Design) int {
	a := d.Hull.Armor
	for _, s := range d.Slots {
		a += s.Count * s.Part.Armor
	}
	return a
}

func designShield(d engine.Design) int {
	sh := 0
	for _, s := range d.Slots {
		sh += s.Count * s.Part.Shield
	}
	return sh
}

// MineDamage applies a standard or heavy hit to fleet f (OBJECTS.md "Hits
// on moving fleets" damage, CONFIRMED MF-9, OB-010-S, OB-002-M, MF-7,
// MF-8): per ship per engine 100 (standard) or 500 (heavy), 125 and 600
// when any design in the fleet has an engine that burns no fuel at warp
// 4; a fleet under 5 ships gets at least 500 (2000) in total, 600 (2500)
// with such an engine, the shortfall to the first design. Per design
// D = (ships·per + shortfall)·engines; shields absorb at most D/2; the
// rest is added to existing damage and spread evenly; a design whose
// average exceeds its armor is destroyed. Speed bumps do no damage.
// exempt, if set, excludes designs (the owner's own mine layers in a
// detonation).
//
// On existing damage (OBJECTS.md "Damage on top of existing damage",
// BINARY-ONLY; prior damage on all ships CONFIRMED MF-8): with n ships,
// armor A and the stored pct and units, X = ⌊⌊pct·n/100⌋·A·units/500⌋;
// total = X + D − min(⌊D/2⌋, shield·n); avg = ⌊total/n⌋. The design is
// destroyed when avg > A; otherwise every ship is damaged (100%) with
// max(1, ⌊avg·500/A⌋) units.
//
// ASSUMPTION O6: a design's ships are the fleet's stacks of that design,
// in order of first appearance, with the first stack's stored damage; the
// minimum and shortfall count only non-exempt ships.
func MineDamage(g *engine.Game, f *engine.Fleet, k MineKind, exempt func(design int) bool) []DesignHit {
	if k == SpeedBump {
		return nil
	}
	type group struct {
		design, ships int
	}
	var groups []group
	idx := map[int]int{}
	scoop := false
	for _, st := range f.Stacks {
		if st.Count <= 0 || st.Design < 0 || st.Design >= len(g.Designs) || (exempt != nil && exempt(st.Design)) {
			continue
		}
		if noFuelAtWarp4(g.Designs[st.Design]) {
			scoop = true
		}
		if i, ok := idx[st.Design]; ok {
			groups[i].ships += st.Count
			continue
		}
		idx[st.Design] = len(groups)
		groups = append(groups, group{st.Design, st.Count})
	}
	if len(groups) == 0 {
		return nil
	}
	per := [2]int{100, 500}[k]
	least := [2]int{500, 2000}[k]
	if scoop {
		per = [2]int{125, 600}[k]
		least = [2]int{600, 2500}[k]
	}
	ships := 0
	for _, gr := range groups {
		ships += gr.ships
	}
	shortfall := 0
	if ships < 5 && ships*per < least {
		shortfall = least - ships*per
	}
	var out []DesignHit
	for i, gr := range groups {
		d := g.Designs[gr.design]
		raw := gr.ships * per
		if i == 0 {
			raw += shortfall
		}
		raw *= max(1, d.Engines)
		absorbed := min(raw/2, gr.ships*designShield(d))
		hit := DesignHit{Design: gr.design, Ships: gr.ships, Damage: raw, Absorbed: absorbed}
		armor := designArmor(d)
		old := firstDamage(f, gr.design)
		existing := old.Pct * gr.ships / 100 * armor * old.Units / 500
		avg := (existing + raw - absorbed) / gr.ships
		if armor <= 0 || avg > armor {
			hit.Destroyed = true
		} else {
			dmg := engine.Damage{Pct: 100, Units: max(1, avg*500/armor)}
			for si := range f.Stacks {
				if f.Stacks[si].Design == gr.design {
					f.Stacks[si].Damage = dmg
				}
			}
		}
		out = append(out, hit)
	}
	kept := f.Stacks[:0]
	for _, st := range f.Stacks {
		dead := false
		for _, h := range out {
			if h.Destroyed && h.Design == st.Design {
				dead = true
			}
		}
		if !dead {
			kept = append(kept, st)
		}
	}
	f.Stacks = kept
	return out
}

// MinesLost is what the paying field loses for one stop (OBJECTS.md "Hits
// on moving fleets", CONFIRMED OB-010-S, OB-024): max(10, ⌊N/20⌋), or
// max(50, ⌊N/100⌋) when ⌊N/20⌋ > 50.
func MinesLost(n int) int {
	if n/20 > 50 {
		return max(50, n/100)
	}
	return max(10, n/20)
}

// PayingField is the field of kind k that pays for a stop of a victim's
// fleet at pos: among the fields of that kind whose owner is not the
// victim and does not treat it as a friend, the smallest d² − N, the
// first in object order on a tie (OBJECTS.md "Which field pays",
// CONFIRMED MF-4). It returns -1 when there is none.
func PayingField(g *engine.Game, s *Space, victim int, k MineKind, pos engine.Point) int {
	best, bestV := -1, 0
	for i, m := range s.Minefields {
		if m.Kind != k || m.Count <= 0 || !stopsFleet(g, m, victim) {
			continue
		}
		if v := d2(m.Pos, pos) - m.Count; best < 0 || v < bestV {
			best, bestV = i, v
		}
	}
	return best
}

// Hit is everything one stop did. The fleet is already at the stop point.
type Hit struct {
	Fleet int // index into Game.Fleets
	Kind  MineKind
	// Field is the paying field after its loss (the victim learns it);
	// Paid is how many mines it lost. Gone is set when it disappeared.
	Field Minefield
	Paid  int
	Gone  bool
	// Designs is the damage per design group.
	Designs []DesignHit
	// Salvage is the minerals dropped at the stop point (mineCargo; none
	// at a planet's exact position), for Space.AddMineSalvage.
	Salvage engine.Minerals
	// Disclosed lists the victim's designs the paying field's owner
	// learns in full when that owner is Space Demolition: the damaged
	// designs, or every design present if none was damaged.
	Disclosed []int
}

// ApplyHit applies a stop of fleet fi by kind k at its current position
// (OBJECTS.md "Hits on moving fleets", "On a hit", CONFIRMED): damage,
// the paying field's loss, salvage from destroyed ships and Space
// Demolition disclosure. The fleet gets no ram-scoop fuel this year and
// has spent the fuel for its whole leg; both are the caller's.
//
// Cargo and salvage follow mineCargo.
func ApplyHit(g *engine.Game, s *Space, fi int, k MineKind, rng engine.Rand) Hit {
	f := &g.Fleets[fi]
	h := Hit{Fleet: fi, Kind: k}
	present := designsIn(f)
	capBefore, fuelBefore := fleetCargoCap(g, f), fleetFuelCap(g, f)
	h.Designs = MineDamage(g, f, k, nil)
	if p := PayingField(g, s, f.Owner, k, f.Pos); p >= 0 {
		m := &s.Minefields[p]
		h.Paid = min(m.Count, MinesLost(m.Count))
		m.Count -= h.Paid
		h.Field = *m
		if m.Count <= 0 {
			h.Gone = true
			s.Minefields = append(s.Minefields[:p], s.Minefields[p+1:]...)
		}
		if prt(g, h.Field.Owner) == engine.PRTSpaceDemolition {
			h.Disclosed = disclosed(h.Designs, present)
		}
	}
	h.Salvage = mineCargo(g, f, h.Designs, capBefore, fuelBefore, true, rng)
	return h
}

func designsIn(f *engine.Fleet) []int {
	var ds []int
	seen := map[int]bool{}
	for _, st := range f.Stacks {
		if !seen[st.Design] {
			seen[st.Design] = true
			ds = append(ds, st.Design)
		}
	}
	return ds
}

func disclosed(hits []DesignHit, present []int) []int {
	var ds []int
	for _, h := range hits {
		if h.Damage > 0 {
			ds = append(ds, h.Design)
		}
	}
	if len(ds) == 0 {
		return present
	}
	return ds
}

func atPlanet(g *engine.Game, p engine.Point) bool {
	for _, pl := range g.Planets {
		if pl.Pos == p {
			return true
		}
	}
	return false
}

// mineCargo applies the cargo rules when hits destroyed ships of fleet f
// (OBJECTS.md "Cargo when ships are destroyed", MEASURED MF-14), given
// the fleet's cargo and fuel capacity before the hit. The destroyed ships'
// share is lost: ⌊C·lost/total⌋ of the cargo C by cargo capacity, split
// per item with the remainder 1 kT at a time over ironium, boranium,
// germanium and colonists (loseShare, as COMBAT.md "Salvage"), and
// ⌊F·lost/total⌋ of the fuel by fuel capacity. With salvage (a mine hit,
// not a detonation) the minerals then left aboard are dropped as salvage
// (Legacy.MineSurvivorSalvage; all of them when the whole fleet died),
// except at a planet's exact position, where they stay aboard. A drop of
// nothing becomes rand(10) kT of each mineral (Legacy.EmptyFleetSalvage,
// MEASURED OB-024). It returns the salvage.
//
// ASSUMPTION O10: the rand(10) draws are made only where salvage can
// form (not at a planet), and only when ships were destroyed.
func mineCargo(g *engine.Game, f *engine.Fleet, hits []DesignHit, capBefore, fuelBefore int, salvage bool, rng engine.Rand) engine.Minerals {
	lostCap, lostFuel, lost := 0, 0, false
	for _, h := range hits {
		if h.Destroyed {
			d := g.Designs[h.Design]
			lostCap += h.Ships * d.CargoCapacity
			lostFuel += h.Ships * d.FuelCapacity
			lost = true
		}
	}
	var out engine.Minerals
	if !lost {
		return out
	}
	whole := len(f.Stacks) == 0
	if !whole {
		if capBefore > 0 {
			loseShare(&f.Cargo, lostCap, capBefore)
		}
		if fuelBefore > 0 {
			f.Fuel -= f.Fuel * lostFuel / fuelBefore
		}
	}
	if !salvage || atPlanet(g, f.Pos) {
		return out
	}
	// Legacy.MineSurvivorSalvage reproduces the original's LEGACY BUG
	// candidate that a mine hit which destroys some ships also drops every
	// mineral the survivors still carry as salvage (OBJECTS.md "Cargo when
	// ships are destroyed", MEASURED MF-14); off, the survivors keep their
	// minerals.
	if whole || g.Rules.Legacy.MineSurvivorSalvage {
		out = f.Cargo.Minerals
		f.Cargo.Minerals = engine.Minerals{}
	}
	// Legacy.EmptyFleetSalvage reproduces the original's LEGACY BUG
	// candidate that a fleet with no minerals that loses ships to a mine
	// hit drops rand(10) kT of each mineral as salvage (OBJECTS.md "Hits
	// on moving fleets", MEASURED OB-024).
	if out == (engine.Minerals{}) && f.Cargo.Minerals == (engine.Minerals{}) && g.Rules.Legacy.EmptyFleetSalvage {
		for m := range engine.NumMinerals {
			out[m] = rng.Intn(10)
		}
	}
	return out
}

// --- Detonation ---

// Detonation is one fleet's damage from a detonating field.
type Detonation struct {
	Fleet   int
	Field   Minefield
	Designs []DesignHit
	// Disclosed is as in Hit, for a Space Demolition field owner
	// (CONFIRMED MF-7).
	Disclosed []int
}

// Detonate runs this year's detonations, before decay (OBJECTS.md
// "Detonation", CONFIRMED OB-002-M, MF-7, MF-8): every fleet inside a
// detonating field, of any owner including the field's, takes hit damage
// of the field's kind, except the owner's own Mini Mine Layer and Super
// Mine Layer hulls. No stop, no salvage, at most one detonation per fleet
// per year. A destroyed ship's share of cargo and fuel is lost as for a
// hit, and the survivors keep the rest (BINARY-ONLY). A speed-bump field
// damages nobody. The extra 25% decay is in Decay.
//
// Fields go off in object order, and the first detonating field that
// contains a fleet marks it, even when it did the fleet no damage (the
// owner's own layer hulls, a speed bump); later fields skip it that year
// (OBJECTS.md "Several detonating fields", BINARY-ONLY). Each field's
// decay touches only itself, so detonating every field before decaying
// any matches "each just before its own decay".
func (s *Space) Detonate(g *engine.Game) []Detonation {
	var out []Detonation
	done := map[int]bool{}
	for _, m := range s.Minefields {
		if !m.Detonate || m.Count <= 0 {
			continue
		}
		for _, fi := range fleetOrder(g) {
			f := &g.Fleets[fi]
			if done[fi] || !m.Contains(f.Pos) {
				continue
			}
			done[fi] = true
			present := designsIn(f)
			exempt := func(d int) bool {
				if f.Owner != m.Owner {
					return false
				}
				hc, ok := engine.Components().Lookup(g.Designs[d].Hull.Name)
				_, layer := hc.Stats["mine_layer_multiplier"]
				return ok && layer
			}
			capBefore, fuelBefore := fleetCargoCap(g, f), fleetFuelCap(g, f)
			hits := MineDamage(g, f, m.Kind, exempt)
			mineCargo(g, f, hits, capBefore, fuelBefore, false, nil)
			det := Detonation{Fleet: fi, Field: m, Designs: hits}
			if prt(g, m.Owner) == engine.PRTSpaceDemolition && len(hits) > 0 {
				det.Disclosed = disclosed(hits, present)
			}
			out = append(out, det)
		}
	}
	return out
}
