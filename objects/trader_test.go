package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Appearance (KERNEL.md "Mystery Trader appearance", CONFIRMED KX-004
// S6–S10): the chance by year index, then warp, coordinates, edge, axis
// and item, in that draw order.
func TestConfirmedTraderAppearance(t *testing.T) {
	s := &Space{}
	c := &count{}
	if _, ok := s.Appear(39, 2, true, c); ok || c.n != 0 {
		t.Errorf("year index 39: appeared or drew (%d)", c.n)
	}
	if _, ok := s.Appear(73, 2, true, c); ok || c.n != 0 {
		t.Errorf("odd year 73: appeared or drew (%d)", c.n)
	}
	if _, ok := s.Appear(72, 2, false, c); ok || c.n != 0 {
		t.Errorf("random events off: appeared")
	}
	// Index 133: the mod-100 test (33) comes before the odd-year rule.
	r := script{1}
	if _, ok := s.Appear(133, 2, true, &r); ok || len(r) != 0 {
		t.Errorf("index 133 made no rand(3) draw")
	}
	// Year 72, medium: chance 0, warp 8+1, free coordinates 1020+5 and
	// 1020+700, edge draw 1 (start on the high edge), axis 1 (free y),
	// item rand(10) = 9 ≥ r → part bit 3.
	r = script{0, 1, 5, 700, 1, 1, 9, 3}
	tr, ok := s.Appear(72, 2, true, &r)
	want := Trader{Warp: 9, Pos: engine.Point{X: 2180, Y: 1025}, Dest: engine.Point{X: 1020, Y: 1720},
		Item: TraderItem{Kind: ItemPart, Bit: BitMegaPolyShell}}
	if !ok || tr.Warp != want.Warp || tr.Pos != want.Pos || tr.Dest != want.Dest || tr.Item != want.Item {
		t.Errorf("appearance %+v, want %+v", tr, want)
	}
	// r: 5 before year index 100, +1 below warp 10, −1 above.
	for _, c := range []struct{ yi, warp, r int }{{50, 9, 6}, {50, 10, 5}, {50, 12, 4}, {150, 10, 3}, {300, 10, 2}} {
		if got := itemChance(c.yi, c.warp); got != c.r {
			t.Errorf("r at %d, warp %d: %d, want %d", c.yi, c.warp, got, c.r)
		}
	}
}

// Movement (OBJECTS.md "Movement", CONFIRMED OB-023, OB-026, OB-031): a
// warp-9 Trader moves 81 ly; one arriving while another exists leaves;
// a lone one stays at the edge with warp max(6, warp − 2) + 1.
func TestConfirmedTraderMovement(t *testing.T) {
	s := &Space{Traders: []Trader{{Pos: engine.Point{X: 1020, Y: 1500}, Dest: engine.Point{X: 2580, Y: 1500}, Warp: 9}}}
	r := script{1}
	s.MoveTraders(3, &r)
	if p := s.Traders[0].Pos; p != (engine.Point{X: 1101, Y: 1500}) {
		t.Errorf("warp 9 moved to %v", p)
	}
	two := &Space{Traders: []Trader{
		{Pos: engine.Point{X: 2570, Y: 1500}, Dest: engine.Point{X: 2580, Y: 1500}, Warp: 9},
		{Pos: engine.Point{X: 1500, Y: 1020}, Dest: engine.Point{X: 1500, Y: 2580}, Warp: 9},
	}}
	r = script{1, 1}
	if moves := two.MoveTraders(3, &r); !moves[0].Left || len(two.Traders) != 1 {
		t.Errorf("arrival with another Trader: %+v", moves)
	}
	for _, c := range []struct{ warp, after int }{{8, 7}, {6, 7}} {
		lone := &Space{Traders: []Trader{{Pos: engine.Point{X: 2570, Y: 1500}, Dest: engine.Point{X: 2580, Y: 1500}, Warp: c.warp}}}
		r = script{1, 1, 50}
		moves := lone.MoveTraders(3, &r)
		tr := lone.Traders[0]
		if moves[0].Left || tr.Warp != c.after || tr.Pos != (engine.Point{X: 2580, Y: 1500}) || tr.Dest.X != 1020 {
			t.Errorf("lone arrival at warp %d: %+v", c.warp, tr)
		}
	}
}

// traderLab is a lab with a Trader at origin.
func traderLab(t *testing.T, item TraderItem) (*lab, *Space) {
	l := newLab(t)
	for i := range l.g.Players {
		l.g.Players[i].Research.Levels = [engine.NumFields]int{3, 3, 3, 3, 3, 3}
	}
	return l, &Space{Traders: []Trader{{Pos: origin, Dest: at(500, 0), Warp: 9, Item: item}}}
}

func cargo(l *lab, fi int, kt int) {
	l.g.Fleets[fi].Cargo.Minerals = engine.Minerals{kt, 0, 0}
}

// Encounters (OBJECTS.md "Encounters", CONFIRMED OB-004, OB-023,
// OB-030-T, OB-026): 4,999 kT refused, 5,000 trades and removes the
// fleet; one reward per player per Trader; a part the player lacks is
// gained.
func TestConfirmedTraderEncounters(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemPart, Bit: BitJumpGate})
	a := l.fleet(0, origin, "Tank", 1)
	b := l.fleet(0, origin, "Tank", 1)
	c := l.fleet(1, origin, "Tank", 1)
	d := l.fleet(1, at(1, 0), "Tank", 1)
	cargo(l, a, 4999)
	cargo(l, b, 5000)
	cargo(l, c, 5000)
	cargo(l, d, 9000)
	ids := []int{l.g.Fleets[a].ID, l.g.Fleets[b].ID, l.g.Fleets[c].ID, l.g.Fleets[d].ID}
	enc := s.Encounters(l.g, TraderContext{}, &script{})
	if len(enc) != 3 || !enc[0].Refused || enc[1].Reward.Kind != ItemPart || enc[2].Reward.Part != BitJumpGate {
		t.Fatalf("encounters %+v", enc)
	}
	if !s.TraderParts.Owns(0, BitJumpGate) || !s.TraderParts.Owns(1, BitJumpGate) {
		t.Errorf("parts %v", s.TraderParts)
	}
	left := map[int]bool{}
	for _, f := range l.g.Fleets {
		left[f.ID] = true
	}
	if !left[ids[0]] || left[ids[1]] || left[ids[2]] || !left[ids[3]] {
		t.Errorf("fleets left %v", left)
	}
	// A second fleet of a served player is kept ("still recovering").
	e := l.fleet(0, origin, "Tank", 1)
	cargo(l, e, 6000)
	if enc := s.Encounters(l.g, TraderContext{}, &script{}); len(enc) != 2 || !enc[1].Recovering {
		t.Errorf("served player: %+v", enc)
	}

	// Two Traders (OB-030-T): Trader 0 takes one fleet of each player and
	// refuses player 0's second, which Trader 1 then takes.
	l, s = traderLab(t, TraderItem{Kind: ItemPart, Bit: BitJumpGate})
	s.Traders = append(s.Traders, Trader{Pos: origin, Warp: 9, Item: TraderItem{Kind: ItemPart, Bit: BitAlienMiner}})
	for _, o := range []int{0, 0, 1} {
		cargo(l, l.fleet(o, origin, "Tank", 1), 5000)
	}
	enc = s.Encounters(l.g, TraderContext{}, &script{})
	if len(l.g.Fleets) != 0 || !s.TraderParts.Owns(0, BitAlienMiner) || s.TraderParts.Owns(1, BitAlienMiner) {
		t.Errorf("two Traders: %d fleets left, parts %v, %+v", len(l.g.Fleets), s.TraderParts, enc)
	}
}

