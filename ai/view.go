package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// View is everything a planner reads: one player's own objects in full
// and that player's knowledge of the rest of the universe (AI.md §1 "What
// it sees", CONFIRMED AI-12). It holds nothing of another player that
// the player was not told. A planner changes its View as it plans (AI.md
// §11 "Supplies": cargo moved at once in the computer player's own
// picture), so callers pass a copy they do not need afterwards.
type View struct {
	Year   int // calendar year the orders are for
	Player int
	Level  Level
	// Rules is the game's ruleset (game.Report.Rules).
	Rules engine.Ruleset

	Self engine.Player
	// Planets are the player's own planets.
	Planets []engine.Planet
	// Fleets are the player's own fleets.
	Fleets []engine.Fleet
	// Ships and Starbases are the player's designs, one per filled slot.
	Ships, Starbases []Design

	// Universe is every planet's id and position: the map every player
	// has.
	Universe []PlanetPos
	// Known is the latest report of every planet the player has had a
	// report of, this year or in its planet history, by planet id. A
	// planet absent from Known has never been seen.
	Known map[int]engine.PlanetReport
	// Seen marks the planets reported this year (the rest of Known comes
	// from the planet history).
	Seen map[int]bool
	// Others are other players' fleets seen this year.
	Others []engine.FleetSighting
	// Wormholes are the wormhole ends the player knows of.
	Wormholes []Wormhole
	// PRT is the primary racial trait of other players, where known.
	PRT map[int]engine.PRT
	// Foreign are other players' designs the player has been shown in
	// full, by design index (game.Report.KnownDesigns with Full set).
	Foreign map[int]engine.Design
}

// Design is one of the player's designs.
type Design struct {
	Slot int
	// Index is the design's index in the game's design list, which fleet
	// stacks and planet starbases refer to.
	Index   int
	Design  engine.Design
	Created int // calendar year stored
	Picture int
}

// PlanetPos is a planet's id and position.
type PlanetPos struct {
	ID  int
	Pos engine.Point
}

// Wormhole is a wormhole end the player knows of.
type Wormhole struct {
	End int // engine.TargetWormhole id
	Pos engine.Point
	// Known marks a wormhole whose movement class the player knows, Class
	// that class (AI.md §11 "Nearest colonizable planet").
	Known bool
	Class int
}

// owner is a planet's owner in the player's view (AI.md §1, CONFIRMED
// AI-12): its own planets are its own; another planet has the owner its
// latest report records, except that a planet the history records as the
// player's own that it no longer has counts as unowned; a planet never
// seen counts as unowned.
func (v *View) owner(id int) int {
	if v.ownPlanet(id) != nil {
		return v.Player
	}
	r, ok := v.Known[id]
	if !ok || r.Owner == v.Player {
		return engine.NoOwner
	}
	return r.Owner
}

func (v *View) ownPlanet(id int) *engine.Planet {
	for i := range v.Planets {
		if v.Planets[i].ID == id {
			return &v.Planets[i]
		}
	}
	return nil
}

// planetAt is the planet at a position (a fleet orbits a planet exactly
// when their positions are equal).
func (v *View) planetAt(pos engine.Point) (int, bool) {
	for _, p := range v.Universe {
		if p.Pos == pos {
			return p.ID, true
		}
	}
	return 0, false
}

func (v *View) planetPos(id int) (engine.Point, bool) {
	for _, p := range v.Universe {
		if p.ID == id {
			return p.Pos, true
		}
	}
	return engine.Point{}, false
}

// shipSlot is the ship slot of a design index, or −1.
func (v *View) shipSlot(index int) int {
	for _, d := range v.Ships {
		if d.Index == index {
			return d.Slot
		}
	}
	return -1
}

func (v *View) ship(slot int) (Design, bool) {
	for _, d := range v.Ships {
		if d.Slot == slot {
			return d, true
		}
	}
	return Design{}, false
}

func (v *View) design(index int) (engine.Design, bool) {
	for _, d := range v.Ships {
		if d.Index == index {
			return d.Design, true
		}
	}
	for _, d := range v.Starbases {
		if d.Index == index {
			return d.Design, true
		}
	}
	return engine.Design{}, false
}

// alive is the number of own ships of each ship slot.
func (v *View) alive() map[int]int {
	n := map[int]int{}
	for _, f := range v.Fleets {
		for _, s := range f.Stacks {
			if k := v.shipSlot(s.Design); k >= 0 {
				n[k] += s.Count
			}
		}
	}
	return n
}

// holds reports whether a fleet has ships of one of the slots.
func (v *View) holds(f *engine.Fleet, slots ...int) bool {
	for _, s := range f.Stacks {
		if s.Count > 0 && slices.Contains(slots, v.shipSlot(s.Design)) {
			return true
		}
	}
	return false
}

// fleetOrder sorts own fleets into fleet order (owner, fleet number, id).
func (v *View) fleetOrder() {
	slices.SortStableFunc(v.Fleets, func(a, b engine.Fleet) int {
		if a.Number != b.Number {
			return a.Number - b.Number
		}
		return a.ID - b.ID
	})
}

func d2(a, b engine.Point) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}
