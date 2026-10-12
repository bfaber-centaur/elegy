package vectors

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// The RG-002 ledger (engine/testdata/vectors/rg/rg002-ledger.txt) lists
// every way the reference run of RG-002 (seed variant 0, every recorded
// order loaded) differs from the original: each mismatched field, each
// expectation or field not compared, each refused order, and each battle
// record that does not match (engine TestBattleRecords). TestRG002Ledger
// fails on any line the run produces that the ledger lacks and on any
// ledger line the run no longer produces, so the file is the exact
// current state, never a skip.
//
// Each line is "class | vector | case and year | object | detail". The
// classes:
//   - sample: an expectation marked as one stream's random outcome, or
//     any result of the chained run after its first year, which follows
//     Elegy's own draws in the years before it (docs/RANDOM-PARITY.md
//     "Samples");
//   - random-driven: a field whose result differs between Elegy's 8
//     seed variants, so Elegy's draws decide it; the line names how many
//     variants match;
//   - not compared: an expectation or field the harness does not compare,
//     with its reason;
//   - unsupported order: an order the loader cannot convert;
//   - missing mechanic, incorrect mechanic, state-import error: a field
//     that differs the same way on all 8 variants, or an order Elegy
//     refuses, classified by hand with its owner in brackets.
//
// The first four are computed. A deterministic line carries the class
// written in the ledger; a new one is written as UNCLASSIFIED and fails
// until someone classifies it. RG002_LEDGER=write go test -run
// TestRG002Ledger ./internal/vectors rewrites the ledger, keeping the
// hand classes of lines that remain.

const (
	rg002Ledger = "../../engine/testdata/vectors/rg/rg002-ledger.txt"
	// rg002Unclassified marks a deterministic line nobody classified.
	rg002Unclassified = "UNCLASSIFIED"
)

// rg002Vectors are the RG-002 runs the ledger covers: the chained six
// years and each year's restart from the recorded state.
var rg002Vectors = []string{"rg002", "rg002-y1", "rg002-y2", "rg002-y3", "rg002-y4", "rg002-y5", "rg002-y6"}

// rg002Hand lists the hand classes; any other class is computed.
var rg002Hand = []string{"missing mechanic", "incorrect mechanic", "state-import error", "unsupported order", "random-driven"}

// ledgerLine is one ledger line: its class and the rest, its key.
type ledgerLine struct {
	class, key string
}

// rg002Field is one field's outcome for one seed variant: "" when it
// matched, else its detail.
type rg002Field struct {
	vector, where, object, field string
	sample, skip                 bool
}

// rg002Describe names an expectation's object: its kind and the fields
// that select it.
func rg002Describe(e pvExpect) string {
	parts := []string{e.Kind}
	for _, f := range []struct {
		name string
		v    *int
	}{{"owner", e.Owner}, {"id", e.ID}, {"planet", e.Planet}, {"slot", e.Slot}, {"x", e.X}, {"y", e.Y}, {"viewer", e.Viewer}, {"seen_by", e.SeenBy}} {
		if f.v != nil {
			parts = append(parts, fmt.Sprintf("%s=%d", f.name, *f.v))
		}
	}
	if len(e.Subject) > 0 {
		parts = append(parts, "subject="+strings.Join(strings.Fields(string(e.Subject)), ""))
	}
	if e.Screen != "" || e.Field != "" {
		parts = append(parts, "reading="+e.Screen+"/"+e.Field)
	}
	if e.Stream != "" {
		parts = append(parts, "stream="+e.Stream)
	}
	return strings.Join(parts, " ")
}

// rg002FieldName is the field a mismatch names: the text before " = ",
// or the whole mismatch when it names none ("fleet gone").
func rg002FieldName(m string) string {
	if f, _, ok := strings.Cut(m, " = "); ok {
		return f
	}
	return m
}

