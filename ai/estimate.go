package ai

import "github.com/bfaber-centaur/elegy/engine"

// estimateYears is the most years a completion estimate runs (ESTIMATES.md
// "Production completion", CONFIRMED ES-001); a head not finished by then
// completes in year 100.
const estimateYears = 99

// estItem is one queue entry of an estimate; head marks the item whose
// completion is estimated.
type estItem struct {
	engine.QueueItem
	head bool
}

// estimate is the completion-estimate copy of a planet: its stock,
// installations and population, and the queue up to the head.
type estimate struct {
	v        *View
	col      engine.Colony
	p        engine.Planet
	q        []estItem
	r        int
	stopped  bool
	finished bool
}

// completion is the year (1 is the coming year) in which the head of
// queue q finishes, by ESTIMATES.md "Production completion" (CONFIRMED,
// ES-001): mining without the random extra kT, the year's resources less
// the research budget (unless the planet sends only leftover resources to
// research), the queue walk with mines and factories installed at once and
// no installation cap, then population growth. It is 100 when the head
// does not finish in 99 years.
//
// ASSUMPTION A62: the walk stops at the head, so the items behind it are
// left out (they matter only after a year in which an automatic head was
// skipped); the head finishes when its count reaches 0, or when an
// automatic head builds its whole count in a year; a starbase costs its
// full build cost (as A12); a packet head is built as a plain item.
func (v *View) completion(p *engine.Planet, q []engine.QueueItem, budget int) int {
	e := &estimate{v: v, col: engine.NewColony(p, &v.Self), p: *p}
	for i, it := range q {
		e.q = append(e.q, estItem{it, i == len(q)-1})
	}
	for y := 1; y <= estimateYears; y++ {
		e.year(budget)
		if e.finished {
			return y
		}
	}
	return estimateYears + 1
}

func (e *estimate) year(budget int) {
	race := e.v.Self.Race
	eff := race.MineOutput
	if race.PRT == engine.PRTAlternateReality {
		eff = 10
	}
	working := e.col.WorkingMines(e.p.Population, e.p.Mines)
	for m := range engine.NumMinerals {
		var gain int
		gain, e.p.Deposits[m] = engine.MineYear(e.p.Deposits[m], working, eff, e.p.Homeworld, noExtra{})
		e.p.Surface[m] += gain
	}
	r := e.col.Resources(e.p.Population, e.p.Factories)
	e.r = r
	if !e.p.LeftoverOnly {
		e.r -= r * budget / 100
	}
	e.stopped = false
	e.walk()
	g := race.GrowthRate
	if race.PRT == engine.PRTHyperExpansion {
		g *= 2
	}
	e.p.Population, e.p.GrowthCarry = engine.GrowPopulation(e.p.Population, e.p.GrowthCarry, e.col.MaxPop, g, e.col.Hab)
}

// walk is the year's queue walk (KERNEL.md "Production"; "Auto Alchemy
// before a multi-count item") over the copy.
func (e *estimate) walk() {
	for i := 0; i < len(e.q) && !e.stopped && !e.finished; {
		it := e.q[i]
		switch {
		case it.Kind == engine.ItemAutoAlchemy:
			if i == len(e.q)-1 {
				return // never the head (blockedQueue)
			}
			if e.q[i+1].Kind == engine.ItemAutoAlchemy {
				i++
				continue
			}
			if isAuto(e.q[i+1].Kind) {
				i = e.auto(i+1, true)
				continue
			}
			if e.plain(i+1, true) {
				e.remove(i + 1)
				e.remove(i)
				continue
			}
			i += 2
		case isAuto(it.Kind):
			i = e.auto(i, false)
		default:
			if e.plain(i, false) {
				e.remove(i)
			} else {
				i++
			}
		}
	}
}

func (e *estimate) remove(i int) { e.q = append(e.q[:i:i], e.q[i+1:]...) }

func (e *estimate) front(it engine.QueueItem) {
	e.q = append([]estItem{{QueueItem: it}}, e.q...)
}

// plain builds the non-automatic item at i and reports whether its count
// reached 0; behind an Auto Alchemy prefix a unit short of minerals buys
// them.
func (e *estimate) plain(i int, alch bool) bool {
	it := &e.q[i]
	c := e.v.itemCost(it.QueueItem)
	built := 0
	for it.Count > 0 {
		if !e.affordable(c, it.Percent) {
			short, byRes := e.limiting(c, it.Percent)
			it.Percent = e.partial(c, it.Percent)
			if alch && !byRes && e.buy(short) {
				continue
			}
			if alch && !byRes {
				e.remainder()
			}
			e.stopped = true
			break
		}
		e.pay(c, it.Percent, 100)
		it.Percent = 0
		it.Count--
		built++
	}
	head := it.head
	e.complete(it.Kind, built)
	if head && it.Count == 0 {
		e.finished = true
	}
	return it.Count == 0
}

