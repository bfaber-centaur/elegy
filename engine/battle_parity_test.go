package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
//   - tokens: Elegy generates the vector's years with its orders, and
//     builds the battle at the record's location from the game as its
//     year's battle phase found it (after movement and production). Its
//     tokens are matched to the record's by owner, fleet or planet, and
//     design. Their values are
//     compared, and Elegy's tokens are put in the record's order (the
//     shuffle's outcome).
//   - hits: round by round, every token is put on the square the record
//     moved it to (movement's outcome), then Elegy fires. Each hit is
//     compared in order: firer, target, kills, shield damage and the
//     target's damage afterwards. A difference fails the test; the
//     battle's comparison ends there, since later hits depend on it.
//   - torpedo rolls: the only draws while firing. Their outcomes are taken
//     from the record too (pvAnswer, compareBattle). Hits recorded after
//     an answered roll are tallied apart as conditioned: they test the
//     mechanics given those outcomes, not random parity. Any other draw
//     ends the battle's comparison without failing.
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
	year                   int // the expectation's year: 1 is the vector's first
	x, y                   int
	tokens                 []pvBattleToken
	actions                []pvBattleAction
	hasActions             bool
}

// BattleStart is the game a vector's year's battles start from: the game
// as Elegy's battle phase found it that year (GenerateTurnAtBattles), with
// the tech levels the year's design costs use and how the vector's fleet
// and planet ids map to Elegy's.
type BattleStart struct {
	Game       Game
	CostLevels [][NumFields]int
	Fleets     map[[2]int]int // (owner, vector fleet id) → Elegy fleet ID
	Planets    map[int]int    // vector planet id → Elegy planet ID
	Skip       string         // why every case of the vector is skipped, if one is
}