// The research reward (OBJECTS.md "Encounters", research, CONFIRMED
// WT-002): a tech-3 player trading 5,000 kT gains 6 levels; L is adjusted
// by the tech sum; an owned offered part gives research.
func TestConfirmedTraderResearch(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemPart, Bit: BitJumpGate})
	s.TraderParts.give(0, BitJumpGate)
	fi := l.fleet(0, origin, "Tank", 1)
	cargo(l, fi, 5000)
	enc := s.Encounters(l.g, TraderContext{}, &lcg{7})
	sum := 0
	for _, v := range l.g.Players[0].Research.Levels {
		sum += v
	}
	if r := enc[0].Reward; r.Kind != ItemResearch || r.Levels != 6 || len(r.Fields) != 6 || sum != 18+6 {
		t.Errorf("research reward %+v, tech sum %d", r, sum)
	}
	// L = min(10, 6 + ⌊(cargo − 5000)/1200⌋), then the tech-sum bands.
	for _, c := range []struct{ cargo, level, l int }{{5000, 3, 6}, {9800, 3, 10}, {20000, 3, 10}, {5000, 10, 5}, {5000, 12, 4}, {5000, 14, 3}, {5000, 16, 2}, {5000, 18, 1}} {
		p := engine.Player{Research: engine.ResearchState{Levels: [engine.NumFields]int{c.level, c.level, c.level, c.level, c.level, c.level}}}
		if r := researchReward(&p, c.cargo, &lcg{1}); r.Levels != c.l {
			t.Errorf("cargo %d at tech %d: L %d, want %d", c.cargo, c.level, r.Levels, c.l)
		}
	}
	// With 1/4 the lowest field (first on ties): draws 3 (lowest), ...
	p := engine.Player{Research: engine.ResearchState{Levels: [engine.NumFields]int{5, 4, 4, 9, 9, 9}}}
	// L = 6: lowest (field 1), lowest (field 2), random field 5, then the
	// lowest three times (fields 0, 1, 2).
	r := script{3, 3, 0, 5, 3, 3, 3}
	researchReward(&p, 5000, &r)
	if p.Research.Levels != [engine.NumFields]int{6, 6, 6, 9, 9, 10} {
		t.Errorf("field choice %v", p.Research.Levels)
	}
}

