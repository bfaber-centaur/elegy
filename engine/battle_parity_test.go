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
	Players []int            `json:"players"`
	Planet  *int             `json:"planet"`
	Sample  bool             `json:"sample"`
	// StartSample is a battle record's `start_sample` (rg only): false
	// when the original's alternative streams all had the same start
	// tokens, their order and start squares aside (vectors README).
	StartSample *bool `json:"start_sample"`
}

// pvRecordedBattle is one battle record: its tokens and actions.
type pvRecordedBattle struct {
	vector, caseID, stream string
	year                   int // the expectation's year: 1 is the vector's first
	x, y                   int
	tokens                 []pvBattleToken
	actions                []pvBattleAction
	hasActions             bool
	players                []int // the record's players, as a set
	planet                 *int  // the vector planet the battle is at, or nil
	// flagged: the vector marks the record sample (it differed in
	// another of the original's random streams)
	flagged bool
	// startExact: the vector says its start tokens were the same in
	// every stream (start_sample false); only their order and start
	// squares were drawn.
	startExact bool
}

// takeTokens gives r the token list of the stream-less record from, with
// what goes with it: the players, the planet and start_sample.
func (r *pvRecordedBattle) takeTokens(from pvRecordedBattle) {
	r.tokens, r.players, r.planet, r.startExact = from.tokens, from.players, from.planet, from.startExact
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
	// Downstream counts the draws Elegy made in the vector's earlier
	// years. A year after a draw starts from a game those draws shaped,
	// not from the record's: its battles are samples (CheckBattleRecords).
	Downstream int
	// InYear counts the draws Elegy made in the year itself before its
	// battle phase (mine hits, for instance). A restart's battle that
	// differs from its record after such draws may be decided by them
	// (CheckBattleRecords).
	InYear int
	// Unflagged has CheckBattleRecords treat the year's records as if the
	// vector flagged none of them sample (for harness tests).
	Unflagged bool
}

// BattleVariants gives a vector's years generated with seed variant k
// (1 to N−1; variant 0 is the starts CheckBattleRecords is given).
type BattleVariants struct {
	N     int
	Start func(name string, k int) []BattleStart
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
	// startGap: the start tokens differ in more than their start
	// squares (a token missing or extra, another field, no battle, other
	// players or planet).
	startGap bool
	// moves checked (pvCheckMove), those on a square Elegy's rule allows,
	// those not checked because Elegy's square scores would draw, and the
	// first move Elegy's rule does not allow.
	moves, movesMatched, movesUnchecked int
	moveStop                            string
}

// pvAltBattle is an `alt_battle` expectation (rg only, vectors README
// "Battle records"): a place where an alternative stream of the original
// fought and the recorded stream did not.
type pvAltBattle struct {
	caseID  string
	year    int
	at      Point
	players []int
}

// loadBattleRecords reads every battle record in the vectors.
func loadBattleRecords(t testing.TB) map[string][]pvRecordedBattle {
	recs, _ := loadBattles(t)
	return recs
}

