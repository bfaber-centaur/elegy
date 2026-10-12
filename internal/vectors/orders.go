package vectors

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// Submitted orders in the parity vectors (FORMAT.md "orders"), converted
// to Elegy orders. A kind the harness cannot convert, or Elegy does not
// model, skips the vector.

type pvOrderBlock struct {
	Year   int               `json:"year"`
	Player int               `json:"player"`
	Orders []json.RawMessage `json:"orders"`
}

type pvRef struct {
	Kind  string `json:"kind"`
	ID    int    `json:"id"`
	Owner *int   `json:"owner"`
}

type pvOrder struct {
	Kind      string            `json:"kind"`
	Fleet     int               `json:"fleet"`
	Fleets    []int             `json:"fleets"`
	Planet    int               `json:"planet"`
	Plan      int               `json:"plan"`
	Slot      int               `json:"slot"`
	Name      string            `json:"name"`
	Tactic    int               `json:"tactic"`
	Primary   int               `json:"primary"`
	Secondary int               `json:"secondary"`
	AttackWho int               `json:"attack_who"`
	DumpCargo bool              `json:"dump_cargo"`
	Percent   int               `json:"percent"`
	Field     string            `json:"field"`
	NextField json.RawMessage   `json:"next_field"`
	Items     []json.RawMessage `json:"items"`
	With      pvRef             `json:"with"`
	Amounts   map[string]int    `json:"amounts"`
	Index     int               `json:"index"`
	Waypoint  pvWaypoint        `json:"waypoint"`
	Ships     []struct {
		Design int `json:"design"`
		Count  int `json:"count"`
	} `json:"ships"`
	Starbase           bool      `json:"starbase"`
	Hull               string    `json:"hull"`
	Slots              []*pvPart `json:"slots"`
	LeftoverToResearch bool      `json:"leftover_to_research"`
	RouteTo            int       `json:"route_to"`
	PacketDestination  int       `json:"packet_destination"`
	PacketWarp         int       `json:"packet_warp"`
}

// pvPlan converts a battle plan's fields as the state's battle_plans give
// them.
func pvPlan(name string, tactic, primary, secondary, attackWho int, dump bool) engine.BattlePlan {
	p := engine.BattlePlan{Name: name, Tactic: engine.Tactic(tactic), Primary: engine.TargetType(primary), Secondary: engine.TargetType(secondary), DumpCargo: dump}
	if attackWho >= int(engine.AttackPlayer) {
		p.Attack, p.Player = engine.AttackPlayer, attackWho-int(engine.AttackPlayer)
	} else {
		p.Attack = engine.AttackWho(attackWho)
	}
	return p
}

// pvNextField converts a next-field choice: a field, "same" or "lowest".
func pvNextField(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return engine.NextSameField, nil
	}
	var next string
	if json.Unmarshal(raw, &next) != nil {
		return 0, fmt.Errorf("next field %s", raw)
	}
	switch next {
	case "same":
		return engine.NextSameField, nil
	case "lowest":
		return engine.NextLowestField, nil
	}
	f, ok := pvFields()[next]
	if !ok {
		return 0, fmt.Errorf("next field %q", next)
	}
	return f, nil
}

