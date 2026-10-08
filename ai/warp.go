package ai

import (
	"math"

	"github.com/bfaber-centaur/elegy/engine"
)

// repickWarps is AI.md §7 step 1 with §11 "Warp choice" (CONFIRMED,
// AI-11): every own fleet with a waypoint after its position, in fleet
// order, gets the warp of that waypoint re-picked; target and task stay,
// and an order is written only when the warp changes.
//
// ASSUMPTION A57: the minefield rule (a fleet inside another player's
// enlarged field gets warp 4, 5 or 6) is not taken: the report does not
// carry minefield positions or sizes, so no fleet is known to be inside
// one, and its Random(10) is not drawn. A turn with a fleet to re-pick
// reports this in Result.Unsupported. A stargate route (warp 11, not yet
// observed) is never re-picked here: the planners order no gate jumps.
func (a *automation) repickWarps() {
	v := a.v
	noted := false
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if len(f.Waypoints) == 0 || f.Waypoints[0].Warp == engine.StargateWarp {
			continue
		}
		if !noted {
			a.res.unsupported("warp re-pick: minefields are not in the report (ASSUMPTION A57)")
			noted = true
		}
		w := v.warpChoice(f)
		if w == f.Waypoints[0].Warp {
			continue
		}
		f.Waypoints[0].Warp = w
		a.res.Orders = append(a.res.Orders, engine.WaypointOrder{Fleet: f.ID, Task: f.Task, Waypoints: append([]engine.Waypoint(nil), f.Waypoints...)})
	}
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
// ASSUMPTION A58: the planet's starbase slot is its StarbaseDesign,
// read whether or not it has a starbase (§11: "the slot is read even
// when the planet has no starbase"), and a design the view does not hold
// counts as no design.
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
		d, ok := v.design(p.StarbaseDesign)
		return ok && d.Hull.Name != orbitalFort
	}
	return false
}
