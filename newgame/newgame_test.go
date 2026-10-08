package newgame

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
	"github.com/bfaber-centaur/elegy/races"
)

// Test names follow the engine's convention: TestConfirmed* tests rules
// UNIVERSE.md marks CONFIRMED (vectors from the UG games are ground
// truth); TestPrediction* tests BINARY-ONLY rules; TestElegyDecision*
// tests Elegy's own choices. Because Elegy does not reproduce the
// original's random stream, game-level tests check invariants and
// distributions over Elegy seeds, and exact vectors go through the rule
// functions directly.

// testRace is an ordinary race centred at 50 with range 15–85 on every
// axis.
func testRace(prt engine.PRT) engine.Race {
	r := engine.Race{
		PRT: prt, GrowthRate: 15,
		ColonistsPerResource: 1000, FactoryOutput: 10, FactoriesOperated: 10,
		MinesOperated: 10, MineOutput: 10, FactoryCost: 10, MineCost: 5,
	}
	for i := range r.Env {
		r.Env[i] = engine.EnvRange{Center: 50, Low: 15, High: 85}
	}
	return r
}

var allPRTs = []engine.PRT{
	engine.PRTHyperExpansion, engine.PRTSuperStealth, engine.PRTWarMonger,
	engine.PRTClaimAdjuster, engine.PRTInnerStrength, engine.PRTSpaceDemolition,
	engine.PRTPacketPhysics, engine.PRTInterstellarTraveler,
	engine.PRTAlternateReality, engine.PRTJackOfAllTrades,
}

func human(prt engine.PRT) PlayerSetup {
	return PlayerSetup{Race: races.Design{Race: testRace(prt), Spend: int(SpendSurfaceMinerals)}}
}

func computer(prt engine.PRT, l Level) PlayerSetup {
	return PlayerSetup{Race: races.Design{Race: testRace(prt), Spend: int(SpendMines)}, Computer: true, Level: l}
}

// withPoints makes a human race worth at least min points by raising its
// colonists per resource, then lowering its growth (each step adds
// points), so that game creation keeps it (RACES.md "At game creation").
func withPoints(ps PlayerSetup, min int) PlayerSetup {
	d := &ps.Race
	for races.Points(*d) < min && d.Race.ColonistsPerResource < races.MaxColonists {
		d.Race.ColonistsPerResource += 100
	}
	for races.Points(*d) < min && d.Race.GrowthRate > 1 {
		d.Race.GrowthRate--
	}
	return ps
}

func generate(t *testing.T, s Settings, seed uint64) Result {
	t.Helper()
	players := append([]PlayerSetup(nil), s.Players...)
	for i, p := range players {
		if !p.Computer && !p.Race.Random {
			players[i] = withPoints(p, 0)
		}
	}
	s.Players = players
	if s.Rules == (engine.Ruleset{}) {
		s.Rules = engine.ElegyRules()
	}
	res, err := Generate(s, NewRand(seed))
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}
	return res
}

func twoPlayers() []PlayerSetup {
	return []PlayerSetup{human(engine.PRTSuperStealth), computer(engine.PRTInnerStrength, Standard)}
}

// UNIVERSE.md "Count" table (CONFIRMED): the nominal count N for every
// cell, and the generated count for the cells the UG games measured
// (PARITY.md "Planet counts"). In the other cells the spacing pass can
// remove more than M − N candidates, so a game has at most N planets.
func TestConfirmedPlanetCounts(t *testing.T) {
	want := [5][4]int{
		{24, 32, 40, 60},
		{96, 128, 160, 240},
		{216, 288, 360, 540},
		{384, 512, 640, 960},
		{600, 800, 999, 999},
	}
	measured := map[[2]int]bool{
		{0, 0}: true, {0, 1}: true, {1, 0}: true, {1, 1}: true, {1, 2}: true,
		{2, 1}: true, {2, 2}: true, {2, 3}: true, {3, 1}: true, {4, 1}: true,
	}
	for size := Tiny; size <= Huge; size++ {
		for d := Sparse; d <= Packed; d++ {
			w := want[size][d]
			if n := planetCount(size, d); n != w {
				t.Errorf("planetCount(%d, %d) = %d, want %d", size, d, n, w)
			}
			for seed := range uint64(3) {
				res := generate(t, Settings{Size: size, Density: d, Players: twoPlayers()}, seed)
				n := len(res.Game.Planets)
				if n > w || measured[[2]int{int(size), int(d)}] && n != w {
					t.Errorf("size %d density %d seed %d: %d planets, want %d", size, d, seed, n, w)
				}
			}
		}
	}
}

// Large packed and huge dense or packed come near the 999-candidate limit
// and their count depends on the seed. UNIVERSE.md gives the observed
// cells (912 in UG04, 940 in UG05) as single observations and 316 sampled
// streams as BINARY-ONLY ranges: 899–940 large packed, 930–962 huge dense
// or packed. Elegy's own spacing pass should land in the same region.
func TestPredictionSeedDependentCounts(t *testing.T) {
	cases := []struct {
		size    Size
		density Density
		lo, hi  int
	}{
		{Large, Packed, 899, 940},
		{Huge, Dense, 930, 962},
		{Huge, Packed, 930, 962},
	}
	const margin = 10
	for _, c := range cases {
		if n := planetCount(c.size, c.density); n < c.hi {
			t.Errorf("size %d density %d: nominal %d below the observed range", c.size, c.density, n)
		}
		for seed := range uint64(40) {
			res := generate(t, Settings{Size: c.size, Density: c.density, Players: twoPlayers()}, seed)
			if n := len(res.Game.Planets); n < c.lo-margin || n > c.hi+margin {
				t.Errorf("size %d density %d seed %d: %d planets, outside %d–%d ± %d",
					c.size, c.density, seed, n, c.lo, c.hi, margin)
			}
		}
	}
}

// Huge dense and huge packed are the same galaxy for the same seed
// (UNIVERSE.md "Count", CONFIRMED UG05, UG15).
func TestConfirmedHugeDenseEqualsPacked(t *testing.T) {
	for seed := range uint64(3) {
		a := generate(t, Settings{Size: Huge, Density: Dense, Players: twoPlayers()}, seed)
		b := generate(t, Settings{Size: Huge, Density: Packed, Players: twoPlayers()}, seed)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("seed %d: huge dense and huge packed differ", seed)
		}
	}
}

// Every planet lies in 1010 .. 1010+W−20 and, without clumping, no two
// are within 12 ly (UNIVERSE.md "Count", CONFIRMED).
func TestConfirmedPlacementAndSpacing(t *testing.T) {
	for size := Tiny; size <= Huge; size++ {
		s := Settings{Size: size, Density: Packed, Players: twoPlayers()}
		res := generate(t, s, 7)
		w := s.Width()
		ps := res.Game.Planets
		for i, p := range ps {
			if p.ID != i {
				t.Fatalf("planet %d has id %d", i, p.ID)
			}
			for _, v := range []int{p.Pos.X, p.Pos.Y} {
				if v < 1010 || v > 1010+w-20 {
					t.Errorf("size %d planet %d at %v, outside 1010..%d", size, i, p.Pos, 1010+w-20)
				}
			}
			for j := i + 1; j < len(ps); j++ {
				if d2(p.Pos, ps[j].Pos) <= 144 {
					t.Errorf("size %d planets %d and %d %v %v are within 12 ly", size, i, j, p.Pos, ps[j].Pos)
				}
			}
		}
	}
}

// Every planet has a distinct name index in 0..998 (UNIVERSE.md "Names",
// CONFIRMED).
func TestConfirmedNamesDistinct(t *testing.T) {
	res := generate(t, Settings{Size: Huge, Density: Packed, Players: twoPlayers()}, 1)
	seen := map[int]bool{}
	for i, k := range res.NameIndex {
		if k < 0 || k >= MaxPlanets || seen[k] {
			t.Fatalf("planet %d: name index %d out of range or repeated", i, k)
		}
		seen[k] = true
	}
	if len(seen) != len(res.Game.Planets) {
		t.Fatalf("%d names for %d planets", len(seen), len(res.Game.Planets))
	}
}

// unowned yields the planets that are neither homeworlds nor second
// planets.
func unowned(res Result) []engine.Planet {
	var out []engine.Planet
	for _, p := range res.Game.Planets {
		if p.Owner == engine.NoOwner {
			out = append(out, p)
		}
	}
	return out
}

