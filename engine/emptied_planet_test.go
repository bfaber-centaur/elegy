package engine

import "testing"

// An emptied planet loses its queue and its leftover setting with its
// owner (TAKEOVER.md "Capture: what a planet keeps", CONFIRMED T-21,
// T-26, T-27), so a later owner starts from its own defaults.
func TestEmptiedPlanetClearsQueue(t *testing.T) {
	l := newTKLab(t, 3)
	pi := l.planet(1, 0, 10)
	p := &l.g.Planets[pi]
	p.HasQueue, p.Queue, p.LeftoverOnly = true, []QueueItem{q(ItemMine, 3, 0)}, true
	l.g.emptyPlanet(pi)
	if p := l.g.Planets[pi]; p.HasQueue || p.Queue != nil || p.LeftoverOnly {
		t.Errorf("emptied planet: has queue %v, queue %v, leftover only %v; want none", p.HasQueue, p.Queue, p.LeftoverOnly)
	}
}
