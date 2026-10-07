package engine

// Relation is how one player regards another. The values follow
// COMBAT.md's oracle specs (0 friend, 1 neutral, 2 enemy).
type Relation int

const (
	RelationFriend Relation = iota
	RelationNeutral
	RelationEnemy
)

// relation is player a's view of player b.
func (g *Game) relation(a, b int) Relation {
	if a == b {
		return RelationFriend
	}
	if r := g.Players[a].Relations; b < len(r) {
		return r[b]
	}
	return RelationNeutral
}

// Tactic is a battle plan's tactic (COMBAT.md numbering).
type Tactic int

const (
	TacticDisengage Tactic = iota
	TacticDisengageIfChallenged
	TacticMinimizeDamage
	TacticMaximizeNet
	TacticMaximizeRatio
	TacticMaximizeDamage
)

// TargetType is a battle plan's primary or secondary target.
type TargetType int

const (
	TargetNone TargetType = iota
	TargetAny
	TargetStarbase
	TargetArmed
	TargetBombersFreighters
	TargetUnarmed
	TargetFuelTransports
	TargetFreighters
)

// AttackWho selects whom a plan attacks.
type AttackWho int

const (
	AttackNobody AttackWho = iota
	AttackEnemies
	AttackNeutralsAndEnemies
	AttackEveryone
	AttackPlayer // the player in BattlePlan.Player
)

// BattlePlan is one of a player's battle plans.
type BattlePlan struct {
	Name               string
	Tactic             Tactic
	Primary, Secondary TargetType
	Attack             AttackWho
	Player             int // for AttackPlayer
	DumpCargo          bool
}

// plan returns a player's plan i, or the zero plan (attack nobody).
func (g *Game) plan(player, i int) BattlePlan {
	if p := g.Players[player].Plans; i >= 0 && i < len(p) {
		return p[i]
	}
	return BattlePlan{}
}

// Salvage is a deep-space salvage object.
type Salvage struct {
	Pos      Point
	Minerals Minerals
}
