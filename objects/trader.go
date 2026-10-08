package objects

import (
	"math"

	"github.com/bfaber-centaur/elegy/engine"
)

// TraderItemKind is what a Mystery Trader offers.
type TraderItemKind int

const (
	ItemResearch TraderItemKind = iota
	ItemShip
	ItemPart // Bit names the part
)

// Trader part bits (OBJECTS.md "Encounters", CONFIRMED WT-003 A). Bit 12
// is a ship gift.
const (
	BitMultiCargoPod = iota
	BitMultiFunctionPod
	BitLangstonShell
	BitMegaPolyShell
	BitAlienMiner
	BitHushABoom
	BitAntiMatterTorpedo
	BitMultiContainedMunition
	BitMiniMorph
	BitEnigmaPulsar
	BitGenesisDevice
	BitJumpGate
	BitShip
	NumTraderBits
)

// TraderPartNames are the parts by bit, as the component table names
// them.
var TraderPartNames = [BitShip]string{
	"Multi Cargo Pod", "Multi Function Pod", "Langston Shell", "Mega Poly Shell",
	"Alien Miner", "Hush-a-Boom", "Anti Matter Torpedo", "Multi Contained Munition",
	"Mini Morph", "Enigma Pulsar", "Genesis Device", "Jump Gate",
}

// TraderItem is a Trader's offer.
type TraderItem struct {
	Kind TraderItemKind
	Bit  int
}

// Trader is one Mystery Trader.
type Trader struct {
	Pos, Dest engine.Point
	Warp      int
	Item      TraderItem
	// Served marks the players this Trader has rewarded or traded with.
	Served []bool
}

// TraderParts is each player's Trader part word: bit b set means the
// player owns part b (OBJECTS.md "Encounters").
type TraderParts []uint16

// Owns reports whether a player owns part bit b.
func (tp TraderParts) Owns(player, b int) bool {
	return player >= 0 && player < len(tp) && tp[player]&(1<<b) != 0
}

func (tp *TraderParts) give(player, b int) {
	for len(*tp) <= player {
		*tp = append(*tp, 0)
	}
	(*tp)[player] |= 1 << b
}

// Items is the player's Trader items by component name, as
// engine.Catalog.ReadDesign takes them.
func (tp TraderParts) Items(player int) map[string]bool {
	m := map[string]bool{}
	for b, n := range TraderPartNames {
		if tp.Owns(player, b) {
			m[n] = true
		}
	}
	return m
}

// --- Appearance ---

// edgeLow and edgeHigh are the Trader's edges for universe size 0..4
// (KERNEL.md "Mystery Trader appearance" step 3, 4).
func edgeLow() int          { return 1020 }
func edgeHigh(size int) int { return 1380 + 400*size }

// freeCoord is a free coordinate along an edge: 1020 + rand(361 + 400·size).
func freeCoord(size int, rng engine.Rand) int { return 1020 + rng.Intn(361+400*size) }

// itemChance is r in r/10 for research or a ship (OBJECTS.md
// "Appearance"): 5 before year index 100, 3 before 250, 2 after; +1 below
// warp 10, −1 above.
func itemChance(yearIndex, warp int) int {
	r := 2
	switch {
	case yearIndex < 100:
		r = 5
	case yearIndex < 250:
		r = 3
	}
	switch {
	case warp < 10:
		r++
	case warp > 10:
		r--
	}
	return r
}

