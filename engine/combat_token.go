package engine

// Battle tokens and their values (COMBAT.md "Board setup").

// Target classes of a token (COMBAT.md "Token values"). They share their
// numbers with the matching TargetType values.
const (
	classArmed         = int(TargetArmed)
	classBomber        = int(TargetBombersFreighters)
	classUnarmed       = int(TargetUnarmed)
	classFuelTransport = int(TargetFuelTransports)
	classFreighter     = int(TargetFreighters)
)

// weaponSlot is one weapon slot of a token.
type weaponSlot struct {
	part  Part
	count int
	init  int // weapon initiative
	reach int // part range, +1 on a starbase
}

// token is one stack (one design's ships from one fleet) or a starbase.
type token struct {
	player   int
	fleet    int // index into Game.Fleets, or -1 for a starbase
	stack    int // index into the fleet's Stacks
	planet   int // index into Game.Planets for a starbase, else -1
	design   int
	starbase bool

	ships     int
	dmg       Damage
	armor     int // per ship
	shield    int // current shield per ship
	maxShield int
	regen     bool // Regenerating Shields

	initiative int
	weapons    []weaponSlot
	computer   int
	jammer     int
	capacitor  int
	deflector  int
	class      int
	cost       int      // per ship: resources + boranium
	minerals   Minerals // per-ship mineral cost, for salvage

	tactic             Tactic
	primary, secondary TargetType

	speed   int
	mass    int
	jitter  int
	counter int // disengage counter
	moves   int // moves left this round
	x, y    int

	dead bool // destroyed
	left bool // disengaged off the board
}

func (t *token) live() bool { return !t.dead && !t.left }

func (t *token) armed() bool { return len(t.weapons) > 0 }

// longestReach and shortestReach are over the token's weapon slots.
func (t *token) longestReach() int {
	r := 0
	for _, w := range t.weapons {
		r = max(r, w.reach)
	}
	return r
}

func (t *token) shortestReach() int {
	r := -1
	for _, w := range t.weapons {
		if r < 0 || w.reach < r {
			r = w.reach
		}
	}
	return max(r, 0)
}

// existingDamage is the armor a stack has already lost, as target
// choice computes it: units·armor/10, then ·pct/10, then ·ships/500.
func (t *token) existingDamage() int {
	return t.dmg.Units * t.armor / 10 * t.dmg.Pct / 10 * t.ships / 500
}

// toughArmor is A in target choice: armor × ships less existing damage,
// at least 1.
func (t *token) toughArmor() int {
	return max(1, t.armor*t.ships-t.existingDamage())
}

// tokenValues computes a design's per-ship battle values (COMBAT.md
// "Token values"; CONFIRMED CB-000..CB-008 and the round-2 replays). cost
// is the design's cost to its owner (designCost).
func tokenValues(d Design, race Race, starbase bool, cost Cost) token {
	t := token{starbase: starbase, initiative: d.Hull.Initiative}
	rs := race.LRT.RegeneratingShields
	c := 0
	jam, jammed := 10000, false
	capac := 1000
	defl := 1000
	shield, armor := 0, d.Hull.Armor
	bomber := false
	for _, s := range d.Slots {
		p := s.Part
		if s.Count <= 0 {
			continue
		}
		if !p.weapon() {
			t.initiative += p.Initiative * s.Count
		}
		for range s.Count {
			if p.Computer > 0 {
				c += (100 - c) * p.Computer / 100
			}
			if p.Jammer > 0 {
				jam = jam * p.Jammer / 100
				jammed = true
			}
			if p.Capacitor > 0 {
				capac = min(2550, capac*(100+p.Capacitor)/100)
			}
			if p.Deflector {
				defl = defl * 90 / 100
			}
		}
		shield += p.Shield * s.Count
		if rs && p.Kind == PartArmor {
			armor += p.Armor / 2 * s.Count
		} else {
			armor += p.Armor * s.Count
		}
		if p.Kind == PartBomb {
			bomber = true
		}
	}
	t.initiative = min(63, t.initiative)
	for _, s := range d.Slots {
		if s.Count <= 0 || !s.Part.weapon() {
			continue
		}
		reach := s.Part.Range
		if starbase {
			reach++
		}
		t.weapons = append(t.weapons, weaponSlot{part: s.Part, count: s.Count,
			init: min(63, s.Part.Initiative+t.initiative), reach: reach})
	}
	t.computer = c
	if jammed {
		t.jammer = min(95, 100-(jam+50)/100)
		if starbase {
			t.jammer -= t.jammer / 4 // BINARY-ONLY
		}
	}
	t.capacitor = capac / 10
	t.deflector = defl / 10
	if rs {
		shield = shield * 7 / 5
	}
	t.shield, t.maxShield, t.armor, t.regen = shield, shield, armor, rs
	t.cost = cost.Resources + cost.Minerals[Boranium]
	t.minerals = cost.Minerals

	switch {
	case starbase:
		t.class = starbaseClass(t, false) // the battle applies the ruleset
	case len(t.weapons) > 0:
		t.class = classArmed
	case bomber:
		t.class = classBomber
	case d.Hull.FuelTransport:
		t.class = classFuelTransport
	case d.CargoCapacity > 0:
		t.class = classFreighter
	default:
		t.class = classUnarmed
	}
	return t
}

