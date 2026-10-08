package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

func TestHistoryKeepsLatestAndLostColonies(t *testing.T) {
	views := func(reps ...engine.PlanetReport) []engine.PlayerView {
		return []engine.PlayerView{{Player: 0, Planets: reps}}
	}
	h := history(nil).record(2400, views(
		engine.PlanetReport{Planet: 1, Level: engine.ReportOwn, Owner: 0},
		engine.PlanetReport{Planet: 2, Level: engine.ReportNormal, Owner: 1},
		engine.PlanetReport{Planet: 3, Level: engine.ReportNone},
	))
	// 2401: planet 1 (the player's colony) is lost and not reported;
	// planet 2 is seen again at position level.
	h2 := h.clone().record(2401, views(engine.PlanetReport{Planet: 2, Level: engine.ReportPosition, Owner: engine.NoOwner}))
	got := h2.list(0)
	if len(got) != 2 {
		t.Fatalf("history %+v: want planets 1 and 2 (a ReportNone is not a report)", got)
	}
	if got[0].Report.Planet != 1 || got[0].Year != 2400 || got[0].Report.Owner != 0 {
		t.Errorf("lost colony record %+v: want the 2400 record, still the player's own", got[0])
	}
	if got[1].Report.Planet != 2 || got[1].Year != 2401 || got[1].Report.Owner != engine.NoOwner {
		t.Errorf("planet 2 record %+v: want the 2401 report (ASSUMPTION G1)", got[1])
	}
	if old := h.list(0); old[1].Year != 2400 {
		t.Error("recording into a clone changed the original history")
	}
}

func TestReportUniverse(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	for p := range g.State.Players {
		r, err := g.Report(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Universe) != len(g.State.Planets) {
			t.Fatalf("player %d: universe has %d planets, game %d", p, len(r.Universe), len(g.State.Planets))
		}
		for i, u := range r.Universe {
			pl := g.State.Planets[i]
			if u.ID != pl.ID || u.Pos != pl.Pos || u.NameIndex != g.nameIndex[i] {
				t.Fatalf("universe planet %d = %+v, game %d at %v name %d", i, u, pl.ID, pl.Pos, g.nameIndex[i])
			}
		}
	}
}

// TestReportWormholesLastSeen: a known end the player does not see this
// year keeps the position and stability it had when last seen (ASSUMPTION
// G2), even after it moved and aged; a forgotten end leaves the report;
// another player sees nothing.
func TestReportWormholesLastSeen(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	sp := space(g.State)
	if sp == nil || len(sp.Wormholes) == 0 {
		t.Fatal("the smoke seed made no wormholes; pick a seed that does")
	}
	e := &sp.Wormholes[0].Ends[0]
	partner := sp.Wormholes[0].Ends[1].Pos
	// 2400: player 0 sees end 0 and its partner, and knows where it leads.
	e.MarkKnown(0)
	e.DestKnown = []bool{true}
	g.views[0].Objects.Wormholes = []int{objects.WormholeEndID(0, 0), objects.WormholeEndID(0, 1)}
	g.wormholes = wormholeHistory(nil).record(2400, g.views, sp)
	seenPos, seenStab := e.Pos, objects.JumpChance(*e)

	end0 := func(r Report) *KnownWormholeEnd {
		for i := range r.Wormholes {
			if r.Wormholes[i].End == 0 {
				return &r.Wormholes[i]
			}
		}
		return nil
	}
	r, _ := g.Report(0)
	k := end0(r)
	if k == nil || k.Year != 2400 || k.Pos != seenPos || k.Stability != seenStab || k.Destination == nil || *k.Destination != partner {
		t.Fatalf("seen end: %+v", k)
	}

	// 2401: the end jiggled and aged, out of the player's sight.
	g.State.Year = 2401
	e.Pos.X += 7
	e.Years += 40
	g.views[0].Objects.Wormholes = nil
	g.wormholes = g.wormholes.clone().record(2401, g.views, sp)
	if objects.JumpChance(*e) == seenStab {
		t.Fatal("test setup: aging did not change the stability")
	}
	r, _ = g.Report(0)
	k = end0(r)
	if k == nil || k.Year != 2400 || k.Pos != seenPos || k.Stability != seenStab || k.Destination != nil {
		t.Fatalf("unseen end: %+v; want the 2400 sighting at %v, stability %d, no destination", k, seenPos, seenStab)
	}
	if r1, _ := g.Report(1); end0(r1) != nil {
		t.Error("player 1 sees an end only player 0 knows")
	}

	// A jump makes everyone forget the end.
	e.Known = nil
	if r, _ = g.Report(0); end0(r) != nil {
		t.Error("a forgotten end is still reported")
	}
}

func TestReportSurvivesSaveLoad(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	drivers := smokeDrivers(3)
	// A rejected order, so the report carries an error.
	drivers[2] = DriverFunc(func(r Report) ([]engine.Order, error) {
		return []engine.Order{engine.RenameOrder{Fleet: -5, Name: "x"}}, nil
	})
	if err := g.Run(3, drivers); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	for p := range g.State.Players {
		a, _ := g.Report(p)
		b, _ := loaded.Report(p)
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if !bytes.Equal(ja, jb) {
			t.Fatalf("player %d's report differs after a reload", p)
		}
		for i := range a.Orders {
			if !errors.Is(a.Orders[i].Err, engine.ErrNoSuchObject) != !errors.Is(b.Orders[i].Err, engine.ErrNoSuchObject) {
				t.Fatalf("player %d order %d: errors.Is differs after a reload", p, i)
			}
		}
	}
	r, _ := loaded.Report(2)
	if len(r.Orders) != 1 || !errors.Is(r.Orders[0].Err, engine.ErrNoSuchObject) {
		t.Fatalf("player 2's outcomes after reload: %+v", r.Orders)
	}
}
