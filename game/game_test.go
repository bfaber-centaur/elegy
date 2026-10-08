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

func TestReportUniverseAndWormholes(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	sp := g.State.Objects.(*objects.Space)
	if len(sp.Wormholes) == 0 {
		t.Fatal("the smoke seed made no wormholes; pick a seed that does")
	}
	// Player 0 knows end 0 of wormhole 0 and where it leads, and sees the
	// other end this year; player 1 knows nothing.
	e := &sp.Wormholes[0].Ends[0]
	e.MarkKnown(0)
	e.DestKnown = []bool{true}
	g.views[0].Objects.Wormholes = append(g.views[0].Objects.Wormholes, objects.WormholeEndID(0, 1))
	r0, err := g.Report(0)
	if err != nil {
		t.Fatal(err)
	}
	r1, err := g.Report(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(r0.Universe) != len(g.State.Planets) || len(r1.Universe) != len(g.State.Planets) {
		t.Fatalf("universe has %d/%d planets, game %d", len(r0.Universe), len(r1.Universe), len(g.State.Planets))
	}
	for i, u := range r0.Universe {
		p := g.State.Planets[i]
		if u.ID != p.ID || u.Pos != p.Pos || u.NameIndex != g.nameIndex[i] {
			t.Fatalf("universe planet %d = %+v, game %d at %v name %d", i, u, p.ID, p.Pos, g.nameIndex[i])
		}
	}
	found := false
	for _, k := range r0.Wormholes {
		if k.End == 0 {
			found = true
			if k.Pos != e.Pos || k.Stability != objects.JumpChance(*e) || k.Destination == nil || *k.Destination != sp.Wormholes[0].Ends[1].Pos {
				t.Errorf("known end %+v", k)
			}
		}
	}
	if !found {
		t.Error("player 0 does not see the end it knows")
	}
	for _, k := range r1.Wormholes {
		if k.End == 0 {
			t.Error("player 1 sees an end only player 0 knows")
		}
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
