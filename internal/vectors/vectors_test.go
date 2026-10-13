package vectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

const testVector = "../../engine/testdata/vectors/pq/c01.json"

func TestReadFile(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "PQ-001-C01" || v.corpus != "pq" || v.path != "pq/c01.json" || len(v.Cases) != 1 {
		t.Fatalf("read %s: id %q, corpus %q, path %q, %d cases", testVector, v.ID, v.corpus, v.path, len(v.Cases))
	}
}

func TestReadDirNoVectors(t *testing.T) {
	if _, err := ReadDir(t.TempDir()); err == nil {
		t.Fatal("ReadDir of an empty directory: no error")
	}
}

// Compare gives every case the result Run gives it. A case fails, or
// differs under a legacy switch, exactly when one of its fields differs.
// A sample's mismatch is a sample, never a difference; an expectation of
// another oracle stream is skipped with the harness's reason.
func TestCompareAgreesWithRun(t *testing.T) {
	vs, err := ReadDir("../../engine/testdata/vectors")
	if err != nil {
		t.Fatal(err)
	}
	fields, samples, streams := 0, 0, 0
	for _, v := range vs {
		c := Compare(v, 0)
		v2, err := ReadFile(filepath.Join("../../engine/testdata/vectors", v.path))
		if err != nil {
			t.Fatal(err)
		}
		if want := Run(v2, 0); !reflect.DeepEqual(c.Cases, want) {
			t.Fatalf("%s: Compare's cases %+v, Run's %+v", v.path, c.Cases, want)
		}
		differ := map[string]bool{}
		for _, f := range c.Fields {
			switch {
			case f.Status == "differ" && f.Sample:
				t.Errorf("%s: a sample's field differs: %+v", v.path, f)
			case f.Status == "differ":
				differ[f.Case] = true
			case f.Status == "sample":
				samples++
			case f.Status == "skip" && strings.HasPrefix(f.Why, "stream "):
				streams++
			}
		}
		for _, r := range c.Cases {
			if failed := r.Status == "fail" || r.Status == "differs"; failed != differ[r.ID] {
				t.Errorf("%s: case %s %s, a field differs: %v", v.path, r.ID, r.Status, differ[r.ID])
			}
		}
		fields += len(c.Fields)
	}
	if fields == 0 || samples == 0 || streams == 0 {
		t.Fatalf("%d fields, %d missed samples, %d other-stream skips; want some of each", fields, samples, streams)
	}
}

func TestFleetIndex(t *testing.T) {
	g := &engine.Game{Fleets: []engine.Fleet{{ID: 4}, {ID: 9}, {ID: 2}}}
	for id, want := range map[int]int{4: 0, 9: 1, 2: 2, 7: -1} {
		if got := fleetIndex(g, id); got != want {
			t.Errorf("fleetIndex(%d) = %d, want %d", id, got, want)
		}
	}
}

// TestReadDirOrder checks that ReadDir returns the vectors in path order.
func TestReadDirOrder(t *testing.T) {
	src, err := os.ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, p := range []string{"b/1.json", "a/2.json", "a/10.json", "a/1.json"} {
		os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, p), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	vs, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range vs {
		got = append(got, v.path)
	}
	if want := []string{"a/1.json", "a/10.json", "a/2.json", "b/1.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ReadDir order %v, want %v", got, want)
	}
}

// TestOrdersWaypointNoFleet checks that a waypoint_change order naming a
// fleet Elegy does not have goes to the engine, which refuses it
// (TestCompareRejected), rather than skipping the vector.
func TestOrdersWaypointNoFleet(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	blocks := []pvOrderBlock{{Year: 1, Player: 0, Orders: []json.RawMessage{
		json.RawMessage(`{"kind": "waypoint_change", "fleet": 40, "index": 1, "waypoint": {"x": 0, "y": 0, "warp": 5, "target": {"kind": "planet", "id": 1}, "task": {"kind": "none"}}}`),
	}}}
	files, why := l.orders(blocks, 1, &l.g)
	if why != "" || len(files) != 1 || !reflect.DeepEqual(files[0].Orders, []engine.Order{engine.WaypointOrder{}}) {
		t.Errorf("orders %+v, skip %q; want an empty waypoint order for the engine to refuse", files, why)
	}
}

// TestFieldsYearNotGenerated checks an expectation for a year the run
// did not reach: reported when the engine stopped generation, skipped
// with the reason when the vector's orders did.
func TestFieldsYearNotGenerated(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	run := pvRun{}
	l, games, _, stop := generate(v, run)
	if stop != "" {
		t.Fatal(stop)
	}
	c := v.Cases[0]
	n := 0
	for _, e := range c.Expect {
		if e.Year != 1 || !run.checks(e) {
			continue
		}
		n++
		fs := l.fields(run, c, e, games[:1], errors.New("stopped"), "")
		if len(fs) != 1 || fs[0].Status != "differ" || fs[0].Why != "year 1 not generated: stopped" {
			t.Errorf("%s year 1, the engine stopped after year 0: %+v", e.Kind, fs)
		}
		fs = l.fields(run, c, e, games[:1], nil, "order x")
		if len(fs) != 1 || fs[0].Status != "skip" || fs[0].Why != "order x" {
			t.Errorf("%s year 1, an order stopped after year 0: %+v", e.Kind, fs)
		}
	}
	if n == 0 {
		t.Fatal("no year-1 expectation to check")
	}
}

// testBlock is one order set for player 0 in year 1, from JSON orders.
func testBlock(t *testing.T, orders ...string) []pvOrderBlock {
	t.Helper()
	b := pvOrderBlock{Year: 1, Player: 0}
	for _, o := range orders {
		b.Orders = append(b.Orders, json.RawMessage(o))
	}
	return []pvOrderBlock{b}
}

