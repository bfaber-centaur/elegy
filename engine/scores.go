package engine

import "sort"

// Scores and victory conditions (stars-elegy KERNEL.md "Scores and victory
// conditions"), computed once a year after every other phase, when the
// year has advanced.

// Victory holds the game's victory settings in the new-game dialog's
// encoding v (KERNEL.md "Victory conditions"; defaults in brackets):
// planets (v+4)·5 % [8 → 60], tech level v+8 [14 → 22] in v+2 fields
// [2 → 4], score (v+1)·1000 [10 → 11000], lead (v+2)·10 % [8 → 100],
// resources (v+1)·10 thousand [9 → 100], capital ships (v+1)·10 [9 →
// 100], highest score after (v+3)·10 years [7 → 100], Needed conditions
// [1] and the minimum years (v+3)·10 [0 → 30]. Enabled switches the first
// seven, in that order.
type Victory struct {
	Planets, TechLevel, TechFields, Score, Lead, Resources, Capital, Highest int
	Needed, MinYears                                                         int
	Enabled                                                                  [NumVictory]bool
}

// Victory conditions, in flag order.
const (
	VictoryPlanets = iota
	VictoryTech
	VictoryScore
	VictoryLead
	VictoryResources
	VictoryCapital
	VictoryHighest
	NumVictory
)

// ScoreRecord is a player's yearly score record (KERNEL.md "Yearly score
// record").
type ScoreRecord struct {
	Player    int
	Score     int
	Resources int // R, from population after growth
	Planets   int
	Starbases int
	Unarmed   int // U
	Escorts   int // E
	Capital   int // C
	TechSum   int
	Rank      int
	// Flags is the player number in the low 5 bits, 0x20, and 0x40 << c
	// for each victory condition c the player meets this year, enabled or
	// not.
	Flags int
}

// Score and victory messages. Player is the recipient.
const (
	EventPlayerDied EventKind = iota + EventNewMinerals + 1 // Count = the dead player
	EventGameWon                                            // Count = the number of winners
	EventGameLost
)

// techScore is a field's score at level l (CONFIRMED, KX-003).
func techScore(l int) int {
	switch {
	case l <= 3:
		return l
	case l <= 6:
		return 2*l - 3
	case l <= 9:
		return 3 * (l - 3)
	}
	return 4*l - 18
}

// shipPower is a design's power for the ship classes (KERNEL.md "Score";
// class boundaries CONFIRMED, capacitors and sappers BINARY-ONLY). The
// speed code is the design's own: empty mass, no cargo, no War Monger
// bonus (CONFIRMED, OT-6). Each slot's term is truncated before summing
// (BINARY-ONLY).
func shipPower(d Design) int {
	beam, torp, bomb := 0, 0, 0
	factor := 1000
	for _, s := range d.Slots {
		p := s.Part
		switch p.Kind {
		case PartBeam:
			v := (p.Range + 3) * p.Damage * s.Count / 4
			if p.Sapper {
				v /= 3
			}
			beam += v
		case PartTorpedo:
			torp += (p.Range - 2) * p.Damage * s.Count / 2
		case PartBomb:
			bomb += (p.KillRate + p.InstallKill) * s.Count * 2
		}
		for range s.Count {
			if p.Capacitor > 0 {
				factor = factor * (100 + p.Capacitor) / 100
			}
		}
	}
	if factor != 1000 {
		beam = beam * min(255, factor/10) / 100
	}
	beam += beam * (speedCode(d, Race{}, d.Mass, false) - 4) / 10
	return beam + torp + bomb
}

// shipsScore is the ships term for N planets and U, E, C ships.
func shipsScore(n, u, e, c int) int {
	s := min(n, u)/2 + 2*min(n, e)
	if c > 0 {
		s += 8 * n * c / (n + c)
	}
	return s
}

// scores computes every player's record for the year (KERNEL.md "Score",
// CONFIRMED KX-003; rank and flags CONFIRMED S1, S2, S3L).
// The record's starbase count and the score's starbase term both count
// the starbases with a dock, so not Orbital Forts (CONFIRMED, KX-003).
func (g *Game) scores() []ScoreRecord {
	recs := make([]ScoreRecord, len(g.Players))
	planetScore := make([]int, len(g.Players))
	for i := range recs {
		recs[i].Player = i
	}
	for i := range g.Planets {
		p := &g.Planets[i]
		if p.Owner == NoOwner {
			continue
		}
		r := &recs[p.Owner]
		r.Planets++
		planetScore[p.Owner] += min(6, ceilDiv(p.Population, 1000))
		if p.StarbaseDock {
			r.Starbases++
		}
		r.Resources += NewColony(p, &g.Players[p.Owner]).Resources(p.Population, p.Factories)
	}
	for _, f := range g.Fleets {
		if f.Owner < 0 || f.Owner >= len(g.Players) {
			continue
		}
		r := &recs[f.Owner]
		for _, s := range f.Stacks {
			switch pw := shipPower(g.Designs[s.Design]); {
			case pw == 0:
				r.Unarmed += s.Count
			case pw < 2000:
				r.Escorts += s.Count
			default:
				r.Capital += s.Count
			}
		}
	}
	for i := range recs {
		r := &recs[i]
		tech := 0
		for _, l := range g.Players[i].Research.Levels {
			tech += techScore(l)
			r.TechSum += l
		}
		r.Score = planetScore[i] + 3*r.Starbases + r.Resources/30 + tech + shipsScore(r.Planets, r.Unarmed, r.Escorts, r.Capital)
	}
	for i := range recs {
		recs[i].Rank = 1
		for j := range recs {
			if recs[j].Score > recs[i].Score {
				recs[i].Rank++
			}
		}
	}
	g.victoryFlags(recs)
	return recs
}

