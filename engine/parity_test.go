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
//
// testdata/vectors/skips.txt pins the census of skipped expectations by
// reason. A skip never fails a case, so a mapping or comparator that
// disappears would only turn its expectations back into skips; the test
// fails when any count differs from the file, up or down, so every change
// shows in the PR's diff. PARITY_BASELINE=write rewrites it too.

const (
	pvBaseline = "testdata/vectors/baseline.txt"
	pvSkips    = "testdata/vectors/skips.txt"
)

func TestParityVectors(t *testing.T) {
	var all []vectors.Result
	// Skipped expectations, by reason, from the reference seed: those of
	// wholly skipped cases too.
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
		t.Logf("expectations skipped: %4d %s", partial[why], why)
	}

	if os.Getenv("PARITY_BASELINE") == "write" {
		if err := os.WriteFile(pvSkips, []byte(formatSkipCensus(partial)), 0o644); err != nil {
			t.Fatal(err)
		}
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
	sb, err := os.ReadFile(pvSkips)
	if err != nil {
		t.Fatal(err)
	}
	want, err := parseSkipCensus(string(sb))
	if err != nil {
		t.Fatalf("%s: %v", pvSkips, err)
	}
	for _, d := range skipCensusDiff(want, partial) {
		t.Errorf("%s (rewrite %s with PARITY_BASELINE=write if intended)", d, pvSkips)
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

// formatSkipCensus writes the skipped-expectation census: a comment
// header, then one "count reason" line per reason, sorted by reason.
func formatSkipCensus(census map[string]int) string {
	var b strings.Builder
	b.WriteString("# Skipped parity expectations by reason, from the reference seed\n")
	b.WriteString("# (TestParityVectors). Any change to a count fails the test until this\n")
	b.WriteString("# file is rewritten with PARITY_BASELINE=write.\n")
	for _, why := range slices.Sorted(maps.Keys(census)) {
		fmt.Fprintf(&b, "%4d %s\n", census[why], why)
	}
	return b.String()
}

// parseSkipCensus reads formatSkipCensus's output.
func parseSkipCensus(s string) (map[string]int, error) {
	census := map[string]int{}
	for i, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n, why, ok := strings.Cut(line, " ")
		count, err := strconv.Atoi(n)
		if !ok || err != nil || count <= 0 || why == "" {
			return nil, fmt.Errorf("line %d: %q is not \"count reason\"", i+1, line)
		}
		if _, dup := census[why]; dup {
			return nil, fmt.Errorf("line %d: reason %q listed twice", i+1, why)
		}
		census[why] = count
	}
	return census, nil
}

// skipCensusDiff lists, sorted by reason, each reason whose count in got
// differs from want, a reason missing from either side counting as 0.
func skipCensusDiff(want, got map[string]int) []string {
	reasons := maps.Clone(want)
	maps.Copy(reasons, got)
	var diffs []string
	for _, why := range slices.Sorted(maps.Keys(reasons)) {
		if w, g := want[why], got[why]; w != g {
			diffs = append(diffs, fmt.Sprintf("skipped expectations %q: %d, was %d", why, g, w))
		}
	}
	return diffs
}
