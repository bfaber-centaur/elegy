package vectors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// recordRand records every draw of r.
type recordRand struct {
	r   engine.Rand
	log [][2]int // {n, value}
}

func (r *recordRand) Intn(n int) int {
	v := r.r.Intn(n)
	r.log = append(r.log, [2]int{n, v})
	return v
}

// replayRand replays a recorded year, with the draws from at on replaced
// by with. bad is set when the year asks for a draw the record does not
// have, so the replay is no longer the recorded year.
type replayRand struct {
	log  [][2]int
	at   int
	with []int
	next int
	bad  string
}

func (r *replayRand) Intn(n int) int {
	p := r.next
	r.next++
	if p >= len(r.log) || r.log[p][0] != n {
		if r.bad == "" {
			r.bad = fmt.Sprintf("draw %d: Intn(%d) is not the recorded draw", p, n)
		}
		return 0
	}
	if k := p - r.at; k >= 0 && k < len(r.with) {
		return r.with[k]
	}
	return r.log[p][1]
}

// TestConditionedArtifactRG001Gift checks player 1's 2430 artifact gift
// in the chained RG-001 conditioned on the original's draws, not as
// parity. The gift's field and points are random draws (TAKEOVER.md
// "Ancient artifacts": rand(6), then 100 + rand(301)), and the original
// gave player 1 +326 energy for planet 81 (stars-elegy 7c06f47). The
// chain cannot replay the original's generator, so on each of Elegy's
// seed variants the year 2430 is generated once to record its draws, then
// again with only the artifact's two draws replaced by energy (0) and 226
// (100 + 226 = 326; the colony has at least 1,000 colonists, so the
// points are not scaled).
//
// With the original's gift, every player expectation of player 1's in
// 2430–2433 (research, tech levels, score record) matches on all seed
// variants; with the gift Elegy draws, energy stays a level short and
// they fail. 2434 brings player 1's next artifact (planet 114), a draw
// that is not conditioned, so it is not checked. Planet 71's surface
// minerals are not checked either: they also carry each year's mining
// draws (±1, PARITY.md RG-001), which no conditioning here covers.
func TestConditionedArtifactRG001Gift(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001.json")
	if err != nil {
		t.Fatal(err)
	}
	for seed := range Seeds {
		var gifts []engine.Event
		turn := func(g engine.Game, o []engine.PlayerOrders, rng engine.Rand) (engine.TurnResult, error) {
			if g.Year != 2429 {
				return engine.GenerateTurn(g, o, rng)
			}
			rec := &recordRand{r: rng}
			if _, err := engine.GenerateTurn(g, o, rec); err != nil {
				return engine.TurnResult{}, err
			}
			at := -1
			for i := 0; i+1 < len(rec.log); i++ {
				if rec.log[i][0] == engine.NumFields && rec.log[i+1][0] == 301 {
					if at >= 0 {
						t.Fatalf("seed %d, 2430: two draw pairs that could be the artifact's, at %d and %d", seed, at, i)
					}
					at = i
				}
			}
			if at < 0 {
				t.Fatalf("seed %d, 2430: no artifact draws", seed)
			}
			re := &replayRand{log: rec.log, at: at, with: []int{0, 226}}
			r, err := engine.GenerateTurn(g, o, re)
			if re.bad != "" {
				t.Fatalf("seed %d, 2430 replay: %s", seed, re.bad)
			}
			for _, e := range r.Events {
				if e.Kind == engine.EventArtifactFound {
					gifts = append(gifts, e)
				}
			}
			return r, err
		}
		bad, compared := map[int][]string{}, map[int]int{}
		runVectorWith(v, pvRun{seed: seed, turn: turn, observe: func(c pvCase, i int, msg string) {
			e := c.Expect[i]
			if e.Kind != "player" || e.ID == nil || *e.ID != 1 || e.Year > 4 || strings.HasPrefix(msg, "skip: ") {
				return
			}
			compared[e.Year]++
			if msg != "" {
				bad[e.Year] = append(bad[e.Year], msg)
			}
		}})
		// Planet 81 is Elegy's planet 82 (vector id + 1).
		if len(gifts) != 1 || gifts[0].Player != 1 || gifts[0].Planet != 82 || gifts[0].Item != 0 || gifts[0].Count != 326 {
			t.Fatalf("seed %d, 2430 gifts %+v, want player 1's energy 326 at planet 81", seed, gifts)
		}
		for y := 1; y <= 4; y++ {
			if compared[y] == 0 {
				t.Errorf("seed %d, %d player 1: nothing compared", seed, 2429+y)
			}
			if len(bad[y]) != 0 {
				t.Errorf("seed %d, %d player 1: %q", seed, 2429+y, bad[y])
			}
		}
	}
}
