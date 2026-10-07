package engine

import (
	"reflect"
	"testing"
)

// Score vectors from stars-elegy KERNEL.md "Scores and victory
// conditions" (CONFIRMED, KX-003 S1).

func TestConfirmedScoreTerms(t *testing.T) {
	// Tech: 6 × level 26 → 516; levels 3, 4, 6, 7, 9, 10 → 69.
	sum := 0
	for _, l := range []int{3, 4, 6, 7, 9, 10} {
		sum += techScore(l)
	}
	if techScore(26)*6 != 516 || sum != 69 {
		t.Errorf("tech %d, %d", techScore(26)*6, sum)
	}
	// Ships: N 5, U 7, E 3, C 10 → 2 + 6 + 26; N 4, U 1, E 5, C 0 → 8.
	if got := shipsScore(5, 7, 3, 10); got != 34 {
		t.Errorf("ships S1 player 0: %d", got)
	}
	if got := shipsScore(4, 1, 5, 0); got != 8 {
		t.Errorf("ships S1 player 1: %d", got)
	}
	// Planets: 287, 1001, 1000, 5739, 7412 → 16; resources 1552 → 51.
	n := 0
	for _, p := range []int{287, 1001, 1000, 5739, 7412} {
		n += min(6, ceilDiv(p, 1000))
	}
	if n != 16 || 1552/30 != 51 {
		t.Errorf("planets %d", n)
	}
}

