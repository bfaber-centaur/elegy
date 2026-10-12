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
//   - random-driven: a field that matches on at least one of Elegy's 8
//     seed variants but not on the reference one, so Elegy's draws
//     decide it; the line names how many variants match;
//   - not compared: an expectation or field the harness does not compare,
//     with its reason;
//   - unsupported order: an order the loader cannot convert;
//   - missing mechanic, incorrect mechanic, state-import error: a field
//     that matches on none of the 8 variants, or an order Elegy refuses,
//     classified by hand with its owner in brackets;
//   - unresolved randomness: a field that matches on none of the 8
//     variants but whose value differs between them, which its owner
//     argues is draw-driven all the same; by hand, with the owner and a
//     one-line reason ("unresolved randomness [lane D]: reason").
//
// A field that misses on every variant but not the same way on all of
// them says so in its detail ("0/8 variants match, values vary"). An
// order refused on other variants but not on the reference one, outside
// the chained run's samples, is a hand line naming those variants; it
// varies, so it may be unresolved randomness.
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
var rg002Hand = []string{"missing mechanic", "incorrect mechanic", "state-import error", "unsupported order"}

// rg002Unresolved is the hand class for a line that misses on every
// variant by a varying amount, argued draw-driven by its owner.
const rg002Unresolved = "unresolved randomness"

