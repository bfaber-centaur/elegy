// Package terraform implements the planet-side rules the turn engine
// applies to a planet's environment and deposits outside its own
// production loop: terraforming by production units, by Orbital
// Adjusters and by Packet Physics packets, and remote mining by fleets.
//
// The rules are stars-elegy docs/KERNEL.md "Terraforming" and "Remote
// mining", OBJECTS.md "Impact" (PP terraforming) and COMPONENTS.md
// "Remote mining" (stars-elegy main at 0779d2f). Choices the spec leaves
// open are labelled ASSUMPTION. The package reads and changes the
// engine's exported state; when each function runs in the year is the
// turn engine's (docs/TERRAFORM-STATUS.md lists the call sites).
package terraform

import (
	"github.com/bfaber-centaur/elegy/engine"
)

// Axes, in the order the spec breaks ties: gravity, temperature,
// radiation.
const (
	Gravity = iota
	Temperature
	Radiation
)

var axisNames = [3]string{"gravity", "temperature", "radiation"}

// Reach is how far a race at these tech levels can terraform each axis
// from its original value (KERNEL.md "Terraforming", reach per axis): the
// largest value among the race's available terraform parts for the axis,
// a Total Terraform part counting for every axis. Availability follows
// the component table's tech and LRT rules (Total Terraform needs TT).
// An immune axis has no reach (CONFIRMED, KB-2C).
func Reach(race engine.Race, levels [engine.NumFields]int) [3]int {
	var reach [3]int
	for _, c := range engine.Components().Components {
		if c.Category != engine.CatTerraform {
			continue
		}
		if ok, err := c.Buildable(race, levels, false); !ok || err != nil {
			continue
		}
		amount, _ := c.Stats["amount"].(float64)
		axis, _ := c.Stats["axis"].(string)
		for a, name := range axisNames {
			if axis == name || axis == "all" {
				reach[a] = max(reach[a], int(amount))
			}
		}
	}
	for a, r := range race.Env {
		if r.Immune {
			reach[a] = 0
		}
	}
	return reach
}

// Limit is the value an axis can be improved to: from the current value
// toward the centre, within orig ± reach clipped to 1–99, stopping at the
// centre (KERNEL.md "Terraforming", CONFIRMED KX-002 T1, T3). It is the
// current value when there is no room.
func Limit(cur, orig, centre, reach int) int {
	switch {
	case cur < centre:
		return max(cur, min(centre, 99, orig+reach))
	case cur > centre:
		return min(cur, max(centre, 1, orig-reach))
	}
	return cur
}

// Capacity is the number of improving clicks still available on a planet
// for the habitat of race with this reach: the sum over the race's
// non-immune axes of the distance from the current value to its limit
// (KERNEL.md "Terraforming", capacity, CONFIRMED KX-002 T1–T3, KB-2C).
func Capacity(p engine.Planet, race engine.Race, reach [3]int) int {
	n := 0
	for a, r := range race.Env {
		if r.Immune {
			continue
		}
		n += abs(Limit(p.Env[a], p.OrigEnv[a], r.Center, reach[a]) - p.Env[a])
	}
	return n
}

// score is the axis-choice score of moving axis a from its current value
// toward target: trunc(|hab(target) − hab(now)|·100 / |target − now|) + 1,
// with hab(target) the habitability with that axis alone at target
// (KERNEL.md "Terraforming", axis choice).
func score(race engine.Race, env [3]int, a, target int) int {
	e := env
	e[a] = target
	return abs(engine.Habitability(race, e)-engine.Habitability(race, env))*100/abs(target-env[a]) + 1
}

// pick returns the axis with room and the highest score, the first axis
// on ties, or -1 when no axis has room. targets[a] == env[a] means no room.
func pick(race engine.Race, env [3]int, targets [3]int) int {
	best, bestScore := -1, 0
	for a := range 3 {
		if targets[a] == env[a] {
			continue
		}
		if s := score(race, env, a, targets[a]); s > bestScore {
			best, bestScore = a, s
		}
	}
	return best
}

