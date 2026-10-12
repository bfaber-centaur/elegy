package vectors

import (
	"math/rand"
	"runtime"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// routed answers the draws made directly by the function whose name ends
// in site with answer, and every other draw from the seed variant's
// stream. It shows which draws a random case's outcome depends on.
type routed struct {
	vary     engine.Rand
	site     string
	answer   func(n int) int
	answered int
}

func (r *routed) Intn(n int) int {
	pc, _, _, _ := runtime.Caller(1)
	if f := runtime.FuncForPC(pc); f != nil && strings.HasSuffix(f.Name(), r.site) {
		r.answered++
		return r.answer(n)
	}
	return r.vary.Intn(n)
}

func (r *routed) turn(g engine.Game, o []engine.PlayerOrders, rng engine.Rand) (engine.TurnResult, error) {
	r.vary = rng
	return engine.GenerateTurn(g, o, r)
}

// CB-009 passes with 7 of the 8 seed variants. The one that fails, 7,
// differs at K8's salvage (x 1300): Elegy's 10 Beta Cruisers kill the 3
// Shield Frigates in one salvo, and a third of 3 × 1 kT boranium leaves
// 1 kT, where the original killed them over two or three salvos (0 kT).
// With the torpedo rolls the same on every variant and every other draw
// left to the variant (the frigates' walk among them), every variant
// passes: the case's random outcome is the torpedo rolls alone.
func TestCB009RandomIsTorpedoRolls(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/cb/cb009.json")
	if err != nil {
		t.Fatal(err)
	}
	for variant := range Seeds {
		fixed := rand.New(rand.NewSource(1))
		r := &routed{site: ".(*battle).torpedoes", answer: fixed.Intn}
		for _, res := range runVectorWith(v, pvRun{seed: variant, turn: r.turn}) {
			if res.Status != "pass" {
				t.Errorf("variant %d, torpedo rolls fixed: %s %s: %s", variant, res.ID, res.Status, res.Why)
			}
		}
		if r.answered == 0 {
			t.Fatalf("variant %d: no torpedo roll was answered", variant)
		}
	}
}

// RG-001-Y5 (2434) with seed variant 4: player 1's one-ship fleet
// crossing player 2's standard field at 1340,1269 at warp 7 hits a mine
// (9 per mille per ly) and is lost, so player 2's fleets that chase it
// stop short and no battle leaves salvage at 1324,1257; the field ends
// with 798 mines, not 824, and player 1 learns of it. In the original
// the fleet crossed unhit (the field is known to player 2 only). With
// every mine-hit draw answered "no hit" and every other draw left to the
// variant, those expectations hold on every variant: the difference is
// the hit draws.
func TestRG001Y5MinefieldIsHitDraws(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001-y5.json")
	if err != nil {
		t.Fatal(err)
	}
	watched := func(e pvExpect) bool {
		switch e.Kind {
		case "minefield", "salvage_at":
			return true
		case "fleet":
			return e.Owner != nil && *e.Owner == 2
		}
		return false
	}
	for variant := range Seeds {
		r := &routed{site: "objects.CheckStep", answer: func(n int) int { return n - 1 }}
		checked := 0
		runVectorWith(v, pvRun{seed: variant, turn: r.turn, observe: func(c pvCase, i int, msg string) {
			if !watched(c.Expect[i]) || strings.HasPrefix(msg, "skip: ") {
				return
			}
			checked++
			if msg != "" {
				t.Errorf("variant %d, no mine hits: expectation %d (%s): %s", variant, i, c.Expect[i].Kind, msg)
			}
		}})
		if r.answered == 0 || checked == 0 {
			t.Fatalf("variant %d: %d mine-hit draws answered, %d expectations checked", variant, r.answered, checked)
		}
	}
}
