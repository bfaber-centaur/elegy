package ai

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// loopSetup is a new tiny game with an idle human and expert Robotoid,
// Rototill and Cybertron, and fresh drivers for them.
func loopSetup(t *testing.T, rules engine.Ruleset, seed uint64) (*game.Game, []game.Driver) {
	t.Helper()
	s := newgame.Settings{Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate,
		Players: []newgame.PlayerSetup{{Race: races.Default()}}}
	for _, typ := range []int{1, 4, 5} {
		ps, err := newgame.ComputerPlayer(typ, newgame.Expert)
		if err != nil {
			t.Fatal(err)
		}
		s.Players = append(s.Players, ps)
	}
	g, err := game.New(rules, s, seed)
	if err != nil {
		t.Fatal(err)
	}
	return g, freshDrivers()
}

func freshDrivers() []game.Driver {
	return []game.Driver{game.Idle, NewDriver(Robotoid, Expert), NewDriver(Rototill, Expert), NewDriver(Cybertron, Expert)}
}

// advance plays years years and returns each year's state hash and every
// rejected order.
func advance(t *testing.T, g *game.Game, drivers []game.Driver, years int) (hashes, rejected []string) {
	t.Helper()
	for range years {
		y, err := g.Advance(drivers)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range y.Result.Orders {
			if o.Err != nil {
				rejected = append(rejected, o.Err.Error())
			}
		}
		h, err := g.Hash()
		if err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, h)
	}
	return hashes, rejected
}

// loopGame plays the game of loopSetup through the game loop
// (game.Game.Advance) for years years. It returns the game, each year's
// state hash and every rejected order.
func loopGame(t *testing.T, rules engine.Ruleset, seed uint64, years int) (*game.Game, []string, []string) {
	t.Helper()
	g, drivers := loopSetup(t, rules, seed)
	hashes, rejected := advance(t, g, drivers, years)
	return g, hashes, rejected
}

// The three personalities play 50 years through the game loop, under
// Elegy's rules and under jrc3-faithful's: every order is accepted, each
// keeps a planet, and the same seed gives the same state every year. A
// smoke test, not a parity check.
func TestDriversInGameLoop(t *testing.T) {
	for _, rules := range []engine.Ruleset{engine.ElegyRules(), engine.FaithfulRules()} {
		t.Run(rules.ID, func(t *testing.T) {
			g, hashes, rejected := loopGame(t, rules, 7, 50)
			for _, r := range rejected {
				t.Errorf("rejected: %s", r)
			}
			for i, name := range []string{"Robotoid", "Rototill", "Cybertron"} {
				owned := 0
				for _, p := range g.State.Planets {
					if p.Owner == i+1 {
						owned++
					}
				}
				t.Logf("year %d: %s owns %d planets", g.State.Year, name, owned)
				if owned == 0 {
					t.Errorf("%s lost every planet", name)
				}
			}
			_, again, _ := loopGame(t, rules, 7, 50)
			for i := range hashes {
				if hashes[i] != again[i] {
					t.Fatalf("year %d: the same seed gave a different state", 2401+i)
				}
			}
		})
	}
}

// A report without the game's stream is refused rather than planned with
// another one.
func TestDriverNeedsRand(t *testing.T) {
	if _, err := NewDriver(Rototill, Expert).Orders(game.Report{}); err != ErrNoRand {
		t.Errorf("error %v, want ErrNoRand", err)
	}
}

// A report without a valid ruleset is refused before any planning: the
// fleet estimates read the game's rules (engine.NewFleetShips).
func TestDriverNeedsRules(t *testing.T) {
	r := game.Report{Rand: &script{t: t}}
	if _, err := NewDriver(Rototill, Expert).Orders(r); !errors.Is(err, engine.ErrNoRuleset) {
		t.Errorf("error %v, want ErrNoRuleset", err)
	}
}

// A game saved in the middle and reloaded into fresh drivers continues
// exactly as the uninterrupted game: every year's state hash matches,
// for 50 years. So does a game saved and reloaded into fresh drivers
// every year, as cmd/elegy's turn command runs it. Under Elegy's rules and
// jrc3-faithful's.
func TestDriversSaveReload(t *testing.T) {
	const years, at = 50, 25
	reload := func(t *testing.T, g *game.Game) *game.Game {
		t.Helper()
		var buf bytes.Buffer
		if err := g.Save(&buf); err != nil {
			t.Fatal(err)
		}
		loaded, err := game.Load(&buf)
		if err != nil {
			t.Fatal(err)
		}
		return loaded
	}
	for _, rules := range []engine.Ruleset{engine.ElegyRules(), engine.FaithfulRules()} {
		t.Run(rules.ID, func(t *testing.T) {
			_, want, _ := loopGame(t, rules, 7, years)
			check := func(how string, got []string) {
				t.Helper()
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("%s: year %d differs from the uninterrupted game", how, 2401+i)
					}
				}
			}

			g, drivers := loopSetup(t, rules, 7)
			first, _ := advance(t, g, drivers, at)
			rest, _ := advance(t, reload(t, g), freshDrivers(), years-at)
			check("reloaded after 2425", append(first, rest...))

			g, _ = loopSetup(t, rules, 7)
			var yearly []string
			for range years {
				h, _ := advance(t, g, freshDrivers(), 1)
				yearly = append(yearly, h...)
				g = reload(t, g)
			}
			check("reloaded every year", yearly)
		})
	}
}

