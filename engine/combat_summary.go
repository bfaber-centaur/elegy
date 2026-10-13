package engine

// EventBattleSummary: Player's summary of a battle at Planet (id, or -1 in
// deep space). Count is the BattleSummary; Axes holds the counts the
// summary texts name (summaryAxes). One per battle player, in player order,
// after the battle's EventBattle (stars-elegy MESSAGES.md "Battles and
// bombing", 0x07e, 0x091–0x0a8 and 0x113–0x116).
const EventBattleSummary EventKind = EventPlanetNotDegraded + 1

// BattleSummary is which battle summary a player is sent (stars-elegy
// MESSAGES.md). The constants follow the message table: SummaryDuel* are
// 0x091–0x095, SummaryStack* 0x096–0x09a, SummaryWonNoLoss to
// SummaryLoneLostSomeKills 0x09b–0x0a3, SummaryMulti* 0x0a4–0x0a8,
// SummaryMultiMixed 0x07e and SummaryLoneWonAll to SummaryLoneEnemySurvived
// 0x113–0x116.
type BattleSummary int

const (
	SummaryNone BattleSummary = iota

	// Exactly 2 ships at the location (a starbase counts as a ship).
	SummaryDuelWonUnhit      // yours survived unhit, the enemy's died
	SummaryDuelLostToUnhit   // yours died, the enemy's was not hit
	SummaryDuelWonHit        // yours survived after being hit, the enemy's died
	SummaryDuelLostToHit     // yours died, the enemy's was hit
	SummaryDuelBothSurvived  // both survived
	SummaryStackWonUnhit     // exactly 2 stacks (summaries): as the Duel ones
	SummaryStackLostToUnhit  //
	SummaryStackWonHit       //
	SummaryStackLostToHit    //
	SummaryStackBothSurvived //

	// Other 2-player battles.
	SummaryWonNoLoss          // the enemy lost everything, you nothing
	SummaryLostAllNoKills     // you lost everything, the enemy nothing
	SummaryWonWithLoss        // the enemy lost everything, you some (or all)
	SummaryLostAllSomeKills   // you lost everything, the enemy some
	SummaryBothSurvived       // both sides have survivors
	SummaryKilledLoneNoLoss   // the enemy's single ship died, you lost nothing
	SummaryLoneLostNoKills    // your single ship died, the enemy lost nothing
	SummaryKilledLoneWithLoss // the enemy's single ship died, you lost some
	SummaryLoneLostSomeKills  // your single ship died, the enemy lost some

	// Battles of 3 or more players; "others" are the other players together.
	SummaryMultiWonNoLoss     // the others lost everything, you nothing
	SummaryMultiWonWithLoss   // the others lost everything, you some (or all)
	SummaryMultiLostNoKills   // you lost everything, the others nothing
	SummaryMultiLostSomeKills // you lost everything, the others some
	SummaryMultiOneSideNoLoss // one side lost nothing, neither lost all
	SummaryMultiMixed         // both sides lost some, neither all

	// Other 2-player battles with a single ship on one side.
	SummaryLoneWonAll        // your single ship survived, the enemy's force died
	SummaryLoneEnemyWonAll   // your force died, the enemy's single ship lost nothing
	SummaryLoneSurvived      // your single ship survived, the enemy not wiped
	SummaryLoneEnemySurvived // your force survived, the enemy's single ship too
)

// The Axes of an EventBattleSummary, by index.
const (
	SummaryOther       = iota // the other player of a 2-player battle; the number of players otherwise
	SummaryTotal              // your ships (+1 for your starbase) at the location before the battle
	SummaryLost               // your ships (+1 for your starbase) destroyed
	SummaryOtherTotal         // the same for the other player, or for the others together
	SummaryOtherLost          //
	SummaryDesign             // your token's design (one-ship and one-stack battles), else -1
	SummaryOtherDesign        // the other token's design, else -1
	summaryAxes
)

// summaryCounts is one side's forces in a battle summary.
type summaryCounts struct{ total, lost int }

func (c summaryCounts) wiped() bool { return c.lost >= c.total }

