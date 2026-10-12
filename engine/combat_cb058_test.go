package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// CB-058: the combat paths still BINARY-ONLY or unobserved after CB-057.
// Pushed as predictions (elegy eaa474e) before lane F's runs; the results
// are stars-elegy #166 (726d943, experiments/cb058, PARITY.md "Round 12").
// Costs and values are Elegy's for a race with no cost LRT at tech 26.

// lateToken is a ship token of catalogue design d for player p.
func lateToken(d Design, i, p, ships int, race Race) *token {
	t := tokenValues(d, race, false, Cost{})
	t.player, t.fleet, t.planet, t.design, t.ships = p, -1, -1, i, ships
	t.tactic, t.primary, t.secondary = TacticMaximizeDamage, TargetAny, TargetAny
	return &t
}

func TestConfirmedCB058ZeroHitTorpedoRecord(t *testing.T) {
	// Q1. COMBAT.md "Torpedoes and missiles" step 4: a target the salvo
	// reaches gets a hit record even with 0 hits. CONFIRMED (CB-058, 6
	// runs: 268 missed one-torpedo salvos each wrote exactly one record
	// with no change). A
	// Destroyer (Long Hump 6, one Alpha Torpedo: accuracy 35, computer 0)
	// fires at a Battleship with six Jammer 50 (jammer 95, no shields):
	// p = 35·(100 − 95)/100 = 1. With the draw a miss (99 of 100 streams)
	// the record is one hit record with 0 hits and 1 miss, and no damage
	// (misses touch shields only, and it has none). The record is read
	// from the battle; the hit or miss itself is the stream's.
	dd := lateDesign(t, "Destroyer",
		SlotFill{Slot: 0, Part: "Long Hump 6", Count: 1},
		SlotFill{Slot: 1, Part: "Alpha Torpedo", Count: 1})
	jam := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 9, Part: "Jammer 50", Count: 3},
		SlotFill{Slot: 10, Part: "Jammer 50", Count: 3})
	d := []Design{dd, jam}
	f, e := lateToken(dd, 0, 0, 1, Race{}), lateToken(jam, 1, 1, 1, Race{})
	if p := hitChance(f.weapons[0].part.Accuracy, f.computer, e.jammer); e.jammer != 95 || e.shield != 0 || p != 1 {
		t.Fatalf("jammer %d shield %d hit chance %d, want 95 0 1", e.jammer, e.shield, p)
	}
	tb := newTestBattle(highRand{}, d, f, e) // the draw a miss
	tb.torpedoes(0, f.weapons[0])
	if want := (BattleHit{Firer: 0, Target: 1, Misses: 1}); len(tb.hits) != 1 || tb.hits[0] != want || e.dmg != (Damage{}) {
		t.Errorf("hits %+v, damage %+v; want one record with 0 hits and 1 miss, no damage", tb.hits, e.dmg)
	}
}

func TestConfirmedCB042TokenCapThreePlayers(t *testing.T) {
	// COMBAT.md "Token cap", CONFIRMED for this layout by CB-042 (two
	// streams; CB-058's Q2 was not run for that reason): with three
	// players, each with 100 one-ship fleets of one design in deep
	// space (fleet ids 0..99, the location's fleets ordered by owner then
	// id), the quota is 255/3 = 85. First pass: player 0's fleet 0, then
	// from the location's last fleet back: player 2's 99..15, player 1's
	// 99..15, player 0's 99..16. That is 255, so the second pass adds
	// nothing. Fleets 0..14 of players 1 and 2 and 1..15 of player 0 sit
	// out, and all three players get the left-out message.
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1})}
	g := &Game{Rules: ElegyRules(), Players: make([]Player, 3), Designs: d}
	var fleets []int
	for o := range 3 {
		for id := range 100 {
			g.Fleets = append(g.Fleets, Fleet{ID: id, Owner: o, Stacks: []Stack{{Design: 0, Count: 1}}})
			fleets = append(fleets, len(g.Fleets)-1)
		}
	}
	b := &battle{g: g, loc: location{planet: -1, fleets: fleets}, players: []int{0, 1, 2}, involved: 3}
	in, missed := b.capTokens(fleets, false)
	joined := map[[2]int]bool{}
	for _, i := range in {
		joined[[2]int{g.Fleets[i].Owner, g.Fleets[i].ID}] = true
	}
	for o := range 3 {
		for id := range 100 {
			want := id >= 15
			if o == 0 {
				want = id == 0 || id >= 16
			}
			if joined[[2]int{o, id}] != want {
				t.Errorf("player %d fleet %d: joined %v, want %v", o, id, !want, want)
			}
		}
	}
	if len(in) != 255 || !reflect.DeepEqual(missed, []int{0, 1, 2}) {
		t.Errorf("%d fleets in, missed %v; want 255 and [0 1 2]", len(in), missed)
	}
	// With a starbase at the location (TestPredictionTokenCapStarbase's
	// counts: 122 and 131 fleets, player 1's fleet 2 with 5 stacks), the
	// starbase counts toward the 255 but not toward its owner's quota.
}

