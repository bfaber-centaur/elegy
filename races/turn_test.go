package races

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// zeroRand always draws 0.
type zeroRand struct{}

func (zeroRand) Intn(int) int { return 0 }

// GenerateTurn runs the race check on its copy of the game: the input
// game's checker keeps its designs, tampered flags included
// (engine.RaceCloner).
func TestGenerateTurnLeavesInputRaces(t *testing.T) {
	designs := slRaces()
	r := &GameRaces{Designs: designs, Computer: []bool{false, false}}
	before := append([]Design(nil), r.Designs...)
	g := engine.Game{Rules: engine.ElegyRules(), Players: []engine.Player{{Race: designs[0].Race, ResearchBudget: 15}, {Race: designs[1].Race, ResearchBudget: 15}}, Races: r}
	res, err := engine.GenerateTurn(withRules(g), nil, zeroRand{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Designs, before) {
		t.Errorf("input checker changed: %+v", r.Designs)
	}
	if out := res.Game.Races.(*GameRaces); !out.Designs[0].Tampered || !out.Designs[1].Tampered {
		t.Errorf("result checker %+v, want both races tampered", out.Designs)
	}
}

// withRules is g under the Elegy ruleset.
func withRules(g engine.Game) engine.Game {
	g.Rules = engine.ElegyRules()
	return g
}
