package engine

import "testing"

// "Fill to" and "wait for" (TAKEOVER.md "Unload and load amounts",
// CONFIRMED FO-01 F, I, J): up to v% of the 210 kT hold, from what the
// planet has; fill clears after its load, wait only once met. ASSUMPTION
// T8: the target is per cargo type and rounded down (33% of 210 is 69),
// so 60 kT of ironium already aboard leaves 45 to fill to 50%, whatever
// else is aboard; T9: wait is met at the target.
func TestFillAndWaitFor(t *testing.T) {
	for _, tt := range []struct {
		name          string
		tr            Transport
		surface, held int
		boranium      int // other cargo aboard
		load          int
		left          Transport
	}{
		{"fill 50%", Transport{FillTo, 50}, 500, 0, 0, 105, Transport{}},
		{"fill 33%, rounded down", Transport{FillTo, 33}, 500, 0, 0, 69, Transport{}},
		{"fill 100%, short", Transport{FillTo, 100}, 100, 0, 0, 100, Transport{}},
		{"fill 50%, some aboard", Transport{FillTo, 50}, 500, 60, 30, 45, Transport{}},
		{"fill 50%, above target", Transport{FillTo, 50}, 500, 150, 0, 0, Transport{}},
		{"fill 100%, hold full of boranium", Transport{FillTo, 100}, 500, 0, 200, 10, Transport{}},
		{"wait 100%, short", Transport{WaitFor, 100}, 100, 0, 0, 100, Transport{WaitFor, 100}},
		{"wait 50%, met", Transport{WaitFor, 50}, 500, 0, 0, 105, Transport{}},
		{"wait 50%, met exactly aboard", Transport{WaitFor, 50}, 0, 105, 0, 0, Transport{}},
		{"wait 50%, one short", Transport{WaitFor, 50}, 0, 104, 0, 0, Transport{WaitFor, 50}},
	} {
		l := newTKLab(t, 3)
		medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
		pi := l.planet(0, 0, 100)
		l.g.Planets[pi].Surface[Ironium] = tt.surface
		fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
		f := &l.g.Fleets[fi]
		f.Cargo.Minerals = Minerals{tt.held, tt.boranium, 0}
		f.Task = Task{Kind: TaskTransport}
		f.Task.Transport[Ironium] = tt.tr
		l.g.load(f)
		if got := f.Cargo.Minerals[Ironium] - tt.held; got != tt.load || l.g.Planets[pi].Surface[Ironium] != tt.surface-tt.load {
			t.Errorf("%s: loaded %d (surface %d), want %d", tt.name, got, l.g.Planets[pi].Surface[Ironium], tt.load)
		}
		if f.Task.Transport[Ironium] != tt.left || (tt.left == Transport{}) != (f.Task.Kind == TaskNone) {
			t.Errorf("%s: task %+v, want action %+v", tt.name, f.Task, tt.left)
		}
	}
}

// A fleet whose transport task is still current after the load pass
// holds its place; one whose loads were satisfied moves (KERNEL.md "Other
// movement rules", CONFIRMED KB-4A T1, FO-01 I and J). The freighter sits
// at its own planet with 50 kT of ironium, heading 30 ly away at warp 6.
func TestTransportTaskHoldsFleet(t *testing.T) {
	for _, tt := range []struct {
		name string
		tr   Transport
		move bool
	}{
		{"load all", Transport{LoadAll, 0}, true},
		{"fill to 100%", Transport{FillTo, 100}, true},
		{"load exactly 30", Transport{LoadExactly, 30}, true},
		{"load exactly 300, short", Transport{LoadExactly, 300}, false},
		{"wait for 100%, short", Transport{WaitFor, 100}, false},
	} {
		l := newTKLab(t, 3)
		medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
		pi := l.planet(0, 0, 100)
		dst := l.planet(0, 30, 100)
		l.g.Planets[pi].Surface[Ironium] = 50
		fi := l.fleet(0, pi, 0, Stack{Design: medium, Count: 1})
		f := &l.g.Fleets[fi]
		f.Fuel = 500
		f.Task = Task{Kind: TaskTransport}
		f.Task.Transport[Ironium] = tt.tr
		f.Waypoints = []Waypoint{{Pos: l.g.Planets[dst].Pos, Warp: 6, Target: TargetPlanet, ID: l.g.Planets[dst].ID}}
		g := l.turn(&seqRand{}).Game
		if moved := g.Fleets[0].Pos != l.g.Planets[pi].Pos; moved != tt.move {
			t.Errorf("%s: at %v, moved %v, want %v", tt.name, g.Fleets[0].Pos, moved, tt.move)
		}
	}
}

// A fill or wait percentage is 0 to 100; other actions keep their
// amount rule and an action past WaitFor is out of range.
func TestValidPercentTransport(t *testing.T) {
	for _, tt := range []struct {
		tr Transport
		ok bool
	}{
		{Transport{FillTo, 100}, true},
		{Transport{WaitFor, 0}, true},
		{Transport{FillTo, 101}, false},
		{Transport{WaitFor, 101}, false},
		{Transport{WaitFor, -1}, false},
		{Transport{LoadExactly, 500}, true},
		{Transport{WaitFor + 1, 0}, false},
	} {
		task := Task{Kind: TaskTransport}
		task.Transport[Ironium] = tt.tr
		if err := validTask(task); (err == nil) != tt.ok {
			t.Errorf("%+v: %v, want ok %v", tt.tr, err, tt.ok)
		}
	}
}