// TestOrdersWaypointsCumulative checks that a file's waypoint orders for
// one fleet apply on top of each other: add, add before, change, delete.
func TestOrdersWaypointsCumulative(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	wp := func(kind string, index, planet int) string {
		return fmt.Sprintf(`{"kind": %q, "fleet": 0, "index": %d, "waypoint": {"x": 0, "y": 0, "warp": 5, "target": {"kind": "planet", "id": %d}, "task": {"kind": "none"}}}`, kind, index, planet)
	}
	files, why := l.orders(testBlock(t,
		wp("waypoint_add", 1, 1),
		wp("waypoint_add", 2, 2),
		wp("waypoint_add", 1, 3), // before 1
		wp("waypoint_change", 3, 4),
		`{"kind": "waypoint_delete", "fleet": 0, "index": 1}`,
		wp("waypoint_change", 2, 5), // on the route the delete left
	), 1, &l.g)
	if why != "" {
		t.Fatal(why)
	}
	var got [][]int
	for _, o := range files[0].Orders {
		var ids []int
		for _, w := range o.(engine.WaypointOrder).Waypoints {
			ids = append(ids, w.ID)
		}
		got = append(got, ids)
	}
	p := l.planet
	want := [][]int{{p[1]}, {p[1], p[2]}, {p[3], p[1], p[2]}, {p[3], p[1], p[4]}, {p[1], p[4]}, {p[1], p[5]}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("waypoint ids after each order = %v, want %v", got, want)
	}
}

// TestFleetAt checks how a vector fleet is found: the initial fleet while
// it exists, else a fleet made during the run with its number, which may
// be a number the initial fleet left.
func TestFleetAt(t *testing.T) {
	l := &pvLoaded{fleetID: map[[2]int]int{pvFleetKey(1, 3): 7, pvFleetKey(1, 4): 8}}
	g := &engine.Game{Fleets: []engine.Fleet{{ID: 9, Owner: 1, Number: 4}, {ID: 7, Owner: 1, Number: 4}, {ID: 10, Owner: 1, Number: 6}, {ID: 8, Owner: 1, Number: 6}}}
	for _, c := range []struct{ id, want int }{{3, 7}, {5, 10}, {4, 8}, {6, 0}} {
		if got := l.fleetIn(g, 1, c.id); got != c.want {
			t.Errorf("with the initial fleets: fleet %d is %d, want %d", c.id, got, c.want)
		}
	}
	g.Fleets = g.Fleets[:1] // fleet 7 (number 4) is gone; 9 now has its number
	if got := l.fleetIn(g, 1, 3); got != 9 {
		t.Errorf("after fleet 7 left: fleet 3 is %d, want 9", got)
	}
	if got := l.fleetIn(g, 2, 3); got != 0 {
		t.Errorf("another owner's fleet 3 is %d, want 0", got)
	}
}

// TestCompareRejected checks that an order Elegy refuses is reported, and
// that a vector without one reports none.
func TestCompareRejected(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	if c := Compare(v, 0); len(c.Rejected) != 0 {
		t.Fatalf("no orders: rejected %v", c.Rejected)
	}
	v.Orders = testBlock(t, `{"kind": "rename", "fleet": 40, "name": "x"}`, `{"kind": "rename", "fleet": 0, "name": "y"}`)
	c := Compare(v, 0)
	if len(c.Rejected) != 1 || !strings.HasPrefix(c.Rejected[0], "year 1 player 0 order 0 (RenameOrder): ") {
		t.Errorf("rejected %q, want one for order 0", c.Rejected)
	}
}

// TestOrdersConversions checks the design, design_delete and
// planet_settings conversions (vectors/README.md "orders").
func TestOrdersConversions(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	files, why := l.orders(testBlock(t,
		`{"kind": "design", "starbase": false, "slot": 12, "hull": "Scout", "slots": [{"count": 1, "part": "Quick Jump 5"}, null, {"count": 2, "part": "Fuel Tank"}]}`,
		`{"kind": "design", "starbase": true, "slot": 3, "hull": "Orbital Fort", "slots": []}`,
		`{"kind": "design_delete", "starbase": true, "slot": 1}`,
		`{"kind": "planet_settings", "planet": 2, "leftover_to_research": true, "packet_destination": 3, "packet_warp": 8, "route_to": -1}`,
		`{"kind": "planet_settings", "planet": 2, "leftover_to_research": false, "packet_destination": -1, "packet_warp": 4, "route_to": 1}`,
		`{"kind": "planet_settings", "planet": 2, "leftover_to_research": false, "packet_destination": 0, "packet_warp": 5, "route_to": 0}`,
	), 1, &l.g)
	if why != "" {
		t.Fatal(why)
	}
	p := l.planet
	want := []engine.Order{
		engine.DesignOrder{Slot: 12, Name: "p0 d12", Hull: "Scout", Fills: []engine.SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}, {Slot: 2, Part: "Fuel Tank", Count: 2}}},
		engine.DesignOrder{Starbase: true, Slot: 3, Name: "p0 d3", Hull: "Orbital Fort"},
		engine.DeleteDesignOrder{Starbase: true, Slot: 1},
		engine.PlanetSettingsOrder{Planet: p[2], LeftoverOnly: true, HasPacketDest: true, PacketDest: p[3], PacketSpeed: 8},
		engine.PlanetSettingsOrder{Planet: p[2], HasRoute: true, RouteTo: p[1], PacketSpeed: 4},
		// Planet 0 is a planet: only -1 is none.
		engine.PlanetSettingsOrder{Planet: p[2], HasRoute: true, RouteTo: p[0], HasPacketDest: true, PacketDest: p[0], PacketSpeed: 5},
	}
	if !reflect.DeepEqual(files[0].Orders, want) {
		t.Errorf("orders = %+v\nwant %+v", files[0].Orders, want)
	}
}