// Appear runs the yearly Mystery Trader appearance right after new
// minerals, at the end of production (KERNEL.md "Mystery Trader
// appearance", CONFIRMED KX-004 S6–S10), in its draw order: the chance,
// warp 8 + rand(5), the start's and destination's free coordinates, the
// edge direction, the axis, then the item. Only with random events on and
// from year index 40. Every player gets the appearance message. A second
// Trader can appear while one exists.
//
// PLACEHOLDER T1: OBJECTS.md says the part draw is rerolled once for four
// of the parts, three of which turn into research with 1/2 before year
// index 120, 150 or 180, but does not say which parts or which limit
// goes with which. Elegy makes no reroll and no conversion until the spec
// names them (BINARY-ONLY there).
func (s *Space) Appear(yearIndex, size int, randomEvents bool, rng engine.Rand) (Trader, bool) {
	if !randomEvents || yearIndex < 40 {
		return Trader{}, false
	}
	var draw int
	switch {
	case yearIndex%100 == 71:
		draw = rng.Intn(2)
	case yearIndex%100 == 33:
		draw = rng.Intn(3)
	case yearIndex%128 == 49:
		draw = rng.Intn(4)
	case yearIndex%2 == 1:
		return Trader{}, false
	default:
		draw = rng.Intn(7)
	}
	if draw != 0 {
		return Trader{}, false
	}
	t := Trader{Warp: 8 + rng.Intn(5)}
	a, b := freeCoord(size, rng), freeCoord(size, rng)
	from, to := edgeLow(), edgeHigh(size)
	if rng.Intn(2) == 1 {
		from, to = to, from
	}
	if rng.Intn(2) == 0 {
		t.Pos, t.Dest = engine.Point{X: a, Y: from}, engine.Point{X: b, Y: to}
	} else {
		t.Pos, t.Dest = engine.Point{X: from, Y: a}, engine.Point{X: to, Y: b}
	}
	if rng.Intn(10) < itemChance(yearIndex, t.Warp) {
		if rng.Intn(6) == 0 {
			t.Item = TraderItem{Kind: ItemShip}
		} else {
			t.Item = TraderItem{Kind: ItemResearch}
		}
	} else {
		bit := rng.Intn(NumTraderBits)
		if bit == BitShip {
			t.Item = TraderItem{Kind: ItemShip}
		} else {
			t.Item = TraderItem{Kind: ItemPart, Bit: bit}
		}
	}
	s.Traders = append(s.Traders, t)
	return t, true
}

// --- Movement ---

// StepToward moves pos toward dest by move ly (OBJECTS.md "Flight and
// decay", the packet rule the Trader shares): it arrives when the
// truncated distance is at most the move; otherwise each coordinate moves
// by its rounded share.
//
// ASSUMPTION T2: the rounded share is dx·move/dist rounded half away from
// zero, with dist the exact distance.
func StepToward(pos, dest engine.Point, move int) (engine.Point, bool) {
	dx, dy := float64(dest.X-pos.X), float64(dest.Y-pos.Y)
	dist := math.Hypot(dx, dy)
	if int(dist) <= move {
		return dest, true
	}
	f := float64(move) / dist
	return engine.Point{X: pos.X + int(math.Round(dx*f)), Y: pos.Y + int(math.Round(dy*f))}, false
}

// TraderMove is what one Trader did in the yearly movement.
type TraderMove struct {
	Trader             int
	WarpRose, NewDest  bool
	Arrived, Left      bool
	From, To           engine.Point
	WarpBefore, Warp   int
	DestBefore, DestTo engine.Point
}

// MoveTraders moves every Trader before fleets (OBJECTS.md "Movement",
// BINARY-ONLY except where noted; CONFIRMED OB-023, OB-026, OB-031):
// below warp 13, with 1/25 its warp rises by 1 and then with 1/3 it picks
// a new destination; it moves warp² ly like a packet. On arrival it leaves
// the galaxy if another Trader exists or with 1/2; otherwise it stays at
// the edge, takes warp max(6, warp − 2) + 1 and a new destination, and
// does not move further that year. Left Traders are removed.
//
// ASSUMPTION T3: the new-destination draw is made only in a year the warp
// rose (WT: the one new destination came with a rise). ASSUMPTION T4: a
// new destination lies on the edge opposite the Trader's current edge
// along the same axis, its free coordinate drawn as at appearance; a
// Trader not on an edge (after a warp rise in flight) keeps its
// destination's edge and draws a new free coordinate on it.
func (s *Space) MoveTraders(size int, rng engine.Rand) []TraderMove {
	var out []TraderMove
	var kept []Trader
	for i := range s.Traders {
		t := s.Traders[i]
		mv := TraderMove{Trader: i, From: t.Pos, WarpBefore: t.Warp, DestBefore: t.Dest}
		if t.Warp < 13 && rng.Intn(25) == 0 {
			t.Warp++
			mv.WarpRose = true
			if rng.Intn(3) == 0 {
				t.Dest = newDestination(t, size, rng)
				mv.NewDest = true
			}
		}
		pos, arrived := StepToward(t.Pos, t.Dest, t.Warp*t.Warp)
		t.Pos = pos
		if arrived {
			mv.Arrived = true
			if len(s.Traders) > 1 || rng.Intn(2) == 0 {
				mv.Left = true
			} else {
				t.Warp = max(6, t.Warp-2) + 1
				t.Dest = newDestination(t, size, rng)
				mv.NewDest = true
			}
		}
		mv.To, mv.Warp, mv.DestTo = t.Pos, t.Warp, t.Dest
		out = append(out, mv)
		if !mv.Left {
			kept = append(kept, t)
		}
	}
	s.Traders = kept
	return out
}

