package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// lab is a small two-player game (players 0 and 1, mutual enemies, JOAT)
// with designs built from the component table, like the OB and MF
// Combat Lab runs.
type lab struct {
	g       *engine.Game
	designs map[string]int
}

func newLab(t *testing.T) *lab {
	t.Helper()
	g := &engine.Game{Rules: engine.ElegyRules(), Players: []engine.Player{
		{Race: engine.Race{PRT: engine.PRTJackOfAllTrades}, Relations: []engine.Relation{engine.RelationFriend, engine.RelationEnemy},
			Plans: []engine.BattlePlan{{Attack: engine.AttackEnemies}}},
		{Race: engine.Race{PRT: engine.PRTJackOfAllTrades}, Relations: []engine.Relation{engine.RelationEnemy, engine.RelationFriend},
			Plans: []engine.BattlePlan{{Attack: engine.AttackEnemies}}},
	}}
	l := &lab{g: g, designs: map[string]int{}}
	cat := engine.Components()
	add := func(name, hull string, fills ...engine.SlotFill) {
		d, err := cat.NewDesign(name, hull, fills)
		if err != nil {
			t.Fatal(err)
		}
		l.designs[name] = len(g.Designs)
		g.Designs = append(g.Designs, d)
	}
	eng := func(e string) engine.SlotFill { return engine.SlotFill{Slot: 0, Part: e, Count: 1} }
	tankArmor := engine.SlotFill{Slot: 4, Part: "Superlatanium", Count: 2}
	add("Laser DD", "Destroyer", eng("Quick Jump 5"), engine.SlotFill{Slot: 1, Part: "Laser", Count: 1}, engine.SlotFill{Slot: 2, Part: "Laser", Count: 1})
	add("Mini Gun DD", "Destroyer", eng("Quick Jump 5"), engine.SlotFill{Slot: 1, Part: "Mini Gun", Count: 1})
	add("Gatling DD", "Destroyer", eng("Quick Jump 5"), engine.SlotFill{Slot: 1, Part: "Gatling Gun", Count: 1}, engine.SlotFill{Slot: 2, Part: "Gatling Gun", Count: 1})
	add("Sapper DD", "Destroyer", eng("Quick Jump 5"), engine.SlotFill{Slot: 1, Part: "Pulsed Sapper", Count: 1})
	add("Tank", "Destroyer", eng("Trans-Galactic Drive"), tankArmor)
	add("Scoop Tank", "Destroyer", eng("Trans-Galactic Fuel Scoop"), tankArmor)
	add("Mizer Tank", "Destroyer", eng("Fuel Mizer"), tankArmor)
	add("Shield Tank", "Destroyer", eng("Trans-Galactic Drive"), tankArmor, engine.SlotFill{Slot: 3, Part: "Complete Phase Shield", Count: 1})
	add("Cloak Tank", "Destroyer", eng("Trans-Galactic Drive"), tankArmor, engine.SlotFill{Slot: 6, Part: "Stealth Cloak", Count: 1})
	add("Freighter", "Medium Freighter", eng("Quick Jump 5"), engine.SlotFill{Slot: 2, Part: "Crobmnium", Count: 1})
	add("MML", "Mini Mine Layer", eng("Quick Jump 5"), engine.SlotFill{Slot: 1, Part: "Mine Dispenser 40", Count: 2})
	add("MML mixed", "Mini Mine Layer", eng("Quick Jump 5"), engine.SlotFill{Slot: 1, Part: "Mine Dispenser 40", Count: 2}, engine.SlotFill{Slot: 2, Part: "Heavy Dispenser 50", Count: 2})
	add("SML", "Super Mine Layer", engine.SlotFill{Slot: 0, Part: "Quick Jump 5", Count: 3}, engine.SlotFill{Slot: 1, Part: "Mine Dispenser 40", Count: 2})
	add("Frigate MD", "Frigate", eng("Quick Jump 5"), engine.SlotFill{Slot: 2, Part: "Mine Dispenser 40", Count: 2})
	add("Frigate MCM", "Frigate", eng("Quick Jump 5"), engine.SlotFill{Slot: 2, Part: "Multi Contained Munition", Count: 1})
	add("Frigate ST", "Frigate", eng("Quick Jump 5"), engine.SlotFill{Slot: 2, Part: "Speed Trap 20", Count: 3})
	add("Laser Fort", "Orbital Fort", engine.SlotFill{Slot: 1, Part: "Laser", Count: 2})
	return l
}

// fleet adds a fleet of owner at pos with (design, count) pairs.
func (l *lab) fleet(owner int, pos engine.Point, stacks ...any) int {
	f := engine.Fleet{ID: len(l.g.Fleets) + 1, Number: len(l.g.Fleets), Owner: owner, Pos: pos}
	for i := 0; i < len(stacks); i += 2 {
		f.Stacks = append(f.Stacks, engine.Stack{Design: l.designs[stacks[i].(string)], Count: stacks[i+1].(int)})
	}
	l.g.Fleets = append(l.g.Fleets, f)
	return len(l.g.Fleets) - 1
}