// TestOrdersWaypointRefusals checks the waypoint orders the harness
// refuses (a waypoint added before the fleet's position, a delete of the
// position or past the end) and a delete for a fleet Elegy does not have,
// which goes to the engine to refuse.
func TestOrdersWaypointRefusals(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ order, why string }{
		{`{"kind": "waypoint_add", "fleet": 0, "index": 0, "waypoint": {"x": 0, "y": 0, "warp": 5, "target": {"kind": "planet", "id": 1}, "task": {"kind": "none"}}}`, "order waypoint_add before the fleet's position"},
		{`{"kind": "waypoint_delete", "fleet": 0, "index": 0}`, "order waypoint_delete index"},
	} {
		if _, why := l.orders(testBlock(t, c.order), 1, &l.g); why != c.why {
			t.Errorf("%s: why %q, want %q", c.order, why, c.why)
		}
	}
	// A delete past the end of the fleet's waypoints is refused and
	// recorded; the year's other orders still load.
	files, why := l.orders(testBlock(t, `{"kind": "waypoint_delete", "fleet": 0, "index": 1}`), 1, &l.g)
	if why != "" || len(files) != 1 || len(files[0].Orders) != 0 {
		t.Errorf("delete past the end: %+v, %q; want no order and no why", files, why)
	}
	if want := []string{"year 1 player 0 vector order 0 (waypoint_delete): waypoint 1 of fleet 0 past its 0 waypoints"}; !reflect.DeepEqual(l.rejected, want) {
		t.Errorf("delete past the end recorded %q, want %q", l.rejected, want)
	}
	l.rejected = nil
	files, why = l.orders(testBlock(t, `{"kind": "waypoint_delete", "fleet": 40, "index": 1}`), 1, &l.g)
	if why != "" || !reflect.DeepEqual(files[0].Orders, []engine.Order{engine.WaypointOrder{}}) {
		t.Errorf("delete for a missing fleet: %+v, %q; want an order for fleet 0", files, why)
	}
}

// TestOrdersWaypointTasks checks that a waypoint order keeps the fleet's
// own task and gives the waypoint its task.
func TestOrdersWaypointTasks(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	g := l.g
	g.Fleets = append([]engine.Fleet(nil), l.g.Fleets...)
	i := fleetIndex(&g, l.fleetIn(&g, 0, 0))
	g.Fleets[i].Task = engine.Task{Kind: engine.TaskColonize}
	files, why := l.orders(testBlock(t,
		`{"kind": "waypoint_add", "fleet": 0, "index": 1, "waypoint": {"x": 0, "y": 0, "warp": 5, "target": {"kind": "planet", "id": 1}, "task": {"kind": "route"}}}`,
	), 1, &g)
	if why != "" {
		t.Fatal(why)
	}
	w := files[0].Orders[0].(engine.WaypointOrder)
	if w.Task.Kind != engine.TaskColonize || len(w.Waypoints) != 1 || w.Waypoints[0].Task.Kind != engine.TaskRoute {
		t.Errorf("order %+v: want the fleet's colonize task and a route waypoint", w)
	}
}

// TestOrdersRunMade checks that orders find a fleet and a design made
// during the run: the order's own fleet, merge, cargo and move_ships
// partners, and move_ships designs.
func TestOrdersRunMade(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	g := l.g
	// Vector fleet 40 (number 41) and design slot 5, both made in the run.
	g.Fleets = append(append([]engine.Fleet(nil), l.g.Fleets...), engine.Fleet{ID: 99, Owner: 0, Number: 41})
	g.Designs = append(append([]engine.Design(nil), l.g.Designs...), engine.Design{Name: "made"})
	made := len(g.Designs) - 1
	g.DesignSlots = append(append([]engine.DesignSlot(nil), l.g.DesignSlots...), engine.DesignSlot{Owner: 0, Slot: 5, Design: made})
	zero := l.fleetIn(&g, 0, 0)
	files, why := l.orders(testBlock(t,
		`{"kind": "rename", "fleet": 40, "name": "x"}`,
		`{"kind": "merge", "fleet": 0, "fleets": [40]}`,
		`{"kind": "cargo", "fleet": 0, "with": {"kind": "fleet", "id": 40}, "amounts": {"ironium": 1}}`,
		`{"kind": "move_ships", "fleet": 0, "with": {"kind": "fleet", "id": 40}, "ships": [{"design": 5, "count": 1}]}`,
	), 1, &g)
	if why != "" {
		t.Fatal(why)
	}
	c := engine.CargoOrder{Fleet: zero, Target: engine.TargetFleet, ID: 99}
	c.Amounts[pvCargo()["ironium"]] = 1
	want := []engine.Order{
		engine.RenameOrder{Fleet: 99, Name: "x"},
		engine.MergeOrder{Into: zero, From: []int{99}},
		c,
		engine.MoveShipsOrder{Fleet: zero, With: 99, Ships: []engine.Stack{{Design: made, Count: 1}}},
	}
	if !reflect.DeepEqual(files[0].Orders, want) {
		t.Errorf("orders = %+v\nwant %+v", files[0].Orders, want)
	}
	f := engine.Fleet{Owner: 0, Stacks: []engine.Stack{{Design: made, Count: 1, Damage: engine.Damage{Pct: 50, Units: 10}}}}
	if msg := l.fleetEquals(&g, &f, pvEq(t, `{"damage": [{"design": 5, "units": 10, "percent_of_ships": 50}]}`)); msg != "" {
		t.Errorf("damage on a design made in the run: %s", msg)
	}
}

// TestCompareRejectedPlayer checks that a refused order is named by its
// own player's file when two players give orders.
func TestCompareRejectedPlayer(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/kx004/s1.json")
	if err != nil {
		t.Fatal(err)
	}
	v.Orders = []pvOrderBlock{
		{Year: 1, Player: 0, Orders: []json.RawMessage{json.RawMessage(`{"kind": "rename", "fleet": 40, "name": "x"}`)}},
		{Year: 1, Player: 1, Orders: []json.RawMessage{json.RawMessage(`{"kind": "battle_plan_delete", "slot": 3}`)}},
	}
	c := Compare(v, 0)
	if len(c.Rejected) == 0 || !strings.HasPrefix(c.Rejected[0], "year 1 player 0 order 0 (RenameOrder): ") {
		t.Errorf("rejected %q, want player 0's rename first", c.Rejected)
	}
}

// TestPlanetHomeworld: a state planet's homeworld flag is loaded, and a
// planet expectation's homeworld is compared (vectors README "planets").
func TestPlanetHomeworld(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/tk2/tk120.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := v.Start()
	if err != nil {
		t.Fatal(err)
	}
	var homes []engine.Point
	for _, p := range s.Game.Planets {
		if p.Homeworld {
			homes = append(homes, p.Pos)
		}
	}
	if want := []engine.Point{{X: 1169, Y: 1145}, {X: 1306, Y: 1060}}; !reflect.DeepEqual(homes, want) {
		t.Errorf("TK-120 homeworlds at %v, want %v", homes, want)
	}
	var l pvLoaded
	p := engine.Planet{Owner: 0}
	pvWant(t, "homeworld differs", l.planetEquals(nil, &p, pvEq(t, `{"homeworld": true}`), nil), "homeworld = false, want true", "")
	p.Homeworld = true
	pvWant(t, "homeworld equal", l.planetEquals(nil, &p, pvEq(t, `{"homeworld": true}`), nil), "", "")
}