// scrapSpy records the fleets a driver orders scrapped.
type scrapSpy struct {
	d       *Driver
	scraped map[int]bool
}

func (s *scrapSpy) Orders(r game.Report) ([]engine.Order, error) {
	os, err := s.d.Orders(r)
	for _, o := range os {
		if w, ok := o.(engine.WaypointOrder); ok && w.Task.Kind == engine.TaskScrap {
			s.scraped[w.Fleet] = true
		}
	}
	return os, err
}

// AI.md §8 (captured 2400 orders): every expert type but Rototill scraps
// a starting fleet at its homeworld. Robotoid scraps its fleets holding a
// slot-0 ship until year index 20 (MEASURED AI-3), Cybertron its early
// slot-0 fleets (cybertron.md §5); the fleets are gone the next year.
func TestDriversScrapStartingScouts(t *testing.T) {
	g, _ := loopSetup(t, engine.ElegyRules(), 1)
	spies := []*scrapSpy{{d: NewDriver(Robotoid, Expert)}, {d: NewDriver(Rototill, Expert)}, {d: NewDriver(Cybertron, Expert)}}
	var scouts []map[int]bool
	for i, s := range spies {
		s.scraped = map[int]bool{}
		r, err := g.Report(i + 1)
		if err != nil {
			t.Fatal(err)
		}
		v := ViewOf(r, Expert)
		m := map[int]bool{}
		for j := range v.Fleets {
			if v.holds(&v.Fleets[j], 0) {
				m[v.Fleets[j].ID] = true
			}
		}
		scouts = append(scouts, m)
	}
	advance(t, g, []game.Driver{game.Idle, spies[0], spies[1], spies[2]}, 1)
	for i, s := range spies {
		want := s.d.Personality != Rototill
		if (len(s.scraped) > 0) != want {
			t.Errorf("%v: scrapped %v in 2400, want a scrap %v", s.d.Personality, s.scraped, want)
		}
		r, err := g.Report(i + 1)
		if err != nil {
			t.Fatal(err)
		}
		for id := range s.scraped {
			if !scouts[i][id] {
				t.Errorf("%v: fleet %d scrapped holds no slot-0 ship", s.d.Personality, id)
			}
			for _, f := range r.Fleets {
				if f.ID == id {
					t.Errorf("%v: fleet %d still there in 2401", s.d.Personality, id)
				}
			}
		}
	}
}

// splitSpy records, for the year just played, the indices of a driver's
// split orders and of the later orders that name a fleet split off the
// same turn (a negative fleet id, the split's NewFleet).
type splitSpy struct {
	d             *Driver
	splits, named []int
}

func (s *splitSpy) Orders(r game.Report) ([]engine.Order, error) {
	os, err := s.d.Orders(r)
	s.splits, s.named = nil, nil
	for k, o := range os {
		names := false
		switch o := o.(type) {
		case engine.SplitOrder:
			if o.NewFleet < 0 {
				s.splits = append(s.splits, k)
			}
			names = o.Fleet < 0
		case engine.WaypointOrder:
			names = o.Fleet < 0 || slices.ContainsFunc(o.Waypoints, func(w engine.Waypoint) bool { return w.Target == engine.TargetFleet && w.ID < 0 })
		case engine.MergeOrder:
			names = o.Into < 0 || slices.ContainsFunc(o.From, func(id int) bool { return id < 0 })
		}
		if names {
			s.named = append(s.named, k)
		}
	}
	return os, err
}