var origin = engine.Point{X: 1000, Y: 1000}

func at(dx, dy int) engine.Point { return engine.Point{X: 1000 + dx, Y: 1000 + dy} }

// script is an engine.Rand returning fixed values, then 999.
type script []int

func (s *script) Intn(n int) int {
	if len(*s) == 0 {
		return n - 1
	}
	v := (*s)[0]
	*s = (*s)[1:]
	return v
}

// count is an engine.Rand that never hits and counts its draws.
type count struct{ n int }

func (c *count) Intn(n int) int { c.n++; return n - 1 }

// Decay (OBJECTS.md "Decay", CONFIRMED OB-002, OB-014-A, OB-015, OB-016).
func TestConfirmedDecay(t *testing.T) {
	cases := []struct {
		name   string
		m      Minefield
		n      int
		sd     bool
		before int
		after  int
	}{
		{"no planets", Minefield{Kind: Standard}, 0, false, 1000, 980},
		{"no planets, 100", Minefield{Kind: Standard}, 0, false, 100, 90},
		{"speed bump 100", Minefield{Kind: SpeedBump}, 0, false, 100, 98},
		{"2 planets", Minefield{Kind: Standard}, 2, false, 1000, 900},
		{"3 planets", Minefield{Kind: Standard}, 3, false, 2000, 1720},
		{"SD 2 planets", Minefield{Kind: Standard}, 2, true, 1000, 960},
		{"22 planets", Minefield{Kind: Standard}, 22, false, 40000, 20000},
		{"SD 22 planets", Minefield{Kind: Standard}, 22, true, 40000, 30400},
		{"detonating", Minefield{Kind: Standard, Detonate: true}, 0, false, 1000, 730},
		{"detonating speed bump", Minefield{Kind: SpeedBump, Detonate: true}, 0, false, 1000, 730},
	}
	for _, c := range cases {
		c.m.Count = c.before
		if got := c.before - DecayLoss(c.m, c.n, c.sd); got != c.after {
			t.Errorf("%s: %d → %d, want %d", c.name, c.before, got, c.after)
		}
	}
	// Decay counts planets inside the field and removes fields that run
	// out; a 10-mine standard field loses its minimum of 10.
	l := newLab(t)
	l.g.Planets = []engine.Planet{{Pos: at(5, 0), Owner: engine.NoOwner}, {Pos: at(0, 20), Owner: 0}, {Pos: at(40, 0)}}
	s := Space{Minefields: []Minefield{{Owner: 0, Kind: Standard, Pos: origin, Count: 1000}, {Owner: 1, Number: 0, Pos: at(500, 0), Count: 10}}}
	s.Decay(l.g)
	if len(s.Minefields) != 1 || s.Minefields[0].Count != 900 {
		t.Errorf("decay: %+v", s.Minefields)
	}
}

// Lay amounts (OBJECTS.md "Laying", CONFIRMED OB-002, OB-024).
func TestConfirmedLayAmounts(t *testing.T) {
	l := newLab(t)
	cases := []struct {
		design string
		ships  int
		want   [NumMineKinds]int
	}{
		{"MML", 1, [NumMineKinds]int{160, 0, 0}},
		{"MML", 3, [NumMineKinds]int{480, 0, 0}},
		{"Frigate MD", 1, [NumMineKinds]int{80, 0, 0}},
		{"Frigate MCM", 1, [NumMineKinds]int{40, 0, 0}},
		{"Frigate ST", 1, [NumMineKinds]int{0, 0, 60}},
		{"MML mixed", 1, [NumMineKinds]int{160, 200, 0}},
		{"SML", 1, [NumMineKinds]int{160, 0, 0}},
		{"Laser DD", 1, [NumMineKinds]int{}},
	}
	for _, c := range cases {
		fi := l.fleet(0, origin, c.design, c.ships)
		if got := LayAmounts(l.g, &l.g.Fleets[fi]); got != c.want {
			t.Errorf("%d %s: %v, want %v", c.ships, c.design, got, c.want)
		}
	}
	// Two kinds lay two fields (OB-002-I); SD half lay (OB-014-C).
	s := Space{}
	fi := l.fleet(0, origin, "MML mixed", 1)
	res := s.Lay(l.g, []Layer{{Fleet: fi}})
	if len(s.Minefields) != 2 || len(res) != 2 || s.Minefields[0].Count+s.Minefields[1].Count != 360 {
		t.Errorf("mixed lay: %+v", s.Minefields)
	}
	s = Space{}
	fi = l.fleet(0, origin, "MML", 1)
	s.Lay(l.g, []Layer{{Fleet: fi, Half: true}})
	if s.Minefields[0].Count != 80 {
		t.Errorf("half lay: %+v", s.Minefields)
	}
}

