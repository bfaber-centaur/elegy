package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// orderGame is a tiny two-human game with a computer player 2, its save
// path and player 0's report.
func orderGame(t *testing.T) (string, game.Report) {
	t.Helper()
	save := filepath.Join(t.TempDir(), "game.json")
	cli(t, "new", "-seed", "5", "-players", "2", "-size", "tiny", "-ai", "robotoid", "-o", save)
	g, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	r, err := g.Report(0)
	if err != nil {
		t.Fatal(err)
	}
	return save, r
}

func readOrders(t *testing.T, path string) game.OrderFile {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := game.DecodeOrders(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestOrderKinds checks what each order kind writes, flag by flag.
func TestOrderKinds(t *testing.T) {
	save, r := orderGame(t)
	hw := strconv.Itoa(r.Self.Homeworld)
	scout := r.Fleets[1]
	file := filepath.Join(filepath.Dir(save), "o.json")
	add := func(args ...string) engine.Order {
		t.Helper()
		cli(t, append([]string{"order", "-game", save, "-o", file}, args...)...)
		f := readOrders(t, file)
		return f.Orders[len(f.Orders)-1]
	}
	self := r.Self
	for _, x := range []struct {
		args []string
		want engine.Order
	}{
		// Unset research flags keep the current setting, then the file's.
		{[]string{"research"}, engine.ResearchOrder{Budget: self.ResearchBudget, Field: self.Research.Current, Next: self.Research.Next}},
		{[]string{"research", "-budget", "0"}, engine.ResearchOrder{Budget: 0, Field: self.Research.Current, Next: self.Research.Next}},
		{[]string{"research", "-field", "Weapons", "-next", "same"}, engine.ResearchOrder{Budget: 0, Field: engine.Weapons, Next: engine.NextSameField}},
		{[]string{"research", "-next", "lowest"}, engine.ResearchOrder{Budget: 0, Field: engine.Weapons, Next: engine.NextLowestField}},
		{[]string{"research", "-next", "energy"}, engine.ResearchOrder{Budget: 0, Field: engine.Weapons, Next: engine.Energy}},
		{[]string{"research", "-next", "biotechnology", "-budget", "100"}, engine.ResearchOrder{Budget: 100, Field: engine.Weapons, Next: engine.Biotech}},
		{[]string{"move", "-fleet", strconv.Itoa(scout.ID), "-to", "1100 , 1200", "-to", "fleet:0", "-warp", "9", "-task", "remote-mine"}, engine.WaypointOrder{Fleet: scout.ID, Task: scout.Task, Waypoints: []engine.Waypoint{
			{Target: engine.TargetSpace, Pos: engine.Point{X: 1100, Y: 1200}, Warp: 9}, {Target: engine.TargetFleet, ID: 0, Warp: 9, Task: engine.Task{Kind: engine.TaskRemoteMine}}}}},
		{[]string{"move", "-fleet", strconv.Itoa(scout.ID), "-to", "planet:" + hw, "-task", "scrap"}, engine.WaypointOrder{Fleet: scout.ID, Task: scout.Task, Waypoints: []engine.Waypoint{
			{Target: engine.TargetPlanet, ID: r.Self.Homeworld, Warp: 6, Task: engine.Task{Kind: engine.TaskScrap}}}}},
		{[]string{"move", "-fleet", strconv.Itoa(scout.ID), "-to", "none"}, engine.WaypointOrder{Fleet: scout.ID, Task: scout.Task}},
		{[]string{"cargo", "-fleet", "3", "-planet", hw, "-ironium", "5", "-boranium", "6", "-germanium", "7", "-colonists", "8", "-fuel", "-9"},
			engine.CargoOrder{Fleet: 3, Target: engine.TargetPlanet, ID: r.Self.Homeworld, Amounts: [engine.NumCargo + 1]int{5, 6, 7, 8, -9}}},
		{[]string{"cargo", "-fleet", "3", "-with", "1", "-fuel", "10"}, engine.CargoOrder{Fleet: 3, Target: engine.TargetFleet, ID: 1, Amounts: [engine.NumCargo + 1]int{engine.NumCargo: 10}}},
		{[]string{"queue", "-planet", hw}, engine.QueueOrder{Planet: r.Self.Homeworld, Queue: []engine.QueueItem{}}},
		{[]string{"queue", "-planet", hw, "-append", "mines=2"}, engine.QueueOrder{Planet: r.Self.Homeworld, Queue: []engine.QueueItem{{Kind: engine.ItemMine, Count: 2}}}},
		{[]string{"queue", "-planet", hw, "-append", "ship:4=1", "starbase:Starbase=1", "auto-mines=3"}, engine.QueueOrder{Planet: r.Self.Homeworld, Queue: []engine.QueueItem{
			{Kind: engine.ItemMine, Count: 2}, {Kind: engine.ItemShip, Count: 1, Slot: 4}, {Kind: engine.ItemStarbase, Count: 1, Slot: 0}, {Kind: engine.ItemAutoMines, Count: 3}}}},
		{[]string{"queue", "-planet", hw, "ship:colony ship=2"}, engine.QueueOrder{Planet: r.Self.Homeworld, Queue: []engine.QueueItem{{Kind: engine.ItemShip, Count: 2, Slot: 2}}}},
	} {
		if got := add(x.args...); !reflect.DeepEqual(got, x.want) {
			t.Errorf("order %v:\n%#v\nwant\n%#v", x.args, got, x.want)
		}
	}
	if n := len(readOrders(t, file).Orders); n != 15 {
		t.Errorf("%d orders in the file, want 15", n)
	}
}

// TestOrderPendingState checks that a move keeps the task the file's
// earlier waypoint order set for the fleet, and that -append builds on
// the current queue when the file has no queue order for the planet.
func TestOrderPendingState(t *testing.T) {
	scout := engine.Fleet{ID: 4, Task: engine.Task{Kind: engine.TaskColonize}}
	r := game.Report{Fleets: []engine.Fleet{{ID: 3}, scout},
		Planets: []engine.Planet{{ID: 7, Queue: []engine.QueueItem{{Kind: engine.ItemFactory, Count: 1, Percent: 40}}}, {ID: 8}}}
	o, err := moveOrder(r, nil, []string{"-fleet", "4", "-to", "planet:1"})
	if err != nil || o.(engine.WaypointOrder).Task != scout.Task {
		t.Fatalf("move keeps the fleet's task: %+v %v", o, err)
	}
	pending := []engine.Order{
		engine.WaypointOrder{Fleet: 4, Task: engine.Task{Kind: engine.TaskScrap}},
		engine.WaypointOrder{Fleet: 3, Task: engine.Task{Kind: engine.TaskRoute}},
		engine.QueueOrder{Planet: 8, Queue: []engine.QueueItem{{Kind: engine.ItemMine, Count: 9}}},
	}
	o, err = moveOrder(r, pending, []string{"-fleet", "4", "-to", "planet:1"})
	if err != nil || o.(engine.WaypointOrder).Task.Kind != engine.TaskScrap {
		t.Fatalf("move keeps the file's task: %+v %v", o, err)
	}
	o, err = queueOrder(r, pending, []string{"-planet", "7", "-append", "mines=1"})
	want := []engine.QueueItem{{Kind: engine.ItemFactory, Count: 1, Percent: 40}, {Kind: engine.ItemMine, Count: 1}}
	if err != nil || !reflect.DeepEqual(o.(engine.QueueOrder).Queue, want) {
		t.Fatalf("append to the current queue: %+v %v", o, err)
	}
	o, err = queueOrder(r, pending, []string{"-planet", "8", "-append", "mines=1"})
	want = []engine.QueueItem{{Kind: engine.ItemMine, Count: 9}, {Kind: engine.ItemMine, Count: 1}}
	if err != nil || !reflect.DeepEqual(o.(engine.QueueOrder).Queue, want) || len(pending[2].(engine.QueueOrder).Queue) != 1 {
		t.Fatalf("append to the file's queue: %+v %v; pending %+v", o, err, pending[2])
	}
}

func TestOrderErrors(t *testing.T) {
	save, r := orderGame(t)
	dir := filepath.Dir(save)
	hw := strconv.Itoa(r.Self.Homeworld)
	file := filepath.Join(dir, "o.json")
	for _, x := range []struct {
		args []string
		want string
	}{
		{nil, "order kinds"},
		{[]string{"build"}, "unknown order kind"},
		{[]string{"research", "-field", "alchemy"}, `field "alchemy"`},
		{[]string{"research", "-next", "never"}, `next field "never"`},
		{[]string{"research", "extra"}, `unexpected "extra"`},
		{[]string{"research", "-budget", "101"}, "order rejected, not added"},
		{[]string{"move", "-fleet", "0"}, "-to required"},
		{[]string{"move", "-fleet", "0", "-to", "moon:3"}, `-to "moon:3"`},
		{[]string{"move", "-fleet", "0", "-to", "planet:x"}, `-to "planet:x"`},
		{[]string{"move", "-fleet", "0", "-to", "1100"}, `-to "1100"`},
		{[]string{"move", "-fleet", "0", "-to", "x,1"}, `-to "x,1"`},
		{[]string{"move", "-fleet", "0", "-to", "1,y"}, `-to "1,y"`},
		{[]string{"move", "-fleet", "0", "-to", "planet:0", "-task", "patrol"}, `task "patrol"`},
		{[]string{"move", "-fleet", "0", "-to", "none", "-task", "colonize"}, "-task needs a waypoint"},
		{[]string{"move", "-fleet", "0", "-to", "none", "-to", "planet:0"}, `-to "none"`},
		{[]string{"move", "-fleet", "0", "-to", "planet:0", "x"}, `unexpected "x"`},
		{[]string{"move", "-fleet", "0", "-to", "planet:0", "-warp", "12"}, "order rejected, not added"},
		{[]string{"cargo", "-fleet", "0"}, "one of -planet or -with"},
		{[]string{"cargo", "-fleet", "0", "-planet", hw, "-with", "1"}, "one of -planet or -with"},
		{[]string{"cargo", "-fleet", "0", "-planet", hw, "x"}, `unexpected "x"`},
		{[]string{"queue", "-planet", hw, "factories"}, "NAME=COUNT"},
		{[]string{"queue", "-planet", hw, "factories=x"}, `count "x"`},
		{[]string{"queue", "-planet", hw, "widgets=1"}, `item "widgets"`},
		{[]string{"queue", "-planet", hw, "factories:x=1"}, "only ship and starbase"},
		{[]string{"queue", "-planet", hw, "ship=1"}, "only ship and starbase"},
		{[]string{"queue", "-planet", hw, "ship:Nope=1"}, `no ship design "Nope"`},
		{[]string{"queue", "-planet", hw, "starbase:Scout=1"}, `no starbase design "Scout"`},
		{[]string{"queue", "-planet", hw, "factories=0"}, "order rejected, not added"},
	} {
		var out bytes.Buffer
		err := run(append([]string{"order", "-game", save, "-o", file}, x.args...), &out)
		if err == nil || !strings.Contains(err.Error(), x.want) {
			t.Errorf("order %v: err %v, want %q", x.args, err, x.want)
		}
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("failed orders wrote %s: %v", file, err)
	}

	// A computer player has no order file.
	if err := run([]string{"order", "-game", save, "-player", "2", "-o", file, "research"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "computer player") {
		t.Errorf("order for a computer player: %v", err)
	}
	if err := run([]string{"order", "-game", save, "-player", "3", "-o", file, "research"}, &bytes.Buffer{}); err == nil {
		t.Error("order for player 3 of 3")
	}
	// Another player's file, a file for another year, a broken file.
	cli(t, "order", "-game", save, "-player", "1", "-o", file, "research")
	if err := run([]string{"order", "-game", save, "-o", file, "research"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "orders for player 1, not 0") {
		t.Errorf("player 0 order into player 1's file: %v", err)
	}
	cli(t, "turn", "-game", save, file)
	if err := run([]string{"order", "-game", save, "-player", "1", "-o", file, "research"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "orders for 2400, the game is at 2401") {
		t.Errorf("order into last year's file: %v", err)
	}
	if err := os.WriteFile(file, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"order", "-game", save, "-o", file, "research"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), file) {
		t.Errorf("order into a broken file: %v", err)
	}
	if err := run([]string{"order", "-game", save, "-o", dir, "research"}, &bytes.Buffer{}); err == nil || !strings.HasPrefix(err.Error(), dir+": read ") {
		t.Errorf("order into a directory: %v", err)
	}
	if err := run([]string{"order", "-game", filepath.Join(dir, "missing.json"), "research"}, &bytes.Buffer{}); err == nil {
		t.Error("order for a missing game")
	}
}

// TestOrderDefaultFile checks the default file name, orders<player>-<year>.json.
func TestOrderDefaultFile(t *testing.T) {
	save, _ := orderGame(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Dir(save)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	if out := cli(t, "order", "-game", save, "-player", "1", "research"); out != "orders1-2400.json: order 0 added (research)\n" {
		t.Fatalf("output %q", out)
	}
	if f := readOrders(t, "orders1-2400.json"); f.Player != 1 || f.Year != 2400 || len(f.Orders) != 1 {
		t.Fatalf("file %+v", f)
	}
}

// TestCheck checks a hand-written file with a bad order: check names it
// and fails, and order warns about it but adds a good order after it.
func TestCheck(t *testing.T) {
	save, _ := orderGame(t)
	dir := filepath.Dir(save)
	g, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, player int, orders ...engine.Order) string {
		var buf bytes.Buffer
		if err := game.EncodeOrders(&buf, game.OrderFile{GameID: g.State.ID, Year: g.State.Year, Player: player, Orders: orders}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for _, x := range []struct {
		player int
		id     uint64
		want   string
	}{{3, g.State.ID, "no player 3"}, {-1, g.State.ID, "no player -1"}, {0, g.State.ID + 1, "orders for game"}} {
		var buf bytes.Buffer
		if err := game.EncodeOrders(&buf, game.OrderFile{GameID: x.id, Year: g.State.Year, Player: x.player}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "other.json")
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"check", "-game", save, path}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), x.want) {
			t.Errorf("file for player %d of game %d: %v", x.player, x.id, err)
		}
	}
	bad := write("bad.json", 0, engine.RenameOrder{Fleet: -1}, engine.ResearchOrder{Budget: 10})
	good := write("good.json", 1, engine.ResearchOrder{Budget: 10})
	var out bytes.Buffer
	err = run([]string{"check", "-game", save, bad, good}, &out)
	if err == nil || !strings.Contains(err.Error(), "1 orders would be rejected") ||
		!strings.Contains(out.String(), "player 0 order 0 would be rejected: ") || !strings.HasSuffix(out.String(), "3 orders checked, 1 would be rejected\n") {
		t.Fatalf("check: %v\n%s", err, out.String())
	}
	if got := cli(t, "check", "-game", save, good); got != "1 orders checked, 0 would be rejected\n" {
		t.Fatalf("check good: %q", got)
	}
	if got := cli(t, "check", "-game", save); got != "0 orders checked, 0 would be rejected\n" {
		t.Fatalf("check nothing: %q", got)
	}
	if err := run([]string{"check", "-game", save, good, good}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "a second order file for player 1") {
		t.Fatalf("two files for one player: %v", err)
	}
	if err := run([]string{"check", "-game", save, filepath.Join(dir, "missing.json")}, &bytes.Buffer{}); err == nil {
		t.Fatal("check of a missing file")
	}
	if err := run([]string{"check", "-game", filepath.Join(dir, "missing.json")}, &bytes.Buffer{}); err == nil {
		t.Fatal("check of a missing game")
	}

	out.Reset()
	if err := run([]string{"order", "-game", save, "-o", bad, "research", "-budget", "20"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "warning: order 0 already in "+bad+" would be rejected: ") || !strings.HasSuffix(out.String(), "order 2 added (research)\n") {
		t.Fatalf("order after a bad one:\n%s", out.String())
	}
	// check does not change the game.
	h, _ := g.Hash()
	if got := cli(t, "hash", "-game", save); !strings.Contains(got, h) {
		t.Fatal("check changed the game")
	}
}

// TestOrderParse checks parsing that the dry run cannot tell apart: id 0
// and a cargo target given twice.
func TestOrderParse(t *testing.T) {
	for _, x := range []struct {
		args []string
		want engine.CargoOrder
	}{
		{[]string{"-fleet", "1", "-planet", "0"}, engine.CargoOrder{Fleet: 1, Target: engine.TargetPlanet, ID: 0}},
		{[]string{"-fleet", "1", "-with", "0"}, engine.CargoOrder{Fleet: 1, Target: engine.TargetFleet, ID: 0}},
	} {
		got, err := cargoOrder(x.args)
		if err != nil || got != x.want {
			t.Errorf("cargo %v: %+v %v", x.args, got, err)
		}
	}
}

// TestDryRunLeavesGame checks that a dry run does not change the game it
// is given.
func TestDryRunLeavesGame(t *testing.T) {
	save, r := orderGame(t)
	g, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := g.Hash()
	res, err := dryRun(g, map[int][]engine.Order{0: {engine.ResearchOrder{Budget: 50}, engine.RenameOrder{Fleet: r.Fleets[0].ID, Name: "x"}}})
	if err != nil || len(res) != 2 || res[0].Err != nil || res[1].Err != nil {
		t.Fatalf("dry run: %+v %v", res, err)
	}
	if after, _ := g.Hash(); after != h || g.State.Year != 2400 {
		t.Fatalf("dry run changed the game: year %d", g.State.Year)
	}
}
