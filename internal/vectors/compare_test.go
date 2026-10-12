package vectors

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// A mismatch is always reported, even when another field of the same
// expectation cannot be compared; that field's skip reason rides along
// for the tally (pvResult). Each checker is given a field it cannot
// compare that sorts before the mismatching one ("aaa").

func pvEq(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	var eq map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &eq); err != nil {
		t.Fatal(err)
	}
	return eq
}

// pvSplit is a check's mismatch and also-skipped reason, as runVector
// reads them.
func pvSplit(msg string) (mismatch, skip string) {
	if strings.HasPrefix(msg, "skip: ") {
		return "", strings.TrimPrefix(msg, "skip: ")
	}
	mismatch, skip, _ = strings.Cut(msg, pvAlsoSkipped)
	return mismatch, skip
}

func pvWant(t *testing.T, name, msg, mismatch, skip string) {
	t.Helper()
	if m, s := pvSplit(msg); m != mismatch || s != skip {
		t.Errorf("%s: %q; want mismatch %q, skipped %q", name, msg, mismatch, skip)
	}
}

func TestPvResult(t *testing.T) {
	for _, c := range []struct {
		errs       []string
		skip, want string
	}{
		{nil, "", ""},
		{nil, "x", "skip: x"},
		{[]string{"a", "b"}, "", "a; b"},
		{[]string{"a"}, "x", "a" + pvAlsoSkipped + "x"},
	} {
		if got := pvResult(c.errs, c.skip); got != c.want {
			t.Errorf("pvResult(%q, %q) = %q, want %q", c.errs, c.skip, got, c.want)
		}
	}
}

func TestPlanetMismatchBesideSkip(t *testing.T) {
	var l pvLoaded
	p := engine.Planet{Owner: 0, Population: 100}
	msg := l.planetEquals(nil, &p, pvEq(t, `{"aaa": 1, "owner": -1, "population": 100}`), nil)
	pvWant(t, "unknown field", msg, "owner = 0, want -1", "planet field aaa")
	msg = l.planetEquals(nil, &p, pvEq(t, `{"environment": null, "owner": -1}`), nil)
	pvWant(t, "null environment", msg, "owner = 0, want -1", "environment null")
	// A null original environment is the current one (TAKEOVER.md
	// "Capture": not stored separately when equal).
	p.Env, p.OrigEnv = [3]int{1, 2, 3}, [3]int{1, 2, 3}
	pvWant(t, "original equal", l.planetEquals(nil, &p, pvEq(t, `{"original_environment": null}`), nil), "", "")
	p.OrigEnv = [3]int{1, 2, 4}
	pvWant(t, "original differs", l.planetEquals(nil, &p, pvEq(t, `{"original_environment": null}`), nil), "original_environment = [1 2 4], want [1 2 3]", "")
	msg = l.planetEquals(nil, &p, pvEq(t, `{"leftover_to_research": true, "population": 7}`), nil)
	pvWant(t, "leftover", msg, "population = 100, want 7", "player default queue settings")
	msg = l.planetEquals(nil, &p, pvEq(t, `{"excess": null, "owner": -1, "population": null}`), nil)
	pvWant(t, "null fields", msg, "owner = 0, want -1", "planet field excess null")
	msg = l.planetEquals(nil, &p, pvEq(t, `{"aaa": 1, "owner": 0}`), nil)
	pvWant(t, "match", msg, "", "planet field aaa")
}

