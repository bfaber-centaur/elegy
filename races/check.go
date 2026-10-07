package races

import "github.com/bfaber-centaur/elegy/engine"

// Leftover is the leftover a legal race takes into a game, min(50,
// points) (RACES.md "Advantage points", CONFIRMED RD-4 boundaries 0, 50,
// 51).
func Leftover(points int) int { return max(0, min(50, points)) }

// EffectiveSpend is the spend a game applies: 5 and 6 act as 0, surface
// minerals (RACES.md "At game creation", CONFIRMED RD-4).
func EffectiveSpend(spend int) int {
	if spend < 0 || spend > 4 {
		return 0
	}
	return spend
}

// CreationResult is what game creation makes of one player's race.
type CreationResult struct {
	Race   Design
	Points int
	// Replaced is set when an illegal human race became the default
	// race; the caller gives it a random computer-player name.
	Replaced bool
}

// AtCreation applies the game-creation rules to one player's race
// (RACES.md "At game creation", CONFIRMED RD-4): repairs (a repaired race
// is kept, marked tampered); an illegal human race (points < 0) becomes
// the default race, marked tampered; computer races are not checked.
// Random races are generated separately (Generate).
//
// The original also refuses race files with a bad checksum; Elegy has no
// race files, so there is nothing to refuse.
func AtCreation(d Design, computer bool) CreationResult {
	if computer {
		return CreationResult{Race: d, Points: Points(d)}
	}
	r, changed := Repaired(d)
	if changed {
		r.Tampered = true
	}
	p := rawPoints(r) / 3
	if p < 0 {
		def := Default()
		def.Tampered = true
		return CreationResult{Race: def, Points: Points(def), Replaced: true}
	}
	return CreationResult{Race: r, Points: p}
}

// YearResult is what the yearly check did to a race.
type YearResult struct {
	// Clamped is set when a silent clamp changed a setting.
	Clamped bool
	// Punished is set when the race was penalised: the player gets the
	// "Your race definition has been tampered with" message and every
	// other player "Hacked race discovered" (BINARY-ONLY for the latter).
	Punished bool
}

// penaltyTarget is the points the penalty raises a race to (RACES.md "In
// a running game" step 3).
const penaltyTarget = 500

// YearlyCheck is the check every player's race gets each year before
// fleets move (RACES.md "In a running game", CONFIRMED RD-P1..RD-P10).
// researchBudget is the player's research share in percent.
//
//  1. Silent clamps: every setting to its range, no message or flag
//     (RD-P5, P6, P7); a research share outside 0–100 becomes 15
//     (BINARY-ONLY).
//  2. A human race is punished when its points are negative or the
//     scoring repair (habitat, growth 0) changes something (RD-P1..P4,
//     P8, P10); exactly 0 is left alone (RD-P9). A computer race gets
//     the scoring repair, which sets the tampered flag, but no penalty
//     and no message (BINARY-ONLY).
//  3. Penalty: tampered; colonists per resource +100 until 500 points or
//     2500; then growth −1 until 500 points or growth 1; then research
//     fields one at a time to "costs 75% extra" in field order until more
//     than 499 points (BINARY-ONLY).
func YearlyCheck(d *Design, researchBudget *int, computer bool) YearResult {
	var res YearResult
	res.Clamped = clampSettings(d)
	if researchBudget != nil && (*researchBudget < 0 || *researchBudget > 100) {
		*researchBudget = 15
		res.Clamped = true
	}
	repaired := scoringRepair(d)
	if computer {
		if repaired {
			d.Tampered = true
		}
		return res
	}
	if !repaired && rawPoints(*d)/3 >= 0 {
		return res
	}
	res.Punished = true
	d.Tampered = true
	points := func() int { return rawPoints(*d) / 3 }
	for points() < penaltyTarget && d.Race.ColonistsPerResource < MaxColonists {
		d.Race.ColonistsPerResource += 100
	}
	for points() < penaltyTarget && d.Race.GrowthRate > MinGrowth {
		d.Race.GrowthRate--
	}
	for f := range engine.NumFields {
		if points() > penaltyTarget-1 {
			break
		}
		d.Race.ResearchCosts[f] = engine.ResearchExpensive
	}
	return res
}