// orders converts the vector's orders for generated year y (1-based)
// against g, the game at the start of that year. why names an order the
// harness cannot convert.
func (l *pvLoaded) orders(blocks []pvOrderBlock, y int, g *engine.Game) (files []engine.PlayerOrders, why string) {
	for _, b := range blocks {
		if b.Year != y {
			continue
		}
		file := engine.PlayerOrders{Player: b.Player, GameID: g.ID, Year: g.Year}
		split := -1 // a split's source fleet, waiting for the move that fills it
		// Each fleet's waypoints as this file's earlier waypoint orders left
		// them: the vector gives one waypoint per order, and each order
		// applies on top of the ones before it.
		route := map[int]engine.WaypointOrder{}
		for _, raw := range b.Orders {
			var o pvOrder
			if err := json.Unmarshal(raw, &o); err != nil {
				return nil, "order " + err.Error()
			}
			fleet := l.fleetIn(g, b.Player, o.Fleet)
			var order engine.Order
			if split >= 0 && o.Kind != "move_ships" {
				return nil, "order split without a move_ships"
			}
			switch o.Kind {
			case "research":
				next, err := pvNextField(o.NextField)
				if err != nil {
					return nil, "order research " + err.Error()
				}
				order = engine.ResearchOrder{Budget: o.Percent, Field: pvFields()[o.Field], Next: next}
			case "production_queue":
				items, why := pvQueue(o.Items)
				if why != "" {
					return nil, "order " + why
				}
				order = engine.QueueOrder{Planet: l.planet[o.Planet], Queue: items}
			case "default_queue":
				// The order file's default-queue record, copied over the
				// player's setting whole (stars-elegy ORDERS.md "The
				// player's default queue", f2f9794).
				items, why := pvDefaultQueue(o.Items)
				if why != "" {
					return nil, "order " + why
				}
				order = engine.DefaultQueueOrder{Queue: items, LeftoverOnly: o.LeftoverToResearch}
			case "battle_plan":
				order = engine.BattlePlanOrder{Index: o.Slot, Plan: pvPlan(o.Name, o.Tactic, o.Primary, o.Secondary, o.AttackWho, o.DumpCargo)}
			case "battle_plan_delete":
				order = engine.DeletePlanOrder{Index: o.Slot}
			case "fleet_battle_plan":
				order = engine.FleetPlanOrder{Fleet: fleet, Plan: o.Plan}
			case "rename":
				order = engine.RenameOrder{Fleet: fleet, Name: o.Name}
			case "merge":
				from := make([]int, len(o.Fleets))
				for i, id := range o.Fleets {
					from[i] = l.fleetIn(g, b.Player, id)
				}
				order = engine.MergeOrder{Into: fleet, From: from}
			case "cargo":
				c := engine.CargoOrder{Fleet: fleet}
				owner := b.Player
				if o.With.Owner != nil {
					owner = *o.With.Owner
				}
				switch o.With.Kind {
				case "planet":
					c.Target, c.ID = engine.TargetPlanet, l.planet[o.With.ID]
				case "fleet":
					c.Target, c.ID = engine.TargetFleet, l.fleetIn(g, owner, o.With.ID)
				case "salvage":
					// Salvage is named by its owner and number
					// (vectors/README.md "cargo").
					if o.With.Owner == nil {
						return nil, "order cargo with salvage without an owner"
					}
					c.Target, c.ID, c.Owner = engine.TargetSalvage, o.With.ID, owner
				default:
					return nil, "order cargo with " + o.With.Kind
				}
				for name, v := range o.Amounts {
					if name == "fuel" {
						c.Amounts[engine.CargoFuel] = v
						continue
					}
					k, ok := pvCargo()[name]
					if !ok {
						return nil, "order cargo " + name
					}
					c.Amounts[k] = v
				}
				order = c
			case "waypoint_change", "waypoint_add":
				w, ok := route[fleet]
				if !ok {
					i := fleetIndex(g, fleet)
					if i < 0 {
						// A fleet Elegy does not have: the order goes to the
						// engine, which refuses it (Comparison.Rejected).
						order = engine.WaypointOrder{Fleet: fleet}
						break
					}
					f := g.Fleets[i]
					w = engine.WaypointOrder{Fleet: fleet, Task: f.Task, Waypoints: append([]engine.Waypoint(nil), f.Waypoints...)}
				}
				task, why := l.task(o.Waypoint)
				if why != "" {
					return nil, "order " + o.Kind + " " + why
				}
				if o.Index == 0 {
					// Index 0 is the fleet's position: only its task changes.
					if o.Kind == "waypoint_add" {
						return nil, "order waypoint_add before the fleet's position"
					}
					w.Task = task
				} else {
					wp, why := l.waypoint(b.Player, o.Waypoint)
					if why != "" {
						return nil, "order " + o.Kind + " " + why
					}
					wp.Task = task
					wps := append([]engine.Waypoint(nil), w.Waypoints...)
					switch k := o.Index - 1; {
					case o.Kind == "waypoint_add" && k <= len(wps):
						// vectors/README.md: inserted before index.
						wps = slices.Insert(wps, k, wp)
					case k < len(wps):
						wps[k] = wp
					case k == len(wps):
						wps = append(wps, wp)
					default:
						return nil, "order " + o.Kind + " index"
					}
					w.Waypoints = wps
				}
				route[fleet] = w
				order = w
			case "waypoint_delete":
				w, ok := route[fleet]
				if !ok {
					i := fleetIndex(g, fleet)
					if i < 0 {
						order = engine.WaypointOrder{Fleet: fleet}
						break
					}
					f := g.Fleets[i]
					w = engine.WaypointOrder{Fleet: fleet, Task: f.Task, Waypoints: append([]engine.Waypoint(nil), f.Waypoints...)}
				}
				k := o.Index - 1
				if k < 0 || k >= len(w.Waypoints) {
					return nil, "order waypoint_delete index"
				}
				w.Waypoints = slices.Delete(append([]engine.Waypoint(nil), w.Waypoints...), k, k+1)
				route[fleet] = w
				order = w
			case "design":
				var fills []engine.SlotFill
				for i, sl := range o.Slots {
					if sl != nil {
						fills = append(fills, engine.SlotFill{Slot: i, Part: sl.Part, Count: sl.Count})
					}
				}
				// The vector gives no design name; it is named as the
				// loader names the state's designs (Elegy choice: names are
				// not compared).
				order = engine.DesignOrder{Starbase: o.Starbase, Slot: o.Slot, Name: fmt.Sprintf("p%d d%d", b.Player, o.Slot), Hull: o.Hull, Fills: fills}
			case "design_delete":
				order = engine.DeleteDesignOrder{Starbase: o.Starbase, Slot: o.Slot}
			case "planet_settings":
				// vectors/README.md: the planet's whole settings, -1 for no
				// route and no packet destination.
				ps := engine.PlanetSettingsOrder{Planet: l.planet[o.Planet], LeftoverOnly: o.LeftoverToResearch, PacketSpeed: o.PacketWarp}
				if o.RouteTo >= 0 {
					ps.HasRoute, ps.RouteTo = true, l.planet[o.RouteTo]
				}
				if o.PacketDestination >= 0 {
					ps.HasPacketDest, ps.PacketDest = true, l.planet[o.PacketDestination]
				}
				order = ps
			case "split":
				// FORMAT.md: a new empty fleet beside it, which the next
				// move_ships fills; Elegy's SplitOrder does both.
				split = o.Fleet
				continue
			case "move_ships":
				owner := b.Player
				if o.With.Owner != nil {
					owner = *o.With.Owner
				}
				if o.With.Kind != "fleet" || owner != b.Player {
					return nil, "order move_ships with another player's object"
				}
				var ships []engine.Stack
				for _, sh := range o.Ships {
					d, ok := l.designIn(g, b.Player, sh.Design)
					if !ok {
						return nil, fmt.Sprintf("order move_ships design %d", sh.Design)
					}
					ships = append(ships, engine.Stack{Design: d, Count: sh.Count})
				}
				if split >= 0 {
					if o.Fleet != split {
						return nil, "order split filled from another fleet"
					}
					for k := range ships {
						if ships[k].Count >= 0 {
							return nil, "order split into the source"
						}
						ships[k].Count = -ships[k].Count
					}
					order, split = engine.SplitOrder{Fleet: fleet, Ships: ships}, -1
					break
				}
				order = engine.MoveShipsOrder{Fleet: fleet, With: l.fleetIn(g, owner, o.With.ID), Ships: ships}
			default:
				return nil, "order " + o.Kind
			}
			file.Orders = append(file.Orders, order)
		}
		if split >= 0 {
			return nil, "order split without a move_ships"
		}
		files = append(files, file)
	}
	return files, ""
}

