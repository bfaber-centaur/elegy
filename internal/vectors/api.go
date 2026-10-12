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
