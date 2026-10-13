package engine_test

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// RG-002's battle records are checked against the RG-002 ledger
// (testdata/vectors/rg/rg002-ledger.txt, internal/vectors
// rg002_ledger_test.go): each record CheckBattleRecords would fail, and
// each chained sample that misses, is a ledger line "class | vector |
// case and year | battle record (x,y) | detail". TestBattleRecords fails
// on a line the ledger lacks and on a ledger line no longer produced.
// A strict record's class is written by hand with its owner; a sample's
// is "sample", and a restart record a seed change alone makes match
// (CheckBattleRecords' random column) is "random-driven" with its count
// of matching variants, so a change in that count fails too.

const rg002BattleLedger = "testdata/vectors/rg/rg002-ledger.txt"

// rgBattleLedgers are the representative games' ledgers by chained run:
// RG-003's records go to its own ledger, with RG-002's classes.
var rgBattleLedgers = map[string]string{
	"rg002": rg002BattleLedger,
	"rg003": "testdata/vectors/rg/rg003-ledger.txt",
}

// rgBattleLedger is the ledger of a battle ledger key: the one named by
// the chained run its vector belongs to ("rg003-y7" → rg003).
func rgBattleLedger(key string) string {
	game, _, _ := strings.Cut(key, " | ")
	game, _, _ = strings.Cut(game, "-")
	return rgBattleLedgers[game]
}

// rg002BattleWhere parses CheckBattleRecords' locator for an RG-002 or
// RG-003 record: vector, case, stream, x, y, year, then the message.
var rg002BattleWhere = regexp.MustCompile(`^rg/(rg00[23][^ ]*)\.json (\S+) "([^"]*)" \((\d+),(\d+)\) y(\d+): (.*)$`)

// rg002Pointer matches a printed pointer, which differs between runs.
var rg002Pointer = regexp.MustCompile(`0x[0-9a-f]+`)

// rg002RandomDraws matches a random record's count of the year's draws,
// which moves with any change to them, keeping the count of matching
// variants.
var rg002RandomDraws = regexp.MustCompile(`^random \(after \d+ draws in the year; (\d+ of \d+ other variants match)\): `)

// rg002SampleDraws matches a sample's count of earlier draws, which
// moves with any change to earlier years' draws.
var rg002SampleDraws = regexp.MustCompile(`^sample \((?:after \d+ draws of earlier years|flagged in the vector)\) `)

// ledgerTB takes CheckBattleRecords' RG-002 failures and sample misses
// as ledger lines, by key, with "", "sample" or "random-driven" (lane D's
// in-year rule, with its count of matching variants) as the class, and passes
// everything else to the test.
type ledgerTB struct {
	testing.TB
	lines map[string]string
}

func (l *ledgerTB) take(msg, class string) bool {
	m := rg002BattleWhere.FindStringSubmatch(msg)
	if m == nil {
		return false
	}
	detail := rg002Pointer.ReplaceAllString(m[7], "0x?")
	if class == "random-driven" {
		m := rg002RandomDraws.FindStringSubmatch(detail)
		if m == nil {
			return false
		}
		detail = "random, " + m[1] + ": " + strings.TrimPrefix(detail, m[0])
	}
	if class == "sample" {
		d, ok := strings.CutPrefix(detail, rg002SampleDraws.FindString(detail))
		if !ok || d == detail {
			return false
		}
		detail = "sample " + d
	}
	if class == "matched" {
		d, ok := strings.CutPrefix(detail, "matched: ")
		if !ok {
			return false
		}
		detail = "matched: " + d
	}
	where := fmt.Sprintf("%s y%s", m[2], m[6])
	if m[3] != "" {
		where += " stream " + m[3]
	}
	key := strings.Join([]string{m[1], where, fmt.Sprintf("battle record (%s,%s)", m[4], m[5]), detail}, " | ")
	l.lines[key] = class
	return true
}

func (l *ledgerTB) Errorf(format string, args ...any) {
	l.TB.Helper()
	if msg := fmt.Sprintf(format, args...); !l.take(msg, "") {
		l.TB.Errorf("%s", msg)
	}
}

