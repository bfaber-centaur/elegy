package objects

import (
	"math"
	"sort"

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
// A first part draw of bit 6, 7, 10 or 11 is replaced by a second
// rand(13), which stands; only that second draw can convert: bit 7 before
// year index 120, bit 10 before 150 or bit 11 before 180 draws rand(2),
// and 1 makes the item research (OBJECTS.md "Appearance", BINARY-ONLY).
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
		switch bit {
		case BitAntiMatterTorpedo, BitMultiContainedMunition, BitGenesisDevice, BitJumpGate:
			bit = rng.Intn(NumTraderBits)
			if limit, ok := conversionLimit[bit]; ok && yearIndex < limit && rng.Intn(2) == 1 {
				t.Item = TraderItem{Kind: ItemResearch}
				s.Traders = append(s.Traders, t)
				return t, true
			}
		}
		if bit == BitShip {
			t.Item = TraderItem{Kind: ItemShip}
		} else {
			t.Item = TraderItem{Kind: ItemPart, Bit: bit}
		}
	}
	s.Traders = append(s.Traders, t)
	return t, true
}

// conversionLimit is the year index before which a rerolled part turns
// into research with 1/2 (OBJECTS.md "Appearance", BINARY-ONLY).
var conversionLimit = map[int]int{BitMultiContainedMunition: 120, BitGenesisDevice: 150, BitJumpGate: 180}

// --- Movement ---