// loadBattles reads every battle record and every alt_battle in the
// vectors, by vector.
func loadBattles(t testing.TB) (map[string][]pvRecordedBattle, map[string][]pvAltBattle) {
	paths, err := filepath.Glob(filepath.Join("testdata", "vectors", "*", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no vectors: %v", err)
	}
	sort.Strings(paths)
	out := map[string][]pvRecordedBattle{}
	alts := map[string][]pvAltBattle{}
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
				if e.Kind == "alt_battle" {
					alts[name] = append(alts[name], pvAltBattle{caseID: c.ID, year: e.Year, at: Point{X: e.X, Y: e.Y}, players: e.Players})
					continue
				}
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
				r.flagged = r.flagged || e.Sample
				if e.Kind == "battle" {
					r.tokens, r.players, r.planet = e.Tokens, e.Players, e.Planet
					r.startExact = e.StartSample != nil && !*e.StartSample
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
						r.takeTokens(out[name][j])
					}
				}
			}
		}
	}
	return out, alts
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
// values, start squares only when squares is set. It returns the first
// difference, or "".
func (l *BattleStart) matchTokens(b *battle, rec []pvBattleToken, squares bool) string {
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
		if !squares {
			got.StartSquare = r.StartSquare
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
		rd := pvRound{total: map[*token]int{}, done: map[*token]int{}}
		for _, t := range b.tokens {
			if t.live() && !t.starbase {
				rd.total[t] = movesInRound(t.speed, r)
			}
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
			if t.live() && !t.starbase {
				out.moves++
				switch why := pvCheckMove(b, t, a.To, rd); why {
				case "":
					out.movesMatched++
				case pvUnchecked:
					out.movesUnchecked++
				default:
					if out.moveStop == "" {
						out.moveStop = fmt.Sprintf("round %d token %d: %s", r, a.Token, why)
					}
				}
				rd.done[t]++
			}
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

// pvUnchecked is pvCheckMove's answer for a move it cannot check: the
// square scores would draw (a one-torpedo estimate, COMBAT.md "Damage
// estimate").
const pvUnchecked = "unchecked"

// pvProbe is the Rand of a move check: it notes any draw.
type pvProbe struct{ drew bool }

func (p *pvProbe) Intn(int) int { p.drew = true; return 0 }

// pvRound is a round's moves so far in a replay: each token's moves this
// round (movesInRound) and its move actions applied so far.
type pvRound struct{ total, done map[*token]int }

// pvCheckMove checks one recorded move of token t (to is nil when it
// leaves the board) against Elegy's movement rule, before the move is
// applied: the record's square must be one that step can choose with
// some outcome of its tie draws (COMBAT.md "Choosing a square"). It keeps
// a disengaging token's counter as step does. It returns "" when the move
// is allowed, pvUnchecked, or why not.
//
// The moves left matter (reach, the radius and each enemy's "at least as
// many moves left"). A tactic-0 token has an action for every move, a
// stay included (CB-034); another token that stays has none in the
// records. So the phase of t's move (its moves left, a) and whether
// another token has already made its phase-a move are known only up to
// such silent stays: every consistent combination is tried, and the move
// is allowed when one of them allows it.
func pvCheckMove(b *battle, t *token, to *[2]int, rd pvRound) string {
	if t.tactic == TacticDisengage {
		if t.counter == 0 {
			if to == nil {
				return ""
			}
			return fmt.Sprintf("Elegy leaves the board; the record moves to %v", *to)
		}
		t.counter--
	}
	if to == nil {
		return fmt.Sprintf("the record leaves the board from (%d,%d); Elegy does not", t.x, t.y)
	}
	saved := b.rng
	probe := &pvProbe{}
	b.rng = probe
	moves := map[*token]int{}
	for _, e := range b.tokens {
		moves[e] = e.moves
	}
	defer func() {
		b.rng = saved
		for e, m := range moves {
			e.moves = m
		}
	}()
	hi := rd.total[t] - rd.done[t]
	lo := 1
	if t.tactic == TacticDisengage {
		lo = hi
	}
	allowed := map[[2]int]bool{}
	tried := false
phases:
	for a := lo; a <= hi; a++ {
		var unknown []*token
		for _, e := range b.tokens {
			if e == t || !e.live() || e.starbase {
				e.moves = 0
				continue
			}
			m, done := rd.total[e], rd.done[e]
			if m < a {
				// Its moves all come in later phases.
				if done > 0 {
					continue phases
				}
				e.moves = m
				continue
			}
			// By its phase-a move it has made m−a+1 moves.
			switch {
			case done > m-a+1:
				continue phases
			case done == m-a+1:
				e.moves = a - 1
			case e.tactic == TacticDisengage && done == m-a:
				e.moves = a
			case e.tactic == TacticDisengage:
				continue phases
			case !b.attacks(t.player, e.player):
				// t's radius and scores read only the tokens it attacks.
				e.moves = a
			default:
				unknown = append(unknown, e)
			}
		}
		if len(unknown) > 8 {
			return pvUnchecked
		}
		for mask := 0; mask < 1<<len(unknown); mask++ {
			for i, e := range unknown {
				e.moves = a
				if mask>>i&1 == 1 {
					e.moves = a - 1
				}
			}
			t.moves = a
			tried = true
			for q := range pvMoveSquares(b, t) {
				allowed[q] = true
			}
		}
	}
	if probe.drew {
		return pvUnchecked
	}
	if !tried {
		return fmt.Sprintf("no order of the round's moves fits the record's %d moves of %d", rd.done[t]+1, rd.total[t])
	}
	if !allowed[*to] {
		var sq [][2]int
		for q := range allowed {
			sq = append(sq, q)
		}
		sort.Slice(sq, func(i, j int) bool { return sq[i][0] < sq[j][0] || sq[i][0] == sq[j][0] && sq[i][1] < sq[j][1] })
		return fmt.Sprintf("the record moves from (%d,%d) to %v; Elegy's rule allows %v", t.x, t.y, *to, sq)
	}
	return ""
}

// pvMoveSquares is every square step can move t to, over all outcomes of
// its tie draws.
func pvMoveSquares(b *battle, t *token) map[[2]int]bool {
	radius, goal, hasGoal := b.radius(t)
	var picks [][2]int
	if hasGoal {
		picks = [][2]int{goal}
	} else {
		best, bestDist := 0, 0
		for x := range boardSize {
			for y := range boardSize {
				d := dist(t.x, t.y, x, y)
				if d > radius {
					continue
				}
				s := b.score(t, x, y)
				switch {
				case len(picks) == 0 || s < best || s == best && d < bestDist:
					picks, best, bestDist = [][2]int{{x, y}}, s, d
				case s == best && d == bestDist:
					picks = append(picks, [2]int{x, y})
				}
			}
		}
	}
	out := map[[2]int]bool{}
	add := func(x, y int) {
		if x < 0 || y < 0 || x >= boardSize || y >= boardSize {
			x, y = t.x, t.y // a step off the board stays
		}
		out[[2]int{x, y}] = true
	}
	for _, p := range picks {
		dx, dy := p[0]-t.x, p[1]-t.y
		if max(abs(dx), abs(dy)) <= 1 {
			add(p[0], p[1])
			continue
		}
		sx, sy := sign(dx), sign(dy)
		switch {
		case abs(dx) == abs(dy):
			add(t.x+sx, t.y+sy)
		case dx == 0 || dy == 0:
			cand := [3][2]int{{t.x + sx, t.y - 1}, {t.x + sx, t.y}, {t.x + sx, t.y + 1}}
			if dx == 0 {
				cand = [3][2]int{{t.x - 1, t.y + sy}, {t.x, t.y + sy}, {t.x + 1, t.y + sy}}
			}
			var scores [3]int
			low := 0
			for i, c := range cand {
				scores[i] = b.score(t, c[0], c[1])
				if i == 0 || scores[i] < low {
					low = scores[i]
				}
			}
			for i, c := range cand {
				if scores[i] == low {
					add(c[0], c[1])
				}
			}
		default:
			straight := [2]int{t.x + sx, t.y}
			if abs(dy) >= abs(dx) {
				straight = [2]int{t.x, t.y + sy}
			}
			sd, ss := b.score(t, t.x+sx, t.y+sy), b.score(t, straight[0], straight[1])
			if sd <= ss {
				add(t.x+sx, t.y+sy)
			}
			if ss <= sd {
				add(straight[0], straight[1])
			}
		}
	}
	return out
}

// pvSalvoBudget bounds the replays one battle's search may run.
const pvSalvoBudget = 20000

// buildBattle builds Elegy's battle for record r from a fresh copy of the
// battle-phase game, its tokens in the record's order.
func (l *BattleStart) buildBattle(r pvRecordedBattle) (*battle, string) {
	return l.buildBattleSquares(r, true)
}

// buildBattleSquares is buildBattle, comparing the tokens' start squares
// only when squares is set.
func (l *BattleStart) buildBattleSquares(r pvRecordedBattle, squares bool) (*battle, string) {
	g := withRules(l.Game.clone()) // the battle changes fleets and planets
	b, ok := setupBattle(&g, l.CostLevels, r.x, r.y)
	if !ok {
		return nil, "Elegy has no battle here"
	}
	if r.players != nil && !sameSet(b.players, r.players) {
		return b, fmt.Sprintf("players: Elegy %v, record %v", b.players, r.players)
	}
	var mine *int
	if b.loc.planet >= 0 {
		id := g.Planets[b.loc.planet].ID
		mine = &id
	}
	var want *int
	if r.planet != nil {
		id, ok := l.Planets[*r.planet]
		if !ok {
			return b, fmt.Sprintf("planet: the record's planet %d is not in Elegy's game", *r.planet)
		}
		want = &id
	}
	if (mine == nil) != (want == nil) || mine != nil && *mine != *want {
		return b, fmt.Sprintf("planet: Elegy %s, record %s", pvPlanetString(mine), pvPlanetString(want))
	}
	return b, l.matchTokens(b, r.tokens, squares)
}

// How a restart's miss that no seed variant explains is taken.
const (
	pvStrict        = iota // a failure, as for any record
	pvFlaggedSample        // a sample: the vector flags the record
	pvStartGap             // a failure: start tokens the vector says were not drawn
)

// flaggedMiss classes a restart's miss on record r that no other seed
// variant matches. A record the vector flags sample differed between the
// original's own random streams, so a miss is a sample, not a parity
// failure; an unflagged record stays strict. With start_sample false only
// the token order and start squares were drawn, so start tokens that
// differ in anything else are a gap, not a sample.
func flaggedMiss(r pvRecordedBattle, l *BattleStart, o pvBattleOutcome) int {
	switch {
	case !r.flagged || l.Unflagged:
		return pvStrict
	case r.startExact && o.startGap:
		return pvStartGap
	}
	return pvFlaggedSample
}

// startGap reports whether Elegy's start tokens for record r differ in
// more than their start squares: no battle, other players or planet, a
// token missing or extra, or any other token field.
func (l *BattleStart) startGap(r pvRecordedBattle) bool {
	_, why := l.buildBattleSquares(r, false)
	return why != ""
}

// sameSet reports whether a and b hold the same numbers.
func sameSet(a, b []int) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func pvPlanetString(p *int) string {
	if p == nil {
		return "none"
	}
	return fmt.Sprint(*p)
}

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
				out.built, out.startGap = true, true
				return out
			}
		}
	}
	build := func() (*battle, string) { return l.buildBattle(r) }
	b, why := build()
	if why != "" {
		out.tokens = why
		out.startGap = l.startGap(r)
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

// sampleMiss is why a sample's comparison missed, or "" when everything
// compared matched: the tokens, every hit and move compared (conditioned
// ones included). A round with draws ends the comparison without a miss,
// as it does for a record.
func (o pvBattleOutcome) sampleMiss() string {
	switch {
	case o.built:
		return "tokens: a recorded fleet is not in Elegy's game"
	case o.tokens != "":
		return "tokens: " + o.tokens
	case o.moveStop != "":
		return "moves: " + o.moveStop
	case !o.draws && o.stop != "":
		return "hits: " + o.stop
	}
	return ""
}

// matchedDetail says what a record that matched was compared on.
func (o pvBattleOutcome) matchedDetail(r pvRecordedBattle) string {
	d := fmt.Sprintf("%d tokens, players and planet; %d of %d hits compared", len(r.tokens), o.matched, o.total)
	if o.conditioned > 0 {
		d += fmt.Sprintf(" and %d conditioned", o.conditioned)
	}
	if o.draws {
		d += ", the replay stopping at a draw"
	}
	return d + fmt.Sprintf("; %d moves allowed", o.movesMatched)
}

// elegyBattles returns the locations where Elegy fights in the
// battle-phase game l, with each battle's players.
func elegyBattles(l *BattleStart) map[Point][]int {
	g := withRules(l.Game.clone())
	out := map[Point][]int{}
	var prev locationHistory
	for _, loc := range g.locations() {
		_, inv, _ := g.whoFights(loc, prev)
		prev = locationHistory{any: true, battle: inv != nil, lastOwner: g.Fleets[loc.fleets[len(loc.fleets)-1]].Owner}
		if inv != nil {
			out[loc.pos] = inv
		}
	}
	return out
}

// checkBattleSet fails on each battle Elegy fights in a year of vector
// name that has battle records but none at that location: a vector
// records every battle of its years (a year with no record is not
// checked, as nothing shows the vector recorded it). Classes as in
// CheckBattleRecords: a chained year past a random year is a sample, an
// extra battle where the vector has an alt_battle of the same players is
// a sample (an alternative stream of the original fought there), and a
// restart's extra battle that another seed variant does not fight is
// random-driven. An alt_battle where Elegy does not fight is a sample
// too, which matched: the recorded stream has no battle there either.
func checkBattleSet(t testing.TB, name string, recs []pvRecordedBattle, alts []pvAltBattle, years []BattleStart, vs BattleVariants) {
	type yearRecs struct {
		caseID string
		at     map[Point]bool
	}
	byYear := map[int]*yearRecs{}
	for _, r := range recs {
		if byYear[r.year] == nil {
			byYear[r.year] = &yearRecs{caseID: r.caseID, at: map[Point]bool{}}
		}
		byYear[r.year].at[Point{X: r.x, Y: r.y}] = true
	}
	var ys []int
	for y := range byYear {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	for _, y := range ys {
		if y < 1 || y >= len(years) || years[y].Skip != "" {
			continue
		}
		l := years[y]
		mine := elegyBattles(&l)
		var extra []Point
		for p := range mine {
			if !byYear[y].at[p] {
				extra = append(extra, p)
			}
		}
		sort.Slice(extra, func(i, j int) bool {
			return extra[i].X < extra[j].X || extra[i].X == extra[j].X && extra[i].Y < extra[j].Y
		})
		for _, p := range extra {
			where := fmt.Sprintf("%s %s %q (%d,%d) y%d", name, byYear[y].caseID, "", p.X, p.Y, y)
			why := fmt.Sprintf("battle set: Elegy fights here (players %v), the record has no battle", mine[p])
			if l.Downstream > 0 {
				t.Logf("%s: sample (after %d draws of earlier years) missed: %s", where, l.Downstream, why)
				continue
			}
			if altBattle(alts, y, p, mine[p]) {
				t.Logf("%s: sample (alt_battle in the vector) missed: %s", where, why)
				continue
			}
			if l.InYear > 0 && vs.N > 1 {
				hit := 0
				for k := 1; k < vs.N; k++ {
					if alt := vs.Start(name, k); y < len(alt) && alt[y].Skip == "" {
						if _, fights := elegyBattles(&alt[y])[p]; !fights {
							hit++
						}
					}
				}
				if hit > 0 {
					t.Logf("%s: random (after %d draws in the year; %d of %d other variants match): %s", where, l.InYear, hit, vs.N-1, why)
					continue
				}
			}
			t.Errorf("%s: %s", where, why)
		}
	}
	// Each alt_battle where Elegy does not fight. One where it fights is
	// an extra battle above, or, in a year with no record, unchecked like
	// every battle of such a year. A year not run is reported by the
	// vectors ledger ("not generated").
	for _, a := range alts {
		if a.year < 1 || a.year >= len(years) || years[a.year].Skip != "" {
			continue
		}
		l := years[a.year]
		if _, fights := elegyBattles(&l)[a.at]; fights {
			continue
		}
		where := fmt.Sprintf("%s %s %q (%d,%d) y%d", name, a.caseID, "", a.at.X, a.at.Y, a.year)
		why := fmt.Sprintf("no battle here, as in the recorded stream (an alternative stream's players %v fought)", sortedInts(a.players))
		if l.Downstream > 0 {
			t.Logf("%s: sample (after %d draws of earlier years) matched: %s", where, l.Downstream, why)
			continue
		}
		t.Logf("%s: sample (alt_battle in the vector) matched: %s", where, why)
	}
}

// altBattle reports whether alts has an alt_battle in year y at p with
// exactly the players Elegy's battle there has.
func altBattle(alts []pvAltBattle, y int, p Point, players []int) bool {
	for _, a := range alts {
		if a.year == y && a.at == p && slices.Equal(sortedInts(a.players), sortedInts(players)) {
			return true
		}
	}
	return false
}

func sortedInts(s []int) []int {
	s = slices.Clone(s)
	slices.Sort(s)
	return s
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

// recordTBInternal keeps a check's failures.
type recordTBInternal struct {
	testing.TB
	errs, logs []string
}

func (r *recordTBInternal) Logf(format string, args ...any) {
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
	r.TB.Logf(format, args...)
}

func (r *recordTBInternal) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

// BattleRecordKeys lists the records of vector name ("rg/rg002.json") as
// "case yN | battle record (x,y)", with "" for each record that has both
// its `battle` and its `battle_actions` expectation, else what it lacks.
func BattleRecordKeys(t testing.TB, name string) map[string]string {
	out := map[string]string{}
	for _, r := range loadBattleRecords(t)[name] {
		where := fmt.Sprintf("%s y%d", r.caseID, r.year)
		if r.stream != "" {
			where += " stream " + r.stream
		}
		var lacks string
		switch {
		case r.tokens == nil:
			lacks = "no battle expectation"
		case !r.hasActions:
			lacks = "no battle_actions expectation"
		}
		out[fmt.Sprintf("%s | battle record (%d,%d)", where, r.x, r.y)] = lacks
	}
	return out
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
// count by corpus ("rg").
//
// A record of a year that follows Elegy's draws in earlier years
// (BattleStart.Downstream: a chained run past a random year) is a sample:
// its fleets and states come from Elegy's own random outcomes, so a match
// does not show parity and a difference does not show a defect. Samples
// are compared in full but tallied on their own, as matched or missed,
// never in the token, hit or move tallies and never as a failure; the
// per-year restarts compare those years from the record.
//
// A restart's record that differs after draws made in the year before the
// battle phase (BattleStart.InYear) is regenerated with each other seed
// variant. If any variant's battle matches the record, the difference is
// random-driven: the year's own draws decide it. It is logged and tallied
// as random, matched variants / variants, and is not a failure. If no
// variant matches, it fails as a deterministic difference. With no
// variants (vs.N < 2) every restart difference fails. TestBattleRecords
// (battle_vectors_test.go) runs it: the vector loader, package vectors,
// imports engine.
func CheckBattleRecords(t testing.TB, starts map[string][]BattleStart, vs BattleVariants) map[string]int {
	records, alts := loadBattles(t)
	type tally struct {
		records, tokens, built, ungen, full, draws int
		compared, matched, total, salvos           int
		conditioned, condMatched                   int
		moves, movesMatched, movesUnchecked        int
		samples, samplesMatched                    int
		random                                     int
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
			if l.Downstream > 0 {
				c.samples++
				if why := o.sampleMiss(); why != "" {
					t.Logf("%s: sample (after %d draws of earlier years) missed: %s", where, l.Downstream, why)
				} else {
					c.samplesMatched++
					t.Logf("%s: sample (after %d draws of earlier years) matched", where, l.Downstream)
				}
				continue
			}
			if why := o.sampleMiss(); why != "" && l.InYear > 0 && vs.N > 1 {
				hit := 0
				for k := 1; k < vs.N; k++ {
					if alt := vs.Start(name, k); r.year < len(alt) {
						a := alt[r.year]
						if a.Skip == "" && compareBattle(&a, r).sampleMiss() == "" {
							hit++
						}
					}
				}
				if hit > 0 {
					c.random++
					t.Logf("%s: random (after %d draws in the year; %d of %d other variants match): %s", where, l.InYear, hit, vs.N-1, why)
					continue
				}
			}
			// A record the vector itself flags sample differed between the
			// original's own random streams: a miss is a sample, never a
			// parity failure. An unflagged record stays strict.
			if why := o.sampleMiss(); why != "" {
				switch flaggedMiss(r, &l, o) {
				case pvStartGap:
					if o.built {
						c.built++
					}
					t.Errorf("%s: start tokens differ though the record's start is the same in every stream (start_sample false): %s", where, why)
					continue
				case pvFlaggedSample:
					c.samples++
					t.Logf("%s: sample (flagged in the vector) missed: %s", where, why)
					continue
				}
			}
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
			failed := false
			c.tokens++
			c.compared += o.compared
			c.matched += o.matched
			c.salvos += o.salvos
			c.conditioned += o.conditioned
			c.condMatched += o.condMatched
			c.moves += o.moves
			c.movesMatched += o.movesMatched
			c.movesUnchecked += o.movesUnchecked
			if o.moveStop != "" {
				failed = true
				t.Errorf("%s: moves: %s", where, o.moveStop)
			}
			switch {
			case o.draws:
				c.draws++
			case o.stop != "":
				failed = true
				t.Errorf("%s: hits: %s", where, o.stop)
			case r.hasActions:
				c.full++
			}
			if !failed {
				t.Logf("%s: matched: %s", where, o.matchedDetail(r))
			}
		}
		checkBattleSet(t, name, records[name], alts[name], starts[name], vs)
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
	// moves: recorded moves on a square Elegy's rule allows / moves
	// checked; unchecked: moves whose square scores would draw, or with
	// too many tokens whose silent stays leave the moves left open;
	// samples: records of a year after earlier years' draws, matched /
	// compared (in none of the other columns); random: restart records
	// that differ after the year's own draws and match on another seed
	// variant (in none of the other columns)
	t.Logf("%-5s %8s %8s %8s %8s %8s %8s %8s  %-17s %-17s %8s  %-13s %9s  %-9s %s", "", "records", "tokens", "missing", "not gen", "replayed", "draws", "salvos", "matched/compared", "conditioned", "recorded", "moves", "unchecked", "samples", "random")
	var all tally
	for _, name := range corpora {
		c := per[name]
		t.Logf("%-5s %8d %8d %8d %8d %8d %8d %8d  %-17s %-17s %8d  %-13s %9d  %-9s %d", name, c.records, c.tokens, c.built, c.ungen, c.full, c.draws, c.salvos,
			fmt.Sprintf("%d/%d", c.matched, c.compared), fmt.Sprintf("%d/%d", c.condMatched, c.conditioned), c.total,
			fmt.Sprintf("%d/%d", c.movesMatched, c.moves-c.movesUnchecked), c.movesUnchecked,
			fmt.Sprintf("%d/%d", c.samplesMatched, c.samples), c.random)
		all.compared += c.compared
		all.movesMatched += c.movesMatched
	}
	if all.compared == 0 {
		t.Error("no hit was compared")
	}
	if all.movesMatched == 0 {
		t.Error("no move was checked")
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
	if o := compareBattle(&l, r); o.tokens != "" || o.stop != "" || o.compared == 0 || o.compared != o.total || o.conditioned != 0 {
		t.Fatalf("CB-010 at 1060: %+v, want every hit replayed", o)
	}
	if o := compareBattle(&l, r); o.moves == 0 || o.movesMatched != o.moves-o.movesUnchecked || o.moveStop != "" {
		t.Fatalf("CB-010 at 1060: moves %+v, want every move checked and allowed", o)
	}
	// A record's players are compared as a set, and its planet.
	players := r
	players.players = append([]int{}, r.players...)
	players.players[0] = 5
	if o := compareBattle(&l, players); !strings.HasPrefix(o.tokens, "players: ") {
		t.Errorf("a changed players list was not noticed: %+v", o)
	}
	players.players = []int{r.players[1], r.players[0]}
	if o := compareBattle(&l, players); o.tokens != "" {
		t.Errorf("players in another order: %+v, want a match", o)
	}
	planet := r
	other := 0
	for id := range l.Planets {
		other = max(other, id+1)
	}
	planet.planet = &other
	if o, want := compareBattle(&l, planet), fmt.Sprintf("planet: the record's planet %d is not in Elegy's game", other); o.tokens != want {
		t.Errorf("a record planet not in the game: %q, want %q", o.tokens, want)
	}
	// Both planets exist but differ: CS-003-C's battle at planet 5 with
	// the record naming planet 4.
	var at5 pvRecordedBattle
	for _, x := range records["cs/cs003c.json"] {
		if x.planet != nil && *x.planet == 5 {
			at5 = x
		}
	}
	if at5.planet == nil {
		t.Fatal("no CS-003-C battle at planet 5")
	}
	if o := compareBattle(&lb, at5); strings.HasPrefix(o.tokens, "planet: ") {
		t.Fatalf("CS-003-C at planet 5: %q, want its planet to match", o.tokens)
	}
	four := 4
	at5.planet = &four
	if o, want := compareBattle(&lb, at5), fmt.Sprintf("planet: Elegy %d, record %d", lb.Planets[5], lb.Planets[4]); o.tokens != want {
		t.Errorf("a record naming another planet in the game: %q, want %q", o.tokens, want)
	}
	// A battle Elegy fights where the year has no record fails.
	var rest []pvRecordedBattle
	for _, x := range records["cb/cb010.json"] {
		if x.year == 1 && x.x != 1060 {
			rest = append(rest, x)
		}
	}
	set := &recordTBInternal{TB: t}
	checkBattleSet(set, "cb/cb010.json", rest, nil, []BattleStart{{}, l}, BattleVariants{})
	if len(set.errs) != 1 || !strings.Contains(set.errs[0], "(1060,") || !strings.Contains(set.errs[0], "battle set: Elegy fights here") {
		t.Errorf("an unrecorded battle: %q, want one battle-set failure at 1060", set.errs)
	}
	set = &recordTBInternal{TB: t}
	checkBattleSet(set, "cb/cb010.json", append(rest, r), nil, []BattleStart{{}, l}, BattleVariants{})
	if len(set.errs) != 0 {
		t.Errorf("every battle recorded: %q, want none", set.errs)
	}
	// An alt_battle of the same players there makes it a sample; one of
	// other players (even as many of them), or of an earlier or later
	// year or another place, does not.
	fought := elegyBattles(&l)[Point{X: r.x, Y: r.y}]
	alt := pvAltBattle{year: 1, at: Point{X: r.x, Y: r.y}, players: []int{fought[1], fought[0]}}
	outsider := slices.Max(fought) + 1
	for _, c := range []struct {
		name   string
		alt    pvAltBattle
		sample bool
	}{
		{"same players, listed in another order", alt, true},
		{"other players", pvAltBattle{year: 1, at: alt.at, players: []int{fought[0]}}, false},
		{"as many players, one of them not fighting there", pvAltBattle{year: 1, at: alt.at, players: []int{fought[0], outsider}}, false},
		{"a later year", pvAltBattle{year: 2, at: alt.at, players: alt.players}, false},
		{"an earlier year", pvAltBattle{year: 0, at: alt.at, players: alt.players}, false},
		{"another place", pvAltBattle{year: 1, at: Point{X: r.x + 1, Y: r.y}, players: alt.players}, false},
	} {
		set = &recordTBInternal{TB: t}
		checkBattleSet(set, "cb/cb010.json", rest, []pvAltBattle{c.alt}, []BattleStart{{}, l}, BattleVariants{})
		sampled := len(set.logs) == 1 && strings.Contains(set.logs[0], "(1060,") && strings.Contains(set.logs[0], ": sample (alt_battle in the vector) missed: battle set: Elegy fights here")
		if sampled != c.sample || len(set.errs) != map[bool]int{true: 0, false: 1}[c.sample] {
			t.Errorf("alt_battle, %s: errors %q, logs %q, want sample %v", c.name, set.errs, set.logs, c.sample)
		}
	}
	// An alt_battle where Elegy does not fight is a matched sample (the
	// recorded stream has none there either); after earlier years' draws
	// it says so.
	away := pvAltBattle{caseID: "C", year: 1, at: Point{X: r.x + 1, Y: r.y}, players: []int{2, 0}}
	for _, c := range []struct {
		name       string
		downstream int
		want       string
	}{
		{"restart", 0, ": sample (alt_battle in the vector) matched: no battle here, as in the recorded stream (an alternative stream's players [0 2] fought)"},
		{"chained", 3, ": sample (after 3 draws of earlier years) matched: no battle here, as in the recorded stream (an alternative stream's players [0 2] fought)"},
	} {
		ly := l
		ly.Downstream = c.downstream
		set = &recordTBInternal{TB: t}
		checkBattleSet(set, "cb/cb010.json", append(rest, r), []pvAltBattle{away}, []BattleStart{{}, ly}, BattleVariants{})
		if len(set.errs) != 0 || len(set.logs) != 1 || !strings.HasSuffix(set.logs[0], c.want) ||
			!strings.Contains(set.logs[0], fmt.Sprintf(`cb/cb010.json C "" (%d,%d) y1`, r.x+1, r.y)) {
			t.Errorf("alt_battle with no battle, %s: errors %q, logs %q, want one log ending %q", c.name, set.errs, set.logs, c.want)
		}
	}
	// A recorded move two squares long is not one Elegy's rule allows.
	jump := r
	jump.actions = append([]pvBattleAction(nil), r.actions...)
	for i, a := range jump.actions {
		if len(a.Hits) == 0 && a.To != nil {
			far := [2]int{(a.To[0] + 5) % boardSize, (a.To[1] + 5) % boardSize}
			a.To = &far
			jump.actions[i] = a
			break
		}
	}
	if o := compareBattle(&l, jump); !strings.Contains(o.moveStop, "Elegy's rule allows") {
		t.Errorf("a changed recorded move was not noticed: %+v", o)
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
	// A hit the record has and Elegy does not fails at the round's end.
	extra := r
	extra.actions = append([]pvBattleAction(nil), r.actions...)
	for i := len(extra.actions) - 1; i >= 0; i-- {
		if a := extra.actions[i]; len(a.Hits) > 0 {
			a.Hits = append(slices.Clone(a.Hits), a.Hits[len(a.Hits)-1])
			extra.actions[i] = a
			break
		}
	}
	if o := compareBattle(&l, extra); !strings.Contains(o.stop, "Elegy has no further hit") {
		t.Errorf("an extra recorded hit was not noticed: %+v", o)
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
	// The hits before the first torpedo group are compared, and every one
	// from that group's first on is conditioned. A replay with no answers
	// stops when the first group is over, before its hits are compared:
	// it compares exactly the hits before it.
	answered := false
	for _, x := range records["cs/cs003c.json"] {
		o := compareBattle(&lb, x)
		if o.salvos == 0 || o.stop != "" {
			continue
		}
		answered = true
		b, why := lb.buildBattle(x)
		if why != "" {
			t.Fatal(why)
		}
		var before pvBattleOutcome
		if replayHits(b, x.actions, nil, &before) == nil || before.conditioned != 0 {
			t.Fatalf("CS-003-C (%d,%d): the unanswered replay did not stop at the first group: %+v", x.x, x.y, before)
		}
		if o.compared != before.compared || o.matched != o.compared || o.conditioned != o.total-before.compared || o.condMatched != o.conditioned {
			t.Errorf("CS-003-C (%d,%d): compared %d/%d, conditioned %d/%d of %d; want %d hits before the first group compared, the rest conditioned",
				x.x, x.y, o.matched, o.compared, o.condMatched, o.conditioned, o.total, before.compared)
		}
	}
	if !answered {
		t.Error("no CS-003-C torpedo battle replayed whole with its salvos answered")
	}
	checkStartSample(t, l, r)
}

// checkStartSample checks the start_sample rule on CB-010's battle at
// 1060 (r, which matches in full): which token differences are a start
// gap, and how flaggedMiss classes a restart's miss.
func checkStartSample(t *testing.T, l BattleStart, r pvRecordedBattle) {
	retoken := func(edit func(*pvBattleToken)) pvRecordedBattle {
		x := r
		x.tokens = append([]pvBattleToken(nil), r.tokens...)
		edit(&x.tokens[0])
		return x
	}
	for _, c := range []struct {
		name string
		r    pvRecordedBattle
		gap  bool
	}{
		{"start square", retoken(func(k *pvBattleToken) { k.StartSquare[0] = (k.StartSquare[0] + 1) % boardSize }), false},
		{"damage", retoken(func(k *pvBattleToken) { k.Damage.Units++ }), true},
		{"ships", retoken(func(k *pvBattleToken) { k.Ships++ }), true},
		{"design", retoken(func(k *pvBattleToken) { k.Design += 3 }), true},
		{"a token missing", func() pvRecordedBattle { x := r; x.tokens = r.tokens[1:]; return x }(), true},
		{"no battle", func() pvRecordedBattle { x := r; x.x++; return x }(), true},
		{"a fleet not in the game", retoken(func(k *pvBattleToken) {
			missing := 0
			for key := range l.Fleets {
				missing = max(missing, key[1]+1)
			}
			k.Fleet = &missing
		}), true},
	} {
		o := compareBattle(&l, c.r)
		if o.tokens == "" && !o.built {
			t.Errorf("start_sample, %s: no token difference noticed", c.name)
		}
		if o.startGap != c.gap {
			t.Errorf("start_sample, %s: start gap %t, want %t (%s)", c.name, o.startGap, c.gap, o.tokens)
		}
	}
	// A stream's record takes start_sample with the stream-less token list.
	var streamed pvRecordedBattle
	streamed.takeTokens(pvRecordedBattle{tokens: r.tokens, startExact: true})
	if !streamed.startExact || len(streamed.tokens) != len(r.tokens) {
		t.Errorf("takeTokens: %+v, want the token list and start_sample", streamed)
	}
	gap := compareBattle(&l, retoken(func(k *pvBattleToken) { k.Damage.Units++ }))
	square := compareBattle(&l, retoken(func(k *pvBattleToken) { k.StartSquare[0] = (k.StartSquare[0] + 1) % boardSize }))
	unflagged := l
	unflagged.Unflagged = true
	for _, c := range []struct {
		name                string
		flagged, startExact bool
		l                   *BattleStart
		o                   pvBattleOutcome
		want                int
	}{
		{"unflagged", false, true, &l, gap, pvStrict},
		{"flagged, Unflagged set", true, true, &unflagged, gap, pvStrict},
		{"start_sample false, a start gap", true, true, &l, gap, pvStartGap},
		{"start_sample false, squares only", true, true, &l, square, pvFlaggedSample},
		{"start_sample false, hits", true, true, &l, pvBattleOutcome{stop: "hit 1"}, pvFlaggedSample},
		{"start_sample true or absent, a start gap", true, false, &l, gap, pvFlaggedSample},
	} {
		x := r
		x.flagged, x.startExact = c.flagged, c.startExact
		if got := flaggedMiss(x, c.l, c.o); got != c.want {
			t.Errorf("flaggedMiss, %s: %d, want %d", c.name, got, c.want)
		}
	}
}

// A sample misses on any difference the comparison reports for a record,
// except a round with draws, which ends the comparison of both.
func TestBattleSampleMiss(t *testing.T) {
	cases := []struct {
		o    pvBattleOutcome
		miss bool
	}{
		{pvBattleOutcome{}, false},
		{pvBattleOutcome{draws: true, stop: "round 3: Elegy draws"}, false},
		{pvBattleOutcome{built: true}, true},
		{pvBattleOutcome{tokens: "token 2"}, true},
		{pvBattleOutcome{moveStop: "round 1"}, true},
		{pvBattleOutcome{stop: "salvo 1"}, true},
	}
	for _, c := range cases {
		if got := c.o.sampleMiss() != ""; got != c.miss {
			t.Errorf("%+v: miss %v, want %v", c.o, got, c.miss)
		}
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

// CheckStarbaseDamageKept replays RG-001's two 2435 battles at player 6's
// Orbital Forts (planets 54 and 55 of the vector) with the record's moves
// and checks that the Forts keep their battle damage through the year's
// repair: a starbase that fought gets none (COMBAT.md "Repair", CONFIRMED
// CB-017). The original ended the year with 360 and 180 (the vector's
// planet samples). l is RG-001-Y6's start (rg/rg001-y6.json, year 1).
func CheckStarbaseDamageKept(t *testing.T, l BattleStart) {
	want := map[int]int{1460: 360, 1308: 180}
	n := 0
	for _, r := range loadBattleRecords(t)["rg/rg001-y6.json"] {
		if _, ok := want[r.y]; !ok || r.x != 1319 || r.year != 1 {
			continue
		}
		n++
		b, why := l.buildBattle(r)
		if why != "" {
			t.Fatalf("(%d,%d): %s", r.x, r.y, why)
		}
		var o pvBattleOutcome
		total := 0
		for _, a := range r.actions {
			total += len(a.Hits)
		}
		if need := replayHits(b, r.actions, nil, &o); need != nil || o.stop != "" || o.matched != total || total == 0 {
			t.Fatalf("(%d,%d): replay %+v of %d hits", r.x, r.y, o, total)
		}
		pi := -1
		for _, tk := range b.tokens {
			if tk.starbase {
				pi = tk.planet
			}
		}
		b.finish()
		g := b.g
		repair(g, nil, battleResult{bases: map[int]bool{pi: true}})
		if got := g.Planets[pi].StarbaseDamage; got != want[r.y] {
			t.Errorf("(%d,%d): starbase damage %d after the year's repair, want %d", r.x, r.y, got, want[r.y])
		}
		// Without the mark, the year's repair would take 50 off.
		repair(g, nil, battleResult{})
		if got := g.Planets[pi].StarbaseDamage; got != want[r.y]-50 {
			t.Errorf("(%d,%d): starbase damage %d after a repair with no battle, want %d", r.x, r.y, got, want[r.y]-50)
		}
	}
	if n != 2 {
		t.Fatalf("%d RG-001 2435 Orbital Fort battles, want 2", n)
	}
}