func TestPlayerMismatchBesideSkip(t *testing.T) {
	var l pvLoaded
	g := &engine.Game{Players: []engine.Player{{}}}
	g.Players[0].Research.Accumulated[engine.Energy] = 15
	msg := l.playerEquals(g, 0, pvEq(t, `{"aaa": 1, "research_accumulated": {"energy": 0}}`), nil)
	pvWant(t, "unknown field", msg, "research_accumulated.energy = 15, want 0", "player field aaa")
	msg = l.playerEquals(g, 0, pvEq(t, `{"race": 7, "research_accumulated": {"energy": 0}}`), nil)
	if m, s := pvSplit(msg); m != "research_accumulated.energy = 15, want 0" || !strings.HasPrefix(s, "race: ") {
		t.Errorf("bad race: %q", msg)
	}
	// A field the harness cannot reproduce is skipped by itself.
	g.Players[0].Research.Levels[engine.Energy] = 3
	msg = l.playerEquals(g, 0, pvEq(t, `{"research_accumulated": {"energy": 15}, "tech": {"energy": 4}}`), map[string]string{"research_accumulated": "why"})
	pvWant(t, "unsure field", msg, "tech.energy = 3, want 4", "why")
}

// TestConcentrationsAndResearchFields compares the planet concentration
// bytes and the player's current and next research fields, which were
// skipped before: a mismatch fails, an unknown value is skipped, never
// passed.
func TestConcentrationsAndResearchFields(t *testing.T) {
	var l pvLoaded
	p := engine.Planet{}
	p.Deposits[0].Concentration, p.Deposits[1].Concentration, p.Deposits[2].Concentration = 74, 62, 63
	pvWant(t, "concentrations match", l.planetEquals(nil, &p, pvEq(t, `{"concentrations": [74, 62, 63]}`), nil), "", "")
	pvWant(t, "concentrations differ", l.planetEquals(nil, &p, pvEq(t, `{"concentrations": [73, 62, 63]}`), nil), "concentrations = [74 62 63], want [73 62 63]", "")
	g := &engine.Game{Players: []engine.Player{{}}}
	g.Players[0].Research.Current = engine.Weapons
	g.Players[0].Research.Next = engine.NextLowestField
	pvWant(t, "fields match", l.playerEquals(g, 0, pvEq(t, `{"research_field": "weapons", "research_next_field": "lowest"}`), nil), "", "")
	msg := l.playerEquals(g, 0, pvEq(t, `{"research_field": "energy", "research_next_field": "same"}`), nil)
	want := fmt.Sprintf("research_field = %d, want %d; research_next_field = %d, want %d", engine.Weapons, engine.Energy, engine.NextLowestField, engine.NextSameField)
	pvWant(t, "fields differ", msg, want, "")
	g.Players[0].Research.Next = engine.Biotech
	pvWant(t, "next field by name", l.playerEquals(g, 0, pvEq(t, `{"research_next_field": "biotechnology"}`), nil), "", "")
	pvWant(t, "unknown field value", l.playerEquals(g, 0, pvEq(t, `{"research_field": "alchemy"}`), nil), "", `research_field "alchemy"`)
	pvWant(t, "unknown next value", l.playerEquals(g, 0, pvEq(t, `{"research_next_field": 3}`), nil), "", "research_next_field 3")
}

// TestStarbase compares a planet's starbase: the owner's starbase design
// slot (the loaded design, or one a design order put there) and damage,
// and null for none.
func TestStarbase(t *testing.T) {
	l := pvLoaded{design: map[[2]int]int{{1, -1 - 2}: 4}}
	g := &engine.Game{Designs: make([]engine.Design, 6)}
	p := engine.Planet{Owner: 1, HasStarbase: true, StarbaseDesign: 4, StarbaseDamage: 180}
	pvWant(t, "match", l.planetEquals(g, &p, pvEq(t, `{"starbase": {"design": 2, "damage": 180}}`), nil), "", "")
	pvWant(t, "damage", l.planetEquals(g, &p, pvEq(t, `{"starbase": {"design": 2, "damage": 0}}`), nil), "starbase = design 4 damage 180, want design 4 damage 0", "")
	pvWant(t, "none wanted", l.planetEquals(g, &p, pvEq(t, `{"starbase": null}`), nil), "starbase = design 4, want none", "")
	pvWant(t, "unknown slot", l.planetEquals(g, &p, pvEq(t, `{"starbase": {"design": 3, "damage": 0}}`), nil), "", "starbase design 3")
	// A design order's slot wins over the loaded one.
	g.DesignSlots = []engine.DesignSlot{{Owner: 1, Starbase: true, Slot: 2, Design: 5}}
	pvWant(t, "ordered design", l.planetEquals(g, &p, pvEq(t, `{"starbase": {"design": 2, "damage": 180}}`), nil), "starbase = design 4 damage 180, want design 5 damage 180", "")
	// No starbase, with leftover design and damage values that would
	// otherwise match.
	p.HasStarbase, p.StarbaseDesign, p.StarbaseDamage = false, 5, 0
	pvWant(t, "none", l.planetEquals(g, &p, pvEq(t, `{"starbase": null}`), nil), "", "")
	pvWant(t, "missing", l.planetEquals(g, &p, pvEq(t, `{"starbase": {"design": 2, "damage": 0}}`), nil), "starbase = none, want design 5 damage 0", "")
}