// Merging into the nearest own field, in fleet order (OBJECTS.md "Laying",
// CONFIRMED OB-002-G, MF-12) and the merge cap (MF-10).
func TestConfirmedLayMerge(t *testing.T) {
	l := newLab(t)
	s := Space{Minefields: []Minefield{{Owner: 0, Kind: Standard, Pos: at(-10, 0), Count: 390}}}
	fi := l.fleet(0, origin, "MML", 1)
	s.Lay(l.g, []Layer{{Fleet: fi}})
	if m := s.Minefields[0]; len(s.Minefields) != 1 || m.Count != 550 || m.Pos != at(-8, 0) {
		t.Errorf("OB-002-G: %+v", s.Minefields)
	}

	for _, order := range []struct {
		first, second engine.Point
		want          engine.Point
	}{{at(10, 0), at(0, 10), at(1, 2)}, {at(0, 10), at(10, 0), at(2, 1)}} {
		l := newLab(t)
		s := Space{Minefields: []Minefield{{Owner: 0, Kind: Standard, Pos: origin, Count: 390}}}
		a := l.fleet(0, order.first, "MML", 1)
		b := l.fleet(0, order.second, "MML", 1)
		s.Lay(l.g, []Layer{{Fleet: b}, {Fleet: a}})
		if m := s.Minefields[0]; len(s.Minefields) != 1 || m.Count != 710 || m.Pos != order.want {
			t.Errorf("MF-12 %v first: %+v, want %v", order.first, s.Minefields, order.want)
		}
	}

	for _, c := range []struct {
		count  int
		merged bool
	}{{1_050_000, false}, {999_500, true}} {
		l := newLab(t)
		s := Space{Minefields: []Minefield{{Owner: 0, Kind: Standard, Pos: origin, Count: c.count}}}
		fi := l.fleet(0, at(30, 0), "MML", 1)
		s.Lay(l.g, []Layer{{Fleet: fi}})
		if merged := len(s.Minefields) == 1; merged != c.merged {
			t.Errorf("MF-10 %d: merged %v, want %v", c.count, merged, c.merged)
		}
	}
}

// The per-player field limit (OBJECTS.md "Laying", MEASURED MF-11, MF-13).
func TestConfirmedFieldLimit(t *testing.T) {
	fields := func(n int) []Minefield {
		var ms []Minefield
		for i := range n {
			ms = append(ms, Minefield{Owner: 0, Number: i, Kind: Standard, Pos: at(0, 100*i+5000), Count: 10})
		}
		return ms
	}
	l := newLab(t)
	fi := l.fleet(0, origin, "MML", 1)
	s := Space{Minefields: fields(511)}
	if r := s.Lay(l.g, []Layer{{Fleet: fi}}); r[0].Field < 0 || s.Minefields[r[0].Field].Number != 511 {
		t.Errorf("511 fields: %+v", r)
	}
	s = Space{Minefields: fields(512)}
	if r := s.Lay(l.g, []Layer{{Fleet: fi}}); r[0].Field != -1 || len(s.Minefields) != 512 {
		t.Errorf("512 fields: %+v", r)
	}
	// A layer inside an existing field merges at the limit.
	s = Space{Minefields: append(fields(511), Minefield{Owner: 0, Number: 511, Pos: origin, Count: 100})}
	if r := s.Lay(l.g, []Layer{{Fleet: fi}}); r[0].Field < 0 || r[0].New {
		t.Errorf("merge at the limit: %+v", r)
	}
	// LEGACY BUG: with another object sorting after, 511 is the limit.
	l.g.Rules = engine.FaithfulRules()
	s = Space{Minefields: fields(511), OtherObjects: 1}
	if r := s.Lay(l.g, []Layer{{Fleet: fi}}); r[0].Field != -1 {
		t.Errorf("MF-13b: %+v", r)
	}
	s = Space{Minefields: append(fields(511), Minefield{Owner: 1, Number: 0, Pos: at(-3000, 0), Count: 10})}
	if r := s.Lay(l.g, []Layer{{Fleet: fi}}); r[0].Field != -1 {
		t.Errorf("MF-13a: %+v", r)
	}
}

