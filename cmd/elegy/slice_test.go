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

// TestCLIPlayableSlice is the playable vertical slice (docs/PLAYING.md):
// a human against a computer player, played only through the command
// line. It creates a game, inspects it, writes orders with order (one
// rejected and not written), checks them, advances four years with a
// save and reload each year, and inspects the result. The same game
// played in one process through the library must reach the same state,
// and show must print the same text for both.
func TestCLIPlayableSlice(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "game.json")
	cli(t, "new", "-seed", "1", "-players", "1", "-size", "tiny", "-ai", "robotoid", "-o", save)
	lib, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	r, err := lib.Report(0)
	if err != nil {
		t.Fatal(err)
	}

	out := cli(t, "show", "-game", save, "all")
	for _, want := range []string{"year 2400, player 0", "Homeworld: ", "Colony Ship #", "Other planets (", "never seen", "ship slot 2: Colony Ship", "<- researching", "Messages for 2399 (0)"} {
		if !strings.Contains(out, want) {
			t.Errorf("show all lacks %q:\n%s", want, out)
		}
	}

	// The colony ship and the nearest planet, from the player's report.
	hw := r.Self.Homeworld
	colony := -1
	var colonyTask engine.Task
	v := newReportView(r, false)
	for _, f := range r.Fleets {
		if v.designName(f.Stacks[0].Design) == "Colony Ship" {
			colony, colonyTask = f.ID, f.Task
		}
	}
	home, _ := v.homeworld()
	target, best := -1, 0.0
	for _, p := range r.Universe {
		if d := dist(p.Pos, home); p.ID != hw && (target < 0 || d < best) {
			target, best = p.ID, d
		}
	}
	if colony < 0 || target < 0 {
		t.Fatalf("colony ship %d, target %d", colony, target)
	}
	slot := -1
	for _, d := range r.Designs {
		if !d.Slot.Starbase && d.Design.Name == "Colony Ship" {
			slot = d.Slot.Slot
		}
	}

	file := filepath.Join(dir, "orders.json")
	order := func(args ...string) string {
		return cli(t, append([]string{"order", "-game", save, "-player", "0", "-o", file}, args...)...)
	}
	itoa := strconv.Itoa
	order("cargo", "-fleet", itoa(colony), "-planet", itoa(hw), "-colonists", "25")
	order("move", "-fleet", itoa(colony), "-to", "planet:"+itoa(target), "-task", "colonize")
	order("research", "-budget", "20", "-field", "propulsion", "-next", "lowest")
	order("queue", "-planet", itoa(hw), "factories=10", "ship:Colony Ship=1")
	if got := order("queue", "-planet", itoa(hw), "-append", "mines=3"); !strings.Contains(got, "order 4 added (queue)") {
		t.Errorf("order output %q", got)
	}
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := run([]string{"order", "-game", save, "-o", file, "move", "-fleet", "999", "-to", "planet:" + itoa(target)}, &buf); err == nil || !strings.Contains(err.Error(), "order rejected, not added") {
		t.Fatalf("an order for fleet 999: err = %v", err)
	}
	if after, _ := os.ReadFile(file); !bytes.Equal(before, after) {
		t.Fatal("a rejected order changed the order file")
	}
	if got := cli(t, "check", "-game", save, file); got != "5 orders checked, 0 would be rejected\n" {
		t.Fatalf("check: %q", got)
	}

	queue := []engine.QueueItem{{Kind: engine.ItemFactory, Count: 10}, {Kind: engine.ItemShip, Count: 1, Slot: slot}}
	want := []engine.Order{
		engine.CargoOrder{Fleet: colony, Target: engine.TargetPlanet, ID: hw, Amounts: [engine.NumCargo + 1]int{engine.CargoColonists: 25}},
		engine.WaypointOrder{Fleet: colony, Task: colonyTask, Waypoints: []engine.Waypoint{{Target: engine.TargetPlanet, ID: target, Warp: 6, Task: engine.Task{Kind: engine.TaskColonize}}}},
		engine.ResearchOrder{Budget: 20, Field: engine.Propulsion, Next: engine.NextLowestField},
		engine.QueueOrder{Planet: hw, Queue: queue},
		engine.QueueOrder{Planet: hw, Queue: append(append([]engine.QueueItem(nil), queue...), engine.QueueItem{Kind: engine.ItemMine, Count: 3})},
	}
	f, err := game.DecodeOrders(bytes.NewReader(before))
	if err != nil {
		t.Fatal(err)
	}
	if f.GameID != lib.State.ID || f.Year != 2400 || f.Player != 0 || !reflect.DeepEqual(f.Orders, want) {
		t.Fatalf("order file %d %d %d:\n%#v\nwant\n%#v", f.GameID, f.Year, f.Player, f.Orders, want)
	}

	// Four years through the command line, each a save and reload; the
	// same four in one process through the library.
	if got := cli(t, "turn", "-game", save, file); strings.Contains(got, "rejected") {
		t.Fatalf("turn: %s", got)
	}
	for range 3 {
		cli(t, "turn", "-game", save)
	}
	ds, err := drivers(lib)
	if err != nil {
		t.Fatal(err)
	}
	first := append([]game.Driver(nil), ds...)
	first[0] = game.DriverFunc(func(game.Report) ([]engine.Order, error) { return want, nil })
	if _, err := lib.Advance(first); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := lib.Advance(ds); err != nil {
			t.Fatal(err)
		}
	}
	h, err := lib.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if got := cli(t, "hash", "-game", save); !strings.Contains(got, "year 2404 hash "+h) {
		t.Fatalf("command-line game %q, library game %s", got, h)
	}

	// The reloaded game shows what the uninterrupted one shows.
	lr, err := lib.Report(0)
	if err != nil {
		t.Fatal(err)
	}
	var libShow bytes.Buffer
	if err := show(&libShow, newReportView(lr, lib.State.SlowerTech), showSections); err != nil {
		t.Fatal(err)
	}
	got := cli(t, "show", "-game", save, "all")
	if got != libShow.String() {
		t.Fatalf("show after reloads:\n%s\nuninterrupted:\n%s", got, libShow.String())
	}
	// The colony ship colonized its target, and the research order held.
	owned := false
	for _, p := range lr.Planets {
		owned = owned || p.ID == target
	}
	if !owned || lr.Self.ResearchBudget != 20 {
		t.Fatalf("after four years: target %d owned %v, budget %d\n%s", target, owned, lr.Self.ResearchBudget, got)
	}
}
