package game

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// The scripted smoke game: a small galaxy of three players, each driven
// by expander below, advanced smokeYears years without the original
// program. The tests check that it runs, that every scripted order is
// accepted, that players expand, and that it is deterministic: the same
// seed and drivers give the same state hash every year, and a game saved
// and loaded mid-way continues identically.

const (
	smokeSeed  = 2026
	smokeYears = 40
	smokeSave  = 17 // the year index at which the reload test saves
)

func smokeSettings() newgame.Settings {
	s := newgame.Settings{Size: newgame.Small, Density: newgame.Normal, Positions: newgame.Moderate}
	for range 3 {
		s.Players = append(s.Players, newgame.PlayerSetup{Race: races.Default()})
	}
	return s
}

func smokeDrivers(n int) []Driver {
	d := make([]Driver, n)
	for i := range d {
		d[i] = DriverFunc(expander)
	}
	return d
}

// expander is a scripted player: research, factories and mines at every
// planet, colony ships loaded at a planet and sent to the nearest
// habitable unowned planet in its planet history, a colony ship built at
// the homeworld every eighth year, and idle ships without cargo space sent
// to a planet it has never seen, picked with the game's stream. It decides from its
// report alone, so it has no state to lose across a save.
func expander(r Report) ([]engine.Order, error) {
	var out []engine.Order
	if r.Year == 2400 {
		out = append(out, engine.ResearchOrder{Budget: 15, Field: r.Self.Research.Current, Next: engine.NextLowestField})
	}

	colonyShip := map[int]bool{} // design index → colonizer
	colonySlot := -1
	for _, d := range r.Designs {
		if !d.Slot.Starbase && strings.Contains(d.Design.Hull.Name, "Colony") {
			colonyShip[d.Index] = true
			if colonySlot < 0 {
				colonySlot = d.Slot.Slot
			}
		}
	}

	hw := -1
	own := map[int]engine.Planet{}
	for _, p := range r.Planets {
		own[p.ID] = p
		if p.Homeworld {
			hw = p.ID
		}
		if len(p.Queue) == 0 {
			q := []engine.QueueItem{{Kind: engine.ItemAutoFactories, Count: 50}, {Kind: engine.ItemAutoMines, Count: 50}}
			out = append(out, engine.QueueOrder{Planet: p.ID, Queue: q})
		}
	}
	if hw >= 0 && colonySlot >= 0 && r.Year > 2400 && r.Year%8 == 0 {
		q := append([]engine.QueueItem{{Kind: engine.ItemShip, Count: 1, Slot: colonySlot}}, own[hw].Queue...)
		out = append(out, engine.QueueOrder{Planet: hw, Queue: q})
	}

	// Planets already targeted by one of the player's fleets.
	taken := map[int]bool{}
	for _, f := range r.Fleets {
		for _, w := range f.Waypoints {
			if w.Target == engine.TargetPlanet {
				taken[w.ID] = true
			}
		}
	}
	var candidates []PlanetRecord
	seen := map[int]bool{}
	for _, h := range r.History {
		rep := h.Report
		if rep.Level >= engine.ReportNormal {
			seen[rep.Planet] = true
		}
		if _, mine := own[rep.Planet]; mine || rep.Level < engine.ReportNormal {
			continue
		}
		if rep.Owner == engine.NoOwner && engine.Habitability(r.Self.Race, rep.Env) > 0 {
			candidates = append(candidates, h)
		}
	}
	var unseen []UniversePlanet
	for _, u := range r.Universe {
		if !seen[u.ID] && !taken[u.ID] {
			unseen = append(unseen, u)
		}
	}

	for _, f := range r.Fleets {
		if len(f.Waypoints) > 0 {
			continue
		}
		colonizer := false
		capacity := 0
		for _, s := range f.Stacks {
			colonizer = colonizer || colonyShip[s.Design]
			capacity += s.Count * designByIndex(r, s.Design).CargoCapacity
		}
		at := -1
		for id, p := range own {
			if p.Pos == f.Pos {
				at = id
			}
		}
		switch {
		case colonizer && at >= 0:
			best := nearest(candidates, f.Pos, taken)
			if best < 0 {
				continue
			}
			target := candidates[best].Report
			taken[target.Planet] = true
			var amounts [engine.NumCargo + 1]int
			amounts[engine.CargoColonists] = capacity - f.Cargo.Colonists
			if amounts[engine.CargoColonists] > 0 {
				out = append(out, engine.CargoOrder{Fleet: f.ID, Target: engine.TargetPlanet, ID: at, Amounts: amounts})
			}
			out = append(out, engine.WaypointOrder{Fleet: f.ID, Waypoints: []engine.Waypoint{
				{Target: engine.TargetPlanet, ID: target.Planet, Warp: 5, Task: engine.Task{Kind: engine.TaskColonize}},
			}})
		case !colonizer && capacity == 0 && len(unseen) > 0 && r.Rand != nil:
			k := r.Rand.Intn(len(unseen))
			pick := unseen[k]
			unseen = append(unseen[:k], unseen[k+1:]...)
			out = append(out, engine.WaypointOrder{Fleet: f.ID, Waypoints: []engine.Waypoint{
				{Target: engine.TargetPlanet, ID: pick.ID, Warp: 6},
			}})
		}
	}
	return out, nil
}

