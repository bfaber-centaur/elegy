package ai

import (
	"math"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// packetLegacy holds the two LEGACY BUGs of Cybertron's packets
// (cybertron.md §6): the scanner shot tests and sets the mark of the
// planet one id higher than its destination, and for w ≥ 14 its
// distance test passes every planet. Both come from the game's ruleset
// (engine.Legacy CybertronPacketMarkNextID, CybertronScannerShotOverflow):
// off in Elegy's rules, on in jrc3-faithful.
type packetLegacy struct {
	markNextID, overflow bool
}

func packetLegacyOf(r engine.Ruleset) packetLegacy {
	return packetLegacy{markNextID: r.Legacy.CybertronPacketMarkNextID, overflow: r.Legacy.CybertronScannerShotOverflow}
}

// packets is cybertron.md §6 (AI-24): own planets with a starbase at the
// start of the step, in the shuffled order, each try supply, then attack,
// then the scanner shot, and do at most one. Packet marks start clear
// every turn (the empty memory, AI.md §1).
func (t *cyberTurn) packets(order []int, budget int, legacy packetLegacy) {
	v := t.v
	marked := map[int]bool{}
	var shooters []*engine.Planet
	for _, id := range order {
		if p := v.ownPlanet(id); p != nil && p.HasStarbase {
			shooters = append(shooters, p)
		}
	}
	for _, p := range shooters {
		if t.supply(p, budget, marked) || t.attackPacket(p, budget, marked) {
			continue
		}
		t.scannerShot(p, budget, marked, legacy)
	}
}

// driver is the planet's starbase driver rating r and two-driver bonus
// (OBJECTS.md "Launch"; objects.DriverWarp); 0, 0 without a driver.
func (t *cyberTurn) driver(p *engine.Planet) (r, two int) {
	if !p.HasStarbase {
		return 0, 0
	}
	d, ok := t.v.design(p.StarbaseDesign)
	if !ok {
		return 0, 0
	}
	r, two, _ = objects.DriverWarp(d)
	return r, two
}

// left is the planet's available minerals and resources less the cost of
// its queue (AI.md §7).
func (t *cyberTurn) left(p *engine.Planet, budget int) funds {
	f := t.v.available(p, budget)
	c := t.v.queueCost(t.q.get(p))
	for m := range engine.NumMinerals {
		f.Minerals[m] -= c.Minerals[m]
	}
	f.Resources -= c.Resources
	return f
}

// aim sets the planet's packet destination at warp w, keeping its other
// settings.
//
// ASSUMPTION A52: without a mass driver (r = 0, w = 3) the packet speed is
// left unset: the order cannot carry warp 3 (engine ASSUMPTION L19), and
// such a planet launches nothing.
func (t *cyberTurn) aim(p *engine.Planet, dest, w int) {
	if w < 4 {
		w = 0
	}
	p.HasPacketDest, p.PacketDest, p.PacketSpeed = true, dest, w
	t.res.Orders = append(t.res.Orders, engine.PlanetSettingsOrder{Planet: p.ID, LeftoverOnly: p.LeftoverOnly,
		HasRoute: p.HasRoute, RouteTo: p.RouteTo, HasPacketDest: true, PacketDest: dest, PacketSpeed: w})
}

// queuePackets puts n packets of mineral m at the front of the queue.
func (t *cyberTurn) queuePackets(p *engine.Planet, m, n int) {
	t.q.add(p, engine.QueueItem{Kind: engine.ItemIroniumPacket + engine.ItemKind(m), Count: n}, true)
}

// low reports the §4.1 low-mineral note of an own starbase planet:
// below 10 kT under an Orbital Fort, else below 1,000 kT.
func (t *cyberTurn) low(p *engine.Planet, m int) bool {
	if !p.HasStarbase {
		return false
	}
	limit := 1000
	if sb, ok := t.v.design(p.StarbaseDesign); ok && sb.Hull.Name == orbitalFort {
		limit = 10
	}
	return p.Surface[m] < limit
}

// supply is §6's supply rule. Not exercised in AIX.
//
// ASSUMPTION A48: the minerals are tried in order ironium, boranium,
// germanium, the first with a target wins; "over 700 kT" and "at least 70
// resources" are the amounts left after the queue; the low notes read the
// target's surface minerals; the shooting planet is not its own target;
// "up to 7" packets is as many as the amount left pays for, at most 7; the
// destination's speed is w.
func (t *cyberTurn) supply(p *engine.Planet, budget int, marked map[int]bool) bool {
	v := t.v
	sb, ok := v.design(p.StarbaseDesign)
	if !ok || sb.Hull.Name != orbitalFort {
		return false
	}
	f := t.left(p, budget)
	if f.Resources < 70 {
		return false
	}
	r, two := t.driver(p)
	for m := range engine.NumMinerals {
		if f.Minerals[m] <= 700 {
			continue
		}
		best, bd := -1, 0
		for i := range v.Planets {
			q := &v.Planets[i]
			if q.ID == p.ID || marked[q.ID] || !t.low(q, m) {
				continue
			}
			qr, qtwo := t.driver(q)
			k := min(r+two, qr+qtwo)
			reach := 3.5 * float64(k*k)
			dd := d2(p.Pos, q.Pos)
			if float64(dd) > reach*reach {
				continue
			}
			if best < 0 || dd < bd {
				best, bd = q.ID, dd
			}
		}
		if best < 0 {
			continue
		}
		per := v.itemCost(engine.QueueItem{Kind: engine.ItemIroniumPacket + engine.ItemKind(m), Count: 1}).Minerals[m]
		n := min(7, f.Minerals[m]/max(1, per))
		t.aim(p, best, r+3)
		t.queuePackets(p, m, n)
		return true
	}
	return false
}

// attackPacket is §6's attack rule. With need the truncated kill mass
// and f = q^(D/w²) (D in ly, the share of the mass that arrives), a target
// qualifies when need ≤ trunc(C·f), C = min(M, 70·((R/2 − 5)/5))
// (BINARY-ONLY); the mass sent is A = trunc(min(need, M)/f), as ⌈A/70⌉
// packets (MEASURED, AI-24).
//
// ASSUMPTION A49: M is the sum of the three minerals left after the queue
// and R the available resources before it (§6 names the queue only for
// M); f is computed in floating point; the range test is inclusive.
//
// ASSUMPTION A50: the report does not carry the parts of another
// player's starbase design, so its catch warp cannot be read: only planets
// with no starbase in view are targets.
func (t *cyberTurn) attackPacket(p *engine.Planet, budget int, marked map[int]bool) bool {
	v := t.v
	switch {
	case v.Level == Easy:
		return false
	case v.Level == Standard && t.rng.Intn(3) != 0:
		return false
	}
	f := t.left(p, budget)
	M := f.Minerals[0] + f.Minerals[1] + f.Minerals[2] - 210
	if M <= 150 {
		return false
	}
	R := v.available(p, budget).Resources
	C := min(M, 70*((R/2-5)/5))
	r, two := t.driver(p)
	w := r + 3
	q := 0.75
	if two > 0 {
		q = 0.875
	}
	reach := 2.5 * float64(w*w)
	best, bd, bn := -1, 0, 0
	for _, pp := range v.Universe {
		o := v.owner(pp.ID)
		rep, known := v.Known[pp.ID]
		if !known || o == engine.NoOwner || o == v.Player || marked[pp.ID] || v.PRT[o] == engine.PRTAlternateReality {
			continue
		}
		if rep.PopEstimate <= 0 || rep.Starbase {
			continue
		}
		dd := d2(p.Pos, pp.Pos)
		dist := math.Sqrt(float64(dd))
		if dist > reach {
			continue
		}
		pop := rep.PopEstimate / 400
		c := 0 // no starbase, no catcher
		need := 16000 * min(1000, 4*(pop+25)) / ((w*w - c*c) * (95 - rep.DefenseEstimate))
		share := math.Pow(q, dist/float64(w*w))
		if need > int(float64(C)*share) {
			continue
		}
		if best < 0 || dd < bd {
			best, bd = pp.ID, dd
			A := int(float64(min(need, M)) / share)
			bn = (A + 69) / 70
		}
	}
	if best < 0 {
		return false
	}
	t.aim(p, best, w)
	// Each packet is of the mineral with the most left.
	var items []engine.QueueItem
	for range bn {
		m := mostLeft(f.Minerals)
		f.Minerals[m] -= 70
		k := engine.ItemIroniumPacket + engine.ItemKind(m)
		if n := len(items); n > 0 && items[n-1].Kind == k {
			items[n-1].Count++
		} else {
			items = append(items, engine.QueueItem{Kind: k, Count: 1})
		}
	}
	for i := len(items) - 1; i >= 0; i-- {
		t.q.add(p, items[i], true)
	}
	marked[best] = true
	return true
}

// mostLeft is the mineral with the most left, ties to ironium, then
// boranium.
func mostLeft(m engine.Minerals) int {
	best := 0
	for i := 1; i < engine.NumMinerals; i++ {
		if m[i] > m[best] {
			best = i
		}
	}
	return best
}

// scannerShot is §6's scanner shot (AI-24): a point near a map edge in one
// of six directions from the planet, and a packet to the planet nearest
// it, which Cybertron's packet scanners then see. Positions are relative
// to the map's corner (1000, 1000).
func (t *cyberTurn) scannerShot(p *engine.Planet, budget int, marked map[int]bool, legacy packetLegacy) {
	v := t.v
	W := galaxyWidth(v)
	r, _ := t.driver(p)
	w := r + 3
	x, y := p.Pos.X-1000, p.Pos.Y-1000

	// 1. Direction: 1..6, with the empty memory's 0 read as 1.
	dir := t.rng.Intn(7)
	if dir == 0 {
		dir = 1
	}
	// 2. Edge point.
	switch dir {
	case 1:
		if W-x > y {
			x, y = x+y, 0
		} else {
			x, y = W, y-(W-x)
		}
	case 2:
		y = 0
	case 3:
		if y < x {
			x, y = x-y, 0
		} else {
			x, y = 0, y-x
		}
	case 4:
		x = 0
	case 5:
		if W-x > y {
			x, y = 0, y+x
		} else {
			x, y = x-(W-y), W
		}
	case 6:
		y = W
	}
	// 3. Slide along the edge, past a corner onto the next edge.
	j := t.rng.Intn(3*W/10) - 3*W/20
	slide := func(along, across *int) {
		*along += j
		e := 0
		switch {
		case *along > W:
			e, *along = *along-W, W
		case *along < 0:
			e, *along = -*along, 0
		}
		if *across == 0 {
			*across = e
		} else {
			*across = W - e
		}
	}
	if x == 0 || x == W {
		slide(&y, &x)
	} else {
		slide(&x, &y)
	}
	// 4. Inset.
	k := t.rng.Intn(w * w)
	switch {
	case y == 0:
		y = k
	case y == W:
		y = W - k
	case x == 0:
		x = k
	case x == W:
		x = W - k
	}
	// 5. Destination: the nearest planet, ties to the lower id.
	pt := engine.Point{X: x + 1000, Y: y + 1000}
	dest, dd := -1, 0
	for _, pp := range v.Universe {
		if d := d2(pt, pp.Pos); dest < 0 || d < dd || (d == dd && pp.ID < dest) {
			dest, dd = pp.ID, d
		}
	}
	if dest < 0 || v.owner(dest) == v.Player {
		return
	}
	pos, _ := v.planetPos(dest)
	if d := d2(p.Pos, pos); d < w*w*w*w && !(legacy.overflow && w >= 14) {
		return
	}
	// 6. Shot.
	t.aim(p, dest, w)
	mark := dest
	if legacy.markNextID {
		mark = dest + 1
	}
	if marked[mark] {
		return
	}
	f := t.left(p, budget)
	m := mostLeft(f.Minerals)
	if f.Minerals[m] < 170 {
		return
	}
	t.queuePackets(p, m, 1)
	marked[mark] = true
}

// galaxyWidth is W (UNIVERSE.md): (size + 1) · 400, with the size read
// from the map's extent (ASSUMPTION A36).
func galaxyWidth(v *View) int { return (universeSize(v) + 1) * 400 }
