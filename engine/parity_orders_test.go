package engine

import (
	"encoding/json"
	"fmt"
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
}

// pvPlan converts a battle plan's fields as the state's battle_plans give
// them.
func pvPlan(name string, tactic, primary, secondary, attackWho int, dump bool) BattlePlan {
	p := BattlePlan{Name: name, Tactic: Tactic(tactic), Primary: TargetType(primary), Secondary: TargetType(secondary), DumpCargo: dump}
	if attackWho >= int(AttackPlayer) {
		p.Attack, p.Player = AttackPlayer, attackWho-int(AttackPlayer)
	} else {
		p.Attack = AttackWho(attackWho)
	}
	return p
}

// pvNextField converts a next-field choice: a field, "same" or "lowest".
func pvNextField(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return NextSameField, nil
	}
	var next string
	if json.Unmarshal(raw, &next) != nil {
		return 0, fmt.Errorf("next field %s", raw)
	}
	switch next {
	case "same":
		return NextSameField, nil
	case "lowest":
		return NextLowestField, nil
	}
	f, ok := pvFields[next]
	if !ok {
		return 0, fmt.Errorf("next field %q", next)
	}
	return f, nil
}

// orders converts the vector's orders for generated year y (1-based)
// against g, the game at the start of that year. why names an order the
// harness cannot convert.
func (l *pvLoaded) orders(blocks []pvOrderBlock, y int, g *Game) (files []PlayerOrders, why string) {
	for _, b := range blocks {
		if b.Year != y {
			continue
		}
		file := PlayerOrders{Player: b.Player, GameID: g.ID, Year: g.Year}
		split := -1 // a split's source fleet, waiting for the move that fills it
		for _, raw := range b.Orders {
			var o pvOrder
			if err := json.Unmarshal(raw, &o); err != nil {
				return nil, "order " + err.Error()
			}
			fleet := l.fleetID[pvFleetKey(b.Player, o.Fleet)]
			var order Order
			if split >= 0 && o.Kind != "move_ships" {
				return nil, "order split without a move_ships"
			}
			switch o.Kind {
			case "research":
				next, err := pvNextField(o.NextField)
				if err != nil {
					return nil, "order research " + err.Error()
				}
				order = ResearchOrder{Budget: o.Percent, Field: pvFields[o.Field], Next: next}
			case "production_queue":
				items, why := pvQueue(o.Items)
				if why != "" {
					return nil, "order " + why
				}
				order = QueueOrder{Planet: l.planet[o.Planet], Queue: items}
			case "battle_plan":
				order = BattlePlanOrder{Index: o.Slot, Plan: pvPlan(o.Name, o.Tactic, o.Primary, o.Secondary, o.AttackWho, o.DumpCargo)}
			case "battle_plan_delete":
				order = DeletePlanOrder{Index: o.Slot}
			case "fleet_battle_plan":
				order = FleetPlanOrder{Fleet: fleet, Plan: o.Plan}
			case "rename":
				order = RenameOrder{Fleet: fleet, Name: o.Name}
			case "merge":
				from := make([]int, len(o.Fleets))
				for i, id := range o.Fleets {
					from[i] = l.fleetID[pvFleetKey(b.Player, id)]
				}
				order = MergeOrder{Into: fleet, From: from}
			case "cargo":
				c := CargoOrder{Fleet: fleet}
				owner := b.Player
				if o.With.Owner != nil {
					owner = *o.With.Owner
				}
				switch o.With.Kind {
				case "planet":
					c.Target, c.ID = TargetPlanet, l.planet[o.With.ID]
				case "fleet":
					c.Target, c.ID = TargetFleet, l.fleetID[pvFleetKey(owner, o.With.ID)]
				default:
					return nil, "order cargo with " + o.With.Kind
				}
				for name, v := range o.Amounts {
					if name == "fuel" {
						c.Amounts[CargoFuel] = v
						continue
					}
					k, ok := pvCargo[name]
					if !ok {
						return nil, "order cargo " + name
					}
					c.Amounts[k] = v
				}
				order = c
			case "waypoint_change":
				i := g.fleetIndex(fleet)
				if i < 0 {
					return nil, "order waypoint_change: no fleet"
				}
				f := g.Fleets[i]
				task, why := l.task(o.Waypoint)
				if why != "" {
					return nil, "order waypoint_change " + why
				}
				w := WaypointOrder{Fleet: fleet, Task: f.Task, Waypoints: append([]Waypoint(nil), f.Waypoints...)}
				if o.Index == 0 {
					w.Task = task
				} else {
					wp, why := l.waypoint(b.Player, o.Waypoint)
					if why != "" {
						return nil, "order waypoint_change " + why
					}
					wp.Task = task
					switch k := o.Index - 1; {
					case k < len(w.Waypoints):
						w.Waypoints[k] = wp
					case k == len(w.Waypoints):
						w.Waypoints = append(w.Waypoints, wp)
					default:
						return nil, "order waypoint_change index"
					}
				}
				order = w
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
				var ships []Stack
				for _, sh := range o.Ships {
					d, ok := l.design[[2]int{b.Player, sh.Design}]
					if !ok {
						return nil, fmt.Sprintf("order move_ships design %d", sh.Design)
					}
					ships = append(ships, Stack{Design: d, Count: sh.Count})
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
					order, split = SplitOrder{Fleet: fleet, Ships: ships}, -1
					break
				}
				order = MoveShipsOrder{Fleet: fleet, With: l.fleetID[pvFleetKey(owner, o.With.ID)], Ships: ships}
			default:
				// design and design_delete need the players' design slots,
				// which the harness does not load.
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
