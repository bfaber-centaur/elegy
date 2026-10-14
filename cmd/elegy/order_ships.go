package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// designOrder puts a design in one of the player's ship or starbase
// slots. The hull and fills come from -hull, or are copied from one of the
// player's designs with -from; each argument SLOT:COUNT:PART then sets
// what a hull slot holds (COUNT 0 empties it). elegy show designs prints
// every design's fills in this form. Without -slot the design takes the
// lowest slot neither the player's designs nor the file's earlier design
// orders use.
func designOrder(r game.Report, pending []engine.Order, args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order design", flag.ContinueOnError)
	slot := fs.Int("slot", -1, "design slot (default: the lowest the player does not use)")
	starbase := fs.Bool("starbase", false, "a starbase design")
	name := fs.String("name", "", "design name")
	hull := fs.String("hull", "", "hull name")
	from := fs.String("from", "", "copy the hull and fills of the player's design of this name or slot")
	picture := fs.Int("picture", 0, "picture, 0..3")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: elegy order design -name NAME (-hull HULL | -from DESIGN) [-slot N] [-starbase] SLOT:COUNT:PART...")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *name == "" {
		return nil, errors.New("order design: -name required")
	}
	o := engine.DesignOrder{Starbase: *starbase, Slot: *slot, Name: *name, Hull: *hull, Picture: *picture}
	fills := map[int]engine.SlotFill{}
	if *from != "" {
		d, ok := ownDesign(r, *starbase, *from)
		if !ok {
			return nil, fmt.Errorf("order design: no %s design %q (elegy show designs)", designKind(*starbase), *from)
		}
		if o.Hull == "" {
			o.Hull = d.Design.Hull.Name
		}
		for _, f := range designFills(d.Design) {
			fills[f.Slot] = f
		}
	}
	if o.Hull == "" {
		return nil, errors.New("order design: -hull or -from required")
	}
	for _, a := range fs.Args() {
		f, err := slotFill(a)
		if err != nil {
			return nil, err
		}
		if f.Count == 0 {
			delete(fills, f.Slot)
		} else {
			fills[f.Slot] = f
		}
	}
	for _, f := range fills {
		o.Fills = append(o.Fills, f)
	}
	sort.Slice(o.Fills, func(a, b int) bool { return o.Fills[a].Slot < o.Fills[b].Slot })
	if o.Slot < 0 {
		used := map[int]bool{}
		for _, d := range r.Designs {
			if d.Slot.Starbase == *starbase {
				used[d.Slot.Slot] = true
			}
		}
		for _, p := range pending {
			switch p := p.(type) {
			case engine.DesignOrder:
				if p.Starbase == *starbase {
					used[p.Slot] = true
				}
			case engine.DeleteDesignOrder:
				if p.Starbase == *starbase {
					used[p.Slot] = false
				}
			}
		}
		o.Slot = 0
		for used[o.Slot] {
			o.Slot++
		}
	}
	return o, nil
}

func designKind(starbase bool) string {
	if starbase {
		return "starbase"
	}
	return "ship"
}

// ownDesign is the player's ship or starbase design of a name or slot.
func ownDesign(r game.Report, starbase bool, nameOrSlot string) (game.OwnDesign, bool) {
	for _, d := range r.Designs {
		if d.Slot.Starbase == starbase && (strings.EqualFold(d.Design.Name, nameOrSlot) || strconv.Itoa(d.Slot.Slot) == nameOrSlot) {
			return d, true
		}
	}
	return game.OwnDesign{}, false
}

// slotFill parses SLOT:COUNT:PART.
func slotFill(s string) (engine.SlotFill, error) {
	bad := fmt.Errorf("order design: fill %q: SLOT:COUNT:PART", s)
	slotText, rest, _ := strings.Cut(s, ":")
	countText, part, _ := strings.Cut(rest, ":")
	if part == "" {
		return engine.SlotFill{}, bad
	}
	slot, err := strconv.Atoi(slotText)
	if err != nil {
		return engine.SlotFill{}, bad
	}
	count, err := strconv.Atoi(countText)
	if err != nil {
		return engine.SlotFill{}, bad
	}
	return engine.SlotFill{Slot: slot, Count: count, Part: part}, nil
}

