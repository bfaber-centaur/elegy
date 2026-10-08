package engine

import (
	"reflect"
	"testing"
)

func TestGenerateTurnAdvancesYear(t *testing.T) {
	game := Game{
		Year:  2400,
		Rules: ElegyRules(),
	}

	result, err := GenerateTurn(game, nil, highRand{})
	if err != nil {
		t.Fatalf("GenerateTurn() error = %v", err)
	}

	if got, want := result.Game.Year, 2401; got != want {
		t.Errorf("year = %d, want %d", got, want)
	}
}

func TestGenerateTurnRejectsNilRand(t *testing.T) {
	if _, err := GenerateTurn(withRules(pgHomeworld()), nil, nil); err != ErrNilRand {
		t.Errorf("err = %v, want ErrNilRand", err)
	}
}

// pgHomeworld is the PG-001..003 homeworld as observed in 2407: one player,
// no production queue, 10 mines and 10 factories, energy research.
func pgHomeworld() Game {
	start := pgDeposits[0]
	p := Planet{
		ID: 7, Owner: 0, Homeworld: true, Env: [3]int{50, 50, 50},
		Population: 486, GrowthCarry: 80,
		Mines: 10, Factories: 10, Defenses: 10,
		Surface: Minerals(start.surface),
	}
	for m := range NumMinerals {
		p.Deposits[m] = Deposit{Concentration: start.conc[m], Fraction: start.frac[m]}
	}
	pl := Player{Race: pgRace(), Research: ResearchState{Current: Energy, Next: NextSameField}}
	pl.Research.Levels[Energy] = 2
	pl.Research.Levels[Electronics] = 5
	pl.Research.Accumulated[Energy] = 65
	return Game{Year: 2407, Players: []Player{pl}, Planets: []Planet{p}}
}

// Ground truth end to end: population and carry every year 2408–2436,
// energy research at every PG003 checkpoint, and mineral deposits
// 2408–2411, from whole GenerateTurn calls.
func TestConfirmedPGHomeworldTurns(t *testing.T) {
	research := map[int][2]int{
		2408: {2, 123}, 2409: {2, 186}, 2410: {3, 54}, 2413: {4, 7},
		2417: {5, 15}, 2421: {5, 638}, 2422: {6, 182}, 2426: {7, 163},
		2431: {8, 389}, 2435: {8, 2274}, 2436: {9, 343},
	}
	g := pgHomeworld()
	for g.Year < 2436 {
		r, err := GenerateTurn(withRules(g), nil, highRand{})
		if err != nil {
			t.Fatal(err)
		}
		g = r.Game
		p := g.Planets[0]
		if want := pgPopulation[g.Year-2400]; p.Population != want[0] || p.GrowthCarry != want[1] {
			t.Errorf("%d: population (%d,%d), want (%d,%d)", g.Year, p.Population, p.GrowthCarry, want[0], want[1])
		}
		rs := g.Players[0].Research
		if w, ok := research[g.Year]; ok && (rs.Levels[Energy] != w[0] || rs.Accumulated[Energy] != w[1]) {
			t.Errorf("%d: energy %d/%d, want %d/%d", g.Year, rs.Levels[Energy], rs.Accumulated[Energy], w[0], w[1])
		}
		for _, d := range pgDeposits {
			if d.year != g.Year {
				continue
			}
			for m := range NumMinerals {
				if got := p.Deposits[m]; got.Concentration != d.conc[m] || got.Fraction != d.frac[m] {
					t.Errorf("%d mineral %d: %+v, want conc %d frac %d", g.Year, m, got, d.conc[m], d.frac[m])
				}
			}
		}
	}
}

func TestGenerateTurnDoesNotModifyInput(t *testing.T) {
	g := pqGame(1050, []QueueItem{q(ItemFactory, 20, 0)})
	if _, err := GenerateTurn(withRules(g), nil, highRand{}); err != nil {
		t.Fatal(err)
	}
	if g.Planets[0].Factories != 0 || g.Planets[0].Queue[0].Count != 20 || g.Planets[0].Population != 1050 {
		t.Errorf("input game changed: %+v", g.Planets[0])
	}
}

// recordRand records the bound of every draw and returns 0.
type recordRand struct{ bounds []int }