// From year index 81 Robotoid splits fleets (AI.md §10 "Splitting") and
// gives the new fleets orders the same turn, naming them by the split's
// NewFleet (engine ASSUMPTION L28). Whether a game reaches a fleet to
// split depends on its trajectory, so the test builds one: after 81 years
// an idle Robotoid fleet of one slot outside the split groups, moved into
// deep space, gets a ship of a slot in a group (0, 1 or 11–13), which the
// split moves out. Over the
// next 5 years Robotoid splits and names a new fleet, and the game
// accepts every such order.
func TestDriversOrderSplitFleets(t *testing.T) {
	g, _ := loopSetup(t, engine.ElegyRules(), 1)
	rb := &splitSpy{d: NewDriver(Robotoid, Expert)}
	drivers := []game.Driver{game.Idle, rb, NewDriver(Rototill, Expert), NewDriver(Cybertron, Expert)}
	advance(t, g, drivers, 81)
	plantMixedFleet(t, g, 1)
	splits, named := 0, 0
	for range 5 {
		y, err := g.Advance(drivers)
		if err != nil {
			t.Fatal(err)
		}
		check := map[int]bool{}
		for _, k := range append(rb.splits, rb.named...) {
			check[k] = true
		}
		for _, o := range y.Result.Orders {
			if o.Player == 1 && check[o.Index] && o.Err != nil {
				t.Errorf("Robotoid order %d rejected: %v", o.Index, o.Err)
			}
		}
		splits, named = splits+len(rb.splits), named+len(rb.named)
	}
	if splits == 0 || named == 0 {
		t.Errorf("Robotoid gave %d splits and %d orders naming a new fleet, want some of each", splits, named)
	}
}

// plantMixedFleet takes the first of player's idle fleets (no waypoints,
// no task) whose ships are all of one slot outside the split groups,
// moves it 1 ly into deep space, where no other own fleet can merge with
// it, and adds a ship of the first design the player has in a group slot
// (0, 1, 11, 12, 13). The new fleet the split makes is idle too, so the
// personality's fleet passes give it orders.
func plantMixedFleet(t *testing.T, g *game.Game, player int) {
	t.Helper()
	r, err := g.Report(player)
	if err != nil {
		t.Fatal(err)
	}
	v := ViewOf(r, Expert)
	grouped := []int{0, 1, 11, 12, 13}
	add := -1
	for _, k := range grouped {
		if d, ok := v.ship(k); ok {
			add = d.Index
			break
		}
	}
	if add < 0 {
		t.Fatalf("player %d has no design in slots %v", player, grouped)
	}
	for i := range g.State.Fleets {
		f := &g.State.Fleets[i]
		if f.Owner != player || len(f.Waypoints) > 0 || f.Task.Kind != engine.TaskNone || len(f.Stacks) != 1 || slices.Contains(grouped, v.shipSlot(f.Stacks[0].Design)) || v.shipSlot(f.Stacks[0].Design) < 0 {
			continue
		}
		// Moved 1 ly off into deep space, where no other own fleet can
		// merge with it.
		f.Pos.X++
		if slices.ContainsFunc(g.State.Fleets, func(o engine.Fleet) bool { return o.ID != f.ID && o.Pos == f.Pos }) ||
			slices.ContainsFunc(g.State.Planets, func(p engine.Planet) bool { return p.Pos == f.Pos }) {
			f.Pos.X--
			continue
		}
		f.Stacks = append(f.Stacks, engine.Stack{Design: add, Count: 1})
		return
	}
	t.Fatalf("player %d has no fleet of one slot outside %v", player, grouped)
}

// taskSpy records, for the year just played, the indices of a driver's
// waypoint orders that give the task kind on waypoint 0 or a waypoint.
type taskSpy struct {
	d    *Driver
	kind engine.TaskKind
	hits []int
}

func (s *taskSpy) Orders(r game.Report) ([]engine.Order, error) {
	os, err := s.d.Orders(r)
	s.hits = nil
	for k, o := range os {
		if w, ok := o.(engine.WaypointOrder); ok && (w.Task.Kind == s.kind || slices.ContainsFunc(w.Waypoints, func(p engine.Waypoint) bool { return p.Task.Kind == s.kind })) {
			s.hits = append(s.hits, k)
		}
	}
	return os, err
}

// From year index 41 Robotoid's idle scouts lay mines (robotoid.md §4),
// and the game accepts every such order over 70 years.
func TestDriversLayMines(t *testing.T) {
	g, _ := loopSetup(t, engine.ElegyRules(), 1)
	rb := &taskSpy{d: NewDriver(Robotoid, Expert), kind: engine.TaskLayMines}
	drivers := []game.Driver{game.Idle, rb, NewDriver(Rototill, Expert), NewDriver(Cybertron, Expert)}
	n := 0
	for range 70 {
		y, err := g.Advance(drivers)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range y.Result.Orders {
			if o.Player == 1 && slices.Contains(rb.hits, o.Index) && o.Err != nil {
				t.Errorf("Robotoid order %d rejected: %v", o.Index, o.Err)
			}
		}
		n += len(rb.hits)
	}
	if n == 0 {
		t.Error("Robotoid gave no lay-mines task in 70 years")
	}
}
