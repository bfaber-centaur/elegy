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