// StepToward moves pos toward dest by move ly (OBJECTS.md "Movement",
// each year's step, shared with packets; BINARY-ONLY in detail): with d
// the exact distance, it arrives when trunc(d) ≤ move. Otherwise, when
// d > 0.0001, each axis moves by trunc(Δ·move/d ± 0.5), the sign of Δ,
// with move/d computed once; a step that lands on dest is an arrival.
func StepToward(pos, dest engine.Point, move int) (engine.Point, bool) {
	dx, dy := float64(dest.X-pos.X), float64(dest.Y-pos.Y)
	d := math.Sqrt(dx*dx + dy*dy)
	if int(d) <= move {
		return dest, true
	}
	if d <= 0.0001 {
		return pos, false
	}
	f := float64(move) / d
	share := func(delta float64) int {
		h := -0.5
		if delta > 0 {
			h = 0.5
		}
		return int(delta*f + h)
	}
	p := engine.Point{X: pos.X + share(dx), Y: pos.Y + share(dy)}
	return p, p == dest
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
// BINARY-ONLY except where noted; CONFIRMED OB-023, OB-026, OB-031): at
// warp 12 or below it draws rand(25), and 0 raises the warp by 1 and only
// then draws rand(3), where 0 picks a new destination; it moves warp² ly
// like a packet at the new warp. On arrival it is removed with no draw if
// another Trader is present, else rand(2): 0 removes it, 1 keeps it at the
// edge with warp max(6, warp − 2) + 1 and a new destination, and it does
// not move further that year.
//
// "Another Trader" is one still in the galaxy at that moment, moved or
// not; one removed earlier this year does not count (BINARY-ONLY).
func (s *Space) MoveTraders(size int, rng engine.Rand) []TraderMove {
	var out []TraderMove
	var kept []Trader
	for i := range s.Traders {
		t := s.Traders[i]
		mv := TraderMove{Trader: i, From: t.Pos, WarpBefore: t.Warp, DestBefore: t.Dest}
		if t.Warp <= 12 && rng.Intn(25) == 0 {
			t.Warp++
			mv.WarpRose = true
			if rng.Intn(3) == 0 {
				t.Dest = newDestination(size, rng)
				mv.NewDest = true
			}
		}
		pos, arrived := StepToward(t.Pos, t.Dest, t.Warp*t.Warp)
		t.Pos = pos
		if arrived {
			mv.Arrived = true
			others := len(kept) + len(s.Traders) - i - 1
			if others > 0 || rng.Intn(2) == 0 {
				mv.Left = true
			} else {
				t.Warp = max(6, t.Warp-2) + 1
				t.Dest = newDestination(size, rng)
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

// newDestination draws a Trader's new destination (OBJECTS.md "Movement",
// new destination; CONFIRMED in part OB-023, draw order BINARY-ONLY):
// rand(2) picks the side (0 the high edge, 1 the low), then the free
// coordinate, then rand(2) the axis (0: x is the edge value). Any of the
// four edges, wherever the Trader is or was heading.
func newDestination(size int, rng engine.Rand) engine.Point {
	edge := edgeHigh(size)
	if rng.Intn(2) == 1 {
		edge = edgeLow()
	}
	c := freeCoord(size, rng)
	if rng.Intn(2) == 0 {
		return engine.Point{X: edge, Y: c}
	}
	return engine.Point{X: c, Y: edge}
}

// --- Encounters ---

// TraderContext is what the encounters need to know about the players
// (MeetTraders fills it from engine.Player).
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
// Draws per level: rand(4); 0–2 then rand(6) for the field, 3 none
// (BINARY-ONLY).
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

// Meet runs each Trader's meetings after battles (OBJECTS.md
// "Encounters" and "Computer players' planets"): for each Trader in
// order, its fleets, then computer players' planets (order BINARY-ONLY).
// Consumed fleets are removed at the end.
func (s *Space) Meet(g *engine.Game, ctx TraderContext, rng engine.Rand) ([]Encounter, []PlanetTrade) {
	var enc []Encounter
	var trades []PlanetTrade
	consumed := map[int]bool{}
	for ti := range s.Traders {
		enc = append(enc, s.fleetMeetings(g, ctx, ti, consumed, rng)...)
		trades = append(trades, s.planetMeetings(g, ctx, ti, rng)...)
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
	return enc, trades
}

// fleetMeetings is one Trader's meetings with fleets (OBJECTS.md
// "Encounters", CONFIRMED OB-004, OB-023, OB-026, OB-030-T,
// WT-002..WT-004): the Trader meets every fleet at exactly its position,
// in fleet order. Cargo (minerals only) below 5,000 kT: refused. Owner
// already served by this Trader: refused, fleet kept. Otherwise the owner
// is marked served, the fleet is consumed (ships and cargo) and the
// reward follows. A consumed fleet is not offered to another Trader. A
// gift fleet created here is offered if it lands later in fleet order
// (BINARY-ONLY); it never trades, and having not moved gets no message
// (the caller sends refusal messages only for fleets that moved).
func (s *Space) fleetMeetings(g *engine.Game, ctx TraderContext, ti int, consumed map[int]bool, rng engine.Rand) []Encounter {
	var out []Encounter
	t := &s.Traders[ti]
	order := fleetOrder(g)
	for k := 0; k < len(order); k++ {
		f := &g.Fleets[order[k]]
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
			id, n := f.ID, len(g.Fleets)
			e.Reward = s.reward(g, ctx, t, f.Owner, f.Pos, cargo, rng)
			if len(g.Fleets) != n {
				// A gift fleet that lands later in fleet order is
				// offered like any fleet; it has no minerals.
				order = fleetOrder(g)
				for j, fi := range order {
					if g.Fleets[fi].ID == id {
						k = j
					}
				}
			}
		}
		out = append(out, e)
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
	// Legacy.TraderLastRedraw reproduces the original's LEGACY BUG that a
	// player whose 25th part redraw found an unowned part gets a ship
	// instead (OBJECTS.md "Encounters", BINARY-ONLY).
	if bit == BitShip || s.TraderParts.Owns(owner, bit) || (redraws == 25 && g.Rules.Legacy.TraderLastRedraw) {
		return s.shipGift(g, ctx, owner, pos, rng)
	}
	s.TraderParts.give(owner, bit)
	return Reward{Kind: ItemPart, Part: bit}
}

// --- Ship gifts ---

// giftDesign is one of the Trader's ship designs (OBJECTS.md
// "Encounters", ship, CONFIRMED WT-003 B, WT-004 B, C, OB-026).
//
// The loadouts are in the hull's slot order, "in two slots" meaning
// consecutive slots, each filled to its maximum (OBJECTS.md "Encounters",
// slot layout, MEASURED WT-004).
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
// ship; CONFIRMED WT-003 B, WT-004, OB-026; draw order, matching, counts
// after year index 100, computer players and the new fleet BINARY-ONLY).
// Computer players draw nothing and get nothing. Then the design:
// rand(4) (rand(3) after year index 100), 0 the Lifeboat, else rand(2)
// for the Scout (0) or Probe (1). A matching earlier gift design is
// reused, else the design goes into the first empty ship design slot. No
// slot or 512 fleets: no ship and no count draws. Then the count: 2 on
// rand(3) = 0, else 1; after year index 100, unless the game has a single
// human player, + rand(⌊year index/100⌋ + 1); capped at 5; then for the
// Scout and Probe + rand(count + 1). The new fleet takes the owner's
// lowest unused fleet number counting from 1 (as a launch, CONFIRMED
// SL-02), battle plan 0, full fuel and no further waypoints, at the trade
// point.
//
// Consumed fleets, the traded one included, still hold their numbers
// then: they leave the fleet list after all the meetings (BINARY-ONLY).
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
	name, hull, fills := giftDesign(kind)
	d, err := engine.Components().NewDesign(name, hull, fills)
	if err != nil {
		panic(err) // the gift designs are fixed and fit their hulls
	}
	fleets := 0
	used := map[int]bool{}
	for _, f := range g.Fleets {
		if f.Owner == owner {
			fleets++
			used[f.Number] = true
		}
	}
	di := -1
	if fleets < maxFleets {
		di = s.giftSlot(g, owner, d)
	}
	if di < 0 {
		r.NoRoom = true
		return r
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
	num := 1
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

// giftSlot returns the design index for a gift design: a matching earlier
// gift design still in one of the owner's ship slots (not rewritten), else
// a new design, marked as a gift, in the first empty ship slot, else -1.
func (s *Space) giftSlot(g *engine.Game, owner int, d engine.Design) int {
	taken := map[int]bool{}
	for _, ds := range g.DesignSlots {
		if ds.Owner != owner || ds.Starbase {
			continue
		}
		taken[ds.Slot] = true
		if ds.Design >= 0 && ds.Design < len(g.Designs) && s.isGift(ds.Design) && sameLoadout(g.Designs[ds.Design], d) {
			return ds.Design
		}
	}
	for slot := range maxShipDesigns {
		if !taken[slot] {
			g.Designs = append(g.Designs, d)
			di := len(g.Designs) - 1
			g.DesignSlots = append(g.DesignSlots, engine.DesignSlot{Owner: owner, Slot: slot, Design: di})
			s.GiftDesigns = append(s.GiftDesigns, di)
			return di
		}
	}
	return -1
}

func (s *Space) isGift(di int) bool {
	for _, x := range s.GiftDesigns {
		if x == di {
			return true
		}
	}
	return false
}

// sameLoadout is the gift match (OBJECTS.md "Encounters", matching,
// BINARY-ONLY): the same hull, number of slots and slot counts, and the
// same part in every non-empty slot; the name is not compared.
func sameLoadout(a, b engine.Design) bool {
	if a.Hull.Name != b.Hull.Name || len(a.Slots) != len(b.Slots) {
		return false
	}
	for i := range a.Slots {
		x, y := a.Slots[i], b.Slots[i]
		if x.Count != y.Count || (x.Count > 0 && x.Part.Name != y.Part.Name) {
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

// planetMeetings is one Trader's trades with computer players' planets,
// after its fleets (OBJECTS.md "Computer players' planets", CONFIRMED in
// part TP-001, TP-002, O-53; order, range and ship items BINARY-ONLY).
// Planets go in planet-number order, and the scan stops at the first
// planet more than 100 ly east of the Trader in x. A planet trades when
// d² ≤ 10,000, it has a starbase, its owner is a Harder or Expert
// computer player (Standard and Easy excluded, BINARY-ONLY) this Trader
// has not served, and its surface minerals reach 5,000 kT, 3,500 for
// Harder (CONFIRMED O-53; the exact edge NOT RUN).
//   - Part item, and a ship item as part bit 12: an owner lacking the bit
//     gains it; one owning it redraws rand(13) up to 50 times for a bit it
//     lacks (bit 12 gives only the bit). Price: all the surface minerals.
//   - Research item, or no bit found: an owner whose tech sums to 150 or
//     more gets nothing and stays available; otherwise the lowest field
//     gains a level six times. Price: 5,000 kT (3,500 for Harder), from
//     germanium, then boranium, then ironium.
//
// The owner is marked served; no message is sent. Human players' planets
// never trade.
func (s *Space) planetMeetings(g *engine.Game, ctx TraderContext, ti int, rng engine.Rand) []PlanetTrade {
	var out []PlanetTrade
	if ctx.Computer == nil || ctx.Level == nil {
		return out
	}
	t := &s.Traders[ti]
	order := make([]int, len(g.Planets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return g.Planets[order[i]].ID < g.Planets[order[j]].ID })
	for _, pi := range order {
		pl := &g.Planets[pi]
		if pl.Pos.X-t.Pos.X > 100 {
			break
		}
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
		if t.Item.Kind != ItemResearch {
			bit := BitShip
			if t.Item.Kind == ItemPart {
				bit = t.Item.Bit
			}
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
	return out
}
