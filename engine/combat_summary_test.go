package engine

import (
	"reflect"
	"testing"
)

func TestBattleSummaryRules(t *testing.T) {
	// stars-elegy MESSAGES.md "Battles and bombing": the summary each
	// battle player gets. Rows name the message; it is CONFIRMED there
	// unless the row says BINARY-ONLY or names an ASSUMPTION (B2).
	c := func(total, lost int) summaryCounts { return summaryCounts{total, lost} }
	for _, r := range []struct {
		name                   string
		players, ships, stacks int
		me, other              summaryCounts
		hit, otherHit          bool
		want                   BattleSummary
	}{
		// One ship a side (0x091–0x095, cs-003, cb035-prevbattle).
		{"0x091 won unhit", 2, 2, 2, c(1, 0), c(1, 1), false, true, SummaryDuelWonUnhit},
		{"0x092 lost to unhit", 2, 2, 2, c(1, 1), c(1, 0), true, false, SummaryDuelLostToUnhit},
		{"0x093 won hit", 2, 2, 2, c(1, 0), c(1, 1), true, true, SummaryDuelWonHit},
		{"0x094 lost to hit", 2, 2, 2, c(1, 1), c(1, 0), true, true, SummaryDuelLostToHit},
		{"0x095 both survived", 2, 2, 2, c(1, 0), c(1, 0), true, true, SummaryDuelBothSurvived},
		{"0x092 both died, your loss reported", 2, 2, 2, c(1, 1), c(1, 1), true, false, SummaryDuelLostToUnhit},
		{"0x094 both died, enemy hit", 2, 2, 2, c(1, 1), c(1, 1), true, true, SummaryDuelLostToHit},
		// One stack a side (0x096–0x09a, cb040, cb026, cb039).
		{"0x096", 2, 9, 2, c(5, 0), c(4, 4), false, true, SummaryStackWonUnhit},
		{"0x097", 2, 9, 2, c(4, 4), c(5, 0), true, false, SummaryStackLostToUnhit},
		{"0x098", 2, 110, 2, c(80, 3), c(30, 30), true, true, SummaryStackWonHit},
		{"0x099", 2, 110, 2, c(30, 30), c(80, 3), true, true, SummaryStackLostToHit},
		{"0x09a", 2, 280, 2, c(140, 10), c(140, 12), true, true, SummaryStackBothSurvived},
		// Other 2-player battles (0x09b–0x0a3, 0x113–0x116).
		{"0x09b", 2, 9, 3, c(5, 0), c(4, 4), false, true, SummaryWonNoLoss},
		{"0x09c", 2, 9, 3, c(4, 4), c(5, 0), true, false, SummaryLostAllNoKills},
		{"0x09d", 2, 9, 3, c(5, 2), c(4, 4), true, true, SummaryWonWithLoss},
		{"0x09d both wiped", 2, 9, 3, c(5, 5), c(4, 4), true, true, SummaryWonWithLoss},
		{"0x09e", 2, 9, 3, c(4, 4), c(5, 2), true, true, SummaryLostAllSomeKills},
		{"0x09f", 2, 9, 3, c(5, 2), c(4, 1), true, true, SummaryBothSurvived},
		{"0x09f no losses", 2, 9, 3, c(5, 0), c(4, 0), false, false, SummaryBothSurvived},
		{"0x0a0", 2, 6, 3, c(5, 0), c(1, 1), false, true, SummaryKilledLoneNoLoss},
		{"0x0a1", 2, 6, 3, c(1, 1), c(5, 0), true, false, SummaryLoneLostNoKills},
		{"0x0a2 (B2) BINARY-ONLY", 2, 6, 3, c(5, 2), c(1, 1), true, true, SummaryKilledLoneWithLoss},
		{"0x0a3 BINARY-ONLY", 2, 6, 3, c(1, 1), c(5, 2), true, true, SummaryLoneLostSomeKills},
		{"0x113 BINARY-ONLY", 2, 6, 3, c(1, 0), c(5, 5), true, true, SummaryLoneWonAll},
		{"0x114 BINARY-ONLY", 2, 6, 3, c(5, 5), c(1, 0), true, true, SummaryLoneEnemyWonAll},
		{"0x115", 2, 6, 3, c(1, 0), c(5, 2), true, true, SummaryLoneSurvived},
		{"0x116", 2, 6, 3, c(5, 2), c(1, 0), true, true, SummaryLoneEnemySurvived},
		{"lone ship wiped with the enemy (B2)", 2, 6, 3, c(1, 1), c(5, 5), true, true, SummaryWonWithLoss},
		{"wiped with the enemy's single ship (B2)", 2, 6, 3, c(5, 5), c(1, 1), true, true, SummaryWonWithLoss},
		// Two ships but not two players' worth of location: a 2-player
		// battle at a location with a third player's ship is not a duel.
		{"not a duel with a bystander", 2, 3, 3, c(1, 0), c(1, 0), true, true, SummaryLoneSurvived},
		// 3 or more players (0x0a4–0x0a8, 0x07e; cb036, CB-033, CB-034, cb042).
		{"0x0a4", 3, 9, 3, c(3, 0), c(6, 6), false, true, SummaryMultiWonNoLoss},
		{"0x0a5 BINARY-ONLY", 3, 9, 3, c(3, 1), c(6, 6), true, true, SummaryMultiWonWithLoss},
		{"0x0a5 everyone wiped BINARY-ONLY", 3, 9, 3, c(3, 3), c(6, 6), true, true, SummaryMultiWonWithLoss},
		{"0x0a6", 3, 9, 3, c(3, 3), c(6, 0), true, false, SummaryMultiLostNoKills},
		{"0x0a7", 3, 9, 3, c(3, 3), c(6, 2), true, true, SummaryMultiLostSomeKills},
		{"0x0a8 you lost nothing", 3, 9, 3, c(3, 0), c(6, 2), false, true, SummaryMultiOneSideNoLoss},
		{"0x0a8 the others lost nothing", 3, 9, 3, c(3, 1), c(6, 0), true, false, SummaryMultiOneSideNoLoss},
		{"0x07e", 3, 9, 3, c(3, 1), c(6, 2), true, true, SummaryMultiMixed},
		{"3 players, 2 ships each side: no duel text", 3, 2, 2, c(1, 0), c(1, 1), false, true, SummaryMultiWonNoLoss},
	} {
		if got := battleSummary(r.players, r.ships, r.stacks, r.me, r.other, r.hit, r.otherHit); got != r.want {
			t.Errorf("%s: %d, want %d", r.name, got, r.want)
		}
	}
}