// starbaseClass is a starbase token's target class. legacy reproduces the
// original's LEGACY BUG that a starbase is always an "armed" target, even
// with no weapons (Legacy.StarbaseArmedClass; COMBAT.md "Starbases in
// battle", CONFIRMED CB-011..013 S4/S5). Without it an unarmed starbase is
// an unarmed target.
func starbaseClass(t token, legacy bool) int {
	if legacy || len(t.weapons) > 0 {
		return classArmed
	}
	return classUnarmed
}

// battleWarp is w in the speed code.
func battleWarp(e Engine) int {
	if e.BattleWarp10 {
		return 10
	}
	for w := 9; w >= 1; w-- {
		if e.Fuel[w] <= 120 {
			return w
		}
	}
	return 0
}

// speedCode is a ship design's battle speed code for a ship of mass
// (battleMass), before the dampener (COMBAT.md "Token values"; CONFIRMED
// by the designer, CB-000; War Monger and cargo in battle, CB-038; dump,
// CB-025).
func speedCode(d Design, race Race, mass int, dumped bool) int {
	if d.Engines <= 0 {
		return 0
	}
	thrust, half := 0, 0
	if d.Engine.EnigmaPulsar {
		half += d.Engines
	}
	for _, s := range d.Slots {
		thrust += s.Part.Thrust * s.Count
		if s.Part.HalfThrust {
			half += s.Count
		}
	}
	code := battleWarp(d.Engine) - 4 + thrust + (half+1)/2 - mass/70/d.Engines
	if race.PRT == PRTWarMonger {
		code += 2
	}
	if dumped && d.CargoCapacity > 0 {
		code--
	}
	return min(8, max(0, code))
}

// movesInRound is the number of squares a token with speed code s moves
// in round r (COMBAT.md "Moves per round", CONFIRMED CB-000..CB-008).
func movesInRound(s, r int) int {
	m := (s + 2) / 4
	switch s % 4 {
	case 0:
		if r%2 == 0 {
			m++
		}
	case 1:
		if r%4 != 2 {
			m++
		}
	case 3:
		if r%4 == 0 {
			m++
		}
	}
	return m
}

// matches reports whether a token of class c (or a starbase) is a target
// of type tt.
func (t *token) matches(tt TargetType) bool {
	switch tt {
	case TargetAny:
		return true
	case TargetStarbase:
		return t.starbase
	case TargetArmed:
		return t.class == classArmed
	case TargetBombersFreighters:
		return t.class == classBomber || t.class == classFreighter
	case TargetUnarmed:
		return t.class == classUnarmed || t.class == classFuelTransport || t.class == classFreighter
	case TargetFuelTransports:
		return t.class == classFuelTransport
	case TargetFreighters:
		return t.class == classFreighter
	}
	return false
}
