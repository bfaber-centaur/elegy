package engine

// withRules is g under the Elegy ruleset.
func withRules(g Game) Game {
	g.Rules = ElegyRules()
	return g
}