// Sweep ratings and amounts (OBJECTS.md "Sweeping", CONFIRMED OB-001,
// OB-007, OB-008, OB-010-S). The ratings agree with the component table's
// mines-swept column.
func TestConfirmedSweep(t *testing.T) {
	cat := engine.Components()
	for _, c := range cat.Components {
		want, ok := c.Stats["mines_swept"].(float64)
		if !ok {
			continue
		}
		d, err := cat.NewDesign("x", "Destroyer", []engine.SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}, {Slot: 1, Part: c.Name, Count: 1}})
		if err != nil {
			t.Fatal(err)
		}
		if got := SweepRating(d); got != int(want) {
			t.Errorf("%s: rating %d, table %v", c.Name, got, want)
		}
	}

	sweep := func(design string, ships, fleets, field int, kind MineKind, pos engine.Point) int {
		l := newLab(t)
		for range fleets {
			l.fleet(0, pos, design, ships)
		}
		s := Space{Minefields: []Minefield{{Owner: 1, Kind: kind, Pos: origin, Count: field}}}
		s.Sweep(l.g)
		if len(s.Minefields) == 0 {
			return 0
		}
		return s.Minefields[0].Count
	}
	cases := []struct {
		name                  string
		design                string
		ships, fleets, before int
		kind                  MineKind
		pos                   engine.Point
		after                 int
	}{
		{"Laser DD", "Laser DD", 1, 1, 980, Standard, origin, 960},
		{"Mini Gun DD", "Mini Gun DD", 1, 1, 980, Standard, origin, 772},
		{"Gatling DD", "Gatling DD", 1, 1, 1960, Standard, origin, 968},
		{"Sapper DD", "Sapper DD", 1, 1, 980, Standard, origin, 980},
		{"three in one fleet", "Laser DD", 3, 1, 980, Standard, origin, 920},
		{"two fleets", "Laser DD", 1, 2, 980, Standard, origin, 940},
		{"speed bump", "Laser DD", 1, 1, 980, SpeedBump, origin, 974},
		{"never past itself", "Gatling DD", 1, 1, 1000, Standard, at(20, 0), 399},
	}
	for _, c := range cases {
		if got := sweep(c.design, c.ships, c.fleets, c.before, c.kind, c.pos); got != c.after {
			t.Errorf("%s: %d → %d, want %d", c.name, c.before, got, c.after)
		}
	}

	// Battle plans and relations (OB-001-E, OB-007, OB-008).
	plans := []struct {
		attack engine.AttackWho
		player int
		rel    engine.Relation
		swept  bool
	}{
		{engine.AttackNobody, 0, engine.RelationEnemy, false},
		{engine.AttackEnemies, 0, engine.RelationNeutral, false},
		{engine.AttackNeutralsAndEnemies, 0, engine.RelationNeutral, true},
		{engine.AttackEveryone, 0, engine.RelationNeutral, true},
		{engine.AttackNeutralsAndEnemies, 0, engine.RelationFriend, false},
		{engine.AttackEveryone, 0, engine.RelationFriend, true},
		{engine.AttackPlayer, 1, engine.RelationFriend, true},
	}
	for _, p := range plans {
		l := newLab(t)
		l.g.Players[0].Plans = []engine.BattlePlan{{Attack: p.attack, Player: p.player}}
		l.g.Players[0].Relations[1] = p.rel
		l.fleet(0, origin, "Laser DD", 1)
		s := Space{Minefields: []Minefield{{Owner: 1, Kind: Standard, Pos: origin, Count: 1000}}}
		if swept := len(s.Sweep(l.g)) > 0; swept != p.swept {
			t.Errorf("plan %d (player %d) vs %d: swept %v, want %v", p.attack, p.player, p.rel, swept, p.swept)
		}
		// The sweeper's owner learns the field (SCANNING.md, BINARY-ONLY).
		if s.Minefields[0].KnownBy(0) != p.swept {
			t.Errorf("plan %d vs %d: known %v", p.attack, p.rel, s.Minefields[0].KnownBy(0))
		}
	}

	// Starbases: a Laser Fort sweeps 80 (OB-007-C), not a friend's field
	// (OB-008-A).
	for _, rel := range []engine.Relation{engine.RelationNeutral, engine.RelationFriend} {
		l := newLab(t)
		l.g.Players[0].Relations[1] = rel
		l.g.Planets = []engine.Planet{{Pos: origin, Owner: 0, HasStarbase: true, StarbaseDesign: l.designs["Laser Fort"]}}
		s := Space{Minefields: []Minefield{{Owner: 1, Kind: Standard, Pos: origin, Count: 940}}}
		s.Sweep(l.g)
		want := 860
		if rel == engine.RelationFriend {
			want = 940
		}
		if got := s.Minefields[0].Count; got != want {
			t.Errorf("Laser Fort vs %d: %d, want %d", rel, got, want)
		}
	}
}

// Effective warp and safe warp (OBJECTS.md "Hits on moving fleets",
// CONFIRMED OB-010, MF-3).
func TestConfirmedEffectiveWarp(t *testing.T) {
	for _, c := range []struct{ d, e int }{{30, 6}, {17, 4}, {36, 6}, {26, 5}, {50, 7}, {81, 9}, {0, 3}, {200, 10}} {
		if got := EffectiveWarp(c.d); got != c.e {
			t.Errorf("%d ly: warp %d, want %d", c.d, got, c.e)
		}
	}
	// 30 ly inside a heavy field at warp 9: no draw at all (OB-010 H0–H4).
	l := newLab(t)
	fi := l.fleet(0, at(-100, 0), "Tank", 1)
	s := &Space{Minefields: []Minefield{{Owner: 1, Kind: Heavy, Pos: origin, Count: 10000}}}
	c := &count{}
	if st := CheckStep(l.g, s, &l.g.Fleets[fi], at(-20, 0), at(200, 0), 30, c); st.Hit || c.n != 0 {
		t.Errorf("heavy at 30 ly: %+v, %d draws", st, c.n)
	}
}

