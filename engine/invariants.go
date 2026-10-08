package engine

import (
	"fmt"
	"strings"
)

// InvariantError lists what a generated year broke (CheckYear).
type InvariantError struct {
	Year     int
	Problems []string
}

func (e *InvariantError) Error() string {
	return fmt.Sprintf("year %d: %d invariant problems: %s", e.Year, len(e.Problems), strings.Join(e.Problems, "; "))
}

// CheckYear checks the structural invariants of one generated year, from
// the game before it (prev) to GenerateTurn's result r, and returns an
// *InvariantError naming each problem, or nil. A game loop can call it
// after each year; a long test run checks it every year. Each check cites
// the stars-elegy rule it follows from or carries a label; an ELEGY CHOICE
// check is a data-model invariant and says what it protects.
//
// Year and planets:
//   - The year advances by one (KERNEL.md "8. Year end, scores and
//     files", step 1).
//   - Players and planets keep their count, and planets their ids and
//     positions. ELEGY CHOICE: the planet list is fixed when the universe
//     is generated (UNIVERSE.md "Planets") and no specified rule adds,
//     removes or moves a planet; it protects the ids and indexes orders,
//     views and saves refer to.
//   - A planet's owner is a player or NoOwner, and a starbase's design
//     exists. ELEGY CHOICE: it protects index lookups.
//   - An unowned planet has no population, defenses, starbase or queue
//     (TAKEOVER.md "Capture: what a planet keeps", CONFIRMED).
//   - Population is never negative. An owned planet may end the year
//     with 0 colonists: a load after movement can take every colonist,
//     and the planet stays owned until the next year's growth
//     (TAKEOVER.md "Unload and load amounts", CONFIRMED TK-201 G; the
//     after-movement loads come after growth, "Where each task happens in
//     the year"). So CheckYear does not require population on an owned
//     planet. ELEGY CHOICE for the sign: it protects arithmetic on counts.
//   - Mines, factories, defenses and surface minerals are not negative.
//     ELEGY CHOICE: counts and amounts, protecting arithmetic on them.
//   - Environment and original environment are each 1..99 (LIMITS.md
//     "Environment", BINARY-ONLY; KERNEL.md "Comet strike" and "Planetary
//     climate change" clamp their shifts to 1..99).
//
// Fleets:
//   - A fleet's owner is a player and fleet ids are unique. ELEGY CHOICE:
//     it protects index lookups.
//   - Fleet numbers are 1 and up and unique per owner. A new fleet takes
//     its owner's lowest unused number (PRODUCTION-LAUNCH.md "The new
//     fleet", CONFIRMED SL-02; Elegy stores the number the client shows,
//     #1 first), and fleet order is by owner, then number (KERNEL.md "Turn
//     order"). ELEGY CHOICE for uniqueness: it protects orders and fleet
//     order, which name a fleet by owner and number.
//   - Every fleet has ships of ship designs that exist; fuel and cargo are
//     not negative and fit the fleet's tanks and holds. ELEGY CHOICE: it
//     protects the fleet arithmetic (FleetShips, loads, movement).
//   - No player owns more than 512 fleets (PRODUCTION-LAUNCH.md "The
//     512-fleet limit", CONFIRMED SL-08..SL-10; maxFleets).
//
// Research:
//   - Levels are 0..26 (LIMITS.md "Tech level", CONFIRMED KX-002 R6).
//   - Levels never fall. ASSUMPTION: no rule in stars-elegy lowers a
//     level; research only raises one (KERNEL.md "Allocation").
//
// Views:
//   - Each player's view holds all of their own planets at the own-planet
//     level and no other planet at that level (SCANNING.md "When
//     knowledge is computed": a player knows everything about its own
//     planets; "What a planet report contains").
//   - Every planet report and fleet sighting names an object that exists,
//     at its real position, and a sighting shows the fleet's real owner
//     (SCANNING.md "Disclosure: what each sighting reveals", Fleets).
//   - A sighting is never one of the viewer's own fleets. ELEGY CHOICE:
//     the player knows its own fleets from its own state (SCANNING.md "When
//     knowledge is computed"); it protects views from listing a fleet
//     twice.
func CheckYear(prev Game, r TurnResult) error {
	g := r.Game
	var bad []string
	add := func(format string, a ...any) { bad = append(bad, fmt.Sprintf(format, a...)) }

	if g.Year != prev.Year+1 {
		add("year %d after %d", g.Year, prev.Year)
	}
	if len(g.Players) != len(prev.Players) {
		add("%d players after %d", len(g.Players), len(prev.Players))
	}
	if len(g.Planets) != len(prev.Planets) {
		add("%d planets after %d", len(g.Planets), len(prev.Planets))
	}
	player := func(i int) bool { return i >= 0 && i < len(g.Players) }

	planetByID := map[int]int{}
	for i, p := range g.Planets {
		if i < len(prev.Planets) && (p.ID != prev.Planets[i].ID || p.Pos != prev.Planets[i].Pos) {
			add("planet %d at %v was planet %d at %v", p.ID, p.Pos, prev.Planets[i].ID, prev.Planets[i].Pos)
		}
		if _, dup := planetByID[p.ID]; dup {
			add("planet id %d twice", p.ID)
		}
		planetByID[p.ID] = i
		switch {
		case p.Owner == NoOwner:
			if p.Population != 0 || p.Defenses != 0 || p.HasStarbase || p.HasQueue {
				add("unowned planet %d: population %d defenses %d starbase %v queue %v", p.ID, p.Population, p.Defenses, p.HasStarbase, p.HasQueue)
			}
		case !player(p.Owner):
			add("planet %d owner %d", p.ID, p.Owner)
		}
		if p.Population < 0 {
			add("planet %d: population %d", p.ID, p.Population)
		}
		if p.Mines < 0 || p.Factories < 0 || p.Defenses < 0 {
			add("planet %d: mines %d factories %d defenses %d", p.ID, p.Mines, p.Factories, p.Defenses)
		}
		for m, v := range p.Surface {
			if v < 0 {
				add("planet %d: surface mineral %d is %d", p.ID, m, v)
			}
		}
		for a := range 3 {
			if p.Env[a] < 1 || p.Env[a] > 99 || p.OrigEnv[a] < 1 || p.OrigEnv[a] > 99 {
				add("planet %d: environment %v original %v", p.ID, p.Env, p.OrigEnv)
				break
			}
		}
		if p.HasStarbase && (p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs)) {
			add("planet %d: starbase design %d", p.ID, p.StarbaseDesign)
		}
	}

	fleetByID := map[int]int{}
	numbers := map[[2]int]bool{}
	perPlayer := map[int]int{}
	for i, f := range g.Fleets {
		if _, dup := fleetByID[f.ID]; dup {
			add("fleet id %d twice", f.ID)
		}
		fleetByID[f.ID] = i
		if !player(f.Owner) {
			add("fleet %d owner %d", f.ID, f.Owner)
			continue
		}
		perPlayer[f.Owner]++
		key := [2]int{f.Owner, f.Number}
		if f.Number < 1 || numbers[key] {
			add("fleet %d: number %d of player %d (duplicate or below 1)", f.ID, f.Number, f.Owner)
		}
		numbers[key] = true
		ships, holds, tanks := 0, 0, 0
		for _, s := range f.Stacks {
			if s.Design < 0 || s.Design >= len(g.Designs) || g.Designs[s.Design].Hull.Starbase {
				add("fleet %d: design %d", f.ID, s.Design)
				continue
			}
			if s.Count <= 0 {
				add("fleet %d: stack of %d", f.ID, s.Count)
			}
			ships += s.Count
			holds += s.Count * g.Designs[s.Design].CargoCapacity
			tanks += s.Count * g.Designs[s.Design].FuelCapacity
		}
		if ships <= 0 {
			add("fleet %d has no ships", f.ID)
		}
		if f.Fuel < 0 || f.Fuel > tanks {
			add("fleet %d: fuel %d of %d", f.ID, f.Fuel, tanks)
		}
		for m, v := range f.Cargo.Minerals {
			if v < 0 {
				add("fleet %d: cargo mineral %d is %d", f.ID, m, v)
			}
		}
		if f.Cargo.Colonists < 0 || f.Cargo.mass() > holds {
			add("fleet %d: cargo %+v in holds of %d", f.ID, f.Cargo, holds)
		}
	}
	for pl, n := range perPlayer {
		if n > maxFleets {
			add("player %d has %d fleets", pl, n)
		}
	}

	for i, pl := range g.Players {
		for k, lv := range pl.Research.Levels {
			if lv < 0 || lv > MaxTechLevel {
				add("player %d: field %d level %d", i, k, lv)
			}
			if i < len(prev.Players) && lv < prev.Players[i].Research.Levels[k] {
				add("player %d: field %d fell from %d to %d", i, k, prev.Players[i].Research.Levels[k], lv)
			}
		}
	}

	if len(r.Views) != len(g.Players) {
		add("%d views for %d players", len(r.Views), len(g.Players))
	}
	for v, view := range r.Views {
		if view.Player != v {
			add("view %d is player %d's", v, view.Player)
		}
		own := map[int]bool{}
		for _, rep := range view.Planets {
			pi, ok := planetByID[rep.Planet]
			if !ok {
				add("player %d's view: planet %d does not exist", v, rep.Planet)
				continue
			}
			mine := g.Planets[pi].Owner == v
			if (rep.Level == ReportOwn) != mine {
				add("player %d's view: planet %d at level %v, owner %d", v, rep.Planet, rep.Level, g.Planets[pi].Owner)
			}
			if rep.Pos != g.Planets[pi].Pos {
				add("player %d's view: planet %d at %v, really %v", v, rep.Planet, rep.Pos, g.Planets[pi].Pos)
			}
			if mine {
				own[rep.Planet] = true
			}
		}
		for _, p := range g.Planets {
			if p.Owner == v && !own[p.ID] {
				add("player %d's view lacks their planet %d", v, p.ID)
			}
		}
		for _, s := range view.Fleets {
			fi, ok := fleetByID[s.Fleet]
			if !ok {
				add("player %d's view: fleet %d does not exist", v, s.Fleet)
				continue
			}
			f := g.Fleets[fi]
			if s.Owner != f.Owner || s.Pos != f.Pos || f.Owner == v {
				add("player %d's view: fleet %d of %d at %v, really of %d at %v", v, s.Fleet, s.Owner, s.Pos, f.Owner, f.Pos)
			}
		}
	}

	if len(bad) > 0 {
		return &InvariantError{Year: g.Year, Problems: bad}
	}
	return nil
}
