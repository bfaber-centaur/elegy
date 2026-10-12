package engine

// withRules is g under the Elegy ruleset.
func withRules(g Game) Game {
	g.Rules = ElegyRules()
	return g
}

// GenerateTurnAtBattles is GenerateTurn, calling at with a clone of the
// game as the battle phase finds it and the tech levels its design costs
// use.
func GenerateTurnAtBattles(game Game, orders []PlayerOrders, rng Rand, at func(g Game, costLevels [][NumFields]int)) (TurnResult, error) {
	return generateTurn(game, orders, rng, turnOptions{beforeBattles: at})
}

// WaypointCheckForTest runs the waypoint check (KERNEL.md "Turn order"
// step 1a.3).
func (g *Game) WaypointCheckForTest() { g.waypointCheck() }