// pvBattleOutcome is what the comparison found for one recorded battle.
type pvBattleOutcome struct {
	tokens string // "" when the token lists match, else the first difference
	// hits compared and matched, in record order, before the battle
	// stopped: compared and matched before any answered torpedo draw,
	// conditioned and condMatched after one. The conditioned hits are not
	// random parity: they rest on draw outcomes taken from the record.
	compared, matched        int
	conditioned, condMatched int
	total                    int    // hits in the record
	stop                     string // why the replay stopped early, if it did
	draws                    bool   // stopped at a round where Elegy would draw
	salvos                   int    // torpedo salvos answered from the record
	built                    bool   // a token's fleet is not in Elegy's game
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
					out[name] = append(out[name], pvRecordedBattle{vector: name, caseID: c.ID, stream: e.Stream, year: e.Year, x: e.X, y: e.Y})
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

// pvAnswer is the Rand of the hit replay. During firing the only
// rand(100) draws are torpedo accuracy rolls, one per torpedo, made
// together for each target a salvo reaches; a hit record always follows,
// so a new hit record or round starts the next group. pvAnswer answers
// group k's draws from script[k]: that many hits first, misses after.
// Past the script it answers misses, counts the group's draws (size) and,
// when the group is over, stops the replay with pvSalvo. Any other draw
// marks the round as needing the original's draws.
type pvAnswer struct {
	b      *battle
	script []int
	drew   bool
	// check compares the hits recorded so far, before each group draws.
	check func()
	// group is the index of the current group, -1 before the first; used
	// counts its draws; at is len(b.hits) and the round when it began.
	group, used, at, round int
	// first is the index in b.hits of the first hit recorded after an
	// answered draw, -1 while there is none.
	first int
}

func (r *pvAnswer) Intn(n int) int {
	if n != 100 {
		r.drew = true
		return 0
	}
	if r.group < 0 || len(r.b.hits) != r.at || r.b.round != r.round {
		r.over()
		r.check()
		r.group, r.used, r.at, r.round = r.group+1, 0, len(r.b.hits), r.b.round
		if r.first < 0 {
			r.first = r.at
		}
	}
	r.used++
	if r.group < len(r.script) && r.used <= r.script[r.group] {
		return 0
	}
	return 99
}

// over stops the replay when the group past the script has drawn.
func (r *pvAnswer) over() {
	if r.group >= len(r.script) {
		panic(pvSalvo{r.used})
	}
}

// pvNoDraws answers every setup draw with 0, which keeps the token order
// the setup made (the record's order is applied afterwards).
type pvNoDraws struct{}

func (pvNoDraws) Intn(int) int { return 0 }

// setupBattle builds Elegy's battle at (x, y) from the game at the battle
// phase, as battles() does, without fighting it. ok is false when no
// battle happens there.
func setupBattle(g *Game, costLevels [][NumFields]int, x, y int) (*battle, bool) {
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
		b := &battle{g: g, rng: pvNoDraws{}, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}, costLevels: costLevels}
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

// pvSalvo stops a replay at a torpedo group the script has no answer
// for, with its n draws (a panic value inside replayHits).
type pvSalvo struct{ n int }

// pvMismatch stops a replay at a hit that differs from the record (a
// panic value inside replayHits).
type pvMismatch struct{ why string }

// replayHits fires b round by round with the record's moves and compares
// the hits. script answers the torpedo groups' draws, in order, with each
// group's number of hits (pvAnswer); need is the group after the
// script's last, when the replay reaches it.
func replayHits(b *battle, actions []pvBattleAction, script []int, out *pvBattleOutcome) (need *pvSalvo) {
	byRound := map[int][]pvBattleAction{}
	last := 0
	for _, a := range actions {
		byRound[a.Round] = append(byRound[a.Round], a)
		last = max(last, a.Round)
	}
	ans := &pvAnswer{b: b, script: script, group: -1, first: -1}
	b.rng = ans
	defer func() {
		switch x := recover().(type) {
		case nil:
		case pvSalvo:
			need = &x
		case pvMismatch:
			out.stop = x.why
		default:
			panic(x)
		}
	}()
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
				return nil
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
					return nil
				}
			}
			for _, h := range a.Hits {
				rec = append(rec, h)
				firer = append(firer, a.Token)
			}
		}
		n := len(b.hits)
		// compare checks the round's hits so far against the record; the
		// round may not be over (whole false) when a group is about to
		// draw. A hit recorded after an answered draw is tallied as
		// conditioned: it depends on outcomes taken from the record.
		compared := 0
		compare := func(whole bool) string {
			got := b.hits[n:]
			for i := compared; i < max(len(got), len(rec)); i++ {
				if i >= len(got) {
					if !whole {
						return ""
					}
					return fmt.Sprintf("round %d hit %d: record has token %d hitting %d, Elegy has no further hit", r, i, firer[i], rec[i].Token)
				}
				h := got[i]
				if i >= len(rec) {
					return fmt.Sprintf("round %d hit %d: Elegy has token %d hitting %d, the record has no further hit", r, i, h.Firer, h.Target)
				}
				cond := ans.first >= 0 && n+i >= ans.first
				compared++
				if cond {
					out.conditioned++
				} else {
					out.compared++
				}
				w := rec[i]
				gotHit := fmt.Sprintf("token %d hits %d: kills %d, shield %d, damage %v", h.Firer, h.Target, h.Kills, h.Shield, pvRecordDamage(h.After))
				wantHit := fmt.Sprintf("token %d hits %d: kills %d, shield %d, damage %v", firer[i], w.Token, w.Kills, w.ShieldDamage, w.ArmorDamage)
				if gotHit != wantHit {
					after := ""
					if cond {
						after = " (after an answered draw)"
					}
					return fmt.Sprintf("round %d hit %d%s: Elegy %s; record %s", r, i, after, gotHit, wantHit)
				}
				if cond {
					out.condMatched++
				} else {
					out.matched++
				}
			}
			return ""
		}
		ans.check = func() {
			if why := compare(false); why != "" {
				panic(pvMismatch{why})
			}
		}
		b.fire()
		if ans.drew {
			out.draws = true
			out.stop = fmt.Sprintf("round %d: Elegy draws", r)
			return nil
		}
		// A group past the script that drew this round ends with it.
		if ans.group >= 0 && ans.round == r {
			ans.over()
		}
		if why := compare(true); why != "" {
			out.stop = why
			return nil
		}
	}
	return nil
}

// pvSalvoBudget bounds the replays one battle's search may run.
const pvSalvoBudget = 20000

