package engine

// A battle's board, setup and rounds (COMBAT.md "Board setup", "Rounds").

// startSquares are the start squares by rank among n involved players
// (CONFIRMED for two players, CB-001..CB-019; BINARY-ONLY otherwise).
var startSquares = [17][][2]int{
	1:  {{4, 4}},
	2:  {{1, 4}, {8, 5}},
	3:  {{4, 1}, {8, 8}, {1, 8}},
	4:  {{1, 1}, {8, 8}, {1, 8}, {8, 1}},
	5:  {{4, 1}, {6, 8}, {1, 4}, {8, 4}, {2, 8}},
	6:  {{1, 4}, {8, 5}, {2, 8}, {7, 1}, {6, 8}, {3, 1}},
	7:  {{1, 1}, {1, 5}, {2, 8}, {6, 8}, {8, 6}, {8, 2}, {5, 1}},
	8:  {{1, 3}, {1, 6}, {3, 8}, {6, 8}, {8, 6}, {8, 3}, {6, 1}, {3, 1}},
	9:  {{1, 3}, {8, 6}, {3, 8}, {6, 1}, {1, 6}, {8, 3}, {6, 8}, {3, 1}, {4, 4}},
	10: {{2, 1}, {5, 1}, {8, 1}, {1, 4}, {8, 4}, {4, 5}, {1, 7}, {8, 7}, {3, 8}, {6, 8}},
	11: {{1, 3}, {8, 6}, {3, 8}, {6, 1}, {1, 6}, {8, 3}, {6, 8}, {3, 1}, {3, 4}, {6, 3}, {6, 6}},
	12: {{1, 4}, {8, 5}, {2, 8}, {7, 1}, {6, 8}, {3, 1}, {1, 6}, {8, 3}, {1, 2}, {4, 8}, {5, 1}, {8, 7}},
	13: {{1, 1}, {1, 3}, {1, 5}, {1, 7}, {3, 1}, {5, 1}, {7, 1}, {8, 3}, {8, 5}, {3, 8}, {5, 8}, {7, 8}, {4, 4}},
	14: {{1, 1}, {1, 3}, {1, 5}, {1, 7}, {2, 8}, {4, 8}, {6, 8}, {8, 8}, {8, 6}, {8, 4}, {8, 2}, {7, 1}, {5, 1}, {3, 1}},
	15: {{1, 1}, {1, 3}, {1, 5}, {1, 7}, {2, 8}, {4, 8}, {6, 8}, {8, 8}, {8, 6}, {8, 4}, {8, 2}, {7, 1}, {5, 1}, {3, 1}, {4, 4}},
	16: {{1, 1}, {1, 3}, {1, 5}, {1, 7}, {2, 8}, {4, 8}, {6, 8}, {8, 8}, {8, 6}, {8, 4}, {8, 2}, {7, 1}, {5, 1}, {3, 1}, {3, 3}, {6, 6}},
}

const (
	boardSize  = 10
	maxRounds  = 16
	maxTokens  = 256
	jitterSpan = 15
)

// BattleHit is one application of damage, for tests and battle reports.
type BattleHit struct {
	Round         int
	Firer, Target int // token indices after the shuffle
	Shield, Armor int // damage taken by shields and by armor (before kills)
	Kills         int
	Hits, Misses  int // torpedoes
	Leftover      int // beam damage left after the target died
}

// battle is one battle in progress.
type battle struct {
	g        *Game
	rng      Rand
	loc      location
	sets     attackSets
	players  []int // involved players, in player order
	tokens   []*token
	round    int
	in       map[int]bool // players still in the battle
	hits     []BattleHit
	salvage  Minerals // deep-space salvage gathered so far
	salvaged bool     // a deep-space kill event happened
	seen     [NumFields]int
	killed   map[int]bool // players that lost ships or a starbase
}

func dist(ax, ay, bx, by int) int {
	return max(abs(ax-bx), abs(ay-by))
}

func (b *battle) attacks(p, q int) bool { return b.sets[p][q] }

