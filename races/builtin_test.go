package races

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// The 24 built-in computer races (AI.md "Built-in races", CONFIRMED AI-0
// for 23 of 24, SS harder BINARY-ONLY): every row reads, is in range (the
// rows were read from game files, so no repair applies), and spot values
// match the table.
func TestConfirmedBuiltInRaces(t *testing.T) {
	for typ := 1; typ <= 6; typ++ {
		for lv := range 4 {
			d, err := BuiltIn(typ, lv)
			if err != nil {
				t.Fatalf("type %d level %d: %v", typ, lv, err)
			}
			if _, changed := Repaired(d); changed {
				t.Errorf("type %d level %d needs a repair: %+v", typ, lv, d)
			}
		}
	}
	he, _ := BuiltIn(1, 0)
	if he.Race.PRT != engine.PRTHyperExpansion || he.Race.GrowthRate != 5 || !he.Race.Env[engine.Radiation].Immune ||
		!he.Race.LRT.BleedingEdgeTech || he.Race.ResearchCosts[2] != engine.ResearchCheap || he.Race.ResearchCosts[5] != engine.ResearchExpensive {
		t.Errorf("HE easy: %+v", he)
	}
	ss, _ := BuiltIn(2, 3)
	if ss.Race.Env[engine.Gravity] != (engine.EnvRange{Low: 31, Center: 62, High: 93}) || !ss.Race.Env[engine.Radiation].Immune ||
		ss.Race.FactoriesOperated != 25 || !ss.ExpensiveAt3 || !ss.Race.FactoryLessGermanium {
		t.Errorf("SS expert: %+v", ss)
	}
	ar, _ := BuiltIn(6, 0)
	if ar.Race.PRT != engine.PRTAlternateReality || ar.Race.ColonistsPerResource != 1600 || !ar.Race.LRT.CheapEngines ||
		ar.Race.ResearchCosts[4] != engine.ResearchExpensive || ar.ExpensiveAt3 {
		t.Errorf("AR easy: %+v", ar)
	}
	is, _ := BuiltIn(3, 3)
	if is.Race.Env[engine.Radiation] != (engine.EnvRange{Low: 0, Center: 50, High: 100}) || !is.Race.LRT.LowStartingPopulation {
		t.Errorf("IS expert: %+v", is)
	}
	if _, err := BuiltIn(0, 0); err == nil {
		t.Error("type 0 accepted")
	}
	if _, err := BuiltIn(1, 4); err == nil {
		t.Error("level 4 accepted")
	}
}
