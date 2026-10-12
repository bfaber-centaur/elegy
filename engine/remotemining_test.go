package engine

import (
	"slices"
	"testing"
)

// remoteMineGame is pgHomeworld with a second player, an unowned planet
// 30 ly east, and fleet 1 of the given owner holding a remote-mining
// task at pos with no further waypoints.
func remoteMineGame(t *testing.T, owner int, pos Point, rate int) (Game, *stubTerraform) {
	t.Helper()
	g := pgHomeworld()
	scout, _ := slScouts(t)
	g.Designs = []Design{scout}
	g.Players = append(g.Players, Player{Race: pgRace(), Research: ResearchState{Next: NextSameField}})
	free := Planet{ID: 8, Owner: NoOwner, Pos: Point{g.Planets[0].Pos.X + 30, g.Planets[0].Pos.Y}}
	g.Planets = append(g.Planets, free)
	g.Fleets = []Fleet{{ID: 1, Number: 1, Owner: owner, Pos: pos, Fuel: 50,
		Stacks: []Stack{{Design: 0, Count: 1}}, Task: Task{Kind: TaskRemoteMine}}}
	stub := &stubTerraform{rate: rate}
	g.Terraform = stub
	return g, stub
}

// KERNEL.md "Remote mining" (stars-elegy b1e38ef) and MESSAGES.md
// 0x075–0x077 and 0x04e (a): a stationary remote-mining fleet mines only
// where it may; otherwise its order is refused with a message, its task
// cancelled, and, at its last waypoint, 0x04e follows (oracle 0x076 then
// 0x04e). Each case's label is its spec evidence.
func TestRemoteMiningRefusals(t *testing.T) {
	home := pgHomeworld().Planets[0]
	free := Point{home.Pos.X + 30, home.Pos.Y}
	deep := Point{home.Pos.X + 15, home.Pos.Y + 15}
	for _, c := range []struct {
		name    string
		owner   int
		ar      int // the Alternate Reality player, or -1
		pos     Point
		rate    int
		mined   bool
		task    TaskKind
		refusal EventKind // 0: none
		planet  int       // the refusal's planet, -1 for none
	}{
		{"unowned planet mines (CONFIRMED T-35)", 0, -1, free, 1, true, TaskRemoteMine, 0, -1},
		{"no modules: 0x075 (CONFIRMED CS-003)", 0, -1, free, 0, false, TaskNone, EventRemoteMineNoModules, 8},
		{"own planet, not AR: 0x076 (CONFIRMED TK-111)", 0, -1, home.Pos, 1, false, TaskNone, EventRemoteMineInhabited, home.ID},
		{"another player's planet: 0x076 (CONFIRMED TK-111)", 1, -1, home.Pos, 1, false, TaskNone, EventRemoteMineInhabited, home.ID},
		{"deep space: 0x077 (BINARY-ONLY)", 0, -1, deep, 1, false, TaskNone, EventRemoteMineDeepSpace, -1},
		{"AR at another player's planet: skipped silently (BINARY-ONLY; keeps task ASSUMPTION)", 1, 1, home.Pos, 1, false, TaskRemoteMine, 0, -1},
		{"AR at its own planet mines (CONFIRMED KB-1B)", 0, 0, home.Pos, 1, true, TaskRemoteMine, 0, -1},
		{"another player's miners at an AR planet: 0x076 (BINARY-ONLY)", 1, 0, home.Pos, 1, false, TaskNone, EventRemoteMineInhabited, home.ID},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, stub := remoteMineGame(t, c.owner, c.pos, c.rate)
			if c.ar >= 0 {
				g.Players[c.ar].Race.PRT = PRTAlternateReality
			}
			r, err := GenerateTurn(withRules(g), nil, highRand{})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(stub.mined) == 1; got != c.mined {
				t.Errorf("mined %v, want %v", stub.mined, c.mined)
			}
			fi := slices.IndexFunc(r.Game.Fleets, func(f Fleet) bool { return f.ID == 1 })
			if k := r.Game.Fleets[fi].Task.Kind; k != c.task {
				t.Errorf("task %v, want %v", k, c.task)
			}
			type msg struct{ kind, player, planet int }
			var got []msg
			for _, e := range r.Events {
				if e.Fleet == 1 && (e.Kind == EventFleetArrived || e.Kind >= EventRemoteMineNoModules && e.Kind <= EventRemoteMineDeepSpace) {
					got = append(got, msg{int(e.Kind), e.Player, e.Planet})
				}
			}
			var want []msg
			if c.refusal != 0 {
				want = []msg{{int(c.refusal), c.owner, c.planet}, {int(EventFleetArrived), c.owner, -1}}
			}
			if !slices.Equal(got, want) {
				t.Errorf("events %+v, want %+v", got, want)
			}
		})
	}
}

// MESSAGES.md Notes: 0x04e follows a refusal only at the fleet's last
// waypoint (DOCUMENTED there; this case is not oracle-run). A refused
// fleet with a waypoint still ahead that cannot move (no fuel) gets the
// refusal alone.
func TestRemoteMiningRefusalBeforeLastWaypoint(t *testing.T) {
	home := pgHomeworld().Planets[0]
	g, _ := remoteMineGame(t, 0, home.Pos, 1)
	g.Fleets[0].Fuel = 0
	g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{home.Pos.X + 30, home.Pos.Y}, Warp: 5}}
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []EventKind
	for _, e := range r.Events {
		if e.Fleet == 1 && (e.Kind == EventFleetArrived || e.Kind == EventRemoteMineInhabited) {
			kinds = append(kinds, e.Kind)
		}
	}
	if !slices.Equal(kinds, []EventKind{EventRemoteMineInhabited}) {
		t.Errorf("events %v, want only the refusal", kinds)
	}
}

// KERNEL.md "Remote mining": the checks run only for a fleet that did not
// move this year. A fleet that moved, here from an owned planet to deep
// space, keeps its task and gets no refusal.
func TestRemoteMiningMovedFleetNotChecked(t *testing.T) {
	home := pgHomeworld().Planets[0]
	g, stub := remoteMineGame(t, 0, home.Pos, 0)
	deep := Point{home.Pos.X + 15, home.Pos.Y + 15}
	g.Fleets[0].Waypoints = []Waypoint{{Pos: deep, Warp: 5, Task: Task{Kind: TaskRemoteMine}}}
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	f := r.Game.Fleets[slices.IndexFunc(r.Game.Fleets, func(f Fleet) bool { return f.ID == 1 })]
	if f.Pos == home.Pos || f.Task.Kind != TaskRemoteMine || len(stub.mined) != 0 {
		t.Errorf("pos %v task %v mined %v; want moved, task kept, nothing mined", f.Pos, f.Task.Kind, stub.mined)
	}
	for _, e := range r.Events {
		if e.Kind >= EventRemoteMineNoModules && e.Kind <= EventRemoteMineDeepSpace {
			t.Errorf("refusal %+v for a fleet that moved", e)
		}
	}
}
