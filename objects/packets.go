package objects

import (
	"math"

	"github.com/bfaber-centaur/elegy/engine"
)

// Packet is a mass-driver mineral packet in flight (OBJECTS.md
// "Mass-driver packets").
type Packet struct {
	Owner, Number int
	Pos           engine.Point
	// Target is the destination planet's ID; From the launching planet's.
	Target, From int
	Warp, Class  int
	Cargo        engine.Minerals
	// New marks a packet launched this year, which flies half a year
	// after production (Space.FlyLaunched).
	New bool
}

// Mixed is the mineral argument for a mixed packet item.
const Mixed = -1

// maxPacketMineral caps one launch per mineral (OBJECTS.md "Launch",
// BINARY-ONLY); mergeLimit is the total under which a packet still takes
// merges (BINARY-ONLY).
const (
	maxPacketMineral = 32760
	mergeLimit       = 16300
)

// DriverWarp is a starbase design's driver warp Dw and the two-driver
// bonus t (OBJECTS.md "Launch", CONFIRMED OB-028-C, D): Dw is the best
// mass driver's rating; t is 1 when two different slots each hold that
// best driver. ok is false without a mass driver.
func DriverWarp(d engine.Design) (dw, t int, ok bool) {
	slots := 0
	for _, sl := range d.Slots {
		if sl.Count <= 0 || sl.Part.Kind != engine.PartMassDriver {
			continue
		}
		w := driverRating(sl.Part.Name)
		switch {
		case w > dw:
			dw, slots = w, 1
		case w == dw:
			slots++
		}
	}
	if dw == 0 {
		return 0, 0, false
	}
	if slots >= 2 {
		t = 1
	}
	return dw, t, true
}

func driverRating(name string) int {
	c, ok := engine.Components().Lookup(name)
	if !ok {
		return 0
	}
	w, _ := c.Stats["warp"].(float64)
	return int(w)
}

// PacketWarp is a launch's packet warp W (OBJECTS.md "Launch", CONFIRMED
// OB-028-B, C, D): the planet's packet-speed setting, or Dw + t when the
// setting is below 5 or above Dw + 3 (unset counts as below 5).
func PacketWarp(setting, dw, t int) int {
	if setting < 5 || setting > dw+3 {
		return dw + t
	}
	return setting
}

// DecayClass is a packet's decay class (OBJECTS.md "Launch", CONFIRMED
// OB-028-A, H, I): max(0, W − Dw − t), +1 for an Interstellar Traveler
// launcher, at most 3.
func DecayClass(w, dw, t int, it bool) int {
	k := max(0, w-dw-t)
	if it {
		k++
	}
	return min(k, 3)
}

// PacketItem is what one packet item launches and spends per mineral
// (OBJECTS.md "Launch", MEASURED OB-028, OB-029): 100 kT of one mineral
// (Packet Physics 70) for 110 kT (Interstellar Traveler 120, PP 70); a
// mixed item 40 kT of each (PP 25) for 44 kT of each (PP 25).
//
// ASSUMPTION P1: an Interstellar Traveler's mixed item spends 48 kT of
// each (its 120% of the 40 kT launched, as for a single mineral).
func PacketItem(race engine.Race, mixed bool) (launch, spend int) {
	switch {
	case race.PRT == engine.PRTPacketPhysics && mixed:
		return 25, 25
	case race.PRT == engine.PRTPacketPhysics:
		return 70, 70
	case mixed && race.PRT == engine.PRTInterstellarTraveler:
		return 40, 48
	case mixed:
		return 40, 44
	case race.PRT == engine.PRTInterstellarTraveler:
		return 100, 120
	}
	return 100, 110
}

// PacketOrder is a planet's packet destination (a planet ID, or -1) and
// packet-speed setting (0 when unset). The engine's Planet has neither
// yet; production supplies them.
type PacketOrder struct {
	Dest  int
	Speed int
}

// Launch is the outcome of one packet launch.
type Launch struct {
	// NoDriver: no mass driver or no destination; nothing is built and
	// the player gets a message (CONFIRMED OB-028-F).
	NoDriver bool
	Packet   int // index into Space.Packets
	Merged   bool
	Cargo    engine.Minerals // launched
	Spend    engine.Minerals // taken from the surface by production
}

