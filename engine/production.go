package engine

// ItemKind is a production-queue item type. Ships, starbases, terraforming,
// packets and scanners are not modelled yet.
type ItemKind int

const (
	ItemMine ItemKind = iota
	ItemFactory
	ItemDefenses
	ItemMineralAlchemy
	ItemAutoMines
	ItemAutoFactories
	ItemAutoDefenses
	ItemAutoAlchemy
)

func (k ItemKind) auto() bool { return k >= ItemAutoMines }

// real is the item an auto item builds.
func (k ItemKind) real() ItemKind {
	switch k {
	case ItemAutoMines:
		return ItemMine
	case ItemAutoFactories:
		return ItemFactory
	case ItemAutoDefenses:
		return ItemDefenses
	case ItemAutoAlchemy:
		return ItemMineralAlchemy
	}
	return k
}

// QueueItem is one production-queue entry. Percent is the completion of the
// first unit; the amount already spent on it is trunc(cost·Percent/100) per
// component.
type QueueItem struct {
	Kind    ItemKind
	Count   int
	Percent int
}

// ItemCost is the per-unit cost of an item for a race (KERNEL.md "Item
// costs", CONFIRMED by PQ-001 and KX-001): factories at the race's cost +
// 4 kT germanium (3 with "factories cost 1 kT less germanium"), mines at
// the race's cost, defenses 15 + 5/5/5 (Inner Strength: 3/5 of each
// component), alchemy 100 resources (25 with Mineral Alchemy).
func ItemCost(race Race, k ItemKind) Cost {
	switch k.real() {
	case ItemMine:
		return Cost{Resources: race.MineCost}
	case ItemFactory:
		ge := 4
		if race.FactoryLessGermanium {
			ge = 3
		}
		return Cost{Resources: race.FactoryCost, Minerals: Minerals{0, 0, ge}}
	case ItemDefenses:
		c := Cost{Resources: 15, Minerals: Minerals{5, 5, 5}}
		if race.PRT == PRTInnerStrength {
			c.Resources = c.Resources * 3 / 5
			for m := range NumMinerals {
				c.Minerals[m] = c.Minerals[m] * 3 / 5
			}
		}
		return c
	}
	if race.LRT.MineralAlchemy {
		return Cost{Resources: 25}
	}
	return Cost{Resources: 100}
}

// EventKind identifies a turn message.
type EventKind int

const (
	EventBuilt EventKind = iota // Item, Count
	EventQueueCompleted
	EventOrderClipped // Item, Count = new count
	EventAlchemy      // Count = kT of each mineral
	EventFleetArrived
	EventOutOfFuel
	EventRamScoopFuel // Count = mg
)

// Event is a turn message. Wording is not modelled.
type Event struct {
	Kind   EventKind
	Player int
	Planet int // planet id, or -1
	Fleet  int // fleet id, or -1
	Item   ItemKind
	Count  int
}

// ProductionInput is everything one planet's production reads.
type ProductionInput struct {
	Colony         Colony
	Resources      int // this year's resources
	GrownPop       int // population after this year's growth (for caps)
	ResearchBudget int // percent
	LeftoverOnly   bool
}

// production is the mutable state of one planet's production run.
type production struct {
	in       ProductionInput
	planet   *Planet
	r        int // resources left
	events   []Event
	stopped  bool
	blocked  bool // an auto item was skipped for minerals
	research int
	bought   int // alchemy units bought by an Auto Alchemy prefix
}