func TestConfirmedShipClasses(t *testing.T) {
	// 4 Omega Torpedoes 1896 (escort), 5 → 2370 (capital); 2 Cherry Bombs
	// 140 (escort); an X-Ray Laser ship is an escort, an unarmed one 0.
	part := func(name string) Part {
		pc, ok := Components().Lookup(name)
		if !ok {
			t.Fatalf("no %s", name)
		}
		p, err := pc.Part()
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	omega, cherry, xray := part("Omega Torpedo"), part("Cherry Bomb"), part("X-Ray Laser")
	for _, c := range []struct {
		slots []Slot
		want  int
	}{{[]Slot{{omega, 4}}, 1896}, {[]Slot{{omega, 5}}, 2370}, {[]Slot{{cherry, 2}}, 140}, {nil, 0}} {
		if got := shipPower(Design{Slots: c.slots}); got != c.want {
			t.Errorf("%v: power %d, want %d", c.slots, got, c.want)
		}
	}
	if got := shipPower(Design{Slots: []Slot{{xray, 1}}}); got <= 0 || got >= 2000 {
		t.Errorf("X-Ray Laser: power %d, want an escort", got)
	}
}

func TestPredictionBeamPower(t *testing.T) {
	// Range 2, damage 40, 2 beams: 5·40·2/4 = 100; one 20% capacitor:
	// factor 1200 → 100·120/100 = 120; speed code 0 (no engine): 120 −
	// 48 = 72. A sapper divides the beam term by 3.
	beam := Part{Kind: PartBeam, Range: 2, Damage: 40}
	capac := Part{Kind: PartElectrical, Capacitor: 20}
	if got := shipPower(Design{Slots: []Slot{{beam, 2}, {capac, 1}}}); got != 72 {
		t.Errorf("beam with capacitor: %d, want 72", got)
	}
	beam.Sapper = true
	if got := shipPower(Design{Slots: []Slot{{beam, 2}}}); got != 33-13 {
		t.Errorf("sapper: %d, want 20", got)
	}
}

func TestConfirmedVictoryFlags(t *testing.T) {
	// S1: 24 planets, planets 20% (v 0), lead 20% (v 0), capital ships 10
	// (v 0, disabled but flagged), tech 22 in 4 fields (v 14, 2). Player 0
	// (623, 5 planets, all fields 26, C 10) → 0x0ae0; player 1 (101, 4
	// planets) → 0x0021.
	g := Game{Year: 2425, Players: make([]Player, 2), Planets: make([]Planet, 24),
		Victory: Victory{TechLevel: 14, TechFields: 2, Score: 10, Resources: 9, Highest: 7}}
	for f := range NumFields {
		g.Players[0].Research.Levels[f] = 26
	}
	g.Players[1].Research.Levels = [NumFields]int{3, 4, 6, 7, 9, 10}
	recs := []ScoreRecord{{Player: 0, Score: 623, Planets: 5, Resources: 1552, Capital: 10, Rank: 1},
		{Player: 1, Score: 101, Planets: 4, Resources: 498, Rank: 2}}
	g.victoryFlags(recs)
	if recs[0].Flags != 0x0ae0 || recs[1].Flags != 0x0021 {
		t.Errorf("flags %#x %#x, want 0xae0 0x21", recs[0].Flags, recs[1].Flags)
	}
}

func TestPredictionDecide(t *testing.T) {
	// A player with no planets and no ships dies; the survivor wins.
	g := Game{Year: 2410, Players: []Player{{Race: pgRace()}, {Race: pgRace()}}, Planets: []Planet{{ID: 1, Owner: 0, Population: 100, Env: [3]int{50, 50, 50}}}}
	ev := g.decide(g.scores())
	want := []Event{
		{Kind: EventPlayerDied, Player: 0, Planet: -1, Fleet: -1, Count: 1},
		{Kind: EventGameWon, Player: 0, Planet: -1, Fleet: -1, Count: 1},
		{Kind: EventGameLost, Player: 1, Planet: -1, Fleet: -1, Count: 1},
	}
	if !reflect.DeepEqual(ev, want) || !g.Players[1].Dead || !g.Decided {
		t.Errorf("events %+v dead %v decided %v", ev, g.Players[1].Dead, g.Decided)
	}
	// Decided again the next year, with the messages again (KERNEL.md "Scores and victory conditions").
	if ev := g.decide(g.scores()); len(ev) != 2 || !g.Decided {
		t.Errorf("next year: events %+v", ev)
	}

	// Two live players: an enabled condition met wins only from the
	// minimum years ((0+3)·10 = 30).
	for _, c := range []struct {
		year int
		won  bool
	}{{2429, false}, {2430, true}} {
		g := Game{Year: c.year, Players: make([]Player, 2), Planets: []Planet{{ID: 1, Owner: 0}, {ID: 2, Owner: 1}},
			Victory: Victory{Planets: 2, Needed: 3}} // needed capped at 1 enabled
		g.Victory.Enabled[VictoryPlanets] = true // 30%: 1 of 2 planets → round(0.6) = 1 each
		ev := g.decide(g.scores())
		if g.Decided != c.won || c.won && (len(ev) != 2 || ev[0].Kind != EventGameWon || ev[0].Count != 2) {
			t.Errorf("year %d: decided %v events %+v", c.year, g.Decided, ev)
		}
	}
}

func TestConfirmedPublicScores(t *testing.T) {
	// E1: own record only through 2419, everyone's from 2420; E0 (option
	// off): own only.
	recs := []ScoreRecord{{Player: 0}, {Player: 1}}
	for _, c := range []struct {
		year   int
		public bool
		want   int
	}{{2419, true, 1}, {2420, true, 2}, {2440, false, 1}} {
		g := Game{Year: c.year, PublicScores: c.public, Players: make([]Player, 2)}
		if got := len(g.visibleScores(0, recs)); got != c.want {
			t.Errorf("%d public %v: %d records, want %d", c.year, c.public, got, c.want)
		}
	}
}

func TestPredictionVictoryAnswers(t *testing.T) {
	// KERNEL.md: needed 0 means nobody wins by conditions; a tie
	// for the top score flags nobody for the lead; the record's starbase
	// count leaves out Orbital Forts (no dock).
	g := Game{Year: 2440, Players: make([]Player, 2), Planets: []Planet{{ID: 1, Owner: 0}, {ID: 2, Owner: 1}}}
	g.Victory.Enabled[VictoryPlanets] = true
	if ev := g.decide(g.scores()); len(ev) != 0 || g.Decided {
		t.Errorf("needed 0: %+v", ev)
	}
	recs := []ScoreRecord{{Player: 0, Score: 50, Rank: 1}, {Player: 1, Score: 50, Rank: 1}}
	g.victoryFlags(recs)
	if recs[0].Flags&(0x40<<VictoryLead) != 0 || recs[1].Flags&(0x40<<VictoryLead) != 0 {
		t.Errorf("tie flagged lead: %#x %#x", recs[0].Flags, recs[1].Flags)
	}
	g.Planets[0].HasStarbase, g.Planets[0].StarbaseHull = true, 1 // Orbital Fort
	g.Planets[1].HasStarbase, g.Planets[1].StarbaseDock = true, true
	if r := g.scores(); r[0].Starbases != 0 || r[1].Starbases != 1 || r[1].Score != 3 {
		t.Errorf("starbases %+v", r)
	}
}