// compareBattle runs the comparison for one recorded battle. A torpedo
// group's draws are answered from the record: the search finds numbers of
// hits, one per group, with which every hit matches the record. Two
// numbers can give the same records for a group (a stack destroyed
// either way) and differ later, in what is left for the next target, so
// the search goes back. salvos counts the groups answered.
//
// The hits recorded after the first answered group are tallied apart
// (conditioned, condMatched): they show that some outcome of the draws
// reproduces the record, the mechanics given those outcomes, not that
// Elegy's draws would.
func compareBattle(l *BattleStart, r pvRecordedBattle) pvBattleOutcome {
	var out pvBattleOutcome
	for _, a := range r.actions {
		out.total += len(a.Hits)
	}
	for _, rt := range r.tokens {
		if rt.Kind == "ships" && rt.Fleet != nil {
			if _, ok := l.Fleets[[2]int{rt.Owner, *rt.Fleet}]; !ok {
				// The record names a fleet Elegy's game does not have.
				out.built = true
				return out
			}
		}
	}
	build := func() (*battle, string) {
		g := withRules(l.Game.clone()) // the battle changes fleets and planets
		b, ok := setupBattle(&g, l.CostLevels, r.x, r.y)
		if !ok {
			return nil, "Elegy has no battle here"
		}
		return b, l.matchTokens(b, r.tokens)
	}
	b, why := build()
	if why != "" {
		out.tokens = why
		return out
	}
	if !r.hasActions {
		return out
	}
	// A depth-first search: level k is group k's number of hits, tried
	// fewest first. A level is pushed when the replay passes every
	// earlier group and reaches group k; a mismatch tries the deepest
	// level's next number, going back a level when it has none left.
	type level struct{ hits, size int }
	var stack []level
	var deepest pvBattleOutcome
	for runs := 0; ; runs++ {
		if runs >= pvSalvoBudget {
			deepest.stop = fmt.Sprintf("salvo %d: search budget spent: %s", deepest.salvos, deepest.stop)
			return deepest
		}
		script := make([]int, len(stack))
		for i, v := range stack {
			script[i] = v.hits
		}
		b, _ = build()
		try := out
		try.salvos = len(script)
		need := replayHits(b, r.actions, script, &try)
		if need == nil && (try.stop == "" || try.draws) {
			return try
		}
		if need != nil {
			stack = append(stack, level{0, need.n})
			continue
		}
		if runs == 0 || len(script) > deepest.salvos ||
			len(script) == deepest.salvos && try.compared+try.conditioned > deepest.compared+deepest.conditioned {
			deepest = try
		}
		for len(stack) > 0 && stack[len(stack)-1].hits == stack[len(stack)-1].size {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			if deepest.salvos > 0 {
				deepest.stop = fmt.Sprintf("salvo %d: no numbers of hits fit the record: %s", deepest.salvos-1, deepest.stop)
			}
			return deepest
		}
		stack[len(stack)-1].hits++
	}
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
// given the record's token order and moves (see the top of this file).
// starts holds, by vector path ("cb/cb010.json"), each generated year's
// battle-phase game, indexed by year (index 0 is unused). Any difference
// in a token list or a compared or conditioned hit fails. Torpedo rolls
// are answered from the record; a round with any other draw ends that
// battle's comparison without failing: that is the random level
// (docs/RANDOM-PARITY.md). A record of a year Elegy did
// not generate is counted, not compared; CheckBattleRecords returns that
// count by corpus ("rg"). TestBattleRecords (battle_vectors_test.go) runs
// it: the vector loader, package vectors, imports engine.
func CheckBattleRecords(t *testing.T, starts map[string][]BattleStart) map[string]int {
	records := loadBattleRecords(t)
	type tally struct {
		records, tokens, built, ungen, full, draws int
		compared, matched, total, salvos           int
		conditioned, condMatched                   int
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
		for _, r := range records[name] {
			where := fmt.Sprintf("%s %s %q (%d,%d) y%d", name, r.caseID, r.stream, r.x, r.y, r.year)
			c.records++
			for _, a := range r.actions {
				c.total += len(a.Hits)
			}
			years := starts[name]
			if r.year < 1 || r.year >= len(years) {
				c.ungen++
				continue
			}
			l := years[r.year]
			if l.Skip != "" {
				t.Errorf("%s: %s", where, l.Skip)
				continue
			}
			o := compareBattle(&l, r)
			switch {
			case o.built:
				// The battle-phase game has every fleet that fights.
				c.built++
				t.Errorf("%s: tokens: a recorded fleet is not in Elegy's game", where)
				continue
			case o.tokens != "":
				t.Errorf("%s: tokens: %s", where, o.tokens)
				continue
			}
			c.tokens++
			c.compared += o.compared
			c.matched += o.matched
			c.salvos += o.salvos
			c.conditioned += o.conditioned
			c.condMatched += o.condMatched
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
	// missing: a recorded fleet is not in Elegy's game; not generated: the
	// record's year was not generated
	// salvos: torpedo groups whose draws were answered from the record;
	// conditioned: hits after an answered draw, which rest on outcomes
	// taken from the record (not random parity)
	t.Logf("%-5s %8s %8s %8s %8s %8s %8s %8s  %-17s %-17s %s", "", "records", "tokens", "missing", "not gen", "replayed", "draws", "salvos", "matched/compared", "conditioned", "recorded")
	var all tally
	for _, name := range corpora {
		c := per[name]
		t.Logf("%-5s %8d %8d %8d %8d %8d %8d %8d  %-17s %-17s %d", name, c.records, c.tokens, c.built, c.ungen, c.full, c.draws, c.salvos,
			fmt.Sprintf("%d/%d", c.matched, c.compared), fmt.Sprintf("%d/%d", c.condMatched, c.conditioned), c.total)
		all.compared += c.compared
	}
	if all.compared == 0 {
		t.Error("no hit was compared")
	}
	ungen := map[string]int{}
	for name, c := range per {
		ungen[name] = c.ungen
	}
	return ungen
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
	n := 0
	for _, x := range records["rg/rg001.json"] {
		if x.year > 1 {
			n++
		}
	}
	if n != 7 {
		t.Errorf("RG-001 has %d battle records after its first year, want 7", n)
	}
	// CS-003-C's torpedo battles replay to the end, their salvos answered
	// from the record.
	answered := false
	for _, x := range records["cs/cs003c.json"] {
		o := compareBattle(&lb, x)
		answered = answered || o.salvos > 0 && o.stop == "" && o.conditioned > 0 && o.matched+o.condMatched == o.total
	}
	if !answered {
		t.Error("no CS-003-C torpedo battle replayed whole with its salvos answered")
	}
}

func TestSetupBattleCostLevels(t *testing.T) {
	// The harness builds battles with the year's cost levels: a token's
	// cost is the design's at those levels, not at the owner's current
	// ones (COMBAT.md "Design cost", #141).
	station, err := Components().NewDesign("Station", "Space Station", nil)
	if err != nil {
		t.Fatal(err)
	}
	g := Game{
		Rules: ElegyRules(),
		Players: []Player{
			{},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackPlayer, Player: 2}}},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackPlayer, Player: 1}}},
		},
		Designs: []Design{station, testDesign(tFrigate, 10, Slot{tLaser, 1})},
		Planets: []Planet{{ID: 1, Pos: Point{5, 5}, Owner: 1, HasStarbase: true, StarbaseDesign: 0}},
		Fleets:  []Fleet{{ID: 1, Owner: 2, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 1}}}},
	}
	g.Players[1].Research.Levels = [NumFields]int{8, 8, 8, 8, 8, 8}
	costLevels := [][NumFields]int{{}, {3, 3, 3, 3, 3, 3}, {}}
	race := g.Players[1].Race
	at, now := designCost(station, race, costLevels[1]), designCost(station, race, g.Players[1].Research.Levels)
	if at == now {
		t.Fatalf("cost %+v the same at both levels: not a test", at)
	}
	b, ok := setupBattle(&g, costLevels, 5, 5)
	if !ok {
		t.Fatal("no battle")
	}
	for _, tk := range b.tokens {
		if tk.starbase {
			if want := at.Resources + at.Minerals[Boranium]; tk.cost != want {
				t.Errorf("starbase token cost %d, want %d at the cost levels (%d at the current ones)", tk.cost, want, now.Resources+now.Minerals[Boranium])
			}
			return
		}
	}
	t.Fatal("no starbase token")
}

func TestReplayAnswerRand(t *testing.T) {
	// pvAnswer answers a group's rand(100) rolls with its scripted hits
	// first, then misses, and flags any other draw.
	b := &battle{}
	r := &pvAnswer{b: b, script: []int{2}, group: -1, first: -1, check: func() {}}
	var got []int
	for range 3 {
		got = append(got, r.Intn(100))
	}
	if !slices.Equal(got, []int{0, 0, 99}) || r.drew {
		t.Errorf("rolls %v, drew %v; want [0 0 99] and no other draw", got, r.drew)
	}
	r.Intn(10)
	if !r.drew {
		t.Error("a rand(10) was not flagged")
	}
}