func designByIndex(r Report, i int) engine.Design {
	for _, d := range r.Designs {
		if d.Index == i {
			return d.Design
		}
	}
	return engine.Design{}
}

// nearest is the index of the untaken record nearest to pos, ties to the
// lowest planet id, or −1.
func nearest(recs []PlanetRecord, pos engine.Point, taken map[int]bool) int {
	best, bestD := -1, 0
	for i, h := range recs {
		if taken[h.Report.Planet] {
			continue
		}
		dx, dy := h.Report.Pos.X-pos.X, h.Report.Pos.Y-pos.Y
		d := dx*dx + dy*dy
		if best < 0 || d < bestD || (d == bestD && h.Report.Planet < recs[best].Report.Planet) {
			best, bestD = i, d
		}
	}
	return best
}

// smokeRun plays the smoke game for years years from g and returns the
// state hash after each year, failing on any rejected order.
func smokeRun(t *testing.T, g *Game, years int) []string {
	t.Helper()
	drivers := smokeDrivers(len(g.State.Players))
	var hashes []string
	for range years {
		y, err := g.Advance(drivers)
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		for _, o := range y.Result.Orders {
			if o.Err != nil {
				order := "the whole file"
				for _, f := range y.Orders {
					if f.Player == o.Player && o.Index >= 0 && o.Index < len(f.Orders) {
						order = fmt.Sprintf("%T %+v", f.Orders[o.Index], f.Orders[o.Index])
					}
				}
				t.Errorf("year %d player %d order %d rejected: %v (%s)", g.State.Year-1, o.Player, o.Index, o.Err, order)
			}
		}
		h, err := g.Hash()
		if err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, h)
	}
	return hashes
}