func TestFleetMismatchBesideSkip(t *testing.T) {
	var l pvLoaded
	f := engine.Fleet{Pos: engine.Point{X: 5, Y: 6}}
	msg := l.fleetEquals(nil, &f, pvEq(t, `{"aaa": 1, "x": 4}`))
	pvWant(t, "unknown field", msg, "x = 5, want 4", "fleet field aaa")
	// waypoint 0's task cannot be read, its position still mismatches,
	// and a later waypoint that cannot be read does not hide it either.
	msg = l.fleetEquals(nil, &f, pvEq(t, `{"waypoints": [{"x": 5, "y": 7, "task": {"kind": "bogus"}}]}`))
	pvWant(t, "waypoint 0 task", msg, "waypoints[0] position = {5 6}, want {5 7}", "waypoint task bogus")
	f.Waypoints = []engine.Waypoint{{Pos: engine.Point{X: 9, Y: 9}}, {Pos: engine.Point{X: 1, Y: 1}}}
	msg = l.fleetEquals(nil, &f, pvEq(t, `{"waypoints": [{"x": 5, "y": 6, "task": {"kind": "none"}}, {"x": 9, "y": 9, "target": {"kind": "space"}, "task": {"kind": "bogus"}}, {"x": 2, "y": 1, "target": {"kind": "space"}, "task": {"kind": "none"}}]}`))
	if m, s := pvSplit(msg); !strings.HasPrefix(m, "waypoints[2] = ") || s != "waypoint task bogus" {
		t.Errorf("later waypoint: %q", msg)
	}
	// A waypoint the harness cannot read is not compared at all: its
	// position differs from the fleet's, and nothing is reported.
	msg = l.fleetEquals(nil, &f, pvEq(t, `{"waypoints": [{"x": 5, "y": 6, "task": {"kind": "none"}}, {"x": 8, "y": 8, "target": {"kind": "space"}, "task": {"kind": "bogus"}}, {"x": 1, "y": 1, "target": {"kind": "space"}, "task": {"kind": "none"}}]}`))
	pvWant(t, "unread waypoint", msg, "", "waypoint task bogus")
	// Waypoints that all match do not clear an earlier field's skip.
	f.Waypoints = nil
	msg = l.fleetEquals(nil, &f, pvEq(t, `{"aaa": 1, "waypoints": [{"x": 5, "y": 6, "task": {"kind": "none"}}]}`))
	pvWant(t, "skip before waypoints", msg, "", "fleet field aaa")
}

