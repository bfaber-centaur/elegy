package engine

// remarkHomeworlds is KERNEL.md "Turn order" step 3.6: every planet's
// homeworld mark is cleared, then set on the planet each player's own
// record names, whoever owns it now (TAKEOVER.md "The homeworld mark
// after capture", CONFIRMED T-41; MEASURED WU-RSG2, stars-elegy 52af513:
// a cleared mark came back on the recorded planet after the year). So a
// captured homeworld keeps its mark, the old owner gets no new one, and
// the mark stays on an unowned planet or a dead player's homeworld.
func (g *Game) remarkHomeworlds() {
	for i := range g.Planets {
		g.Planets[i].Homeworld = false
	}
	for _, p := range g.Players {
		if !p.HasHomeworld {
			continue
		}
		if pi := g.planetIndex(p.Homeworld); pi >= 0 {
			g.Planets[pi].Homeworld = true
		}
	}
}
