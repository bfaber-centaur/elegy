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
		events := g.levelEvents(0, before)
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
	if got := levelMessages(g.levelEvents(0, before)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := g.levelEvents(0, g.Players[0].Research.Levels); got != nil {
		t.Errorf("no levels gained: %+v, want none", got)
	}
}

// TestLevelEventsFieldOrder checks a year that raises energy 9 → 10 and
// electronics 18 → 19: Battle Nexus needs both, so it is announced after
// electronics 19, as if the fields rose in field order (ASSUMPTION,
// MESSAGES.md "Research order").
func TestLevelEventsFieldOrder(t *testing.T) {
	g := Game{Players: []Player{{Race: pgRace()}}}
	before := [NumFields]int{Energy: 9, Electronics: 18}
	g.Players[0].Research.Levels = before
	g.Players[0].Research.Levels[Energy] = 10
	g.Players[0].Research.Levels[Electronics] = 19
	got := levelMessages(g.levelEvents(0, before))
	if len(got) < 3 || !reflect.DeepEqual(got[:3], []string{"level 0 10", "part 0 Bear Neutrino Barrier", "part 0 Laser Battery"}) {
		t.Fatalf("energy 10 messages %q", got)
	}
	nexus, elec := -1, -1
	for i, m := range got {
		switch m {
		case "part 4 Battle Nexus":
			nexus = i
		case "level 4 19":
			elec = i
		case "part 0 Battle Nexus":
			t.Errorf("Battle Nexus announced at energy 10: %q", got)
		}
	}
	if elec < 0 || nexus < elec {
		t.Errorf("Battle Nexus at %d, electronics 19 at %d: %q", nexus, elec, got)
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
		if got := levelMessages(g.levelEvents(0, before)); !reflect.DeepEqual(got, c.want) {
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
