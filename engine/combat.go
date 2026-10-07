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
//
// gained holds the players that gained tech this turn, from a battle or a
// capture (TAKEOVER.md "Capture").
func battles(g *Game, rng Rand, gained map[int]bool) battleResult {
	res := battleResult{fleets: map[int]bool{}, bases: map[int]bool{}}
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
		// Destroyed: the planet has no starbase. Queued ships
		// and packets are not modelled in Elegy's queue yet.
		pl.HasStarbase, pl.StarbaseHull, pl.StarbaseDock, pl.StarbaseDamage = false, 0, false, 0
		if g.Players[t.player].Race.PRT == PRTAlternateReality {
			// An Alternate Reality planet without its starbase is left
			// uninhabited (CONFIRMED, CB-041), and emptied as TAKEOVER.md
			// "Capture" lists: the growth carry stays.
			g.emptyPlanet(t.planet)
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

// legacyObserverTechMask reproduces the original's LEGACY BUG in the
// tech attempt of players outside the battle (COMBAT.md "Tech from
// battle", CONFIRMED, CB-031-obs, CB-037): it tests the player's number against the
// observer bitmask instead of the player's bit. Set it to false to test
// the bit, as the game means to.
const legacyObserverTechMask = true

// techAttempts makes the battle's tech-from-battle attempts (COMBAT.md
// "Tech from battle"): every player of the game is considered once, in
// player-number order. A wiped-out participant of a two-player battle makes
// no attempt (CONFIRMED, CB-029), nor does any participant when an
// Alternate Reality starbase was destroyed (CONFIRMED, CB-041); with n ≠ 2
// every participant attempts (CONFIRMED, CB-031-n3).
func (b *battle) techAttempts(gained map[int]bool) []Event {
	g := b.g
	owner, sbOwnerless := NoOwner, false
	if b.loc.planet >= 0 {
		pl := g.Planets[b.loc.planet]
		owner, sbOwnerless = pl.Owner, !pl.HasStarbase
	}
	inBattle := map[int]bool{}
	for _, p := range b.players {
		inBattle[p] = true
	}
	// What each participant still has, and whether an Alternate Reality
	// starbase was destroyed.
	has := map[int]bool{}
	arStarbaseLost := false
	for _, t := range b.tokens {
		switch {
		case t.starbase && t.dead:
			if g.Players[t.player].Race.PRT == PRTAlternateReality {
				arStarbaseLost = true
			}
		case t.starbase || t.ships > 0:
			has[t.player] = true
		}
	}
	// Observers: players present but not in the battle, and the owner of
	// a planet there without a starbase, even when that owner is a
	// participant (CONFIRMED, CB-037-owner): its bit counts for other players, though
	// a participant never gets the observer attempt itself.
	observers := 0
	present := map[int]bool{}
	for _, i := range b.loc.fleets {
		o := g.Fleets[i].Owner
		present[o] = true
		if !inBattle[o] {
			observers |= 1 << o
		}
	}
	if owner != NoOwner && sbOwnerless {
		observers |= 1 << owner
	}
	var events []Event
	for p := range g.Players {
		attempt := false
		switch {
		case inBattle[p]:
			attempt = (owner == NoOwner || owner == p) && !arStarbaseLost && (b.involved != 2 || has[p])
		case owner == p:
			attempt = true
		default:
			mask := 1 << p
			if legacyObserverTechMask {
				mask = p
			}
			attempt = mask&observers != 0 && present[p]
		}
		if attempt {
			events = append(events, techAttempt(g, b.rng, p, b.seen, gained)...)
		}
	}
	return events
}

// techAttempt is one player's tech attempt against the seen levels
// (COMBAT.md "Tech from battle" steps 1–5; also a capture's attempt,
// TAKEOVER.md "Capture"). Mystery Trader items are not modelled, so the
// 13 item tries draw but never give an item.
func techAttempt(g *Game, rng Rand, p int, seen [NumFields]int, gained map[int]bool) []Event {
	if gained[p] {
		return nil
	}
	if rng.Intn(100) < 50 {
		return nil
	}
	for range 13 {
		rng.Intn(13)
	}
	pl := &g.Players[p]
	for range 6 {
		f := rng.Intn(NumFields)
		lvl := pl.Research.Levels[f]
		if lvl >= seen[f] {
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
		// Inner Strength doubles r, not f (CONFIRMED, CB-024).
		if g.Players[f.Owner].Race.PRT == PRTInnerStrength {
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
		// 50 units, or 75 for Inner Strength (CONFIRMED, CB-024).
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