// auto builds the automatic item at i, with no installation cap, and
// returns the next index.
func (e *estimate) auto(i int, alch bool) int {
	it := e.q[i]
	c := e.v.itemCost(it.QueueItem)
	built := 0
	for built < it.Count {
		if e.affordable(c, 0) {
			e.pay(c, 0, 100)
			built++
			continue
		}
		short, _ := e.limiting(c, 0)
		if e.mineralShort(c) {
			if !alch {
				break // skipped; the walk continues
			}
			if e.buy(short) {
				continue
			}
			e.complete(it.Kind, built)
			e.remainder()
			e.stopped = true
			return i + 1
		}
		pct := e.partial(c, 0)
		e.complete(it.Kind, built)
		e.front(engine.QueueItem{Kind: realKind(it.Kind), Count: 1, Percent: pct})
		e.stopped = true
		return i + 2
	}
	e.complete(it.Kind, built)
	if it.head && built == it.Count {
		e.finished = true
	}
	return i + 1
}

// isAuto reports whether k is an automatic item (KERNEL.md "Production").
func isAuto(k engine.ItemKind) bool {
	return realKind(k) != k || k == engine.ItemAutoMinTerraform || k == engine.ItemAutoMaxTerraform
}

// realKind is the plain item an automatic item builds.
func realKind(k engine.ItemKind) engine.ItemKind {
	switch k {
	case engine.ItemAutoMines:
		return engine.ItemMine
	case engine.ItemAutoFactories:
		return engine.ItemFactory
	case engine.ItemAutoDefenses:
		return engine.ItemDefenses
	case engine.ItemAutoAlchemy:
		return engine.ItemMineralAlchemy
	case engine.ItemAutoPackets:
		return engine.ItemMixedPacket
	}
	return k
}

// complete installs mines and factories at once and adds alchemy's
// minerals; other units change nothing the estimate reads.
func (e *estimate) complete(k engine.ItemKind, n int) {
	switch realKind(k) {
	case engine.ItemMine:
		e.p.Mines += n
	case engine.ItemFactory:
		e.p.Factories += n
	case engine.ItemMineralAlchemy:
		for m := range engine.NumMinerals {
			e.p.Surface[m] += n
		}
	}
}

func spentOf(c engine.Cost, pct int) engine.Cost {
	s := engine.Cost{Resources: c.Resources * pct / 100}
	for m := range engine.NumMinerals {
		s.Minerals[m] = c.Minerals[m] * pct / 100
	}
	return s
}

func (e *estimate) affordable(c engine.Cost, pct int) bool {
	s := spentOf(c, pct)
	if c.Resources-s.Resources > e.r {
		return false
	}
	for m := range engine.NumMinerals {
		if c.Minerals[m]-s.Minerals[m] > e.p.Surface[m] {
			return false
		}
	}
	return true
}

func (e *estimate) mineralShort(c engine.Cost) bool {
	for m := range engine.NumMinerals {
		if c.Minerals[m] > e.p.Surface[m] {
			return true
		}
	}
	return false
}

func (e *estimate) pay(c engine.Cost, from, to int) {
	a, b := spentOf(c, from), spentOf(c, to)
	e.r -= b.Resources - a.Resources
	for m := range engine.NumMinerals {
		e.p.Surface[m] -= b.Minerals[m] - a.Minerals[m]
	}
}

// largestPct is KERNEL.md "Production"'s partial percentage for one
// component (PARITY.md PQ-001; KX-001 M4).
func largestPct(cost, avail int) int {
	return max((avail+1)*100/cost-1, avail*100/cost)
}

func (e *estimate) partial(c engine.Cost, pct int) int {
	s := spentOf(c, pct)
	p := 100
	if c.Resources > 0 {
		p = min(p, largestPct(c.Resources, e.r+s.Resources))
	}
	for m := range engine.NumMinerals {
		if c.Minerals[m] > 0 {
			p = min(p, largestPct(c.Minerals[m], e.p.Surface[m]+s.Minerals[m]))
		}
	}
	e.pay(c, pct, p)
	return p
}

// limiting is the lowest component's shortfall and whether resources were
// strictly the lowest (KERNEL.md "Production": Fe, Bo, Ge, then
// resources; a later one replaces only when strictly lower).
func (e *estimate) limiting(c engine.Cost, pct int) (short int, byRes bool) {
	s := spentOf(c, pct)
	best := -1
	for m := range engine.NumMinerals {
		if c.Minerals[m] == 0 {
			continue
		}
		avail := e.p.Surface[m] + s.Minerals[m]
		if p := min(100, largestPct(c.Minerals[m], avail)); best < 0 || p < best {
			best, short = p, c.Minerals[m]-avail
		}
	}
	if c.Resources > 0 {
		avail := e.r + s.Resources
		if p := min(100, largestPct(c.Resources, avail)); best < 0 || p < best {
			short, byRes = c.Resources-avail, true
		}
	}
	return short, byRes
}

// buy is an Auto Alchemy prefix buying up to short kT of each mineral.
func (e *estimate) buy(short int) bool {
	rate := max(1, engine.ItemCost(e.v.Self.Race, engine.ItemMineralAlchemy).Resources)
	k := min(e.r/rate, short)
	e.r -= k * rate
	for m := range engine.NumMinerals {
		e.p.Surface[m] += k
	}
	return k == short
}

// remainder leaves the resources left as a Mineral Alchemy ×1 partial at
// the front.
func (e *estimate) remainder() {
	if e.r > 0 {
		c := engine.ItemCost(e.v.Self.Race, engine.ItemMineralAlchemy)
		e.front(engine.QueueItem{Kind: engine.ItemMineralAlchemy, Count: 1, Percent: e.partial(c, 0)})
	}
}
