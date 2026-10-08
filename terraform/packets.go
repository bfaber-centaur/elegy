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
// terraforming"; CONFIRMED in part OB-029-T1..T3, the draws BINARY-ONLY).
// pp is the packet's owner and u its uncaught share (Uncaught). The
// caller applies it only when the launcher is a Packet Physics player and
// the packet was not fully caught, before the impact's damage.
//
// Each mineral works on one axis: ironium gravity, boranium temperature,
// germanium radiation (OB-029-T1..T3). For each 100 kT chunk of u (the
// last may be partial) one draw rand(200) < min(chunk, 100) is a
// success, and each success draws rand(10) == 0 for a permanent one. The
// permanent count moves the original value toward the PP player's ideal,
// capped there, or toward the nearer extreme on an axis the PP player is
// immune to. Then the current value moves by the success count toward the
// ideal, within the PP player's reach around the (new) original value
// (Limit); on an immune axis by half the count toward the nearer extreme.
//
// ASSUMPTION P1: the minerals are drawn and applied in mineral order
// (ironium, boranium, germanium), each mineral's chunks in order.
//
// ASSUMPTION P2: "the PP player's terraforming tech could improve the
// planet" is read per axis: the current value moves only when Limit for
// that axis, with the PP player's reach and the new original value, is
// not the current value; it then moves at most to that limit.
//
// ASSUMPTION P3: the extremes are 1 and 99 (the clip of KERNEL.md
// "Terraforming"); the nearer extreme is the one nearer to the value
// being moved, 1 on a tie (value 50); half the count is rounded down;
// the immune-axis move is not gated by P2.
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
			p.Env[a] = toward(cur, extreme(cur), r.Successes[a]/2)
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

// toward moves v by at most n toward target, stopping at it.
func toward(v, target, n int) int {
	if v < target {
		return min(v+n, target)
	}
	return max(v-n, target)
}

func extreme(v int) int {
	if v > 50 {
		return 99
	}
	return 1
}