// fleetIndex is the index in g.Fleets of the fleet with ID id, or -1.
func fleetIndex(g *engine.Game, id int) int {
	for i := range g.Fleets {
		if g.Fleets[i].ID == id {
			return i
		}
	}
	return -1
}

// fleetIn is the Elegy id of the vector's fleet (owner, id) in g
// (fleetAt), or 0.
func (l *pvLoaded) fleetIn(g *engine.Game, owner, id int) int {
	if f := l.fleetAt(g, owner, id); f != nil {
		return f.ID
	}
	return 0
}

// fleetAt is the vector's fleet (owner, id) in g: the initial state's
// fleet while it exists, else a fleet made during the run (built or
// given), known by its fleet number: the vector stores it from 0, the
// client shows it from 1 (PRODUCTION-LAUNCH.md "The new fleet"). A
// number the initial fleet left is used again for a new one
// (PRODUCTION-LAUNCH.md "the owner's lowest unused fleet number").
func (l *pvLoaded) fleetAt(g *engine.Game, owner, id int) *engine.Fleet {
	fid, ok := l.fleetID[pvFleetKey(owner, id)]
	started := map[int]bool{}
	for _, v := range l.fleetID {
		started[v] = true
	}
	var made *engine.Fleet
	for i := range g.Fleets {
		f := &g.Fleets[i]
		if ok && f.ID == fid {
			return f
		}
		if made == nil && !started[f.ID] && f.Owner == owner && f.Number == id+1 {
			made = f
		}
	}
	return made
}

// designIn is the Game.Designs index in owner's ship design slot in g,
// which a design order may have changed since the start.
func (l *pvLoaded) designIn(g *engine.Game, owner, slot int) (int, bool) {
	if d, ok := g.PlayerDesign(owner, false, slot); ok {
		return d, true
	}
	d, ok := l.design[[2]int{owner, slot}]
	return d, ok
}
