package vectors

import (
	"encoding/json"
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
// fleet the state does not have skips the vector with its reason.
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
	if _, why := l.orders(blocks, 1, &l.g); why != "order waypoint_change: no fleet" {
		t.Errorf("skip reason %q, want %q", why, "order waypoint_change: no fleet")
	}
}
