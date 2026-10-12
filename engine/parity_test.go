package engine_test

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/internal/vectors"
)

// Parity harness: runs the stars-elegy parity vectors (testdata/vectors,
// format in its FORMAT.md) through GenerateTurn with package vectors and
// compares each case's expectations. A case is skipped, not failed, when
// it needs something Elegy does not model yet (a task, target, object or
// expectation kind), or when its outcome is random. CONFIRMED and LEGACY
// BUG cases are exact-match targets; MEASURED passes are tallied apart,
// and every failure, MEASURED or not, counts as a failure.
//
// testdata/vectors/baseline.txt lists the cases that pass, MEASURED ones
// included, and the "random" cases with how many seed variants they pass
// with (vectors.Seeds); the test fails when a listed case no longer
// passes, or a random one passes with fewer seeds. Run with -v to see the
// per-corpus report, and with PARITY_BASELINE=write to rewrite the
// baseline.

const pvBaseline = "testdata/vectors/baseline.txt"

func TestParityVectors(t *testing.T) {
	var all []vectors.Result
	// Expectations skipped inside checked cases, by reason, from the
	// reference seed.
	partial := map[string]int{}
	vs, err := vectors.ReadDir(filepath.Join("testdata", "vectors"))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		runs := make([][]vectors.Result, vectors.Seeds)
		for k := range runs {
			runs[k] = vectors.Run(v, k)
		}
		for _, r := range runs[0] {
			for _, why := range r.Skipped {
				if strings.HasPrefix(why, "stream ") {
					why = "stream (another oracle random stream)"
				}
				partial[why]++
			}
		}
		all = append(all, vectors.StreamCheck(runs)...)
	}
	type tally struct{ pass, fail, skip, measured, differs, random, randomFail, sampleOnly int }
	per := map[string]*tally{}
	var corpora []string
	for _, r := range all {
		tl := per[r.Corpus]
		if tl == nil {
			tl = &tally{}
			per[r.Corpus] = tl
			corpora = append(corpora, r.Corpus)
		}
		switch {
		case r.Status == "skip":
			tl.skip++
		case r.Status == "differs":
			tl.differs++
		case r.Status == "sample-only":
			tl.sampleOnly++
		case r.Status == "random" && r.RefStatus == "fail":
			tl.randomFail++
		case r.Status == "random":
			tl.random++
		case r.Status == "pass" && r.Tag == "MEASURED":
			tl.measured++
		case r.Status == "pass":
			tl.pass++
		default:
			tl.fail++
		}
	}
	for _, c := range corpora {
		tl := per[c]
		t.Logf("%-5s pass %3d  measured pass %3d  sample-only %3d  fail %3d  skip %3d  differs %3d  random %3d  random, reference seed fails %3d", c, tl.pass, tl.measured, tl.sampleOnly, tl.fail, tl.skip, tl.differs, tl.random, tl.randomFail)
	}
	for _, r := range all {
		if r.Status != "pass" && r.Status != "sample-only" {
			t.Logf("%s %-10s %-4s %s", r.Status, r.Tag, r.ID, r.Why)
		}
	}
	for _, why := range slices.Sorted(maps.Keys(partial)) {
		t.Logf("expectations skipped in checked cases: %4d %s", partial[why], why)
	}

	if os.Getenv("PARITY_BASELINE") == "write" {
		f, err := os.Create(pvBaseline)
		if err != nil {
			t.Fatal(err)
		}
		w := bufio.NewWriter(f)
		fmt.Fprintln(w, "# Parity cases Elegy passes (TestParityVectors). One case id per line;")
		fmt.Fprintf(w, "# \"random k\" after an id: the case passes with k of the %d seed variants.\n", vectors.Seeds)
		fmt.Fprintln(w, "# \"sample-only\" after an id: every expectation the case compares is a sample")
		fmt.Fprintln(w, "# that matched, so it is evidence of the rule, not of an exact value; \"sample-only k\":")
		fmt.Fprintln(w, "# so on k seeds, with every sample missing on the others.")
		for _, r := range all {
			switch r.Status {
			case "pass":
				fmt.Fprintln(w, r.ID)
			case "sample-only":
				if r.Passes < vectors.Seeds {
					fmt.Fprintf(w, "%s sample-only %d\n", r.ID, r.Passes)
				} else {
					fmt.Fprintln(w, r.ID, "sample-only")
				}
			case "random":
				fmt.Fprintf(w, "%s random %d\n", r.ID, r.Passes)
			}
		}
		w.Flush()
		f.Close()
		return
	}
	b, err := os.ReadFile(pvBaseline)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]vectors.Result{}
	for _, r := range all {
		status[r.ID] = r
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		r := status[fields[0]]
		if len(fields) >= 2 && fields[1] == "sample-only" {
			want := vectors.Seeds
			if len(fields) == 3 {
				want, _ = strconv.Atoi(fields[2])
			}
			if r.Status != "pass" && (r.Status != "sample-only" || r.Passes < want) {
				t.Errorf("baseline case %s was sample-only with %d of %d seeds, now %s: %s", fields[0], want, vectors.Seeds, r.Status, r.Why)
			}
			continue
		}
		want := vectors.Seeds
		if len(fields) == 3 && fields[1] == "random" {
			want, _ = strconv.Atoi(fields[2])
		}
		got := r.Passes
		if r.Status == "pass" {
			got = vectors.Seeds
		}
		if got < want {
			t.Errorf("baseline case %s passes with %d of %d seeds, was %d: %s %s", fields[0], got, vectors.Seeds, want, r.Status, r.Why)
		}
	}
}