func TestCompareMismatchBesideSkip(t *testing.T) {
	got := map[string]any{"b": 1, "c": 2}
	msg := pvCompare("thing", got, pvEq(t, `{"a": 1, "b": 2, "c": 3}`), 0, nil)
	pvWant(t, "missing field", msg, "thing: b = 1, want 2; thing: c = 2, want 3", "thing field a")
	msg = pvCompare("thing", got, pvEq(t, `{"a": 1, "c": 3}`), 0, map[string]string{"a": "no a"})
	pvWant(t, "skipped field", msg, "thing: c = 2, want 3", "no a")
	// The first field skipped names the skip, whichever way each was.
	msg = pvCompare("thing", got, pvEq(t, `{"a": 1, "b": 1, "x": 1}`), 0, map[string]string{"b": "no b", "x": "no x"})
	pvWant(t, "missing then skipped", msg, "", "thing field a")
	msg = pvCompare("thing", map[string]any{}, pvEq(t, `{"a": 1, "b": 1}`), 0, map[string]string{"a": "no a"})
	pvWant(t, "skipped then missing", msg, "", "no a")
	msg = pvCompare("thing", map[string]any{}, pvEq(t, `{"a": 1, "b": 1}`), 0, nil)
	pvWant(t, "two missing", msg, "", "thing field a")
	msg = pvCompare("thing", map[string]any{}, pvEq(t, `{"a": 1, "b": 1}`), 0, map[string]string{"a": "no a", "b": "no b"})
	pvWant(t, "two skipped", msg, "", "no a")
}

// TestViewExact checks that a view is compared exactly: a fleet seen
// without its cargo is level 3, and a 4 expected is a mismatch.
func TestViewExact(t *testing.T) {
	f := engine.Fleet{ID: 7, Owner: 1}
	viewer := 0
	l := pvLoaded{views: [][]engine.PlayerView{nil, {{Player: 0, Fleets: []engine.FleetSighting{{Fleet: 7, Owner: 1}}}}}}
	e := pvExpect{Year: 1, Kind: "view", Viewer: &viewer, Subject: json.RawMessage(`{"kind": "fleet", "owner": 1, "id": 7}`)}
	fleet := func(owner, id int) *engine.Fleet { return &f }
	msg := l.viewCheck(nil, e, pvEq(t, `{"level": 4}`), fleet)
	pvWant(t, "level", msg, "fleet: level = 3, want 4", "")
	msg = l.viewCheck(nil, e, pvEq(t, `{"level": 3, "known": true}`), fleet)
	pvWant(t, "match", msg, "", "")
}

// TestRunReportsMismatchBesideSkip runs PQ-001-C01 with a field the
// harness cannot compare added to its first planet expectation, whose
// mines are also made wrong: the case fails on the mines, and the
// unknown field is still tallied as skipped.
func TestRunReportsMismatchBesideSkip(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	e := &v.Cases[0].Expect[0]
	eq := pvEq(t, string(e.Equals))
	eq["aaa"] = json.RawMessage("1")
	eq["mines"] = json.RawMessage("-1")
	if e.Equals, err = json.Marshal(eq); err != nil {
		t.Fatal(err)
	}
	r := Run(v, 0)[0]
	if r.Status != "fail" || !strings.Contains(r.Why, "mines = ") || strings.Contains(r.Why, "aaa") {
		t.Errorf("status %s, why %q; want a fail on mines alone", r.Status, r.Why)
	}
	if len(r.Skipped) == 0 || r.Skipped[0] != "planet field aaa" {
		t.Errorf("skipped %q, want planet field aaa first", r.Skipped)
	}
}

func TestPlayerUnsure(t *testing.T) {
	g := &engine.Game{Players: []engine.Player{{}, {Computer: true}, {Computer: true}, {}, {}}}
	g.Players[3].Race.PRT = engine.PRTSuperStealth
	g.Players[4].Race.PRT = engine.PRTSuperStealth
	g.Players = append(g.Players, engine.Player{Computer: true})
	l := pvLoaded{g: *g, queued: map[int]bool{0: true, 4: true, 5: true}, ordered: map[int]bool{2: true}}
	research := []string{"tech", "research_accumulated", "research_field", "research_next_field"}
	for _, c := range []struct {
		id   int
		want map[string]string
	}{
		{0, map[string]string{"tech": "production queue", "research_accumulated": "production queue", "research_field": "production queue", "research_next_field": "production queue"}},
		{1, map[string]string{"tech": pvPlannedWhy, "research_accumulated": pvPlannedWhy, "research_field": pvPlannedWhy, "research_next_field": pvPlannedWhy}},
		{2, map[string]string{}},
		{3, map[string]string{"tech": "Super Stealth research stealing", "research_accumulated": "Super Stealth research stealing", "research_field": "Super Stealth research stealing", "research_next_field": "Super Stealth research stealing"}},
		// The first reason found is kept.
		{4, map[string]string{"tech": "production queue", "research_accumulated": "production queue", "research_field": "production queue", "research_next_field": "production queue"}},
		{5, map[string]string{"tech": "production queue", "research_accumulated": "production queue", "research_field": "production queue", "research_next_field": "production queue"}},
	} {
		got := l.playerUnsure(g, c.id)
		if len(got) != len(c.want) {
			t.Errorf("player %d: %v, want %v", c.id, got, c.want)
		}
		for _, k := range research {
			if got[k] != c.want[k] {
				t.Errorf("player %d %s: %q, want %q", c.id, k, got[k], c.want[k])
			}
		}
	}
}

