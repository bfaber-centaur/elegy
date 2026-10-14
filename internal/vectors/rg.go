package vectors

import (
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
	Name   string // the chained run, "rg002"
	Ledger string // the ledger's path under the corpus directory; "" for RG-001
	Years  int    // the restarts' count
}

// rgUnledgered reports whether game has no ledger by design: RG-001
// predates the ledgers, and its vectors are held to exact matches.
func rgUnledgered(game string) bool { return game == "rg001" }

// RGGames lists the representative games in dir, the "rg" corpus
// directory, by name. A game is a chained run rgNNN.json. Its restarts
// must run from y1 with no gap, and its ledger must exist (RG-001 aside),
// so a deleted ledger fails instead of dropping its game. A ledger or a
// restart with no chained run fails too.
func RGGames(dir string) ([]RGGame, error) {
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
	var out []RGGame
	for game := range chained {
		g := RGGame{Name: game, Years: len(years[game])}
		for k := 1; k <= g.Years; k++ {
			if !years[game][k] {
				return nil, fmt.Errorf("%s: restarts %v skip year %d", game, sortedKeys(years[game]), k)
			}
		}
		switch {
		case ledgers[game] && rgUnledgered(game):
			return nil, fmt.Errorf("%s: has a ledger, but is held to exact matches", game)
		case ledgers[game]:
			g.Ledger = filepath.Join(dir, game+"-ledger.txt")
		case !rgUnledgered(game):
			return nil, fmt.Errorf("%s: no ledger %s-ledger.txt", game, game)
		}
		out = append(out, g)
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
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
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