func newDestination(t Trader, size int, rng engine.Rand) engine.Point {
	lo, hi := edgeLow(), edgeHigh(size)
	c := freeCoord(size, rng)
	opposite := func(v int) int {
		if v == lo {
			return hi
		}
		return lo
	}
	switch {
	case t.Pos.Y == lo || t.Pos.Y == hi:
		return engine.Point{X: c, Y: opposite(t.Pos.Y)}
	case t.Pos.X == lo || t.Pos.X == hi:
		return engine.Point{X: opposite(t.Pos.X), Y: c}
	case t.Dest.Y == lo || t.Dest.Y == hi:
		return engine.Point{X: c, Y: t.Dest.Y}
	}
	return engine.Point{X: t.Dest.X, Y: c}
}

// --- Encounters ---

// TraderContext is what the encounters need to know about the game that
// the engine does not hold yet.
type TraderContext struct {
	YearIndex int
	// Computer reports a computer player; Level its level, 0 easy .. 3
	// expert.
	Computer func(player int) bool
	Level    func(player int) int
	// Humans is the number of human players.
	Humans int
}

// minTrade is the cargo a fleet needs to trade (OBJECTS.md "Encounters",
// CONFIRMED OB-004: 4,999 kT kept the fleet, 5,000 traded).
const minTrade = 5000

// Encounter is one fleet's meeting with a Trader.
type Encounter struct {
	Trader, Fleet int // Fleet is the fleet's ID
	Owner         int
	Refused       bool // too little cargo
	Recovering    bool // the owner was already served by this Trader
	// Reward, for a traded fleet.
	Reward Reward
}

// Reward is what a player got for a trade.
type Reward struct {
	Kind TraderItemKind
	// Part is the bit gained (ItemPart).
	Part int
	// Levels is the research message's L and Fields the fields raised
	// (ItemResearch); Nothing is the "unable to teach you anything new"
	// branch.
	Levels  int
	Fields  []int
	Nothing bool
	// Ship gift: the design index, the new fleet's ID and the ship
	// count; NoRoom when no design slot or fleet number was free.
	Design, NewFleet, Ships int
	NoRoom                  bool
}

func lowestField(levels [engine.NumFields]int) int {
	lo := 0
	for f := range engine.NumFields {
		if levels[f] < levels[lo] {
			lo = f
		}
	}
	return lo
}

// researchReward raises the player's fields (OBJECTS.md "Encounters",
// research, CONFIRMED WT-002 A, B; field odds MEASURED): L = min(10, 6 +
// ⌊(cargo − 5000)/1200⌋), adjusted by the tech sum T; each level, with 3/4
// a uniformly random field (the lowest if that one is at 26), with 1/4
// the lowest (first on ties); stops once the lowest is at 26. Each step
// raises a level and leaves accumulated research unchanged.
//
// ASSUMPTION T5: each level draws rand(4), then rand(6) only on the 3/4
// branch.
func researchReward(p *engine.Player, cargo int, rng engine.Rand) Reward {
	l := min(10, 6+(cargo-minTrade)/1200)
	t := 0
	for _, v := range p.Research.Levels {
		t += v
	}
	switch {
	case t >= 108:
		l = 1
	case t >= 96:
		l = 2
	case t >= 84:
		l -= 3
	case t >= 72:
		l -= 2
	case t >= 60:
		l--
	}
	r := Reward{Kind: ItemResearch, Levels: l}
	for range l {
		lv := &p.Research.Levels
		lo := lowestField(*lv)
		if lv[lo] >= engine.MaxTechLevel {
			break
		}
		f := lo
		if rng.Intn(4) < 3 {
			if g := rng.Intn(engine.NumFields); lv[g] < engine.MaxTechLevel {
				f = g
			}
		}
		lv[f]++
		r.Fields = append(r.Fields, f)
	}
	return r
}

// allMaxed reports every field at 26.
func allMaxed(p *engine.Player) bool {
	for _, v := range p.Research.Levels {
		if v < engine.MaxTechLevel {
			return false
		}
	}
	return true
}