// RunProduction runs one planet's production queue for the year and returns
// the resources it sends to research.
//
// KERNEL.md "Production" and PARITY.md "PQ-001": CONFIRMED in all 15 PQ-001
// cases. The empty-queue and zero-resource rules are BINARY-ONLY.
func RunProduction(p *Planet, in ProductionInput) (research int, events []Event) {
	if !p.HasQueue {
		return in.Resources, nil
	}
	if len(p.Queue) == 0 {
		return 0, nil // BINARY-ONLY: an empty queue contributes nothing, not even the tax
	}
	if in.Resources <= 0 {
		return 0, nil // BINARY-ONLY: builds nothing, sends no messages
	}
	pr := &production{in: in, planet: p, r: in.Resources}
	if !in.LeftoverOnly {
		tax := pr.r * in.ResearchBudget / 100
		pr.r -= tax
		pr.research += tax
	}
	pr.walk()
	pr.research += pr.r
	if len(p.Queue) == 0 {
		p.HasQueue = false
		p.Queue = nil
		pr.event(EventQueueCompleted, 0, 0)
	} else if !pr.stopped && !pr.blocked {
		pr.event(EventQueueCompleted, 0, 0)
	}
	return pr.research, pr.events
}

func (pr *production) event(kind EventKind, item ItemKind, count int) {
	pr.events = append(pr.events, Event{Kind: kind, Player: pr.planet.Owner, Planet: pr.planet.ID, Fleet: -1, Item: item, Count: count})
}

func (pr *production) cost(k ItemKind) Cost {
	return ItemCost(pr.in.Colony.Race, k)
}

func (pr *production) installed(k ItemKind) *int {
	switch k.real() {
	case ItemMine:
		return &pr.planet.Mines
	case ItemFactory:
		return &pr.planet.Factories
	case ItemDefenses:
		return &pr.planet.Defenses
	}
	return nil
}

// operable is the installation count the post-growth population operates.
func (pr *production) operable(k ItemKind) int {
	c, pop := pr.in.Colony, pr.in.GrownPop
	switch k.real() {
	case ItemMine:
		return c.OperableMines(pop)
	case ItemFactory:
		return c.OperableFactories(pop)
	}
	return c.OperableDefenses(pop)
}

func (pr *production) maximum(k ItemKind) int {
	c := pr.in.Colony
	switch k.real() {
	case ItemMine:
		return c.MaxMines()
	case ItemFactory:
		return c.MaxFactories()
	}
	return c.MaxDefenses()
}

// spent is what a unit at pct percent has already consumed.
func spent(c Cost, pct int) Cost {
	s := Cost{Resources: c.Resources * pct / 100}
	for m := range NumMinerals {
		s.Minerals[m] = c.Minerals[m] * pct / 100
	}
	return s
}

// affordable reports whether the rest of a unit at pct can be paid now.
func (pr *production) affordable(c Cost, pct int) bool {
	s := spent(c, pct)
	if c.Resources-s.Resources > pr.r {
		return false
	}
	for m := range NumMinerals {
		if c.Minerals[m]-s.Minerals[m] > pr.planet.Surface[m] {
			return false
		}
	}
	return true
}

// pay moves a unit from from% to to% (100 completes it).
func (pr *production) pay(c Cost, from, to int) {
	a, b := spent(c, from), spent(c, to)
	pr.r -= b.Resources - a.Resources
	for m := range NumMinerals {
		pr.planet.Surface[m] -= b.Minerals[m] - a.Minerals[m]
	}
}

// largestPercent is the partial percentage one component pays for, where
// avail includes what is already spent: max(trunc((a+1)·100/c) − 1,
// trunc(a·100/c)) (PARITY.md PQ-001 model). Where c does not divide
// (a+1)·100 this is one less than the largest p with trunc(c·p/100) ≤ a
// (CONFIRMED, KX-001 M4: 4 of 9 resources → 54%, not 55%).
func largestPercent(cost, avail int) int {
	return max((avail+1)*100/cost-1, avail*100/cost)
}

// partial charges a unit at pct as far as current stock allows and returns
// its new percentage: the minimum over components of the largest whole
// percentage whose truncated cost fits.
func (pr *production) partial(c Cost, pct int) int {
	s := spent(c, pct)
	p := 100
	if c.Resources > 0 {
		p = min(p, largestPercent(c.Resources, pr.r+s.Resources))
	}
	for m := range NumMinerals {
		if c.Minerals[m] > 0 {
			p = min(p, largestPercent(c.Minerals[m], pr.planet.Surface[m]+s.Minerals[m]))
		}
	}
	pr.pay(c, pct, p)
	return p
}