// rg002Run generates one RG-002 vector with one seed variant and returns
// each field outcome, keyed by everything but the detail, with its
// detail ("" for a match), and the run's refused orders and loader stop.
func rg002Run(t *testing.T, name string, seed int) (fields map[rg002Field]string, rejected []string, stop string) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	run := pvRun{seed: seed}
	l, games, genErr, stop := generate(v, run)
	if genErr != nil {
		t.Fatalf("%s seed %d: %v", name, seed, genErr)
	}
	fields = map[rg002Field]string{}
	for _, c := range v.Cases {
		for i, e := range c.Expect {
			// The expectation's index keeps two alike (two messages)
			// apart.
			k := rg002Field{vector: name, where: fmt.Sprintf("%s y%d #%d", c.ID, e.Year, i), object: rg002Describe(e),
				sample: run.sample(e) || v.Years > 1 && e.Year > 1}
			if e.Year >= len(games) {
				k.field, k.skip = "year", true
				fields[k] = "not generated: " + stop
				continue
			}
			msg, also, _ := strings.Cut(l.pvCheck(&games[e.Year], e), pvAlsoSkipped)
			if also != "" {
				ks := k
				ks.field, ks.skip = "also skipped", true
				fields[ks] = also
			}
			if r, ok := strings.CutPrefix(msg, "skip: "); ok {
				ks := k
				ks.field, ks.skip = "expectation", true
				fields[ks] = r
				continue
			}
			if msg == "" {
				k.field = "expectation"
				fields[k] = ""
				continue
			}
			for _, m := range strings.Split(msg, "; ") {
				k.field = rg002FieldName(m)
				fields[k] = m
			}
		}
	}
	return fields, l.rejected, stop
}