// TestConcentrationFractions: a planet's stored depletion fractions load
// into its deposits (vectors README "planets", KERNEL.md "Mining").
func TestConcentrationFractions(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range l.g.Planets {
		var got [3]int
		for m := range got {
			got[m] = p.Deposits[m].Fraction
		}
		want := [3]int{}
		if p.ID == l.planet[7] {
			want = [3]int{242, 86, 157}
		}
		if got != want {
			t.Errorf("planet %d fractions %v, want %v", p.ID, got, want)
		}
	}
	// A stored 0 is a full 256, which Deposit.Fraction also writes as 0.
	for i := range v.InitialState.Planets {
		if v.InitialState.Planets[i].ID == 7 {
			v.InitialState.Planets[i].ConcFractions = []int{0, 128, 255}
		}
	}
	if l, err = loadVector(v); err != nil {
		t.Fatal(err)
	}
	d := l.g.Planets[l.planet[7]-1].Deposits
	if got := [3]int{d[0].Fraction, d[1].Fraction, d[2].Fraction}; got != [3]int{0, 128, 255} {
		t.Errorf("fractions [0 128 255] loaded as %v", got)
	}
}

// TestLoadMalformedStarbase: a state's starbase record without a design
// is no starbase only as design null with damage -1; any other, or a
// starbase at a planet nobody owns, refuses the state rather than
// loading a planet without a starbase.
func TestLoadMalformedStarbase(t *testing.T) {
	design := 0
	for _, c := range []struct {
		name    string
		planet  int
		design  *int
		damage  int
		refused bool
	}{
		{"none as written", 7, nil, -1, false},
		{"no design", 7, nil, 0, true},
		{"unowned planet", 0, &design, 0, true},
	} {
		v, err := ReadFile(testVector)
		if err != nil {
			t.Fatal(err)
		}
		for i := range v.InitialState.Planets {
			if p := &v.InitialState.Planets[i]; p.ID == c.planet {
				p.Starbase = &struct {
					Design *int `json:"design"`
					Damage int  `json:"damage"`
				}{c.design, c.damage}
			}
		}
		l, err := loadVector(v)
		if refused := err != nil; refused != c.refused {
			t.Errorf("%s: load error %v, want refused %v", c.name, err, c.refused)
			continue
		}
		if !c.refused && l.g.Planets[l.planet[c.planet]-1].HasStarbase {
			t.Errorf("%s: planet %d has a starbase", c.name, c.planet)
		}
	}
}

// TestLoadHomeworldRecord loads WU-RSG2's players' homeworld records
// (vectors/README.md "homeworld", stars-elegy ac792b0): player 1's names
// planet 8, which starts without the flag, and an absent field leaves no
// record.
func TestLoadHomeworldRecord(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/wu/wu-rsg2.json")
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]int{0: 17, 1: 8} {
		if pl := l.g.Players[id]; !pl.HasHomeworld || pl.Homeworld != l.planet[want] {
			t.Errorf("player %d record %v %d, want planet %d (Elegy %d)", id, pl.HasHomeworld, pl.Homeworld, want, l.planet[want])
		}
	}
	v.InitialState.Players[1].Homeworld = nil
	if l, err = loadVector(v); err != nil {
		t.Fatal(err)
	}
	if pl := l.g.Players[1]; pl.HasHomeworld || pl.Homeworld != 0 {
		t.Errorf("absent field: record %v %d, want none", pl.HasHomeworld, pl.Homeworld)
	}
}

// TestMisslottedPartDropped: CB-009's designs with a part in a slot that
// does not take it keep the part in the year that reads them and lose it
// at the end of that year (COMBAT.md "Parts the game drops from a
// design").
func TestMisslottedPartDropped(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/cb/cb009.json")
	if err != nil {
		t.Fatal(err)
	}
	l, games, genErr, stop := generate(v, pvRun{})
	if genErr != nil || stop != "" || len(games) < 2 {
		t.Fatalf("generate: %v %q, %d games", genErr, stop, len(games))
	}
	// Player 1's slot 1: a Small Freighter with a Beam Deflector in its
	// shield-or-armor slot.
	d := l.design[[2]int{1, 1}]
	if n := len(games[0].Designs[d].Slots); n != 2 {
		t.Errorf("at the start: %d parts, want the engine and the deflector", n)
	}
	if n := len(games[1].Designs[d].Slots); n != 1 {
		t.Errorf("after the year: %d parts, want the engine only", n)
	}
}

// TestRGNoRejectedOrders checks that the engine accepts every order of
// RG-001, chained and restarted: an order Elegy refuses there means a
// state that has drifted from the original's (before #147, player 6's
// fleet 0 in 2433 and 2435) or an order kind the loader converts wrongly.
// The restarts and the chained run's first year are strict on every seed
// variant, since a refusal can hang on the year's draws (the player
// shuffle decides which player's orders replay first). A later
// chained year follows Elegy's own random draws in the years before it
// (mining, artifacts, battles), so a refusal there is a sample, logged at
// -v and never a failure, as lane D's chained battle records are
// (docs/RANDOM-PARITY.md).
func TestRGNoRejectedOrders(t *testing.T) {
	names := []string{"rg001", "rg001-y1", "rg001-y2", "rg001-y3", "rg001-y4", "rg001-y5", "rg001-y6"}
	if checked := checkRGRejections(t, names); checked != len(names)*Seeds {
		t.Errorf("checked %d runs, want every variant of the %d vectors (%d)", checked, len(names), len(names)*Seeds)
	}
}

// checkRGRejections runs checkRejected on every seed variant of the named
// rg vectors and returns how many runs it checked.
func checkRGRejections(t testing.TB, names []string) int {
	checked := 0
	for _, name := range names {
		v, err := ReadFile("../../engine/testdata/vectors/rg/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		for k, rejected := range variantRejections(v) {
			checkRejected(t, fmt.Sprintf("%s variant %d", name, k), rejected)
			checked++
		}
	}
	return checked
}