// summaryBattle is a battle at (100,100) between player 0's fleet of
// ships0 ships and player 1's fleets of one ship each, all of design 0,
// with left0 of player 0's ships and left1 of player 1's fleets left.
func summaryBattle(ships0, fleets1, left0, left1 int) *battle {
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1})}
	g := &Game{Rules: ElegyRules(), Players: make([]Player, 2), Designs: d}
	g.Fleets = append(g.Fleets, Fleet{ID: 0, Owner: 0, Pos: Point{100, 100}, Stacks: []Stack{{Design: 0, Count: ships0}}})
	for i := range fleets1 {
		g.Fleets = append(g.Fleets, Fleet{ID: i, Owner: 1, Pos: Point{100, 100}, Stacks: []Stack{{Design: 0, Count: 1}}})
	}
	b := &battle{g: g, loc: location{pos: Point{100, 100}, planet: -1}, players: []int{0, 1}, involved: 2}
	for i := range g.Fleets {
		t := tok(d, 0, g.Fleets[i].Owner, g.Fleets[i].Stacks[0].Count)
		t.fleet = i
		b.loc.fleets = append(b.loc.fleets, i)
		b.tokens = append(b.tokens, t)
	}
	b.tokens[0].ships = left0
	killed := fleets1 - left1
	for i := 1; i <= killed; i++ {
		b.tokens[i].ships, b.tokens[i].dead = 0, true
	}
	return b
}