// LegacyTraderLastRedraw reproduces the original's LEGACY BUG that a
// player whose 25th part redraw found an unowned part gets a ship instead
// (OBJECTS.md "Encounters", BINARY-ONLY). On by default.
var LegacyTraderLastRedraw = true

// Encounters runs the meetings after battles (OBJECTS.md "Encounters",
// CONFIRMED OB-004, OB-023, OB-026, OB-030-T, WT-002..WT-004): each
// Trader in order meets every fleet at exactly its position, in fleet
// order. Cargo (minerals only) below 5,000 kT: refused. Owner already
// served by this Trader: refused, fleet kept. Otherwise the owner is
// marked served, the fleet is consumed (ships and cargo) and the reward
// follows. A consumed fleet is not offered to another Trader.
func (s *Space) Encounters(g *engine.Game, ctx TraderContext, rng engine.Rand) []Encounter {
	var out []Encounter
	consumed := map[int]bool{}
	for ti := range s.Traders {
		t := &s.Traders[ti]
		for _, fi := range fleetOrder(g) {
			f := &g.Fleets[fi]
			if consumed[f.ID] || f.Pos != t.Pos {
				continue
			}
			e := Encounter{Trader: ti, Fleet: f.ID, Owner: f.Owner}
			cargo := f.Cargo.Minerals[engine.Ironium] + f.Cargo.Minerals[engine.Boranium] + f.Cargo.Minerals[engine.Germanium]
			switch {
			case cargo < minTrade:
				e.Refused = true
			case has(t.Served, f.Owner):
				e.Recovering = true
			default:
				t.Served = mark(t.Served, f.Owner)
				consumed[f.ID] = true
				e.Reward = s.reward(g, ctx, t, f.Owner, f.Pos, cargo, rng)
			}
			out = append(out, e)
		}
	}
	if len(consumed) > 0 {
		kept := g.Fleets[:0]
		for _, f := range g.Fleets {
			if !consumed[f.ID] {
				kept = append(kept, f)
			}
		}
		g.Fleets = kept
	}
	return out
}

func (s *Space) reward(g *engine.Game, ctx TraderContext, t *Trader, owner int, pos engine.Point, cargo int, rng engine.Rand) Reward {
	p := &g.Players[owner]
	item := t.Item
	if item.Kind == ItemPart && !s.TraderParts.Owns(owner, item.Bit) {
		s.TraderParts.give(owner, item.Bit)
		return Reward{Kind: ItemPart, Part: item.Bit}
	}
	if item.Kind == ItemShip {
		return s.shipGift(g, ctx, owner, pos, rng)
	}
	// Research, or an offered part already owned.
	if !allMaxed(p) {
		return researchReward(p, cargo, rng)
	}
	if rng.Intn(5) == 0 {
		return Reward{Kind: ItemResearch, Nothing: true}
	}
	bit := rng.Intn(NumTraderBits)
	redraws := 0
	for bit != BitShip && s.TraderParts.Owns(owner, bit) && redraws < 25 {
		bit = rng.Intn(NumTraderBits)
		redraws++
	}
	if bit == BitShip || s.TraderParts.Owns(owner, bit) || (redraws == 25 && LegacyTraderLastRedraw) {
		return s.shipGift(g, ctx, owner, pos, rng)
	}
	s.TraderParts.give(owner, bit)
	return Reward{Kind: ItemPart, Part: bit}
}

// --- Ship gifts ---

// giftDesign is one of the Trader's ship designs (OBJECTS.md
// "Encounters", ship, CONFIRMED WT-003 B, WT-004 B, C, OB-026).
//
// ASSUMPTION T6: the parts go into the hull's slots in the order the
// spec lists them, "in two slots" filling consecutive slots.
func giftDesign(kind int) (string, string, []engine.SlotFill) {
	f := func(slot int, part string, n int) engine.SlotFill {
		return engine.SlotFill{Slot: slot, Part: part, Count: n}
	}
	switch kind {
	case 0:
		return "M.T. Lifeboat", "Nubian", []engine.SlotFill{
			f(0, "Enigma Pulsar", 3),
			f(1, "Mega Poly Shell", 3), f(2, "Mega Poly Shell", 3),
			f(3, "Anti Matter Torpedo", 3), f(4, "Anti Matter Torpedo", 3),
			f(5, "Langston Shell", 3), f(6, "Langston Shell", 3),
			f(7, "Multi Function Pod", 3), f(8, "Multi Function Pod", 3),
			f(9, "Multi Cargo Pod", 3),
			f(10, "Multi Contained Munition", 3), f(11, "Multi Contained Munition", 3), f(12, "Multi Contained Munition", 3),
		}
	}
	name, shell := "M.T. Scout", "Langston Shell"
	if kind == 2 {
		name, shell = "M.T. Probe", "Mega Poly Shell"
	}
	return name, "Mini Morph", []engine.SlotFill{
		f(0, "Enigma Pulsar", 2), f(1, shell, 3), f(2, "Multi Function Pod", 1),
		f(3, "Multi Cargo Pod", 1), f(4, "Jump Gate", 1),
		f(5, "Anti Matter Torpedo", 2), f(6, "Anti Matter Torpedo", 2),
	}
}

