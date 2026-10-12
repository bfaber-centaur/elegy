package vectors

import (
	"encoding/json"
	"errors"
	"fmt"
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
		{`{"kind": "waypoint_delete", "fleet": 0, "index": 1}`, "order waypoint_delete index"},
	} {
		if _, why := l.orders(testBlock(t, c.order), 1, &l.g); why != c.why {
			t.Errorf("%s: why %q, want %q", c.order, why, c.why)
		}
	}
	files, why := l.orders(testBlock(t, `{"kind": "waypoint_delete", "fleet": 40, "index": 1}`), 1, &l.g)
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
