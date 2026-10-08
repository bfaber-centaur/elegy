package engine

import "testing"

func TestConfirmedScrapMinerals(t *testing.T) {
	// TAKEOVER.md "Other waypoint tasks" (CONFIRMED T-34): per mineral of
	// the fleet's cost C, 4C/5 at a starbase, C/3 without, 9C/10 and 9C/20
	// when the planet's owner has Ultimate Recycling, C/3 as salvage in
	// deep space; mineral cargo on top; colonists join only the owner's
	// own planet.
	for _, tt := range []struct {
		name               string
		owner              int // planet owner, NoOwner, or −2 for deep space
		starbase, ur       bool
		num, den, colonist int
	}{
		{"own starbase", 0, true, false, 4, 5, 10},
		{"own planet", 0, false, false, 1, 3, 10},
		{"UR starbase", 1, true, true, 9, 10, 0},
		{"UR planet", 1, false, true, 9, 20, 0},
		{"unowned", NoOwner, false, false, 1, 3, 0},
		{"deep space", -2, false, false, 1, 3, 0},
	} {
		l := newTKLab(t, 3)
		l.g.Players[1].Race.LRT.UltimateRecycling = tt.ur
		scout := l.design("Scout", SlotFill{0, "Long Hump 6", 1})
		pi := l.planet(max(tt.owner, NoOwner), 0, 100)
		if tt.owner == -2 {
			l.g.Planets[pi].Pos = Point{500, 500}
		}
		if tt.starbase {
			l.g.Planets[pi].HasStarbase = true
			l.g.Planets[pi].StarbaseDesign = l.design("Orbital Fort")
		}
		fi := l.fleet(0, 0, 0, Stack{Design: scout, Count: 30})
		if tt.owner == -2 {
			l.g.Fleets[fi].Pos = Point{0, 0}
			l.g.Planets = l.g.Planets[:0]
			pi = -1
		}
		l.g.Fleets[fi].Cargo = Cargo{Minerals: Minerals{5, 0, 0}, Colonists: 10}
		l.g.Fleets[fi].Task = Task{Kind: TaskScrap}
		c := designCost(l.g.Designs[scout], l.g.Players[0].Race, l.g.Players[0].Research.Levels)
		var want Minerals
		for m := range NumMinerals {
			want[m] = 30 * c.Minerals[m] * tt.num / tt.den
		}
		want[Ironium] += 5
		g := l.turn(&seqRand{}).Game
		if len(g.Fleets) != 0 {
			t.Errorf("%s: fleet not scrapped", tt.name)
		}
		if pi < 0 {
			if len(g.Salvage) != 1 || g.Salvage[0].Minerals != want || g.Salvage[0].Owner != 0 {
				t.Errorf("%s: salvage %+v, want one object of player 0 with %v", tt.name, g.Salvage, want)
			}
			continue
		}
		p := g.Planets[pi]
		if p.Surface != want {
			t.Errorf("%s: surface %v, want %v", tt.name, p.Surface, want)
		}
		if got := p.Population - l.g.Planets[pi].Population; tt.owner >= 0 && got < tt.colonist {
			t.Errorf("%s: population rose by %d, want at least the %d colonists", tt.name, got, tt.colonist)
		}
	}
}

func TestConfirmedScrapRecycledResources(t *testing.T) {
	// KERNEL.md "Production" (CONFIRMED KB-2A): x 2,410, r 500 → 914.
	if got := recycledResources(500, 2410); got != 914 {
		t.Errorf("recycledResources(500, 2410) = %d, want 914", got)
	}
	if got := recycledResources(500, 0); got != 500 {
		t.Errorf("no scrap: %d, want 500", got)
	}
}

func TestScrapOnlyBeforeMovement(t *testing.T) {
	// TAKEOVER.md (CONFIRMED T-34): a fleet that arrives with a scrap order
	// is intact at the end of that year and scrapped the next.
	l := newTKLab(t, 3)
	scout := l.design("Scout", SlotFill{0, "Long Hump 6", 1})
	pi := l.planet(0, 0, 100)
	l.arriving(0, pi, scout, 0, Task{Kind: TaskScrap})
	g := l.turn(&seqRand{}).Game
	if len(g.Fleets) != 1 || g.Fleets[0].Pos != g.Planets[pi].Pos {
		t.Fatalf("after arriving: fleets %+v, want the scout at the planet", g.Fleets)
	}
	l.g = g
	if g = l.turn(&seqRand{}).Game; len(g.Fleets) != 0 {
		t.Errorf("next year: %d fleets, want the scout scrapped", len(g.Fleets))
	}
}
