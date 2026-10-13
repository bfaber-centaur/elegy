package engine

import "testing"

// noAxisTerraform has room for clicks but no useful axis to move.
type noAxisTerraform struct{ stubTerraform }

func (*noAxisTerraform) Capacity(Planet, Race, [3]int) int { return 5 }

// TestTerraformUnitNoAxisNoMessage: a completed terraform unit with no
// useful axis is used up with no message (MESSAGES.md 0x07b at
// stars-elegy de1bd6f, CONFIRMED KX-002).
func TestTerraformUnitNoAxisNoMessage(t *testing.T) {
	g := pgHomeworld()
	g.Terraform = &noAxisTerraform{}
	p := &g.Planets[0]
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemTerraform, Count: 2}}
	_, ev := g.PlanetProduction(0, ProductionInput{Colony: NewColony(p, &g.Players[0]), Resources: 200, GrownPop: p.Population})
	for _, e := range ev {
		if e.Kind == EventTerraformed || (e.Kind == EventBuilt && e.Item == ItemTerraform) {
			t.Errorf("unexpected %+v", e)
		}
	}
	if len(p.Queue) != 0 || p.Env != [3]int{50, 50, 50} {
		t.Errorf("queue %+v, env %v: want both units used up and no change", p.Queue, p.Env)
	}
}