// maxShipDesigns and maxFleets are the per-player limits (LIMITS.md,
// PRODUCTION-LAUNCH.md "The 512-fleet limit").
const (
	maxShipDesigns = 16
	maxFleets      = 512
)

// shipGift gives a player the Trader's ship (OBJECTS.md "Encounters",
// ship; counts after year index 100 and for computer players
// BINARY-ONLY): computer players get nothing; design Lifeboat with 1/4
// (1/3 after year index 100), else Scout or Probe evenly; count 2 with
// 1/3, else 1, plus rand(⌊year index/100⌋ + 1) after year index 100
// unless the game has a single human player, capped at 5, then for the
// Scout and Probe + rand(count + 1). A matching design the player has is
// reused, else it goes into the first empty ship design slot; the new
// fleet has full fuel at the trade point. No free slot or 512 fleets: no
// ship.
//
// ASSUMPTION T7: a matching design has the same name, hull and parts;
// the draws are design kind, then the Scout/Probe coin, then the count
// draws; the new fleet takes the owner's lowest unused fleet number and
// battle plan 0.
func (s *Space) shipGift(g *engine.Game, ctx TraderContext, owner int, pos engine.Point, rng engine.Rand) Reward {
	r := Reward{Kind: ItemShip, Design: -1, NewFleet: -1}
	if ctx.Computer != nil && ctx.Computer(owner) {
		return r
	}
	lifeboat := 4
	if ctx.YearIndex > 100 {
		lifeboat = 3
	}
	kind := 0
	if rng.Intn(lifeboat) != 0 {
		kind = 1 + rng.Intn(2)
	}
	n := 1
	if rng.Intn(3) == 0 {
		n = 2
	}
	if ctx.YearIndex > 100 && ctx.Humans != 1 {
		n += rng.Intn(ctx.YearIndex/100 + 1)
	}
	n = min(n, 5)
	if kind != 0 {
		n += rng.Intn(n + 1)
	}
	name, hull, fills := giftDesign(kind)
	d, err := engine.Components().NewDesign(name, hull, fills)
	if err != nil {
		panic(err) // the gift designs are fixed and fit their hulls
	}
	di := s.giftSlot(g, owner, d)
	fleets := 0
	used := map[int]bool{}
	for _, f := range g.Fleets {
		if f.Owner == owner {
			fleets++
			used[f.Number] = true
		}
	}
	if di < 0 || fleets >= maxFleets {
		r.NoRoom = true
		return r
	}
	num := 0
	for used[num] {
		num++
	}
	id := 0
	for _, f := range g.Fleets {
		id = max(id, f.ID)
	}
	g.Fleets = append(g.Fleets, engine.Fleet{
		ID: id + 1, Number: num, Owner: owner, Pos: pos,
		Stacks: []engine.Stack{{Design: di, Count: n}},
		Fuel:   n * g.Designs[di].FuelCapacity,
	})
	r.Design, r.NewFleet, r.Ships = di, id+1, n
	return r
}

// giftSlot returns the design index for a gift design: an identical
// design in one of the owner's ship slots, else a new design in the first
// empty ship slot, else -1.
func (s *Space) giftSlot(g *engine.Game, owner int, d engine.Design) int {
	taken := map[int]bool{}
	for _, ds := range g.DesignSlots {
		if ds.Owner != owner || ds.Starbase {
			continue
		}
		taken[ds.Slot] = true
		if ds.Design >= 0 && ds.Design < len(g.Designs) && sameDesign(g.Designs[ds.Design], d) {
			return ds.Design
		}
	}
	for slot := range maxShipDesigns {
		if !taken[slot] {
			g.Designs = append(g.Designs, d)
			di := len(g.Designs) - 1
			g.DesignSlots = append(g.DesignSlots, engine.DesignSlot{Owner: owner, Slot: slot, Design: di})
			return di
		}
	}
	return -1
}

