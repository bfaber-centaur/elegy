package objects

import "github.com/bfaber-centaur/elegy/engine"

// Salvage rules (stars-elegy OBJECTS.md "Salvage", BINARY-ONLY except
// where marked). The objects live in engine.Game.Salvage: the engine
// creates battle and scrap salvage itself, and these functions number,
// join, decay and load them.

// salvageSteps is one salvage object's limit, 30,000 kT in 10 kT steps
// (COMBAT.md "Salvage", the overflow CONFIRMED CB-040).
const salvageSteps = 3000

// objectCount is every space object, salvage included (LIMITS.md "Space
// objects").
func (s *Space) objectCount(g *engine.Game) int {
	return len(s.Minefields) + len(s.Packets) + 2*len(s.Wormholes) + len(s.Traders) + len(g.Salvage)
}

// packetNumber is the owner's lowest unused number in the pool its
// packets and salvage share (OBJECTS.md "Launch", "Numbering", and
// "Salvage", "Owner"): 0..510, and 511 only when nothing sorts after the
// owner's packets and salvage. A higher-numbered player's packets or
// salvage, any wormhole and any Trader sort after them; minefields and
// lower players' objects never do.
func (s *Space) packetNumber(g *engine.Game, owner int) (int, bool) {
	used := map[int]bool{}
	later := len(s.Wormholes) > 0 || len(s.Traders) > 0
	note := func(o, n int) {
		if o == owner {
			used[n] = true
		} else if o > owner {
			later = true
		}
	}
	for _, q := range s.Packets {
		note(q.Owner, q.Number)
	}
	for _, sv := range g.Salvage {
		note(sv.Owner, sv.Number)
	}
	limit := maxPacketNumber
	if !later {
		limit++
	}
	n := 0
	for used[n] {
		n++
	}
	return n, n < limit
}

// SalvageNumber is the number a new salvage object of owner takes, and
// whether there is room for it: a free number in the owner's packet pool
// and a free object slot.
//
// ASSUMPTION S1: with no number or slot no object is made and its
// minerals are lost (LIMITS.md "Space objects" lists this as UNKNOWN).
func (s *Space) SalvageNumber(g *engine.Game, owner int) (int, bool) {
	n, ok := s.packetNumber(g, owner)
	return n, ok && s.objectCount(g) < MaxObjects
}

// NewSalvage makes a salvage object of owner at pos holding m, marked
// fresh, splitting off further objects at the same spot past 30,000 kT
// (addSalvage). It reports false when there was no room (S1).
func (s *Space) NewSalvage(g *engine.Game, owner int, pos engine.Point, m engine.Minerals) bool {
	n, ok := s.SalvageNumber(g, owner)
	if !ok {
		return false
	}
	g.Salvage = append(g.Salvage, engine.Salvage{Pos: pos, Owner: owner, Number: n, Fresh: true})
	s.addSalvage(g, len(g.Salvage)-1, owner, m)
	return true
}

// AddMineSalvage puts a mine hit's salvage m, from a fleet of owner
// stopped at pos, into the first salvage object lying exactly at pos, in
// object order (owner, then number), whatever its owner and age, or into
// a new object of owner (OBJECTS.md "Salvage", "Joining existing
// salvage"). The object is marked fresh, so it skips the next decay. A
// planet's exact position takes no salvage (the caller's: ApplyHit
// returns none there). It reports false when a needed object found no
// room (S1).
func (s *Space) AddMineSalvage(g *engine.Game, owner int, pos engine.Point, m engine.Minerals) bool {
	if m == (engine.Minerals{}) {
		return true
	}
	first := -1
	for i, sv := range g.Salvage {
		if sv.Pos != pos {
			continue
		}
		if f := g.Salvage; first < 0 || sv.Owner < f[first].Owner || (sv.Owner == f[first].Owner && sv.Number < f[first].Number) {
			first = i
		}
	}
	if first < 0 {
		return s.NewSalvage(g, owner, pos, m)
	}
	g.Salvage[first].Fresh = true
	return s.addSalvage(g, first, owner, m)
}

// addSalvage adds m to salvage object i under the 30,000 kT limit
// (COMBAT.md "Salvage", the overflow CONFIRMED CB-040): the object's
// contents are taken out and added back with m, ironium, boranium,
// germanium, each mineral using ⌈kT/10⌉ steps; a mineral that does not fit
// fills the object to exactly 3,000 steps with 10 × the free steps, and
// the rest goes, in a new pass, into a new object at the same spot
// belonging to owner (the addition's), not marked fresh (OBJECTS.md
// "Salvage", "Owner", "Decay").
func (s *Space) addSalvage(g *engine.Game, i, owner int, m engine.Minerals) bool {
	add := g.Salvage[i].Minerals
	for k := range engine.NumMinerals {
		add[k] += m[k]
	}
	g.Salvage[i].Minerals = engine.Minerals{}
	for {
		obj := &g.Salvage[i]
		used, full := 0, false
		for k := range engine.NumMinerals {
			a := add[k]
			if a == 0 {
				continue
			}
			if steps := (a + 9) / 10; used+steps <= salvageSteps {
				obj.Minerals[k] += a
				used += steps
				add[k] = 0
				continue
			}
			fit := 10 * (salvageSteps - used)
			obj.Minerals[k] += fit
			add[k] -= fit
			full = true
			break
		}
		obj.Steps = tenths(obj.Minerals)
		if !full {
			return true
		}
		n, ok := s.SalvageNumber(g, owner)
		if !ok {
			return false
		}
		g.Salvage = append(g.Salvage, engine.Salvage{Pos: obj.Pos, Owner: owner, Number: n})
		i = len(g.Salvage) - 1
	}
}

// DecaySalvage is the salvage's yearly decay, with packets (OBJECTS.md
// "Salvage", "Decay"; KERNEL.md "Turn order" 3a): a fresh object only
// loses its mark; otherwise each non-empty mineral loses max(10, ⌊m/10⌋),
// not below 0, and an object with nothing left is removed.
func DecaySalvage(g *engine.Game) {
	kept := g.Salvage[:0]
	for _, sv := range g.Salvage {
		if sv.Fresh {
			sv.Fresh = false
			kept = append(kept, sv)
			continue
		}
		for k, m := range sv.Minerals {
			if m > 0 {
				sv.Minerals[k] = max(0, m-max(10, m/10))
			}
		}
		if sv.Minerals != (engine.Minerals{}) {
			sv.Steps = tenths(sv.Minerals)
			kept = append(kept, sv)
		}
	}
	g.Salvage = kept
}

// SalvageLoad is how much of mineral k a fleet loading `want` kT gets
// from a salvage object: the usual load amount, capped by what it holds
// (OBJECTS.md "Salvage", "Loading"). Colonists and fuel cannot be loaded
// from salvage. An emptied object stays until the next decay.
func SalvageLoad(sv engine.Salvage, k, want int) int {
	if k < 0 || k >= engine.NumMinerals || want <= 0 {
		return 0
	}
	return min(want, sv.Minerals[k])
}

// SalvageRoom is how many kT can be unloaded into a salvage object: the
// slack between its stored size (Steps, 10 kT each) and what it holds
// now (OBJECTS.md "Salvage", "Loading").
func SalvageRoom(sv engine.Salvage) int {
	t := 0
	for _, m := range sv.Minerals {
		t += m
	}
	return max(0, 10*sv.Steps-t)
}