// TestCheckRGRejections checks checkRGRejections on every variant of
// RG-002 Y1. Its one variant-only refusal (player 2's waypoint-0 follow,
// refused on variants 1 and 6) cleared with the waypoint-0 follow change
// (ORDERS.md "Waypoint 0, following"), so no run errs;
// TestRejectedSample pins that a strict refusal fails.
func TestCheckRGRejections(t *testing.T) {
	e := &errorCounter{TB: t}
	if checked := checkRGRejections(e, []string{"rg002-y1"}); checked != Seeds || e.errors != 0 {
		t.Errorf("checkRGRejections(rg002-y1): %d runs, %d errors; want %d runs, none", checked, e.errors, Seeds)
	}
}

// variantRejections returns each seed variant's refused orders.
func variantRejections(v *Vector) [][]string {
	var out [][]string
	for k := 0; k < Seeds; k++ {
		out = append(out, Compare(v, k).Rejected)
	}
	return out
}

// TestVariantRejections checks variantRejections on RG-002 Y1, whose
// player 2's order 10 was refused on variants 1 and 6 only, where the
// player shuffle replays player 6's move_ships first. Since the
// waypoint-0 follow change no variant refuses it.
func TestVariantRejections(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg002-y1.json")
	if err != nil {
		t.Fatal(err)
	}
	var got []int
	for k, rejected := range variantRejections(v) {
		if len(rejected) != 0 {
			got = append(got, k)
		}
	}
	if len(got) != 0 {
		t.Errorf("variants refusing an RG-002 Y1 order: %v, want none", got)
	}
}

// checkRejected fails on each strict refusal and logs the samples
// (rejectedSample).
func checkRejected(t testing.TB, name string, rejected []string) {
	var strict, samples []string
	for _, r := range rejected {
		if rejectedSample(r) {
			samples = append(samples, r)
		} else {
			strict = append(strict, r)
		}
	}
	if len(strict) != 0 {
		t.Errorf("%s: rejected %q", name, strict)
	}
	if len(samples) != 0 {
		t.Logf("%s: %d sample rejections after Elegy's own draws: %q", name, len(samples), samples)
	}
}

// rejectedSample reports whether a refused order (Comparison.Rejected) is
// a sample: one from a year after the vector's first, whose state follows
// Elegy's random draws in the earlier years. A restart has only year 1.
func rejectedSample(rejected string) bool {
	var y int
	if _, err := fmt.Sscanf(rejected, "year %d ", &y); err != nil {
		return false
	}
	return y > 1
}

// errorCounter counts a test's errors without failing it.
type errorCounter struct {
	testing.TB
	errors int
}

func (e *errorCounter) Errorf(string, ...any) { e.errors++ }
func (e *errorCounter) Logf(string, ...any)   {}

// TestRejectedSample pins which refusals TestRGNoRejectedOrders lets
// through as samples, and that a strict one fails it.
func TestRejectedSample(t *testing.T) {
	for _, c := range []struct {
		rejected []string
		errors   int
	}{
		{[]string{"year 1 player 0 order 0 (CargoOrder): x"}, 1},
		{[]string{"year 2 player 0 order 0 (CargoOrder): x"}, 0},
		{nil, 0},
	} {
		e := &errorCounter{TB: t}
		if checkRejected(e, "v", c.rejected); e.errors != c.errors {
			t.Errorf("checkRejected(%q): %d errors, want %d", c.rejected, e.errors, c.errors)
		}
	}
	for _, c := range []struct {
		rejected string
		want     bool
	}{
		{"year 3 player 5 order 10 (FleetPlanOrder): fleet 0: orders: no such object", true},
		{"year 2 player 0 order 0 (CargoOrder): x", true},
		{"year 1 player 5 order 10 (FleetPlanOrder): fleet 0: orders: no such object", false},
		{"player 5 order 10: no year", false},
	} {
		if got := rejectedSample(c.rejected); got != c.want {
			t.Errorf("rejectedSample(%q) = %v, want %v", c.rejected, got, c.want)
		}
	}
}