func (l *ledgerTB) Logf(format string, args ...any) {
	l.TB.Helper()
	msg := fmt.Sprintf(format, args...)
	switch {
	case strings.Contains(msg, ": sample (after "), strings.Contains(msg, ": sample (flagged in the vector) "):
		l.take(msg, "sample")
	case strings.Contains(msg, ": random (after "):
		l.take(msg, "random-driven")
	case strings.Contains(msg, ": matched: "):
		l.take(msg, "matched")
	}
	l.TB.Logf("%s", msg)
}

// rg002BattleHand reports whether class is a hand class with its owner
// ("incorrect mechanic [lane A]"), as internal/vectors rg002HandClass, or
// the unresolved randomness class with its owner and reason
// ("unresolved randomness [lane D]: reason"), as internal/vectors
// rg002UnresolvedClass: a miss on every harness variant that a named
// random outcome explains. A battle line may also be hand-classed
// "random-driven [owner]: evidence" when oracle evidence, not the harness
// variants, shows the original's own streams differ there (RG-003-Y9's
// battle at (1388,1386), stars-elegy 045bc90).
func rg002BattleHand(class string) bool {
	for _, h := range []string{"missing mechanic", "incorrect mechanic", "state-import error"} {
		if rest, ok := strings.CutPrefix(class, h+" ["); ok && strings.HasSuffix(rest, "]") && len(rest) > 1 {
			return true
		}
	}
	for _, h := range []string{"unresolved randomness [", "random-driven ["} {
		if rest, ok := strings.CutPrefix(class, h); ok {
			owner, reason, ok := strings.Cut(rest, "]: ")
			return ok && owner != "" && strings.TrimSpace(reason) != ""
		}
	}
	return false
}

// readBattleLedger reads a ledger's battle record lines, by key.
func readBattleLedger(t testing.TB, path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, " | ", 5)
		if strings.HasPrefix(line, "#") || len(parts) < 5 || !strings.HasPrefix(parts[3], "battle record ") {
			continue
		}
		out[strings.Join(parts[1:], " | ")] = parts[0]
	}
	return out
}

// checkBattleLedger compares the lines taken with the ledger's.
func checkBattleLedger(t testing.TB, got, ledger map[string]string) {
	var keys []string
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		class, ok := ledger[k]
		switch {
		case !ok:
			t.Errorf("new battle ledger line: %s | %s", got[k], k)
		case got[k] != "" && class != got[k]:
			t.Errorf("battle ledger line is %s, was %s: %s", got[k], class, k)
		case got[k] == "" && !rg002BattleHand(class):
			t.Errorf("battle ledger line needs a hand class with its owner, has %q: %s", class, k)
		}
	}
	var gone []string
	for k, class := range ledger {
		if _, ok := got[k]; !ok {
			gone = append(gone, class+" | "+k)
		}
	}
	sort.Strings(gone)
	for _, l := range gone {
		t.Errorf("battle ledger line no longer produced (resolved or changed): %s", l)
	}
}

// writeBattleLedger rewrites a ledger's battle record lines from got,
// keeping each remaining line's hand class and every other line, for
// RG002_LEDGER=write.
func writeBattleLedger(t testing.TB, path string, got, ledger map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if parts := strings.SplitN(line, " | ", 5); strings.HasPrefix(line, "#") || len(parts) < 5 || !strings.HasPrefix(parts[3], "battle record ") {
			out = append(out, line)
		}
	}
	var keys []string
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		class := got[k]
		if class == "" {
			class = ledger[k]
			if !rg002BattleHand(class) {
				class = "UNCLASSIFIED"
			}
		}
		out = append(out, class+" | "+k)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckBattleLedger pins what TestBattleRecords fails on for RG-002:
// a new record line, a line gone, a sample the ledger holds as strict,
// a strict record without a hand class, and how a failure is taken.
func TestCheckBattleLedger(t *testing.T) {
	ledger := map[string]string{
		"rg002-y2 | C y1 | battle record (1,2) | tokens: x":     "incorrect mechanic [lane A]",
		"rg002 | C y2 | battle record (3,4) | sample missed: y": "sample",
	}
	for _, c := range []struct {
		name   string
		got    map[string]string
		errors int
	}{
		{"same", map[string]string{"rg002-y2 | C y1 | battle record (1,2) | tokens: x": "", "rg002 | C y2 | battle record (3,4) | sample missed: y": "sample"}, 0},
		{"new", map[string]string{"rg002-y2 | C y1 | battle record (1,2) | tokens: x": "", "rg002 | C y2 | battle record (3,4) | sample missed: y": "sample", "rg002 | C y3 | battle record (5,6) | z": ""}, 1},
		{"gone", map[string]string{"rg002-y2 | C y1 | battle record (1,2) | tokens: x": ""}, 1},
		{"sample held strict", map[string]string{"rg002-y2 | C y1 | battle record (1,2) | tokens: x": "sample", "rg002 | C y2 | battle record (3,4) | sample missed: y": "sample"}, 1},
		{"strict held as sample", map[string]string{"rg002-y2 | C y1 | battle record (1,2) | tokens: x": "", "rg002 | C y2 | battle record (3,4) | sample missed: y": ""}, 1},
		{"random held strict", map[string]string{"rg002-y2 | C y1 | battle record (1,2) | tokens: x": "random-driven", "rg002 | C y2 | battle record (3,4) | sample missed: y": "sample"}, 1},
	} {
		r := &recordTB{TB: t}
		if checkBattleLedger(r, c.got, ledger); len(r.errs) != c.errors {
			t.Errorf("%s: errors %q, want %d", c.name, r.errs, c.errors)
		}
	}
	for _, c := range []struct {
		class string
		hand  bool
	}{
		{"incorrect mechanic [lane A]", true},
		{"incorrect mechanic []", false},
		{"incorrect mechanic", false},
		{"unresolved randomness [lane D]: a mine stop", true},
		{"unresolved randomness [lane D]", false},
		{"unresolved randomness [lane D]:  ", false},
		{"unresolved randomness []: a mine stop", false},
		{"random-driven [lane D]: the oracle fought here in 3 of 4 streams", true},
		{"random-driven [lane D]", false},
		{"random-driven []: 3 of 4 streams", false},
		{"random-driven", false},
		{"sample", false},
	} {
		if got := rg002BattleHand(c.class); got != c.hand {
			t.Errorf("rg002BattleHand(%q) = %v, want %v", c.class, got, c.hand)
		}
	}
	r := &recordTB{TB: t}
	l := &ledgerTB{TB: r, lines: map[string]string{}}
	l.Errorf("%s", `rg/rg002-y2.json C "" (1,2) y1: tokens: Elegy {Planet:0xc000123}`)
	l.Logf("%s", `rg/rg002.json C "" (3,4) y2: sample (after 99 draws of earlier years) missed: y`)
	l.Errorf("%s", `rg/rg001.json C "" (1,2) y1: tokens: x`)
	l.Logf("%s", `rg/rg002-y3.json C "" (5,6) y1: random (after 12 draws in the year; 2 of 7 other variants match): tokens: z`)
	l.Logf("%s", `rg/rg002-y4.json C "" (7,8) y1: sample (flagged in the vector) missed: tokens: w`)
	want := map[string]string{
		"rg002-y4 | C y1 | battle record (7,8) | sample missed: tokens: w":                       "sample",
		"rg002-y2 | C y1 | battle record (1,2) | tokens: Elegy {Planet:0x?}":                     "",
		"rg002 | C y2 | battle record (3,4) | sample missed: y":                                  "sample",
		"rg002-y3 | C y1 | battle record (5,6) | random, 2 of 7 other variants match: tokens: z": "random-driven",
	}
	if fmt.Sprint(l.lines) != fmt.Sprint(want) || len(r.errs) != 1 {
		t.Errorf("taken %q, passed on %q; want %q and the rg001 failure", l.lines, r.errs, want)
	}
}
