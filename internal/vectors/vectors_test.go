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
	want := [][]int{{p[1]}, {p[1], p[2]}, {p[3], p[1], p[2]}, {p[3], p[1], p[4]}, {p[1], p[4]}}
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
		`{"kind": "design_delete", "starbase": true, "slot": 1}`,
		`{"kind": "planet_settings", "planet": 2, "leftover_to_research": true, "packet_destination": 3, "packet_warp": 8, "route_to": -1}`,
		`{"kind": "planet_settings", "planet": 2, "leftover_to_research": false, "packet_destination": -1, "packet_warp": 4, "route_to": 1}`,
	), 1, &l.g)
	if why != "" {
		t.Fatal(why)
	}
	p := l.planet
	want := []engine.Order{
		engine.DesignOrder{Slot: 12, Name: "p0 d12", Hull: "Scout", Fills: []engine.SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}, {Slot: 2, Part: "Fuel Tank", Count: 2}}},
		engine.DeleteDesignOrder{Starbase: true, Slot: 1},
		engine.PlanetSettingsOrder{Planet: p[2], LeftoverOnly: true, HasPacketDest: true, PacketDest: p[3], PacketSpeed: 8},
		engine.PlanetSettingsOrder{Planet: p[2], HasRoute: true, RouteTo: p[1], PacketSpeed: 4},
	}
	if !reflect.DeepEqual(files[0].Orders, want) {
		t.Errorf("orders = %+v\nwant %+v", files[0].Orders, want)
	}
}
