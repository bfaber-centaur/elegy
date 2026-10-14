package vectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RGGame is one representative game of the "rg" corpus: its chained run
// rgNNN.json, its restarts rgNNN-y1.json to rgNNN-yN.json and its
// discrepancy ledger rgNNN-ledger.txt.
type RGGame struct {
	Name          string // the chained run, "rg002"
	Ledger        string // the ledger's path under the corpus directory; "" for a game without one
	Years         int    // the restarts' count
	BattleRecords int    // the `battle` expectations in its runs, as its ledger header pins them
}

// rgNoLedger marks a game in GAMES that has no ledger and is held to
// exact matches (RG-001, which predates the ledgers).
const rgNoLedger = "no-ledger"

// RGGames lists the representative games in dir, the "rg" corpus
// directory, by name. The games are the ones the hand-kept list
// dir/GAMES names, one per line ("rg002", or "rg001 no-ledger"), and
// each is a chained run rgNNN.json with restarts from y1 and no gap.
//
// It fails, rather than drop or shrink a game, when a listed game's
// chained run or ledger is missing, when a run, restart or ledger has no
// listed game, and when a ledger's hand-kept header counts ("# restarts:
// N", "# battle records: N") disagree with the restarts on disk or the
// `battle` expectations in the runs' JSON.
func RGGames(dir string) ([]RGGame, error) {
	listed, err := rgListed(filepath.Join(dir, "GAMES"))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	chained := map[string]bool{}
	ledgers := map[string]bool{}
	years := map[string]map[int]bool{}
	for _, e := range entries {
		name := e.Name()
		switch {
		case rgName(strings.TrimSuffix(name, ".json")) && strings.HasSuffix(name, ".json"):
			chained[strings.TrimSuffix(name, ".json")] = true
		case strings.HasSuffix(name, "-ledger.txt") && rgName(strings.TrimSuffix(name, "-ledger.txt")):
			ledgers[strings.TrimSuffix(name, "-ledger.txt")] = true
		case strings.HasSuffix(name, ".json"):
			game, year, ok := strings.Cut(strings.TrimSuffix(name, ".json"), "-y")
			k, err := strconv.Atoi(year)
			if !ok || !rgName(game) || err != nil || k < 1 {
				return nil, fmt.Errorf("%s: not a representative game's run (rgNNN.json, rgNNN-yK.json)", name)
			}
			if years[game] == nil {
				years[game] = map[int]bool{}
			}
			years[game][k] = true
		}
	}
	for game := range chained {
		if _, ok := listed[game]; !ok {
			return nil, fmt.Errorf("%s.json: %s is not in GAMES", game, game)
		}
	}
	for game := range ledgers {
		if !chained[game] {
			return nil, fmt.Errorf("%s-ledger.txt: no chained run %s.json", game, game)
		}
	}
	for game := range years {
		if !chained[game] {
			return nil, fmt.Errorf("%s: restarts with no chained run %s.json", game, game)
		}
	}
	var out []RGGame
	for game, noLedger := range listed {
		if !chained[game] {
			return nil, fmt.Errorf("GAMES lists %s, but there is no %s.json", game, game)
		}
		g := RGGame{Name: game, Years: len(years[game])}
		for k := 1; k <= g.Years; k++ {
			if !years[game][k] {
				return nil, fmt.Errorf("%s: restarts %v skip year %d", game, sortedKeys(years[game]), k)
			}
		}
		switch {
		case noLedger && ledgers[game]:
			return nil, fmt.Errorf("%s: has a ledger, but GAMES holds it to exact matches", game)
		case noLedger:
			out = append(out, g)
			continue
		case !ledgers[game]:
			return nil, fmt.Errorf("%s: no ledger %s-ledger.txt", game, game)
		}
		g.Ledger = filepath.Join(dir, game+"-ledger.txt")
		restarts, records, err := rgHeaderCounts(g.Ledger)
		if err != nil {
			return nil, err
		}
		if restarts != g.Years {
			return nil, fmt.Errorf("%s: %d restarts on disk, its ledger header pins %d", game, g.Years, restarts)
		}
		if g.BattleRecords, err = rgBattleRecords(dir, game, g.Years); err != nil {
			return nil, err
		}
		if records != g.BattleRecords {
			return nil, fmt.Errorf("%s: %d battle records in its runs, its ledger header pins %d", game, g.BattleRecords, records)
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// rgListed reads GAMES: each game, and whether it has no ledger.
// Blank lines and # comments are skipped.
func rgListed(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for i, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 0 || strings.HasPrefix(f[0], "#"):
			continue
		case !rgName(f[0]) || len(f) > 2 || len(f) == 2 && f[1] != rgNoLedger:
			return nil, fmt.Errorf("%s:%d: %q is not a game (rgNNN, then %s or nothing)", path, i+1, line, rgNoLedger)
		}
		if _, ok := out[f[0]]; ok {
			return nil, fmt.Errorf("%s:%d: %s listed twice", path, i+1, f[0])
		}
		out[f[0]] = len(f) == 2
	}
	return out, nil
}

// rgHeaderCounts reads the hand-kept counts in a ledger's leading
// comment lines, which a ledger rewrite keeps and never writes.
func rgHeaderCounts(ledger string) (restarts, records int, err error) {
	data, err := os.ReadFile(ledger)
	if err != nil {
		return 0, 0, err
	}
	counts := map[string]int{}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "#") {
			break
		}
		for _, key := range []string{"restarts", "battle records"} {
			if v, ok := strings.CutPrefix(line, "# "+key+": "); ok {
				if _, dup := counts[key]; dup {
					return 0, 0, fmt.Errorf("%s: header pins %s twice", ledger, key)
				}
				if counts[key], err = strconv.Atoi(v); err != nil {
					return 0, 0, fmt.Errorf("%s: header %q: %v", ledger, line, err)
				}
			}
		}
	}
	for _, key := range []string{"restarts", "battle records"} {
		if _, ok := counts[key]; !ok {
			return 0, 0, fmt.Errorf("%s: no \"# %s: N\" header line", ledger, key)
		}
	}
	return counts["restarts"], counts["battle records"], nil
}

// rgBattleRecords counts the `battle` expectations in a game's runs from
// their raw JSON, apart from the vectors loader.
func rgBattleRecords(dir, game string, years int) (int, error) {
	n := 0
	for k := 0; k <= years; k++ {
		name := game + ".json"
		if k > 0 {
			name = fmt.Sprintf("%s-y%d.json", game, k)
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return 0, err
		}
		var v struct {
			Cases []struct {
				Expect []struct {
					Kind string `json:"kind"`
				} `json:"expect"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return 0, fmt.Errorf("%s: %v", name, err)
		}
		for _, c := range v.Cases {
			for _, e := range c.Expect {
				if e.Kind == "battle" {
					n++
				}
			}
		}
	}
	return n, nil
}

// rgName reports whether s names a representative game, "rg" and three
// digits.
func rgName(s string) bool {
	if len(s) != 5 || !strings.HasPrefix(s, "rg") {
		return false
	}
	for _, c := range s[2:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func sortedKeys(m map[int]bool) []int {
	var out []int
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
