package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// TestOrderDesigns checks the design orders a player writes, and what
// order prints about the design the year will store.
func TestOrderDesigns(t *testing.T) {
	save, _ := orderGame(t)
	file := filepath.Join(filepath.Dir(save), "o.json")
	add := func(args ...string) (string, engine.Order) {
		t.Helper()
		out := cli(t, append([]string{"order", "-game", save, "-o", file}, args...)...)
		f := readOrders(t, file)
		return out, f.Orders[len(f.Orders)-1]
	}
	scout := []engine.SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 2, Part: "Fuel Tank", Count: 1}}
	for _, x := range []struct {
		args []string
		want engine.Order
		out  string
	}{
		// A copy of the Scout with its scanner taken out, in the lowest free slot.
		{[]string{"design", "-name", "Long Scout", "-from", "scout", "1:0:x"}, engine.DesignOrder{Slot: 6, Name: "Long Scout", Hull: "Scout", Fills: scout},
			"order 0 added (design)\nthe design will be:\nship slot 6: Long Scout (Scout hull), 20 kT, 1 Long Hump 6, fuel 300 mg\n  parts: 1 Long Hump 6, 1 Fuel Tank\n  slots: 0:1:Long Hump 6, 2:1:Fuel Tank\n"},
		// The next free slot counts the file's design; a part the slot
		// does not take is dropped, with a warning.
		{[]string{"design", "-name", "Odd", "-hull", "Scout", "-picture", "2", "1:1:Colonization Module", "0:1:Long Hump 6"}, engine.DesignOrder{Slot: 7, Name: "Odd", Hull: "Scout", Picture: 2,
			Fills: []engine.SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 1, Part: "Colonization Module", Count: 1}}},
			"  slots: 0:1:Long Hump 6\nwarning: the engine dropped or cut"},
		{[]string{"delete-design", "-slot", "6"}, engine.DeleteDesignOrder{Slot: 6}, "order 2 added (delete-design)\n"},
		// A slot the file deletes is free again.
		{[]string{"design", "-name", "Again", "-from", "1", "-hull", "Destroyer"}, engine.DesignOrder{Slot: 6, Name: "Again", Hull: "Destroyer",
			Fills: []engine.SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 1, Part: "Rhino Scanner", Count: 1}, {Slot: 2, Part: "Fuel Tank", Count: 1}}}, "ship slot 6: Again (Destroyer hull)"},
		{[]string{"design", "-name", "Fort", "-starbase", "-slot", "3", "-from", "Starbase", "1:0:x", "2:0:x", "3:0:x", "4:0:x", "5:0:x", "7:0:x", "9:0:x", "11:0:x"},
			engine.DesignOrder{Starbase: true, Slot: 3, Name: "Fort", Hull: "Space Station"}, "starbase slot 3: Fort (Space Station hull)"},
		{[]string{"design", "-name", "Base", "-starbase", "-hull", "Space Station"}, engine.DesignOrder{Starbase: true, Slot: 1, Name: "Base", Hull: "Space Station"}, "starbase slot 1: Base"},
		{[]string{"delete-design", "-starbase", "-slot", "3"}, engine.DeleteDesignOrder{Starbase: true, Slot: 3}, ""},
		// Starbase slots in the file do not count for a ship design.
		{[]string{"design", "-name", "Big", "-starbase", "-slot", "8", "-hull", "Space Station"}, engine.DesignOrder{Starbase: true, Slot: 8, Name: "Big", Hull: "Space Station"}, "starbase slot 8"},
		{[]string{"delete-design", "-starbase", "-slot", "7"}, engine.DeleteDesignOrder{Starbase: true, Slot: 7}, ""},
		{[]string{"design", "-name", "Late", "-from", "Scout"}, engine.DesignOrder{Slot: 8, Name: "Late", Hull: "Scout",
			Fills: []engine.SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 1, Part: "Rhino Scanner", Count: 1}, {Slot: 2, Part: "Fuel Tank", Count: 1}}}, "ship slot 8: Late"},
	} {
		out, got := add(x.args...)
		if !reflect.DeepEqual(got, x.want) {
			t.Errorf("order %v:\n%#v\nwant\n%#v", x.args, got, x.want)
		}
		if !strings.Contains(out, x.out) {
			t.Errorf("order %v printed:\n%s\nwant %q", x.args, out, x.out)
		}
		if _, ok := x.want.(engine.DesignOrder); ok && x.args[2] != "Odd" && x.args[2] != "Again" && strings.Contains(out, "warning") {
			t.Errorf("order %v warned:\n%s", x.args, out)
		}
	}

	for _, x := range []struct {
		args []string
		want string
	}{
		{[]string{"design", "-hull", "Scout"}, "-name required"},
		{[]string{"design", "-name", "X"}, "-hull or -from required"},
		{[]string{"design", "-name", "X", "-from", "Wagon"}, `no ship design "Wagon"`},
		{[]string{"design", "-name", "X", "-from", "Starbase"}, `no ship design "Starbase"`},
		{[]string{"design", "-name", "X", "-starbase", "-from", "Scout"}, `no starbase design "Scout"`},
		{[]string{"design", "-name", "X", "-hull", "Wagon"}, "order rejected, not added"},
		{[]string{"design", "-name", "X", "-hull", "Scout", "0:1"}, `fill "0:1"`},
		{[]string{"design", "-name", "X", "-hull", "Scout", "0:1:"}, `fill "0:1:"`},
		{[]string{"design", "-name", "X", "-hull", "Scout", "a:1:Fuel Tank"}, `fill "a:1:Fuel Tank"`},
		{[]string{"design", "-name", "X", "-hull", "Scout", "0:b:Fuel Tank"}, `fill "0:b:Fuel Tank"`},
		{[]string{"delete-design", "-slot", "99"}, "order rejected, not added"},
		{[]string{"delete-design", "-slot", "1", "x"}, `unexpected "x"`},
	} {
		err := run(append([]string{"order", "-game", save, "-o", file}, x.args...), &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), x.want) {
			t.Errorf("order %v: err %v, want %q", x.args, err, x.want)
		}
	}
}

