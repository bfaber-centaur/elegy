package ai

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/terraform"
)

// idealWarp is a fleet's ideal warp (stars-elegy PRODUCTION-LAUNCH.md
// "Ideal warp of the fleet", CONFIRMED for Long Hump 6 and Quick Jump 5,
// BINARY-ONLY for other engines; ESTIMATES.md "Fleets" uses the same
// value): from warp 10, each design in fleet order lowers it.
//
// The engine has the same rule unexported; this copy is the planner's
// until the engine exports one.
func (v *View) idealWarp(f *engine.Fleet) int {
	w := 10
	for _, s := range f.Stacks {
		d, ok := v.design(s.Design)
		if !ok || d.Engines == 0 {
			w = 0
			continue
		}
		fuel := d.Engine.Fuel
		for w > 0 && fuel[w] >= 121 {
			w--
		}
		name := d.Engine.Name
		if fuel[w] > 0 && name != "Trans-Galactic Mizer Scoop" && name != "Galaxy Scoop" {
			switch {
			case w >= 5 && fuel[w-1] == 0:
				w--
			case w >= 6 && fuel[w-2] == 0:
				w -= 2
			case w >= 7 && fuel[w-3] == 0:
				w -= 3
			}
		}
		switch name {
		case "Interspace-10", "Enigma Pulsar", "Trans-Star 10", "Trans-Galactic Mizer Scoop", "Galaxy Scoop":
		default:
			if w == 10 {
				w = 9
			}
		}
	}
	return w
}

// cargoCapacity is the fleet's cargo hold in kT.
func (v *View) cargoCapacity(f *engine.Fleet) int {
	n := 0
	for _, s := range f.Stacks {
		if d, ok := v.design(s.Design); ok {
			n += s.Count * d.CargoCapacity
		}
	}
	return n
}

func cargoMass(c engine.Cargo) int {
	return c.Minerals[engine.Ironium] + c.Minerals[engine.Boranium] + c.Minerals[engine.Germanium] + c.Colonists
}

// engineRank is an engine's place in the engine list (Settler's Delight
// 0, Quick Jump 5 1, Fuel Mizer 2, Long Hump 6 3, …), or −1.
func engineRank(name string) int {
	c, ok := engine.Components().Lookup(name)
	if !ok || c.Category != engine.CatEngine {
		return -1
	}
	return c.Index
}

// terraformedHab is a planet's habitability for the player after the
// terraforming its tech allows (ESTIMATES.md "Value and optimal value",
// CONFIRMED ES-001): each non-immune axis moves toward the race's centre
// within orig ± reach, clipped to 1..99. ok is false when the report
// does not hold the environment.
//
// ASSUMPTION A6: a report's current environment stands for the planet's
// original values; a player's report does not carry them.
func (v *View) terraformedHab(r engine.PlanetReport) (int, bool) {
	if r.Level < engine.ReportNormal {
		return 0, false
	}
	reach := terraform.Reach(v.Self.Race, v.Self.Research.Levels)
	env := r.Env
	for a, er := range v.Self.Race.Env {
		if !er.Immune {
			env[a] = terraform.Limit(env[a], r.Env[a], er.Center, reach[a])
		}
	}
	return engine.Habitability(v.Self.Race, env), true
}

// moveOrder is AI.md §11's move order: it keeps waypoint 0 (the fleet's
// position and task) and sets waypoint 1 to the target with the task and
// warp, dropping later waypoints; at the target already, the task goes on
// waypoint 0 and the route is cut to it. The fleet in the planner's
// picture changes too.
func moveOrder(f *engine.Fleet, wp engine.Waypoint) engine.WaypointOrder {
	if wp.Pos == f.Pos {
		f.Task, f.Waypoints = wp.Task, nil
	} else {
		f.Waypoints = []engine.Waypoint{wp}
	}
	return engine.WaypointOrder{Fleet: f.ID, Task: f.Task, Waypoints: append([]engine.Waypoint(nil), f.Waypoints...)}
}

// toPlanet is a waypoint at a planet.
func toPlanet(id int, pos engine.Point, warp int, task engine.Task) engine.Waypoint {
	return engine.Waypoint{Pos: pos, Warp: warp, Target: engine.TargetPlanet, ID: id, Task: task}
}

// toWormhole is a waypoint at a wormhole end (AI.md §11 "Wormhole
// order"). ASSUMPTION A46: the engine's waypoints cannot target a
// wormhole end yet (an engine request is open), so the waypoint is the
// point in space where the planner last saw the end.
func toWormhole(w Wormhole, warp int) engine.Waypoint {
	return engine.Waypoint{Pos: w.Pos, Warp: warp, Target: engine.TargetSpace}
}

// cutRoute cuts the route to waypoint 0 and clears its task.
func cutRoute(f *engine.Fleet) engine.WaypointOrder {
	f.Task, f.Waypoints = engine.Task{}, nil
	return engine.WaypointOrder{Fleet: f.ID}
}

// armed reports a design with a beam or torpedo.
func armed(d engine.Design) bool {
	for _, s := range d.Slots {
		if s.Count > 0 && (s.Part.Kind == engine.PartBeam || s.Part.Kind == engine.PartTorpedo) {
			return true
		}
	}
	return false
}

// Hull roles (AI.md §11 "Fleet classes").
func freighterHull(h string) bool {
	switch h {
	case "Small Freighter", "Medium Freighter", "Large Freighter", "Super Freighter":
		return true
	}
	return false
}

func privateerHull(h string) bool {
	switch h {
	case "Privateer", "Rogue", "Galleon":
		return true
	}
	return false
}

// transport reports a transport fleet (AI.md §11 "Fleet classes"): any
// freighter or privateer design. ASSUMPTION A7: an armed Meta Morph with
// 500 kT of cargo, also a transport there, is not recognised yet; it
// needs the design-power formula, and Rototill never has one.
func (v *View) transport(f *engine.Fleet) bool {
	for _, s := range f.Stacks {
		if d, ok := v.design(s.Design); ok && s.Count > 0 && (freighterHull(d.Hull.Name) || privateerHull(d.Hull.Name)) {
			return true
		}
	}
	return false
}