// Every field at 26 (OBJECTS.md "Encounters", CONFIRMED WT-004 C): with
// 1/5 nothing; otherwise a part, and a player owning all twelve gets a
// ship.
func TestConfirmedTraderMaxedResearch(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemResearch})
	l.g.Players[0].Research.Levels = [engine.NumFields]int{26, 26, 26, 26, 26, 26}
	fi := l.fleet(0, origin, "Tank", 1)
	cargo(l, fi, 5000)
	if enc := s.Encounters(l.g, TraderContext{}, &script{0}); !enc[0].Reward.Nothing {
		t.Errorf("1/5 branch: %+v", enc[0].Reward)
	}
	s.Traders[0].Served = nil
	fi = l.fleet(0, origin, "Tank", 1)
	cargo(l, fi, 5000)
	if enc := s.Encounters(l.g, TraderContext{}, &script{1, 4}); enc[0].Reward.Kind != ItemPart || enc[0].Reward.Part != BitAlienMiner {
		t.Errorf("part branch: %+v", enc[0].Reward)
	}
	s.Traders[0].Served = nil
	s.TraderParts[0] = 1<<BitShip - 1
	fi = l.fleet(0, origin, "Tank", 1)
	cargo(l, fi, 5000)
	if enc := s.Encounters(l.g, TraderContext{Humans: 1}, &lcg{3}); enc[0].Reward.Kind != ItemShip || enc[0].Reward.Ships < 1 {
		t.Errorf("all parts owned: %+v", enc[0].Reward)
	}
}