// TestOrderPlans checks the battle-plan orders.
func TestOrderPlans(t *testing.T) {
	save, r := orderGame(t)
	file := filepath.Join(filepath.Dir(save), "o.json")
	add := func(args ...string) engine.Order {
		t.Helper()
		cli(t, append([]string{"order", "-game", save, "-o", file}, args...)...)
		f := readOrders(t, file)
		return f.Orders[len(f.Orders)-1]
	}
	def := r.Self.Plans[0]
	everyone := def
	everyone.Attack = engine.AttackEveryone
	raid := engine.BattlePlan{Name: "Raid", Tactic: engine.TacticMaximizeDamage, Primary: engine.TargetFreighters, Secondary: engine.TargetStarbase,
		Attack: engine.AttackPlayer, Player: 1, DumpCargo: true}
	calm := raid
	calm.Tactic = engine.TacticDisengage
	last := r.Self.Plans[len(r.Self.Plans)-1]
	last.Name = "Last"
	for _, x := range []struct {
		args []string
		want engine.Order
	}{
		// Unset flags keep the plan as the report has it.
		{[]string{"plan", "-index", "0", "-attack", "everyone"}, engine.BattlePlanOrder{Index: 0, Plan: everyone}},
		{[]string{"plan", "-index", "4", "-name", "Last"}, engine.BattlePlanOrder{Index: 4, Plan: last}},
		{[]string{"plan", "-index", "5", "-name", "Raid", "-tactic", "maximize-damage", "-primary", "freighters", "-secondary", "Starbase", "-attack", "player", "-player", "1", "-dump"},
			engine.BattlePlanOrder{Index: 5, Plan: raid}},
		// ... or as the file's last order for it has it.
		{[]string{"plan", "-index", "5", "-tactic", "disengage"}, engine.BattlePlanOrder{Index: 5, Plan: calm}},
		{[]string{"plan", "-index", "0"}, engine.BattlePlanOrder{Index: 0, Plan: everyone}},
		{[]string{"plan", "-index", "5", "-dump=false"}, engine.BattlePlanOrder{Index: 5, Plan: func() engine.BattlePlan { p := calm; p.DumpCargo = false; return p }()}},
		{[]string{"fleet-plan", "-fleet", "1", "-plan", "5"}, engine.FleetPlanOrder{Fleet: 1, Plan: 5}},
		{[]string{"delete-plan", "-index", "4"}, engine.DeletePlanOrder{Index: 4}},
	} {
		if got := add(x.args...); !reflect.DeepEqual(got, x.want) {
			t.Errorf("order %v:\n%#v\nwant\n%#v", x.args, got, x.want)
		}
	}
	for _, x := range []struct {
		args []string
		want string
	}{
		{[]string{"plan", "-index", "1", "-tactic", "flee"}, `tactic "flee"`},
		{[]string{"plan", "-index", "1", "-primary", "ships"}, `primary "ships"`},
		{[]string{"plan", "-index", "1", "-secondary", "ships"}, `secondary "ships"`},
		{[]string{"plan", "-index", "1", "-attack", "friends"}, `attack "friends"`},
		{[]string{"plan", "-index", "1", "x"}, `unexpected "x"`},
		{[]string{"plan", "-index", "9"}, "order rejected, not added"},
		{[]string{"plan", "-index", "1", "-attack", "player", "-player", "0"}, "order rejected, not added"},
		{[]string{"delete-plan", "-index", "0"}, "order rejected, not added"},
		{[]string{"delete-plan", "-index", "1", "x"}, `unexpected "x"`},
		{[]string{"fleet-plan", "-fleet", "1", "-plan", "99"}, "order rejected, not added"},
		{[]string{"fleet-plan", "-fleet", "1", "x"}, `unexpected "x"`},
	} {
		err := run(append([]string{"order", "-game", save, "-o", file}, x.args...), &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), x.want) {
			t.Errorf("order %v: err %v, want %q", x.args, err, x.want)
		}
	}
}