// ledgerLine is one ledger line: its class and the rest, its key, and
// whether its value varies between seed variants that all miss.
type ledgerLine struct {
	class, key string
	varies     bool
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
		out = append(out, ledgerLine{class: class, key: strings.Join([]string{vector, where, object, detail}, " | ")})
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
		others := [][]string{nil}
		for seed := 1; seed < Seeds; seed++ {
			fs, rs, _ := rg002Run(t, name, seed)
			others = append(others, rs)
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
		for r, seeds := range rg002VariantRefusals(name == "rg002", rejected, others) {
			out = append(out, rg002VariantRefusalLine(name, r, seeds))
		}
		for k, d := range ref {
			if d == "" {
				continue
			}
			out = append(out, rg002FieldLine(name+" | "+k.where+" | "+k.object, d, k.skip, k.sample, matches[k], same[k]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

// rg002FieldClass classifies a reference-run field miss from how many of
// the Seeds variants matched and how many missed exactly as the reference
// did: its computed class, or "" for a hand line, and a note for its
// detail. A field is random-driven only when some variant matches; one
// that misses on all of them is a hand line, noted when its value varies.
func rg002FieldClass(skip, sample bool, matches, same int) (class, note string) {
	switch {
	case skip:
		return "not compared", ""
	case sample:
		return "sample", ""
	case matches > 0:
		return "random-driven", fmt.Sprintf("%d/%d variants match", matches, Seeds)
	case same != Seeds:
		return "", fmt.Sprintf("0/%d variants match, values vary", Seeds)
	}
	return "", ""
}

// rg002FieldLine is the ledger line of a reference-run field miss, at
// where ("vector | case and year | object"), with its detail and variant
// counts as rg002FieldClass takes them.
func rg002FieldLine(where, detail string, skip, sample bool, matches, same int) ledgerLine {
	class, note := rg002FieldClass(skip, sample, matches, same)
	if note != "" {
		detail += " (" + note + ")"
	}
	return ledgerLine{class: class, key: where + " | " + detail, varies: class == "" && note != ""}
}

// rg002VariantRefusals lists, by refusal, the other variants (others[k]
// is variant k's refusals; others[0] is unused) that refuse an order the
// reference run (ref) accepts, leaving out a chained run's samples.
func rg002VariantRefusals(chained bool, ref []string, others [][]string) map[string][]int {
	refused := map[string]bool{}
	for _, r := range ref {
		refused[r] = true
	}
	out := map[string][]int{}
	for k := 1; k < len(others); k++ {
		for _, r := range others[k] {
			if !refused[r] && !(chained && rejectedSample(r)) {
				out[r] = append(out[r], k)
			}
		}
	}
	return out
}

// rg002VariantRefusalLine is the ledger line of an order refused on the
// other variants listed (seeds, in order) but not on the reference one:
// a hand line, its refusal depending on Elegy's draws (the player
// shuffle, for one), so it varies.
func rg002VariantRefusalLine(vector, refusal string, seeds []int) ledgerLine {
	var vs []string
	for _, k := range seeds {
		vs = append(vs, fmt.Sprint(k))
	}
	detail := fmt.Sprintf("%s (refused on variants %s of %d, not on the reference)", refusal, strings.Join(vs, ", "), Seeds)
	return ledgerLine{key: strings.Join([]string{vector, strings.SplitN(refusal, " player", 2)[0], "order", detail}, " | "), varies: true}
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
// brackets ("incorrect mechanic [lane A]"), optionally followed by a
// note ("incorrect mechanic [lane A]: note").
func rg002HandClass(class string) bool {
	for _, h := range rg002Hand {
		rest, ok := strings.CutPrefix(class, h+" [")
		if !ok {
			continue
		}
		owner, note, ok := strings.Cut(rest, "]")
		if ok && owner != "" && (note == "" || strings.HasPrefix(note, ": ") && strings.TrimSpace(note[2:]) != "") {
			return true
		}
	}
	return false
}

// rg002UnresolvedClass reports whether class is the unresolved
// randomness class with an owner and a reason ("unresolved randomness
// [lane D]: reason").
func rg002UnresolvedClass(class string) bool {
	rest, ok := strings.CutPrefix(class, rg002Unresolved+" [")
	if !ok {
		return false
	}
	owner, reason, ok := strings.Cut(rest, "]: ")
	return ok && owner != "" && strings.TrimSpace(reason) != ""
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
		case l.class == "" && rg002UnresolvedClass(class) && !l.varies:
			t.Errorf("ledger line is the same on every variant, so not %q: %s", class, l.key)
			l.class = class
		case l.class == "" && !rg002HandClass(class) && !rg002UnresolvedClass(class):
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
#
# Order coverage (lane F, at stars-elegy fe0ccc5): the 42 order files hold
# 1867 order records and the vectors carry 1867 orders, none dropped, so a
# refused order here is Elegy's gap, never the decoder's. Known vector
# omissions, both UNRESOLVED in meaning and constant across RG-002: the
# lay-mines task's second word (6 orders, all "indefinitely") and the high
# nibble of the waypoint target type (0x31 and 0x51, 23 orders).
`

// TestCheckLedger pins what TestRG002Ledger fails on: a new line, a line
// gone, a computed class that changed, a deterministic line without a
// hand class and owner, and unresolved randomness on a line that does not
// vary or without its owner and reason.
func TestCheckLedger(t *testing.T) {
	random := ledgerLine{class: "random-driven", key: "v | c y1 #0 | o | random"}
	fixed := ledgerLine{key: "v | c y1 #1 | o | fixed"}
	varies := ledgerLine{key: "v | c y1 #2 | o | varies", varies: true}
	ledger := map[string]string{random.key: "random-driven", fixed.key: "incorrect mechanic [lane A]"}
	with := func(class string) map[string]string {
		return map[string]string{random.key: "random-driven", fixed.key: "incorrect mechanic [lane A]", varies.key: class}
	}
	for _, c := range []struct {
		name   string
		got    []ledgerLine
		ledger map[string]string
		errors int
	}{
		{"same", []ledgerLine{random, fixed}, ledger, 0},
		{"new line", []ledgerLine{random, fixed, {class: "sample", key: "v | c y2 #0 | o | x"}}, ledger, 1},
		{"line gone", []ledgerLine{random}, ledger, 1},
		{"class changed", []ledgerLine{{class: "sample", key: random.key}, fixed}, ledger, 1},
		{"no owner", []ledgerLine{random, fixed}, map[string]string{random.key: "random-driven", fixed.key: "incorrect mechanic"}, 1},
		{"unclassified", []ledgerLine{random, fixed}, map[string]string{random.key: "random-driven", fixed.key: rg002Unclassified}, 1},
		{"varying line, hand class", []ledgerLine{random, fixed, varies}, with("incorrect mechanic [economy]"), 0},
		{"varying line, unresolved randomness", []ledgerLine{random, fixed, varies}, with("unresolved randomness [lane D]: mining draws"), 0},
		{"varying line, random-driven", []ledgerLine{random, fixed, varies}, with("random-driven"), 1},
		{"varying line, unresolved without a reason", []ledgerLine{random, fixed, varies}, with("unresolved randomness [lane D]"), 1},
		{"fixed line, unresolved randomness", []ledgerLine{random, fixed},
			map[string]string{random.key: "random-driven", fixed.key: "unresolved randomness [lane D]: mining draws"}, 1},
	} {
		e := &errorCounter{TB: t}
		if checkLedger(e, c.got, c.ledger); e.errors != c.errors {
			t.Errorf("%s: %d errors, want %d", c.name, e.errors, c.errors)
		}
	}
	for class, want := range map[string]bool{
		"incorrect mechanic [lane A]": true, "missing mechanic [economy]": true, "random-driven [lane D]": false,
		"incorrect mechanic []": false, "incorrect mechanic": false, "sample": false, "lucky guess [lane A]": false,
		"unresolved randomness [lane D]: mining draws": false, "incorrect mechanic [lane A]: refused": true,
		"incorrect mechanic [lane A]:  ": false, "incorrect mechanic [lane A] refused": false, "incorrect mechanic [lane A]:": false,
		"incorrect mechanic []: refused": false,
	} {
		if rg002HandClass(class) != want {
			t.Errorf("rg002HandClass(%q) = %v, want %v", class, !want, want)
		}
	}
	for class, want := range map[string]bool{
		"unresolved randomness [lane D]: mining draws": true, "unresolved randomness [lane D]": false,
		"unresolved randomness [lane D]:  ": false, "unresolved randomness []: mining draws": false,
		"unresolved randomness: mining draws": false, "incorrect mechanic [lane D]: mining draws": false,
	} {
		if rg002UnresolvedClass(class) != want {
			t.Errorf("rg002UnresolvedClass(%q) = %v, want %v", class, !want, want)
		}
	}
}

// TestRG002VariantRefusalLine pins a variant-only refusal's line: the
// variants that refuse it, a hand class, and that it varies.
func TestRG002VariantRefusalLine(t *testing.T) {
	got := rg002VariantRefusalLine("rg002-y1", "year 1 player 2 order 10 (FollowOrder): no such object", []int{1, 6})
	want := ledgerLine{key: "rg002-y1 | year 1 | order | year 1 player 2 order 10 (FollowOrder): no such object (refused on variants 1, 6 of 8, not on the reference)", varies: true}
	if got != want {
		t.Errorf("rg002VariantRefusalLine = %+v, want %+v", got, want)
	}
}

// TestRG002VariantRefusals pins which refusals of other variants are
// listed: not the reference's own, and not a chained run's samples.
func TestRG002VariantRefusals(t *testing.T) {
	ref := []string{"year 1 player 1 order 2 (X): a"}
	others := [][]string{nil, {"year 1 player 1 order 2 (X): a", "year 1 player 2 order 3 (X): b"}, {"year 2 player 2 order 4 (X): c"}, {"year 1 player 2 order 3 (X): b"}}
	for _, c := range []struct {
		chained bool
		want    string
	}{
		{false, "map[year 1 player 2 order 3 (X): b:[1 3] year 2 player 2 order 4 (X): c:[2]]"},
		{true, "map[year 1 player 2 order 3 (X): b:[1 3]]"},
	} {
		if got := fmt.Sprint(rg002VariantRefusals(c.chained, ref, others)); got != c.want {
			t.Errorf("rg002VariantRefusals(chained %v) = %s, want %s", c.chained, got, c.want)
		}
	}
}

// TestRG002FieldClass pins how a field miss is classed from its variant
// counts: random-driven only when some variant matches, otherwise a hand
// line, noted when its value varies between the variants.
func TestRG002FieldClass(t *testing.T) {
	for _, c := range []struct {
		skip, sample  bool
		matches, same int
		class, note   string
		varies        bool
	}{
		{true, false, 3, 2, "not compared", "", false},
		{false, true, 3, 2, "sample", "", false},
		{false, false, 1, 7, "random-driven", "1/8 variants match", false},
		{false, false, 7, 1, "random-driven", "7/8 variants match", false},
		{false, false, 0, 3, "", "0/8 variants match, values vary", true},
		{false, false, 0, 7, "", "0/8 variants match, values vary", true},
		{false, false, 0, 8, "", "", false},
	} {
		class, note := rg002FieldClass(c.skip, c.sample, c.matches, c.same)
		if class != c.class || note != c.note {
			t.Errorf("rg002FieldClass(%v, %v, %d, %d) = %q, %q; want %q, %q", c.skip, c.sample, c.matches, c.same, class, note, c.class, c.note)
		}
		key := "v | c y1 #0 | o | d"
		if note != "" {
			key += " (" + note + ")"
		}
		want := ledgerLine{class: c.class, key: key, varies: c.varies}
		if got := rg002FieldLine("v | c y1 #0 | o", "d", c.skip, c.sample, c.matches, c.same); got != want {
			t.Errorf("rg002FieldLine(%v, %v, %d, %d) = %+v, want %+v", c.skip, c.sample, c.matches, c.same, got, want)
		}
	}
}