func TestConfirmedCB058CargoShare(t *testing.T) {
	// Q3. COMBAT.md "Salvage": a destroyed ship's share of its fleet's
	// cargo goes with it, by cargo capacity. CONFIRMED (CB-058, 6 runs:
	// the per-event shares were pushed before the run; the run's salvage,
	// 764/528/257, is eaa474e applied afterwards to the observed kill
	// events: these shares plus a third of each hull, less a quarter per
	// event; the fuel share CB-023). A fleet of 4 Large Freighters (Long Hump 6 ×2:
	// capacity 1200, fuel 2600 each) carrying 1000 Fe, 700 Bo, 333 Ge and
	// 500 kT of colonists, with a plan that does not dump cargo, loses one
	// freighter per kill event. moved = C·1200/capacity before, each
	// share truncated, and what is left of moved goes 1 kT at a time to
	// Fe, Bo, Ge, colonists in turn. Cargo left after each:
	//   1 lost: moved 2533·1200/4800 = 633; shares 249 174 83 124 (630),
	//           +1 Fe, Bo, Ge: 250 175 84 124 → 750 525 249 376
	//   2 lost: 1900·1200/3600 = 633 → 500 350 166 251
	//   3 lost: 1267·1200/2400 = 633 → 250 175 83 126
	//   4 lost: everything.
	// The minerals of each share go to the kill event's salvage; the
	// colonists die.
	lf := lateDesign(t, "Large Freighter", SlotFill{Slot: 0, Part: "Long Hump 6", Count: 2})
	if lf.CargoCapacity != 1200 {
		t.Fatalf("Large Freighter capacity %d, want 1200", lf.CargoCapacity)
	}
	d := []Design{lf}
	e := lateToken(lf, 0, 1, 4, Race{})
	e.fleet = 0
	tb := newTestBattle(panicRand{}, d, e)
	tb.g.Fleets = []Fleet{{Owner: 1, Cargo: Cargo{Minerals: Minerals{1000, 700, 333}, Colonists: 500}, Stacks: []Stack{{Design: 0, Count: 4}}}}
	want := []Cargo{
		{Minerals: Minerals{750, 525, 249}, Colonists: 376},
		{Minerals: Minerals{500, 350, 166}, Colonists: 251},
		{Minerals: Minerals{250, 175, 83}, Colonists: 126},
		{},
	}
	for k, w := range want {
		e.ships = 3 - k
		tb.cargoShare(e, 1)
		if got := tb.g.Fleets[0].Cargo; got != w {
			t.Errorf("after %d lost: cargo %+v, want %+v", k+1, got, w)
		}
	}
}

func TestMeasuredCB058SalvageTwoPoints(t *testing.T) {
	// Q4. Salvage at two points in one turn: battles go in the order of
	// each location's first fleet (fleets by owner then id), and each
	// battle's deep-space salvage takes its owner's lowest free number
	// (OBJECTS.md "Salvage", "Owner"). MEASURED (CB-058, 6 runs),
	// consistent with eaa474e: the numbers pushed before the run were for
	// other victims, so this is not CONFIRMED by prediction. The
	// battle order by first fleet stays BINARY-ONLY: CB-058's first-fleet
	// point also had the smaller y, so ascending y fits as well. Player
	// 0's fleets 1 and 3 attack player 1's freighters 2 at (100,100) and 4
	// at (200,200) and destroy them all: the salvage at (100,100), the
	// location of player 0's fleet 1, is player 1's number 0, and the one
	// at (200,200) number 1.
	sf := testDesign(Hull{Name: "Small Freighter", Armor: 25}, 10)
	sf.Hull.Cost.Minerals = Minerals{4, 0, 5}
	for seed := range int64(8) {
		g := combatLabGame(sf, 6, 3)
		// An invented beam reaching the whole board (range 9) kills
		// every freighter before it can get away.
		g.Designs[0].Slots = []Slot{{Part{Name: "Long Beam", Kind: PartBeam, Damage: 500, Range: 9, Initiative: 10}, 2}}
		g.Fleets = append(g.Fleets,
			Fleet{ID: 3, Owner: 0, Pos: Point{200, 200}, Stacks: []Stack{{Design: 0, Count: 6}}},
			Fleet{ID: 4, Owner: 1, Pos: Point{200, 200}, Stacks: []Stack{{Design: 1, Count: 3}}})
		// The second location listed first in the fleet table: the order
		// is by first fleet, not by table position.
		g.Fleets[0], g.Fleets[2] = g.Fleets[2], g.Fleets[0]
		g.Fleets[1], g.Fleets[3] = g.Fleets[3], g.Fleets[1]
		battles(&g, rand.New(rand.NewSource(seed)), map[int]bool{}, nil)
		at := map[Point]int{}
		for _, s := range g.Salvage {
			if s.Owner != 1 {
				t.Errorf("seed %d: salvage %+v not player 1's", seed, s)
			}
			at[s.Pos] = s.Number
		}
		if len(g.Salvage) != 2 || at[Point{100, 100}] != 0 || at[Point{200, 200}] != 1 {
			t.Errorf("seed %d: salvage %+v, want number 0 at (100,100) and 1 at (200,200)", seed, g.Salvage)
		}
	}
}