// Environment, artifacts and concentrations of ordinary planets
// (UNIVERSE.md "Environment", "Mineral concentrations", CONFIRMED),
// checked as ranges and distributions over a few huge galaxies.
func TestConfirmedPlanetDistributions(t *testing.T) {
	var ps []engine.Planet
	artifacts := 0
	for seed := range uint64(4) {
		res := generate(t, Settings{Size: Huge, Density: Normal, Players: twoPlayers()}, seed)
		for _, p := range unowned(res) {
			ps = append(ps, p)
			if res.Artifact[p.ID] {
				artifacts++
			}
		}
	}
	n := float64(len(ps))
	var sum [3]float64
	edge := [3]int{} // values 1..10 per axis
	low := 0         // concentrations hit by an override (1..30)
	for _, p := range ps {
		for a, v := range p.Env {
			if v < 1 || v > 99 {
				t.Fatalf("planet %d axis %d = %d, outside 1..99", p.ID, a, v)
			}
			sum[a] += float64(v)
		}
		for a, v := range p.Env {
			if v <= 10 {
				edge[a]++
			}
		}
		for m, d := range p.Deposits {
			c := d.Concentration
			if c < 1 || c > 119 {
				t.Fatalf("planet %d mineral %d concentration %d, outside 1..119", p.ID, m, c)
			}
			if c <= 30 {
				low++
			}
		}
		if p.Surface != (engine.Minerals{}) {
			t.Fatalf("unowned planet %d has surface minerals %v", p.ID, p.Surface)
		}
	}
	for a, s := range sum {
		if mean := s / n; mean < 47 || mean > 53 {
			t.Errorf("axis %d mean %.1f, want about 50", a, mean)
		}
	}
	// 1 + rand(90) + rand(10) puts 55/900 = 6.1% of planets at 1..10;
	// uniform radiation 1 + rand(99) puts 10/99 = 10.1% there.
	for a, want := range [3]float64{0.061, 0.061, 0.101} {
		if f := float64(edge[a]) / n; f < want-0.02 || f > want+0.02 {
			t.Errorf("axis %d: fraction at 1..10 %.3f, want about %.3f", a, f, want)
		}
	}
	if f := float64(artifacts) / n; f < 0.30 || f > 0.37 {
		t.Errorf("artifact fraction %.3f, want about 1/3", f)
	}
	// Expected overrides per planet: (4 + 3·2 + 2·4 + 1·2 + 1·9)/27 = 29/27,
	// a little less in distinct minerals since one can be hit twice.
	if f := float64(low) / n; f < 0.85 || f > 1.10 {
		t.Errorf("low concentrations per planet %.3f, want a little under 29/27", f)
	}
}

// Options: maximum minerals sets every concentration to 100; BBS raises
// those below 40 by 5; no random events means no artifacts and no
// wormholes (UNIVERSE.md, CONFIRMED UG02, UG09, UG03, UG08).
func TestConfirmedOptions(t *testing.T) {
	res := generate(t, Settings{Size: Small, Density: Normal, MaxMinerals: true, Players: twoPlayers()}, 3)
	for _, p := range unowned(res) {
		for _, d := range p.Deposits {
			if d.Concentration != 100 {
				t.Fatalf("max minerals: planet %d concentration %d", p.ID, d.Concentration)
			}
		}
	}
	res = generate(t, Settings{Size: Medium, Density: Normal, BBS: true, Players: twoPlayers()}, 3)
	for _, p := range unowned(res) {
		for _, d := range p.Deposits {
			// Base draws below 40 move to 36..44; overrides are 1..30.
			if c := d.Concentration; c > 30 && c < 36 {
				t.Fatalf("BBS: planet %d concentration %d", p.ID, c)
			}
		}
	}
	res = generate(t, Settings{Size: Huge, Density: Normal, NoRandomEvents: true, Players: twoPlayers()}, 3)
	for i, a := range res.Artifact {
		if a {
			t.Fatalf("no random events: planet %d has an artifact", i)
		}
	}
	if len(res.Wormholes) != 0 {
		t.Fatalf("no random events: %d wormholes", len(res.Wormholes))
	}
}

// script is a Rand that returns fixed draws, for rule vectors.
type script struct {
	t     *testing.T
	draws []int
}

func (s *script) Intn(n int) int {
	s.t.Helper()
	if len(s.draws) == 0 {
		s.t.Fatalf("script ran out of draws (rand(%d))", n)
	}
	v := s.draws[0]
	s.draws = s.draws[1:]
	if v < 0 || v >= n {
		s.t.Fatalf("scripted draw %d is outside rand(%d)", v, n)
	}
	return v
}