// complete applies a finished unit of kind k.
func (pr *production) complete(k ItemKind, n int) {
	if n == 0 {
		return
	}
	if inst := pr.installed(k); inst != nil {
		*inst += n
		pr.event(EventBuilt, k.real(), n)
		return
	}
	for m := range NumMinerals {
		pr.planet.Surface[m] += n
	}
	pr.event(EventAlchemy, ItemMineralAlchemy, n)
}

func (pr *production) insertFront(it QueueItem) {
	q := pr.planet.Queue
	pr.planet.Queue = append([]QueueItem{it}, q...)
}

func (pr *production) remove(i int) {
	q := pr.planet.Queue
	pr.planet.Queue = append(q[:i:i], q[i+1:]...)
}

func (pr *production) walk() {
	for i := 0; i < len(pr.planet.Queue) && !pr.stopped; {
		it := pr.planet.Queue[i]
		switch {
		case it.Kind == ItemAutoAlchemy:
			i = pr.autoAlchemy(i)
		case it.Kind.auto():
			i = pr.autoInstall(i, false)
		default:
			if pr.plain(i, false) {
				pr.remove(i)
			} else {
				i++
			}
		}
	}
}

// plain processes a non-auto item at index i and reports whether it
// finished (count reached 0). Installation orders are first cut to
// max(maximum, operable) − installed (KERNEL.md "Caps", CONFIRMED PQ C10).
// A unit that cannot be finished becomes partial and stops the queue;
// behind an Auto Alchemy prefix (alch), a unit not limited strictly by
// resources first takes its partial and then buys its shortfall.
func (pr *production) plain(i int, alch bool) bool {
	it := &pr.planet.Queue[i]
	if inst := pr.installed(it.Kind); inst != nil {
		limit := max(pr.maximum(it.Kind), pr.operable(it.Kind)) - *inst
		if it.Count > limit {
			it.Count = max(0, limit)
			pr.event(EventOrderClipped, it.Kind, it.Count)
		}
	}
	c := pr.cost(it.Kind)
	built := 0
	for it.Count > 0 {
		if !pr.affordable(c, it.Percent) {
			l := pr.limiting(c, it.Percent)
			it.Percent = pr.partial(c, it.Percent)
			if alch && !l.resources && pr.buy(l.short) {
				continue // retry the unit; it now completes
			}
			if alch && !l.resources {
				pr.alchemyRemainder()
			}
			pr.stopped = true
			break
		}
		pr.pay(c, it.Percent, 100)
		it.Percent = 0
		it.Count--
		built++
	}
	pr.complete(it.Kind, built)
	return it.Count == 0
}

// autoInstall processes Auto Mines/Factories/Defenses at index i and returns
// the next index. They build at most operable − installed (KERNEL.md
// "Caps", CONFIRMED PQ C04, C09, C13, C14). Behind an Auto Alchemy prefix
// (alch), a mineral-short unit buys the lowest component's shortfall
// without a partial.
func (pr *production) autoInstall(i int, alch bool) int {
	it := pr.planet.Queue[i]
	c := pr.cost(it.Kind)
	limit := min(it.Count, pr.operable(it.Kind)-*pr.installed(it.Kind))
	built := 0
	for built < limit {
		if pr.affordable(c, 0) {
			pr.pay(c, 0, 100)
			built++
			continue
		}
		// Any short mineral blocks an auto item, even when resources give
		// the lower percentage (CONFIRMED, KX-001 A6, A7).
		if l := pr.limiting(c, 0); l.mineral {
			if !alch {
				pr.blocked = true // skipped; the walk continues
				break
			}
			if pr.buy(l.short) {
				continue
			}
			pr.complete(it.Kind, built)
			pr.alchemyRemainder()
			pr.stopped = true
			return i + 1
		}
		pct := pr.partial(c, 0)
		pr.complete(it.Kind, built)
		pr.insertFront(QueueItem{Kind: it.Kind.real(), Count: 1, Percent: pct})
		pr.stopped = true
		return i + 2
	}
	pr.complete(it.Kind, built)
	return i + 1
}

