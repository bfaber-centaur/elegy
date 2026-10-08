package terraform

import (
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// AdjusterClicks is the number of Orbital Adjusters in fleet f: one click
// a year each, whatever the part's value (KERNEL.md "Terraforming",
// Orbital Adjusters, CONFIRMED KX-005). An adjuster is a part with a
// terraform_pct statistic (COMPONENTS.md).
func AdjusterClicks(g *engine.Game, f *engine.Fleet) int {
	cat := engine.Components()
	n := 0
	for _, st := range f.Stacks {
		if st.Design < 0 || st.Design >= len(g.Designs) {
			continue
		}
		per := 0
		for _, sl := range g.Designs[st.Design].Slots {
			c, ok := cat.Lookup(sl.Part.Name)
			if !ok {
				continue
			}
			if v, ok := c.Stats["terraform_pct"].(float64); ok && v > 0 {
				per += sl.Count
			}
		}
		n += per * st.Count
	}
	return n
}

// Adjustment is what one fleet's Orbital Adjusters did to a planet.
type Adjustment struct {
	Fleet  int // index into Game.Fleets
	Planet int // index into Game.Planets
	// Hostile: the fleet owner is not the planet owner and does not treat
	// it as a friend, so the clicks worsened the planet.
	Hostile bool
	// Axes are the axes moved, one per click made.
	Axes []int
	// HabChanged: the planet owner's habitability changed. The fleet
	// owner is told about every changed planet, the planet owner only
	// when this is set (the turn engine sends the messages).
	HabChanged bool
}

// Adjust runs KERNEL.md "Turn order" step 7.4, remote terraforming by
// Orbital Adjusters (CONFIRMED KX-005 T0–T2): every fleet orbiting an
// owned planet makes one click per adjuster on it, with the fleet owner's
// reach at its current tech levels (the turn engine calls it after this
// year's research) and the planet owner's habitat. A fleet that arrived
// this year counts.
//
//   - Fleet owner = planet owner, or the fleet owner treats the planet
//     owner as a friend: each click improves the planet (Improve),
//     starbase or not.
//   - Otherwise: nothing if the planet has a starbase; else each click
//     worsens it (Worsen).
//
// Only planets that changed are returned.
//
// ASSUMPTION T3: fleets act one at a time in fleet order (owner, then
// number), each on the state the previous left.
func Adjust(g *engine.Game) []Adjustment {
	var out []Adjustment
	for _, fi := range fleetOrder(g) {
		f := &g.Fleets[fi]
		n := AdjusterClicks(g, f)
		if n == 0 {
			continue
		}
		pi := planetAt(g, f.Pos)
		if pi < 0 {
			continue
		}
		p := &g.Planets[pi]
		if p.Owner == engine.NoOwner || p.Owner < 0 || p.Owner >= len(g.Players) {
			continue
		}
		hostile := p.Owner != f.Owner && relation(g, f.Owner, p.Owner) != engine.RelationFriend
		if hostile && p.HasStarbase {
			continue
		}
		race := g.Players[p.Owner].Race
		reach := Reach(g.Players[f.Owner].Race, g.Players[f.Owner].Research.Levels)
		before := engine.Habitability(race, p.Env)
		adj := Adjustment{Fleet: fi, Planet: pi, Hostile: hostile}
		for range n {
			var a int
			if hostile {
				a = Worsen(p, race, reach)
			} else {
				a = Improve(p, race, reach)
			}
			if a < 0 {
				break
			}
			adj.Axes = append(adj.Axes, a)
		}
		if len(adj.Axes) == 0 {
			continue
		}
		adj.HabChanged = engine.Habitability(race, p.Env) != before
		out = append(out, adj)
	}
	return out
}

// planetAt is the index of the planet at pos (a fleet orbits a planet
// exactly when it is at its position), or -1.
func planetAt(g *engine.Game, pos engine.Point) int {
	for i := range g.Planets {
		if g.Planets[i].Pos == pos {
			return i
		}
	}
	return -1
}

// relation is player a's view of player b: a player is its own friend and
// a missing entry is neutral (engine.Player.Relations).
func relation(g *engine.Game, a, b int) engine.Relation {
	if a == b {
		return engine.RelationFriend
	}
	if r := g.Players[a].Relations; b >= 0 && b < len(r) {
		return r[b]
	}
	return engine.RelationNeutral
}

// fleetOrder is the fleet indices by owner, then number, then id.
func fleetOrder(g *engine.Game) []int {
	order := make([]int, len(g.Fleets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := &g.Fleets[order[i]], &g.Fleets[order[j]]
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		return a.ID < b.ID
	})
	return order
}