func sameDesign(a, b engine.Design) bool {
	if a.Name != b.Name || a.Hull.Name != b.Hull.Name || len(a.Slots) != len(b.Slots) {
		return false
	}
	for i := range a.Slots {
		if a.Slots[i].Part.Name != b.Slots[i].Part.Name || a.Slots[i].Count != b.Slots[i].Count {
			return false
		}
	}
	return true
}

// --- Computer players' planets ---

// PlanetTrade is one computer player's planet trade.
type PlanetTrade struct {
	Trader, Planet int
	Owner          int
	Part           int // bit gained, or -1
	Fields         []int
	Price          engine.Minerals
}

// PlanetTrades runs the Trader's trades with computer players' planets
// after the fleets (OBJECTS.md "Computer players' planets", CONFIRMED in
// part TP-001, TP-002, O-53): only Harder and Expert computer players
// (Standard and Easy excluded, BINARY-ONLY), a planet with a starbase
// within 100 ly (BINARY-ONLY) whose owner this Trader has not served,
// with at least 5,000 kT of surface minerals, 3,500 for Harder (CONFIRMED
// O-53; the exact edge NOT RUN).
//   - Part item: an owner lacking the part gains it; one owning it draws a
//     random part it lacks, with up to 50 redraws (bit 12 counts and gives
//     only the bit). Price: all the surface minerals.
//   - Research item, or no part found: an owner whose tech sums to 150 or
//     more gets nothing and stays available; otherwise the lowest field
//     gains a level six times. Price: 5,000 kT (3,500 for Harder, CONFIRMED O-53), from
//     germanium, then boranium, then ironium.
//
// The owner is marked served; no message is sent. Human players' planets
// never trade.
//
// ASSUMPTION T8: planets are visited in planet order; a ship item is
// treated as research; "within 100 ly" is d² ≤ 10,000.
func (s *Space) PlanetTrades(g *engine.Game, ctx TraderContext, rng engine.Rand) []PlanetTrade {
	var out []PlanetTrade
	if ctx.Computer == nil || ctx.Level == nil {
		return out
	}
	for ti := range s.Traders {
		t := &s.Traders[ti]
		for pi := range g.Planets {
			pl := &g.Planets[pi]
			o := pl.Owner
			if o < 0 || o >= len(g.Players) || !ctx.Computer(o) || has(t.Served, o) || !pl.HasStarbase || d2(pl.Pos, t.Pos) > 10000 {
				continue
			}
			lv := ctx.Level(o)
			if lv < 2 {
				continue
			}
			need := 5000
			if lv == 2 {
				need = 3500
			}
			surface := pl.Surface[engine.Ironium] + pl.Surface[engine.Boranium] + pl.Surface[engine.Germanium]
			if surface < need {
				continue
			}
			tr := PlanetTrade{Trader: ti, Planet: pi, Owner: o, Part: -1}
			if t.Item.Kind == ItemPart {
				bit := t.Item.Bit
				for i := 0; i < 50 && s.TraderParts.Owns(o, bit); i++ {
					bit = rng.Intn(NumTraderBits)
				}
				if !s.TraderParts.Owns(o, bit) {
					s.TraderParts.give(o, bit)
					tr.Part = bit
					tr.Price = pl.Surface
					pl.Surface = engine.Minerals{}
					t.Served = mark(t.Served, o)
					out = append(out, tr)
					continue
				}
			}
			p := &g.Players[o]
			sum := 0
			for _, v := range p.Research.Levels {
				sum += v
			}
			if sum >= 150 {
				continue
			}
			for range 6 {
				f := lowestField(p.Research.Levels)
				p.Research.Levels[f]++
				tr.Fields = append(tr.Fields, f)
			}
			left := need
			for _, m := range []int{engine.Germanium, engine.Boranium, engine.Ironium} {
				take := min(left, pl.Surface[m])
				pl.Surface[m] -= take
				tr.Price[m] = take
				left -= take
			}
			t.Served = mark(t.Served, o)
			out = append(out, tr)
		}
	}
	return out
}
