package terraform

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// KERNEL.md "Remote mining" (BINARY-ONLY, stars-elegy 8a3b9b5): an
// Alternate Reality owner's miners at its own homeworld use the homeworld
// floor of 30, as the planet's own mining does; an unowned planet has no
// floor, even one marked as a former homeworld. A 10-point miner at
// concentration 20 makes 300 hundredths with the floor, 200 without.
func TestRemoteMineHomeworldFloor(t *testing.T) {
	l := newLab(t)
	l.g.Players[0].Race.PRT = engine.PRTAlternateReality
	ar := l.planet(0, [3]int{50, 50, 50}, [3]int{20, 20, 20})
	l.g.Planets[ar].Homeworld = true
	if gain, ok := RemoteMine(l.g, l.fleet(0, ar, "Midget Miner", 1), &count{}); !ok || gain != (engine.Minerals{3, 3, 3}) {
		t.Errorf("own AR homeworld: %v %v, want 3/3/3", gain, ok)
	}
	l = newLab(t)
	free := l.planet(engine.NoOwner, [3]int{50, 50, 50}, [3]int{20, 20, 20})
	l.g.Planets[free].Homeworld = true
	if gain, ok := RemoteMine(l.g, l.fleet(0, free, "Midget Miner", 1), &count{}); !ok || gain != (engine.Minerals{2, 2, 2}) {
		t.Errorf("unowned planet: %v %v, want 2/2/2", gain, ok)
	}
}
