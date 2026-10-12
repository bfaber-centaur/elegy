package engine

import (
	"reflect"
	"testing"
)

// artifactLab is an unowned artifact planet in a game with random events
// on.
func artifactLab(t *testing.T) (*tkLab, int) {
	l := newTKLab(t, 3)
	l.g.RandomEvents = true
	pi := l.planet(NoOwner, 0, 0)
	l.g.Planets[pi].Artifact = true
	return l, pi
}

func TestConfirmedArtifactColony(t *testing.T) {
	// TAKEOVER.md "Ancient artifacts" (CONFIRMED TK-303; stars-elegy
	// 7c06f47): a colony of 2,500 colonists finds the artifact: field
	// rand(6) first, then 100 + rand(301) points, message 0x05e after the
	// landing; the artifact is gone.
	l, pi := artifactLab(t)
	before := l.g.Players[0].Research.Accumulated
	rng := &scriptRand{draws: []int{Propulsion, 222}}
	ev := l.g.resolveQueue([]drop{{pi, 0, 25}}, rng, map[int]bool{})
	want := Event{Kind: EventArtifactFound, Player: 0, Planet: l.g.Planets[pi].ID, Fleet: -1, Item: ItemKind(Propulsion), Count: 322}
	if rng.calls != 2 || len(ev) == 0 || !reflect.DeepEqual(ev[len(ev)-1], want) {
		t.Fatalf("%d draws, events %+v; want 2 draws ending in %+v", rng.calls, ev, want)
	}
	got := l.g.Players[0].Research.Accumulated
	if l.g.Planets[pi].Artifact || got[Propulsion]-before[Propulsion] != 322 {
		t.Errorf("artifact kept %v, propulsion %d → %d; want gone and +322", l.g.Planets[pi].Artifact, before[Propulsion], got[Propulsion])
	}
}

func TestMeasuredArtifactScaledBelow1000(t *testing.T) {
	// TK-303 A3 (MEASURED): below 1,000 colonists the points are scaled
	// by pop/1000; 500 colonists turn 100 + 300 into 200, 100 + 0 into
	// 50. ASSUMPTION A1: the scaling truncates (100 + 99 → 99).
	for _, c := range []struct{ units, draw, want int }{{5, 300, 200}, {5, 0, 50}, {5, 99, 99}, {10, 0, 100}, {9, 1, 90}} {
		l, pi := artifactLab(t)
		ev := l.g.resolveQueue([]drop{{pi, 0, c.units}}, &scriptRand{draws: []int{Energy, c.draw}}, map[int]bool{})
		if e := ev[len(ev)-1]; e.Kind != EventArtifactFound || e.Count != c.want || l.g.Players[0].Research.Accumulated[Energy] != c.want {
			t.Errorf("%d00 colonists, draw %d: %+v, energy %d; want %d", c.units, c.draw, e, l.g.Players[0].Research.Accumulated[Energy], c.want)
		}
	}
}

func TestConfirmedArtifactCaptureAndDefender(t *testing.T) {
	// TK-306 C1, C2 (CONFIRMED): a capture gives the artifact to the
	// captor, after its tech attempt (draw order BINARY-ONLY); a defender
	// that beats the drop off finds it itself.
	l, pi := groundLab(t, 3, 100, 0)
	l.g.RandomEvents, l.g.Planets[pi].Artifact = true, true
	// Player 0 already gained this year, so the capture's tech attempt
	// makes no draws; the artifact's are the only two.
	rng := &scriptRand{draws: []int{Electronics, 0}}
	ev := l.g.resolveQueue([]drop{{pi, 0, 300}}, rng, map[int]bool{0: true})
	if e := ev[len(ev)-1]; l.g.Planets[pi].Owner != 0 || rng.calls != 2 || e.Kind != EventArtifactFound || e.Player != 0 || e.Item != ItemKind(Electronics) || e.Count != 100 {
		t.Errorf("capture: owner %d, %d draws, last event %+v; want player 0's 100 electronics", l.g.Planets[pi].Owner, rng.calls, e)
	}

	l, pi = groundLab(t, 3, 100, 0)
	l.g.RandomEvents, l.g.Planets[pi].Artifact = true, true
	rng = &scriptRand{draws: []int{Biotech, 300}}
	ev = l.g.resolveQueue([]drop{{pi, 0, 10}}, rng, map[int]bool{})
	if e := ev[len(ev)-1]; l.g.Planets[pi].Owner != 1 || l.g.Planets[pi].Artifact || e.Kind != EventArtifactFound || e.Player != 1 || e.Count != 400 || l.g.Players[1].Research.Accumulated[Biotech] != 400 {
		t.Errorf("defender: owner %d, artifact %v, last event %+v; want player 1's 400 biotechnology", l.g.Planets[pi].Owner, l.g.Planets[pi].Artifact, e)
	}
}

func TestArtifactKept(t *testing.T) {
	// No find with random events off (BINARY-ONLY), nor when nobody owns
	// the planet after the drops (a tie empties it); the artifact stays
	// and nothing is drawn.
	l, pi := artifactLab(t)
	l.g.RandomEvents = false
	l.g.resolveQueue([]drop{{pi, 0, 25}}, panicRand{}, map[int]bool{})
	if p := l.g.Planets[pi]; p.Owner != 0 || !p.Artifact {
		t.Errorf("random events off: owner %d, artifact %v; want 0 and kept", p.Owner, p.Artifact)
	}

	l, pi = artifactLab(t)
	l.g.resolveQueue([]drop{{pi, 0, 25}, {pi, 1, 25}}, panicRand{}, map[int]bool{})
	if p := l.g.Planets[pi]; p.Owner != NoOwner || !p.Artifact {
		t.Errorf("tie: owner %d, artifact %v; want unowned and kept", p.Owner, p.Artifact)
	}
}

func TestMeasuredArtifactNotHalvedBySlowerTech(t *testing.T) {
	// TK-304 (MEASURED; LEGACY BUG, Legacy.ArtifactFullSlowerTech, on in
	// both built-ins): slower tech gives the same points.
	for _, r := range []Ruleset{ElegyRules(), FaithfulRules()} {
		l, pi := artifactLab(t)
		l.g.Rules, l.g.SlowerTech = r, true
		l.g.resolveQueue([]drop{{pi, 0, 25}}, &scriptRand{draws: []int{Weapons, 200}}, map[int]bool{})
		if got := l.g.Players[0].Research.Accumulated[Weapons]; got != 300 {
			t.Errorf("%s: weapons %d, want 300", r.ID, got)
		}
	}
}

func TestElegyChoiceArtifactHalvedSwitchOff(t *testing.T) {
	// ELEGY CHOICE, no built-in: with ArtifactFullSlowerTech off, slower
	// tech stores half the points, rounded up (301 → 151); the message
	// keeps 301. Without slower tech the switch changes nothing.
	for _, c := range []struct {
		slower bool
		want   int
	}{{true, 151}, {false, 301}} {
		l, pi := artifactLab(t)
		l.g.Rules.Legacy.ArtifactFullSlowerTech = false
		l.g.SlowerTech = c.slower
		ev := l.g.resolveQueue([]drop{{pi, 0, 25}}, &scriptRand{draws: []int{Weapons, 201}}, map[int]bool{})
		if got := l.g.Players[0].Research.Accumulated[Weapons]; got != c.want || ev[len(ev)-1].Count != 301 {
			t.Errorf("slower tech %v: weapons %d, message %d; want %d and 301", c.slower, got, ev[len(ev)-1].Count, c.want)
		}
	}
}
