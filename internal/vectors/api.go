// Package vectors loads stars-elegy parity vectors (the format is
// FORMAT.md beside the vectors, engine/testdata/vectors), builds each
// vector's start as an engine.Game, replays its orders through
// engine.GenerateTurn year by year and compares every case's
// expectations with the generated years.
//
// The engine's parity test (TestParityVectors) and the elegy vector
// command both run vectors through this package. How a run draws its
// random numbers, Elegy's own seed variants or a replay of the oracle's
// stream, is pvRun (streams.go).
package vectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// ReadFile reads one vector. Its corpus is the name of the directory
// holding it.
func ReadFile(path string) (*Vector, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v Vector
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	v.corpus = filepath.Base(filepath.Dir(path))
	v.path = v.corpus + "/" + filepath.Base(path)
	return &v, nil
}

// ReadDir reads every vector in dir's corpus directories (dir/*/*.json),
// in path order. It is an error to find none.
func ReadDir(dir string) ([]*Vector, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*", "*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no vectors in %s", dir)
	}
	sort.Strings(paths)
	var out []*Vector
	for _, p := range paths {
		v, err := ReadFile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// Run loads v, generates its years with Elegy's own seeding for the seed
// variant (0 is the reference) and returns one result per case, in case
// order.
func Run(v *Vector, variant int) []Result {
	return runVector(v, variant)
}

// StreamCheck folds one vector's seed variants (Run with variants 0 to
// n-1, n > 0) into one result per case: a case whose status differs
// between variants is "random", or "sample-only" when it only ever
// matched samples. runs[0] is the reference and is updated in place.
func StreamCheck(runs [][]Result) []Result {
	return pvStreamCheck(runs)
}

// Path is the vector's corpus and file name, such as "cb/cb010.json".
func (v *Vector) Path() string { return v.path }

// Start is a vector's starting state as the harness builds it, and how
// the vector's fleet and planet ids map to Elegy's.
type Start struct {
	Game    engine.Game
	Fleets  map[[2]int]int // (owner, vector fleet id) → Elegy fleet ID
	Planets map[int]int    // vector planet id → Elegy planet ID
	// Skip is why every case of the vector is skipped, if one is.
	Skip string
}

// Start builds v's starting state.
func (v *Vector) Start() (Start, error) {
	l, err := loadVector(v)
	if err != nil {
		return Start{}, err
	}
	return Start{Game: l.g, Fleets: l.fleetID, Planets: l.planet, Skip: l.global}, nil
}
