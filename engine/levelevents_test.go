package engine

import (
	"fmt"
	"reflect"
	"testing"
)

// ownedItems stands in for space objects: the level-up messages only ask
// which Mystery Trader parts a player owns.
type ownedItems struct {
	SpaceObjects
	items map[string]bool
}

func (o ownedItems) TraderItems(int) map[string]bool { return o.items }

// levelMessages is the level-up events as "level f n" and "part f name".
func levelMessages(events []Event) []string {
	var out []string
	for _, ev := range events {
		switch ev.Kind {
		case EventLevelUp:
			out = append(out, fmt.Sprintf("level %d %d", ev.Item, ev.Count))
		case EventPartAvailable:
			out = append(out, fmt.Sprintf("part %d %s", ev.Item, Components().Components[ev.Count].Name))
		}
	}
	return out
}

// TestLevelEventsAnnounceParts checks the level-up messages (MESSAGES.md
// 0x050 and 0x05f at stars-elegy b4c0086) for energy 9 → 10: the level,
// then the components whose energy requirement is 10, in catalog order.
// Battle Nexus (energy 10, electronics 19) is announced only once
// electronics is 19 (MEASURED KX-005 R1: not announced at electronics
// below 19).
func TestLevelEventsAnnounceParts(t *testing.T) {
	for _, c := range []struct {
		name        string
		electronics int
		want        []string
	}{
		{"electronics 18", 18, []string{"level 0 10", "part 0 Bear Neutrino Barrier", "part 0 Laser Battery"}},
		{"electronics 19", 19, []string{"level 0 10", "part 0 Bear Neutrino Barrier", "part 0 Battle Nexus", "part 0 Laser Battery"}},
	} {
		g := Game{Players: []Player{{Race: pgRace()}}}
		before := [NumFields]int{Energy: 9, Electronics: c.electronics}
		g.Players[0].Research.Levels = before
		g.Players[0].Research.Levels[Energy] = 10
		events := g.levelEvents(0, before, fieldOrder(before, g.Players[0].Research.Levels))
		if got := levelMessages(events); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
		for _, ev := range events {
			if ev.Player != 0 || ev.Planet != -1 || ev.Fleet != -1 {
				t.Errorf("%s: event %+v, want player 0, no planet or fleet", c.name, ev)
			}
		}
	}
}

// TestLevelEventsSeveralLevels checks that each level gained has its own
// message, followed by its own components (MESSAGES.md 0x050 "One message
// per level", "Research order"): energy 2 → 4 announces Cow-hide Shield
// (energy 3) after level 3; level 4 has no component this race may use
// (Mass Driver 5 is Packet Physics only).
func TestLevelEventsSeveralLevels(t *testing.T) {
	g := Game{Players: []Player{{Race: pgRace()}}}
	before := [NumFields]int{Energy: 2}
	g.Players[0].Research.Levels = before
	g.Players[0].Research.Levels[Energy] = 4
	want := []string{"level 0 3", "part 0 Cow-hide Shield", "level 0 4"}
	if got := levelMessages(g.levelEvents(0, before, fieldOrder(before, g.Players[0].Research.Levels))); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := g.levelEvents(0, g.Players[0].Research.Levels, nil); got != nil {
		t.Errorf("no levels gained: %+v, want none", got)
	}
}

