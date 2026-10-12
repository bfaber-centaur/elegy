package engine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

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
func pvStart(v *pvVector, stream string, y int) oraclerng.Start {
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
func pvReplay(v *pvVector, stream string, seed oraclerng.Seeder) (func(y int) Rand, string) {
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
	return func(y int) Rand {
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
	replay func(y int) Rand
	// observe, when set, receives every expectation's outcome: "" when it
	// matched, else the mismatch or "skip: " and the reason.
	observe func(c pvCase, i int, msg string)
}

// rand is the random source for year y.
func (r pvRun) rand(y int) Rand {
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

// pvRandom reports whether e is a random outcome: a sample, or one
// stream's value of an expectation that differed between streams.
func pvRandom(e pvExpect) bool {
	return e.Sample || e.Kind == "sample" || e.Stream != ""
}

// pvOriginal starts the original's generator for replaying oracle
// streams. It is nil: stars-elegy has not published the generator or its
// seeding as a rule (KERNEL.md "Conventions" at d4eafd9 gives only
// rand(n)), and the clean room implements it only from that text.
var pvOriginal oraclerng.Seeder

// pvStreamLabels returns the stream labels v's expectations use.
func pvStreamLabels(v *pvVector) []string {
	seen := map[string]bool{}
	for _, c := range v.Cases {
		for _, e := range c.Expect {
			if e.Stream != "" {
				seen[e.Stream] = true
			}
		}
	}
	var out []string
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// pvRandomTally is one corpus's random-parity report, in expectations.
type pvRandomTally struct {
	// Other-stream expectations (one stream's value of an expectation that
	// differed between streams; the harness's main run skips them) and
	// single-stream samples: total, and how many of them the harness
	// compares at all. The rest wait on Elegy modelling the quantity,
	// whatever the stream.
	stream, streamCompared, sample, sampleCompared int
	// Deterministic: the exact expectations of cases that also hold a
	// random outcome, with the reference seed: matched, mismatched, not
	// compared.
	detMatch, detMiss, detSkip int
	// Results with Elegy's own seeds, over the compared random outcomes:
	// matched with the reference seed, with another seed only, never.
	resRef, resSome, resNever int
	// Exact: compared random outcomes replayed from the oracle's stream
	// (matched, mismatched), or waiting: on the generator with the start
	// in the vector, on the generator with a tick from ORACLE.md, or on a
	// recorded start.
	exMatch, exMiss, waitVector, waitTable, waitStart int
}

func (a *pvRandomTally) add(b *pvRandomTally) {
	a.stream += b.stream
	a.streamCompared += b.streamCompared
	a.sample += b.sample
	a.sampleCompared += b.sampleCompared
	a.detMatch += b.detMatch
	a.detMiss += b.detMiss
	a.detSkip += b.detSkip
	a.resRef += b.resRef
	a.resSome += b.resSome
	a.resNever += b.resNever
	a.exMatch += b.exMatch
	a.exMiss += b.exMiss
	a.waitVector += b.waitVector
	a.waitTable += b.waitTable
	a.waitStart += b.waitStart
}

// pvStartSource says where a stream's starts come from for every year:
// "vector" (random_start), "table" (only ORACLE.md ticks), or "" when a
// year has none.
func pvStartSource(v *pvVector, stream string) string {
	src := "vector"
	for y := 1; y <= v.Years; y++ {
		if !pvStart(v, stream, y).Recorded() {
			return ""
		}
		inVector := false
		for _, r := range v.RandomStart {
			if r.Stream == stream && r.Year == y && (r.Tick != nil || len(r.State) > 0 && string(r.State) != "null") {
				inVector = true
			}
		}
		if !inVector {
			src = "table"
		}
	}
	return src
}

// pvStartRank orders pvStartSource's answers, best first.
var pvStartRank = map[string]int{"": 0, "table": 1, "vector": 2}

// TestRandomStreams reports, per corpus, how far Elegy matches the
// original's random outcomes, in three separate categories:
//
//   - deterministic: the exact expectations of every case that also holds
//     a random outcome;
//   - results: each random outcome under Elegy's own seed variants. A
//     match shows Elegy reached the original's outcome with its own
//     draws, not that it draws the same numbers;
//   - exact: each random outcome replayed from the oracle's stream. This
//     needs the original's generator (pvOriginal) and each year's start
//     (pvStart).
//
// It also counts the random outcomes a recorded start would make exactly
// checkable. It fails when an exact replay misses.
func TestRandomStreams(t *testing.T) {
	per := map[string]*pvRandomTally{}
	var corpora []string
	for _, v := range loadVectors(t) {
		type key struct {
			c string
			i int
		}
		random := map[string]bool{}
		for _, c := range v.Cases {
			for _, e := range c.Expect {
				if pvRandom(e) {
					random[c.ID] = true
				}
			}
		}
		if len(random) == 0 {
			continue
		}
		tl := per[v.corpus]
		if tl == nil {
			tl = &pvRandomTally{}
			per[v.corpus] = tl
			corpora = append(corpora, v.corpus)
		}
		ref := map[key]string{}
		matched := map[key]bool{}
		for k := 0; k < pvSeeds; k++ {
			runVectorWith(v, pvRun{seed: k, stream: pvAllStreams, observe: func(c pvCase, i int, msg string) {
				if k == 0 {
					ref[key{c.ID, i}] = msg
				}
				if msg == "" {
					matched[key{c.ID, i}] = true
				}
			}})
		}
		// Exact replay, one generation per stream. A single-stream vector
		// has the one stream "". An unlabelled sample of a several-stream
		// vector came out the same in every stream, so every replayed
		// stream checks it, and its start is the best of the streams'.
		exact := map[key]string{}
		source := map[string]string{}
		streams := pvStreamLabels(v)
		if len(streams) == 0 {
			streams = []string{""}
		}
		for _, s := range streams {
			source[s] = pvStartSource(v, s)
			if pvStartRank[source[s]] > pvStartRank[source[""]] {
				source[""] = source[s]
			}
			replay, _ := pvReplay(v, s, pvOriginal)
			if replay == nil {
				continue
			}
			runVectorWith(v, pvRun{stream: s, replay: replay, observe: func(c pvCase, i int, msg string) {
				e := c.Expect[i]
				if !pvRandom(e) || (e.Stream != s && e.Stream != "") {
					return
				}
				if prev, ok := exact[key{c.ID, i}]; !ok || prev == "" {
					exact[key{c.ID, i}] = msg
				}
			}})
		}
		for _, c := range v.Cases {
			if !random[c.ID] {
				continue
			}
			for i, e := range c.Expect {
				k := key{c.ID, i}
				msg := ref[k]
				compared := !strings.HasPrefix(msg, "skip: ")
				if !pvRandom(e) {
					switch {
					case !compared:
						tl.detSkip++
					case msg == "":
						tl.detMatch++
					default:
						tl.detMiss++
						t.Logf("deterministic mismatch, reference seed: %s %s y%d %s: %s", v.path, c.ID, e.Year, e.Kind, msg)
					}
					continue
				}
				if e.Stream != "" {
					tl.stream++
				} else {
					tl.sample++
				}
				if !compared {
					continue
				}
				if e.Stream != "" {
					tl.streamCompared++
				} else {
					tl.sampleCompared++
				}
				switch {
				case msg == "":
					tl.resRef++
				case matched[k]:
					tl.resSome++
				default:
					tl.resNever++
				}
				xm, replayed := exact[k]
				switch {
				case replayed && xm == "":
					tl.exMatch++
				case replayed:
					tl.exMiss++
					t.Errorf("%s %s expectation %d (y%d %s, stream %q): replay of the oracle's stream: %s", v.path, c.ID, i, e.Year, e.Kind, e.Stream, xm)
				case source[e.Stream] == "vector":
					tl.waitVector++
				case source[e.Stream] == "table":
					tl.waitTable++
				default:
					tl.waitStart++
				}
			}
		}
	}
	sort.Strings(corpora)
	var sum pvRandomTally
	t.Log("corpus | other-stream compared/all | samples compared/all | deterministic matched/compared | results: reference seed, other seed, never | exact matched/replayed | waiting: generator (start in vector), generator (ORACLE.md tick), start")
	for _, c := range corpora {
		t.Log(pvRandomLine(c, per[c]))
		sum.add(per[c])
	}
	t.Log(pvRandomLine("all", &sum))
}

func pvRandomLine(corpus string, tl *pvRandomTally) string {
	return fmt.Sprintf("| %s | %d/%d | %d/%d | %d/%d | %d, %d, %d | %d/%d | %d, %d, %d |",
		corpus, tl.streamCompared, tl.stream, tl.sampleCompared, tl.sample,
		tl.detMatch, tl.detMatch+tl.detMiss,
		tl.resRef, tl.resSome, tl.resNever,
		tl.exMatch, tl.exMatch+tl.exMiss, tl.waitVector, tl.waitTable, tl.waitStart)
}

// TestRandomStreamSeam checks the replay seam with a stand-in generator:
// where each year's start comes from, that every year of a replay draws
// from its own start, and what a replay checks.
func TestRandomStreamSeam(t *testing.T) {
	var v *pvVector
	for _, w := range loadVectors(t) {
		if w.path == "kx004/s1.json" {
			v = w
		}
	}
	if v == nil {
		t.Fatal("no kx004/s1.json")
	}
	var starts []oraclerng.Start
	var recs []*oraclerng.Recorder
	seed := func(s oraclerng.Start) (oraclerng.Rand, bool) {
		if !s.HasTick {
			return nil, false
		}
		starts = append(starts, s)
		r := &oraclerng.Recorder{R: rand.New(rand.NewSource(int64(s.Tick)))}
		recs = append(recs, r)
		return r, true
	}

	// Starts: ORACLE.md's tick for a labelled stream, none for the
	// unlabelled stream or a cycles value without a measured tick.
	if s := pvStart(v, "cycles 10500", 1); !s.HasTick || s.Tick != 384 {
		t.Errorf("cycles 10500: %+v, want tick 384", s)
	}
	for _, label := range []string{"", "cycles 25000"} {
		if s := pvStart(v, label, 1); s.Recorded() {
			t.Errorf("%q: %+v, want nothing recorded", label, s)
		}
		if r, why := pvReplay(v, label, seed); r != nil || !strings.Contains(why, "no start recorded") {
			t.Errorf("%q: replay %v, %q", label, r != nil, why)
		}
	}
	if r, why := pvReplay(v, "cycles 10500", nil); r != nil || why != "no generator" {
		t.Errorf("no generator: replay %v, %q", r != nil, why)
	}

	// A vector's random_start wins over the table; a state-only start
	// reaches the generator as recorded, and a null entry records nothing.
	w := *v
	tick := 999
	w.RandomStart = []pvRandomStart{
		{Stream: "cycles 10500", Year: 1, Tick: &tick},
		{Stream: "cycles 25000", Year: 1, State: json.RawMessage(`"00ff"`)},
		{Stream: "cycles 9800", Year: 1, State: json.RawMessage(`null`)},
	}
	if s := pvStart(&w, "cycles 10500", 1); s.Tick != 999 {
		t.Errorf("random_start tick: %+v, want 999", s)
	}
	if s := pvStart(&w, "cycles 25000", 1); string(s.State) != `"00ff"` || s.HasTick {
		t.Errorf("random_start state: %+v", s)
	}
	if r, why := pvReplay(&w, "cycles 25000", seed); r != nil || !strings.Contains(why, "cannot start") {
		t.Errorf("state-only start with a tick-only generator: replay %v, %q", r != nil, why)
	}
	if s := pvStart(&w, "cycles 9800", 1); s.Tick != 384 {
		t.Errorf("null random_start falls back to the table: %+v", s)
	}
	if got := pvStartSource(&w, "cycles 10500"); got != "vector" {
		t.Errorf("source %q, want vector", got)
	}
	if got := pvStartSource(&w, "cycles 9800"); got != "table" {
		t.Errorf("source %q, want table", got)
	}

	// A replay draws every year from its own start and checks the
	// unlabelled expectations and its stream's, nothing else.
	replay, why := pvReplay(v, "cycles 10500", seed)
	if replay == nil {
		t.Fatal(why)
	}
	starts, recs = nil, nil
	other := 0
	runVectorWith(v, pvRun{stream: "cycles 10500", replay: replay, observe: func(c pvCase, i int, msg string) {
		e := c.Expect[i]
		skipped := strings.HasPrefix(msg, "skip: stream ")
		if e.Stream != "" && e.Stream != "cycles 10500" {
			other++
			if !skipped {
				t.Errorf("%s: another stream's expectation was compared: %q", c.ID, msg)
			}
		} else if skipped {
			t.Errorf("%s: %q", c.ID, msg)
		}
	}})
	if other == 0 {
		t.Error("kx004/s1 has no other stream's expectation")
	}
	if len(starts) != v.Years {
		t.Fatalf("%d starts for %d years", len(starts), v.Years)
	}
	for _, s := range starts {
		if s.Tick != 384 {
			t.Errorf("start %+v, want tick 384", s)
		}
	}
	for _, r := range recs {
		if len(r.Bounds) == 0 {
			t.Error("a year drew nothing from its replay")
		}
	}
}

func TestRandomStreamRunTruthTable(t *testing.T) {
	replay := func(int) Rand { return highRand{} }
	for _, c := range []struct {
		run             pvRun
		e               pvExpect
		checks, sample1 bool
	}{
		{pvRun{}, pvExpect{}, true, false},
		{pvRun{}, pvExpect{Sample: true}, true, true},
		{pvRun{}, pvExpect{Stream: "cycles 1"}, false, true},
		{pvRun{stream: pvAllStreams}, pvExpect{Stream: "cycles 1"}, true, true},
		{pvRun{stream: "cycles 1", replay: replay}, pvExpect{Stream: "cycles 1"}, true, false},
		{pvRun{stream: "cycles 1", replay: replay}, pvExpect{Stream: "cycles 2"}, false, false},
		{pvRun{stream: pvAllStreams, replay: replay}, pvExpect{Stream: "cycles 2"}, false, false},
		{pvRun{stream: "cycles 1", replay: replay}, pvExpect{Sample: true}, true, false},
	} {
		if got := c.run.checks(c.e); got != c.checks {
			t.Errorf("stream %q replay %v, %+v: checks %v", c.run.stream, c.run.replay != nil, c.e, got)
		}
		if got := c.run.sample(c.e); got != c.sample1 {
			t.Errorf("stream %q replay %v, %+v: sample %v", c.run.stream, c.run.replay != nil, c.e, got)
		}
	}
}

// TestConfirmedYearDrawOrderKX004: in the KX-004 states (no fleets in
// motion, no battles or drops), the year draws the player shuffle, then
// mining, then the random events: comet rand(20), climate change rand(20),
// new minerals rand(15 − size) (KERNEL.md "Random draws" items 1, 4 and 5
// and "Random events", d4eafd9; CONFIRMED by KX-004, KB-1C, KB-3A). With
// every draw at its highest value no event fires. The S1 to S9 states mine
// twice, so the events begin at draw 4 (KERNEL.md, MEASURED).
func TestConfirmedYearDrawOrderKX004(t *testing.T) {
	n := 0
	for _, v := range loadVectors(t) {
		if v.corpus != "kx004" {
			continue
		}
		n++
		rec := &oraclerng.Recorder{R: highRand{}}
		replay := func(y int) Rand {
			if y == 1 {
				return rec
			}
			return highRand{}
		}
		runVectorWith(v, pvRun{replay: replay})
		b := rec.Bounds
		i := 2
		for i < len(b) && b[i] == 100 {
			i++
		}
		if len(b) < i+3 || b[0] != 2 || b[1] != 1 || b[i] != 20 || b[i+1] != 20 || b[i+2] != 15 {
			t.Errorf("%s: draws %v, want 2 1, mining 100s, then 20 20 15", v.path, b)
			continue
		}
		if v.path != "kx004/s10.json" && i != 4 {
			t.Errorf("%s: events begin at draw %d, want 4 (draws %v)", v.path, i, b)
		}
	}
	if n == 0 {
		t.Fatal("no kx004 vectors")
	}
}