func newSmoke(t *testing.T, seed uint64) *Game {
	t.Helper()
	g, err := New(engine.ElegyRules(), smokeSettings(), seed)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// diverged fails with where two games differ.
func diverged(t *testing.T, what string, year int, a, b *Game) {
	t.Helper()
	d, err := Diff(a, b, 12)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%s: diverged after year %d; first differences:\n  %s", what, year, strings.Join(d, "\n  "))
}

func TestSmokeGame(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	start := ownedPlanets(g.State)
	smokeRun(t, g, smokeYears)
	if g.State.Year != 2400+smokeYears {
		t.Fatalf("year %d after %d years", g.State.Year, smokeYears)
	}
	end := ownedPlanets(g.State)
	grew := 0
	for p := range g.State.Players {
		t.Logf("player %d: %d planets in 2400, %d in %d; population %d", p, start[p], end[p], g.State.Year, population(g.State, p))
		if end[p] > start[p] {
			grew++
		}
	}
	if grew == 0 {
		t.Error("no player colonized a planet in the scripted game")
	}
	for p := range g.State.Players {
		r, err := g.Report(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Planets) != end[p] || len(r.History) < len(r.Planets) {
			t.Errorf("player %d: report has %d planets (game %d) and %d history records", p, len(r.Planets), end[p], len(r.History))
		}
	}
}

func TestSmokeDeterministic(t *testing.T) {
	a, b := newSmoke(t, smokeSeed), newSmoke(t, smokeSeed)
	ha, hb := smokeRun(t, a, smokeYears), smokeRun(t, b, smokeYears)
	for i := range ha {
		if ha[i] != hb[i] {
			t.Fatalf("same seed and drivers: hashes differ after year index %d", i)
		}
	}
	if d, _ := Diff(a, b, 1); len(d) > 0 {
		diverged(t, "same seed and drivers", a.State.Year, a, b)
	}
	c := newSmoke(t, smokeSeed+1)
	if hc := smokeRun(t, c, 1); hc[0] == ha[0] {
		t.Error("a different seed gave the same game")
	}
}

func TestSmokeSaveLoadContinues(t *testing.T) {
	straight := newSmoke(t, smokeSeed)
	want := smokeRun(t, straight, smokeYears)

	g := newSmoke(t, smokeSeed)
	smokeRun(t, g, smokeSave)
	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	saved := buf.String()
	loaded, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	h0, _ := g.Hash()
	h1, _ := loaded.Hash()
	if h0 != h1 || h0 != want[smokeSave-1] {
		diverged(t, "save and load", g.State.Year, g, loaded)
	}
	// Save the loaded game again: the same bytes.
	var again bytes.Buffer
	if err := loaded.Save(&again); err != nil {
		t.Fatal(err)
	}
	if again.String() != saved {
		t.Fatal("a loaded game saves to different bytes")
	}
	got := smokeRun(t, loaded, smokeYears-smokeSave)
	for i, h := range got {
		if h != want[smokeSave+i] {
			ref := newSmoke(t, smokeSeed)
			smokeRun(t, ref, smokeSave+i+1)
			diverged(t, "continuing a loaded game", 2400+smokeSave+i, ref, loaded)
		}
	}
}

func TestAdvanceFailureLeavesGameUnchanged(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	before, _ := g.Hash()
	boom := errors.New("no orders today")
	drivers := smokeDrivers(len(g.State.Players))
	drivers[1] = DriverFunc(func(r Report) ([]engine.Order, error) {
		r.Rand.Intn(10) // a draw before failing must not stick
		return nil, boom
	})
	drivers[0] = DriverFunc(func(r Report) ([]engine.Order, error) {
		r.Rand.Intn(10)
		return nil, nil
	})
	_, err := g.Advance(drivers)
	var de *DriverError
	if !errors.As(err, &de) || de.Player != 1 || de.Year != 2400 || !errors.Is(err, boom) {
		t.Fatalf("err = %v, want player 1's driver error in 2400", err)
	}
	if after, _ := g.Hash(); after != before {
		t.Fatal("a failed year changed the game")
	}
	if _, err := g.Advance(drivers[:2]); !errors.Is(err, ErrDrivers) {
		t.Fatalf("two drivers for three players: %v", err)
	}
}

// TestDriversDrawInPlayerOrder checks that drivers run in player order on
// the game's stream, so a driver's draws shift the turn's later draws.
func TestDriversDrawInPlayerOrder(t *testing.T) {
	var order []int
	draw := DriverFunc(func(r Report) ([]engine.Order, error) {
		order = append(order, r.Player)
		r.Rand.Intn(1000)
		return nil, nil
	})
	a, b := newSmoke(t, smokeSeed), newSmoke(t, smokeSeed)
	if _, err := a.Advance([]Driver{draw, draw, draw}); err != nil {
		t.Fatal(err)
	}
	if !sort.IntsAreSorted(order) || len(order) != 3 {
		t.Fatalf("drivers ran in order %v", order)
	}
	if _, err := b.Advance([]Driver{nil, nil, nil}); err != nil {
		t.Fatal(err)
	}
	if a.rng.State() == b.rng.State() {
		t.Fatal("driver draws did not advance the game's stream")
	}
}

func TestLoadRejects(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	b, err := g.Encode()
	if err != nil {
		t.Fatal(err)
	}
	for name, doc := range map[string]string{
		"not json":      "{",
		"format":        strings.Replace(string(b), `"elegy-save"`, `"other"`, 1),
		"version":       strings.Replace(string(b), fmt.Sprintf(`"version": %d`, SaveVersion), `"version": 99`, 1),
		"older version": strings.Replace(string(b), fmt.Sprintf(`"version": %d`, SaveVersion), fmt.Sprintf(`"version": %d`, SaveVersion-1), 1),
		"levels":        strings.Replace(string(b), `"levels": [`, `"levels": [0, `, 1),
		"unknown field": strings.Replace(string(b), `"seed"`, `"extra": 1, "seed"`, 1),
	} {
		if _, err := Load(strings.NewReader(doc)); !errors.Is(err, ErrSave) {
			t.Errorf("%s: err = %v, want ErrSave", name, err)
		}
	}
}

func ownedPlanets(g engine.Game) map[int]int {
	out := map[int]int{}
	for _, p := range g.Planets {
		if p.Owner != engine.NoOwner {
			out[p.Owner]++
		}
	}
	return out
}

func population(g engine.Game, player int) int {
	n := 0
	for _, p := range g.Planets {
		if p.Owner == player {
			n += p.Population
		}
	}
	return n
}