func (r *recordRand) Intn(n int) int {
	r.bounds = append(r.bounds, n)
	return 0
}

// TestPredictionTurnAppliesOrders: GenerateTurn applies the year's order
// files first (KERNEL.md "Turn order" step 1): the player shuffle makes
// the year's first draws, one per player, then each file applies, and a
// colonist drop given by an order joins the before-movement drop
// resolution (TAKEOVER.md "Manual cargo transfers to other players").
func TestPredictionTurnAppliesOrders(t *testing.T) {
	g := ordersGame()
	for i := range g.Players {
		g.Players[i].Race = pgRace()
	}
	g.Fleets[0].Pos = g.Planets[1].Pos
	files := []PlayerOrders{
		{Player: 0, GameID: g.ID, Year: g.Year, Orders: []Order{
			ResearchOrder{Budget: 30, Field: Weapons, Next: NextSameField},
			CargoOrder{Fleet: 1, Target: TargetPlanet, ID: 2, Amounts: [NumCargo + 1]int{0, 0, 0, -10}},
		}},
		{Player: 1, GameID: g.ID, Year: g.Year - 1},
	}
	rng := &recordRand{}
	r, err := GenerateTurn(withRules(*g), files, rng)
	if err != nil {
		t.Fatal(err)
	}
	if len(rng.bounds) < 2 || rng.bounds[0] != 2 || rng.bounds[1] != 1 {
		t.Errorf("first draws %v, want the shuffle's Random(2), Random(1)", rng.bounds)
	}
	want := []OrderResult{{Player: 1, Index: -1, Err: ErrOutOfDate}, {Player: 0, Index: 0}, {Player: 0, Index: 1}}
	if !reflect.DeepEqual(r.Orders, want) {
		t.Errorf("order results %+v, want %+v", r.Orders, want)
	}
	if pl := r.Game.Players[0]; pl.ResearchBudget != 30 || pl.Research.Current != Weapons {
		t.Errorf("research budget %d field %d, want 30 and weapons", pl.ResearchBudget, pl.Research.Current)
	}
	if got := r.Game.Fleets[0].Cargo.Colonists; got != 0 {
		t.Errorf("fleet 1 colonists %d, want 0 (dropped)", got)
	}
	// The 10 attackers lose to planet 2's 80 units: the defenders stay.
	if p := r.Game.Planets[1]; p.Owner != 1 {
		t.Errorf("planet 2 owner %d, want 1", p.Owner)
	}
}

// stubRaces is a RaceChecker that keeps a count, to test where
// GenerateTurn calls the race check and that it copies the checker.
type stubRaces struct{ calls int }

func (s *stubRaces) CheckRaces(g *Game) []Event {
	s.calls++
	g.Players[0].Race.GrowthRate = 1
	return []Event{{Kind: EventRacePenalized, Player: 0, Planet: -1, Fleet: -1}}
}

func (s *stubRaces) CloneRaces() RaceChecker { c := *s; return &c }

// The race check runs once a year at KERNEL.md step 2a, and its change
// applies to the year's growth (RACES.md "In a running game"); the input
// game's checker is left unchanged.
func TestPredictionRaceCheckInTurn(t *testing.T) {
	g := pgHomeworld()
	before := pgHomeworld()
	stub := &stubRaces{}
	g.Races = stub
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	if stub.calls != 0 || r.Game.Races.(*stubRaces).calls != 1 {
		t.Errorf("calls: input %d, result %d; want 0 and 1", stub.calls, r.Game.Races.(*stubRaces).calls)
	}
	n := 0
	for _, e := range r.Events {
		if e.Kind == EventRacePenalized {
			n++
		}
	}
	if n != 1 || r.Game.Players[0].Race.GrowthRate != 1 {
		t.Errorf("%d penalty events, growth rate %d; want 1 and 1", n, r.Game.Players[0].Race.GrowthRate)
	}
	plain, _ := GenerateTurn(withRules(before), nil, highRand{})
	if r.Game.Planets[0].Population >= plain.Game.Planets[0].Population {
		t.Errorf("population %d with the checked race, %d without; want less growth", r.Game.Planets[0].Population, plain.Game.Planets[0].Population)
	}
}