// Concentration vectors worked from UNIVERSE.md "Mineral concentrations"
// (CONFIRMED rule).
func TestConfirmedConcentrationRule(t *testing.T) {
	cases := []struct {
		name      string
		bbs       bool
		radiation int
		draws     []int
		want      [3]int
	}{
		// 31+a+b per mineral, then r = 26: no override.
		{"plain", false, 50, []int{0, 0, 44, 44, 10, 5, 26}, [3]int{31, 119, 46}},
		// Radiation 90: c += rand(99−c)/2 when c < 99; 31 + rand(68)=67 → +33.
		{"radiation", false, 90, []int{0, 0, 67, 44, 44, 44, 44, 26}, [3]int{64, 119, 119}},
		// BBS: 31 → 36; 39 → 44; 40 stays.
		{"bbs", true, 50, []int{0, 0, 4, 4, 5, 4, 26}, [3]int{36, 44, 40}},
		// r = 0: four overrides; boranium is hit twice, the last wins.
		{"four overrides", false, 50, []int{10, 10, 10, 10, 10, 10, 0, 1, 4, 1, 9, 0, 19, 2, 29}, [3]int{20, 10, 30}},
		// r = 9..17: one override.
		{"one override", false, 50, []int{10, 10, 10, 10, 10, 10, 17, 2, 0}, [3]int{51, 51, 1}},
	}
	for _, c := range cases {
		g := &generator{s: Settings{BBS: c.bbs}, rng: &script{t: t, draws: c.draws}}
		if got := g.concentrations(c.radiation); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	g := &generator{s: Settings{MaxMinerals: true}, rng: &script{t: t}}
	if got := g.concentrations(95); got != [3]int{100, 100, 100} {
		t.Errorf("max minerals: %v", got)
	}
}

// Clumping moves a planet toward its nearest neighbour by the d² table
// (UNIVERSE.md "Clumping option", CONFIRMED UG04, UG10).
func TestConfirmedClumpingTable(t *testing.T) {
	cases := []struct {
		b    engine.Point
		want engine.Point
	}{
		{engine.Point{X: 1100, Y: 1000}, engine.Point{X: 1066, Y: 1000}}, // d² 10000: (a+2b)/3
		{engine.Point{X: 1040, Y: 1000}, engine.Point{X: 1020, Y: 1000}}, // 1600: (a+b)/2
		{engine.Point{X: 1025, Y: 1000}, engine.Point{X: 1008, Y: 1000}}, // 625: (2a+b)/3
		{engine.Point{X: 1018, Y: 1000}, engine.Point{X: 1003, Y: 1000}}, // 324: (4a+b)/5
		{engine.Point{X: 1012, Y: 1000}, engine.Point{X: 1000, Y: 1000}}, // 144: stays
	}
	for _, c := range cases {
		pos := []engine.Point{{X: 1000, Y: 1000}, c.b, {X: 1900, Y: 1900}}
		g := &generator{rng: &script{t: t, draws: []int{0}}}
		g.clump(pos, 1)
		if pos[0] != c.want {
			t.Errorf("toward %v: %v, want %v", c.b, pos[0], c.want)
		}
	}
}

// Homeworld spacing (UNIVERSE.md "Homeworld placement", CONFIRMED):
// every pair at least the (possibly shrunk) minimum apart, each later
// homeworld within the maximum of an earlier one and in its box.
func TestConfirmedHomeworldPlacement(t *testing.T) {
	// v for a medium galaxy (W 1200), 4 players, farther positions:
	// a = 1200²/4 − 7200 = 352800 → 317520; v = 7200 + 2·317520/3.
	if sp := homeworldSpacing(1200, 4, Farther); sp.v != 218880 || sp.min != 196992 || sp.max != 255360 {
		t.Errorf("homeworldSpacing = %+v", sp)
	}
	// Tiny, 16 players: a = 10000 − 2400 = 7600 → 6840; v = 2400 + 6840.
	if sp := homeworldSpacing(400, 16, Distant); sp.v != 9240 {
		t.Errorf("tiny, 16 players: v = %d, want 9240", sp.v)
	}
	for _, size := range []Size{Tiny, Small, Medium, Large, Huge} {
		for _, pos := range []Positions{Close, Moderate, Farther, Distant} {
			for _, np := range []int{1, 2, 4, 8, 16} {
				s := Settings{Size: size, Density: Normal, Positions: pos}
				for range np {
					s.Players = append(s.Players, human(engine.PRTInnerStrength))
				}
				g := &generator{s: s, w: s.Width(), rng: NewRand(uint64(np) * 31), cat: engine.Components()}
				g.makePlanets(g.placePlanets())
				hws, sp, err := g.homeworldList(np)
				if err != nil {
					t.Fatalf("size %d pos %d players %d: %v", size, pos, np, err)
				}
				w := s.Width()
				var wide box
				switch {
				case np >= 5:
					wide = box{Origin + w/20, Origin + 19*w/20}
				case np >= 3:
					wide = box{Origin + w/10, Origin + 9*w/10}
				default:
					wide = box{Origin + 3*w/20, Origin + 17*w/20}
				}
				ps := g.res.Game.Planets
				for k, h := range hws {
					near := k == 0
					for _, e := range hws[:k] {
						d := d2(ps[h].Pos, ps[e].Pos)
						if d < sp.min {
							t.Errorf("size %d pos %d players %d: homeworlds %d, %d at d² %d < %d", size, pos, np, e, h, d, sp.min)
						}
						near = near || d <= sp.max
					}
					if !near {
						t.Errorf("size %d pos %d players %d: homeworld %d not within %d of an earlier one", size, pos, np, h, sp.max)
					}
					if k > 0 && !wide.contains(ps[h].Pos) {
						t.Errorf("size %d pos %d players %d: homeworld %d outside the box", size, pos, np, h)
					}
				}
			}
		}
	}
}

// Starting tech by PRT and the race options (UNIVERSE.md "Starting tech",
// CONFIRMED; the WM, SD, IT, PP, JOAT and SS rows are UG16–UG21
// observations).
func TestConfirmedStartingTech(t *testing.T) {
	want := map[engine.PRT][6]int{
		engine.PRTHyperExpansion:       {0, 0, 0, 0, 0, 0},
		engine.PRTInnerStrength:        {0, 0, 0, 0, 0, 0},
		engine.PRTSuperStealth:         {0, 0, 0, 0, 5, 0},
		engine.PRTWarMonger:            {1, 6, 1, 0, 0, 0},
		engine.PRTClaimAdjuster:        {1, 1, 1, 2, 0, 6},
		engine.PRTSpaceDemolition:      {0, 0, 2, 0, 0, 2},
		engine.PRTPacketPhysics:        {4, 0, 0, 0, 0, 0},
		engine.PRTInterstellarTraveler: {0, 0, 5, 5, 0, 0},
		engine.PRTAlternateReality:     {1, 0, 0, 0, 0, 0},
		engine.PRTJackOfAllTrades:      {3, 3, 3, 3, 3, 3},
	}
	for prt, w := range want {
		if got := StartingTech(testRace(prt), false); got != w {
			t.Errorf("PRT %d: %v, want %v", prt, got, w)
		}
	}
	r := testRace(engine.PRTWarMonger)
	r.ResearchCosts = [6]engine.ResearchCost{engine.ResearchExpensive, engine.ResearchExpensive, engine.ResearchNormal, engine.ResearchExpensive, engine.ResearchCheap, engine.ResearchNormal}
	if got := StartingTech(r, true); got != [6]int{3, 6, 1, 3, 0, 0} {
		t.Errorf("WM expensive at 3: %v", got)
	}
	if got := StartingTech(r, false); got != want[engine.PRTWarMonger] {
		t.Errorf("WM without the option: %v", got)
	}
	j := testRace(engine.PRTJackOfAllTrades)
	j.ResearchCosts[engine.Biotech] = engine.ResearchExpensive
	j.LRT.CheapEngines, j.LRT.ImprovedFuelEfficiency = true, true
	if got := StartingTech(j, true); got != [6]int{3, 3, 5, 3, 3, 4} {
		t.Errorf("JOAT expensive at 4 with CE and IFE: %v", got)
	}
}

func playerGame(t *testing.T, size Size, seed uint64, players ...PlayerSetup) Result {
	t.Helper()
	return generate(t, Settings{Size: size, Density: Normal, Players: players}, seed)
}

func designAt(res Result, d int) engine.Design { return res.Game.Designs[d] }

// The homeworld's starting state (UNIVERSE.md "Homeworld", CONFIRMED).
func TestConfirmedHomeworld(t *testing.T) {
	lsp := human(engine.PRTInnerStrength)
	lsp.Race.LRT.LowStartingPopulation = true
	immune := human(engine.PRTHyperExpansion)
	immune.Race.Env[engine.Temperature] = engine.EnvRange{Immune: true}
	immune.Race.Env[engine.Gravity] = engine.EnvRange{Center: 30, Low: 10, High: 51}
	res := playerGame(t, Small, 5, human(engine.PRTSuperStealth), lsp, immune)
	for i, st := range res.Players {
		hw := res.Game.Planets[st.Homeworld]
		if hw.Owner != i || !hw.Homeworld || res.Artifact[hw.ID] {
			t.Errorf("player %d homeworld: owner %d homeworld %v artifact %v", i, hw.Owner, hw.Homeworld, res.Artifact[hw.ID])
		}
		if hw.Mines != 10 || hw.Factories != 10 || hw.Defenses != 10 || !hw.HasScanner {
			t.Errorf("player %d: installations %d/%d/%d scanner %v", i, hw.Mines, hw.Factories, hw.Defenses, hw.HasScanner)
		}
		if !hw.HasStarbase || hw.StarbaseDesign != st.StarbaseDesigns[0] || designAt(res, hw.StarbaseDesign).Hull.Name != "Space Station" {
			t.Errorf("player %d: starbase %v design %d", i, hw.HasStarbase, hw.StarbaseDesign)
		}
		if res.Game.Players[i].ResearchBudget != 15 {
			t.Errorf("player %d: research %d%%", i, res.Game.Players[i].ResearchBudget)
		}
	}
	pop := func(i int) int { return res.Game.Planets[res.Players[i].Homeworld].Population }
	if pop(0) != 250 || pop(1) != 175 {
		t.Errorf("populations %d, %d; want 250 (25,000) and 175 (17,500 with LSP)", pop(0), pop(1))
	}
	env := res.Game.Planets[res.Players[0].Homeworld].Env
	if env != [3]int{50, 50, 50} {
		t.Errorf("SS homeworld environment %v, want the race centre", env)
	}
	env = res.Game.Planets[res.Players[2].Homeworld].Env
	if env[engine.Gravity] != 30 || env[engine.Radiation] != 50 || env[engine.Temperature] < 1 || env[engine.Temperature] > 99 {
		t.Errorf("immune homeworld environment %v, want 30, 1..99, 50", env)
	}
}

// AR homeworlds (UNIVERSE.md "Homeworld", CONFIRMED): no installations or
// scanner; starbase design 1 is a Space Station at the homeworld and
// design 0 an empty Orbital Fort.
func TestConfirmedAlternateReality(t *testing.T) {
	ar := human(engine.PRTAlternateReality)
	ar.Race.Spend = int(SpendMines)
	res := playerGame(t, Small, 2, ar, computer(engine.PRTInnerStrength, Standard))
	st := res.Players[0]
	hw := res.Game.Planets[st.Homeworld]
	if hw.Mines != 0 || hw.Factories != 0 || hw.Defenses != 0 || hw.HasScanner {
		t.Errorf("AR installations %d/%d/%d scanner %v", hw.Mines, hw.Factories, hw.Defenses, hw.HasScanner)
	}
	d0, d1 := designAt(res, st.StarbaseDesigns[0]), designAt(res, st.StarbaseDesigns[1])
	if d0.Hull.Name != "Orbital Fort" || len(d0.Slots) != 0 {
		t.Errorf("AR design 0: %s with %d slots", d0.Hull.Name, len(d0.Slots))
	}
	if d1.Hull.Name != "Space Station" || hw.StarbaseDesign != st.StarbaseDesigns[1] || hw.StarbaseHull != d1.Hull.StarbaseNumber {
		t.Errorf("AR homeworld starbase: design %d hull %d", hw.StarbaseDesign, hw.StarbaseHull)
	}
	// AR colony ship carries the Orbital Construction Module.
	cs := designAt(res, st.ShipDesigns[1])
	if cs.Hull.Name != "Colony Ship" || cs.Slots[1].Part.Name != "Orbital Construction Module" {
		t.Errorf("AR colony ship: %s %v", cs.Hull.Name, cs.Slots)
	}
}

// BBS population (UNIVERSE.md "BBS option", CONFIRMED UG03, UG21):
// × (growth%·k + 5)/5. A 10% race starts with 750 units, an IT race on a
// map larger than tiny keeps 4/5 of that (600) and its second planet
// gets 2/5 (300).
func TestConfirmedBBSPopulation(t *testing.T) {
	ss := human(engine.PRTSuperStealth)
	ss.Race.GrowthRate = 10
	it := human(engine.PRTInterstellarTraveler)
	it.Race.GrowthRate = 10
	he := human(engine.PRTHyperExpansion)
	he.Race.GrowthRate = 10
	res := generate(t, Settings{Size: Small, Density: Normal, BBS: true, Players: []PlayerSetup{ss, it, he}}, 4)
	pop := func(i int) int { return res.Game.Planets[res.Players[i].Homeworld].Population }
	if pop(0) != 750 || pop(1) != 600 || pop(2) != 250*25/5 {
		t.Errorf("BBS populations %d, %d, %d; want 750, 600, 1250", pop(0), pop(1), pop(2))
	}
	if sp := res.Game.Planets[res.Players[1].SecondPlanet].Population; sp != 300 {
		t.Errorf("IT second planet %d, want 300", sp)
	}
	if got := homeworldPopulation(player{Race: testRace(engine.PRTInnerStrength), Computer: true, Level: Expert}, false); got != 275 {
		t.Errorf("expert computer population %d, want 275", got)
	}
}

// Shared starting minerals (UNIVERSE.md, LEGACY BUG CONFIRMED UG16–UG21):
// every homeworld gets planet 0's concentrations floored at 30 and one
// shared surface draw.
func TestConfirmedSharedHomeworldMinerals(t *testing.T) {
	// UG16: planet 0 has 15/70/90; homeworlds 30/70/90.
	if got := floorConcentrations([3]int{15, 70, 90}); got != [3]int{30, 70, 90} {
		t.Errorf("floor: %v", got)
	}
	var players []PlayerSetup
	for _, prt := range allPRTs {
		players = append(players, human(prt))
		players[len(players)-1].Race.Spend = int(SpendMines)
	}
	for seed := range uint64(5) {
		res := generate(t, Settings{Size: Medium, Density: Normal, Players: players}, seed)
		ref := floorConcentrations(concentrationsOf(&res.Game.Planets[0]))
		if res.Game.Planets[0].Homeworld {
			// Planet 0's own concentrations were replaced; skip.
			continue
		}
		first := res.Game.Planets[res.Players[0].Homeworld]
		if first.Surface[0] < 165 {
			t.Errorf("seed %d: surface %v below the 10 + 155 minimum", seed, first.Surface)
		}
		for i, st := range res.Players {
			hw := res.Game.Planets[st.Homeworld]
			if c := concentrationsOf(&hw); c != ref {
				t.Errorf("seed %d player %d: concentrations %v, want planet 0's %v", seed, i, c, ref)
			}
			if hw.Surface != first.Surface {
				t.Errorf("seed %d player %d: surface %v, want the shared %v", seed, i, hw.Surface, first.Surface)
			}
		}
	}
}

// The surface draw (UNIVERSE.md "Shared starting minerals", CONFIRMED):
// 10 + rand(10·c), plus 155 + rand(150) below 200; BBS adds a quarter.
func TestConfirmedSurfaceDraw(t *testing.T) {
	// 10 + 140 = 150 < 200 → +155 + 0; 10 + 190 = 200 stays; 10 + 0 → +155 + 149.
	g := &generator{rng: &script{t: t, draws: []int{140, 0, 190, 0, 149}}}
	if got := g.surfaceDraw([3]int{15, 70, 1}); got != (engine.Minerals{305, 200, 314}) {
		t.Errorf("surface draw %v", got)
	}
	g = &generator{s: Settings{BBS: true}, rng: &script{t: t, draws: []int{390, 390, 390}}}
	if got := g.surfaceDraw([3]int{40, 40, 40}); got != (engine.Minerals{500, 500, 500}) {
		t.Errorf("BBS surface draw %v", got)
	}
}

// Leftover-point spends (UNIVERSE.md "Leftover advantage points",
// CONFIRMED): JOAT UG16 minerals 423/253/234 → 548/378/484; JOAT UG20
// concentrations 53/30/82 → 66/68/95; WM mines 35, SD factories 20, IT
// defenses 15 (UG16).
func TestConfirmedLeftoverSpends(t *testing.T) {
	if got := SurfaceSpend(engine.Minerals{423, 253, 234}, 50); got != (engine.Minerals{548, 378, 484}) {
		t.Errorf("surface spend: %v", got)
	}
	if got := ConcentrationSpend([3]int{53, 30, 82}, 50); got != [3]int{66, 68, 95} {
		t.Errorf("concentration spend: %v", got)
	}
	// Ties: the last of equal smallest; the first of equal lowest.
	// L 7: 70 kT, q = 17, remainder 2.
	if got := SurfaceSpend(engine.Minerals{100, 100, 100}, 7); got != (engine.Minerals{117, 117, 136}) {
		t.Errorf("surface spend tie, L 7: %v", got)
	}
	if got := ConcentrationSpend([3]int{40, 40, 50}, 2); got != [3]int{42, 41, 51} {
		t.Errorf("concentration spend tie, L 2: %v", got)
	}

	// Races worth at least 50 points have L = 50.
	wm := withPoints(human(engine.PRTWarMonger), 80)
	wm.Race.Spend = int(SpendMines)
	sd := withPoints(human(engine.PRTSpaceDemolition), 50)
	sd.Race.Spend = int(SpendFactories)
	it := withPoints(human(engine.PRTInterstellarTraveler), 50)
	it.Race.Spend = int(SpendDefenses)
	joat := withPoints(human(engine.PRTJackOfAllTrades), 50)
	joat.Race.Spend = int(SpendSurfaceMinerals)
	res := playerGame(t, Medium, 9, wm, sd, it, joat)
	hw := func(i int) engine.Planet { return res.Game.Planets[res.Players[i].Homeworld] }
	if m := hw(0).Mines; m != 35 {
		t.Errorf("WM mines %d, want 35", m)
	}
	if f := hw(1).Factories; f != 20 {
		t.Errorf("SD factories %d, want 20", f)
	}
	if d := hw(2).Defenses; d != 15 {
		t.Errorf("IT defenses %d, want 15", d)
	}
	if got, want := hw(3).Surface, SurfaceSpend(hw(0).Surface, 50); got != want {
		t.Errorf("JOAT surface %v, want %v", got, want)
	}
}

// Harder and expert computer players with the surface spend also get the
// concentration boost (UNIVERSE.md "Computer players", CONFIRMED).
func TestConfirmedComputerConcentrationBoost(t *testing.T) {
	std := computer(engine.PRTInnerStrength, Standard)
	std.Race.Spend = int(SpendSurfaceMinerals)
	exp := computer(engine.PRTInnerStrength, Expert)
	exp.Race.Spend = int(SpendSurfaceMinerals)
	res := playerGame(t, Small, 11, human(engine.PRTInnerStrength), std, exp)
	c := func(i int) [3]int {
		p := res.Game.Planets[res.Players[i].Homeworld]
		return concentrationsOf(&p)
	}
	if c(1) != c(0) {
		t.Errorf("standard computer: %v, want the shared %v", c(1), c(0))
	}
	if c(2) != ConcentrationSpend(c(0), 50) {
		t.Errorf("expert computer: %v, want %v", c(2), ConcentrationSpend(c(0), 50))
	}
	if res.Players[1].Name != "Computer 2" {
		t.Errorf("computer name %q", res.Players[1].Name)
	}
}

type shipWant struct {
	hull  string
	parts []string
}

func shipsOf(res Result, i int) (designs []shipWant, fleets []string) {
	for _, d := range res.Players[i].ShipDesigns {
		des := res.Game.Designs[d]
		var parts []string
		for _, s := range des.Slots {
			parts = append(parts, s.Part.Name)
		}
		designs = append(designs, shipWant{des.Hull.Name, parts})
	}
	for _, f := range res.Players[i].Fleets {
		fl := res.Game.Fleets[f]
		fleets = append(fleets, res.Game.Designs[fl.Stacks[0].Design].Hull.Name)
	}
	return designs, fleets
}

// Starting ships by PRT, with upgrades at each PRT's starting tech
// (UNIVERSE.md "Starting ships", "Part upgrades", CONFIRMED).
func TestConfirmedStartingShips(t *testing.T) {
	qj, ds := "Quick Jump 5", "Daddy Long Legs 7"
	cases := []struct {
		prt    engine.PRT
		size   Size
		fleets []string
		check  map[int]shipWant // design slot → expected hull and parts
	}{
		{engine.PRTHyperExpansion, Small, []string{"Scout", "Mini-Colony Ship", "Mini-Colony Ship", "Mini-Colony Ship"},
			map[int]shipWant{1: {"Mini-Colony Ship", []string{"Settler's Delight", "Colonization Module"}}}},
		// SS: electronics 5 → Possum Scanner; energy 0 → fuel-tank scout.
		{engine.PRTSuperStealth, Small, []string{"Scout", "Small Freighter", "Colony Ship"},
			map[int]shipWant{
				0: {"Scout", []string{qj, "Possum Scanner", "Fuel Tank"}},
				1: {"Small Freighter", []string{qj, "Transport Cloaking", "Mole-skin Shield"}},
			}},
		// WM, construction 0 (UG21): a Yakimora scout and a colony ship.
		{engine.PRTWarMonger, Small, []string{"Scout", "Colony Ship"},
			map[int]shipWant{0: {"Scout", []string{qj, "Bat Scanner", "Yakimora Light Phaser"}}}},
		{engine.PRTClaimAdjuster, Small, []string{"Scout", "Colony Ship", "Mini-Miner"},
			map[int]shipWant{2: {"Mini-Miner", []string{qj, "Bat Scanner", "Orbital Adjuster", "Orbital Adjuster"}}}},
		{engine.PRTInnerStrength, Small, []string{"Scout", "Colony Ship"}, nil},
		{engine.PRTSpaceDemolition, Small, []string{"Scout", "Colony Ship", "Mini Mine Layer", "Mini Mine Layer"},
			map[int]shipWant{
				2: {"Mini Mine Layer", []string{qj, "Mine Dispenser 40", "Mine Dispenser 40", "Bat Scanner"}},
				3: {"Mini Mine Layer", []string{qj, "Speed Trap 20", "Speed Trap 20", "Bat Scanner"}},
			}},
		{engine.PRTPacketPhysics, Small, []string{"Scout", "Colony Ship", "Scout"}, nil},
		{engine.PRTPacketPhysics, Tiny, []string{"Scout", "Colony Ship"}, nil},
		// IT, propulsion 5: Daddy Long Legs 7; construction 5:
		// Crobmnium for Tritanium.
		{engine.PRTInterstellarTraveler, Small, []string{"Scout", "Colony Ship", "Destroyer", "Privateer", "Scout"},
			map[int]shipWant{
				2: {"Destroyer", []string{ds, "Laser", "Alpha Torpedo", "Bat Scanner", "Crobmnium", "Fuel Tank", "Battle Computer"}},
				3: {"Privateer", []string{ds, "Crobmnium", "Bat Scanner", "Laser", "Alpha Torpedo"}},
			}},
		{engine.PRTAlternateReality, Small, []string{"Scout", "Colony Ship"}, nil},
		// JOAT, construction 3: Medium Freighter, Crobmnium; weapons 3:
		// X-Ray Laser stays and Laser becomes X-Ray Laser; propulsion 3:
		// Long Hump 6; electronics 3: Rhino Scanner. UG21: six designs
		// and six fleets.
		{engine.PRTJackOfAllTrades, Small, []string{"Scout", "Scout", "Colony Ship", "Medium Freighter", "Destroyer", "Mini-Miner"},
			map[int]shipWant{
				0: {"Scout", []string{"Long Hump 6", "Rhino Scanner", "X-Ray Laser"}},
				3: {"Medium Freighter", []string{"Long Hump 6", "Rhino Scanner", "Crobmnium"}},
				4: {"Destroyer", []string{"Long Hump 6", "X-Ray Laser", "Alpha Torpedo", "Rhino Scanner", "Crobmnium", "Fuel Tank", "Battle Computer"}},
				5: {"Mini-Miner", []string{"Long Hump 6", "Rhino Scanner", "Robo-Mini-Miner", "Robo-Mini-Miner"}},
			}},
	}
	for _, c := range cases {
		res := playerGame(t, c.size, 1, human(c.prt), computer(engine.PRTInnerStrength, Easy))
		designs, fleets := shipsOf(res, 0)
		if !reflect.DeepEqual(fleets, c.fleets) {
			t.Errorf("PRT %d size %d: fleets %v, want %v", c.prt, c.size, fleets, c.fleets)
		}
		for slot, w := range c.check {
			if slot >= len(designs) || !reflect.DeepEqual(designs[slot], w) {
				t.Errorf("PRT %d design %d: %+v, want %+v", c.prt, slot, designs, w)
			}
		}
		for _, f := range res.Players[0].Fleets {
			fl := res.Game.Fleets[f]
			d := res.Game.Designs[fl.Stacks[0].Design]
			if fl.Owner != 0 || fl.Stacks[0].Count != 1 || fl.Fuel != d.FuelCapacity || fl.Plan != 0 {
				t.Errorf("PRT %d fleet %d: %+v", c.prt, f, fl)
			}
		}
	}
	// JOAT six designs.
	res := playerGame(t, Small, 1, human(engine.PRTJackOfAllTrades), computer(engine.PRTInnerStrength, Easy))
	if n := len(res.Players[0].ShipDesigns); n != 6 {
		t.Errorf("JOAT: %d ship designs, want 6", n)
	}
	// HE: three Mini-Colony Ships share one design.
	res = playerGame(t, Small, 1, human(engine.PRTHyperExpansion), computer(engine.PRTInnerStrength, Easy))
	if n := len(res.Players[0].ShipDesigns); n != 2 {
		t.Errorf("HE: %d ship designs, want 2", n)
	}
	// SS computer players have no Small Freighter.
	res = playerGame(t, Small, 1, human(engine.PRTInnerStrength), computer(engine.PRTSuperStealth, Easy))
	if _, fleets := shipsOf(res, 1); !reflect.DeepEqual(fleets, []string{"Scout", "Colony Ship"}) {
		t.Errorf("SS computer fleets %v", fleets)
	}
}

// Construction- and energy-dependent ships (UNIVERSE.md "Starting
// ships", CONFIRMED rule): WM with construction 3 adds a Destroyer and a
// Mini Bomber; SS with energy 2 gets the cloaked scout; JOAT with
// construction 4 a Privateer; ARM without OBRM adds two Midget Miners.
func TestConfirmedConditionalShips(t *testing.T) {
	wm := human(engine.PRTWarMonger)
	wm.Race.ResearchCosts[engine.Construction] = engine.ResearchExpensive
	wm.Race.ExpensiveAt3 = true
	ss := human(engine.PRTSuperStealth)
	ss.Race.ResearchCosts[engine.Energy] = engine.ResearchExpensive
	ss.Race.ExpensiveAt3 = true
	joat := human(engine.PRTJackOfAllTrades)
	joat.Race.ResearchCosts[engine.Construction] = engine.ResearchExpensive
	joat.Race.ExpensiveAt3 = true
	arm := human(engine.PRTInnerStrength)
	arm.Race.LRT.AdvancedRemoteMining = true
	res := playerGame(t, Small, 2, wm, ss, joat, arm)
	want := [][]string{
		{"Scout", "Destroyer", "Mini Bomber", "Colony Ship"},
		{"Scout", "Small Freighter", "Colony Ship"},
		{"Scout", "Scout", "Colony Ship", "Privateer", "Destroyer", "Mini-Miner"},
		{"Scout", "Colony Ship", "Midget Miner", "Midget Miner"},
	}
	for i, w := range want {
		designs, fleets := shipsOf(res, i)
		if !reflect.DeepEqual(fleets, w) {
			t.Errorf("player %d: fleets %v, want %v", i, fleets, w)
		}
		if i == 1 && designs[0].parts[2] != "Stealth Cloak" {
			t.Errorf("SS with energy 3: scout %v", designs[0])
		}
		// Weapons 6: Black Cat Bomb for the Lady Finger Bomb.
		if i == 0 && !reflect.DeepEqual(designs[2].parts, []string{"Quick Jump 5", "Black Cat Bomb"}) {
			t.Errorf("WM Mini Bomber %v", designs[2])
		}
	}
	if d := res.Players[3].ShipDesigns; len(d) != 3 {
		t.Errorf("ARM: %d designs, want 3", len(d))
	}
}

// Part upgrades: the first available replacement, the Colony Ship ram
// scoop exception, and race restrictions (UNIVERSE.md "Part upgrades",
// CONFIRMED for the cases run).
func TestConfirmedPartUpgrades(t *testing.T) {
	g := &generator{cat: engine.Components()}
	race := testRace(engine.PRTInnerStrength)
	var tech [6]int
	tech[engine.Energy], tech[engine.Propulsion] = 2, 6
	up := func(part, hull string, r engine.Race, lv [6]int) string {
		t.Helper()
		p, err := g.upgrade(part, hull, r, lv)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	if p := up("Quick Jump 5", "Scout", race, tech); p != "Radiating Hydro-Ram Scoop" {
		t.Errorf("scout engine %s", p)
	}
	// Radiation centre 50: the Colony Ship skips the ram scoop.
	if p := up("Quick Jump 5", "Colony Ship", race, tech); p != "Daddy Long Legs 7" {
		t.Errorf("colony ship engine %s", p)
	}
	hot := race
	hot.Env[engine.Radiation] = engine.EnvRange{Center: 85, Low: 70, High: 100}
	if p := up("Quick Jump 5", "Colony Ship", hot, tech); p != "Radiating Hydro-Ram Scoop" {
		t.Errorf("radiation 85 colony ship engine %s", p)
	}
	nrse := race
	nrse.LRT.NoRamScoopEngines = true
	if p := up("Quick Jump 5", "Scout", nrse, tech); p != "Daddy Long Legs 7" {
		t.Errorf("NRSE scout engine %s", p)
	}
	ife := race
	ife.LRT.ImprovedFuelEfficiency = true
	if p := up("Quick Jump 5", "Scout", ife, [6]int{0, 0, 2, 0, 0, 0}); p != "Fuel Mizer" {
		t.Errorf("IFE propulsion 2 engine %s", p)
	}
	if p := up("Bat Scanner", "Scout", race, [6]int{0, 0, 0, 0, 4, 0}); p != "Mole Scanner" {
		t.Errorf("electronics 4 scanner %s", p)
	}
	if p := up("Bat Scanner", "Scout", race, [6]int{}); p != "Bat Scanner" {
		t.Errorf("tech 0 scanner %s", p)
	}
	if p := up("Tritanium", "Destroyer", race, [6]int{0, 0, 0, 0, 0, 4}); p != "Carbonic Armor" {
		t.Errorf("biotech 4 armor %s", p)
	}
	if p := up("Mole-skin Shield", "Small Freighter", race, [6]int{6, 0, 0, 0, 0, 0}); p != "Wolverine Diffuse Shield" {
		t.Errorf("energy 6 shield %s", p)
	}
	if p := up("Alpha Torpedo", "Destroyer", race, [6]int{0, 5, 1, 0, 0, 0}); p != "Beta Torpedo" {
		t.Errorf("torpedo %s", p)
	}
	if p := up("Lady Finger Bomb", "Mini Bomber", race, [6]int{0, 5, 0, 0, 0, 0}); p != "Black Cat Bomb" {
		t.Errorf("bomb %s", p)
	}
	if p := up("Robo-Mini-Miner", "Mini-Miner", race, [6]int{0, 0, 0, 4, 2, 0}); p != "Robo-Miner" {
		t.Errorf("robot %s", p)
	}
}

// Second planet for PP and IT on a map larger than tiny (UNIVERSE.md
// "Second planet: PP and IT", CONFIRMED UG16–UG21; tiny UG19).
func TestConfirmedSecondPlanet(t *testing.T) {
	for _, prt := range []engine.PRT{engine.PRTPacketPhysics, engine.PRTInterstellarTraveler} {
		for seed := range uint64(5) {
			s := Settings{Size: Medium, Density: Normal, Players: []PlayerSetup{human(prt), human(engine.PRTInnerStrength)}}
			res := generate(t, s, seed)
			st := res.Players[0]
			if st.SecondPlanet < 0 {
				t.Fatalf("PRT %d seed %d: no second planet", prt, seed)
			}
			hw, sp := res.Game.Planets[st.Homeworld], res.Game.Planets[st.SecondPlanet]
			if !secondBand(d2(hw.Pos, sp.Pos), s.Width()) {
				// Allowed only when no unowned planet lay in the band.
				t.Logf("PRT %d seed %d: second planet outside the band", prt, seed)
			}
			if sp.Owner != 0 || sp.Homeworld || sp.Mines != 10 || sp.Factories != 4 || !sp.HasScanner {
				t.Errorf("PRT %d seed %d: second planet %+v", prt, seed, sp)
			}
			if hw.Population != 200 || sp.Population != 100 {
				t.Errorf("PRT %d seed %d: populations %d / %d, want 200 / 100", prt, seed, hw.Population, sp.Population)
			}
			for m, v := range sp.Surface {
				if v < 100 || v > 299 {
					t.Errorf("PRT %d seed %d: mineral %d surface %d", prt, seed, m, v)
				}
			}
			if hab := engine.Habitability(testRace(prt), sp.Env); hab < 10 && sp.Env != hw.Env {
				t.Errorf("PRT %d seed %d: second planet habitability %d", prt, seed, hab)
			}
			d1 := res.Game.Designs[st.StarbaseDesigns[1]]
			want := map[engine.PRT]string{engine.PRTPacketPhysics: "Mass Driver 5", engine.PRTInterstellarTraveler: "Stargate 100/250"}[prt]
			if !sp.HasStarbase || sp.StarbaseDesign != st.StarbaseDesigns[1] || d1.Hull.Name != "Orbital Fort" || d1.Slots[0].Part.Name != want {
				t.Errorf("PRT %d seed %d: second planet starbase %+v", prt, seed, d1)
			}
			d0 := res.Game.Designs[st.StarbaseDesigns[0]]
			if d0.Slots[0].Part.Name != want {
				t.Errorf("PRT %d: design 0 lacks %s", prt, want)
			}
			last := res.Game.Fleets[st.Fleets[len(st.Fleets)-1]]
			if last.Pos != sp.Pos || last.Stacks[0].Design != st.ShipDesigns[0] {
				t.Errorf("PRT %d seed %d: no design-0 scout at the second planet", prt, seed)
			}
		}
		res := playerGame(t, Tiny, 1, human(prt), human(engine.PRTInnerStrength))
		if st := res.Players[0]; st.SecondPlanet != -1 || len(st.StarbaseDesigns) != 1 || res.Game.Planets[st.Homeworld].Population != 250 {
			t.Errorf("PRT %d tiny: %+v", prt, st)
		}
	}
}

// The redraw rule and its LEGACY BUG fallback (UNIVERSE.md "Second
// planet", CONFIRMED UG29, UG30).
func TestConfirmedSecondPlanetRedraw(t *testing.T) {
	r := testRace(engine.PRTPacketPhysics)
	// A narrow race: 50 ± 1 on every axis makes most redraws fail.
	for i := range r.Env {
		r.Env[i] = engine.EnvRange{Center: 50, Low: 49, High: 51}
	}
	ps := human(engine.PRTPacketPhysics)
	ps.Race.Race = r
	res := generate(t, Settings{Size: Medium, Density: Normal, Players: []PlayerSetup{ps}}, 3)
	st := res.Players[0]
	sp, hw := res.Game.Planets[st.SecondPlanet], res.Game.Planets[st.Homeworld]
	if sp.Env != hw.Env {
		t.Errorf("legacy: second planet %v, want the homeworld's %v", sp.Env, hw.Env)
	}
	fixed := rulesWith(func(l *engine.Legacy) { l.SecondPlanetFallback = false })
	res = generate(t, Settings{Rules: fixed, Size: Medium, Density: Normal, Players: []PlayerSetup{ps}}, 3)
	sp = res.Game.Planets[res.Players[0].SecondPlanet]
	if sp.Env == hw.Env {
		t.Errorf("fixed: second planet took the homeworld environment")
	}
}

// Relations (UNIVERSE.md "Relations", MEASURED): with exactly one human
// every player is an enemy of every other.
func TestConfirmedRelations(t *testing.T) {
	res := playerGame(t, Small, 1, human(engine.PRTSuperStealth), computer(engine.PRTInnerStrength, Easy), computer(engine.PRTClaimAdjuster, Easy))
	for i, p := range res.Game.Players {
		for j, r := range p.Relations {
			want := engine.RelationEnemy
			if i == j {
				want = engine.RelationFriend
			}
			if r != want {
				t.Errorf("player %d → %d: %d", i, j, r)
			}
		}
		if len(p.Relations) != 3 {
			t.Errorf("player %d: %d relations", i, len(p.Relations))
		}
	}
	res = playerGame(t, Small, 1, human(engine.PRTSuperStealth), human(engine.PRTInnerStrength))
	for i, p := range res.Game.Players {
		if p.Relations != nil {
			t.Errorf("two humans: player %d relations %v", i, p.Relations)
		}
	}
}

// Wormholes at creation (OBJECTS.md "Creation", CONFIRMED OB-006,
// UG01–UG21): rand(v) + m pairs by size, stability class 0..2, 0 years.
func TestConfirmedWormholes(t *testing.T) {
	ranges := [5][2]int{{0, 2}, {1, 3}, {1, 5}, {3, 6}, {4, 8}}
	for size := Tiny; size <= Huge; size++ {
		seen := map[int]bool{}
		for seed := range uint64(30) {
			s := Settings{Size: size, Density: Sparse, Players: twoPlayers()}
			res := generate(t, s, seed)
			n := len(res.Wormholes)
			seen[n] = true
			if n < ranges[size][0] || n > ranges[size][1] {
				t.Errorf("size %d seed %d: %d wormholes", size, seed, n)
			}
			for _, wh := range res.Wormholes {
				for _, e := range wh.Ends {
					if e.Class < 0 || e.Class > 2 || e.Years != 0 {
						t.Errorf("size %d: end %+v", size, e)
					}
					if e.Pos.X < Origin || e.Pos.X > Origin+s.Width() || e.Pos.Y < Origin || e.Pos.Y > Origin+s.Width() {
						t.Errorf("size %d: end at %v outside the galaxy", size, e.Pos)
					}
					for _, p := range res.Game.Planets {
						if p.Pos == e.Pos {
							t.Errorf("size %d: end on planet %d", size, p.ID)
						}
					}
				}
			}
		}
		if len(seen) != ranges[size][1]-ranges[size][0]+1 {
			t.Errorf("size %d: counts seen %v, want every count in %v", size, seen, ranges[size])
		}
	}
}

// The badness rule keeps ends apart when it can (OBJECTS.md "Placement
// badness", BINARY-ONLY: "in effect ends settle ≥ 70 ly from their
// partner").
func TestPredictionWormholeSpacing(t *testing.T) {
	for seed := range uint64(10) {
		res := generate(t, Settings{Size: Huge, Density: Sparse, Players: twoPlayers()}, seed)
		for _, wh := range res.Wormholes {
			if d := d2(wh.Ends[0].Pos, wh.Ends[1].Pos); d < 4900 {
				t.Errorf("seed %d: partner ends at d² %d", seed, d)
			}
		}
	}
}

// Elegy's generator is deterministic: the same settings and seed give the
// same game; another seed gives another game (UNIVERSE.md "Randomness
// and seeds": Elegy's equivalent of the definition-file seed).
func TestElegyDecisionDeterministic(t *testing.T) {
	s := Settings{Size: Medium, Density: Dense, Positions: Farther, Clumping: true, Players: twoPlayers()}
	a, b := generate(t, s, 42), generate(t, s, 42)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed, different games")
	}
	c := generate(t, s, 43)
	if reflect.DeepEqual(a.Game.Planets, c.Game.Planets) {
		t.Fatal("different seeds, same planets")
	}
	if _, err := Generate(s, nil); !errors.Is(err, ErrNilRand) {
		t.Fatalf("nil rand: %v", err)
	}
	elegy := engine.ElegyRules()
	mislabelled := elegy
	mislabelled.Legacy.MergeOverflow = true
	for _, bad := range []Settings{
		{Rules: elegy, Size: 5, Players: twoPlayers()},
		{Rules: elegy, Density: -1, Players: twoPlayers()},
		{Rules: elegy},
		{Rules: elegy, Players: []PlayerSetup{{Race: races.Design{Race: testRace(engine.PRTOther)}, Computer: true}}},
		{Players: twoPlayers()},
		{Rules: mislabelled, Players: twoPlayers()},
	} {
		if _, err := Generate(bad, NewRand(1)); !errors.Is(err, ErrSettings) {
			t.Errorf("settings %+v: %v", bad, err)
		}
	}
}

// Elegy's generator draws uniformly (the Intn bound is unbiased).
func TestElegyDecisionRandUniform(t *testing.T) {
	r := NewRand(1)
	var counts [7]int
	const n = 70000
	for range n {
		counts[r.Intn(7)]++
	}
	for v, c := range counts {
		if c < 9500 || c > 10500 {
			t.Errorf("value %d drawn %d times of %d", v, c, n)
		}
	}
}

// With the shared-minerals LEGACY BUG switched off, each homeworld has
// its own concentrations, floored at 30.
func TestElegyDecisionSharedMineralsSwitch(t *testing.T) {
	var players []PlayerSetup
	for range 8 {
		p := human(engine.PRTInnerStrength)
		p.Race.Spend = int(SpendMines)
		players = append(players, p)
	}
	fixed := rulesWith(func(l *engine.Legacy) { l.SharedHomeworldMinerals = false })
	res := generate(t, Settings{Rules: fixed, Size: Large, Density: Normal, Players: players}, 2)
	distinct := map[[3]int]bool{}
	for _, st := range res.Players {
		hw := res.Game.Planets[st.Homeworld]
		c := concentrationsOf(&hw)
		for _, v := range c {
			if v < 30 {
				t.Errorf("homeworld %d concentration %v below 30", hw.ID, c)
			}
		}
		distinct[c] = true
	}
	if len(distinct) < 2 {
		t.Errorf("fixed rule: every homeworld has the same concentrations")
	}
}

// A generated game runs through the engine's turn: every PRT, a few
// years, no error.
func TestElegyDecisionGameRuns(t *testing.T) {
	var players []PlayerSetup
	for _, prt := range allPRTs {
		players = append(players, human(prt))
	}
	res := generate(t, Settings{Size: Medium, Density: Normal, Players: players}, 8)
	g := res.Game
	// The game carries its wormholes into the turn as its space objects.
	if sp, ok := g.Objects.(*objects.Space); !ok || len(sp.Wormholes) != len(res.Wormholes) || len(res.Wormholes) == 0 {
		t.Fatalf("objects %#v, %d wormholes", g.Objects, len(res.Wormholes))
	}
	rng := NewRand(99)
	for range 5 {
		out, err := engine.GenerateTurn(g, nil, rng)
		if err != nil {
			t.Fatalf("year %d: %v", g.Year, err)
		}
		g = out.Game
	}
	if g.Year != 2405 {
		t.Fatalf("year %d", g.Year)
	}
	if sp, ok := g.Objects.(*objects.Space); !ok || len(sp.Wormholes) != len(res.Wormholes) {
		t.Errorf("objects after five years: %#v", g.Objects)
	}
}

func loadout(d engine.Design) map[string]int {
	out := map[string]int{}
	for _, s := range d.Slots {
		out[s.Part.Name] += s.Count
	}
	return out
}

// Starbase loadouts (UNIVERSE.md "Starbases", CONFIRMED in every UG game):
// the Space Station has 32 Lasers and 32 Mole-skin Shields in eight slots;
// the PP fort uses Cow-hide, the IT fort Mole-skin; AR's design 1 is the
// plain station and design 0 an empty fort.
func TestConfirmedStarbaseLoadouts(t *testing.T) {
	res := playerGame(t, Small, 1, human(engine.PRTInnerStrength), human(engine.PRTPacketPhysics),
		human(engine.PRTInterstellarTraveler), human(engine.PRTAlternateReality))
	sb := func(i, k int) engine.Design { return designAt(res, res.Players[i].StarbaseDesigns[k]) }
	station := map[string]int{"Laser": 32, "Mole-skin Shield": 32}
	if got := loadout(sb(0, 0)); !reflect.DeepEqual(got, station) || len(sb(0, 0).Slots) != 8 {
		t.Errorf("Space Station: %v in %d slots", got, len(sb(0, 0).Slots))
	}
	if got := loadout(sb(1, 0)); got["Mass Driver 5"] != 1 || got["Laser"] != 32 {
		t.Errorf("PP design 0: %v", got)
	}
	if got := loadout(sb(1, 1)); !reflect.DeepEqual(got, map[string]int{"Mass Driver 5": 1, "Laser": 12, "Cow-hide Shield": 12}) {
		t.Errorf("PP fort: %v", got)
	}
	if got := loadout(sb(2, 1)); !reflect.DeepEqual(got, map[string]int{"Stargate 100/250": 1, "Laser": 12, "Mole-skin Shield": 12}) {
		t.Errorf("IT fort: %v", got)
	}
	if got := loadout(sb(3, 1)); !reflect.DeepEqual(got, station) {
		t.Errorf("AR design 1: %v", got)
	}
}

// Research, relations and production at the start (UNIVERSE.md
// "Relations, research and production", MEASURED): 15% research on
// energy with next field "same" for every player, and no production
// queue anywhere.
func TestConfirmedResearchAndQueues(t *testing.T) {
	res := playerGame(t, Small, 1, human(engine.PRTPacketPhysics), computer(engine.PRTClaimAdjuster, Expert))
	for i, p := range res.Game.Players {
		if p.ResearchBudget != 15 || p.Research.Current != engine.Energy || p.Research.Next != engine.NextSameField {
			t.Errorf("player %d research %d%% %+v", i, p.ResearchBudget, p.Research)
		}
	}
	for _, p := range res.Game.Planets {
		if p.HasQueue {
			t.Errorf("planet %d has a production queue", p.ID)
		}
	}
}

// The second-planet band truncates each bound before squaring and
// includes both ends (UNIVERSE.md "Second planet").
func TestConfirmedSecondPlanetBand(t *testing.T) {
	// W 1210: 15W/100 = 181 (181.5), 23W/100 = 278 (278.3).
	cases := []struct {
		d2   int
		want bool
	}{{181*181 - 1, false}, {181 * 181, true}, {278 * 278, true}, {278*278 + 1, false}}
	for _, c := range cases {
		if got := secondBand(c.d2, 1210); got != c.want {
			t.Errorf("d² %d: %v", c.d2, got)
		}
	}
}

// Wormhole badness flags (OBJECTS.md "Placement badness", CONFIRMED at
// creation): bands set 8, 4, 2, 1; the edge sets 4; flags are OR-ed.
func TestConfirmedWormholeBadness(t *testing.T) {
	g := &generator{w: 400}
	g.res.Game.Planets = []engine.Planet{{Pos: engine.Point{X: 1200, Y: 1200}}, {Pos: engine.Point{X: 1210, Y: 1200}}}
	cases := []struct {
		p       engine.Point
		partner *engine.Point
		want    int
	}{
		{engine.Point{X: 1300, Y: 1300}, nil, 0},
		{engine.Point{X: 1200, Y: 1200}, nil, 15},                                // on a planet
		{engine.Point{X: 1204, Y: 1200}, nil, 8 | 4},                             // d² 16 (closest band) and 36
		{engine.Point{X: 1200, Y: 1215}, nil, 2},                                 // d² 225, 325
		{engine.Point{X: 1005, Y: 1300}, nil, 4},                                 // edge
		{engine.Point{X: 1300, Y: 1300}, &engine.Point{X: 1330, Y: 1300}, 1},     // partner d² 900: the 4900 band
		{engine.Point{X: 1300, Y: 1300}, &engine.Point{X: 1329, Y: 1300}, 2},     // partner d² 841
		{engine.Point{X: 1300, Y: 1300}, &engine.Point{X: 1300, Y: 1309}, 4},     // partner d² 81
		{engine.Point{X: 1395, Y: 1300}, &engine.Point{X: 1395, Y: 1302}, 4 | 8}, // edge and partner d² 4
	}
	for _, c := range cases {
		if got := g.wormholeBadness(c.p, c.partner); got != c.want {
			t.Errorf("%v partner %v: %d, want %d", c.p, c.partner, got, c.want)
		}
	}
}

// Races at creation (RACES.md "At game creation", CONFIRMED RD-4): an
// illegal human race becomes the default race with a computer name and
// L 25; a Random race is generated with 0..50 points and keeps a name
// other than "Random"; computer races are not checked.
func TestConfirmedRacesAtCreation(t *testing.T) {
	illegal := human(engine.PRTInterstellarTraveler) // −82 points as built
	illegal.Race.Name = "Cheaters"
	illegal.Race.Spend = int(SpendMines)
	if p := races.Points(illegal.Race); p >= 0 {
		t.Fatalf("test race is legal (%d points)", p)
	}
	random := PlayerSetup{Race: races.RandomTemplate("Random")}
	zorgon := PlayerSetup{Race: races.RandomTemplate("Zorgon")}
	cpu := computer(engine.PRTInterstellarTraveler, Easy)
	s := Settings{Rules: engine.ElegyRules(), Size: Small, Density: Normal, Players: []PlayerSetup{illegal, random, zorgon, cpu}}
	res, err := Generate(s, NewRand(5))
	if err != nil {
		t.Fatal(err)
	}
	st := res.Players[0]
	def := races.Default()
	def.Tampered = true
	if st.Race != def || st.Name != "Computer 1" || st.Points != 25 {
		t.Errorf("illegal race: %+v name %q points %d", st.Race, st.Name, st.Points)
	}
	if p := res.Game.Players[0].Race.PRT; p != engine.PRTJackOfAllTrades {
		t.Errorf("illegal race plays as PRT %d", p)
	}
	// L 25 with the default race's spend (surface minerals): no mines spend.
	if hw := res.Game.Planets[st.Homeworld]; hw.Mines != 10 {
		t.Errorf("illegal race homeworld mines %d", hw.Mines)
	}
	for i, name := range []string{"Computer 2", "Zorgon"} {
		st := res.Players[i+1]
		if st.Name != name || st.Race.Random || st.Points < 0 || st.Points > 50 {
			t.Errorf("random race %d: name %q points %d", i, st.Name, st.Points)
		}
	}
	if st := res.Players[3]; st.Race.Race.PRT != engine.PRTInterstellarTraveler || st.Race.Tampered {
		t.Errorf("computer race changed: %+v", st.Race)
	}
}

// A game with one computer player of each built-in type (AI.md "Built-in
// races") is created, and each keeps its built-in race unchecked
// (RACES.md "At game creation": computer races are not checked).
func TestConfirmedBuiltInComputerPlayers(t *testing.T) {
	ps := []PlayerSetup{withPoints(human(engine.PRTJackOfAllTrades), 0)}
	for typ := 1; typ <= 6; typ++ {
		c, err := ComputerPlayer(typ, Level(typ%4))
		if err != nil {
			t.Fatal(err)
		}
		ps = append(ps, c)
	}
	res := generate(t, Settings{Size: Medium, Density: Normal, Players: ps}, 11)
	for i := 1; i < len(ps); i++ {
		if got := res.Players[i].Race; got.Race != ps[i].Race.Race || got.Tampered {
			t.Errorf("computer %d: race %+v, want %+v", i, got.Race, ps[i].Race.Race)
		}
	}
	if _, err := ComputerPlayer(7, Easy); err == nil {
		t.Error("type 7 accepted")
	}
}

// Starting design slots (UNIVERSE.md "Starbases" and "Starting ships",
// CONFIRMED UG01..UG21): the starbase designs are starbase slots 0 and 1
// and each new ship design takes the next ship slot, recorded in
// Game.DesignSlots in PlayerStart order.
func TestConfirmedStartingDesignSlots(t *testing.T) {
	var players []PlayerSetup
	for _, prt := range allPRTs {
		players = append(players, human(prt))
	}
	res := generate(t, Settings{Size: Medium, Density: Normal, Players: players}, 8)
	g := res.Game
	for i, st := range res.Players {
		for k, d := range st.StarbaseDesigns {
			if got, ok := g.PlayerDesign(i, true, k); !ok || got != d {
				t.Errorf("player %d starbase slot %d = %d, %v; want design %d", i, k, got, ok, d)
			}
		}
		for k, d := range st.ShipDesigns {
			if got, ok := g.PlayerDesign(i, false, k); !ok || got != d {
				t.Errorf("player %d ship slot %d = %d, %v; want design %d", i, k, got, ok, d)
			}
		}
	}
	if want := func() (n int) {
		for _, st := range res.Players {
			n += len(st.StarbaseDesigns) + len(st.ShipDesigns)
		}
		return
	}(); len(g.DesignSlots) != want {
		t.Errorf("%d design slots, want %d", len(g.DesignSlots), want)
	}
}

// rulesWith is a custom ruleset: the Elegy ruleset changed by change.
func rulesWith(change func(*engine.Legacy)) engine.Ruleset {
	r := engine.ElegyRules()
	r.ID = "elegy-test-variant"
	change(&r.Legacy)
	return r
}

// A new game carries its settings' ruleset.
func TestGenerateCarriesRules(t *testing.T) {
	res := generate(t, Settings{Rules: engine.FaithfulRules(), Size: Tiny, Density: Normal, Players: twoPlayers()}, 1)
	if res.Game.Rules != engine.FaithfulRules() {
		t.Errorf("game rules %+v", res.Game.Rules)
	}
}

// Games generated at the same time under different rulesets each follow
// their own: with SharedHomeworldMinerals (Elegy) every homeworld has
// planet 0's concentrations; without it each has its own.
func TestRulesetsCoexistAtCreation(t *testing.T) {
	own := rulesWith(func(l *engine.Legacy) { l.SharedHomeworldMinerals = false })
	rulesets := []engine.Ruleset{engine.ElegyRules(), own}
	settings := func(r engine.Ruleset) Settings {
		var players []PlayerSetup
		for range 4 {
			p := withPoints(human(engine.PRTInnerStrength), 0)
			p.Race.Spend = int(SpendMines)
			players = append(players, p)
		}
		return Settings{Rules: r, Size: Medium, Density: Normal, Players: players}
	}
	const runs = 20
	got := make([]Result, runs*len(rulesets))
	errs := make([]error, len(got))
	var wg sync.WaitGroup
	for k := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[k], errs[k] = Generate(settings(rulesets[k%len(rulesets)]), NewRand(7))
		}()
	}
	wg.Wait()
	for k, res := range got {
		r := rulesets[k%len(rulesets)]
		if errs[k] != nil {
			t.Fatalf("%s: %v", r.ID, errs[k])
		}
		if res.Game.Rules != r {
			t.Errorf("run %d: rules %s, want %s", k, res.Game.Rules.ID, r.ID)
		}
		distinct := map[[engine.NumMinerals]int]bool{}
		for _, st := range res.Players {
			hw := res.Game.Planets[st.Homeworld]
			distinct[concentrationsOf(&hw)] = true
		}
		if shared := len(distinct) == 1; shared != r.Legacy.SharedHomeworldMinerals {
			t.Errorf("run %d (%s): %d distinct homeworld concentrations", k, r.ID, len(distinct))
		}
		if !reflect.DeepEqual(res, got[k%len(rulesets)]) {
			t.Errorf("run %d (%s) differs from run %d", k, r.ID, k%len(rulesets))
		}
	}
}
