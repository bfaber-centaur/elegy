package engine_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// The parity harness's space objects (FORMAT.md "objects", and the
// minefield, wormhole, trader, packet, object and object_gone
// expectations). This file is in package engine_test because package
// objects imports the engine.

func init() { engine.SetParitySpace(pvObjects{}) }

type pvObjects struct{}

// pvObject is one entry of initial_state.objects, or an object
// expectation's equals.
type pvObject struct {
	Kind     string `json:"kind"`
	Owner    int    `json:"owner"`
	ID       int    `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Mines    int    `json:"mines"`
	Type     string `json:"type"`
	Detonate bool   `json:"detonating"`
	KnownTo  []int  `json:"known_to"`
	// Wormhole ends.
	Partner   int   `json:"partner"`
	Class     int   `json:"stability_class"`
	Years     int   `json:"years_since_jump"`
	DestKnown []int `json:"destination_known_to"`
	// Traders.
	Dest  []int  `json:"destination"`
	Warp  int    `json:"warp"`
	Met   []int  `json:"met"`
	Offer string `json:"offer"`
	// Packets.
	Target   int   `json:"destination_planet"`
	Minerals []int `json:"minerals"`
	Decay    int   `json:"decay_class"`
}

var pvMineKinds = map[string]objects.MineKind{"standard": objects.Standard, "heavy": objects.Heavy, "speed_bump": objects.SpeedBump}

func pvMarks(players []int) []bool {
	var m []bool
	for _, p := range players {
		for len(m) <= p {
			m = append(m, false)
		}
		m[p] = true
	}
	return m
}

func pvMarked(m []bool) []int {
	out := []int{}
	for p, ok := range m {
		if ok {
			out = append(out, p)
		}
	}
	return out
}

func pvOffer(name string) (objects.TraderItem, error) {
	switch name {
	case "research":
		return objects.TraderItem{Kind: objects.ItemResearch}, nil
	case "ship":
		return objects.TraderItem{Kind: objects.ItemShip}, nil
	}
	if b := slices.Index(objects.TraderPartNames[:], name); b >= 0 {
		return objects.TraderItem{Kind: objects.ItemPart, Bit: b}, nil
	}
	return objects.TraderItem{}, fmt.Errorf("trader offer %q", name)
}

func (pvObjects) Load(g *engine.Game, raw []json.RawMessage, m engine.PVMaps) (engine.SpaceObjects, error) {
	s := &objects.Space{}
	var traders []pvObject
	for _, r := range raw {
		var o pvObject
		if err := json.Unmarshal(r, &o); err != nil {
			return nil, err
		}
		switch o.Kind {
		case "minefield":
			k, ok := pvMineKinds[o.Type]
			if !ok {
				return nil, fmt.Errorf("minefield type %q", o.Type)
			}
			s.Minefields = append(s.Minefields, objects.Minefield{Owner: o.Owner, Number: o.ID, Kind: k, Pos: engine.Point{X: o.X, Y: o.Y}, Count: o.Mines, Detonate: o.Detonate, Known: pvMarks(o.KnownTo)})
		case "wormhole":
			id := m.EndID[o.ID]
			for len(s.Wormholes) <= id/2 {
				s.Wormholes = append(s.Wormholes, objects.Wormhole{})
			}
			s.Wormholes[id/2].Ends[id%2] = objects.WormholeEnd{Pos: engine.Point{X: o.X, Y: o.Y}, Class: o.Class, Years: o.Years, Known: pvMarks(o.KnownTo), DestKnown: pvMarks(o.DestKnown)}
		case "trader":
			traders = append(traders, o)
		case "packet":
			if len(o.Minerals) != engine.NumMinerals {
				return nil, fmt.Errorf("packet minerals %v", o.Minerals)
			}
			s.Packets = append(s.Packets, objects.Packet{Owner: o.Owner, Number: o.ID, Pos: engine.Point{X: o.X, Y: o.Y}, Target: m.Planet[o.Target], From: -1, Warp: o.Warp, Class: o.Decay, Cargo: engine.Minerals{o.Minerals[0], o.Minerals[1], o.Minerals[2]}})
		case "salvage":
			if len(o.Minerals) != engine.NumMinerals {
				return nil, fmt.Errorf("salvage minerals %v", o.Minerals)
			}
			m := engine.Minerals{o.Minerals[0], o.Minerals[1], o.Minerals[2]}
			steps := 0
			for _, x := range m {
				steps += (x + 9) / 10
			}
			g.Salvage = append(g.Salvage, engine.Salvage{Pos: engine.Point{X: o.X, Y: o.Y}, Minerals: m, Owner: o.Owner, Number: o.ID, Steps: steps})
		default:
			return nil, fmt.Errorf("object kind %q", o.Kind)
		}
	}
	sort.Slice(traders, func(i, j int) bool { return traders[i].ID < traders[j].ID })
	for _, o := range traders {
		item, err := pvOffer(o.Offer)
		if err != nil {
			return nil, err
		}
		if len(o.Dest) != 2 {
			return nil, fmt.Errorf("trader destination %v", o.Dest)
		}
		s.Traders = append(s.Traders, objects.Trader{Pos: engine.Point{X: o.X, Y: o.Y}, Dest: engine.Point{X: o.Dest[0], Y: o.Dest[1]}, Warp: o.Warp, Item: item, Served: pvMarks(o.Met)})
	}
	s.SortMinefields()
	return s, nil
}

func pvMismatch(field string, got, want any) string {
	return fmt.Sprintf("%s = %v, want %v", field, got, want)
}

// pvCompare checks each field in eq against got (both as JSON values);
// fields in skip are left unchecked, and the result is a skip naming them
// only when every checked field matched.
func pvCompare(what string, got map[string]any, eq map[string]json.RawMessage, tol int, skip map[string]string) string {
	var why string
	keys := make([]string, 0, len(eq))
	for k := range eq {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if reason, ok := skip[k]; ok {
			if why == "" {
				why = reason
			}
			continue
		}
		v, ok := got[k]
		if !ok {
			return "skip: " + what + " field " + k
		}
		var want any
		json.Unmarshal(eq[k], &want)
		gb, _ := json.Marshal(v)
		var g any
		json.Unmarshal(gb, &g)
		if gn, ok := g.(float64); ok && tol > 0 {
			if wn, ok := want.(float64); ok && gn-wn <= float64(tol) && wn-gn <= float64(tol) {
				continue
			}
		}
		if fmt.Sprint(g) != fmt.Sprint(want) {
			return what + ": " + pvMismatch(k, v, want)
		}
	}
	if why != "" {
		return "skip: " + why
	}
	return ""
}

func pvMinefield(s *objects.Space, owner, id int) *objects.Minefield {
	for i := range s.Minefields {
		if m := &s.Minefields[i]; m.Owner == owner && m.Number == id {
			return m
		}
	}
	return nil
}

func pvPacket(s *objects.Space, owner, id int) *objects.Packet {
	for i := range s.Packets {
		if p := &s.Packets[i]; p.Owner == owner && p.Number == id {
			return p
		}
	}
	return nil
}

var pvKindNames = map[objects.MineKind]string{objects.Standard: "standard", objects.Heavy: "heavy", objects.SpeedBump: "speed_bump"}

// pvRadius is the reason a minefield's radius is not compared: SCANNING.md
// defines no per-player known radius.
const pvRadius = "minefield radius (SCANNING.md defines no known radius)"

func minefieldFields(m *objects.Minefield) map[string]any {
	return map[string]any{"kind": "minefield", "owner": m.Owner, "id": m.Number, "x": m.Pos.X, "y": m.Pos.Y, "mines": m.Count, "type": pvKindNames[m.Kind], "detonating": m.Detonate, "known_to": pvMarked(m.Known)}
}

func traderFields(t objects.Trader, id int) map[string]any {
	offer := "research"
	switch t.Item.Kind {
	case objects.ItemShip:
		offer = "ship"
	case objects.ItemPart:
		offer = objects.TraderPartNames[t.Item.Bit]
	}
	return map[string]any{"kind": "trader", "id": id, "x": t.Pos.X, "y": t.Pos.Y, "destination": []int{t.Dest.X, t.Dest.Y}, "warp": t.Warp, "met": pvMarked(t.Served), "offer": offer}
}

func packetFields(p *objects.Packet, planet map[int]int) map[string]any {
	dest := -1
	for v, e := range planet {
		if e == p.Target {
			dest = v
		}
	}
	return map[string]any{"kind": "packet", "owner": p.Owner, "id": p.Number, "x": p.Pos.X, "y": p.Pos.Y, "destination_planet": dest, "warp": p.Warp, "minerals": p.Cargo[:], "decay_class": p.Class}
}

func (pvObjects) Check(g *engine.Game, e engine.PVExpect, m engine.PVMaps) string {
	s, ok := g.Objects.(*objects.Space)
	if !ok {
		return "skip: no space objects"
	}
	var eq map[string]json.RawMessage
	if len(e.Equals) > 0 && json.Unmarshal(e.Equals, &eq) != nil {
		return "skip: " + e.Kind + " list"
	}
	tol := 0
	json.Unmarshal(e.Tolerance, &tol)
	switch e.Kind {
	case "minefield":
		f := pvMinefield(s, *e.Owner, *e.ID)
		if f == nil {
			return fmt.Sprintf("minefield %d/%d gone", *e.Owner, *e.ID)
		}
		return pvCompare("minefield", minefieldFields(f), eq, tol, map[string]string{"radius": pvRadius})
	case "wormhole":
		id, ok := m.EndID[*e.ID]
		if !ok || id/2 >= len(s.Wormholes) {
			return fmt.Sprintf("wormhole end %d missing", *e.ID)
		}
		end := s.Wormholes[id/2].Ends[id%2]
		return pvCompare("wormhole", map[string]any{"x": end.Pos.X, "y": end.Pos.Y, "known_to": pvMarked(end.Known), "destination_known_to": pvMarked(end.DestKnown), "stability_class": end.Class, "years_since_jump": end.Years}, eq, tol, nil)
	case "trader":
		if *e.ID >= len(s.Traders) {
			return fmt.Sprintf("trader %d gone", *e.ID)
		}
		return pvCompare("trader", traderFields(s.Traders[*e.ID], *e.ID), eq, tol, nil)
	case "packet":
		p := pvPacket(s, *e.Owner, *e.ID)
		if p == nil {
			return fmt.Sprintf("packet %d/%d gone", *e.Owner, *e.ID)
		}
		return pvCompare("packet", packetFields(p, m.Planet), eq, tol, nil)
	case "object":
		var o pvObject
		json.Unmarshal(e.Equals, &o)
		switch o.Kind {
		case "trader":
			if o.ID >= len(s.Traders) {
				return fmt.Sprintf("trader %d missing", o.ID)
			}
			return pvCompare("trader", traderFields(s.Traders[o.ID], o.ID), eq, tol, nil)
		case "packet":
			p := pvPacket(s, o.Owner, o.ID)
			if p == nil {
				return fmt.Sprintf("packet %d/%d missing", o.Owner, o.ID)
			}
			return pvCompare("packet", packetFields(p, m.Planet), eq, tol, nil)
		case "minefield":
			f := pvMinefield(s, o.Owner, o.ID)
			if f == nil {
				return fmt.Sprintf("minefield %d/%d missing", o.Owner, o.ID)
			}
			return pvCompare("minefield", minefieldFields(f), eq, tol, map[string]string{"radius": pvRadius})
		}
		return "skip: object " + o.Kind
	case "object_gone":
		var sub struct {
			Kind  string `json:"kind"`
			Owner int    `json:"owner"`
			ID    int    `json:"id"`
		}
		json.Unmarshal(e.Subject, &sub)
		switch sub.Kind {
		case "minefield":
			if pvMinefield(s, sub.Owner, sub.ID) != nil {
				return "minefield still exists"
			}
		case "packet":
			if pvPacket(s, sub.Owner, sub.ID) != nil {
				return "packet still exists"
			}
		case "trader":
			if sub.ID < len(s.Traders) {
				return "trader still exists"
			}
		default:
			return "skip: object_gone " + sub.Kind
		}
		return ""
	}
	return "skip: " + e.Kind
}