// Stops along a path: one draw per whole ly inside, the stop where the
// hitting draw falls, no stop for the owner, a friend or the field owner's
// friend (OBJECTS.md "Hits on moving fleets", CONFIRMED MF-1, MF-5, MF-6).
func TestConfirmedCheckStep(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, at(-100, 0), "Tank", 1)
	f := &l.g.Fleets[fi]
	s := &Space{Minefields: []Minefield{{Owner: 1, Kind: Standard, Pos: origin, Count: 2500}}} // radius 50
	// 81 ly at warp 9 from 100 ly west: inside from 50 to 81.
	c := &count{}
	if st := CheckStep(l.g, s, f, f.Pos, at(200, 0), 81, c); st.Hit || c.n != 31 {
		t.Errorf("draws: %+v, %d, want 31", st, c.n)
	}
	// The odds at warp 9 are (9 − 4)·3 = 15 per mille: a draw of 14 hits,
	// 15 does not.
	r := script{999, 999, 15, 14}
	if st := CheckStep(l.g, s, f, f.Pos, at(200, 0), 81, &r); !st.Hit || st.Distance != 53 || st.Kind != Standard {
		t.Errorf("stop: %+v, want 53 ly", st)
	}
	// A fleet starting inside stops at its start on the first draw.
	r = script{0}
	if st := CheckStep(l.g, s, f, at(-10, 0), at(200, 0), 81, &r); !st.Hit || st.Distance != 0 {
		t.Errorf("first draw: %+v", st)
	}
	// The field owner's own fleets and fleets it treats as friends are
	// never checked.
	l.g.Players[1].Relations[0] = engine.RelationFriend
	c = &count{}
	if st := CheckStep(l.g, s, f, f.Pos, at(200, 0), 81, c); st.Hit || c.n != 0 {
		t.Errorf("owner treats victim as friend: %d draws", c.n)
	}
	l.g.Players[1].Relations[0] = engine.RelationEnemy
	own := l.fleet(1, at(-100, 0), "Tank", 1)
	c = &count{}
	if CheckStep(l.g, s, &l.g.Fleets[own], at(-100, 0), at(200, 0), 81, c); c.n != 0 {
		t.Errorf("owner's fleet: %d draws", c.n)
	}
	// Overlapping fields of one kind are one stretch; a second kind is
	// visited by entry.
	s.Minefields = append(s.Minefields, Minefield{Owner: 1, Number: 1, Kind: Standard, Pos: at(-30, 0), Count: 400})
	c = &count{}
	if CheckStep(l.g, s, f, f.Pos, at(200, 0), 81, c); c.n != 31 {
		t.Errorf("overlap: %d draws, want 31", c.n)
	}
}

// Damage (OBJECTS.md "Hits on moving fleets", CONFIRMED MF-9; detonation
// damage on existing damage, MF-8).
func TestConfirmedMineDamage(t *testing.T) {
	cases := []struct {
		name   string
		kind   MineKind
		stacks []any
		units  []int // per stack, -1 destroyed
	}{
		{"Tank standard", Standard, []any{"Tank", 1}, []int{78}},
		{"Tank heavy", Heavy, []any{"Tank", 1}, []int{312}},
		{"Scoop Tank standard", Standard, []any{"Scoop Tank", 1}, []int{93}},
		{"Scoop Tank heavy", Heavy, []any{"Scoop Tank", 1}, []int{390}},
		{"Mizer Tank standard", Standard, []any{"Mizer Tank", 1}, []int{93}},
		{"Mizer Tank heavy", Heavy, []any{"Mizer Tank", 1}, []int{390}},
		{"Shield Tank standard", Standard, []any{"Shield Tank", 1}, []int{39}},
		{"Shield Tank heavy", Heavy, []any{"Shield Tank", 1}, []int{234}},
		{"Tank + Cloak Tank standard", Standard, []any{"Tank", 1, "Cloak Tank", 1}, []int{62, 15}},
		{"Tank + Cloak Tank heavy", Heavy, []any{"Tank", 1, "Cloak Tank", 1}, []int{234, 78}},
		{"five Laser DDs", Standard, []any{"Laser DD", 5}, []int{250}},
	}
	for _, c := range cases {
		l := newLab(t)
		fi := l.fleet(0, origin, c.stacks...)
		MineDamage(l.g, &l.g.Fleets[fi], c.kind, nil)
		f := l.g.Fleets[fi]
		for i, want := range c.units {
			if got := f.Stacks[i].Damage.Units; got != want {
				t.Errorf("%s stack %d: %d/500, want %d", c.name, i, got, want)
			}
		}
	}
	// MF-8: five Tanks at 250/500 take 100 each → 265/500; a Laser DD at
	// 250/500 takes 500 on 200 armor and is destroyed.
	l := newLab(t)
	fi := l.fleet(0, origin, "Tank", 5)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 100, Units: 250}
	MineDamage(l.g, &l.g.Fleets[fi], Standard, nil)
	if got := l.g.Fleets[fi].Stacks[0].Damage.Units; got != 265 {
		t.Errorf("MF-8 Tanks: %d/500, want 265", got)
	}
	fi = l.fleet(0, origin, "Laser DD", 1)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 100, Units: 250}
	if h := MineDamage(l.g, &l.g.Fleets[fi], Standard, nil); !h[0].Destroyed || len(l.g.Fleets[fi].Stacks) != 0 {
		t.Errorf("MF-8 Laser DD: %+v", h)
	}
	// Speed bumps only stop (OB-024).
	fi = l.fleet(0, origin, "Tank", 1)
	if h := MineDamage(l.g, &l.g.Fleets[fi], SpeedBump, nil); h != nil || l.g.Fleets[fi].Stacks[0].Damage.Units != 0 {
		t.Errorf("speed bump damage: %+v", h)
	}
}

