package vectors

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// zeroRand draws 0 every time.
type zeroRand struct{}

func (zeroRand) Intn(int) int { return 0 }

func TestMeasuredOB024CLostBeforeMoving(t *testing.T) {
	// OB-024-C (MEASURED; stars-elegy fda4173 KX-006): in the 30000,
	// 40000 and 60000 cycle runs fleet C, one unrated Laser DD at warp 10,
	// was lost before moving (0x0e1) and the speed-bump field only
	// decayed, 400 → 392. With every draw 0 its loss draw is 0, so the
	// vector's expectations for those runs hold. This checks the rule's
	// outcome, not the original's stream (KERNEL.md "Random draws").
	var ob *Vector
	for _, v := range loadVectors(t) {
		if v.path == "ob/ob024.json" {
			ob = v
		}
	}
	if ob == nil {
		t.Fatal("no ob/ob024.json")
	}
	for _, stream := range []string{"cycles 30000", "cycles 40000", "cycles 60000"} {
		checked := 0
		runVectorWith(ob, pvRun{stream: stream, replay: func(int) engine.Rand { return zeroRand{} }, observe: func(c pvCase, i int, msg string) {
			if c.ID != "OB-024-C" || c.Expect[i].Stream != stream {
				return
			}
			checked++
			if msg != "" {
				t.Errorf("%s expectation %d (%s): %s", stream, i, c.Expect[i].Kind, msg)
			}
		}})
		if checked != 2 {
			t.Errorf("%s: %d OB-024-C expectations checked, want 2", stream, checked)
		}
	}
}
