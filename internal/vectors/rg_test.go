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
// instead of dropping or shrinking a game: a deleted ledger, game or
// trailing restart, and a count that disagrees with its ledger header.
func TestRGGames(t *testing.T) {
	games, err := RGGames(rgCorpus)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(games), fmt.Sprintf("[{rg001  6 0} {rg002 %[1]s/rg002-ledger.txt 6 88} {rg003 %[1]s/rg003-ledger.txt 12 204} {rg004 %[1]s/rg004-ledger.txt 12 110}]", rgCorpus); got != want {
		t.Errorf("RGGames(%s) = %s, want %s", rgCorpus, got, want)
	}
	const (
		battle    = `{"cases": [{"expect": [{"kind": "battle"}, {"kind": "battle_actions"}]}]}`
		header    = "# RG-002\n# restarts: 2\n# battle records: 2\n"
		gamesFile = "# the gamesFile\n\nrg001 no-ledger\nrg002\n"
		withNew   = gamesFile + "rg005\n"
		newGame   = "# restarts: 1\n# battle records: 0\n"
	)
	base := map[string]string{
		"GAMES": gamesFile, "README.md": "",
		"rg001.json": "", "rg001-y1.json": "",
		"rg002.json": battle, "rg002-y1.json": battle, "rg002-y2.json": "{}", "rg002-ledger.txt": header + "sample | x\n",
	}
	for _, c := range []struct {
		name          string
		set           map[string]string // files added or rewritten
		remove        []string
		want, wantErr string
	}{
		{"as is", nil, nil, "[{rg001  1 0} {rg002 rg002-ledger.txt 2 2}]", ""},
		{"new game", map[string]string{"GAMES": withNew, "rg005.json": "{}", "rg005-y1.json": "{}", "rg005-ledger.txt": newGame},
			nil, "[{rg001  1 0} {rg002 rg002-ledger.txt 2 2} {rg005 rg005-ledger.txt 1 0}]", ""},
		{"new game, no restarts", map[string]string{"GAMES": withNew, "rg005.json": battle, "rg005-ledger.txt": "# restarts: 0\n# battle records: 1\n"},
			nil, "[{rg001  1 0} {rg002 rg002-ledger.txt 2 2} {rg005 rg005-ledger.txt 0 1}]", ""},
		{"new game, not in GAMES", map[string]string{"rg005.json": "{}", "rg005-y1.json": "{}", "rg005-ledger.txt": newGame}, nil, "", "rg005.json: rg005 is not in GAMES"},
		{"game deleted", nil, []string{"rg002.json", "rg002-y1.json", "rg002-y2.json", "rg002-ledger.txt"}, "", "GAMES lists rg002, but there is no rg002.json"},
		{"ledger deleted", nil, []string{"rg002-ledger.txt"}, "", "rg002: no ledger rg002-ledger.txt"},
		{"trailing restart deleted", nil, []string{"rg002-y2.json"}, "", "rg002: 1 restarts on disk, its ledger header pins 2"},
		{"restart added", map[string]string{"rg002-y3.json": "{}"}, nil, "", "rg002: 3 restarts on disk, its ledger header pins 2"},
		{"battle record lost", map[string]string{"rg002-y1.json": "{}"}, nil, "", "rg002: 1 battle records in its runs, its ledger header pins 2"},
		{"header count wrong", map[string]string{"rg002-ledger.txt": "# restarts: 2\n# battle records: 3\n"}, nil, "", "rg002: 2 battle records in its runs, its ledger header pins 3"},
		{"header without restarts", map[string]string{"rg002-ledger.txt": "# battle records: 2\n"}, nil, "", `no "# restarts: N" header line`},
		{"header without records", map[string]string{"rg002-ledger.txt": "# restarts: 2\n"}, nil, "", `no "# battle records: N" header line`},
		{"count below the header", map[string]string{"rg002-ledger.txt": "# restarts: 2\nsample | x\n# battle records: 2\n"}, nil, "", `no "# battle records: N" header line`},
		{"count twice", map[string]string{"rg002-ledger.txt": header + "# restarts: 2\n"}, nil, "", "header pins restarts twice"},
		{"count not a number", map[string]string{"rg002-ledger.txt": "# restarts: two\n# battle records: 2\n"}, nil, "", `header "# restarts: two"`},
		{"broken run", map[string]string{"rg002-y2.json": "{"}, nil, "", "rg002-y2.json: unexpected end of JSON input"},
		{"no GAMES", nil, []string{"GAMES"}, "", "GAMES: no such file"},
		{"GAMES lists a game twice", map[string]string{"GAMES": gamesFile + "rg002\n"}, nil, "", "GAMES:5: rg002 listed twice"},
		{"GAMES with a bad name", map[string]string{"GAMES": gamesFile + "rg02\n"}, nil, "", `GAMES:5: "rg02" is not a game`},
		{"GAMES with a bad marker", map[string]string{"GAMES": gamesFile + "rg005 strict\n"}, nil, "", `GAMES:5: "rg005 strict" is not a game`},
		{"GAMES with a third field", map[string]string{"GAMES": gamesFile + "rg005 no-ledger x\n"}, nil, "", `GAMES:5: "rg005 no-ledger x" is not a game`},
		{"RG-001 with a ledger", map[string]string{"rg001-ledger.txt": newGame}, nil, "", "rg001: has a ledger, but GAMES holds it to exact matches"},
		{"restart gap", map[string]string{"rg002-y4.json": "{}"}, nil, "", "rg002: restarts [1 2 4] skip year 3"},
		{"restarts from y2", nil, []string{"rg002-y1.json"}, "", "rg002: restarts [2] skip year 1"},
		{"chained run deleted", nil, []string{"rg002.json"}, "", "no chained run rg002.json"},
		{"ledger alone", map[string]string{"GAMES": withNew, "rg005-ledger.txt": newGame}, nil, "", "rg005-ledger.txt: no chained run rg005.json"},
		{"restarts alone", map[string]string{"rg005-y1.json": ""}, nil, "", "rg005: restarts with no chained run rg005.json"},
		{"stray run", map[string]string{"rg002-y1b.json": ""}, nil, "", "rg002-y1b.json: not a representative game's run"},
		{"four digits", map[string]string{"rg0021.json": ""}, nil, "", "rg0021.json: not a representative game's run"},
		{"letters", map[string]string{"rg00a.json": ""}, nil, "", "rg00a.json: not a representative game's run"},
		{"year 0", map[string]string{"rg002-y0.json": ""}, nil, "", "rg002-y0.json: not a representative game's run"},
	} {
		dir := t.TempDir()
		files := map[string]string{}
		for f, data := range base {
			files[f] = data
		}
		for f, data := range c.set {
			files[f] = data
		}
		for _, f := range c.remove {
			delete(files, f)
		}
		for f, data := range files {
			if err := os.WriteFile(filepath.Join(dir, f), []byte(data), 0o644); err != nil {
				t.Fatal(err)
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