// rg002Lines runs every RG-002 vector on all seed variants and returns
// the reference run's ledger lines, with the computed class or "" for a
// line whose class is written by hand.
func rg002Lines(t *testing.T) []ledgerLine {
	var out []ledgerLine
	add := func(class, vector, where, object, detail string) {
		out = append(out, ledgerLine{class, strings.Join([]string{vector, where, object, detail}, " | ")})
	}
	for _, name := range rg002Vectors {
		ref, rejected, stop := rg002Run(t, name, 0)
		matches := map[rg002Field]int{}
		same := map[rg002Field]int{}
		for k, d := range ref {
			if d == "" {
				matches[k]++
			} else {
				same[k]++
			}
		}
		for seed := 1; seed < Seeds; seed++ {
			fs, _, _ := rg002Run(t, name, seed)
			for k := range ref {
				// A field another variant does not list matched there
				// (its expectation matched or named other fields).
				switch d, ok := fs[k]; {
				case !ok || d == "":
					matches[k]++
				case d == ref[k]:
					same[k]++
				}
			}
		}
		if stop != "" {
			add("unsupported order", name, "load", "vector", stop)
		}
		for _, r := range rejected {
			class := ""
			if rejectedSample(r) && name == "rg002" {
				class = "sample"
			}
			add(class, name, strings.SplitN(r, " player", 2)[0], "order", r)
		}
		for k, d := range ref {
			if d == "" {
				continue
			}
			object := k.object
			class := ""
			switch {
			case k.skip:
				class = "not compared"
			case k.sample:
				class = "sample"
			case same[k] != Seeds:
				class = "random-driven"
				d = fmt.Sprintf("%s (%d/%d variants match)", d, matches[k], Seeds)
			}
			add(class, name, k.where, object, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

// readLedger reads the ledger's lines of kind other than "battle", by
// key.
func readLedger(t *testing.T, path string) map[string]string {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return map[string]string{}
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]string{}
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		class, key, ok := strings.Cut(line, " | ")
		if !ok {
			t.Fatalf("ledger line without a class: %q", line)
		}
		if rg002BattleLine(key) {
			continue
		}
		if _, dup := out[key]; dup {
			t.Fatalf("ledger line twice: %q", line)
		}
		out[key] = class
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// rg002HandClass reports whether class is a hand class with an owner in
// brackets ("incorrect mechanic [lane A]").
func rg002HandClass(class string) bool {
	for _, h := range rg002Hand {
		if rest, ok := strings.CutPrefix(class, h+" ["); ok && strings.HasSuffix(rest, "]") && len(rest) > 1 {
			return true
		}
	}
	return false
}

// checkLedger compares the run's lines with the ledger's. A computed
// class must match the ledger's; a hand line takes the ledger's class,
// which must be a hand class with an owner. It returns the lines to
// write.
func checkLedger(t testing.TB, got []ledgerLine, ledger map[string]string) []ledgerLine {
	var write []ledgerLine
	seen := map[string]bool{}
	for _, l := range got {
		seen[l.key] = true
		class, ok := ledger[l.key]
		switch {
		case !ok:
			if l.class == "" {
				l.class = rg002Unclassified
			}
			t.Errorf("new ledger line: %s | %s", l.class, l.key)
		case l.class == "" && !rg002HandClass(class):
			t.Errorf("ledger line needs a hand class with its owner, has %q: %s", class, l.key)
			l.class = class
		case l.class == "":
			l.class = class
		case l.class != class:
			t.Errorf("ledger line is %s, was %s: %s", l.class, class, l.key)
		}
		write = append(write, l)
	}
	var gone []string
	for key, class := range ledger {
		if !seen[key] {
			gone = append(gone, class+" | "+key)
		}
	}
	sort.Strings(gone)
	for _, l := range gone {
		t.Errorf("ledger line no longer produced (resolved or changed): %s", l)
	}
	return write
}

func TestRG002Ledger(t *testing.T) {
	ledger := readLedger(t, rg002Ledger)
	battles := readBattleLines(t, rg002Ledger)
	write := checkLedger(t, rg002Lines(t), ledger)
	census := map[string]int{}
	for _, l := range write {
		class, _, _ := strings.Cut(l.class, " [")
		census[class]++
	}
	var classes []string
	for c := range census {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	for _, c := range classes {
		t.Logf("%-20s %5d", c, census[c])
	}
	if os.Getenv("RG002_LEDGER") != "write" {
		return
	}
	var b strings.Builder
	b.WriteString(rg002LedgerHeader)
	for _, l := range write {
		b.WriteString(l.class + " | " + l.key + "\n")
	}
	for _, l := range battles {
		b.WriteString(l + "\n")
	}
	if err := os.WriteFile(rg002Ledger, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rg002BattleLine reports whether a ledger key is a battle record's,
// which engine TestBattleRecords checks.
func rg002BattleLine(key string) bool {
	parts := strings.SplitN(key, " | ", 4)
	return len(parts) == 4 && strings.HasPrefix(parts[2], "battle record ")
}

// readBattleLines returns the ledger's battle lines, which engine
// TestBattleRecords checks, so a rewrite keeps them.
func readBattleLines(t *testing.T, path string) []string {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if _, key, ok := strings.Cut(line, " | "); ok && rg002BattleLine(key) {
			out = append(out, line)
		}
	}
	return out
}

const rg002LedgerHeader = `# RG-002 discrepancy ledger: every difference of the reference run (seed
# variant 0, every recorded order loaded) from the original, one per line:
# class | vector | case and year | object | detail
# Checked exactly by internal/vectors TestRG002Ledger and, for the
# "battle" lines, engine TestBattleRecords; see rg002_ledger_test.go for
# the classes. Rewrite with RG002_LEDGER=write go test -run TestRG002Ledger
# ./internal/vectors, then classify every UNCLASSIFIED line by hand.
`

// TestCheckLedger pins what TestRG002Ledger fails on: a new line, a line
// gone, a computed class that changed, and a deterministic line without
// a hand class and owner.
func TestCheckLedger(t *testing.T) {
	ledger := map[string]string{
		"v | c y1 #0 | o | random": "random-driven",
		"v | c y1 #1 | o | fixed":  "incorrect mechanic [lane A]",
	}
	for _, c := range []struct {
		name   string
		got    []ledgerLine
		ledger map[string]string
		errors int
	}{
		{"same", []ledgerLine{{"random-driven", "v | c y1 #0 | o | random"}, {"", "v | c y1 #1 | o | fixed"}}, ledger, 0},
		{"new line", []ledgerLine{{"random-driven", "v | c y1 #0 | o | random"}, {"", "v | c y1 #1 | o | fixed"}, {"sample", "v | c y2 #0 | o | x"}}, ledger, 1},
		{"line gone", []ledgerLine{{"random-driven", "v | c y1 #0 | o | random"}}, ledger, 1},
		{"class changed", []ledgerLine{{"sample", "v | c y1 #0 | o | random"}, {"", "v | c y1 #1 | o | fixed"}}, ledger, 1},
		{"no owner", []ledgerLine{{"random-driven", "v | c y1 #0 | o | random"}, {"", "v | c y1 #1 | o | fixed"}},
			map[string]string{"v | c y1 #0 | o | random": "random-driven", "v | c y1 #1 | o | fixed": "incorrect mechanic"}, 1},
		{"unclassified", []ledgerLine{{"random-driven", "v | c y1 #0 | o | random"}, {"", "v | c y1 #1 | o | fixed"}},
			map[string]string{"v | c y1 #0 | o | random": "random-driven", "v | c y1 #1 | o | fixed": rg002Unclassified}, 1},
	} {
		e := &errorCounter{TB: t}
		if checkLedger(e, c.got, c.ledger); e.errors != c.errors {
			t.Errorf("%s: %d errors, want %d", c.name, e.errors, c.errors)
		}
	}
	for class, want := range map[string]bool{
		"incorrect mechanic [lane A]": true, "missing mechanic [economy]": true, "random-driven [lane D]": true,
		"incorrect mechanic []": false, "incorrect mechanic": false, "sample": false, "lucky guess [lane A]": false,
	} {
		if rg002HandClass(class) != want {
			t.Errorf("rg002HandClass(%q) = %v, want %v", class, !want, want)
		}
	}
}