func TestConfirmedCB058LangstonShellRegeneratingShields(t *testing.T) {
	// Q5. COMBAT.md "Armor": RS halves each armor slot's total, and a
	// shield part's own armor is not halved, in any slot: the halving goes
	// by the part's kind. CONFIRMED (CB-058, 6 runs: every Mega Disruptor
	// hit replays only with 645, 2195 and 712; Croby Sharmor CB-056). The
	// designs separate part kind from slot kind:
	//   A: Rogue, 2 Trans-Galactic Drive, 3 Langston Shell in slot 1
	//      (shield or armor): 450 + 195 = 645 (by slot kind, halved: 547).
	//   B: Battleship, 4 Trans-Galactic Drive, 3 Langston Shell in slot 2
	//      (shield only): 2000 + 195 = 2195 either way.
	//   C: Rogue, 2 Trans-Galactic Drive, 3 Organic Armor in slot 1:
	//      450 + 525/2 = 712 either way (plain 975).
	// Shields: A and B 3 × 125 × 7/5 = 525.
	rs := Race{LRT: LRTs{RegeneratingShields: true}}
	a := lateDesign(t, "Rogue",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 2},
		SlotFill{Slot: 1, Part: "Langston Shell", Count: 3})
	b := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 2, Part: "Langston Shell", Count: 3})
	c := lateDesign(t, "Rogue",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 2},
		SlotFill{Slot: 1, Part: "Organic Armor", Count: 3})
	ta, tb, tc := tokenValues(a, rs, false, Cost{}), tokenValues(b, rs, false, Cost{}), tokenValues(c, rs, false, Cost{})
	got := []int{ta.armor, ta.shield, tb.armor, tb.shield, tc.armor, tokenValues(c, Race{}, false, Cost{}).armor}
	if want := []int{645, 525, 2195, 525, 712, 975}; !reflect.DeepEqual(got, want) {
		t.Errorf("armor A, shield A, armor B, shield B, armor C, plain armor C: %v, want %v", got, want)
	}
}

func TestConfirmedCB058DeathStarReach(t *testing.T) {
	// Q6. COMBAT.md "Token values": a starbase's weapons reach one square
	// further. A Death Star with 8 Heavy Blasters (range 3 in the
	// component table) in slot 1 reaches a ship 4 squares away
	// (Chebyshev) and not one 5 away: CONFIRMED (CB-058, 6 runs). Its
	// armor is the hull's 1500: MEASURED as consistent (the hits fit
	// 1126–1500).
	ds := lateDesign(t, "Death Star", SlotFill{Slot: 1, Part: "Heavy Blaster", Count: 8})
	sb := tokenValues(ds, Race{}, true, Cost{})
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 0, 0, 1, -1, -1
	sb.tactic, sb.primary, sb.secondary = TacticMaximizeDamage, TargetAny, TargetAny
	if sb.armor != 1500 || sb.weapons[0].part.Range != 3 || sb.weapons[0].reach != 4 {
		t.Fatalf("armor %d, range %d, reach %d; want 1500, 3, 4", sb.armor, sb.weapons[0].part.Range, sb.weapons[0].reach)
	}
	d := []Design{ds, testDesign(Hull{Armor: 100}, 10)}
	for _, c := range []struct {
		x    int
		want bool
	}{{4, true}, {5, false}} {
		e := tok(d, 1, 1, 1)
		e.x, e.y = sb.x+c.x, sb.y
		tb := newTestBattle(panicRand{}, d, &sb, e)
		if got := len(tb.eligible(&sb, sb.weapons[0])) == 1; got != c.want {
			t.Errorf("ship %d squares away: in reach %v, want %v", c.x, got, c.want)
		}
	}
}
