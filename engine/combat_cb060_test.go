package engine

import (
	"reflect"
	"testing"
)

// CB-060 (stars-elegy a564ab4, PARITY.md "Late-tech weapons (CB-060)",
// experiments/cb060): the two rules where Elegy's reading failed.

var (
	tGNC    = Part{Name: "Gatling Neutrino Cannon", Kind: PartBeam, Damage: 80, Range: 2, Initiative: 13, Gatling: true}
	tSyncro = Part{Name: "Syncro Sapper", Kind: PartBeam, Damage: 541, Range: 3, Initiative: 14, Sapper: true}
)

func TestConfirmedCB060G1GatlingHitsSecondaryType(t *testing.T) {
	// G1 (CONFIRMED by prediction, 3 of 6 streams took the branch): plan
	// primary "armed ships", secondary "any"; A armed at distance 1, F
	// unarmed at distance 2, 4000 shields each. One shot wrote 80 on A
	// and 80 on F.
	shields := Part{Name: "Shield", Kind: PartShield, Shield: 4000}
	d := []Design{
		testDesign(tFrigate, 10, Slot{tGNC, 1}),
		testDesign(Hull{Armor: 1000}, 10, Slot{tLaser, 1}, Slot{shields, 1}),
		testDesign(Hull{Armor: 1000}, 10, Slot{shields, 1}),
	}
	shot := func(secondary TargetType) []int {
		f := tok(d, 0, 0, 1)
		f.primary, f.secondary = TargetArmed, secondary
		a, u := tok(d, 1, 1, 1), tok(d, 2, 1, 1)
		a.x, u.x = 1, 2
		tb := newTestBattle(panicRand{}, d, f, a, u)
		tb.gatling(0, f.weapons[0])
		var got []int
		for _, h := range tb.hits {
			got = append(got, h.Target, h.Shield, h.Armor)
		}
		return got
	}
	if got, want := shot(TargetAny), []int{1, 80, 0, 2, 80, 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("secondary any: hits (target, shield, armor) %v, want %v", got, want)
	}
	// ASSUMPTION (NOT RUN): a target that matches neither type is left out.
	if got, want := shot(TargetNone), []int{1, 80, 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("secondary none: hits (target, shield, armor) %v, want %v", got, want)
	}
}

func TestMeasuredCB060S2SapperCarry(t *testing.T) {
	// S2 (MEASURED, 5 streams): 2 Syncro Sappers (1082). L has 300
	// shields; M is 2 ships with 4000 shields each. The shot that took
	// L's 300 went on to M by the beam carry rule: 673 with L and M at
	// distance 3, 685 with L at 2 and M at 3, 725 with L at 1 and M at 2.
	// In the stream with M 6 away, the shot stopped after L.
	d := []Design{
		testDesign(tFrigate, 10, Slot{tSyncro, 2}),
		testDesign(Hull{Armor: 1000}, 10, Slot{Part{Name: "Shield", Kind: PartShield, Shield: 300}, 1}),
		testDesign(Hull{Armor: 1000}, 10, Slot{Part{Name: "Shield", Kind: PartShield, Shield: 4000}, 1}),
	}
	for _, c := range []struct {
		l, m int
		want []int // target, shield, armor
	}{
		{3, 3, []int{1, 300, 0, 2, 673, 0}},
		{2, 3, []int{1, 300, 0, 2, 685, 0}},
		{1, 2, []int{1, 300, 0, 2, 725, 0}},
		{3, 6, []int{1, 300, 0}},
	} {
		f := tok(d, 0, 0, 1)
		l, m := tok(d, 1, 1, 1), tok(d, 2, 1, 2)
		l.x, m.x = c.l, c.m
		tb := newTestBattle(panicRand{}, d, f, l, m)
		tb.beam(0, f.weapons[0])
		var got []int
		for _, h := range tb.hits {
			got = append(got, h.Target, h.Shield, h.Armor)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("L at %d, M at %d: hits (target, shield, armor) %v, want %v", c.l, c.m, got, c.want)
		}
	}
}

// bbEngines fills a Battleship's engine slot; CB-060's ships had no
// computer, capacitor or shield.
var bbEngines = SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4}

