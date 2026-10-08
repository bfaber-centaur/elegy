package terraform

import "github.com/bfaber-centaur/elegy/engine"

// PacketResult is what a Packet Physics packet's terraforming did to a
// planet, per axis (gravity, temperature, radiation).
type PacketResult struct {
	Successes [3]int // successful draws
	Permanent [3]int // successes that also moved the original value
	// Moved is the change of the current value, signed.
	Moved [3]int
	// OrigMoved is the change of the original value, signed.
	OrigMoved [3]int
}

// Changed reports whether the planet's environment changed.
func (r PacketResult) Changed() bool { return r.Moved != [3]int{} || r.OrigMoved != [3]int{} }

// Uncaught is the per-mineral uncaught share u = ⌊m·(1000 − q)/1000⌋ of a
// packet's cargo m (after arrival decay) when its catcher caught q per
// mille (OBJECTS.md "Impact", PP terraforming).
func Uncaught(m engine.Minerals, q int) engine.Minerals {
	var u engine.Minerals
	for k := range m {
		u[k] = m[k] * (1000 - q) / 1000
	}
	return u
}

// PacketTerraform applies a Packet Physics packet's terraforming to the
// planet at index pi, owned or not (OBJECTS.md "Impact" step 4, "PP
// terraforming"; CONFIRMED in part OB-029-T1..T3, the draws and their
// order BINARY-ONLY). pp is the packet's owner and u its uncaught share
// (Uncaught). The caller applies it only when the launcher is a Packet
// Physics player and the packet was not fully caught, before the
// impact's damage.
//
// Each mineral works on one axis: ironium gravity, boranium temperature,
// germanium radiation (OB-029-T1..T3), in that order, each axis's moves
// applied before the next axis draws; the moves make no draws. For each
// 100 kT chunk of u (the last may be partial) one draw
// rand(200) < min(chunk, 100) is a success, and each success draws
// rand(10) == 0 for a permanent one.
//
//   - An axis the PP player is not immune to: the permanent count moves
//     the original value toward the PP player's ideal, capped there. The
//     limit is then the new original ± the PP player's reach, clamped to
//     1..99, in the improving direction only and never past the ideal
//     (Limit); the current value moves toward it by at most the success
//     count. With no limit nothing moves.
//   - An axis the PP player is immune to: both moves go toward 1 when the
//     original value is below 50, else toward 99. The permanent count
//     moves the original value; half the success count, rounded down,
//     moves the current value, but only if some axis the PP player is not
//     immune to still has a limit at that moment. An immune axis never has
//     a limit of its own.
func PacketTerraform(g *engine.Game, pi, pp int, u engine.Minerals, rng engine.Rand) PacketResult {
	var r PacketResult
	p := &g.Planets[pi]
	race := g.Players[pp].Race
	reach := Reach(race, g.Players[pp].Research.Levels)
	for a := range 3 {
		for rem := u[a]; rem > 0; rem -= 100 {
			if rng.Intn(200) < min(rem, 100) {
				r.Successes[a]++
				if rng.Intn(10) == 0 {
					r.Permanent[a]++
				}
			}
		}
		if r.Successes[a] == 0 {
			continue
		}
		env := race.Env[a]
		orig, cur := p.OrigEnv[a], p.Env[a]
		if env.Immune {
			p.OrigEnv[a] = toward(orig, extreme(orig), r.Permanent[a])
			if anyLimit(p, race, reach) {
				p.Env[a] = toward(cur, extreme(orig), r.Successes[a]/2)
			}
		} else {
			p.OrigEnv[a] = toward(orig, env.Center, r.Permanent[a])
			if lim := Limit(cur, p.OrigEnv[a], env.Center, reach[a]); lim != cur {
				p.Env[a] = toward(cur, lim, r.Successes[a])
			}
		}
		r.OrigMoved[a] = p.OrigEnv[a] - orig
		r.Moved[a] = p.Env[a] - cur
	}
	return r
}

// anyLimit reports whether some axis the race is not immune to still has
// room on p with this reach.
func anyLimit(p *engine.Planet, race engine.Race, reach [3]int) bool {
	for a, r := range race.Env {
		if !r.Immune && Limit(p.Env[a], p.OrigEnv[a], r.Center, reach[a]) != p.Env[a] {
			return true
		}
	}
	return false
}

// toward moves v by at most n toward target, stopping at it.
func toward(v, target, n int) int {
	if v < target {
		return min(v+n, target)
	}
	return max(v-n, target)
}

// extreme is the end an immune axis moves toward: 1 for an original
// value below 50, else 99.
func extreme(orig int) int {
	if orig < 50 {
		return 1
	}
	return 99
}