// setup creates the tokens (COMBAT.md "Setup steps", "Token values").
// fought receives the fleets that take part.
func (b *battle) setup(fought map[int]bool) []Event {
	g := b.g
	var events []Event
	isInvolved := map[int]bool{}
	for _, p := range b.players {
		isInvolved[p] = true
	}
	var fleets []int
	for _, i := range b.loc.fleets {
		if isInvolved[g.Fleets[i].Owner] {
			fleets = append(fleets, i)
		}
	}
	fleets, missed := b.capTokens(fleets)
	for _, p := range missed {
		events = append(events, Event{Kind: EventFleetsMissedBattle, Player: p, Planet: -1, Fleet: -1})
	}

	for _, i := range fleets {
		f := &g.Fleets[i]
		fought[i] = true
		plan := g.plan(f.Owner, f.Plan)
		race := g.Players[f.Owner].Race
		dumped := false
		if plan.DumpCargo && f.Cargo.Minerals != (Minerals{}) {
			dumped = true
			if b.loc.planet >= 0 {
				for m := range NumMinerals {
					g.Planets[b.loc.planet].Surface[m] += f.Cargo.Minerals[m]
				}
			} else {
				for m := range NumMinerals {
					b.salvage[m] += f.Cargo.Minerals[m]
				}
				b.salvaged = true
			}
			f.Cargo.Minerals = Minerals{}
		}
		capacity := g.fleetCargoCapacity(f)
		for si, s := range f.Stacks {
			if s.Count <= 0 {
				continue
			}
			d := g.Designs[s.Design]
			t := tokenValues(d, race, false)
			t.player, t.fleet, t.stack, t.planet, t.design = f.Owner, i, si, -1, s.Design
			t.ships, t.dmg = s.Count, s.Damage
			t.mass = d.Mass
			if capacity > 0 {
				t.mass += f.Cargo.mass() * d.CargoCapacity / capacity
			}
			t.speed = speedCode(d, race, t.mass, dumped)
			if t.armed() {
				t.tactic, t.primary, t.secondary = plan.Tactic, plan.Primary, plan.Secondary
			} else {
				t.tactic = TacticDisengage
			}
			t.jitter = b.rng.Intn(jitterSpan)
			b.tokens = append(b.tokens, &t)
		}
	}
	if owner, ok := g.battleStarbase(b.loc.planet); ok && isInvolved[owner] {
		pl := &g.Planets[b.loc.planet]
		t := tokenValues(g.Designs[pl.StarbaseDesign], g.Players[owner].Race, true)
		t.player, t.fleet, t.stack, t.planet, t.design = owner, -1, -1, b.loc.planet, pl.StarbaseDesign
		t.ships, t.dmg = 1, Damage{Pct: 100, Units: pl.StarbaseDamage}
		t.tactic, t.primary, t.secondary = TacticMaximizeDamage, TargetAny, TargetAny
		t.mass = 65535
		b.tokens = append(b.tokens, &t)
	}

	// Shuffle.
	n := len(b.tokens)
	for i := range n {
		j := i + b.rng.Intn(n-i)
		b.tokens[i], b.tokens[j] = b.tokens[j], b.tokens[i]
	}

	// Energy Dampener (CONFIRMED, CB-002 C8).
	damp := false
	for _, t := range b.tokens {
		d := g.Designs[t.design]
		for _, s := range d.Slots {
			if s.Count > 0 && s.Part.Dampener {
				damp = true
			}
		}
	}
	rank := map[int]int{}
	for r, p := range b.players {
		rank[p] = r
	}
	squares := startSquares[min(len(b.players), 16)]
	for _, t := range b.tokens {
		if damp && !t.starbase {
			t.speed = max(0, t.speed-4)
		}
		if t.tactic == TacticDisengage {
			t.counter = 7
		}
		sq := squares[rank[t.player]%len(squares)]
		t.x, t.y = sq[0], sq[1]
	}
	return events
}

func (g *Game) fleetCargoCapacity(f *Fleet) int {
	c := 0
	for _, s := range f.Stacks {
		c += s.Count * g.Designs[s.Design].CargoCapacity
	}
	return c
}

// capTokens applies the 256-token cap (BINARY-ONLY): each player gets
// 255/players stacks, fleets beyond the quota are left out, then left-out
// fleets are added back while room remains. It returns the fleets that
// fight and the players with a fleet left out.
func (b *battle) capTokens(fleets []int) ([]int, []int) {
	g := b.g
	stacks := func(i int) int {
		n := 0
		for _, s := range g.Fleets[i].Stacks {
			if s.Count > 0 {
				n++
			}
		}
		return n
	}
	total := 0
	for _, i := range fleets {
		total += stacks(i)
	}
	if _, ok := g.battleStarbase(b.loc.planet); ok {
		total++
	}
	if total <= maxTokens {
		return fleets, nil
	}
	quota := 255 / len(b.players)
	used := map[int]int{}
	in := map[int]bool{}
	count := 0
	for _, i := range fleets {
		o := g.Fleets[i].Owner
		if used[o]+stacks(i) <= quota {
			used[o] += stacks(i)
			in[i] = true
			count += stacks(i)
		}
	}
	for _, i := range fleets {
		if !in[i] && count+stacks(i) <= maxTokens-1 {
			in[i] = true
			count += stacks(i)
		}
	}
	var out []int
	missed := map[int]bool{}
	var missedPlayers []int
	for _, i := range fleets {
		if in[i] {
			out = append(out, i)
		} else if o := g.Fleets[i].Owner; !missed[o] {
			missed[o] = true
			missedPlayers = append(missedPlayers, o)
		}
	}
	return out, missedPlayers
}

// livePlayers is the number of players with live tokens.
func (b *battle) livePlayers() int {
	seen := map[int]bool{}
	for _, t := range b.tokens {
		if t.live() {
			seen[t.player] = true
		}
	}
	return len(seen)
}

// fight runs the rounds (COMBAT.md "Rounds").
func (b *battle) fight() {
	for b.round = 0; b.round < maxRounds; b.round++ {
		if b.round > 0 {
			b.regenerate()
		}
		if b.livePlayers() <= 1 {
			return
		}
		b.move()
		for _, t := range b.tokens {
			if t.live() {
				t.jitter = b.rng.Intn(jitterSpan)
			}
		}
		b.in = map[int]bool{}
		for _, t := range b.tokens {
			if t.live() && b.hasPrey(t.player) {
				b.in[t.player] = true
			}
		}
		if len(b.in) <= 1 {
			return
		}
		b.fire()
	}
}

// hasPrey reports whether player p has a live token of a player it
// attacks.
func (b *battle) hasPrey(p int) bool {
	for _, t := range b.tokens {
		if t.live() && b.attacks(p, t.player) {
			return true
		}
	}
	return false
}

// regenerate is Regenerating Shields at the start of a round after the
// first (CONFIRMED, CB-007/008).
func (b *battle) regenerate() {
	for _, t := range b.tokens {
		if t.live() && t.regen && t.shield > 0 {
			t.shield = min(t.maxShield, t.shield+t.maxShield/10)
		}
	}
}