func TestConfirmedCB060LateWeaponValues(t *testing.T) {
	// CONFIRMED by prediction (CB-060 G2, G3, S1, B), from the catalog's
	// parts on a Battleship with no computer or capacitor, against
	// targets with 4000 shields and no deflector. Only the distances seen
	// are pinned: distance 2 for the Big Mutha Cannon and Mini Gun, and
	// distance 0 for the sappers and the plain beams, are NOT RUN.
	target := testDesign(Hull{Armor: 1000}, 10, Slot{Part{Name: "Shield", Kind: PartShield, Shield: 4000}, 1})
	for _, c := range []struct {
		part  string
		count int
		init  int   // the part's initiative
		shots []int // shield damage at distance 0, 1, 2, ...; -1 for no shot
	}{
		{"Big Mutha Cannon", 1, -1, []int{204, 204}},
		{"Mini Gun", 1, -1, []int{13, 13}},
		{"Phased Sapper", 2, -1, []int{-1, 409, 396, 379}},
		{"Syncro Sapper", 2, -1, []int{-1, 1049, 1017, 973}},
		{"Myopic Disruptor", 1, 9, []int{-1, 152}},
		{"Disruptor", 1, 8, []int{-1, 160, 152}},
		{"Mega Disruptor", 1, 6, []int{-1, 163, 158, 152}},
		{"Mini Blaster", 1, 9, []int{-1, 59}},
		{"Mark IV Blaster", 1, 7, []int{-1, 62, 59}},
		{"Heavy Blaster", 1, 5, []int{-1, 64, 62, 59}},
		{"Streaming Pulverizer", 1, 9, []int{-1, 389}},
		{"Anti-Matter Pulverizer", 1, 8, []int{-1, 411, 389}},
	} {
		fd := lateDesign(t, "Battleship", bbEngines, SlotFill{Slot: 3, Part: c.part, Count: c.count})
		d := []Design{fd, target}
		f := tokenValues(fd, Race{}, false, Cost{})
		f.player, f.design, f.ships, f.fleet, f.planet = 0, 0, 1, -1, -1
		f.tactic, f.primary, f.secondary = TacticMaximizeDamage, TargetAny, TargetAny
		w := f.weapons[0]
		if c.init >= 0 && w.part.Initiative != c.init {
			t.Errorf("%s: initiative %d, want %d", c.part, w.part.Initiative, c.init)
		}
		// Every beam fired at its full catalog range and never beyond; the
		// two gatlings' range 2 is the catalog's (distance 2 NOT RUN).
		reach := len(c.shots) - 1
		if w.part.Gatling {
			reach = 2
		}
		if w.reach != reach {
			t.Errorf("%s: reach %d, want %d", c.part, w.reach, reach)
		}
		for x, want := range c.shots {
			if want < 0 {
				continue
			}
			e := tok(d, 1, 1, 1)
			e.x = x
			ff := f
			tb := newTestBattle(panicRand{}, d, &ff, e)
			if w.part.Gatling {
				tb.gatling(0, w)
			} else {
				tb.beam(0, w)
			}
			if len(tb.hits) != 1 || tb.hits[0].Shield != want || tb.hits[0].Armor != 0 {
				t.Errorf("%s at distance %d: hits %+v, want %d to shields only", c.part, x, tb.hits, want)
			}
		}
	}
}

func TestConfirmedCB060BattleNexus(t *testing.T) {
	// N (CONFIRMED by prediction): a Battleship with one Alpha Torpedo and
	// one Battle Nexus had token initiative 13 and computer 50; without
	// the Nexus, 10 and 0.
	with := lateDesign(t, "Battleship", bbEngines, SlotFill{Slot: 3, Part: "Alpha Torpedo", Count: 1}, SlotFill{Slot: 1, Part: "Battle Nexus", Count: 1})
	without := lateDesign(t, "Battleship", bbEngines, SlotFill{Slot: 3, Part: "Alpha Torpedo", Count: 1})
	for _, c := range []struct {
		d                  Design
		initiative, comput int
	}{{with, 13, 50}, {without, 10, 0}} {
		tk := tokenValues(c.d, Race{}, false, Cost{})
		if tk.initiative != c.initiative || tk.computer != c.comput {
			t.Errorf("%v: initiative %d, computer %d; want %d, %d", c.d.Slots, tk.initiative, tk.computer, c.initiative, c.comput)
		}
	}
}
