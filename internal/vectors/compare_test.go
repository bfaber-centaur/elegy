package vectors

import (
	"encoding/json"
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
	research := []string{"tech", "research_accumulated", "research_field"}
	for _, c := range []struct {
		id   int
		want map[string]string
	}{
		{0, map[string]string{"tech": "production queue", "research_accumulated": "production queue", "research_field": "production queue"}},
		{1, map[string]string{"tech": pvPlannedWhy, "research_accumulated": pvPlannedWhy, "research_field": pvPlannedWhy}},
		{2, map[string]string{}},
		{3, map[string]string{"tech": "Super Stealth research stealing", "research_accumulated": "Super Stealth research stealing"}},
		// The first reason found is kept.
		{4, map[string]string{"tech": "production queue", "research_accumulated": "production queue", "research_field": "production queue"}},
		{5, map[string]string{"tech": "production queue", "research_accumulated": "production queue", "research_field": "production queue"}},
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