// TestSalvageFresh loads salvage's stored skip-decay mark (vectors/README.md
// "objects", stars-elegy d86c4f8): RG-001 Y6's battle salvage from 2434 is
// marked and survives 2435's decay unchanged, MF-13B's is not, and a record
// without the field makes the vector unsupported rather than guessed.
func TestSalvageFresh(t *testing.T) {
	for _, c := range []struct {
		path  string
		pos   engine.Point
		fresh bool
	}{
		{"rg/rg001-y6.json", engine.Point{X: 1324, Y: 1257}, true},
		{"mf/mf13b.json", engine.Point{}, false},
	} {
		v, err := ReadFile("../../engine/testdata/vectors/" + c.path)
		if err != nil {
			t.Fatal(err)
		}
		l, err := loadVector(v)
		if err != nil {
			t.Fatal(err)
		}
		if len(l.g.Salvage) != 1 {
			t.Fatalf("%s: %d salvage objects, want 1", c.path, len(l.g.Salvage))
		}
		s := l.g.Salvage[0]
		if s.Fresh != c.fresh || c.fresh && s.Pos != c.pos {
			t.Errorf("%s: salvage at %v fresh %v, want %v", c.path, s.Pos, s.Fresh, c.fresh)
		}
		if !c.fresh {
			continue
		}
		_, games, genErr, stop := generate(v, pvRun{})
		if genErr != nil || stop != "" || len(games) < 2 {
			t.Fatalf("generate: %v %q, %d games", genErr, stop, len(games))
		}
		kept := false
		for _, s := range games[1].Salvage {
			kept = kept || s.Pos == c.pos && s.Minerals == engine.Minerals{3, 3, 3}
		}
		if !kept {
			t.Errorf("%s: the marked salvage did not keep [3 3 3] through the decay: %v", c.path, games[1].Salvage)
		}
	}
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001-y6.json")
	if err != nil {
		t.Fatal(err)
	}
	for i, raw := range v.InitialState.Objects {
		var o map[string]any
		if json.Unmarshal(raw, &o) == nil && o["kind"] == "salvage" {
			delete(o, "fresh")
			v.InitialState.Objects[i], _ = json.Marshal(o)
		}
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(l.global, "salvage 1/0 without fresh") {
		t.Errorf("salvage without fresh: global %q, want the vector refused", l.global)
	}
}

func TestVictoryConditions(t *testing.T) {
	load := func(path string, edit func(*Vector)) (*pvLoaded, error) {
		t.Helper()
		v, err := ReadFile("../../engine/testdata/vectors/" + path)
		if err != nil {
			t.Fatal(err)
		}
		if edit != nil {
			edit(v)
		}
		return loadVector(v)
	}
	// KX-003-r1: planets 20%, tech level 9 in 3 fields and lead 20% on;
	// score 1000, resources 10, capital ships 10 and highest after 30
	// years off; 1 needed, 30 years minimum.
	l, err := load("kx003/r1.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := engine.Victory{TechLevel: 1, TechFields: 1, Needed: 1,
		Enabled: [engine.NumVictory]bool{engine.VictoryPlanets: true, engine.VictoryTech: true, engine.VictoryLead: true}}
	if l.g.Victory != want || l.g.PublicScores {
		t.Errorf("KX-003-r1: victory %+v public %v, want %+v false", l.g.Victory, l.g.PublicScores, want)
	}
	// KX-004-S1: the new-game defaults, with public scores.
	if l, err = load("kx004/s1.json", nil); err != nil {
		t.Fatal(err)
	}
	want = engine.Victory{Planets: 8, TechLevel: 14, TechFields: 2, Score: 10, Lead: 8, Resources: 9,
		Capital: 9, Highest: 7, Needed: 1,
		Enabled: [engine.NumVictory]bool{engine.VictoryPlanets: true, engine.VictoryTech: true, engine.VictoryLead: true}}
	if l.g.Victory != want || !l.g.PublicScores {
		t.Errorf("KX-004-S1: victory %+v public %v, want %+v true", l.g.Victory, l.g.PublicScores, want)
	}
	if l, err = load("kx004/s1.json", func(v *Vector) { v.InitialState.Game.Victory = nil }); err != nil {
		t.Fatal(err)
	}
	if l.g.Victory != (engine.Victory{}) {
		t.Errorf("no victory conditions: %+v, want none", l.g.Victory)
	}
	for _, c := range []struct {
		name  string
		value int
		drop  bool // drop enabled
		want  string
	}{
		{"planets_owned_percent", 62, false, "victory condition planets_owned_percent"},
		{"planets_owned_percent", 15, false, "victory condition planets_owned_percent"},
		{"minimum_years", 25, false, "victory condition minimum_years"},
		{"score", 11000, true, "victory condition score without enabled"},
		{"conditions_needed", -1, false, "victory condition conditions_needed"},
	} {
		_, err := load("kx004/s1.json", func(v *Vector) {
			m := maps.Clone(v.InitialState.Game.Victory)
			x := m[c.name]
			x.Value = c.value
			if c.drop {
				x.Enabled = nil
			}
			if c.value < 0 {
				delete(m, c.name)
			} else {
				m[c.name] = x
			}
			v.InitialState.Game.Victory = m
		})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %d: %v, want %q", c.name, c.value, err, c.want)
		}
	}
}

func TestDefaultQueue(t *testing.T) {
	load := func(edit func(*pvPlayer)) (*pvLoaded, error) {
		t.Helper()
		v, err := ReadFile("../../engine/testdata/vectors/kx004/s1.json")
		if err != nil {
			t.Fatal(err)
		}
		// Start from players without default queue settings (the vector
		// carries them since stars-elegy's full rebuild).
		for i := range v.InitialState.Players {
			v.InitialState.Players[i].DefaultQueue = nil
			v.InitialState.Players[i].DefaultLeftoverToResearch = nil
		}
		edit(&v.InitialState.Players[0])
		return loadVector(v)
	}
	raw := func(s ...string) *[]json.RawMessage {
		var q []json.RawMessage
		for _, x := range s {
			q = append(q, json.RawMessage(x))
		}
		return &q
	}
	yes := true
	l, err := load(func(p *pvPlayer) {
		p.DefaultQueue = raw(`{"kind": 1, "id": 0, "count": 5}`, `{"kind": 1, "id": 1, "count": 10}`)
		p.DefaultLeftoverToResearch = &yes
	})
	if err != nil {
		t.Fatal(err)
	}
	pl := l.g.Players[0]
	want := []engine.QueueItem{{Kind: engine.ItemAutoMines, Count: 5}, {Kind: engine.ItemAutoFactories, Count: 10}}
	if !reflect.DeepEqual(pl.DefaultQueue, want) || !pl.DefaultLeftoverOnly || !l.defaults[0] || l.defaults[1] {
		t.Errorf("default queue %+v leftover %v, defaults %v; want %+v true, {0}", pl.DefaultQueue, pl.DefaultLeftoverOnly, l.defaults, want)
	}
	// An empty default queue is carried too.
	if l, err = load(func(p *pvPlayer) { p.DefaultQueue, p.DefaultLeftoverToResearch = raw(), new(bool) }); err != nil || !l.defaults[0] {
		t.Errorf("empty default queue: %v, defaults %v", err, l.defaults)
	}
	// An item Elegy does not build leaves the defaults unloaded.
	if l, err = load(func(p *pvPlayer) {
		p.DefaultQueue, p.DefaultLeftoverToResearch = raw(`{"id": 0, "count": 5}`), &yes
	}); err != nil || l.defaults[0] || l.g.Players[0].DefaultQueue != nil || l.g.Players[0].DefaultLeftoverOnly {
		t.Errorf("unbuilt item: %v, defaults %v, player %+v", err, l.defaults, l.g.Players[0].DefaultQueue)
	}
	// A design item has no place in a default queue (vectors/README.md
	// "players": planetary items only).
	if l, err = load(func(p *pvPlayer) {
		p.DefaultQueue, p.DefaultLeftoverToResearch = raw(`{"kind": 2, "id": 0, "count": 1}`), &yes
	}); err != nil || l.defaults[0] || l.g.Players[0].DefaultQueue != nil {
		t.Errorf("design item: %v, defaults %v, player %+v", err, l.defaults, l.g.Players[0].DefaultQueue)
	}
	for _, edit := range []func(*pvPlayer){
		func(p *pvPlayer) { p.DefaultQueue = raw() },
		func(p *pvPlayer) { p.DefaultLeftoverToResearch = &yes },
	} {
		if _, err := load(edit); err == nil || !strings.Contains(err.Error(), "go together") {
			t.Errorf("one of the pair: %v", err)
		}
	}
}