// Launch launches count packet items of mineral (engine.Ironium ..
// engine.Germanium, or Mixed) from planet pi (OBJECTS.md "Launch",
// CONFIRMED OB-028, OB-029; caps and the merge limit BINARY-ONLY). The
// launch needs a mass driver on the planet's starbase and a destination.
// The amount is per item × count, at most 32,760 per mineral. A packet
// launched from the same planet this year with the same warp,
// destination and class takes the cargo while its total is under 16,300
// kT. Production spends Launch.Spend from the surface.
//
// ASSUMPTION P2: the 16,300 kT test is on the earlier packet's total
// before this launch is added. ASSUMPTION P3: a new packet takes its
// owner's lowest unused packet number, from 0; packets stay in object
// order (owner, then number).
func (s *Space) Launch(g *engine.Game, pi int, o PacketOrder, mineral, count int) Launch {
	p := &g.Planets[pi]
	if !p.HasStarbase || o.Dest < 0 || p.Owner < 0 {
		return Launch{NoDriver: true, Packet: -1}
	}
	dw, t, ok := DriverWarp(g.Designs[p.StarbaseDesign])
	if !ok {
		return Launch{NoDriver: true, Packet: -1}
	}
	race := g.Players[p.Owner].Race
	w := PacketWarp(o.Speed, dw, t)
	k := DecayClass(w, dw, t, race.PRT == engine.PRTInterstellarTraveler)
	per, spend := PacketItem(race, mineral == Mixed)
	var l Launch
	for m := range engine.NumMinerals {
		if mineral == Mixed || mineral == m {
			l.Cargo[m] = min(per*count, maxPacketMineral)
			l.Spend[m] = spend * count
		}
	}
	for i := range s.Packets {
		q := &s.Packets[i]
		if q.New && q.From == p.ID && q.Warp == w && q.Target == o.Dest && q.Class == k && total(q.Cargo) < mergeLimit {
			for m := range q.Cargo {
				q.Cargo[m] += l.Cargo[m]
			}
			l.Packet, l.Merged = i, true
			return l
		}
	}
	used := map[int]bool{}
	for _, q := range s.Packets {
		if q.Owner == p.Owner {
			used[q.Number] = true
		}
	}
	n := 0
	for used[n] {
		n++
	}
	at := len(s.Packets)
	for i, q := range s.Packets {
		if q.Owner > p.Owner || (q.Owner == p.Owner && q.Number > n) {
			at = i
			break
		}
	}
	np := Packet{Owner: p.Owner, Number: n, Pos: p.Pos, Target: o.Dest, From: p.ID, Warp: w, Class: k, Cargo: l.Cargo, New: true}
	s.Packets = append(s.Packets[:at], append([]Packet{np}, s.Packets[at:]...)...)
	l.Packet = at
	return l
}

func total(m engine.Minerals) int {
	return m[engine.Ironium] + m[engine.Boranium] + m[engine.Germanium]
}

// decayPct is the yearly decay in percent by class, and for a Packet
// Physics owner (OBJECTS.md "Flight and decay", CONFIRMED OB-003, OB-023).
var decayPct = [2][4]int{{0, 10, 25, 50}, {0, 5, 12, 25}}

// Decay takes share of a year's decay from a packet (OBJECTS.md "Flight
// and decay", CONFIRMED OB-003, OB-023, OB-028): each non-empty mineral
// loses its class's rate × share, at least 10 kT (Packet Physics 5) and
// at most what it has.
//
// ASSUMPTION P4: the loss is ⌊m · rate · share / 100⌋, computed in
// floating point (the vectors agree: 100 kT class 2 for half a year →
// 88, 500 kT class 3 for half of 0.7 → 413).
func Decay(c engine.Minerals, class int, pp bool, share float64) engine.Minerals {
	row, floor := 0, 10
	if pp {
		row, floor = 1, 5
	}
	pct := decayPct[row][class]
	if pct == 0 {
		return c
	}
	for m := range c {
		if c[m] <= 0 {
			continue
		}
		loss := int(float64(c[m]) * float64(pct) * share / 100)
		c[m] -= min(c[m], max(loss, floor))
	}
	return c
}