// Ship gifts (OBJECTS.md "Encounters", ship, CONFIRMED WT-003 B, WT-004,
// OB-026): a computer player gets nothing and still loses the fleet; a
// human gets a Nubian Lifeboat as a new design at the trade point, with
// full fuel, and a second gift reuses the design.
func TestConfirmedTraderShipGift(t *testing.T) {
	l, s := traderLab(t, TraderItem{Kind: ItemShip})
	ctx := TraderContext{YearIndex: 50, Humans: 1, Computer: func(p int) bool { return p == 1 }}
	h := l.fleet(0, origin, "Tank", 1)
	c := l.fleet(1, origin, "Tank", 1)
	cargo(l, h, 5000)
	cargo(l, c, 5000)
	designs := len(l.g.Designs)
	// Lifeboat (rand(4) = 0), count 1 (rand(3) = 1).
	enc := s.Encounters(l.g, ctx, &script{0, 1})
	got := enc[0].Reward
	if got.Kind != ItemShip || got.Ships != 1 || got.Design != designs || l.g.Designs[got.Design].Hull.Name != "Nubian" {
		t.Fatalf("human gift %+v", got)
	}
	if enc[1].Reward.Ships != 0 || len(l.g.Fleets) != 1 {
		t.Errorf("computer gift %+v, %d fleets", enc[1].Reward, len(l.g.Fleets))
	}
	f := l.g.Fleets[0]
	if f.Pos != origin || f.Owner != 0 || f.Fuel != l.g.Designs[got.Design].FuelCapacity || f.ID != got.NewFleet {
		t.Errorf("gift fleet %+v", f)
	}
	// The Scout and Probe fit the Mini Morph; counts add rand(count + 1).
	for kind := range 3 {
		name, hull, fills := giftDesign(kind)
		if _, err := engine.Components().NewDesign(name, hull, fills); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	s.Traders[0].Served = nil
	h = l.fleet(0, origin, "Tank", 1)
	cargo(l, h, 5000)
	if enc := traded(s.Encounters(l.g, ctx, &script{0, 0})); enc.Reward.Design != designs || enc.Reward.Ships != 2 || len(l.g.Designs) != designs+1 {
		t.Errorf("reused design: %+v", enc.Reward)
	}
	// Scout, count 2, then + rand(3) = 2: 4 ships.
	s.Traders[0].Served = nil
	h = l.fleet(0, origin, "Tank", 1)
	cargo(l, h, 5000)
	if enc := traded(s.Encounters(l.g, ctx, &script{1, 0, 0, 2})); enc.Reward.Ships != 4 || l.g.Designs[enc.Reward.Design].Hull.Name != "Mini Morph" {
		t.Errorf("scout gift %+v", enc.Reward)
	}
}

// traded is the one encounter that traded (the gift fleets at the trade
// point have no cargo and are refused).
func traded(enc []Encounter) Encounter {
	for _, e := range enc {
		if !e.Refused && !e.Recovering {
			return e
		}
	}
	return Encounter{}
}

// Computer players' planets (OBJECTS.md "Computer players' planets",
// CONFIRMED TP-001, TP-002).
func TestConfirmedTraderPlanets(t *testing.T) {
	ctx := TraderContext{Computer: func(p int) bool { return p == 1 }, Level: func(int) int { return 3 }}
	planet := func(l *lab, owner int, surface engine.Minerals, levels [engine.NumFields]int) {
		l.g.Planets = []engine.Planet{{Pos: at(50, 0), Owner: owner, HasStarbase: true, Surface: surface}}
		l.g.Players[owner].Research.Levels = levels
	}
	// TP-001-A: a part the owner lacks; the price is all the surface.
	l, s := traderLab(t, TraderItem{Kind: ItemPart, Bit: BitAlienMiner})
	planet(l, 1, engine.Minerals{3000, 2000, 1000}, [engine.NumFields]int{10, 10, 10, 10, 10, 10})
	if tr := s.PlanetTrades(l.g, ctx, &script{}); len(tr) != 1 || tr[0].Part != BitAlienMiner || l.g.Planets[0].Surface != (engine.Minerals{}) {
		t.Errorf("TP-001-A: %+v", tr)
	}
	// TP-001-B: research, the lowest field six times; 5,000 kT from
	// germanium first.
	l, s = traderLab(t, TraderItem{Kind: ItemResearch})
	planet(l, 1, engine.Minerals{3000, 2000, 1000}, [engine.NumFields]int{10, 10, 10, 13, 10, 10})
	tr := s.PlanetTrades(l.g, ctx, &script{})
	if len(tr) != 1 || l.g.Players[1].Research.Levels != [engine.NumFields]int{12, 11, 11, 13, 11, 11} || l.g.Planets[0].Surface != (engine.Minerals{1000, 0, 0}) {
		t.Errorf("TP-001-B: %+v levels %v surface %v", tr, l.g.Players[1].Research.Levels, l.g.Planets[0].Surface)
	}
	// TP-002-A: the part owned, a new bit drawn instead.
	l, s = traderLab(t, TraderItem{Kind: ItemPart, Bit: BitAlienMiner})
	s.TraderParts.give(1, BitAlienMiner)
	planet(l, 1, engine.Minerals{6000, 0, 0}, [engine.NumFields]int{10, 10, 10, 10, 10, 10})
	if tr := s.PlanetTrades(l.g, ctx, &script{4, 7}); len(tr) != 1 || tr[0].Part != BitMultiContainedMunition {
		t.Errorf("TP-002-A: %+v", tr)
	}
	// TP-002-B: tech sum ≥ 150, nothing, and the Trader stays available.
	l, s = traderLab(t, TraderItem{Kind: ItemResearch})
	planet(l, 1, engine.Minerals{6000, 0, 0}, [engine.NumFields]int{25, 25, 25, 25, 25, 25})
	if tr := s.PlanetTrades(l.g, ctx, &script{}); len(tr) != 0 || has(s.Traders[0].Served, 1) {
		t.Errorf("TP-002-B: %+v", tr)
	}
	// Human players' planets never trade (TP-001-C).
	l, s = traderLab(t, TraderItem{Kind: ItemResearch})
	planet(l, 0, engine.Minerals{6000, 0, 0}, [engine.NumFields]int{10, 10, 10, 10, 10, 10})
	if tr := s.PlanetTrades(l.g, ctx, &script{}); len(tr) != 0 {
		t.Errorf("human planet traded: %+v", tr)
	}
}
