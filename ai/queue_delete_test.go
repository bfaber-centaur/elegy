package ai

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestDeleteDropsQueuedShips: a design delete drops every queue entry
// that builds the slot, even with progress, as the engine does when the
// order applies (ORDERS.md "Design delete effect", MEASURED CO-07). The
// queue orders written later in the turn must not name the slot: the
// engine refuses a queue item whose slot holds no design.
func TestDeleteDropsQueuedShips(t *testing.T) {
	ship := func(slot, n, pct int) engine.QueueItem {
		return engine.QueueItem{Kind: engine.ItemShip, Slot: slot, Count: n, Percent: pct}
	}
	factory := engine.QueueItem{Kind: engine.ItemFactory, Count: 1}
	base7 := engine.QueueItem{Kind: engine.ItemStarbase, Slot: 7, Count: 1} // a starbase design slot: kept
	v := heView(2435, [engine.NumFields]int{})
	v.Ships = []Design{{Slot: 3}, {Slot: 7}}
	v.Planets = []engine.Planet{
		{ID: 5, Queue: []engine.QueueItem{ship(7, 1, 0), ship(3, 1, 10)}},
		{ID: 112, Queue: []engine.QueueItem{ship(3, 1, 0), ship(7, 2, 32), base7, ship(7, 1, 0)}},
		{ID: 140, Queue: []engine.QueueItem{ship(7, 1, 50)}},
	}
	orig140, orig5 := v.Planets[2].Queue, v.Planets[0].Queue
	var res Result
	tr := newTurn(v, top{}, &res)
	// Planet 5's queue is changed before the delete, planet 112's after.
	tr.q.add(v.ownPlanet(5), factory, false)
	tr.sd.delete(7)
	tr.q.add(v.ownPlanet(112), factory, false)

	got := map[int][]engine.QueueItem{}
	for _, o := range tr.q.flush() {
		qo := o.(engine.QueueOrder)
		got[qo.Planet] = qo.Queue
	}
	want := map[int][]engine.QueueItem{
		5:   {ship(3, 1, 10), factory},
		112: {ship(3, 1, 0), base7, factory},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("queue orders %+v, want %+v", got, want)
	}
	// An unchanged queue reads as the engine leaves it.
	if q := tr.q.get(v.ownPlanet(140)); len(q) != 0 {
		t.Errorf("planet 140 queue %+v, want empty", q)
	}
	// The report's own queues are not written through.
	if orig140[0] != ship(7, 1, 50) || orig5[0] != ship(7, 1, 0) {
		t.Errorf("report queues changed: %+v, %+v", orig140, orig5)
	}
	// Deleting an absent slot drops nothing.
	tr.sd.delete(7)
	v.Planets[2].Queue = []engine.QueueItem{ship(7, 1, 0)}
	tr.sd.delete(7)
	if q := tr.q.get(v.ownPlanet(140)); len(q) != 1 {
		t.Errorf("second delete dropped %+v", q)
	}
}
