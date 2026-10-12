package engine

import (
	"slices"
	"testing"
)

// MESSAGES.md 0x04e: a fleet that reaches its last waypoint at the end of
// movement is told it has finished its orders (b), but not when its task
// is transport, colonize, remote mining, scrap, lay mines or patrol. A
// transport or colonize task that then ends in the year's waypoint tasks,
// done or refused, tells it then, after the refusal (a). Each row's label
// is its evidence: RM-4 (stars-elegy #167 at f2650a7) for remote mining,
// TK-201 H for colonize; DOCUMENTED rows follow the 0x04e row's wording
// and were not oracle-run as such. The fleet arrives 10 ly east of its
// waypoint's planet and keeps any task that does not end.
func TestFinishedOrdersAtLastWaypoint(t *testing.T) {
	home := pgHomeworld().Planets[0]
	free := Point{home.Pos.X + 30, home.Pos.Y}
	deep := Point{home.Pos.X + 15, home.Pos.Y + 15}
	waitLoad := Task{Kind: TaskTransport}
	waitLoad.Transport[Ironium] = Transport{Action: SetAmount, Amount: 1 << 20}
	unload := Task{Kind: TaskTransport}
	unload.Transport[Ironium] = Transport{Action: UnloadAll}
	for _, c := range []struct {
		name string
		ar   bool
		to   Point
		task Task
		kept bool        // the fleet keeps its task
		want []EventKind // its finished-orders message and the refusal before it
		more bool        // a further waypoint follows, 30 ly on
	}{
		{"remote mining, AR at its own planet (CONFIRMED RM-4 A)", true, home.Pos, Task{Kind: TaskRemoteMine}, true, nil, false},
		{"remote mining, AR at an unowned planet (CONFIRMED RM-4 U)", true, free, Task{Kind: TaskRemoteMine}, true, nil, false},
		{"remote mining, not AR, unowned planet (DOCUMENTED)", false, free, Task{Kind: TaskRemoteMine}, true, nil, false},
		{"remote mining, not AR, own planet (DOCUMENTED)", false, home.Pos, Task{Kind: TaskRemoteMine}, true, nil, false},
		{"remote mining, deep space (DOCUMENTED)", false, deep, Task{Kind: TaskRemoteMine}, true, nil, false},
		{"scrap (DOCUMENTED)", false, free, Task{Kind: TaskScrap}, true, nil, false},
		{"lay mines (DOCUMENTED)", false, deep, Task{Kind: TaskLayMines, Years: YearsIndefinitely}, true, nil, false},
		{"patrol (DOCUMENTED)", false, deep, Task{Kind: TaskPatrol, Range: 50}, true, nil, false},
		{"transport still waiting (DOCUMENTED)", false, home.Pos, waitLoad, true, []EventKind{EventLoadWaiting}, false},
		{"transport done: told when it ends (DOCUMENTED)", false, home.Pos, unload, false, []EventKind{EventFleetArrived}, false},
		{"transport done, a waypoint still ahead: not told (DOCUMENTED)", false, home.Pos, unload, false, nil, true},
		{"colonize refused in deep space, then told (CONFIRMED TK-201 H)", false, deep, Task{Kind: TaskColonize}, false, []EventKind{EventColonizeFailed, EventFleetArrived}, false},
		{"no task is told at arrival (CONFIRMED)", false, free, Task{}, true, []EventKind{EventFleetArrived}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, _ := remoteMineGame(t, 0, Point{c.to.X - 10, c.to.Y}, 1)
			if c.ar {
				g.Players[0].Race.PRT = PRTAlternateReality
			}
			g.Fleets[0].Task = Task{}
			g.Fleets[0].Cargo.Minerals[Ironium] = 5
			g.Fleets[0].Waypoints = []Waypoint{{Pos: c.to, Warp: 5, Task: c.task}}
			if c.more {
				g.Fleets[0].Waypoints = append(g.Fleets[0].Waypoints, Waypoint{Pos: Point{c.to.X + 30, c.to.Y}, Warp: 5})
			}
			r, err := GenerateTurn(withRules(g), nil, highRand{})
			if err != nil {
				t.Fatal(err)
			}
			fi := slices.IndexFunc(r.Game.Fleets, func(f Fleet) bool { return f.ID == 1 })
			f := r.Game.Fleets[fi]
			if want := c.task.Kind; !c.kept {
				want = TaskNone
				if f.Task.Kind != want {
					t.Errorf("task %v, want ended", f.Task.Kind)
				}
			} else if f.Pos != c.to || f.Task.Kind != want {
				t.Errorf("pos %v task %v, want arrived at %v with task %v", f.Pos, f.Task.Kind, c.to, want)
			}
			var got []EventKind
			for _, e := range r.Events {
				if e.Fleet == 1 && (e.Kind == EventFleetArrived || e.Kind == EventColonizeFailed || e.Kind == EventLoadWaiting) {
					got = append(got, e.Kind)
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("events %v, want %v", got, c.want)
			}
		})
	}
}
