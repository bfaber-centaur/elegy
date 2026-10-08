package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Jump chance and stability (OBJECTS.md "Yearly movement", MEASURED
// OB-025; "Stability", BINARY-ONLY): class 0, 1, 2 cannot jump before 15,
// 10, 5 years; the chance caps at 6%.
func TestConfirmedJumpChance(t *testing.T) {
	cases := []struct{ class, years, pct int }{
		{0, 0, 0}, {0, 14, 0}, {0, 15, 1}, {1, 9, 0}, {1, 10, 1}, {2, 4, 0}, {2, 5, 1},
		{2, 30, 6}, {2, 100, 6}, {1, 30, 5},
	}
	for _, c := range cases {
		e := WormholeEnd{Class: c.class, Years: c.years}
		if got := JumpChance(e); got != c.pct {
			t.Errorf("class %d at %d years: %d%%, want %d", c.class, c.years, got, c.pct)
		}
	}
	if got := StabilityName(WormholeEnd{Class: 2}); got != "Rock Solid" {
		t.Errorf("class 2 at 0 years: %s", got)
	}
	if got := StabilityName(WormholeEnd{Class: 2, Years: 30}); got != "Extremely Volatile" {
		t.Errorf("class 2 at 30 years: %s", got)
	}
}

// Jiggle: years + 1, at most 12 per axis, never the old position, class
// kept (CONFIRMED OB-005, OB-017, OB-025).
func TestConfirmedJiggle(t *testing.T) {
	l := newLab(t)
	r := engine.Rand(&lcg{1})
	s := &Space{Wormholes: []Wormhole{{Ends: [2]WormholeEnd{
		{Pos: at(400, 400), Class: 1},
		{Pos: at(1200, 1200), Class: 0},
	}}}}
	for range 200 {
		before := s.Wormholes[0].Ends
		moves := s.MoveWormholes(l.g, 1600, r)
		for i, mv := range moves {
			e := s.Wormholes[0].Ends[i]
			if mv.Jumped {
				if e.Years != 0 || e.Class != before[i].Class {
					t.Fatalf("jump: %+v", e)
				}
				continue
			}
			dx, dy := mv.To.X-mv.From.X, mv.To.Y-mv.From.Y
			if dx < -12 || dx > 12 || dy < -12 || dy > 12 || mv.To == mv.From {
				t.Fatalf("jiggle %v → %v", mv.From, mv.To)
			}
			if e.Years != before[i].Years+1 || e.Class != before[i].Class {
				t.Fatalf("jiggle years/class: %+v from %+v", e, before[i])
			}
		}
	}
}

// A jump resets the years and clears who knows the end, but not who
// knows where it leads (CONFIRMED OB-025, WT-004).
func TestConfirmedJump(t *testing.T) {
	l := newLab(t)
	s := &Space{Wormholes: []Wormhole{{Ends: [2]WormholeEnd{
		{Pos: at(400, 400), Class: 2, Years: 30, Known: []bool{true}, DestKnown: []bool{true}},
		{Pos: at(1200, 1200)},
	}}}}
	// rand(100) = 0 < 6 jumps; then uniform tries; the partner jiggles.
	r := script{0}
	moves := s.MoveWormholes(l.g, 1600, &r)
	e := s.Wormholes[0].Ends[0]
	if !moves[0].Jumped || e.Years != 0 || e.KnownBy(0) || !e.DestinationKnownBy(0) || e.Class != 2 {
		t.Errorf("jump: %+v %+v", moves[0], e)
	}
	if moves[1].Jumped {
		t.Errorf("class 0 at 0 years jumped")
	}
}

// Transit (OBJECTS.md "Travel", CONFIRMED OB-005 C, D, WT-001): the fleet
// lands on the partner end; both ends become known and record the
// destination for its owner only; the destination shows only while the
// partner is seen.
func TestConfirmedTransit(t *testing.T) {
	l := newLab(t)
	fi := l.fleet(0, at(400, 400), "Tank", 1)
	s := &Space{Wormholes: []Wormhole{{Ends: [2]WormholeEnd{{Pos: at(400, 400)}, {Pos: at(1200, 900)}}}}}
	s.Transit(l.g, fi, 0, 0)
	if l.g.Fleets[fi].Pos != at(1200, 900) {
		t.Errorf("exit at %v", l.g.Fleets[fi].Pos)
	}
	for i, e := range s.Wormholes[0].Ends {
		if !e.KnownBy(0) || !e.DestinationKnownBy(0) || e.KnownBy(1) || e.DestinationKnownBy(1) {
			t.Errorf("end %d knowledge %+v", i, e)
		}
	}
	if p, ok := s.Destination(0, 0, 0, true); !ok || p != at(1200, 900) {
		t.Errorf("destination %v %v", p, ok)
	}
	if _, ok := s.Destination(0, 0, 0, false); ok {
		t.Error("destination shown with the partner unseen")
	}
	if _, ok := s.Destination(0, 0, 1, true); ok {
		t.Error("destination shown to a player who never transited")
	}
}

// Badness rejects a try outside the galaxy or on an object (OBJECTS.md
// "Placement badness"); the creation flags are tested in newgame.
func TestPredictionBadnessRejects(t *testing.T) {
	sr := Surroundings{Width: 400, Objects: []engine.Point{at(200, 200)}}
	for _, p := range []engine.Point{at(-1, 50), at(50, 400), at(200, 200)} {
		if got := sr.Badness(p); got != Rejected {
			t.Errorf("%v: badness %d, want rejected", p, got)
		}
	}
	if got := sr.Badness(at(100, 100)); got != 0 {
		t.Errorf("open space: %d", got)
	}
}

// lcg is a small deterministic engine.Rand for loops.
type lcg struct{ x uint64 }

func (l *lcg) Intn(n int) int {
	l.x = l.x*6364136223846793005 + 1442695040888963407
	return int((l.x >> 33) % uint64(n))
}