func TestPlayerUnsureSuperStealthNarrow(t *testing.T) {
	// Stealing is modelled (engine stealResearch), so a Super Stealth
	// player's research is skipped only when another living player's
	// spending is not reproduced: a queue not loaded or planned orders.
	g := engine.Game{Players: []engine.Player{{}, {}, {Computer: true, Dead: true}}}
	g.Players[0].Race.PRT = engine.PRTSuperStealth
	l := pvLoaded{g: g, queued: map[int]bool{2: true}, ordered: map[int]bool{}}
	if got := l.playerUnsure(&g, 0); len(got) != 0 {
		t.Errorf("others reproduced (the planned one dead): %v, want nothing skipped", got)
	}
	const why = "Super Stealth research stealing"
	ss := map[string]string{"tech": why, "research_accumulated": why, "research_field": why, "research_next_field": why}
	g.Players[2].Dead = false
	l.g = g
	if got := l.playerUnsure(&g, 0); !reflect.DeepEqual(got, ss) {
		t.Errorf("a living planned player: %v, want %v", got, ss)
	}
	g.Players[2].Computer = false
	l = pvLoaded{g: g, queued: map[int]bool{1: true}, ordered: map[int]bool{}}
	if got := l.playerUnsure(&g, 0); !reflect.DeepEqual(got, ss) {
		t.Errorf("another player's queue not loaded: %v, want %v", got, ss)
	}
	l = pvLoaded{g: g, queued: map[int]bool{0: true}, ordered: map[int]bool{}}
	if got := l.playerUnsure(&g, 0); got["tech"] != "production queue" {
		t.Errorf("its own queue not loaded: %v, want production queue", got)
	}
}

// TestRunComparesYearsBeforeUnappliedOrders gives PQ-001-C01 a year-2
// order the harness cannot apply: its year-1 expectations are still
// compared (a wrong year-1 mines count fails the case), and each year-2
// expectation is skipped with the order's reason.
func TestRunComparesYearsBeforeUnappliedOrders(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	v.Orders = append(v.Orders, pvOrderBlock{Year: 2, Player: 0, Orders: []json.RawMessage{json.RawMessage(`{"kind": "no_such_kind"}`)}})
	r := Run(v, 0)[0]
	if r.Status != "pass" || !slices.Contains(r.Skipped, "order no_such_kind") {
		t.Errorf("status %s, why %q, skipped %q; want a pass with order no_such_kind skipped", r.Status, r.Why, r.Skipped)
	}
	e := &v.Cases[0].Expect[0]
	if e.Year != 1 {
		t.Fatalf("first expectation in year %d", e.Year)
	}
	eq := pvEq(t, string(e.Equals))
	eq["mines"] = json.RawMessage("-1")
	if e.Equals, err = json.Marshal(eq); err != nil {
		t.Fatal(err)
	}
	r = Run(v, 0)[0]
	if r.Status != "fail" || !strings.Contains(r.Why, "y1 planet: mines = ") {
		t.Errorf("status %s, why %q; want a fail on year-1 mines", r.Status, r.Why)
	}
}