// Mines lost and the paying field (OBJECTS.md "Hits on moving fleets",
// CONFIRMED OB-010-S, OB-024, MF-4).
func TestConfirmedPayingField(t *testing.T) {
	// 400, 3000 and 6000 are measured; 100 and 1020 follow the rule
	// (⌊1020/20⌋ = 51 > 50, so max(50, 10)).
	for _, c := range []struct{ n, lost int }{{400, 20}, {3000, 50}, {6000, 60}, {100, 10}, {1020, 50}} {
		if got := MinesLost(c.n); got != c.lost {
			t.Errorf("%d mines: loses %d, want %d", c.n, got, c.lost)
		}
	}
	l := newLab(t)
	s := &Space{Minefields: []Minefield{
		{Owner: 1, Number: 0, Kind: Heavy, Pos: at(-90, 0), Count: 10000},
		{Owner: 1, Number: 1, Kind: Heavy, Pos: origin, Count: 100},
	}}
	if p := PayingField(l.g, s, 0, Heavy, origin); p != 0 {
		t.Errorf("paying field %d, want the 10,000 field", p)
	}
	fi := l.fleet(0, origin, "Tank", 1)
	h := ApplyHit(l.g, s, fi, Heavy, &script{})
	if h.Paid != 100 || s.Minefields[0].Count != 9900 || s.Minefields[1].Count != 100 {
		t.Errorf("hit: paid %d, fields %+v", h.Paid, s.Minefields)
	}
	if got := l.g.Fleets[fi].Stacks[0].Damage.Units; got != 312 {
		t.Errorf("hit damage %d/500", got)
	}
}

// Destroyed ships leave their cargo share as salvage; an empty fleet's
// loss drops rand(10) kT of each mineral (LEGACY BUG candidate, OB-024).
func TestPredictionHitSalvage(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, at(5, 5), "Laser DD", 1)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 100, Units: 499}
	s := &Space{Minefields: []Minefield{{Owner: 1, Kind: Standard, Pos: origin, Count: 400}}}
	h := ApplyHit(l.g, s, fi, Standard, &script{3, 4, 5})
	if h.Salvage != (engine.Minerals{3, 4, 5}) {
		t.Errorf("empty fleet salvage %v", h.Salvage)
	}
	l.g.Rules.Legacy.EmptyFleetSalvage = false
	fi = l.fleet(0, at(5, 5), "Laser DD", 1)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 100, Units: 499}
	if h := ApplyHit(l.g, s, fi, Standard, &script{3, 4, 5}); h.Salvage != (engine.Minerals{}) {
		t.Errorf("switch off: %v", h.Salvage)
	}
}

// Detonation (OBJECTS.md "Detonation", CONFIRMED OB-002-M, MF-7, MF-8).
func TestConfirmedDetonation(t *testing.T) {
	// OB-002-M: owner's five Laser DDs take half their armor, the enemy's
	// five freighters (125 armor) 100 each, the owner's layer nothing.
	l := newLab(t)
	dd := l.fleet(0, origin, "Laser DD", 5)
	fr := l.fleet(1, at(3, 0), "Freighter", 5)
	ml := l.fleet(0, at(0, 3), "MML", 1)
	s := &Space{Minefields: []Minefield{{Owner: 0, Kind: Standard, Pos: origin, Count: 1000, Detonate: true}}}
	s.Detonate(l.g)
	if u := l.g.Fleets[dd].Stacks[0].Damage.Units; u != 250 {
		t.Errorf("Laser DDs %d/500, want 250", u)
	}
	if u := l.g.Fleets[fr].Stacks[0].Damage.Units; u != 400 {
		t.Errorf("freighters %d/500, want 400 (80%%)", u)
	}
	if u := l.g.Fleets[ml].Stacks[0].Damage.Units; u != 0 {
		t.Errorf("layer %d/500, want 0", u)
	}
	s.Decay(l.g)
	if s.Minefields[0].Count != 730 {
		t.Errorf("after decay %d, want 730", s.Minefields[0].Count)
	}
	// MF-7: heavy, owner's and a friend's Tank take 2000; speed bump
	// damages nobody; an SD owner learns the damaged designs.
	l = newLab(t)
	l.g.Players = append(l.g.Players, engine.Player{})
	l.g.Players[0].Relations = []engine.Relation{engine.RelationFriend, engine.RelationEnemy, engine.RelationFriend}
	l.g.Players[0].Race.PRT = engine.PRTSpaceDemolition
	own := l.fleet(0, origin, "Tank", 1)
	enemy := l.fleet(1, origin, "Tank", 1)
	friend := l.fleet(2, origin, "Tank", 1)
	s = &Space{Minefields: []Minefield{{Owner: 0, Kind: Heavy, Pos: origin, Count: 1000, Detonate: true}}}
	dets := s.Detonate(l.g)
	for _, fi := range []int{own, enemy, friend} {
		if u := l.g.Fleets[fi].Stacks[0].Damage.Units; u != 312 {
			t.Errorf("heavy detonation fleet %d: %d/500", fi, u)
		}
	}
	if len(dets) != 3 || len(dets[1].Disclosed) != 1 || dets[1].Disclosed[0] != l.designs["Tank"] {
		t.Errorf("SD disclosure: %+v", dets)
	}
	l = newLab(t)
	fi := l.fleet(1, origin, "Tank", 1)
	s = &Space{Minefields: []Minefield{{Owner: 0, Kind: SpeedBump, Pos: origin, Count: 1000, Detonate: true}}}
	s.Detonate(l.g)
	if u := l.g.Fleets[fi].Stacks[0].Damage.Units; u != 0 {
		t.Errorf("speed bump detonation %d/500", u)
	}
}