// showStoredDesign prints the design a design order left in its slot at
// the end of the dry run: the engine drops or cuts malformed fills rather
// than refusing the design (engine.DesignOrder), so this is what the
// player will actually have, and a warning says when it differs from the
// order's fills.
func showStoredDesign(out io.Writer, next *game.Game, player int, o engine.DesignOrder) error {
	r, err := next.Report(player)
	if err != nil {
		return err
	}
	for _, d := range r.Designs {
		if d.Slot.Starbase == o.Starbase && d.Slot.Slot == o.Slot {
			fmt.Fprintln(out, "the design will be:")
			designLines(out, d)
			if got := designFills(d.Design); !slices.Equal(got, o.Fills) {
				fmt.Fprintln(out, "warning: the engine dropped or cut some of the order's fills (a part the slot does not take, above the player's tech, or over the slot's size)")
			}
			return nil
		}
	}
	return nil
}

func deleteDesignOrder(args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order delete-design", flag.ContinueOnError)
	slot := fs.Int("slot", -1, "design slot")
	starbase := fs.Bool("starbase", false, "a starbase design")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order delete-design: unexpected %q", fs.Arg(0))
	}
	return engine.DeleteDesignOrder{Starbase: *starbase, Slot: *slot}, nil
}

// planOrder sets battle plan -index: a new plan at the next number, or a
// change to an existing one. A flag left unset keeps the plan's setting
// as the file's last order for it, or the report, has it.
func planOrder(r game.Report, pending []engine.Order, args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order plan", flag.ContinueOnError)
	index := fs.Int("index", -1, "plan number (elegy show plans); the next number adds a plan")
	name := fs.String("name", "", "plan name")
	tactic := fs.String("tactic", "", "tactic: "+strings.Join(tacticNames, ", "))
	primary := fs.String("primary", "", "primary target: "+strings.Join(targetNames, ", "))
	secondary := fs.String("secondary", "", "secondary target")
	attack := fs.String("attack", "", "attack: "+strings.Join(attackNames, ", "))
	player := fs.Int("player", 0, "the player attacked with -attack player")
	dump := fs.Bool("dump", false, "dump cargo when the fleet is attacked")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order plan: unexpected %q", fs.Arg(0))
	}
	o := engine.BattlePlanOrder{Index: *index}
	if *index >= 0 && *index < len(r.Self.Plans) {
		o.Plan = r.Self.Plans[*index]
	}
	for _, p := range pending {
		if po, ok := p.(engine.BattlePlanOrder); ok && po.Index == *index {
			o.Plan = po.Plan
		}
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if set["name"] {
		o.Plan.Name = *name
	}
	for _, c := range []struct {
		flag, value string
		names       []string
		to          func(int)
	}{
		{"tactic", *tactic, tacticNames, func(i int) { o.Plan.Tactic = engine.Tactic(i) }},
		{"primary", *primary, targetNames, func(i int) { o.Plan.Primary = engine.TargetType(i) }},
		{"secondary", *secondary, targetNames, func(i int) { o.Plan.Secondary = engine.TargetType(i) }},
		{"attack", *attack, attackNames, func(i int) { o.Plan.Attack = engine.AttackWho(i) }},
	} {
		if !set[c.flag] {
			continue
		}
		i, err := lookup(c.flag, c.value, c.names)
		if err != nil {
			return nil, err
		}
		c.to(i)
	}
	if set["player"] {
		o.Plan.Player = *player
	}
	if set["dump"] {
		o.Plan.DumpCargo = *dump
	}
	return o, nil
}

func deletePlanOrder(args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order delete-plan", flag.ContinueOnError)
	index := fs.Int("index", -1, "plan number; plans after it move down one")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order delete-plan: unexpected %q", fs.Arg(0))
	}
	return engine.DeletePlanOrder{Index: *index}, nil
}

func fleetPlanOrder(args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order fleet-plan", flag.ContinueOnError)
	fleet := fs.Int("fleet", -1, "fleet id")
	plan := fs.Int("plan", 0, "plan number")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order fleet-plan: unexpected %q", fs.Arg(0))
	}
	return engine.FleetPlanOrder{Fleet: *fleet, Plan: *plan}, nil
}
