package vectors

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/oraclerng"
)

// The parity harness's random streams. A vector run in the original drew
// from the stream its startup tick seeded; Elegy's own draws are not the
// original's, so outcomes decided by the draws compare only as samples
// unless the harness replays the oracle's stream (oraclerng).

// pvAllStreams, as a run's stream, checks the expectations of every
// stream against one generation. It is only meaningful with Elegy's own
// seeding, where the stream does not change what is generated.
const pvAllStreams = "*"

// pvRandomStart records where one stream's year started in the oracle
// (proposed vector field "random_start"; FORMAT.md does not have it yet):
//
//	"random_start": [
//	  {"stream": "cycles 20000", "year": 1, "tick": 164, "state": null}
//	]
//
// stream is a stream label, or "" for a single-stream vector; year is the
// generated year (1 is the first). tick is the startup tick in
// milliseconds that seeded the stream, and state the generator's state
// before the year's first draw, in the published generator's terms. Either
// may be absent or null. A year with no entry, or with neither field, has
// no recorded start.
type pvRandomStart struct {
	Stream string          `json:"stream"`
	Year   int             `json:"year"`
	Tick   *int            `json:"tick"`
	State  json.RawMessage `json:"state"`
}

// pvStart returns where stream's year y started: the vector's
// random_start entry, else, for a stream labelled with its cycles value,
// the startup tick ORACLE.md measured for that value. The zero Start
// means nothing is recorded.
func pvStart(v *Vector, stream string, y int) oraclerng.Start {
	for _, r := range v.RandomStart {
		if r.Stream != stream || r.Year != y {
			continue
		}
		s := oraclerng.Start{State: r.State}
		if r.Tick != nil {
			s.Tick, s.HasTick = *r.Tick, true
		}
		if s.Recorded() {
			return s
		}
	}
	if tick, ok := oraclerng.StreamTick(stream); ok {
		return oraclerng.Start{Tick: tick, HasTick: true}
	}
	return oraclerng.Start{}
}

// pvReplay returns the random source for each year of stream's replay, or
// why there is none: a year whose start is not recorded, or one seed
// cannot start from.
func pvReplay(v *Vector, stream string, seed oraclerng.Seeder) (func(y int) engine.Rand, string) {
	for y := 1; y <= v.Years; y++ {
		s := pvStart(v, stream, y)
		if !s.Recorded() {
			return nil, fmt.Sprintf("year %d: no start recorded", y)
		}
		if seed == nil {
			return nil, "no generator"
		}
		if _, ok := seed(s); !ok {
			return nil, fmt.Sprintf("year %d: the generator cannot start from %+v", y, s)
		}
	}
	return func(y int) engine.Rand {
		r, _ := seed(pvStart(v, stream, y))
		return r
	}, ""
}

// pvRun selects how runVectorWith generates a vector and which
// expectations it checks.
type pvRun struct {
	// seed is the seed variant of Elegy's own seeding (0 is the
	// reference).
	seed int
	// stream is the oracle stream whose expectations are checked besides
	// the unlabelled ones: "" for none, a stream label, or pvAllStreams.
	stream string
	// replay, when set, replays the oracle's stream (pvReplay): year y
	// draws from replay(y), and samples compare exactly.
	replay func(y int) engine.Rand
	// observe, when set, receives every expectation's outcome: "" when it
	// matched, else the mismatch or "skip: " and the reason.
	observe func(c pvCase, i int, msg string)
}

// rand is the random source for year y.
func (r pvRun) rand(y int) engine.Rand {
	if r.replay != nil {
		return r.replay(y)
	}
	return rand.New(rand.NewSource(int64(y) + int64(r.seed)<<32))
}

// checks reports whether the run compares e.
func (r pvRun) checks(e pvExpect) bool {
	return e.Stream == "" || e.Stream == r.stream || (r.stream == pvAllStreams && r.replay == nil)
}

// sample reports whether a mismatch of e is only a missed sample: a
// random outcome compared without the oracle's draws.
func (r pvRun) sample(e pvExpect) bool {
	return r.replay == nil && (e.Sample || e.Stream != "")
}

func (r pvRun) observe1(c pvCase, i int, msg string) {
	if r.observe != nil {
		r.observe(c, i, msg)
	}
}
