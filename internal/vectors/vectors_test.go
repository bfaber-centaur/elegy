package vectors

import (
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

func TestFleetIndex(t *testing.T) {
	g := &engine.Game{Fleets: []engine.Fleet{{ID: 4}, {ID: 9}, {ID: 2}}}
	for id, want := range map[int]int{4: 0, 9: 1, 2: 2, 7: -1} {
		if got := fleetIndex(g, id); got != want {
			t.Errorf("fleetIndex(%d) = %d, want %d", id, got, want)
		}
	}
}