// TestRunTalliesWhollySkippedCase gives PQ-001-C01 a year-1 order the
// harness cannot apply, so no expectation is compared: the case is a
// skip, and each of its expectations is still tallied with its reason.
func TestRunTalliesWhollySkippedCase(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	v.Orders = append(v.Orders, pvOrderBlock{Year: 1, Player: 0, Orders: []json.RawMessage{json.RawMessage(`{"kind": "no_such_kind"}`)}})
	r := Run(v, 0)[0]
	if r.Status != "skip" || len(r.Skipped) != len(v.Cases[0].Expect) {
		t.Fatalf("status %s, skipped %q; want a skip tallying all %d expectations", r.Status, r.Skipped, len(v.Cases[0].Expect))
	}
	for _, why := range r.Skipped {
		if why != "order no_such_kind" && !strings.HasPrefix(why, "stream ") {
			t.Errorf("skipped %q", why)
		}
	}
}

func TestLeftoverToResearch(t *testing.T) {
	l := pvLoaded{g: engine.Game{Planets: []engine.Planet{{ID: 3, Owner: 1, LeftoverOnly: true}}}, defaults: map[int]bool{}}
	g := &engine.Game{}
	p := engine.Planet{ID: 3, Owner: 1, LeftoverOnly: true}
	pvWant(t, "kept", l.planetEquals(g, &p, pvEq(t, `{"leftover_to_research": true}`), nil), "", "")
	pvWant(t, "differs", l.planetEquals(g, &p, pvEq(t, `{"leftover_to_research": false}`), nil), "leftover_to_research = true, want false", "")
	// A planet with a new owner takes the owner's default setting.
	p.Owner = 2
	pvWant(t, "new owner", l.planetEquals(g, &p, pvEq(t, `{"leftover_to_research": false}`), nil), "", "player default queue settings")
	l.defaults[2] = true
	pvWant(t, "new owner, defaults loaded", l.planetEquals(g, &p, pvEq(t, `{"leftover_to_research": false}`), nil), "leftover_to_research = true, want false", "")
}

func TestQueueOfNewOwner(t *testing.T) {
	planet := 3
	l := pvLoaded{g: engine.Game{Planets: []engine.Planet{{ID: 4, Owner: 1}}}, planet: map[int]int{3: 4},
		defaults: map[int]bool{}, ordered: map[int]bool{}, queued: map[int]bool{}, unsupported: map[[2]int]string{}}
	g := &engine.Game{Planets: []engine.Planet{{ID: 4, Owner: 2, HasQueue: true,
		Queue: []engine.QueueItem{{Kind: engine.ItemAutoMines, Count: 5}}}}}
	e := pvExpect{Kind: "production_queue", Planet: &planet, Equals: json.RawMessage(`[{"kind": 1, "id": 0, "count": 6}]`)}
	pvWant(t, "no defaults", l.queueEquals(g, e), "", "player default queue settings")
	l.defaults[2] = true
	if msg := l.queueEquals(g, e); msg == "" || strings.HasPrefix(msg, "skip: ") {
		t.Errorf("defaults loaded: %q, want the queue compared", msg)
	}
	e.Equals = json.RawMessage(`[{"kind": 1, "id": 0, "count": 5}]`)
	pvWant(t, "defaults loaded, match", l.queueEquals(g, e), "", "")
}

