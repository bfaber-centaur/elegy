package engine

import "testing"

func TestGenerateTurnAdvancesYear(t *testing.T) {
	game := Game{
		Year: 2400,
	}

	result, err := GenerateTurn(game, nil, nil, highRand{})
	if err != nil {
		t.Fatalf("GenerateTurn() error = %v", err)
	}

	if got, want := result.Game.Year, 2401; got != want {
		t.Errorf("year = %d, want %d", got, want)
	}
}

func TestGenerateTurnRejectsNilRand(t *testing.T) {
	if _, err := GenerateTurn(pgHomeworld(), nil, Jrc3(), nil); err != ErrNilRand {
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
		r, err := GenerateTurn(g, nil, Jrc3(), highRand{})
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
	if _, err := GenerateTurn(g, nil, Jrc3(), highRand{}); err != nil {
		t.Fatal(err)
	}
	if g.Planets[0].Factories != 0 || g.Planets[0].Queue[0].Count != 20 || g.Planets[0].Population != 1050 {
		t.Errorf("input game changed: %+v", g.Planets[0])
	}
}
