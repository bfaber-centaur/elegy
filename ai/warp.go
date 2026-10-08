package ai

import (
	"math"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// repickWarps is AI.md §7 step 1 with §11 "Warp choice" (CONFIRMED,
// AI-11): every own fleet with a waypoint after its position, in fleet
// order, gets the warp of that waypoint re-picked; target and task stay,
// and an order is written only when the warp changes. A fleet inside
// another player's (enlarged) minefield takes the minefield warp, which
// may draw Random(10); any other fleet takes warpChoice. A stargate route
// (warp 11, not yet observed) is never re-picked here: the planners order
// no gate jumps.
func (a *automation) repickWarps() {
	v := a.v
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if len(f.Waypoints) == 0 || f.Waypoints[0].Warp == engine.StargateWarp {
			continue
		}
		w, inside := v.minefieldWarp(f, a.rng)
		if !inside {
			w = v.warpChoice(f)
		}
		if w == f.Waypoints[0].Warp {
			continue
		}
		f.Waypoints[0].Warp = w
		a.res.Orders = append(a.res.Orders, engine.WaypointOrder{Fleet: f.ID, Task: f.Task, Waypoints: append([]engine.Waypoint(nil), f.Waypoints...)})
	}
}

// minefieldWarp is §11's minefield rule: the computer player sees a field
// of n mines as ⌊√n + 10.5⌋² in squared radius, and a fleet whose squared
// distance from the centre of another player's field is below that, the
// first such field in object order (the report's order), gets 6 for a
// heavy field, or for a standard field 4 when Random(10) < 4 and 5
// otherwise, plus 1 for an SS race. Speed bump fields are not considered.
func (v *View) minefieldWarp(f *engine.Fleet, rng engine.Rand) (int, bool) {
	for _, m := range v.Minefields {
		if m.Owner == v.Player || m.Kind == objects.SpeedBump {
			continue
		}
		r := int(math.Sqrt(float64(m.Mines)) + 10.5)
		dx, dy := f.Pos.X-m.Pos.X, f.Pos.Y-m.Pos.Y
		if dx*dx+dy*dy >= r*r {
			continue
		}
		w := 6
		if m.Kind == objects.Standard {
			w = 5
			if rng.Intn(10) < 4 {
				w = 4
			}
		}
		if v.Self.Race.PRT == engine.PRTSuperStealth {
			w++
		}
		return w, true
	}
	return 0, false
}

// warpChoice is §11 "Warp choice" without the minefield rule for the leg
// from the fleet's position to its first waypoint.
func (v *View) warpChoice(f *engine.Fleet) int {
	s := v.ships(f)
	wp := f.Waypoints[0]
	ideal, raw := s.IdealWarp(), s.RawWarp()
	dx, dy := float64(wp.Pos.X-f.Pos.X), float64(wp.Pos.Y-f.Pos.Y)
	dist := math.Sqrt(dx*dx + dy*dy)
	// 1. From 9 (or ideal when it is 9 or more), down while above ideal
	// and the leg's fuel estimate (ESTIMATES.md "Fleets") exceeds the
	// fleet's fuel.
	w := 9
	if ideal >= 9 {
		w = ideal
	}
	for w > ideal && s.EstLegFuel(dist, w) > f.Fuel {
		w--
	}
	// 2. Cap at raw, with the exceptions.
	if w > raw && !v.uncapped(f, wp) {
		w = raw
	}
	// 3. The slowest warp with the same whole years.
	if wp.Target != engine.TargetFleet {
		d := int(dist)
		years := func(w int) int { return (d + w*w - 1) / (w * w) }
		for w > 2 && years(w-1) == years(w) {
			w--
		}
	}
	return w
}

// uncapped reports §11's exceptions to the raw cap: waypoint 1's task is
// colonize or scrap; a ship carries a colonization or orbital
// construction module; or the player owns a planet exactly at waypoint
// 1 whose starbase slot holds a design other than an Orbital Fort.
//
// ASSUMPTION A58: the planet's starbase slot is its StarbaseDesign, read
// whether or not it has a starbase (§11: "the slot is read even when the
// planet has no starbase"). A planet without a starbase holds the design
// of its last starbase (the engine keeps it when a starbase is destroyed
// or scrapped) or, if it never had one, 0. The index is looked up only
// among the player's starbase designs; one that is not among them (a
// ship design, or a deleted starbase design) counts as no design, so the
// cap stays.
func (v *View) uncapped(f *engine.Fleet, wp engine.Waypoint) bool {
	if wp.Task.Kind == engine.TaskColonize || wp.Task.Kind == engine.TaskScrap {
		return true
	}
	for _, st := range f.Stacks {
		d, _ := v.design(st.Design)
		for _, sl := range d.Slots {
			if st.Count > 0 && sl.Count > 0 && sl.Part.Colonizer {
				return true
			}
		}
	}
	for i := range v.Planets {
		p := &v.Planets[i]
		if p.Pos != wp.Pos {
			continue
		}
		for _, d := range v.Starbases {
			if d.Index == p.StarbaseDesign {
				return d.Design.Hull.Name != orbitalFort
			}
		}
		return false
	}
	return false
}