// limit describes what stops a unit at pct from completing (KERNEL.md
// "Production", stars-elegy #18): components are compared Fe, Bo, Ge, then
// resources, and one replaces the current lowest percentage only if
// strictly lower.
type limit struct {
	short     int  // the lowest component's shortfall: cost − available − spent
	resources bool // resources were strictly the lowest
	mineral   bool // some mineral is short
}

func (pr *production) limiting(c Cost, pct int) limit {
	s := spent(c, pct)
	var l limit
	best := -1
	for m := range NumMinerals {
		if c.Minerals[m] == 0 {
			continue
		}
		avail := pr.planet.Surface[m] + s.Minerals[m]
		if avail < c.Minerals[m] {
			l.mineral = true
		}
		if p := min(100, largestPercent(c.Minerals[m], avail)); best < 0 || p < best {
			best = p
			l.short = c.Minerals[m] - avail
		}
	}
	if c.Resources > 0 {
		avail := pr.r + s.Resources
		if p := min(100, largestPercent(c.Resources, avail)); best < 0 || p < best {
			l.short = c.Resources - avail
			l.resources = true
		}
	}
	return l
}

// buy is an Auto Alchemy prefix buying up to short units with the
// resources left, each 1 kT of every mineral. It reports whether it bought
// all of them.
func (pr *production) buy(short int) bool {
	rate := pr.cost(ItemMineralAlchemy).Resources
	k := min(pr.r/rate, short)
	pr.r -= k * rate
	for m := range NumMinerals {
		pr.planet.Surface[m] += k
	}
	pr.bought += k
	return k == short
}

// alchemyRemainder turns the resources left into a Mineral Alchemy ×1
// partial at the queue front.
func (pr *production) alchemyRemainder() {
	if pr.r > 0 {
		pct := pr.partial(pr.cost(ItemMineralAlchemy), 0)
		pr.insertFront(QueueItem{Kind: ItemMineralAlchemy, Count: 1, Percent: pct})
	}
}

// autoAlchemy processes Auto Alchemy at index i and returns the next index.
// KERNEL.md "Auto Alchemy before a multi-count item" (CONFIRMED, PQ-001
// C06/C07, KX-001): as a prefix it does nothing itself and lets the next
// item buy minerals one unit at a time.
func (pr *production) autoAlchemy(i int) int {
	q := pr.planet.Queue
	if i == len(q)-1 {
		// Last: converts all it can, ignoring its count, and leaves a
		// Mineral Alchemy partial at the front.
		pr.alchemize()
		return len(pr.planet.Queue)
	}
	next := q[i+1].Kind
	if next == ItemAutoAlchemy {
		return i + 1
	}
	defer pr.boughtEvent(len(pr.events))
	if next.auto() {
		return pr.autoInstall(i+1, true)
	}
	if pr.plain(i+1, true) {
		// The item finished: it and its prefix leave the queue.
		pr.remove(i + 1)
		pr.remove(i)
		return i
	}
	return i + 2
}

// boughtEvent reports a prefix's alchemy as one message ahead of the
// item's own messages, which start at index at (PQ-001 C06).
func (pr *production) boughtEvent(at int) {
	if pr.bought == 0 {
		return
	}
	pr.event(EventAlchemy, ItemMineralAlchemy, pr.bought)
	ev := pr.events[len(pr.events)-1]
	copy(pr.events[at+1:], pr.events[at:])
	pr.events[at] = ev
	pr.bought = 0
}

// alchemize converts the remaining resources into whole alchemy units and
// leaves a Mineral Alchemy ×1 partial at the queue front for the rest.
func (pr *production) alchemize() {
	rate := pr.cost(ItemMineralAlchemy).Resources
	units := pr.r / rate
	pr.r -= units * rate
	pr.complete(ItemMineralAlchemy, units)
	if pr.r > 0 {
		pct := pr.partial(pr.cost(ItemMineralAlchemy), 0)
		pr.insertFront(QueueItem{Kind: ItemMineralAlchemy, Count: 1, Percent: pct})
		pr.stopped = true
	}
}