// victoryFlags sets each record's flag word (KERNEL.md "Victory
// conditions": planets, tech, lead and capital ships CONFIRMED S1; score,
// resources and highest score BINARY-ONLY).
//
// The planets test rounds halves up, and a tie for the top score flags
// nobody for the lead (BINARY-ONLY).
func (g *Game) victoryFlags(recs []ScoreRecord) {
	v := g.Victory
	sorted := make([]int, len(recs))
	firsts := 0
	for i, r := range recs {
		sorted[i] = r.Score
		if r.Rank == 1 {
			firsts++
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sorted)))
	second := 0
	if len(sorted) > 1 {
		second = sorted[1]
	}
	for i := range recs {
		r := &recs[i]
		met := [NumVictory]bool{
			VictoryPlanets:   r.Planets >= (len(g.Planets)*(v.Planets+4)*5+50)/100,
			VictoryScore:     r.Score >= (v.Score+1)*1000,
			VictoryLead:      r.Rank == 1 && firsts == 1 && second*(100+(v.Lead+2)*10)/100 <= r.Score,
			VictoryResources: r.Resources/1000 >= (v.Resources+1)*10,
			VictoryCapital:   r.Capital >= (v.Capital+1)*10,
			VictoryHighest:   g.yearIndex() >= (v.Highest+3)*10 && firsts == 1 && r.Rank == 1,
		}
		fields := 0
		for _, l := range g.Players[i].Research.Levels {
			if l >= v.TechLevel+8 {
				fields++
			}
		}
		met[VictoryTech] = fields >= v.TechFields+2
		r.Flags = i&0x1f | 0x20
		for c, ok := range met {
			if ok {
				r.Flags |= 0x40 << c
			}
		}
	}
}

// decide applies KERNEL.md's "Deciding the game" (BINARY-ONLY) once the
// year's records exist: new deaths, then a survivor or the winners. The
// game is decided again every year the conditions hold, with the messages
// each time, and a year without a winner leaves it undecided. Dead players
// get the loss message too. The needed count is the raw setting capped at
// the number of enabled conditions; 0 means nobody wins by conditions.
func (g *Game) decide(recs []ScoreRecord) []Event {
	g.Decided = false
	if len(g.Players) <= 1 {
		return nil
	}
	var events []Event
	ships := make([]int, len(g.Players))
	for _, f := range g.Fleets {
		if f.Owner >= 0 && f.Owner < len(ships) {
			ships[f.Owner] += fleetShips(&f)
		}
	}
	for i := range g.Players {
		if g.Players[i].Dead || recs[i].Planets > 0 || ships[i] > 0 {
			continue
		}
		g.Players[i].Dead = true
		for j := range g.Players {
			if j != i && !g.Players[j].Dead {
				events = append(events, Event{Kind: EventPlayerDied, Player: j, Planet: -1, Fleet: -1, Count: i})
			}
		}
	}
	var alive, winners []int
	for i := range g.Players {
		if !g.Players[i].Dead {
			alive = append(alive, i)
		}
	}
	switch {
	case len(alive) == 1:
		winners = alive
	case g.yearIndex() >= (g.Victory.MinYears+3)*10:
		enabled := 0
		for _, on := range g.Victory.Enabled {
			if on {
				enabled++
			}
		}
		needed := min(g.Victory.Needed, enabled)
		if needed <= 0 {
			break
		}
		for i, r := range recs {
			n := 0
			for c := range NumVictory {
				if g.Victory.Enabled[c] && r.Flags&(0x40<<c) != 0 {
					n++
				}
			}
			if n >= needed && !g.Players[i].Dead {
				winners = append(winners, i)
			}
		}
	}
	if len(winners) == 0 {
		return events
	}
	g.Decided = true
	won := map[int]bool{}
	for _, w := range winners {
		won[w] = true
	}
	for i := range g.Players {
		kind := EventGameLost
		if won[i] {
			kind = EventGameWon
		}
		events = append(events, Event{Kind: kind, Player: i, Planet: -1, Fleet: -1, Count: len(winners)})
	}
	return events
}

// visibleScores is the score records player v's file holds (KERNEL.md
// "Game options during a turn", public scores CONFIRMED KX-004 E0, E1;
// decided game and dead players BINARY-ONLY): its own, and another's if
// the game is decided, that player is dead, or public scores are on from
// year index 20.
func (g *Game) visibleScores(v int, recs []ScoreRecord) []ScoreRecord {
	var out []ScoreRecord
	for _, r := range recs {
		if r.Player == v || g.Decided || g.Players[r.Player].Dead || g.PublicScores && g.yearIndex() >= 20 {
			out = append(out, r)
		}
	}
	return out
}
