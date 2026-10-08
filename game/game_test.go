package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/objects"
	"github.com/bfaber-centaur/elegy/races"
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

// TestReportObjects: a player's report shows each space object it saw
// this year in full, whoever owns it (stars-elegy SCANNING.md "Space
// objects", MEASURED SC-038); an object it did not see is not reported.
func TestReportObjects(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	sp := space(g.State)
	if sp == nil {
		t.Fatal("the game has no space objects")
	}
	a, b := 0, 1
	posA, posB, posHidden := engine.Point{X: 1100, Y: 1100}, engine.Point{X: 1150, Y: 1100}, engine.Point{X: 1300, Y: 1300}
	sp.Minefields = append(sp.Minefields,
		objects.Minefield{Owner: a, Number: 0, Kind: objects.Standard, Pos: posA, Count: 1600, Detonate: true},
		objects.Minefield{Owner: b, Number: 0, Kind: objects.Heavy, Pos: posB, Count: 900, Detonate: true},
	)
	sp.Packets = append(sp.Packets,
		objects.Packet{Owner: a, Number: 0, Pos: posA, Target: 7, Warp: 9, Class: 2, Cargo: engine.Minerals{88, 0, 0}},
		objects.Packet{Owner: b, Number: 0, Pos: posB, Target: 2, Warp: 10, Class: 1, Cargo: engine.Minerals{0, 50, 0}},
		objects.Packet{Owner: b, Number: 1, Pos: posHidden, Target: 3, Warp: 8, Class: 1, Cargo: engine.Minerals{0, 0, 40}},
	)
	g.views[a].Objects.Minefields = [][2]int{{a, 0}, {b, 0}}
	g.views[a].Objects.Packets = [][2]int{{a, 0}, {b, 0}}
	g.views[b].Objects.Minefields = [][2]int{{b, 0}}
	g.views[b].Objects.Packets = [][2]int{{b, 0}, {b, 1}}

	r, err := g.Report(a)
	if err != nil {
		t.Fatal(err)
	}
	want := objects.ObjectReport{
		Minefields: []objects.MinefieldSighting{
			{Owner: a, Number: 0, Pos: posA, Mines: 1600, Kind: objects.Standard, Detonate: true},
			{Owner: b, Number: 0, Pos: posB, Mines: 900, Kind: objects.Heavy, Detonate: true},
		},
		Packets: []objects.PacketSighting{
			{Owner: a, Number: 0, Pos: posA, Target: 7, Warp: 9, Class: 2, Cargo: engine.Minerals{88, 0, 0}},
			{Owner: b, Number: 0, Pos: posB, Target: 2, Warp: 10, Class: 1, Cargo: engine.Minerals{0, 50, 0}},
		},
	}
	if !reflect.DeepEqual(r.Objects, want) {
		t.Fatalf("player %d's objects =\n%+v\nwant\n%+v", a, r.Objects, want)
	}

	// Player b sees only what it saw: its own field and both its packets,
	// the hidden one included, and nothing of a's. Every sighting is the
	// whole object (stars-elegy SCANNING.md "Space objects", MEASURED
	// SC-038).
	r, err = g.Report(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Objects.Minefields) != 1 || !r.Objects.Minefields[0].Detonate || r.Objects.Minefields[0].Owner != b {
		t.Errorf("player %d's minefields: %+v", b, r.Objects.Minefields)
	}
	wantB := []objects.PacketSighting{
		{Owner: b, Number: 0, Pos: posB, Target: 2, Warp: 10, Class: 1, Cargo: engine.Minerals{0, 50, 0}},
		{Owner: b, Number: 1, Pos: posHidden, Target: 3, Warp: 8, Class: 1, Cargo: engine.Minerals{0, 0, 40}},
	}
	if !reflect.DeepEqual(r.Objects.Packets, wantB) {
		t.Errorf("player %d's packets =\n%+v\nwant\n%+v", b, r.Objects.Packets, wantB)
	}
}

