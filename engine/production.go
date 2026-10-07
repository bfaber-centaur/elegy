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

// J-RC3 costs that do not come from the race (PQ-001, observed for one
// race; the defense and alchemy costs are not known to vary).
var (
	DefenseCost = Cost{Resources: 15, Minerals: Minerals{5, 5, 5}}
	AlchemyCost = Cost{Resources: 100}
)

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
		return 0, nil // BINARY-ONLY: an empty queue block contributes nothing
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
	switch k.real() {
	case ItemMine:
		return pr.in.Colony.Race.MineCost
	case ItemFactory:
		return pr.in.Colony.Race.FactoryCost
	case ItemDefenses:
		return DefenseCost
	}
	return AlchemyCost
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

func (pr *production) mineralShort(c Cost, pct int) bool {
	s := spent(c, pct)
	for m := range NumMinerals {
		if c.Minerals[m]-s.Minerals[m] > pr.planet.Surface[m] {
			return true
		}
	}
	return false
}

// pay moves a unit from from% to to% (100 completes it).
func (pr *production) pay(c Cost, from, to int) {
	a, b := spent(c, from), spent(c, to)
	pr.r -= b.Resources - a.Resources
	for m := range NumMinerals {
		pr.planet.Surface[m] -= b.Minerals[m] - a.Minerals[m]
	}
}

// largestPercent is the largest whole percentage p with trunc(cost·p/100) ≤
// avail, where avail includes what is already spent.
func largestPercent(cost, avail int) int {
	return (100*(avail+1) - 1) / cost
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
			i = pr.autoInstall(i)
		default:
			if pr.plain(i) {
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
// A unit that cannot be finished becomes partial and stops the queue.
func (pr *production) plain(i int) bool {
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
			it.Percent = pr.partial(c, it.Percent)
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
// "Caps", CONFIRMED PQ C04, C09, C13, C14).
func (pr *production) autoInstall(i int) int {
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
		if pr.mineralShort(c, 0) {
			pr.blocked = true // skipped; the walk continues
			break
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

// autoAlchemy processes Auto Alchemy at index i and returns the next index.
func (pr *production) autoAlchemy(i int) int {
	q := pr.planet.Queue
	if i == len(q)-1 {
		// Last: converts all it can, ignoring its count, and leaves a
		// Mineral Alchemy partial at the front.
		pr.alchemize(i)
		return len(pr.planet.Queue)
	}

	next := q[i+1]
	units := 0
	if next.Count > 0 {
		c := pr.cost(next.Kind)
		s := spent(c, next.Percent)
		// KERNEL/PQ-001 only cover a ×1 item; the shortfall here is for
		// the item's whole remaining count.
		for m := range NumMinerals {
			need := c.Minerals[m]*next.Count - s.Minerals[m]
			units = max(units, need-pr.planet.Surface[m])
		}
	}
	if units*AlchemyCost.Resources <= pr.r {
		pr.r -= units * AlchemyCost.Resources
		for m := range NumMinerals {
			pr.planet.Surface[m] += units
		}
		if units > 0 {
			pr.event(EventAlchemy, ItemMineralAlchemy, units)
		}
		if pr.plain(i + 1) {
			// The item finished: it and its alchemy prefix leave the queue.
			pr.remove(i + 1)
			pr.remove(i)
			return i
		}
		return i + 2
	}

	// Short of resources for the whole shortfall (PQ-001 C07): the item
	// takes its partial on current stock, then alchemy converts what is
	// left and leaves its own partial at the front.
	pr.plain(i + 1)
	pr.alchemize(i)
	return len(pr.planet.Queue)
}

// alchemize converts the remaining resources into whole alchemy units and
// leaves a Mineral Alchemy ×1 partial at the queue front for the rest.
func (pr *production) alchemize(i int) {
	units := pr.r / AlchemyCost.Resources
	pr.r -= units * AlchemyCost.Resources
	pr.complete(ItemMineralAlchemy, units)
	if pr.r > 0 {
		pct := pr.partial(AlchemyCost, 0)
		pr.insertFront(QueueItem{Kind: ItemMineralAlchemy, Count: 1, Percent: pct})
		pr.stopped = true
	}
}