// Improve makes one improving click on p for the habitat of race with
// this reach and returns the axis moved, or -1 when the planet has no
// room. Each axis with room is scored toward its limit, the highest wins
// and the first axis wins ties (KERNEL.md "Terraforming", axis choice,
// CONFIRMED KX-002 T2, KX-005). A production unit is one click with the
// planet owner's race and reach; so is a friendly Orbital Adjuster's
// click, with the fleet owner's reach.
func Improve(p *engine.Planet, race engine.Race, reach [3]int) int {
	var targets [3]int
	for a, r := range race.Env {
		targets[a] = p.Env[a]
		if !r.Immune {
			targets[a] = Limit(p.Env[a], p.OrigEnv[a], r.Center, reach[a])
		}
	}
	return step(p, race, targets)
}

// Worsen makes one worsening click on p, the click of a neutral or enemy
// Orbital Adjuster: the habitat is the planet owner's race, the reach the
// fleet owner's. Per axis the target is whichever of orig − reach and
// orig + reach (clipped to 1–99) is farther from the habitat's centre,
// the lower end on a tie, provided it is farther from the centre than the
// current value; the axis is chosen by the same score as Improve
// (KERNEL.md "Terraforming", Orbital Adjusters, CONFIRMED KX-005).
//
// ASSUMPTION T1: an axis the planet owner is immune to is never worsened
// (it has no centre, and changing it does not change the habitability).
func Worsen(p *engine.Planet, race engine.Race, reach [3]int) int {
	var targets [3]int
	for a, r := range race.Env {
		targets[a] = p.Env[a]
		if r.Immune {
			continue
		}
		lo := max(1, p.OrigEnv[a]-reach[a])
		hi := min(99, p.OrigEnv[a]+reach[a])
		t := lo
		if abs(hi-r.Center) > abs(lo-r.Center) {
			t = hi
		}
		if abs(t-r.Center) > abs(p.Env[a]-r.Center) {
			targets[a] = t
		}
	}
	return step(p, race, targets)
}

func step(p *engine.Planet, race engine.Race, targets [3]int) int {
	a := pick(race, p.Env, targets)
	switch {
	case a < 0:
	case targets[a] > p.Env[a]:
		p.Env[a]++
	default:
		p.Env[a]--
	}
	return a
}

// UnitCost is the resource cost of one Terraform Environment unit for a
// race: 100, 70 with Total Terraforming, halved (rounded down) for a Claim
// Adjuster; no minerals (KERNEL.md "Terraforming", cost, CONFIRMED KX-002
// T1, T2, KX-005).
//
// ASSUMPTION T2: a Claim Adjuster with Total Terraforming pays ⌊70/2⌋ =
// 35, the halving applied to the TT price as COMPONENTS.md's cost rule
// halves a CA's terraform parts.
func UnitCost(race engine.Race) int {
	c := 100
	if race.LRT.TotalTerraforming {
		c = 70
	}
	if race.PRT == engine.PRTClaimAdjuster {
		c /= 2
	}
	return c
}

// ClipOrder is what becomes of a Terraform Environment order (or the part
// of it left) when the queue reaches it: cut to the planet's capacity,
// with a message, and removed when that is 0 (KERNEL.md "Terraforming",
// capacity, CONFIRMED KX-002 T1, T3, KB-2C). clipped reports a cut (the
// message); remove reports that the item leaves the queue.
func ClipOrder(count, capacity int) (n int, clipped, remove bool) {
	if count <= capacity {
		return count, false, false
	}
	return capacity, true, capacity == 0
}

// AutoUnits is how many units an Auto Max Terraform (minOnly false) or
// Auto Min Terraform (minOnly true) item with this count builds this year
// (KERNEL.md "Terraforming", CONFIRMED KX-005): up to the capacity, the
// count being a per-year limit; Auto Min builds only when the planet's
// population change this year is negative or its habitability for the
// owner is 0 or less. The item stays in the queue.
func AutoUnits(count, capacity int, minOnly bool, popChange, hab int) int {
	if minOnly && popChange >= 0 && hab > 0 {
		return 0
	}
	return min(count, capacity)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
