package vectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestComputerLevel pins the computer_level mapping (vectors/README.md
// "Computer level": easy 0 to expert 3, engine.Player.Level) and the
// explicit skip when a computer player has no level: RG-005 Y2 loads two
// Traders, so player 1's planet 218, its player record and every Trader
// depend on whether player 1 makes a planet trade.
func TestComputerLevel(t *testing.T) {
	const path = "../../engine/testdata/vectors/rg/rg005-y2.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	load := func(level string) (*pvLoaded, error) {
		t.Helper()
		b := string(raw)
		if n := strings.Count(b, `"computer_level": "expert"`); n != 5 {
			t.Fatalf("%d computer_level keys, want 5", n)
		}
		// Player 1's is the second.
		i := strings.Index(b, `"computer_level": "expert"`)
		i += 1 + strings.Index(b[i+1:], `"computer_level": "expert"`)
		b = b[:i] + level + b[i+len(`"computer_level": "expert"`):]
		p := filepath.Join(t.TempDir(), "rg", "rg005-y2.json")
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
		v, err := ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return loadVector(v)
	}
	for level, want := range map[string]int{`"computer_level": "easy"`: 0, `"computer_level": "standard"`: 1, `"computer_level": "harder"`: 2, `"computer_level": "expert"`: 3} {
		l, err := load(level)
		if err != nil {
			t.Fatal(err)
		}
		if got := l.g.Players[1].Level; got != want || len(l.noLevel) != 0 {
			t.Errorf("%s: Level %d, no level %v; want %d", level, got, l.noLevel, want)
		}
		if l.g.Players[0].Level != 3 {
			t.Errorf("%s: player 0 Level %d, want 3", level, l.g.Players[0].Level)
		}
	}
	if _, err := load(`"computer_level": "expert2"`); err == nil || !strings.Contains(err.Error(), "computer_level") {
		t.Errorf("unknown level: %v, want a load error", err)
	}

	// No level: the level-dependent expectations are skipped with the
	// reason, the others are compared.
	l, err := load(`"computer_level_absent": 0`)
	if err != nil {
		t.Fatal(err)
	}
	if !l.noLevel[1] || l.g.Players[1].Level != 0 {
		t.Fatalf("no level: %v, Level %d", l.noLevel, l.g.Players[1].Level)
	}
	v, _ := ReadFile(path)
	g := l.g
	skip := "skip: " + pvNoLevelWhy
	n := map[string]int{}
	for _, c := range v.Cases {
		for _, e := range c.Expect {
			if e.Year != 1 {
				continue
			}
			got := l.pvCheck(&g, e)
			var owner int
			switch {
			case pvTraderExpect(e):
				n["trader"]++
				if got != skip {
					t.Errorf("trader expectation: %q, want %q", got, skip)
				}
				continue
			case e.Kind == "planet" && e.ID != nil:
				owner = l.startOwner(*e.ID)
			case e.Kind == "player":
				owner = *e.ID
			case e.Kind == "production_queue" && e.Planet != nil:
				owner = l.startOwner(*e.Planet)
			default:
				continue
			}
			if (owner == 1) != (got == skip) {
				t.Errorf("%s of player %d: %q", e.Kind, owner, got)
			}
			if owner == 1 {
				n[e.Kind]++
			}
		}
	}
	if n["trader"] == 0 || n["planet"] == 0 || n["player"] == 0 {
		t.Errorf("skipped %v; want Traders, planets and the player record", n)
	}
	// Player 1's planet 218 (Y2 #121) has no queue expectation here.
	id := 218
	if o := l.startOwner(id); o != 1 {
		t.Fatalf("planet 218 owner %d, want 1", o)
	}
	if got := l.queueEquals(&g, pvExpect{Kind: "production_queue", Year: 1, Planet: &id}); got != skip {
		t.Errorf("player 1's queue: %q, want %q", got, skip)
	}
	// With no Trader in the run, nothing depends on the level.
	l.traders = nil
	if l.levelUnknown(1, 1) || l.levelUnknown(-2, 1) {
		t.Error("level unknown with no Trader")
	}
}
