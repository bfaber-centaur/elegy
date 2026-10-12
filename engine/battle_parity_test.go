package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Battle-record parity: the deterministic-mechanics level of combat
// parity, below exact replay of the original's draws. The vectors'
// `battle` and `battle_actions` expectations (vectors README "cases" and
// "Combat vectors") hold the original's battle record: the token list in
// shuffled order with each token's values, and every move and shot. A
// battle's draws (jitter, shuffle, movement ties, torpedo rolls; COMBAT.md
// "Random draws in a battle") cannot be replayed without the original's
// generator, so this test takes them from the record instead:
//
//   - tokens: Elegy builds the battle at the record's location from the
//     vector's starting state, then its tokens are matched to the
//     record's by owner, fleet or planet, and design. Their values are
//     compared, and Elegy's tokens are put in the record's order (the
//     shuffle's outcome).
//   - hits: round by round, every token is put on the square the record
//     moved it to (movement's outcome), then Elegy fires. Each hit is
//     compared in order: firer, target, kills, shield damage and the
//     target's damage afterwards. A difference fails the test; the
//     battle's comparison ends there, since later hits depend on it. It
//     also ends, without failing, at a round where Elegy would draw (a
//     torpedo salvo of at most 200 that can miss).
//
// The `flags` of a hit are not compared: Elegy does not model them.
// Movement itself is not compared: it depends on the jitter draws.

// pvBattleDamage is a stack's damage as the record shows it.
type pvBattleDamage struct {
	Units int `json:"units"`
	Pct   int `json:"percent_of_ships"`
}

type pvBattleToken struct {
	Owner       int            `json:"owner"`
	Kind        string         `json:"kind"`
	Design      int            `json:"design"`
	StartSquare [2]int         `json:"start_square"`
	Initiative  int            `json:"initiative"`
	Jammer      int            `json:"jammer"`
	Computer    int            `json:"computer"`
	Capacitor   int            `json:"capacitor"`
	Deflector   int            `json:"deflector"`
	Mass        int            `json:"mass"`
	Shield      int            `json:"shield"`
	Ships       int            `json:"ships"`
	Damage      pvBattleDamage `json:"damage"`
	Fleet       *int           `json:"fleet"`
	Planet      *int           `json:"planet"`
}

type pvBattleHit struct {
	Token        int            `json:"token"`
	Kills        int            `json:"kills"`
	ShieldDamage int            `json:"shield_damage"`
	ArmorDamage  pvBattleDamage `json:"armor_damage"`
}

type pvBattleAction struct {
	Round int           `json:"round"`
	Token int           `json:"token"`
	To    *[2]int       `json:"to"`
	Hits  []pvBattleHit `json:"hits"`
}

// pvBattleExpect is a `battle` or `battle_actions` expectation.
type pvBattleExpect struct {
	Kind    string           `json:"kind"`
	Year    int              `json:"year"`
	Stream  string           `json:"stream"`
	X       int              `json:"x"`
	Y       int              `json:"y"`
	Tokens  []pvBattleToken  `json:"tokens"`
	Actions []pvBattleAction `json:"actions"`
}

// pvRecordedBattle is one battle record: its tokens and actions.
type pvRecordedBattle struct {
	vector, caseID, stream string
	x, y                   int
	tokens                 []pvBattleToken
	actions                []pvBattleAction
	hasActions             bool
}

// BattleStart is a vector's starting state for this test, with how the
// vector's fleet and planet ids map to Elegy's.
type BattleStart struct {
	Game    Game
	Fleets  map[[2]int]int // (owner, vector fleet id) → Elegy fleet ID
	Planets map[int]int    // vector planet id → Elegy planet ID
	Skip    string         // why every case of the vector is skipped, if one is
}

// pvBattleOutcome is what the comparison found for one recorded battle.
type pvBattleOutcome struct {
	tokens string // "" when the token lists match, else the first difference
	// hits compared and matched, in record order, before the battle
	// stopped.
	compared, matched int
	total             int    // hits in the record
	stop              string // why the replay stopped early, if it did
	draws             bool   // stopped at a round where Elegy would draw
	built             bool   // a token's fleet was built this year: not compared
}