func TestConfirmedBattleSummaryDesignGroups(t *testing.T) {
	// CB-048 (stars-elegy cb7/cb048, CONFIRMED): 80 ships in one fleet
	// against thirty one-ship fleets, one design a side. Every Mini
	// Morph died, player 0 lost some: player 0 got 0x098 and player 1
	// 0x099, the one-stack texts, though the location had 31 fleet stacks.
	b := summaryBattle(80, 30, 77, 0)
	b.hits = []BattleHit{{Target: 0, Shield: 10}, {Target: 1, Armor: 3}}
	ev := b.summaries(-1)
	if len(ev) != 2 || ev[0].Player != 0 || ev[1].Player != 1 {
		t.Fatalf("events %+v, want one for player 0 then player 1", ev)
	}
	if BattleSummary(ev[0].Count) != SummaryStackWonHit || BattleSummary(ev[1].Count) != SummaryStackLostToHit {
		t.Errorf("summaries %d, %d; want %d (0x098), %d (0x099)", ev[0].Count, ev[1].Count, SummaryStackWonHit, SummaryStackLostToHit)
	}
	// Other player, totals and losses, and the two groups' designs.
	if want := []int{1, 80, 3, 30, 30, 0, 0}; !reflect.DeepEqual(ev[0].Axes, want) {
		t.Errorf("player 0's axes %v, want %v", ev[0].Axes, want)
	}
	if want := []int{0, 30, 30, 80, 3, 0, 0}; !reflect.DeepEqual(ev[1].Axes, want) {
		t.Errorf("player 1's axes %v, want %v", ev[1].Axes, want)
	}
	// Player 0 never hit: 0x096 and 0x097.
	b.hits = []BattleHit{{Target: 1, Armor: 10}, {Target: 0, Hits: 0, Misses: 1}}
	ev = b.summaries(-1)
	if BattleSummary(ev[0].Count) != SummaryStackWonUnhit || BattleSummary(ev[1].Count) != SummaryStackLostToUnhit {
		t.Errorf("unhit: summaries %d, %d; want %d, %d", ev[0].Count, ev[1].Count, SummaryStackWonUnhit, SummaryStackLostToUnhit)
	}
	// A second design on player 1's side: no longer one stack a side.
	b = summaryBattle(80, 30, 77, 0)
	b.g.Designs = append(b.g.Designs, b.g.Designs[0])
	b.g.Fleets[5].Stacks[0].Design = 1
	ev = b.summaries(-1)
	if BattleSummary(ev[0].Count) != SummaryWonWithLoss || ev[0].Axes[SummaryDesign] != -1 {
		t.Errorf("two designs: summary %d, design %d; want %d (0x09d), -1", ev[0].Count, ev[0].Axes[SummaryDesign], SummaryWonWithLoss)
	}
	// One ship each, both survive: 0x095.
	b = summaryBattle(1, 1, 1, 1)
	ev = b.summaries(7)
	if BattleSummary(ev[0].Count) != SummaryDuelBothSurvived || BattleSummary(ev[1].Count) != SummaryDuelBothSurvived || ev[0].Planet != 7 {
		t.Errorf("duel: %+v, want 0x095 for both at planet 7", ev)
	}
}

func TestBattleSummaryStarbase(t *testing.T) {
	// A starbase is a ship and a stack of its own, and a destroyed one is
	// a loss: one ship against a starbase is a duel. An Alternate Reality
	// starbase's loss sends no summary (ASSUMPTION B4).
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), testDesign(Hull{Name: "Orbital Fort", Armor: 100}, 10)}
	g := &Game{Rules: ElegyRules(), Players: make([]Player, 2), Designs: d,
		Planets: []Planet{{ID: 4, Owner: 1, HasStarbase: true}},
		Fleets:  []Fleet{{ID: 0, Owner: 0, Pos: Point{5, 5}, Stacks: []Stack{{Design: 0, Count: 1}}}}}
	sb := tok(d, 1, 1, 1)
	sb.starbase, sb.planet, sb.dead, sb.ships = true, 0, true, 0
	sh := tok(d, 0, 0, 1)
	sh.fleet = 0
	b := &battle{g: g, loc: location{pos: Point{5, 5}, planet: 0, fleets: []int{0}}, players: []int{0, 1}, involved: 2, tokens: []*token{sh, sb}}
	b.hits = []BattleHit{{Target: 1, Armor: 100}}
	ev := b.summaries(4)
	if len(ev) != 2 || BattleSummary(ev[0].Count) != SummaryDuelWonUnhit || BattleSummary(ev[1].Count) != SummaryDuelLostToUnhit {
		t.Fatalf("ship against a starbase: %+v, want 0x091 and 0x092", ev)
	}
	if ev[1].Axes[SummaryTotal] != 1 || ev[1].Axes[SummaryLost] != 1 || ev[1].Axes[SummaryDesign] != 1 {
		t.Errorf("starbase owner's axes %v, want total 1, lost 1, design 1", ev[1].Axes)
	}
	g.Players[1].Race.PRT = PRTAlternateReality
	if ev := b.summaries(4); len(ev) != 0 {
		t.Errorf("AR starbase destroyed: %+v, want no summary", ev)
	}
}
