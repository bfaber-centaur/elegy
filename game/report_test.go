package game

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// twoPlayers is a small two-player new game and the views of its start.
func twoPlayers(t *testing.T) (engine.Game, []engine.PlayerView) {
	t.Helper()
	s := newgame.Settings{Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate}
	for range 2 {
		s.Players = append(s.Players, newgame.PlayerSetup{Race: races.Default()})
	}
	rng := newgame.NewRand(7)
	res, err := newgame.Generate(s, rng)
	if err != nil {
		t.Fatal(err)
	}
	g := res.Game
	return g, engine.Views(g, engine.PopulationEstimates(g, rng))
}

func TestReportHoldsOnlyOwnObjects(t *testing.T) {
	g, views := twoPlayers(t)
	for p := range g.Players {
		r, err := NewReport(g, p, views, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if r.Player != p || r.Year != g.Year || r.GameID != g.ID || r.View.Player != p {
			t.Fatalf("player %d: report stamped %d/%d/%d, view %d", p, r.Player, r.Year, r.GameID, r.View.Player)
		}
		if len(r.Planets) == 0 || len(r.Fleets) == 0 || len(r.Designs) == 0 {
			t.Fatalf("player %d: %d planets, %d fleets, %d designs; a new player owns some of each", p, len(r.Planets), len(r.Fleets), len(r.Designs))
		}
		for _, pl := range r.Planets {
			if pl.Owner != p {
				t.Errorf("player %d: report holds planet %d of owner %d", p, pl.ID, pl.Owner)
			}
		}
		for _, f := range r.Fleets {
			if f.Owner != p {
				t.Errorf("player %d: report holds fleet %d of owner %d", p, f.ID, f.Owner)
			}
		}
		for _, d := range r.Designs {
			if d.Slot.Owner != p || !reflect.DeepEqual(d.Design, g.Designs[d.Index]) {
				t.Errorf("player %d: design %+v is not the player's design %d", p, d.Slot, d.Index)
			}
		}
	}
}

func TestReportIsACopy(t *testing.T) {
	g, views := twoPlayers(t)
	type state struct {
		Players []engine.Player
		Planets []engine.Planet
		Fleets  []engine.Fleet
		Designs []engine.Design
	}
	snap := func() state { return deepCopy(state{g.Players, g.Planets, g.Fleets, g.Designs}) }
	before, beforeViews := snap(), deepCopy(views)
	r, err := NewReport(g, 0, views, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Self.Research.Levels[0] = -1
	if len(r.Self.DefaultQueue) > 0 {
		r.Self.DefaultQueue[0].Count = -1
	}
	if len(r.Planets[0].Queue) > 0 {
		r.Planets[0].Queue[0].Count = -1
	}
	r.Planets[0].Population = -1
	r.Fleets[0].Stacks[0].Count = -1
	r.Designs[0].Design.Slots[0].Count = -1
	if len(r.View.Planets) > 0 {
		r.View.Planets[0].Owner = 99
	}
	if !reflect.DeepEqual(snap(), before) || !reflect.DeepEqual(views, beforeViews) {
		t.Fatal("changing a report changed the game or the views")
	}
}

func TestReportFiltersEventsAndOrders(t *testing.T) {
	g, views := twoPlayers(t)
	events := []engine.Event{
		{Kind: engine.EventBuilt, Player: 0, Planet: 1, Fleet: -1},
		{Kind: engine.EventBuilt, Player: 1, Planet: 2, Fleet: -1},
	}
	results := []engine.OrderResult{
		{Player: 1, Index: -1, Err: engine.ErrOutOfDate},
		{Player: 0, Index: 0},
		{Player: 0, Index: 1, Err: engine.ErrNotYours},
	}
	r, err := NewReport(g, 0, views, events, results)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Events) != 1 || r.Events[0].Planet != 1 {
		t.Errorf("events = %+v, want only player 0's", r.Events)
	}
	if len(r.Orders) != 2 || r.Orders[0].Message != "" || !errors.Is(r.Orders[1].Err, engine.ErrNotYours) || r.Orders[1].Message != engine.ErrNotYours.Error() {
		t.Errorf("orders = %+v, want player 0's two results", r.Orders)
	}
}

func TestNewReportRejectsBadPlayer(t *testing.T) {
	g, views := twoPlayers(t)
	for _, p := range []int{-1, 2} {
		if _, err := NewReport(g, p, views, nil, nil); err == nil {
			t.Errorf("player %d: no error", p)
		}
	}
	if _, err := NewReport(g, 1, views[:1], nil, nil); err == nil {
		t.Error("missing view: no error")
	}
}

func TestDrivers(t *testing.T) {
	g, views := twoPlayers(t)
	r, err := NewReport(g, 1, views, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o, err := Idle.Orders(r); o != nil || err != nil {
		t.Errorf("Idle = %v, %v", o, err)
	}
	var got Report
	d := DriverFunc(func(r Report) ([]engine.Order, error) {
		got = r
		return []engine.Order{engine.ResearchOrder{}}, nil
	})
	if o, err := d.Orders(r); len(o) != 1 || err != nil || got.Player != 1 {
		t.Errorf("DriverFunc = %v, %v, saw player %d", o, err, got.Player)
	}
}

// TestDeepCopy checks deepCopy's assumption: the state types a report
// copies hold only exported fields and no interfaces, so a JSON copy
// loses nothing.
func TestDeepCopy(t *testing.T) {
	for _, v := range []any{engine.Player{}, engine.Planet{}, engine.Fleet{}, engine.Design{}, engine.PlayerView{}, engine.Event{}} {
		if bad := opaqueFields(reflect.TypeOf(v), map[reflect.Type]bool{}); len(bad) > 0 {
			t.Errorf("%T: %v would not survive a JSON copy", v, bad)
		}
	}
}

// opaqueFields lists the unexported, interface, function and channel
// fields reachable from t.
func opaqueFields(t reflect.Type, seen map[reflect.Type]bool) []string {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return opaqueFields(t.Elem(), seen)
	case reflect.Map:
		return append(opaqueFields(t.Key(), seen), opaqueFields(t.Elem(), seen)...)
	case reflect.Interface, reflect.Func, reflect.Chan:
		return []string{t.String()}
	case reflect.Struct:
		if seen[t] {
			return nil
		}
		seen[t] = true
		var out []string
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				out = append(out, t.String()+"."+f.Name)
				continue
			}
			out = append(out, opaqueFields(f.Type, seen)...)
		}
		return out
	}
	return nil
}