// battleSummary picks a player's summary (stars-elegy MESSAGES.md "Battles
// and bombing" and its notes "Battle counts", "Exact-2 shortcuts",
// "Precedence" and "0x07e vs 0x0a8"). players is the number of players in
// the battle; ships and stacks count the whole location (a starbase is a
// ship and a stack; summaries says what a stack is); hit and otherHit say
// whether the two sides of a one-ship or one-stack battle took damage.
//
// ASSUMPTION B1: "hit" is taking shield or armor damage in a hit record.
// ASSUMPTION B2: a one-ship side that loses its ship while the enemy is
// wiped too gets SummaryWonWithLoss, as the "Precedence" note gives for
// other 2-player battles; a side of several ships that loses some (not
// all) while a single enemy ship dies gets SummaryKilledLoneWithLoss.
// ASSUMPTION B3: a 2-player battle is one whose player list P has 2
// players.
func battleSummary(players, ships, stacks int, me, other summaryCounts, hit, otherHit bool) BattleSummary {
	if players == 2 && (ships == 2 || stacks == 2) {
		base := SummaryDuelWonUnhit
		if ships != 2 {
			base = SummaryStackWonUnhit
		}
		switch {
		case me.wiped() && !otherHit:
			return base + 1
		case me.wiped():
			return base + 3
		case other.wiped() && !hit:
			return base
		case other.wiped():
			return base + 2
		}
		return base + 4
	}
	if players != 2 {
		switch {
		case other.wiped() && me.lost == 0:
			return SummaryMultiWonNoLoss
		case other.wiped():
			return SummaryMultiWonWithLoss
		case me.wiped() && other.lost == 0:
			return SummaryMultiLostNoKills
		case me.wiped():
			return SummaryMultiLostSomeKills
		case me.lost == 0 || other.lost == 0:
			return SummaryMultiOneSideNoLoss
		}
		return SummaryMultiMixed
	}
	meOne, otherOne := me.total == 1, other.total == 1
	switch {
	case other.wiped():
		switch {
		case me.lost == 0 && otherOne:
			return SummaryKilledLoneNoLoss
		case me.lost == 0 && meOne:
			return SummaryLoneWonAll
		case me.lost == 0:
			return SummaryWonNoLoss
		case otherOne && !me.wiped():
			return SummaryKilledLoneWithLoss
		}
		return SummaryWonWithLoss
	case me.wiped():
		switch {
		case meOne && other.lost == 0:
			return SummaryLoneLostNoKills
		case meOne:
			return SummaryLoneLostSomeKills
		case otherOne:
			return SummaryLoneEnemyWonAll
		case other.lost == 0:
			return SummaryLostAllNoKills
		}
		return SummaryLostAllSomeKills
	case meOne:
		return SummaryLoneSurvived
	case otherOne:
		return SummaryLoneEnemySurvived
	}
	return SummaryBothSurvived
}

// summaries makes the battle's EventBattleSummary events, before finish
// writes the losses back: the fleets still hold their stacks' counts from
// before the battle.
//
// A "stack" in the one-stack texts is one player's ships of one design at
// the location, whatever fleets hold them: CB-039 (140 one-ship fleets a
// side, one design each) sent 0x09a and CB-048 (80 ships in one fleet
// against 30 one-ship fleets, one design each) sent 0x098 and 0x099
// (stars-elegy cb5/cb039, cb7/cb048; MESSAGES.md's "exactly 2 stacks"
// read as 2 fleet stacks would give neither). A starbase is a stack of its
// own. A group is hit when any of its tokens is (ASSUMPTION B1).
//
// ASSUMPTION B4: when an Alternate Reality starbase is destroyed nobody
// gets a summary (MESSAGES.md: its owner gets 0x08d or 0x08e and every
// other participant 0x144, which Elegy does not send).
func (b *battle) summaries(planetID int) []Event {
	g := b.g
	type group struct{ player, design int }
	total := map[int]int{}
	groups := map[group]bool{}
	ships := 0
	for _, i := range b.loc.fleets {
		for _, s := range g.Fleets[i].Stacks {
			if s.Count > 0 {
				total[g.Fleets[i].Owner] += s.Count
				ships += s.Count
				groups[group{g.Fleets[i].Owner, s.Design}] = true
			}
		}
	}
	if b.loc.planet >= 0 && g.Planets[b.loc.planet].HasStarbase {
		total[g.Planets[b.loc.planet].Owner]++
		ships++
		groups[group{g.Planets[b.loc.planet].Owner, -1}] = true
	}
	hitToken := map[int]bool{}
	for _, h := range b.hits {
		if h.Shield > 0 || h.Armor > 0 {
			hitToken[h.Target] = true
		}
	}
	lost := map[int]int{}
	hit := map[int]bool{}
	design := map[int]int{}
	for i, t := range b.tokens {
		if t.starbase {
			if t.dead {
				if g.Players[t.player].Race.PRT == PRTAlternateReality {
					return nil
				}
				lost[t.player]++
			}
		} else {
			lost[t.player] += g.Fleets[t.fleet].Stacks[t.stack].Count - t.ships
		}
		hit[t.player] = hit[t.player] || hitToken[i]
		design[t.player] = t.design
	}
	n := len(b.players)
	var events []Event
	for _, p := range b.players {
		me := summaryCounts{total[p], lost[p]}
		var other summaryCounts
		axes := make([]int, summaryAxes)
		axes[SummaryOther] = n
		q := -1
		for _, r := range b.players {
			if r != p {
				other.total += total[r]
				other.lost += lost[r]
				q = r
			}
		}
		axes[SummaryTotal], axes[SummaryLost] = me.total, me.lost
		axes[SummaryOtherTotal], axes[SummaryOtherLost] = other.total, other.lost
		axes[SummaryDesign], axes[SummaryOtherDesign] = -1, -1
		if n == 2 {
			axes[SummaryOther] = q
			if len(groups) == 2 {
				axes[SummaryDesign], axes[SummaryOtherDesign] = design[p], design[q]
			}
		}
		kind := battleSummary(n, ships, len(groups), me, other, hit[p], hit[q])
		events = append(events, Event{Kind: EventBattleSummary, Player: p, Planet: planetID, Fleet: -1, Count: int(kind), Axes: axes})
	}
	return events
}