// ImpactContext supplies what the impact needs from the engine.
type ImpactContext struct {
	// DefenseShare is the share of planet pi's normal-bomb kill that gets
	// through its defenses (TAKEOVER.md), 0..1.
	DefenseShare func(pi int) float64
}

// Impact is one packet's arrival at its target.
type Impact struct {
	Owner, Number int
	Planet        int // planet index
	Caught        int // per mille
	Added         engine.Minerals
	Damage        int // dmg, units of 100 colonists
	Killed        int
	DefensesLost  int
	// Emptied: the damage reached the population; the engine empties the
	// planet as after bombing.
	Emptied bool
	// Unhandled: a Packet Physics launcher's terraforming and design
	// disclosure (OBJECTS.md "Impact" step 4) are not modelled.
	Unhandled bool
}

// CatcherWarp is planet pi's catcher warp C: its own Dw + t, 0 when
// unowned or without a starbase or driver.
func CatcherWarp(g *engine.Game, pi int) int {
	p := &g.Planets[pi]
	if p.Owner < 0 || !p.HasStarbase {
		return 0
	}
	dw, t, _ := DriverWarp(g.Designs[p.StarbaseDesign])
	return dw + t
}

// hit applies a packet's impact on planet pi (OBJECTS.md "Impact",
// CONFIRMED OB-003, OB-009, OB-022, OB-030-A; marked parts BINARY-ONLY).
// With w² = W² and c² = C² (⌊C²/2⌋ for an Interstellar Traveler owner):
// caught q = 1000 when w² ≤ c², else ⌊c²·1000/w²⌋, 0 with no catcher;
// each mineral adds ⌊m·(q + ⌊(1000 − q)/9⌋)/1000⌋. Unless fully caught,
// unowned or Alternate Reality: dmg = ⌊s·⌊(w² − c²)·M/160⌋⌋, kill =
// max(⌊P·dmg/1000⌋, dmg); kill ≥ P empties the planet; otherwise the
// population drops by kill and defenses by Dk = ⌊def·dmg/1000⌋, 1 if that
// is 0 and rand(20) < dmg, then min(def, max(Dk, ⌊dmg/20⌋)). The packet's
// owner is not checked.
func hit(g *engine.Game, ctx ImpactContext, pk Packet, pi int, rng engine.Rand) Impact {
	p := &g.Planets[pi]
	im := Impact{Owner: pk.Owner, Number: pk.Number, Planet: pi}
	w2 := pk.Warp * pk.Warp
	c := CatcherWarp(g, pi)
	c2 := c * c
	if p.Owner >= 0 && g.Players[p.Owner].Race.PRT == engine.PRTInterstellarTraveler {
		c2 /= 2
	}
	q := 0
	switch {
	case w2 <= c2:
		q = 1000
	case c > 0:
		q = c2 * 1000 / w2
	}
	im.Caught = q
	for m := range pk.Cargo {
		im.Added[m] = pk.Cargo[m] * (q + (1000-q)/9) / 1000
		p.Surface[m] += im.Added[m]
	}
	if q == 1000 {
		return im
	}
	if pk.Owner >= 0 && pk.Owner < len(g.Players) && g.Players[pk.Owner].Race.PRT == engine.PRTPacketPhysics {
		im.Unhandled = true
	}
	if p.Owner < 0 || g.Players[p.Owner].Race.PRT == engine.PRTAlternateReality {
		return im
	}
	s := 1.0
	if ctx.DefenseShare != nil {
		s = ctx.DefenseShare(pi)
	}
	dmg0 := (w2 - c2) * total(pk.Cargo) / 160
	dmg := int(s * float64(dmg0))
	im.Damage = dmg
	P := p.Population
	kill := max(P*dmg/1000, dmg)
	if kill >= P {
		im.Killed, im.Emptied = P, true
		p.Population = 0
		return im
	}
	p.Population -= kill
	im.Killed = kill
	def := p.Defenses
	dk := def * dmg / 1000
	if dk == 0 && def > 0 && rng.Intn(20) < dmg {
		dk = 1
	}
	dk = min(def, max(dk, dmg/20))
	p.Defenses -= dk
	im.DefensesLost = dk
	return im
}