// TestOrderCargoSalvage: a cargo order's salvage partner is named by its
// owner and number (vectors/README.md "cargo").
func TestOrderCargoSalvage(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	g := l.g
	files, why := l.orders(testBlock(t,
		`{"kind": "cargo", "fleet": 0, "with": {"kind": "salvage", "owner": 2, "id": 1}, "amounts": {"ironium": 30}}`,
	), 1, &g)
	if why != "" {
		t.Fatal(why)
	}
	c := engine.CargoOrder{Fleet: l.fleetIn(&g, 0, 0), Target: engine.TargetSalvage, ID: 1, Owner: 2}
	c.Amounts[pvCargo()["ironium"]] = 30
	if !reflect.DeepEqual(files[0].Orders, []engine.Order{c}) {
		t.Errorf("orders = %+v, want %+v", files[0].Orders, c)
	}
	_, why = l.orders(testBlock(t,
		`{"kind": "cargo", "fleet": 0, "with": {"kind": "salvage", "id": 1}, "amounts": {"ironium": 30}}`,
	), 1, &g)
	if why != "order cargo with salvage without an owner" {
		t.Errorf("salvage without an owner: %q", why)
	}
}

// TestLoadArtifacts loads RG-001-Y6's start-state artifacts (vectors/
// README.md "artifact", stars-elegy 7c06f47): planet 123, colonized by
// player 1 in 2435, holds one; planet 114, found in 2434, does not.
func TestLoadArtifacts(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001-y6.json")
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, p := range l.g.Planets {
		if p.Artifact {
			n++
		}
		if want := p.ID == l.planet[123]; p.ID == l.planet[123] || p.ID == l.planet[114] {
			if p.Artifact != want {
				t.Errorf("planet %d artifact %v, want %v", p.ID, p.Artifact, want)
			}
		}
	}
	if n != 9 {
		t.Errorf("%d artifacts, want 9", n)
	}
}

// TestOrderDefaultQueue: the default-queue record loads as a
// DefaultQueueOrder (stars-elegy ORDERS.md "The player's default queue").
func TestOrderDefaultQueue(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	g := l.g
	files, why := l.orders(testBlock(t,
		`{"kind": "default_queue", "items": [], "leftover_to_research": false}`,
		`{"kind": "default_queue", "items": [{"kind": 1, "id": 1, "count": 10}], "leftover_to_research": true}`,
	), 1, &g)
	if why != "" {
		t.Fatal(why)
	}
	want := []engine.Order{
		engine.DefaultQueueOrder{},
		engine.DefaultQueueOrder{Queue: []engine.QueueItem{{Kind: engine.ItemAutoFactories, Count: 10}}, LeftoverOnly: true},
	}
	if !reflect.DeepEqual(files[0].Orders, want) {
		t.Errorf("orders = %+v, want %+v", files[0].Orders, want)
	}
	_, why = l.orders(testBlock(t, `{"kind": "default_queue", "items": [{"kind": 2, "id": 0, "count": 1}]}`), 1, &g)
	if why != "order default queue with a design item" {
		t.Errorf("design item: %q", why)
	}
}

// TestOrderFollow: a waypoint-0 change aimed at another fleet loads as a
// FollowOrder, and a later waypoint order for the fleet starts from that
// one waypoint (stars-elegy ORDERS.md "Waypoint 0 aimed at a fleet"); a
// waypoint-0 change aimed at a planet still changes only the task.
func TestOrderFollow(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	g := l.g
	files, why := l.orders(testBlock(t,
		`{"kind": "waypoint_add", "fleet": 0, "index": 1, "waypoint": {"x": 0, "y": 0, "warp": 5, "target": {"kind": "planet", "id": 2}, "task": {"kind": "none"}}}`,
		`{"kind": "waypoint_change", "fleet": 0, "index": 0, "waypoint": {"x": 0, "y": 0, "warp": 6, "target": {"kind": "fleet", "owner": 0, "id": 1}, "task": {"kind": "colonize"}}}`,
		`{"kind": "waypoint_add", "fleet": 0, "index": 1, "waypoint": {"x": 0, "y": 0, "warp": 5, "target": {"kind": "planet", "id": 1}, "task": {"kind": "none"}}}`,
		`{"kind": "waypoint_change", "fleet": 2, "index": 0, "waypoint": {"x": 0, "y": 0, "warp": 6, "target": {"kind": "planet", "id": 1}, "task": {"kind": "colonize"}}}`,
	), 1, &g)
	if why != "" {
		t.Fatal(why)
	}
	f0 := l.fleetIn(&g, 0, 0)
	want := engine.FollowOrder{Fleet: f0, Leader: pvElegyFleetID(0, 1), Task: engine.Task{Kind: engine.TaskColonize}}
	if len(files[0].Orders) != 4 || !reflect.DeepEqual(files[0].Orders[1], want) {
		t.Fatalf("orders = %+v, want %+v second", files[0].Orders, want)
	}
	if w := files[0].Orders[2].(engine.WaypointOrder); len(w.Waypoints) != 1 || w.Waypoints[0].ID != l.planet[1] {
		t.Errorf("waypoint after the follow = %+v, want one waypoint at planet 1", w.Waypoints)
	}
	if w := files[0].Orders[2].(engine.WaypointOrder); w.Task.Kind != engine.TaskColonize {
		t.Errorf("waypoint after the follow: task %+v, want the follow's colonize", w.Task)
	}
	if w := files[0].Orders[3].(engine.WaypointOrder); w.Task.Kind != engine.TaskColonize {
		t.Errorf("waypoint 0 at a planet: %+v, want only the task changed", w)
	}
	g = l.g
	files, why = l.orders(testBlock(t,
		`{"kind": "waypoint_change", "fleet": 0, "index": 1, "waypoint": {"x": 0, "y": 0, "warp": 6, "target": {"kind": "fleet", "owner": 0, "id": 1}, "task": {"kind": "none"}}}`,
	), 1, &g)
	if why != "" {
		t.Fatal(why)
	}
	if _, ok := files[0].Orders[0].(engine.WaypointOrder); !ok {
		t.Errorf("waypoint 1 aimed at a fleet = %+v, want a waypoint order, not a follow", files[0].Orders[0])
	}
	for _, c := range []struct{ order, why string }{
		{`{"kind": "waypoint_add", "fleet": 0, "index": 0, "waypoint": {"x": 0, "y": 0, "warp": 6, "target": {"kind": "fleet", "owner": 0, "id": 1}, "task": {"kind": "none"}}}`, "order waypoint_add before the fleet's position"},
		{`{"kind": "waypoint_change", "fleet": 0, "index": 0, "waypoint": {"x": 0, "y": 0, "warp": 12, "target": {"kind": "fleet", "owner": 0, "id": 1}, "task": {"kind": "none"}}}`, "order waypoint_change warp 12"},
	} {
		g = l.g
		if _, why := l.orders(testBlock(t, c.order), 1, &g); why != c.why {
			t.Errorf("%s: %q, want %q", c.order, why, c.why)
		}
	}
}