// TestCheckComputerPlayerFile checks that check refuses an order file for
// a computer player, as turn does.
func TestCheckComputerPlayerFile(t *testing.T) {
	save, _ := orderGame(t)
	g, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := game.EncodeOrders(&buf, game.OrderFile{GameID: g.State.ID, Year: g.State.Year, Player: 2}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(save), "ai.json")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"check", "turn"} {
		err := run([]string{cmd, "-game", save, path}, &bytes.Buffer{})
		if err == nil || err.Error() != path+": an order file for player 2, a computer player" {
			t.Errorf("%s: %v", cmd, err)
		}
	}
}

func TestDesignFills(t *testing.T) {
	d := engine.Design{Slots: []engine.Slot{{Part: engine.Part{Name: "B"}, Count: 2}, {Part: engine.Part{Name: "A"}, Count: 1}}, SlotPos: []int{3, 1}}
	want := []engine.SlotFill{{Slot: 1, Part: "A", Count: 1}, {Slot: 3, Part: "B", Count: 2}}
	if got := designFills(d); !reflect.DeepEqual(got, want) {
		t.Errorf("fills %+v", got)
	}
	d.SlotPos = d.SlotPos[:1]
	if got := designFills(d); got != nil {
		t.Errorf("fills without positions %+v", got)
	}
	if got, err := slotFill("2:3:Mole-skin Shield"); err != nil || got != (engine.SlotFill{Slot: 2, Count: 3, Part: "Mole-skin Shield"}) {
		t.Errorf("slotFill: %+v %v", got, err)
	}
	// Without -slot and with every slot free, slot 0.
	o, err := designOrder(game.Report{}, nil, []string{"-name", "X", "-hull", "Scout"})
	if err != nil || o.(engine.DesignOrder).Slot != 0 {
		t.Errorf("design in an empty report: %+v %v", o, err)
	}
}