// TestLevelEventsChainOrder checks a year whose research levels biotech,
// the current field, and whose leftover then levels weapons, the next
// field: biotech's level comes first, then weapons', though weapons comes
// first in field order (MEASURED, stars-elegy d89b8a3 MESSAGES.md EM-1:
// RG-002 2468 gave biotech 7, its parts, then weapons 8, its parts).
// Radiation Terraform ±7
// (weapons 5, biotech 2) is announced at weapons 5 because biotech 2 came
// first; in field order it would not be announced at all.
func TestLevelEventsChainOrder(t *testing.T) {
	r := pgRace()
	s := ResearchState{Current: Biotech, Next: Weapons}
	s.Levels = [NumFields]int{Weapons: 4, Biotech: 1}
	// Biotech 2 then weapons 5, with nothing to spare for weapons 6.
	bio := ResearchLevelCost(2, 5, r.ResearchCosts[Biotech], false)
	weap := ResearchLevelCost(5, 6, r.ResearchCosts[Weapons], false)
	after, gained := addResearch(s, r, bio+weap, false)
	if !reflect.DeepEqual(gained, []int{Biotech, Weapons}) || after.Levels[Biotech] != 2 || after.Levels[Weapons] != 5 {
		t.Fatalf("setup: gained %v, levels %v; want biotech then weapons, to 2 and 5", gained, after.Levels)
	}
	g := Game{Players: []Player{{Race: r, Research: after}}}
	got := levelMessages(g.levelEvents(0, s.Levels, gained))
	want := []string{"level 5 2", "level 1 5", "part 1 Black Cat Bomb", "part 1 Radiation Terraform ±7"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestLevelEventsTerraformByOtherField checks that a terraforming part is
// announced by its non-biotech field: Gravity Terraform ±7 (propulsion 5,
// biotech 2) on propulsion 4 → 5 (MEASURED, stars-elegy d89b8a3
// MESSAGES.md EM-1: Gravity ±7 was announced on propulsion 5).
func TestLevelEventsTerraformByOtherField(t *testing.T) {
	g := Game{Players: []Player{{Race: pgRace()}}}
	before := [NumFields]int{Propulsion: 4, Biotech: 2}
	g.Players[0].Research.Levels = before
	g.Players[0].Research.Levels[Propulsion] = 5
	got := levelMessages(g.levelEvents(0, before, []int{Propulsion}))
	if len(got) == 0 || got[0] != "level 2 5" || !contains(got, "part 2 Gravity Terraform ±7") {
		t.Errorf("got %q, want level 2 5 announcing Gravity Terraform ±7", got)
	}
}

// TestLevelEventsRaceParts checks that a component the race may not use
// is not announced: Super Driver 8 (energy 11) is Packet Physics only, and
// Multi Function Pod (energy, propulsion and electronics 11) is a Mystery
// Trader part, announced only to a player who owns it (MESSAGES.md 0x05f
// "one per component the race may use").
func TestLevelEventsRaceParts(t *testing.T) {
	for _, c := range []struct {
		name string
		prt  PRT
		own  bool
		want []string
	}{
		{"jack of all trades", PRTJackOfAllTrades, false, []string{"level 0 11"}},
		{"packet physics", PRTPacketPhysics, false, []string{"level 0 11", "part 0 Super Driver 8"}},
		{"owns the pod", PRTJackOfAllTrades, true, []string{"level 0 11", "part 0 Multi Function Pod"}},
	} {
		r := pgRace()
		r.PRT = c.prt
		g := Game{Players: []Player{{Race: r}}, Objects: ownedItems{items: map[string]bool{"Multi Function Pod": c.own}}}
		before := [NumFields]int{Energy: 10, Propulsion: 11, Electronics: 11}
		g.Players[0].Research.Levels = before
		g.Players[0].Research.Levels[Energy] = 11
		if got := levelMessages(g.levelEvents(0, before, fieldOrder(before, g.Players[0].Research.Levels))); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// TestTurnLevelEvents checks that a whole turn reports a level gained in
// the research step: pgHomeworld's player 0 stores 65 energy research at
// level 2, and a stored 199 reaches level 3 (cost 200) with this year's
// research.
func TestTurnLevelEvents(t *testing.T) {
	g := pgHomeworld()
	g.Players[0].Research.Accumulated[Energy] = 199
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Game.Players[0].Research.Levels[Energy]; got != 3 {
		t.Fatalf("setup: energy level %d, want 3", got)
	}
	got := levelMessages(r.Events)
	if want := []string{"level 0 3", "part 0 Cow-hide Shield"}; len(got) < 2 || !reflect.DeepEqual(got[:2], want) {
		t.Errorf("level messages %q, want them to start %q", got, want)
	}
}

// TestTurnOvercrowded checks the overcrowding message (MESSAGES.md 0x026
// at stars-elegy b4c0086, CONFIRMED KX-002): a planet over capacity with a
// hab value of 0 or more that shrinks and survives reports the loss; a
// hostile planet (0x025 instead), a planet that grows or stays the
// same, and one that dies out do not.
func TestTurnOvercrowded(t *testing.T) {
	for _, c := range []struct {
		name string
		env  [3]int
		pop  int
		want bool
	}{
		{"hab 100, over capacity", [3]int{50, 50, 50}, 20000, true},
		{"hab 0, over capacity", [3]int{15, 15, 15}, 20000, true},
		{"hab 100, growing", [3]int{50, 50, 50}, 486, false},
		{"hab 100, at capacity (unchanged)", [3]int{50, 50, 50}, MaxPopulation(pgRace(), 100, 0), false},
		{"hostile", [3]int{0, 50, 50}, 20000, false},
		{"hostile, dies out", [3]int{0, 0, 0}, 1, false},
	} {
		g := pgHomeworld()
		g.Planets[0].Env, g.Planets[0].Population = c.env, c.pop
		r, err := GenerateTurn(withRules(g), nil, highRand{})
		if err != nil {
			t.Fatal(err)
		}
		end := r.Game.Planets[0].Population
		var got []Event
		for _, ev := range r.Events {
			if ev.Kind == EventOvercrowded {
				got = append(got, ev)
			}
		}
		if !c.want {
			if got != nil {
				t.Errorf("%s: population %d → %d: unexpected %+v", c.name, c.pop, end, got)
			}
			continue
		}
		want := []Event{{Kind: EventOvercrowded, Player: 0, Planet: 7, Fleet: -1, Count: c.pop - end}}
		if end <= 0 || end >= c.pop || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: population %d → %d: events %+v, want %+v", c.name, c.pop, end, got, want)
		}
	}
}

// TestTurnLevelEventsAfterDrops checks the level-up messages of the last
// level-up check, after drops resolve (KERNEL.md "Turn order"): player 1
// captures player 0's artifact planet, and the research it gains there
// (the artifact, and any tech from the capture) takes energy up a level
// that the earlier checks did not reach (MESSAGES.md 0x050 "also after
// battles/invasions").
func TestTurnLevelEventsAfterDrops(t *testing.T) {
	l := newTKLab(t, 3)
	l.g.RandomEvents = true
	pi := l.planet(0, 0, 1)
	l.g.Planets[pi].Artifact = true
	pl := &l.g.Players[1]
	pl.Research.Accumulated[Energy] = ResearchLevelCost(4, pl.Research.levelSum(), pl.Race.ResearchCosts[Energy], false) - 1
	freighter := l.design("Small Freighter", SlotFill{0, "Quick Jump 5", 1})
	l.arriving(1, pi, freighter, 50, unloadColonists)
	r := l.turn(&seqRand{})
	if p := r.Game.Planets[pi]; p.Owner != 1 || p.Artifact || r.Game.Players[1].Research.Levels[Energy] < 4 {
		t.Fatalf("setup: owner %d, artifact %v, energy %d; want 1, found, 4 or more", p.Owner, p.Artifact, r.Game.Players[1].Research.Levels[Energy])
	}
	var found bool
	for _, ev := range r.Events {
		if ev.Kind == EventLevelUp && ev.Player == 1 && ev.Item == ItemKind(Energy) && ev.Count == 4 {
			found = true
		}
	}
	if !found {
		t.Errorf("no energy 4 level-up for player 1 in %+v", r.Events)
	}
}

// TestTurnLevelEventsChainOrder checks that a whole turn announces the
// research step's levels in the order gained: biotech, the current field,
// then weapons, which the leftover reaches (MEASURED, stars-elegy d89b8a3
// MESSAGES.md EM-1).
func TestTurnLevelEventsChainOrder(t *testing.T) {
	g := pgHomeworld()
	pl := &g.Players[0]
	pl.Research.Current, pl.Research.Next = Biotech, Weapons
	pl.Research.Levels[Weapons], pl.Research.Levels[Biotech] = 4, 1
	sum := pl.Research.levelSum()
	pl.Research.Accumulated = [NumFields]int{
		Biotech: ResearchLevelCost(2, sum, pl.Race.ResearchCosts[Biotech], false) - 1,
		// Short of weapons 5 now; the leftover moved after biotech 2 covers
		// it, though the level costs 10 more then.
		Weapons: ResearchLevelCost(5, sum, pl.Race.ResearchCosts[Weapons], false) - 1,
	}
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range levelMessages(r.Events) {
		if m == "level 5 2" || m == "level 1 5" {
			got = append(got, m)
		}
	}
	if want := []string{"level 5 2", "level 1 5"}; !reflect.DeepEqual(got, want) {
		t.Errorf("level messages %q, want %q (levels now %v)", got, want, r.Game.Players[0].Research.Levels)
	}
}

// TestFieldOrder checks the order LevelUpCheck's levels are announced in:
// each field in turn, energy first, a field's levels together.
func TestFieldOrder(t *testing.T) {
	before := [NumFields]int{Energy: 3, Weapons: 1, Biotech: 2}
	after := [NumFields]int{Energy: 4, Weapons: 3, Biotech: 2}
	if got, want := fieldOrder(before, after), []int{Energy, Weapons, Weapons}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
