package terraform

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// tfGame is player 0's planet at env (original the same) with the given
// queue, tech levels 1 everywhere (reach ±3) and the terraforming rules.
// Only gravity is off the race's centre.
func tfGame(env [3]int, queue ...engine.QueueItem) *engine.Game {
	r := race()
	g := &engine.Game{
		Rules:     engine.ElegyRules(),
		Players:   []engine.Player{{Race: r, Research: engine.ResearchState{Levels: levels(1, 1, 1, 1, 1, 1)}}},
		Planets:   []engine.Planet{{ID: 1, Owner: 0, Env: env, OrigEnv: env, Population: 1000, HasQueue: true, Queue: queue}},
		Terraform: Rules{},
	}
	return g
}

func tfProduce(g *engine.Game, resources, grownPop int) []engine.Event {
	p := &g.Planets[0]
	in := engine.ProductionInput{Colony: engine.NewColony(p, &g.Players[0]), Resources: resources, GrownPop: grownPop}
	_, ev := g.PlanetProduction(0, in)
	return ev
}

// A Terraform Environment order is cut to the capacity with a message and
// each unit moves one click (KERNEL.md "Terraforming", CONFIRMED KX-002
// T1: ×5 → ×3, 60 → 57 with gravity ±3), at 100 resources a unit.
func TestConfirmedTerraformItemClipAndClicks(t *testing.T) {
	g := tfGame([3]int{60, 50, 50}, engine.QueueItem{Kind: engine.ItemTerraform, Count: 5})
	if r := Reach(race(), g.Players[0].Research.Levels); r[0] != 3 {
		t.Fatalf("reach %v, want gravity ±3", r)
	}
	ev := tfProduce(g, 250, 1000)
	p := g.Planets[0]
	if p.Env != [3]int{58, 50, 50} || len(p.Queue) != 1 || p.Queue[0].Count != 1 || p.Queue[0].Percent != 50 {
		t.Errorf("env %v, queue %+v", p.Env, p.Queue)
	}
	if len(ev) < 2 || ev[0].Kind != engine.EventOrderClipped || ev[0].Count != 3 || ev[1].Kind != engine.EventBuilt || ev[1].Item != engine.ItemTerraform || ev[1].Count != 2 {
		t.Errorf("events %+v", ev)
	}
}

// An order reaching a planet with no capacity is removed with nothing
// built (KERNEL.md "Terraforming", CONFIRMED KB-2C).
func TestConfirmedTerraformItemNoCapacity(t *testing.T) {
	g := tfGame([3]int{50, 50, 50}, engine.QueueItem{Kind: engine.ItemTerraform, Count: 2}, engine.QueueItem{Kind: engine.ItemMine, Count: 1})
	tfProduce(g, 100, 1000)
	if p := g.Planets[0]; len(p.Queue) != 0 || p.Env != [3]int{50, 50, 50} || p.Mines != 1 {
		t.Errorf("planet %+v", p)
	}
}

// Auto Max builds up to the capacity and stays; Auto Min builds only when
// the population falls or the habitability is 0 or less (KERNEL.md
// "Terraforming", CONFIRMED KX-005).
func TestConfirmedAutoTerraform(t *testing.T) {
	max := engine.QueueItem{Kind: engine.ItemAutoMaxTerraform, Count: 9}
	g := tfGame([3]int{60, 50, 50}, max)
	tfProduce(g, 1000, 1100)
	if p := g.Planets[0]; p.Env != [3]int{57, 50, 50} || len(p.Queue) != 1 || p.Queue[0] != max {
		t.Errorf("auto max: env %v, queue %+v", p.Env, p.Queue)
	}
	min := engine.QueueItem{Kind: engine.ItemAutoMinTerraform, Count: 9}
	g = tfGame([3]int{60, 50, 50}, min)
	tfProduce(g, 1000, 1100)
	if p := g.Planets[0]; p.Env != [3]int{60, 50, 50} {
		t.Errorf("auto min, growing: env %v", p.Env)
	}
	g = tfGame([3]int{60, 50, 50}, min)
	tfProduce(g, 1000, 900)
	if p := g.Planets[0]; p.Env != [3]int{57, 50, 50} || len(p.Queue) != 1 {
		t.Errorf("auto min, shrinking: env %v, queue %+v", p.Env, p.Queue)
	}
}

// Without terraforming rules a terraform item stops the queue.
func TestTerraformItemNotModelled(t *testing.T) {
	g := tfGame([3]int{60, 50, 50}, engine.QueueItem{Kind: engine.ItemTerraform, Count: 1}, engine.QueueItem{Kind: engine.ItemMine, Count: 1})
	g.Terraform = nil
	tfProduce(g, 1000, 1000)
	if p := g.Planets[0]; len(p.Queue) != 2 || p.Mines != 0 || p.Env != [3]int{60, 50, 50} {
		t.Errorf("planet %+v", p)
	}
}