// The path cut (OBJECTS.md "Arithmetic details", "Path cut",
// BINARY-ONLY; east legs CONFIRMED as rates MF-1, MF-3; due north and
// south LEGACY BUG MEASURED MF-15).
func TestConfirmedPathCut(t *testing.T) {
	type c struct {
		from, to, centre engine.Point
		l, n             int
		a, b             int
		ok               bool
	}
	p := func(x, y int) engine.Point { return engine.Point{X: x, Y: y} }
	for i, k := range []c{
		{p(0, 0), p(100, 0), p(50, 3), 100, 100, 41, 59, true},  // exact entry rounded up, exit down
		{p(0, 0), p(100, 0), p(-5, 0), 100, 100, 0, 5, true},    // centre behind the start
		{p(0, 0), p(100, 0), p(50, 10), 100, 100, 0, 0, false},  // tangent
		{p(0, 0), p(100, 0), p(50, 0), 30, 100, 0, 0, false},    // beyond this year's travel
		{p(0, 0), p(0, 100), p(0, 50), 100, 100, 0, 0, false},   // due north from outside: never cut
		{p(0, 90), p(0, 200), p(0, 0), 100, 10000, 0, 43, true}, // MF-15: inside, south, 43 ly
	} {
		a, b, ok := cut(k.from, k.to, k.l, k.centre, k.n, true)
		if a != k.a || b != k.b || ok != k.ok {
			t.Errorf("case %d: %d..%d %v, want %d..%d %v", i, a, b, ok, k.a, k.b, k.ok)
		}
	}
	a, b, ok := cut(p(0, 0), p(0, 100), 100, p(0, 50), 100, false)
	if a != 40 || b != 60 || !ok {
		t.Errorf("exact foot: %d..%d %v", a, b, ok)
	}
}

// Stretches merge on overlap, touch, or ending 1 ly before; at most eight
// per kind (OBJECTS.md "Stretches", BINARY-ONLY).
func TestPredictionStretches(t *testing.T) {
	iv := addStretch(nil, 10, 20)
	iv = addStretch(iv, 5, 9) // ends 1 ly before 10
	if len(iv) != 1 || iv[0] != [2]int{5, 20} {
		t.Errorf("1 ly before: %v", iv)
	}
	iv = addStretch(iv, 21, 30) // starts 1 ly after: separate
	iv = addStretch(iv, 20, 21) // touches both: joins the first
	if len(iv) != 2 || iv[0] != [2]int{5, 21} {
		t.Errorf("touch: %v", iv)
	}
	iv = nil
	for i := range 9 {
		iv = addStretch(iv, 10*i, 10*i+3)
	}
	if len(iv) != 8 || iv[7][0] != 70 {
		t.Errorf("ninth: %v", iv)
	}
	// Stop point: offset·s over the rounded leg length, rounded.
	if got := StopPoint(engine.Point{X: 1000, Y: 1000}, engine.Point{X: 1003, Y: 1004}, 2); got != (engine.Point{X: 1001, Y: 1002}) {
		t.Errorf("stop point %v", got)
	}
}

