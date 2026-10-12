package engine_test

import (
	"errors"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

func TestWaypointOrderObjectTargets(t *testing.T) {
	// A wormhole end or Trader target takes the object's position
	// (ORDERS.md "Waypoint upkeep"); a missing one is rejected.
	// ASSUMPTION L27: the player need not know it.
	space := &objects.Space{
		Wormholes: []objects.Wormhole{{Ends: [2]objects.WormholeEnd{{Pos: engine.Point{X: 1200, Y: 1300}}, {Pos: engine.Point{X: 1800, Y: 1700}}}}},
		Traders:   []objects.Trader{{Pos: engine.Point{X: 1500, Y: 1500}}},
	}
	g := engine.Game{
		ID: 7, Year: 2410, Size: 1,
		Players: []engine.Player{{}},
		Fleets:  []engine.Fleet{{ID: 1, Owner: 0, Pos: engine.Point{X: 1100, Y: 1100}}},
		Objects: space,
	}
	order := func(target engine.TargetKind, id int) engine.Order {
		return engine.WaypointOrder{Fleet: 1, Waypoints: []engine.Waypoint{{Target: target, ID: id, Warp: 6}}}
	}
	for _, tt := range []struct {
		target engine.TargetKind
		id     int
		want   engine.Point
		err    error
	}{
		{engine.TargetWormhole, objects.WormholeEndID(0, 1), engine.Point{X: 1800, Y: 1700}, nil},
		{engine.TargetTrader, 0, engine.Point{X: 1500, Y: 1500}, nil},
		{engine.TargetWormhole, objects.WormholeEndID(1, 0), engine.Point{}, engine.ErrNoSuchObject},
		{engine.TargetTrader, 3, engine.Point{}, engine.ErrNoSuchObject},
	} {
		gg := g
		gg.Fleets = append([]engine.Fleet(nil), g.Fleets...)
		a := engine.ApplyOrders(&gg, []engine.PlayerOrders{{Player: 0, GameID: 7, Year: 2410, Orders: []engine.Order{order(tt.target, tt.id)}}}, []int{0})
		var err error
		for _, r := range a.Results {
			if r.Err != nil {
				err = r.Err
			}
		}
		if !errors.Is(err, tt.err) && !(err == nil && tt.err == nil) {
			t.Errorf("target %v %d: error %v, want %v", tt.target, tt.id, err, tt.err)
			continue
		}
		if tt.err == nil {
			wp := gg.Fleets[0].Waypoints[0]
			if wp.Pos != tt.want || wp.Target != tt.target || wp.ID != tt.id {
				t.Errorf("target %v %d: waypoint %+v, want at %v", tt.target, tt.id, wp, tt.want)
			}
		}
	}
}

func TestWaypointOrderPacketObjectTargets(t *testing.T) {
	// A packet or salvage target, named by its owner and number, takes the
	// object's position. ASSUMPTION W9: a missing one keeps the order's
	// coordinates, and the waypoint check after the orders makes it a
	// plain go-to there. An owner who is not a player, or a negative
	// number, rejects the order; another target drops the owner.
	space := &objects.Space{Packets: []objects.Packet{{Owner: 1, Number: 2, Pos: engine.Point{X: 1500, Y: 1400}, Target: 9, Warp: 5}}}
	g := engine.Game{
		ID: 7, Year: 2410, Size: 1,
		Players: []engine.Player{{}, {}},
		Fleets:  []engine.Fleet{{ID: 1, Owner: 0, Pos: engine.Point{X: 1100, Y: 1100}}},
		Salvage: []engine.Salvage{{Owner: 1, Number: 0, Pos: engine.Point{X: 1300, Y: 1200}}},
		Objects: space,
	}
	pkt := func(owner, id int) engine.Waypoint {
		return engine.Waypoint{Target: engine.TargetPacketObject, Owner: owner, ID: id, Pos: engine.Point{X: 1250, Y: 1250}, Warp: 6}
	}
	for _, tt := range []struct {
		name string
		wp   engine.Waypoint
		want engine.Waypoint
		err  error
	}{
		{"salvage", pkt(1, 0), engine.Waypoint{Target: engine.TargetPacketObject, Owner: 1, ID: 0, Pos: engine.Point{X: 1300, Y: 1200}, Warp: 6}, nil},
		{"packet", pkt(1, 2), engine.Waypoint{Target: engine.TargetPacketObject, Owner: 1, ID: 2, Pos: engine.Point{X: 1500, Y: 1400}, Warp: 6}, nil},
		{"missing, W9", pkt(0, 2), engine.Waypoint{Pos: engine.Point{X: 1250, Y: 1250}, Warp: 6}, nil},
		{"owner not a player", pkt(2, 0), engine.Waypoint{}, engine.ErrOutOfRange},
		{"negative owner", pkt(-1, 0), engine.Waypoint{}, engine.ErrOutOfRange},
		{"negative number", pkt(1, -1), engine.Waypoint{}, engine.ErrOutOfRange},
		{"space drops the owner", engine.Waypoint{Owner: 1, Pos: engine.Point{X: 1250, Y: 1250}, Warp: 6}, engine.Waypoint{Pos: engine.Point{X: 1250, Y: 1250}, Warp: 6}, nil},
	} {
		gg := g
		gg.Fleets = append([]engine.Fleet(nil), g.Fleets...)
		order := engine.WaypointOrder{Fleet: 1, Waypoints: []engine.Waypoint{tt.wp}}
		a := engine.ApplyOrders(&gg, []engine.PlayerOrders{{Player: 0, GameID: 7, Year: 2410, Orders: []engine.Order{order}}}, []int{0})
		var err error
		for _, r := range a.Results {
			if r.Err != nil {
				err = r.Err
			}
		}
		if !errors.Is(err, tt.err) && !(err == nil && tt.err == nil) {
			t.Errorf("%s: error %v, want %v", tt.name, err, tt.err)
			continue
		}
		if tt.err != nil {
			continue
		}
		// Straight after the replay the waypoint holds the object's
		// position, or the order's for a missing object; the waypoint
		// check then makes that one plain.
		if wp := gg.Fleets[0].Waypoints[0]; wp.Pos != tt.want.Pos {
			t.Errorf("%s: after the orders %+v, want at %v", tt.name, wp, tt.want.Pos)
		}
		gg.WaypointCheckForTest()
		if wp := gg.Fleets[0].Waypoints[0]; wp != tt.want {
			t.Errorf("%s: waypoint %+v, want %+v", tt.name, wp, tt.want)
		}
	}
}