// TestWaypointPacketObject: a packet_object waypoint target names the
// object by owner and number (vectors/README.md "fleets").
func TestWaypointPacketObject(t *testing.T) {
	l := &pvLoaded{}
	var w pvWaypoint
	if err := json.Unmarshal([]byte(`{"x": 10, "y": 20, "warp": 6, "target": {"kind": "packet_object", "owner": 3, "id": 7}}`), &w); err != nil {
		t.Fatal(err)
	}
	wp, why := l.waypoint(0, w)
	if want := (engine.Waypoint{Pos: engine.Point{X: 10, Y: 20}, Warp: 6, Target: engine.TargetPacketObject, Owner: 3, ID: 7}); why != "" || wp != want {
		t.Errorf("waypoint = %+v %q, want %+v", wp, why, want)
	}
	w.Target.Owner = nil
	if _, why := l.waypoint(0, w); why != "packet_object target without an owner and id" {
		t.Errorf("without an owner: %q", why)
	}
	w.Target.Owner, w.Target.ID = w.Target.ID, nil
	if _, why := l.waypoint(0, w); why != "packet_object target without an owner and id" {
		t.Errorf("without an id: %q", why)
	}
}

// TestLoadTarget0 checks waypoint 0's packet or salvage target
// (engine.Fleet.Target0): a start's waypoint 0 sets it, and a
// waypoint_change at index 0 replaces it, a planet target clearing it.
func TestLoadTarget0(t *testing.T) {
	v, err := ReadFile(testVector)
	if err != nil {
		t.Fatal(err)
	}
	owner, id := 1, 4
	w := &v.InitialState.Fleets[0].Waypoints[0]
	w.Target.Kind, w.Target.Owner, w.Target.ID = "packet_object", &owner, &id
	l, err := loadVector(v)
	if err != nil {
		t.Fatal(err)
	}
	fl := v.InitialState.Fleets[0]
	f := l.fleetAt(&l.g, fl.Owner, fl.ID)
	if f == nil || f.Target0 == nil || *f.Target0 != (engine.ObjectRef{Owner: 1, Number: 4}) {
		t.Errorf("start: target %v, want packet 1/4", f.Target0)
	}
	for _, c := range []struct {
		target string
		want   *engine.ObjectRef
	}{
		{`{"kind": "packet_object", "owner": 0, "id": 2}`, &engine.ObjectRef{Owner: 0, Number: 2}},
		{`{"kind": "planet", "id": 7}`, nil},
	} {
		blocks := []pvOrderBlock{{Year: 1, Player: fl.Owner, Orders: []json.RawMessage{
			json.RawMessage(fmt.Sprintf(`{"kind": "waypoint_change", "fleet": %d, "index": 0, "waypoint": {"x": %d, "y": %d, "warp": 0, "target": %s, "task": {"kind": "none"}}}`, fl.ID, fl.X, fl.Y, c.target)),
		}}}
		files, why := l.orders(blocks, 1, &l.g)
		if why != "" || len(files) != 1 || len(files[0].Orders) != 1 {
			t.Fatalf("%s: orders %+v, skip %q", c.target, files, why)
		}
		o, ok := files[0].Orders[0].(engine.WaypointOrder)
		if !ok || !o.Retarget0 || (o.Target0 == nil) != (c.want == nil) || o.Target0 != nil && *o.Target0 != *c.want {
			t.Errorf("%s: order %+v, want Retarget0 with %v", c.target, files[0].Orders[0], c.want)
		}
	}
}

// TestLoadGalaxySize checks that a vector's universe size is required:
// the galaxy clamp reads it, so a missing size is a load error rather than
// tiny, and bounds that are not the size's box are an error too.
func TestLoadGalaxySize(t *testing.T) {
	for _, c := range []struct {
		name   string
		size   string
		bounds []int
		want   int // -1: refused
	}{
		{"tiny with its box", "tiny", []int{1000, 1000, 1400, 1400}, 0},
		{"medium with its box", "medium", []int{1000, 1000, 2200, 2200}, 2},
		{"huge, no bounds", "huge", nil, 4},
		{"missing size", "", nil, -1},
		{"unknown size", "enormous", nil, -1},
		{"bounds of another size", "small", []int{1000, 1000, 1400, 1400}, -1},
		{"bounds not starting at 1000", "tiny", []int{0, 0, 400, 400}, -1},
	} {
		v, err := ReadFile(testVector)
		if err != nil {
			t.Fatal(err)
		}
		v.InitialState.Game.Size, v.InitialState.Game.Bounds = c.size, c.bounds
		l, err := loadVector(v)
		if refused := err != nil; refused != (c.want < 0) {
			t.Errorf("%s: load error %v, want refused %v", c.name, err, c.want < 0)
			continue
		}
		if c.want >= 0 && l.g.Size != c.want {
			t.Errorf("%s: size %d, want %d", c.name, l.g.Size, c.want)
		}
	}

	// A new-game vector has no starting state to size; one that has a
	// state still needs its size.
	for _, players := range []bool{false, true} {
		v, err := ReadFile(testVector)
		if err != nil {
			t.Fatal(err)
		}
		v.NewGame = json.RawMessage(`{}`)
		v.InitialState.Game.Size = ""
		if !players {
			v.InitialState = pvState{}
		}
		if _, err := loadVector(v); (err != nil) != players {
			t.Errorf("new-game vector, players in its state %v: load error %v", players, err)
		}
	}
}
