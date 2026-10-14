package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// readOrderFiles reads order files for the game's current year, by
// player. A file for another game or year, for a player the game does not
// have or a computer player, or a second file for one player is an error.
func readOrderFiles(g *game.Game, names []string) (map[int][]engine.Order, error) {
	byPlayer := map[int][]engine.Order{}
	for _, name := range names {
		fh, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		f, err := game.DecodeOrders(fh)
		fh.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if err := orderFileFits(g, f); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if _, computer := g.ComputerLevel(f.Player); computer {
			return nil, fmt.Errorf("%s: an order file for player %d, a computer player", name, f.Player)
		}
		if _, dup := byPlayer[f.Player]; dup {
			return nil, fmt.Errorf("%s: a second order file for player %d", name, f.Player)
		}
		byPlayer[f.Player] = f.Orders
	}
	return byPlayer, nil
}

// orderFileFits checks that an order file is for the game's current year.
func orderFileFits(g *game.Game, f game.OrderFile) error {
	switch {
	case f.GameID != g.State.ID:
		return fmt.Errorf("orders for game %d, this is game %d", f.GameID, g.State.ID)
	case f.Year != g.State.Year:
		return fmt.Errorf("orders for %d, the game is at %d", f.Year, g.State.Year)
	case f.Player < 0 || f.Player >= len(g.State.Players):
		return fmt.Errorf("no player %d", f.Player)
	}
	return nil
}

