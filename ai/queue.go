package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// maxQueueForAI is AI.md §10 "Queueing items": a queue with more than 200
// items gets nothing more.
const maxQueueForAI = 200

// queues are the production queues as the planner changes them during
// its turn. Each changed planet gets one queue order at the end.
type queues struct {
	v *View
	m map[int][]engine.QueueItem
}

func (q *queues) get(p *engine.Planet) []engine.QueueItem {
	if l, ok := q.m[p.ID]; ok {
		return l
	}
	return p.Queue
}

// add puts an item at the back, or at the front with front set, through
// the shared queueing rule (AI.md §10 "Queueing items"): nothing when the
// planet cannot build it, the count is below 1, or the queue already has
// more than 200 items. It reports whether the item went in.
func (q *queues) add(p *engine.Planet, it engine.QueueItem, front bool) bool {
	l := q.get(p)
	if it.Count < 1 || len(l) > maxQueueForAI || !q.canBuild(p, it) {
		return false
	}
	if q.m == nil {
		q.m = map[int][]engine.QueueItem{}
	}
	if front {
		q.m[p.ID] = append([]engine.QueueItem{it}, l...)
	} else {
		q.m[p.ID] = append(slices.Clone(l), it)
	}
	return true
}

// canBuild is the production list's test (PRODUCTION-LAUNCH.md "Can the
// planet build it"): a ship needs a design and a starbase that docks its
// hull; a starbase item needs a design.
func (q *queues) canBuild(p *engine.Planet, it engine.QueueItem) bool {
	v := q.v
	switch it.Kind {
	case engine.ItemShip:
		d, ok := v.ship(it.Slot)
		if !ok || !p.HasStarbase {
			return false
		}
		sb, ok := v.design(p.StarbaseDesign)
		if !ok {
			return false
		}
		return sb.Hull.Dock == engine.DockUnlimited || (sb.Hull.Dock > 0 && d.Design.Hull.Mass <= sb.Hull.Dock)
	case engine.ItemStarbase:
		for _, d := range v.Starbases {
			if d.Slot == it.Slot {
				return true
			}
		}
		return false
	}
	return true
}

// count is the units of kind k queued at p.
func (q *queues) count(p *engine.Planet, k engine.ItemKind) int {
	n := 0
	for _, it := range q.get(p) {
		if it.Kind == k {
			n += it.Count
		}
	}
	return n
}

func (q *queues) has(p *engine.Planet, k engine.ItemKind) bool {
	return slices.ContainsFunc(q.get(p), func(it engine.QueueItem) bool { return it.Kind == k })
}

// flush is one queue order per changed planet, in planet-id order.
func (q *queues) flush() []engine.Order {
	ids := make([]int, 0, len(q.m))
	for id := range q.m {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	var out []engine.Order
	for _, id := range ids {
		out = append(out, engine.QueueOrder{Planet: id, Queue: q.m[id]})
	}
	return out
}