// fly moves packet i by move ly toward its target. On arrival it decays
// for the share of the move it flew (half that on its launch year) and
// hits; the packet is then marked for removal.
func (s *Space) fly(g *engine.Game, ctx ImpactContext, i, move int, launchYear bool, rng engine.Rand) (Impact, bool) {
	pk := &s.Packets[i]
	ti := planetByID(g, pk.Target)
	if ti < 0 {
		return Impact{}, false
	}
	dest := g.Planets[ti].Pos
	dx, dy := float64(dest.X-pk.Pos.X), float64(dest.Y-pk.Pos.Y)
	d := math.Sqrt(dx*dx + dy*dy)
	pos, arrived := StepToward(pk.Pos, dest, move)
	pk.Pos = pos
	if !arrived {
		return Impact{}, false
	}
	share := 1.0
	if move > 0 {
		share = min(1, d/float64(move))
	}
	if launchYear {
		share /= 2
	}
	pk.Cargo = Decay(pk.Cargo, pk.Class, s.ppOwner(g, pk.Owner), share)
	return hit(g, ctx, *pk, ti, rng), true
}

func (s *Space) ppOwner(g *engine.Game, owner int) bool {
	return owner >= 0 && owner < len(g.Players) && g.Players[owner].Race.PRT == engine.PRTPacketPhysics
}

func planetByID(g *engine.Game, id int) int {
	for i := range g.Planets {
		if g.Planets[i].ID == id {
			return i
		}
	}
	return -1
}

// MovePackets moves every packet already in flight W² ly toward its
// target (OBJECTS.md "Flight and decay", "Turn placement" step 3,
// CONFIRMED OB-003 J, K), in object order; one that arrives decays for
// the share of the year it flew and hits (CONFIRMED OB-003-C). Arrived
// packets are removed.
func (s *Space) MovePackets(g *engine.Game, ctx ImpactContext, rng engine.Rand) []Impact {
	return s.flyAll(g, ctx, false, rng)
}

// FlyLaunched flies the packets launched this year half a year, ⌊W²/2⌋
// ly (OBJECTS.md "Turn placement" step 7, CONFIRMED OB-028 A–I); one that
// arrives decays for half of the share it flew (OB-028-G) and hits, one
// that does not decays half a year (OB-028). Afterwards no packet is new.
func (s *Space) FlyLaunched(g *engine.Game, ctx ImpactContext, rng engine.Rand) []Impact {
	out := s.flyAll(g, ctx, true, rng)
	for i := range s.Packets {
		if s.Packets[i].New {
			s.Packets[i].Cargo = Decay(s.Packets[i].Cargo, s.Packets[i].Class, s.ppOwner(g, s.Packets[i].Owner), 0.5)
			s.Packets[i].New = false
		}
	}
	return out
}

func (s *Space) flyAll(g *engine.Game, ctx ImpactContext, launched bool, rng engine.Rand) []Impact {
	var out []Impact
	gone := map[int]bool{}
	for i := range s.Packets {
		pk := s.Packets[i]
		if pk.New != launched {
			continue
		}
		move := pk.Warp * pk.Warp
		if launched {
			move /= 2
		}
		if im, ok := s.fly(g, ctx, i, move, launched, rng); ok {
			out = append(out, im)
			gone[i] = true
		}
	}
	if len(gone) > 0 {
		kept := s.Packets[:0]
		for i, pk := range s.Packets {
			if !gone[i] {
				kept = append(kept, pk)
			}
		}
		s.Packets = kept
	}
	return out
}

// DecayPackets decays every packet in flight for a full year (OBJECTS.md
// "Turn placement" step 5, CONFIRMED OB-003 J, K, OB-023).
func (s *Space) DecayPackets(g *engine.Game) {
	for i := range s.Packets {
		pk := &s.Packets[i]
		pk.Cargo = Decay(pk.Cargo, pk.Class, s.ppOwner(g, pk.Owner), 1)
	}
}
