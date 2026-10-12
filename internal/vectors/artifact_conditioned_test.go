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

// TestConditionedArtifactRG001Planet71 checks player 1's 2430 artifact
// gift in the chained RG-001 conditioned on the original's draws, not as
// parity. The gift's field and points are random draws (TAKEOVER.md
// "Ancient artifacts": rand(6), then 100 + rand(301)), and the original
// gave player 1 +326 energy for planet 81 (stars-elegy 7c06f47). The
// chain cannot replay the original's generator, so on Elegy's reference
// seed the year 2430 is generated once to record its draws, then again
// with only the artifact's two draws replaced by energy (0) and 226
// (100 + 226 = 326; the colony has at least 1,000 colonists, so the
// points are not scaled).
//
// With the original's gift, planet 71 (player 1's homeworld) matches the
// original's surface minerals in 2433 and 2434; without any gift the
// reference seed has them short in both years (the economy lane's
// reading: the year's builds there are priced at the lower energy
// level). 2435 still differs with the original's gift, from another
// cause, so it is not checked here.
func TestConditionedArtifactRG001Planet71(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001.json")
	if err != nil {
		t.Fatal(err)
	}
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
					t.Fatalf("2430: two draw pairs that could be the artifact's, at %d and %d", at, i)
				}
				at = i
			}
		}
		if at < 0 {
			t.Fatal("2430: no artifact draws")
		}
		re := &replayRand{log: rec.log, at: at, with: []int{0, 226}}
		r, err := engine.GenerateTurn(g, o, re)
		if re.bad != "" {
			t.Fatalf("2430 replay: %s", re.bad)
		}
		for _, e := range r.Events {
			if e.Kind == engine.EventArtifactFound {
				gifts = append(gifts, e)
			}
		}
		return r, err
	}
	bad, compared := map[int][]string{}, map[int]int{}
	runVectorWith(v, pvRun{turn: turn, observe: func(c pvCase, i int, msg string) {
		e := c.Expect[i]
		// A skipped planet field makes a matching check a skip (pvResult);
		// the fields compared, surface minerals among them, matched.
		if e.Kind != "planet" || e.ID == nil || *e.ID != 71 || strings.HasPrefix(msg, "skip: ") && !strings.HasPrefix(msg, "skip: planet field ") {
			return
		}
		compared[e.Year]++
		if !strings.HasPrefix(msg, "skip: ") && msg != "" {
			bad[e.Year] = append(bad[e.Year], msg)
		}
	}})
	// Planet 81 is Elegy's planet 82 (vector id + 1).
	if len(gifts) != 1 || gifts[0].Player != 1 || gifts[0].Planet != 82 || gifts[0].Item != 0 || gifts[0].Count != 326 {
		t.Fatalf("2430 gifts %+v, want player 1's energy 326 at planet 81", gifts)
	}
	for y, year := range map[int]int{4: 2433, 5: 2434} {
		if compared[y] == 0 {
			t.Errorf("%d planet 71: nothing compared", year)
		}
		if len(bad[y]) != 0 {
			t.Errorf("%d planet 71: %q", year, bad[y])
		}
	}
}