// dryRun generates the year on a copy of the game with only the given
// order files (every other player keeps its standing orders, computer
// players included) and returns the outcome of every order and the copy,
// at the next year. The game is not changed.
//
// ELEGY CHOICE: the check is against the game as it stands. An order
// that another player's orders for the same year would make fail (or
// succeed) is not caught.
func dryRun(g *game.Game, byPlayer map[int][]engine.Order) ([]engine.OrderResult, *game.Game, error) {
	b, err := g.Encode()
	if err != nil {
		return nil, nil, err
	}
	c, err := game.Load(bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	ds := make([]game.Driver, len(c.State.Players))
	for p := range ds {
		if orders, ok := byPlayer[p]; ok {
			ds[p] = game.DriverFunc(func(game.Report) ([]engine.Order, error) { return orders, nil })
		}
	}
	y, err := c.Advance(ds)
	if err != nil {
		return nil, nil, err
	}
	return y.Result.Orders, c, nil
}

func cmdCheck(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	byPlayer, err := readOrderFiles(g, fs.Args())
	if err != nil {
		return err
	}
	results, _, err := dryRun(g, byPlayer)
	if err != nil {
		return err
	}
	n, rejected := 0, 0
	for _, o := range results {
		n++
		if o.Err != nil {
			rejected++
			fmt.Fprintf(out, "player %d order %d would be rejected: %v\n", o.Player, o.Index, o.Err)
		}
	}
	fmt.Fprintf(out, "%d orders checked, %d would be rejected\n", n, rejected)
	if rejected > 0 {
		return fmt.Errorf("check: %d orders would be rejected", rejected)
	}
	return nil
}

// orderKindsUsage lists order's kinds.
const orderKindsUsage = "order kinds: research, move, cargo, queue, design, delete-design, plan, delete-plan, fleet-plan; elegy order <kind> -h for flags"

// cmdOrder adds one order to a player's order file for the current year,
// creating the file if needed. The order is checked with a dry run of the
// year (dryRun) with the whole file, and a rejected order is not added.
func cmdOrder(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("order", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file")
	player := fs.Int("player", 0, "player index")
	file := fs.String("o", "", "order file to add to (default orders<player>-<year>.json)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	r, err := g.Report(*player)
	if err != nil {
		return err
	}
	if r.Self.Computer {
		return fmt.Errorf("player %d is a computer player", *player)
	}
	name := *file
	if name == "" {
		name = fmt.Sprintf("orders%d-%d.json", *player, g.State.Year)
	}
	f := game.OrderFile{GameID: g.State.ID, Year: g.State.Year, Player: *player}
	if b, err := os.ReadFile(name); err == nil {
		if f, err = game.DecodeOrders(bytes.NewReader(b)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := orderFileFits(g, f); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if f.Player != *player {
			return fmt.Errorf("%s: orders for player %d, not %d", name, f.Player, *player)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: %w", name, err)
	}

	kind, kargs := fs.Arg(0), fs.Args()[min(1, fs.NArg()):]
	var o engine.Order
	switch kind {
	case "research":
		o, err = researchOrder(r, f.Orders, kargs)
	case "move":
		o, err = moveOrder(r, f.Orders, kargs)
	case "cargo":
		o, err = cargoOrder(kargs)
	case "queue":
		o, err = queueOrder(r, f.Orders, kargs)
	case "design":
		o, err = designOrder(r, f.Orders, kargs)
	case "delete-design":
		o, err = deleteDesignOrder(kargs)
	case "plan":
		o, err = planOrder(r, f.Orders, kargs)
	case "delete-plan":
		o, err = deletePlanOrder(kargs)
	case "fleet-plan":
		o, err = fleetPlanOrder(kargs)
	default:
		err = fmt.Errorf("unknown order kind %q; %s", kind, orderKindsUsage)
	}
	if err != nil {
		return err
	}

	f.Orders = append(f.Orders, o)
	results, next, err := dryRun(g, map[int][]engine.Order{*player: f.Orders})
	if err != nil {
		return err
	}
	last := len(f.Orders) - 1
	for _, res := range results {
		if res.Err == nil {
			continue
		}
		if res.Index == last {
			return fmt.Errorf("order rejected, not added: %w", res.Err)
		}
		fmt.Fprintf(out, "warning: order %d already in %s would be rejected: %v\n", res.Index, name, res.Err)
	}
	var buf bytes.Buffer
	if err := game.EncodeOrders(&buf, f); err != nil {
		return err
	}
	tmp := name + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "%s: order %d added (%s)\n", name, last, kind); err != nil {
		return err
	}
	if d, ok := o.(engine.DesignOrder); ok {
		return showStoredDesign(out, next, *player, d)
	}
	return nil
}

// researchOrder sets the research budget, field and next field; a flag
// left unset keeps the setting of the file's last research order, or the
// player's current setting.
func researchOrder(r game.Report, pending []engine.Order, args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order research", flag.ContinueOnError)
	budget := fs.Int("budget", -1, "percent of resources to research, 0..100 (default: unchanged)")
	field := fs.String("field", "", "field to research: "+strings.Join(fieldNames[:], ", ")+" (default: unchanged)")
	next := fs.String("next", "", "field after a level is gained: same, lowest, or a field (default: unchanged)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order research: unexpected %q", fs.Arg(0))
	}
	o := engine.ResearchOrder{Budget: r.Self.ResearchBudget, Field: r.Self.Research.Current, Next: r.Self.Research.Next}
	for _, p := range pending {
		if ro, ok := p.(engine.ResearchOrder); ok {
			o = ro
		}
	}
	if *budget >= 0 {
		o.Budget = *budget
	}
	if *field != "" {
		f, err := lookup("field", *field, fieldNames[:])
		if err != nil {
			return nil, err
		}
		o.Field = f
	}
	switch strings.ToLower(*next) {
	case "":
	case "same":
		o.Next = engine.NextSameField
	case "lowest":
		o.Next = engine.NextLowestField
	default:
		f, err := lookup("next field", *next, append([]string{"same", "lowest"}, fieldNames[:]...))
		if err != nil {
			return nil, err
		}
		o.Next = f - 2
	}
	return o, nil
}

// stringList is a repeatable string flag.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, " ") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

// moveOrder replaces a fleet's waypoints. The fleet keeps its task where
// it is (as the file's last waypoint order for it set it, or as the
// report shows it); -task is the task at the last waypoint.
func moveOrder(r game.Report, pending []engine.Order, args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order move", flag.ContinueOnError)
	fleet := fs.Int("fleet", -1, "fleet id (elegy show fleets)")
	var to stringList
	fs.Var(&to, "to", "a waypoint, in order: planet:ID, fleet:ID or X,Y; none clears the fleet's waypoints")
	warp := fs.Int("warp", 6, "warp for every waypoint, 0..11")
	task := fs.String("task", "none", "task at the last waypoint: none, colonize, remote-mine, scrap")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order move: unexpected %q", fs.Arg(0))
	}
	if len(to) == 0 {
		return nil, errors.New("order move: -to required")
	}
	o := engine.WaypointOrder{Fleet: *fleet}
	for _, f := range r.Fleets {
		if f.ID == *fleet {
			o.Task = f.Task
		}
	}
	for _, p := range pending {
		if wo, ok := p.(engine.WaypointOrder); ok && wo.Fleet == *fleet {
			o.Task = wo.Task
		}
	}
	tk, err := lookup("task", *task, []string{"none", "colonize", "remote-mine", "scrap"})
	if err != nil {
		return nil, err
	}
	last := []engine.TaskKind{engine.TaskNone, engine.TaskColonize, engine.TaskRemoteMine, engine.TaskScrap}[tk]
	if len(to) == 1 && to[0] == "none" {
		if last != engine.TaskNone {
			return nil, errors.New("order move: -task needs a waypoint")
		}
		return o, nil
	}
	for i, t := range to {
		wp, err := waypoint(t)
		if err != nil {
			return nil, err
		}
		wp.Warp = *warp
		if i == len(to)-1 {
			wp.Task.Kind = last
		}
		o.Waypoints = append(o.Waypoints, wp)
	}
	return o, nil
}

func waypoint(s string) (engine.Waypoint, error) {
	bad := fmt.Errorf("order move: -to %q: planet:ID, fleet:ID or X,Y", s)
	if kind, id, ok := strings.Cut(s, ":"); ok {
		n, err := strconv.Atoi(id)
		if err != nil {
			return engine.Waypoint{}, bad
		}
		switch kind {
		case "planet":
			return engine.Waypoint{Target: engine.TargetPlanet, ID: n}, nil
		case "fleet":
			return engine.Waypoint{Target: engine.TargetFleet, ID: n}, nil
		}
		return engine.Waypoint{}, bad
	}
	xs, ys, _ := strings.Cut(s, ",")
	x, errx := strconv.Atoi(strings.TrimSpace(xs))
	y, erry := strconv.Atoi(strings.TrimSpace(ys))
	if errx != nil || erry != nil {
		return engine.Waypoint{}, bad
	}
	return engine.Waypoint{Target: engine.TargetSpace, Pos: engine.Point{X: x, Y: y}}, nil
}

// cargoOrder moves cargo between a fleet and a planet or another fleet at
// its position. Positive amounts load into the fleet, negative amounts
// unload from it.
func cargoOrder(args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order cargo", flag.ContinueOnError)
	fleet := fs.Int("fleet", -1, "fleet id")
	planet := fs.Int("planet", -1, "planet id the fleet is at")
	with := fs.Int("with", -1, "or another fleet's id at the same position")
	var amounts [engine.NumCargo + 1]int
	fs.IntVar(&amounts[engine.Ironium], "ironium", 0, "kT of ironium (negative unloads)")
	fs.IntVar(&amounts[engine.Boranium], "boranium", 0, "kT of boranium")
	fs.IntVar(&amounts[engine.Germanium], "germanium", 0, "kT of germanium")
	fs.IntVar(&amounts[engine.CargoColonists], "colonists", 0, "colonists in kT, 100 colonists each")
	fs.IntVar(&amounts[engine.NumCargo], "fuel", 0, "mg of fuel (to or from a fleet only)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("order cargo: unexpected %q", fs.Arg(0))
	}
	o := engine.CargoOrder{Fleet: *fleet, Amounts: amounts}
	switch {
	case *planet >= 0 && *with < 0:
		o.Target, o.ID = engine.TargetPlanet, *planet
	case *with >= 0 && *planet < 0:
		o.Target, o.ID = engine.TargetFleet, *with
	default:
		return nil, errors.New("order cargo: one of -planet or -with")
	}
	return o, nil
}

// queueOrder replaces a planet's production queue with the items given,
// or with -append adds them after the planet's queue: the one the file's
// last queue order for the planet sets, or else the planet's current one. An item is
// NAME=COUNT (factories=10), or for a design ship:DESIGN=COUNT or
// starbase:DESIGN=COUNT, DESIGN being the design's name or slot.
func queueOrder(r game.Report, pending []engine.Order, args []string) (engine.Order, error) {
	fs := flag.NewFlagSet("order queue", flag.ContinueOnError)
	planet := fs.Int("planet", -1, "planet id")
	add := fs.Bool("append", false, "add after the planet's current queue instead of replacing it")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: elegy order queue -planet ID [-append] ITEM=COUNT...\nitems: %s; ship:DESIGN and starbase:DESIGN by name or slot\n", strings.Join(itemNames, ", "))
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	var base, items []engine.QueueItem
	if *add {
		for _, p := range r.Planets {
			if p.ID == *planet {
				base = p.Queue
			}
		}
		for _, p := range pending {
			if qo, ok := p.(engine.QueueOrder); ok && qo.Planet == *planet {
				base = qo.Queue
			}
		}
	}
	for _, a := range fs.Args() {
		it, err := queueItem(r, a)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	// A new slice, so the order shares nothing with the report or the
	// file; an empty queue is written as [], not null.
	q := slices.Concat(base, items)
	if q == nil {
		q = []engine.QueueItem{}
	}
	return engine.QueueOrder{Planet: *planet, Queue: q}, nil
}

func queueItem(r game.Report, s string) (engine.QueueItem, error) {
	i := strings.LastIndex(s, "=")
	if i < 0 {
		return engine.QueueItem{}, fmt.Errorf("order queue: item %q: NAME=COUNT", s)
	}
	count, err := strconv.Atoi(s[i+1:])
	if err != nil {
		return engine.QueueItem{}, fmt.Errorf("order queue: item %q: count %q", s, s[i+1:])
	}
	name, design, hasDesign := strings.Cut(s[:i], ":")
	k, err := lookup("item", name, itemNames)
	if err != nil {
		return engine.QueueItem{}, err
	}
	it := engine.QueueItem{Kind: engine.ItemKind(k), Count: count}
	isDesign := it.Kind == engine.ItemShip || it.Kind == engine.ItemStarbase
	if isDesign != hasDesign {
		return engine.QueueItem{}, fmt.Errorf("order queue: item %q: only ship and starbase name a design (ship:DESIGN=COUNT)", s)
	}
	if !isDesign {
		return it, nil
	}
	starbase := it.Kind == engine.ItemStarbase
	for _, d := range r.Designs {
		if d.Slot.Starbase == starbase && (strings.EqualFold(d.Design.Name, design) || strconv.Itoa(d.Slot.Slot) == design) {
			it.Slot = d.Slot.Slot
			return it, nil
		}
	}
	return engine.QueueItem{}, fmt.Errorf("order queue: item %q: no %s design %q (elegy show designs)", s, name, design)
}
