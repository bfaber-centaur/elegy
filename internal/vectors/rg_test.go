package vectors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRGGames pins how the representative games are found: the corpus as
// it stands, a new game that needs no code, and the layouts that fail
// instead of dropping a game, a deleted ledger first.
func TestRGGames(t *testing.T) {
	games, err := RGGames(rgCorpus)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(games), fmt.Sprintf("[{rg001  6} {rg002 %[1]s/rg002-ledger.txt 6} {rg003 %[1]s/rg003-ledger.txt 12} {rg004 %[1]s/rg004-ledger.txt 12}]", rgCorpus); got != want {
		t.Errorf("RGGames(%s) = %s, want %s", rgCorpus, got, want)
	}
	base := []string{"rg001.json", "rg001-y1.json", "rg002.json", "rg002-y1.json", "rg002-y2.json", "rg002-ledger.txt", "README.md"}
	for _, c := range []struct {
		name          string
		add, remove   []string
		want, wantErr string
	}{
		{"as is", nil, nil, "[{rg001  1} {rg002 rg002-ledger.txt 2}]", ""},
		{"new game", []string{"rg005.json", "rg005-y1.json", "rg005-ledger.txt"}, nil, "[{rg001  1} {rg002 rg002-ledger.txt 2} {rg005 rg005-ledger.txt 1}]", ""},
		{"new game, no restarts", []string{"rg005.json", "rg005-ledger.txt"}, nil, "[{rg001  1} {rg002 rg002-ledger.txt 2} {rg005 rg005-ledger.txt 0}]", ""},
		{"ledger deleted", nil, []string{"rg002-ledger.txt"}, "", "rg002: no ledger rg002-ledger.txt"},
		{"new game, no ledger", []string{"rg005.json", "rg005-y1.json"}, nil, "", "rg005: no ledger rg005-ledger.txt"},
		{"RG-001 with a ledger", []string{"rg001-ledger.txt"}, nil, "", "rg001: has a ledger, but is held to exact matches"},
		{"restart gap", []string{"rg002-y4.json"}, nil, "", "rg002: restarts [1 2 4] skip year 3"},
		{"restarts from y2", nil, []string{"rg002-y1.json"}, "", "rg002: restarts [2] skip year 1"},
		{"chained run deleted", nil, []string{"rg002.json"}, "", "no chained run rg002.json"},
		{"ledger alone", []string{"rg005-ledger.txt"}, nil, "", "rg005-ledger.txt: no chained run rg005.json"},
		{"stray run", []string{"rg002-y1b.json"}, nil, "", "rg002-y1b.json: not a representative game's run"},
		{"restarts alone", []string{"rg005-y1.json"}, nil, "", "rg005: restarts with no chained run rg005.json"},
		{"four digits", []string{"rg0021.json"}, nil, "", "rg0021.json: not a representative game's run"},
		{"letters", []string{"rg00a.json"}, nil, "", "rg00a.json: not a representative game's run"},
		{"year 0", []string{"rg002-y0.json"}, nil, "", "rg002-y0.json: not a representative game's run"},
	} {
		dir := t.TempDir()
		for _, f := range append(append([]string{}, base...), c.add...) {
			if !contains(c.remove, f) {
				if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		games, err := RGGames(dir)
		got := strings.ReplaceAll(fmt.Sprint(games), dir+string(filepath.Separator), "")
		switch {
		case c.wantErr == "" && (err != nil || got != c.want):
			t.Errorf("%s: %s, %v; want %s", c.name, got, err, c.want)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: %s, %v; want the error %q", c.name, got, err, c.wantErr)
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// A new game's ledger starts as its header alone: its first difference
// fails, a computed line as a new line and a deterministic one also as
// UNCLASSIFIED, which still fails once written, until someone classifies
// it.
func TestRGNewGameLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rg005-ledger.txt")
	if err := os.WriteFile(path, []byte("# RG-005 discrepancy ledger\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := readLedger(t, path)
	fixed := ledgerLine{key: "rg005 | c y1 #0 | o | fixed"}
	sample := ledgerLine{class: "sample", key: "rg005 | c y2 #0 | o | sample"}
	for _, c := range []struct {
		name   string
		got    []ledgerLine
		ledger map[string]string
		errors int
	}{
		{"no difference", nil, empty, 0},
		{"deterministic line", []ledgerLine{fixed}, empty, 1},
		{"computed line", []ledgerLine{sample}, empty, 1},
		{"deterministic line written", []ledgerLine{fixed}, map[string]string{fixed.key: rg002Unclassified}, 1},
		{"deterministic line classified", []ledgerLine{fixed}, map[string]string{fixed.key: "missing mechanic [lane D]"}, 0},
	} {
		e := &errorCounter{TB: t}
		write := checkLedger(e, c.got, c.ledger)
		if e.errors != c.errors {
			t.Errorf("%s: %d errors, want %d", c.name, e.errors, c.errors)
		}
		if c.name == "deterministic line" && (len(write) != 1 || write[0].class != rg002Unclassified) {
			t.Errorf("%s: writes %+v, want it UNCLASSIFIED", c.name, write)
		}
	}
}

// TestRG00xLedger runs every game without a test of its own, and a
// rewrite keeps a ledger's leading comment lines as its header.
func TestRGOtherGames(t *testing.T) {
	ledgers := map[string]rgLedger{"rg002": {}, "rg003": {}, "rg004": {}, "rg006": {}, "rg005": {}}
	if got := fmt.Sprint(rgOtherGames(ledgers)); got != "[rg005 rg006]" {
		t.Errorf("rgOtherGames = %s, want [rg005 rg006]", got)
	}
	for data, want := range map[string]string{
		"":                          "",
		"# a\n# b\n":                "# a\n# b\n",
		"# a\n#\n# b\nx | y\n# c\n": "# a\n#\n# b\n",
		"x | y\n# a\n":              "",
	} {
		if got := ledgerHeader(data); got != want {
			t.Errorf("ledgerHeader(%q) = %q, want %q", data, got, want)
		}
	}
}