// Another player's design is reported with hull and mass when seen
// partially, with all its parts once seen in full, and stays known in
// later years out of sight (MEASURED SC-037, computer players' files),
// through a save and load.
// What is reported is what was shown: a design the owner later puts in
// the same slot, under the same index, is not revealed.
func TestReportKnownDesigns(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	slotOf := func(owner int, starbase bool) engine.DesignSlot {
		for _, s := range g.State.DesignSlots {
			if s.Owner == owner && s.Starbase == starbase {
				return s
			}
		}
		t.Fatalf("no design of player %d", owner)
		return engine.DesignSlot{}
	}
	theirs, other := slotOf(1, true).Design, slotOf(2, false).Design
	sight := func(d int, full bool) engine.DesignSighting {
		des := g.State.Designs[d]
		return engine.DesignSighting{Design: d, Full: full, Hull: des.Hull.Name, Mass: des.Mass}
	}
	see := func(year int, s ...engine.DesignSighting) {
		g.State.Year = year
		g.views[0].Designs = s
		g.designs = g.designs.clone().record(year, g.views, g.State.Designs)
	}
	known := func(r Report, d int) *KnownDesign {
		for i := range r.KnownDesigns {
			if r.KnownDesigns[i].Index == d {
				return &r.KnownDesigns[i]
			}
		}
		return nil
	}
	want := deepCopy(g.State.Designs[theirs])
	wantOther := deepCopy(g.State.Designs[other])
	g.designs = nil

	// 2400: player 0 sees player 1's starbase partially.
	see(2400, sight(theirs, false))
	r, _ := g.Report(0)
	k := known(r, theirs)
	if k == nil || k.Full || k.Design != nil || k.Hull != want.Hull.Name || k.Mass != want.Mass || k.Year != 2400 {
		t.Fatalf("partial sighting: %+v", k)
	}
	// 2401: a battle shows it in full, and player 2's design partially.
	see(2401, sight(theirs, true), sight(other, false))
	// 2402: out of sight.
	see(2402)

	// The owners put new designs in both slots, under the same indexes
	// (engine.DesignOrder replaces an unused slot's design in place).
	g.State.Designs = append([]engine.Design(nil), g.State.Designs...)
	g.State.Designs[theirs].Name, g.State.Designs[theirs].Mass = "Replaced", want.Mass+100
	g.State.Designs[other].Hull.Name, g.State.Designs[other].Mass = "Other Hull", wantOther.Mass+7

	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []*Game{g, loaded} {
		r, _ = x.Report(0)
		k = known(r, theirs)
		if k == nil || !k.Full || k.Year != 2401 || k.Design == nil || !reflect.DeepEqual(*k.Design, want) || k.Mass != want.Mass {
			t.Fatalf("full design out of sight, then replaced: %+v", k)
		}
		if o := known(r, other); o == nil || o.Full || o.Design != nil || o.Hull != wantOther.Hull.Name || o.Mass != wantOther.Mass {
			t.Fatalf("partial design, then replaced: %+v", o)
		}
		if len(r.KnownDesigns) != 2 {
			t.Fatalf("%d known designs, want 2", len(r.KnownDesigns))
		}
		if r1, _ := x.Report(1); len(r1.KnownDesigns) != 0 {
			t.Fatalf("player 1 knows %+v, which only player 0 saw", r1.KnownDesigns)
		}
	}

	// 2403: a partial sighting of the new design (another mass) replaces
	// the record and drops the old full design.
	see(2403, sight(theirs, false))
	r, _ = g.Report(0)
	if k = known(r, theirs); k == nil || k.Full || k.Design != nil || k.Mass != want.Mass+100 || k.Year != 2403 {
		t.Fatalf("partial sighting of the replacement: %+v", k)
	}
	// 2404: seen in full. 2405: the owner refits it with other parts,
	// keeping hull and mass; a partial sighting keeps the 2404 design in
	// full, stale (ASSUMPTION G3), and does not reveal that the parts
	// changed.
	see(2404, sight(theirs, true))
	if len(g.State.Designs[theirs].Slots) == 0 {
		t.Fatal("test setup: the starbase design has no parts to change")
	}
	g.State.Designs = append([]engine.Design(nil), g.State.Designs...)
	g.State.Designs[theirs].Name = "Refitted"
	g.State.Designs[theirs].Slots = g.State.Designs[theirs].Slots[1:]
	see(2405, sight(theirs, false))
	r, _ = g.Report(0)
	if k = known(r, theirs); k == nil || !k.Full || k.Design == nil || k.Design.Name != "Replaced" || k.Year != 2405 {
		t.Fatalf("partial sighting with the same hull and mass: %+v", k)
	}
}