func TestScoreRecord(t *testing.T) {
	players := []engine.Player{{}, {}}
	rec := engine.ScoreRecord{Player: 1, Score: 120, Resources: 30, Planets: 4, Starbases: 1, Unarmed: 32001,
		Escorts: 3, Capital: 0, TechSum: 20, Rank: 2, Flags: 1 | 0x20 | 0x40<<engine.VictoryTech}
	l := pvLoaded{g: engine.Game{Players: players}, queued: map[int]bool{}, ordered: map[int]bool{},
		views: [][]engine.PlayerView{nil, {{Player: 0, Scores: []engine.ScoreRecord{rec}}, {Player: 1}}}}
	g := &engine.Game{Players: players}
	id, seen := 1, 0
	e := pvExpect{Year: 1, Kind: "player", ID: &id, SeenBy: &seen}
	const want = `{"rank": 2, "score": 120, "resources": 30, "planets": 4, "starbases": 1, "unarmed_ships": 16192,
		"escort_ships": 3, "capital_ships": 0, "tech_level_sum": 20, "victory_conditions_met": ["tech_level"]}`
	check := func(name, raw, mismatch, skip string) {
		t.Helper()
		pvWant(t, name, l.scoreRecordCheck(g, e, json.RawMessage(raw)), mismatch, skip)
	}
	check("match", want, "", "")
	check("score", strings.Replace(want, `"score": 120`, `"score": 121`, 1), "score_record.score = 120, want 121", "")
	check("tech sum", strings.Replace(want, `"tech_level_sum": 20`, `"tech_level_sum": 21`, 1), "score_record.tech_level_sum = 20, want 21", "")
	check("unpacked count", strings.Replace(want, `16192`, `32001`, 1), "score_record.unarmed_ships = 16192, want 32001", "")
	check("rank", strings.Replace(want, `"rank": 2`, `"rank": 1`, 1), "score_record.rank = 2, want 1", "")
	check("victory", strings.Replace(want, `["tech_level"]`, `["tech_level", "score"]`, 1),
		"score_record.victory_conditions_met = [tech_level], want [score tech_level]", "")
	// Only the seen_by player's scores count.
	seen = 1
	check("not seen", want, "score_record: not among player 1's scores", "")
	seen = 0
	// Another living player's unsure research skips rank and victory only.
	l.queued[0] = true
	check("other unsure", strings.Replace(want, `"rank": 2`, `"rank": 1`, 1), "", "score_record rank and victory: player 0's production queue")
	check("other unsure, score", strings.Replace(want, `"score": 120`, `"score": 121`, 1), "score_record.score = 120, want 121", "score_record rank and victory: player 0's production queue")
	// A dead player's research does not count.
	players[0].Dead = true
	check("dead unsure", strings.Replace(want, `"rank": 2`, `"rank": 1`, 1), "score_record.rank = 2, want 1", "")
	players[0].Dead = false
	l.queued[0], l.queued[1] = false, true
	check("own unsure", strings.Replace(want, `"score": 120`, `"score": 121`, 1), "", "score_record: production queue")
}

func TestPackCount(t *testing.T) {
	for n, w := range map[int]int{0: 0, 8191: 8191, 8192: 10240, 16000: 12192, 16383: 12287, 32766: 16383, 32768: 18432, 33000: 18446} {
		if got := pvPackCount(n); got != w {
			t.Errorf("pvPackCount(%d) = %d, want %d", n, got, w)
		}
	}
}

func TestDefaultQueueExpectation(t *testing.T) {
	var l pvLoaded
	g := &engine.Game{Players: []engine.Player{{DefaultQueue: []engine.QueueItem{{Kind: engine.ItemAutoMines, Count: 5}}, DefaultLeftoverOnly: true}}}
	pvWant(t, "match", l.playerEquals(g, 0, pvEq(t, `{"default_queue": [{"kind": 1, "id": 0, "count": 5}], "default_leftover_to_research": true}`), nil), "", "")
	pvWant(t, "queue", l.playerEquals(g, 0, pvEq(t, `{"default_queue": []}`), nil), "default_queue = [{4 5 0 0}], want []", "")
	pvWant(t, "leftover", l.playerEquals(g, 0, pvEq(t, `{"default_leftover_to_research": false}`), nil), "default_leftover_to_research = true, want false", "")
	pvWant(t, "design item", l.playerEquals(g, 0, pvEq(t, `{"default_queue": [{"kind": 2, "id": 0, "count": 1}]}`), nil), "", "default queue with a design item")
	g.Players[0].DefaultQueue = nil
	pvWant(t, "empty", l.playerEquals(g, 0, pvEq(t, `{"default_queue": []}`), nil), "", "")
}