// loadBattleRecords reads every battle record in the vectors.
func loadBattleRecords(t *testing.T) map[string][]pvRecordedBattle {
	paths, err := filepath.Glob(filepath.Join("testdata", "vectors", "*", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no vectors: %v", err)
	}
	sort.Strings(paths)
	out := map[string][]pvRecordedBattle{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Cases []struct {
				ID     string           `json:"id"`
				Expect []pvBattleExpect `json:"expect"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		name := filepath.Base(filepath.Dir(p)) + "/" + filepath.Base(p)
		for _, c := range v.Cases {
			first := len(out[name])
			at := map[string]int{} // stream and location → index in out[name]
			for _, e := range c.Expect {
				if e.Kind != "battle" && e.Kind != "battle_actions" {
					continue
				}
				key := fmt.Sprintf("%s %d %d", e.Stream, e.X, e.Y)
				i, ok := at[key]
				if !ok {
					i = len(out[name])
					at[key] = i
					out[name] = append(out[name], pvRecordedBattle{vector: name, caseID: c.ID, stream: e.Stream, x: e.X, y: e.Y})
				}
				r := &out[name][i]
				if e.Kind == "battle" {
					r.tokens = e.Tokens
				} else {
					r.actions, r.hasActions = e.Actions, true
				}
			}
			// A token list the same in every stream is written once,
			// without a stream (vectors README "Randomness"); each stream's
			// actions use it.
			for i := first; i < len(out[name]); i++ {
				r := &out[name][i]
				if r.tokens == nil {
					if j, ok := at[fmt.Sprintf(" %d %d", r.x, r.y)]; ok {
						r.tokens = out[name][j].tokens
					}
				}
			}
		}
	}
	return out
}

// pvDrawFlag is a Rand for the hit replay: any draw marks the round as
// needing the original's draws.
type pvDrawFlag struct{ drew *bool }

func (r pvDrawFlag) Intn(int) int { *r.drew = true; return 0 }

// pvNoDraws answers every setup draw with 0, which keeps the token order
// the setup made (the record's order is applied afterwards).
type pvNoDraws struct{}

func (pvNoDraws) Intn(int) int { return 0 }

// setupBattle builds Elegy's battle at (x, y) from the starting game, as
// battles() does, without fighting it. ok is false when no battle happens
// there.
func setupBattle(g *Game, x, y int) (*battle, bool) {
	var prev locationHistory
	for _, loc := range g.locations() {
		sets, inv, n := g.whoFights(loc, prev)
		prev = locationHistory{any: true, battle: inv != nil, lastOwner: g.Fleets[loc.fleets[len(loc.fleets)-1]].Owner}
		if loc.pos != (Point{X: x, Y: y}) {
			continue
		}
		if inv == nil {
			return nil, false
		}
		b := &battle{g: g, rng: pvNoDraws{}, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
		b.setup(map[int]bool{})
		return b, true
	}
	return nil, false
}

// pvRecordDamage is a token's damage as the record writes it: an undamaged
// stack (no units) shows 0%.
func pvRecordDamage(d Damage) pvBattleDamage {
	if d.Units == 0 {
		return pvBattleDamage{}
	}
	return pvBattleDamage{Units: d.Units, Pct: d.Pct}
}

// pvRecordCapacitor is a capacitor % as the record writes it: 0 for none.
func pvRecordCapacitor(c int) int {
	if c == 100 {
		return 0
	}
	return c
}

// matchTokens puts b's tokens in the record's order and compares their
// values. It returns the first difference, or "".
func (l *BattleStart) matchTokens(b *battle, rec []pvBattleToken) string {
	g := b.g
	key := func(owner int, starbase bool, slot, obj int) string {
		return fmt.Sprintf("%d %t %d %d", owner, starbase, slot, obj)
	}
	mine := map[string]*token{}
	for _, t := range b.tokens {
		var slot, obj int
		for _, s := range g.DesignSlots {
			if s.Design == t.design {
				slot = s.Slot
				if s.Starbase {
					slot += 16 // the record numbers starbase designs 16 and up
				}
			}
		}
		if t.starbase {
			obj = g.Planets[t.planet].ID
		} else {
			obj = g.Fleets[t.fleet].ID
		}
		mine[key(t.player, t.starbase, slot, obj)] = t
	}
	if len(mine) != len(b.tokens) {
		return "two Elegy tokens share a key"
	}
	var order []*token
	for i, r := range rec {
		var obj int
		switch {
		case r.Kind == "starbase" && r.Planet != nil:
			obj = l.Planets[*r.Planet]
		case r.Kind == "ships" && r.Fleet != nil:
			obj = l.Fleets[[2]int{r.Owner, *r.Fleet}]
		default:
			return fmt.Sprintf("token %d: kind %q", i, r.Kind)
		}
		t := mine[key(r.Owner, r.Kind == "starbase", r.Design, obj)]
		if t == nil {
			return fmt.Sprintf("token %d (owner %d, design %d): not among Elegy's %d tokens", i, r.Owner, r.Design, len(b.tokens))
		}
		got := pvBattleToken{
			Owner: r.Owner, Kind: r.Kind, Design: r.Design, Fleet: r.Fleet, Planet: r.Planet,
			StartSquare: [2]int{t.x, t.y}, Initiative: t.initiative, Jammer: t.jammer,
			Computer: t.computer, Capacitor: pvRecordCapacitor(t.capacitor), Deflector: t.deflector,
			Mass: t.mass, Shield: t.shield, Ships: t.ships, Damage: pvRecordDamage(t.dmg),
		}
		if got != r {
			return fmt.Sprintf("token %d: Elegy %+v, record %+v", i, got, r)
		}
		order = append(order, t)
	}
	if len(order) != len(b.tokens) {
		return fmt.Sprintf("Elegy has %d tokens, the record %d", len(b.tokens), len(rec))
	}
	b.tokens = order
	return ""
}

// replayHits fires b round by round with the record's moves and compares
// the hits.
func replayHits(b *battle, actions []pvBattleAction, out *pvBattleOutcome) {
	byRound := map[int][]pvBattleAction{}
	last := 0
	for _, a := range actions {
		byRound[a.Round] = append(byRound[a.Round], a)
		last = max(last, a.Round)
	}
	drew := false
	b.rng = pvDrawFlag{&drew}
	for r := 0; r <= last; r++ {
		b.round = r
		if r > 0 {
			b.regenerate()
		}
		// The round's moves come first, one action per square moved; the
		// firing actions follow, each with its hits. In every record a
		// round's actions without hits all come before those with hits.
		var fires []pvBattleAction
		for i, a := range byRound[r] {
			if len(a.Hits) > 0 {
				fires = byRound[r][i:]
				break
			}
			if a.Token < 0 || a.Token >= len(b.tokens) {
				out.stop = fmt.Sprintf("round %d: token %d out of range", r, a.Token)
				return
			}
			t := b.tokens[a.Token]
			if a.To == nil {
				t.left = true
			} else {
				t.x, t.y = a.To[0], a.To[1]
			}
		}
		var rec []pvBattleHit
		var firer []int
		for _, a := range fires {
			if a.To != nil {
				if t := b.tokens[a.Token]; t.x != a.To[0] || t.y != a.To[1] {
					out.stop = fmt.Sprintf("round %d: token %d fires from %v, but its move left it on (%d,%d)", r, a.Token, *a.To, t.x, t.y)
					return
				}
			}
			for _, h := range a.Hits {
				rec = append(rec, h)
				firer = append(firer, a.Token)
			}
		}
		n := len(b.hits)
		b.fire()
		got := b.hits[n:]
		if drew {
			out.draws = true
			out.stop = fmt.Sprintf("round %d: Elegy draws (torpedo rolls)", r)
			return
		}
		for i := range max(len(got), len(rec)) {
			if i >= len(got) {
				out.stop = fmt.Sprintf("round %d hit %d: record has token %d hitting %d, Elegy has no further hit", r, i, firer[i], rec[i].Token)
				return
			}
			h := got[i]
			if i >= len(rec) {
				out.stop = fmt.Sprintf("round %d hit %d: Elegy has token %d hitting %d, the record has no further hit", r, i, h.Firer, h.Target)
				return
			}
			out.compared++
			w := rec[i]
			gotHit := fmt.Sprintf("token %d hits %d: kills %d, shield %d, damage %v", h.Firer, h.Target, h.Kills, h.Shield, pvRecordDamage(h.After))
			wantHit := fmt.Sprintf("token %d hits %d: kills %d, shield %d, damage %v", firer[i], w.Token, w.Kills, w.ShieldDamage, w.ArmorDamage)
			if gotHit != wantHit {
				out.stop = fmt.Sprintf("round %d hit %d: Elegy %s; record %s", r, i, gotHit, wantHit)
				return
			}
			out.matched++
		}
	}
}

// compareBattle runs the comparison for one recorded battle.
func compareBattle(l *BattleStart, r pvRecordedBattle) pvBattleOutcome {
	var out pvBattleOutcome
	for _, a := range r.actions {
		out.total += len(a.Hits)
	}
	g := withRules(l.Game.clone()) // the battle changes fleets and planets
	for _, rt := range r.tokens {
		if rt.Kind == "ships" && rt.Fleet != nil {
			if _, ok := l.Fleets[[2]int{rt.Owner, *rt.Fleet}]; !ok {
				// Production comes before battles (COMBAT.md
				// "Starbases"), so a fleet built this year fights; this
				// test starts from the year's start.
				out.built = true
				return out
			}
		}
	}
	b, ok := setupBattle(&g, r.x, r.y)
	if !ok {
		out.tokens = "Elegy has no battle here at the start of the year"
		return out
	}
	if why := l.matchTokens(b, r.tokens); why != "" {
		out.tokens = why
		return out
	}
	if r.hasActions {
		replayHits(b, r.actions, &out)
	}
	return out
}

// BattleRecordVectors lists the vectors holding battle records, by path
// ("cb/cb010.json").
func BattleRecordVectors(t *testing.T) []string {
	var names []string
	for name := range loadBattleRecords(t) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CheckBattleRecords compares Elegy's battles with every recorded battle,
// given the record's token order and moves (see the top of this file),
// from each vector's start in starts, by path. Any difference in a token
// list or a compared hit fails. A round that needs the original's draws
// ends that battle's comparison without failing: that is the random level
// (docs/RANDOM-PARITY.md). TestBattleRecords (battle_vectors_test.go)
// runs it: the vector loader, package vectors, imports engine.
func CheckBattleRecords(t *testing.T, starts map[string]BattleStart) {
	records := loadBattleRecords(t)
	type tally struct {
		records, tokens, built, full, draws int
		compared, matched, total            int
	}
	per := map[string]*tally{}
	var names []string
	for name := range records {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		corpus := strings.Split(name, "/")[0]
		if per[corpus] == nil {
			per[corpus] = &tally{}
		}
		c := per[corpus]
		l, ok := starts[name]
		var err error
		switch {
		case !ok:
			err = fmt.Errorf("no start")
		case l.Skip != "":
			err = fmt.Errorf("%s", l.Skip)
		}
		for _, r := range records[name] {
			where := fmt.Sprintf("%s %s %q (%d,%d)", name, r.caseID, r.stream, r.x, r.y)
			c.records++
			for _, a := range r.actions {
				c.total += len(a.Hits)
			}
			if err != nil {
				t.Errorf("%s: load: %v", where, err)
				continue
			}
			o := compareBattle(&l, r)
			switch {
			case o.built:
				c.built++
				continue
			case o.tokens != "":
				t.Errorf("%s: tokens: %s", where, o.tokens)
				continue
			}
			c.tokens++
			c.compared += o.compared
			c.matched += o.matched
			switch {
			case o.draws:
				c.draws++
			case o.stop != "":
				t.Errorf("%s: hits: %s", where, o.stop)
			case r.hasActions:
				c.full++
			}
		}
	}
	var corpora []string
	for c := range per {
		corpora = append(corpora, c)
	}
	sort.Strings(corpora)
	t.Logf("%-5s %8s %8s %8s %8s %8s  %s", "", "records", "tokens", "built", "replayed", "draws", "hits matched/compared of recorded")
	var all tally
	for _, name := range corpora {
		c := per[name]
		t.Logf("%-5s %8d %8d %8d %8d %8d  %d/%d of %d", name, c.records, c.tokens, c.built, c.full, c.draws, c.matched, c.compared, c.total)
		all.compared += c.compared
	}
	if all.compared == 0 {
		t.Error("no hit was compared")
	}
}

// CheckBattleRecordHarness checks that the comparison notices a
// difference: a recorded token value or hit changed by one, and a round
// with draws. l is CB-010's start (cb/cb010.json), lb CS-003-C's
// (cs/cs003c.json).
func CheckBattleRecordHarness(t *testing.T, l, lb BattleStart) {
	records := loadBattleRecords(t)
	var r pvRecordedBattle
	for _, x := range records["cb/cb010.json"] {
		if x.x == 1060 {
			r = x
		}
	}
	if o := compareBattle(&l, r); o.tokens != "" || o.stop != "" || o.compared == 0 || o.compared != o.total {
		t.Fatalf("CB-010 at 1060: %+v, want every hit replayed", o)
	}
	mass := r
	mass.tokens = append([]pvBattleToken(nil), r.tokens...)
	mass.tokens[3].Mass++
	if o := compareBattle(&l, mass); o.tokens == "" {
		t.Error("a changed token mass was not noticed")
	}
	kills := r
	kills.actions = append([]pvBattleAction(nil), r.actions...)
	for i, a := range kills.actions {
		if len(a.Hits) > 0 {
			a.Hits = append([]pvBattleHit(nil), a.Hits...)
			a.Hits[0].Kills++
			kills.actions[i] = a
			break
		}
	}
	if o := compareBattle(&l, kills); o.stop == "" || o.draws {
		t.Errorf("a changed kill count was not noticed: %+v", o)
	}
	drawn := false
	for _, x := range records["cs/cs003c.json"] {
		drawn = drawn || compareBattle(&lb, x).draws
	}
	if !drawn {
		t.Error("no CS-003-C torpedo battle stopped at its draws")
	}
}