// A design's creation year and picture reach the owner's report and
// survive a save and load; the hash and Diff see them.
func TestDesignYearAndPictureSaved(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	for _, d := range mustReport(t, g, 0).Designs {
		if d.Slot.Created != 2400 || d.Slot.Picture != 0 {
			t.Fatalf("starting design slot %+v: want created 2400, picture 0 (ASSUMPTION B2)", d.Slot)
		}
	}
	scout := mustReport(t, g, 0).Designs[0]
	for _, d := range mustReport(t, g, 0).Designs {
		if !d.Slot.Starbase {
			scout = d
			break
		}
	}
	order := engine.DesignOrder{Slot: 9, Name: "Pictured", Hull: scout.Design.Hull.Name, Picture: 2}
	drivers := make([]Driver, len(g.State.Players))
	drivers[0] = DriverFunc(func(Report) ([]engine.Order, error) { return []engine.Order{order}, nil })
	y, err := g.Advance(drivers)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range y.Result.Orders {
		if o.Err != nil {
			t.Fatalf("design order rejected: %v", o.Err)
		}
	}
	find := func(x *Game) OwnDesign {
		for _, d := range mustReport(t, x, 0).Designs {
			if !d.Slot.Starbase && d.Slot.Slot == 9 {
				return d
			}
		}
		t.Fatal("the new design is not in the report")
		return OwnDesign{}
	}
	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []*Game{g, loaded} {
		if d := find(x); d.Slot.Created != 2400 || d.Slot.Picture != 2 {
			t.Fatalf("new design slot %+v: want created 2400 (the year its order applied), picture 2", d.Slot)
		}
	}
	h1, _ := g.Hash()
	h2, _ := loaded.Hash()
	if h1 != h2 {
		t.Fatal("hash changed across a save and load")
	}
	for i, s := range loaded.State.DesignSlots {
		if s.Owner == 0 && !s.Starbase && s.Slot == 9 {
			slots := append([]engine.DesignSlot(nil), loaded.State.DesignSlots...)
			slots[i].Picture = 3
			loaded.State.DesignSlots = slots
		}
	}
	if h3, _ := loaded.Hash(); h3 == h1 {
		t.Fatal("the hash does not cover a design's picture")
	}
	diffs, err := Diff(g, loaded, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || !strings.Contains(diffs[0], "Picture") {
		t.Fatalf("Diff: %v", diffs)
	}
}

func mustReport(t *testing.T, g *Game, p int) Report {
	t.Helper()
	r, err := g.Report(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
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

func TestGameID(t *testing.T) {
	a, b, c := newSmoke(t, 1), newSmoke(t, 1), newSmoke(t, 2)
	if a.State.ID == 0 || a.State.ID != b.State.ID || a.State.ID == c.State.ID {
		t.Fatalf("game ids %d, %d (same seed), %d (another seed)", a.State.ID, b.State.ID, c.State.ID)
	}
}

// TestRulesetsCoexist: games with the Elegy and the faithful rulesets
// run side by side in one process, each keeps its own ruleset through
// every year and a save and load, and the two save differently.
func TestRulesetsCoexist(t *testing.T) {
	newGame := func(rules engine.Ruleset) *Game {
		g, err := New(rules, smokeSettings(), smokeSeed)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	elegy, faithful := newGame(engine.ElegyRules()), newGame(engine.FaithfulRules())
	he, _ := elegy.Hash()
	hf, _ := faithful.Hash()
	if he == hf {
		t.Fatal("two rulesets give the same save")
	}
	// Interleave the years of the two games.
	drivers := smokeDrivers(3)
	for range 10 {
		for _, g := range []*Game{elegy, faithful} {
			if _, err := g.Advance(drivers); err != nil {
				t.Fatal(err)
			}
		}
	}
	for want, g := range map[engine.Ruleset]*Game{engine.ElegyRules(): elegy, engine.FaithfulRules(): faithful} {
		if g.State.Rules != want {
			t.Fatalf("after ten years the game has ruleset %q, want %q", g.State.Rules.ID, want.ID)
		}
		var buf bytes.Buffer
		if err := g.Save(&buf); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(&buf)
		if err != nil {
			t.Fatal(err)
		}
		h0, _ := g.Hash()
		h1, _ := loaded.Hash()
		if r, _ := loaded.Report(0); r.Rules != want {
			t.Fatalf("%s: the report carries ruleset %q", want.ID, r.Rules.ID)
		}
		if loaded.State.Rules != want || h0 != h1 {
			t.Fatalf("%s: reload gives ruleset %+v, hash equal %v", want.ID, loaded.State.Rules, h0 == h1)
		}
	}
	// The same game alone matches the interleaved one.
	alone := newGame(engine.FaithfulRules())
	if err := alone.Run(10, drivers); err != nil {
		t.Fatal(err)
	}
	ha, _ := alone.Hash()
	hf, _ = faithful.Hash()
	if ha != hf {
		diverged(t, "a game run next to another ruleset's", alone.State.Year, alone, faithful)
	}
}

func TestLoadRejectsInvalidRuleset(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	b, err := g.Encode()
	if err != nil {
		t.Fatal(err)
	}
	doc := strings.Replace(string(b), `"id": "`+engine.ElegyRulesID+`"`, `"id": ""`, 1)
	if doc == string(b) {
		t.Fatal("test setup: ruleset id not found in the save")
	}
	if _, err := Load(strings.NewReader(doc)); !errors.Is(err, ErrSave) {
		t.Fatalf("err = %v, want ErrSave", err)
	}
}

// A computer player's level is part of the game and survives a save and
// load; a human player has none.
func TestComputerLevelSaved(t *testing.T) {
	ca, err := newgame.ComputerPlayer(4, newgame.Harder)
	if err != nil {
		t.Fatal(err)
	}
	s := newgame.Settings{Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate,
		Players: []newgame.PlayerSetup{{Race: races.Default()}, ca}}
	g, err := New(engine.ElegyRules(), s, 3)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	g2, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []*Game{g, g2} {
		if _, ok := x.ComputerLevel(0); ok {
			t.Error("player 0 is human but has a computer level")
		}
		if lvl, ok := x.ComputerLevel(1); !ok || lvl != newgame.Harder || !x.State.Players[1].Computer {
			t.Errorf("player 1: level %v, computer %v", lvl, ok)
		}
	}
}
