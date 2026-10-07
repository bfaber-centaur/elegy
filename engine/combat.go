package engine

// The battle phase of a turn (COMBAT.md), and repair.

// Battle events.
const (
	EventBattle             EventKind = iota + EventRamScoopFuel + 1 // Planet (id or -1), Count = tokens
	EventFleetsMissedBattle                                          // some fleets were left out by the token cap
	EventBattleTech                                                  // Count = field gaining research
)

// battleResult is what later steps of the turn need from the battles.
type battleResult struct {
	events []Event
	fleets map[int]bool // ids of fleets that fought
	bases  map[int]bool // planet indices whose starbase fought
	hits   [][]BattleHit
}

// battles fights every battle of the turn, location by location
// (COMBAT.md "Where battles happen in the turn", BINARY-ONLY order).
func battles(g *Game, rng Rand) battleResult {
	res := battleResult{fleets: map[int]bool{}, bases: map[int]bool{}}
	gained := map[int]bool{} // players that gained tech from a battle this turn
	var prev locationHistory
	for _, loc := range g.locations() {
		sets, inv, n := g.whoFights(loc, prev)
		prev = locationHistory{any: true, battle: inv != nil, lastOwner: g.Fleets[loc.fleets[len(loc.fleets)-1]].Owner}
		if inv == nil {
			continue
		}
		b := &battle{g: g, rng: rng, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
		fought := map[int]bool{}
		res.events = append(res.events, b.setup(fought)...)
		for i := range fought {
			res.fleets[g.Fleets[i].ID] = true
		}
		for _, t := range b.tokens {
			if t.starbase {
				res.bases[t.planet] = true
			}
		}
		b.fight()
		planetID := -1
		if loc.planet >= 0 {
			planetID = g.Planets[loc.planet].ID
		}
		for _, p := range inv {
			res.events = append(res.events, Event{Kind: EventBattle, Player: p, Planet: planetID, Fleet: -1, Count: len(b.tokens)})
		}
		res.events = append(res.events, b.techAttempts(gained)...)
		b.finish()
		res.hits = append(res.hits, b.hits)
	}
	removeEmptyFleets(g)
	return res
}

// finish writes the battle back into the game: stack counts and damage,
// the starbase, and deep-space salvage.
func (b *battle) finish() {
	g := b.g
	for _, t := range b.tokens {
		if !t.starbase {
			st := &g.Fleets[t.fleet].Stacks[t.stack]
			st.Count = t.ships
			st.Damage = t.dmg
			if t.ships == 0 {
				st.Damage = Damage{}
			}
			continue
		}
		pl := &g.Planets[t.planet]
		if !t.dead {
			pl.StarbaseDamage = t.dmg.Units
			continue
		}
		// Destroyed (BINARY-ONLY): the planet has no starbase. Queued ships
		// and packets are not modelled in Elegy's queue yet.
		pl.HasStarbase, pl.StarbaseHull, pl.StarbaseDock, pl.StarbaseDamage = false, 0, false, 0
		if g.Players[t.player].Race.PRT == PRTAlternateReality {
			// An Alternate Reality planet without its starbase is left
			// uninhabited (BINARY-ONLY).
			pl.Owner, pl.Population, pl.GrowthCarry = NoOwner, 0, 0
		}
	}
	// Deep-space salvage, after the tech attempts in the draw order.
	for _, add := range b.pending {
		b.addSalvage(add)
	}
	for _, m := range b.salvage {
		g.Salvage = append(g.Salvage, Salvage{Pos: b.loc.pos, Minerals: m})
	}
}

// techAttempts makes the battle's tech-from-battle attempts (COMBAT.md
// "Tech from battle"; who attempts is BINARY-ONLY in detail).
func (b *battle) techAttempts(gained map[int]bool) []Event {
	g := b.g
	owner := NoOwner
	if b.loc.planet >= 0 {
		owner = g.Planets[b.loc.planet].Owner
	}
	locationOK := func(p int) bool { return owner == NoOwner || owner == p }
	simple := len(b.players) == 2 && len(b.tokens) == 2
	var events []Event
	involved := false
	for _, p := range b.players {
		if p == owner {
			involved = true
		}
		if !locationOK(p) {
			continue
		}
		if !simple {
			// ASSUMPTION A9 (docs/COMBAT-STATUS.md): in larger battles,
			// only when another player's ships were destroyed. COMBAT.md
			// gives this as "probably" and "not fully settled".
			other := false
			for q := range b.killed {
				if q != p {
					other = true
				}
			}
			if !other {
				continue
			}
		}
		events = append(events, b.techAttempt(p, gained)...)
	}
	if owner != NoOwner && !involved {
		events = append(events, b.techAttempt(owner, gained)...)
	}
	return events
}

// techAttempt is one player's attempt. Mystery Trader items are not
// modelled, so the 13 item tries draw but never give an item.
func (b *battle) techAttempt(p int, gained map[int]bool) []Event {
	if gained[p] {
		return nil
	}
	if b.rng.Intn(100) < 50 {
		return nil
	}
	for range 13 {
		b.rng.Intn(13)
	}
	pl := &b.g.Players[p]
	for range 6 {
		f := b.rng.Intn(NumFields)
		lvl := pl.Research.Levels[f]
		if lvl >= b.seen[f] {
			continue
		}
		// The cost of the next level at the normal speed, even under
		// slower tech.
		pl.Research.Accumulated[f] += ResearchLevelCost(lvl+1, pl.Research.levelSum(), pl.Race.ResearchCosts[f], false)
		gained[p] = true
		return []Event{{Kind: EventBattleTech, Player: p, Planet: -1, Fleet: -1, Count: f}}
	}
	return nil
}

// removeEmptyFleets drops destroyed stacks and fleets.
func removeEmptyFleets(g *Game) {
	fleets := g.Fleets[:0]
	for _, f := range g.Fleets {
		stacks := f.Stacks[:0]
		for _, s := range f.Stacks {
			if s.Count > 0 {
				stacks = append(stacks, s)
			}
		}
		f.Stacks = stacks
		if len(f.Stacks) > 0 {
			fleets = append(fleets, f)
		}
	}
	g.Fleets = fleets
}

// repair repairs fleets and starbases that did not fight (COMBAT.md
// "Repair"). moved holds the ids of fleets that moved this turn.
func repair(g *Game, moved map[int]bool, res battleResult) {
	for i := range g.Fleets {
		f := &g.Fleets[i]
		if res.fleets[f.ID] {
			continue
		}
		r := 10
		switch pi := g.planetAt(f.Pos); {
		case moved[f.ID]:
			r = 5
		case pi < 0:
			r = 10
		case g.Planets[pi].Owner != f.Owner:
			r = 15
		case !g.hasStarbase(pi) || res.bases[pi]:
			r = 25
		case g.Planets[pi].StarbaseHull < 2: // Orbital Fort: no dock
			r = 40
		default:
			r = 100
		}
		// COMBAT.md: "Interstellar Traveler doubles r" (see A8 below).
		if g.Players[f.Owner].Race.PRT == PRTInterstellarTraveler {
			r *= 2
		}
		bonus := 0
		for _, s := range f.Stacks {
			if s.Count > 0 {
				bonus = max(bonus, g.Designs[s.Design].Hull.RepairBonus)
			}
		}
		for j := range f.Stacks {
			d := &f.Stacks[j].Damage
			d.Units = max(0, d.Units-r-bonus)
		}
	}
	for i := range g.Planets {
		p := &g.Planets[i]
		if !p.HasStarbase || res.bases[i] || p.Owner == NoOwner {
			continue
		}
		// COMBAT.md: "repairs 50 units (IS 75)". The stars-elegy docs use
		// IS for Inner Strength; the fleet rule above names Interstellar
		// Traveler. ASSUMPTION A8 (docs/COMBAT-STATUS.md): each rule
		// follows its own wording until the spec resolves the pair.
		r := 50
		if g.Players[p.Owner].Race.PRT == PRTInnerStrength {
			r = 75
		}
		p.StarbaseDamage = max(0, p.StarbaseDamage-r)
	}
}

func (g *Game) planetAt(pos Point) int {
	for i := range g.Planets {
		if g.Planets[i].Pos == pos {
			return i
		}
	}
	return -1
}

func (g *Game) hasStarbase(pi int) bool {
	p := &g.Planets[pi]
	return p.HasStarbase || p.StarbaseHull > 0
}