// MF-14 (MEASURED; LEGACY BUG candidate): four 70 kT freighters and a
// 250 kT Privateer carrying 100/100/100 kT and 50 kT of colonists; the
// freighters die, taking 53/53/52/26, and the other 47/47/48 kT are
// dropped as salvage, leaving 24 kT of colonists aboard.
func TestConfirmedMineCargoMF14(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		g := &engine.Game{Rules: engine.ElegyRules(), Designs: []engine.Design{{CargoCapacity: 70, FuelCapacity: 100}, {CargoCapacity: 250, FuelCapacity: 400}}}
		g.Fleets = []engine.Fleet{{Pos: origin, Stacks: []engine.Stack{{Design: 1, Count: 1}},
			Cargo: engine.Cargo{Minerals: engine.Minerals{100, 100, 100}, Colonists: 50}, Fuel: 800}}
		f := &g.Fleets[0]
		g.Rules.Legacy.MineSurvivorSalvage = legacy
		got := mineCargo(g, f, []DesignHit{{Design: 0, Ships: 4, Destroyed: true}}, 530, 800, true, &count{})
		want, aboard := engine.Minerals{47, 47, 48}, engine.Minerals{}
		if !legacy {
			want, aboard = engine.Minerals{}, engine.Minerals{47, 47, 48}
		}
		if got != want || f.Cargo.Minerals != aboard || f.Cargo.Colonists != 24 || f.Fuel != 400 {
			t.Errorf("legacy %v: salvage %v, fleet %+v fuel %d", legacy, got, f.Cargo, f.Fuel)
		}
	}
	// At a planet's exact position no salvage forms; the minerals stay.
	g := &engine.Game{Rules: engine.ElegyRules(), Designs: []engine.Design{{CargoCapacity: 70}, {CargoCapacity: 250}}, Planets: []engine.Planet{{Pos: origin}}}
	g.Fleets = []engine.Fleet{{Pos: origin, Stacks: []engine.Stack{{Design: 1, Count: 1}}, Cargo: engine.Cargo{Minerals: engine.Minerals{100, 100, 100}, Colonists: 50}}}
	if got := mineCargo(g, &g.Fleets[0], []DesignHit{{Design: 0, Ships: 4, Destroyed: true}}, 530, 0, true, &count{}); got != (engine.Minerals{}) || g.Fleets[0].Cargo.Minerals != (engine.Minerals{47, 47, 48}) {
		t.Errorf("at a planet: %v, %+v", got, g.Fleets[0].Cargo)
	}
}

// Existing damage counts ⌊pct·n/100⌋ damaged ships, and the stored units
// round down with a minimum of 1 (OBJECTS.md "Damage on top of existing
// damage", BINARY-ONLY).
func TestPredictionMineDamageRounding(t *testing.T) {
	l := newLab(t)
	// 5 Laser DDs (armor 200) at 30% / 100 units: ⌊1.5⌋ = 1 damaged ship,
	// X = ⌊1·200·100/500⌋ = 40; 5 × 100 standard, no shields: total 540,
	// avg 108, units ⌊108·500/200⌋ = 270 (rounding up the damaged count
	// would give 290).
	fi := l.fleet(1, origin, "Laser DD", 5)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 30, Units: 100}
	MineDamage(l.g, &l.g.Fleets[fi], Standard, nil)
	if got := l.g.Fleets[fi].Stacks[0].Damage; got != (engine.Damage{Pct: 100, Units: 270}) {
		t.Errorf("damage %+v", got)
	}
}

// The detonate setting's chosen rule (OBJECTS.md "The detonate setting",
// BINARY-ONLY): only the owner, only Space Demolition, only standard
// fields; both on and off.
func TestPredictionSetDetonate(t *testing.T) {
	l := newLab(t)
	g := l.g
	g.Players[0].Race.PRT = engine.PRTSpaceDemolition
	s := &Space{Minefields: []Minefield{
		{Owner: 0, Number: 0, Kind: Standard, Count: 1000},
		{Owner: 0, Number: 1, Kind: Heavy, Count: 1000},
		{Owner: 1, Number: 0, Kind: Standard, Count: 1000},
	}}
	cases := []struct {
		player, number int
		on             bool
		want           error
	}{
		{0, 0, true, nil},
		{0, 1, true, ErrNotStandardMine},
		{0, 2, true, ErrNoField},
		{1, 0, true, ErrNotDemolition},
		{0, 0, false, nil},
	}
	for _, c := range cases {
		if err := s.SetDetonate(g, c.player, c.number, c.on); err != c.want {
			t.Fatalf("SetDetonate(%d, %d, %v) = %v, want %v", c.player, c.number, c.on, err, c.want)
		}
		if c.want == nil && s.Minefields[0].Detonate != c.on {
			t.Fatalf("SetDetonate(%d, %d, %v) left Detonate %v", c.player, c.number, c.on, s.Minefields[0].Detonate)
		}
	}
	if s.Minefields[1].Detonate || s.Minefields[2].Detonate {
		t.Fatal("a refused setting changed a field")
	}
	// The number names the player's own field: player 1's order reaches its
	// field 0, not player 0's.
	g.Players[1].Race.PRT = engine.PRTSpaceDemolition
	if err := s.SetDetonate(g, 1, 0, true); err != nil || !s.Minefields[2].Detonate || s.Minefields[0].Detonate {
		t.Fatalf("player 1's own field: err %v, fields %v %v", err, s.Minefields[0].Detonate, s.Minefields[2].Detonate)
	}
}
