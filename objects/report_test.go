package objects

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// ASSUMPTION V4: a sighting shows owner, number and position; minefields
// add mines and kind; only the owner sees its fields' detonate setting
// and its packets' warp, destination and cargo.
func TestPredictionObjectReport(t *testing.T) {
	l := newLab(t)
	g := l.g
	g.Salvage = []engine.Salvage{{Pos: engine.Point{X: 1200, Y: 1200}, Minerals: engine.Minerals{100, 0, 0}, Owner: 1, Number: 3}}
	s := &Space{
		Minefields: []Minefield{
			{Owner: 0, Number: 0, Kind: Standard, Pos: engine.Point{X: 1100, Y: 1100}, Count: 1600, Detonate: true},
			{Owner: 1, Number: 2, Kind: Heavy, Pos: engine.Point{X: 1150, Y: 1100}, Count: 900, Detonate: true},
		},
		Packets: []Packet{
			{Owner: 0, Number: 0, Pos: engine.Point{X: 1010, Y: 1020}, Target: 7, Warp: 9, Class: 2, Cargo: engine.Minerals{88, 0, 0}},
			{Owner: 1, Number: 5, Pos: engine.Point{X: 1030, Y: 1040}, Target: 2, Warp: 10, Class: 1, Cargo: engine.Minerals{0, 50, 0}},
		},
		Traders: []Trader{{Pos: engine.Point{X: 1300, Y: 1020}, Dest: engine.Point{X: 1300, Y: 1380}, Warp: 9}},
	}
	seen := engine.ObjectsSeen{
		Minefields: [][2]int{{0, 0}, {1, 2}, {1, 9}},
		Packets:    [][2]int{{0, 0}, {1, 5}},
		Traders:    []int{0},
		Salvage:    [][2]int{{1, 3}},
	}
	got := s.Report(g, 0, seen)
	want := ObjectReport{
		Minefields: []MinefieldSighting{
			{Owner: 0, Number: 0, Pos: engine.Point{X: 1100, Y: 1100}, Mines: 1600, Kind: Standard, Detonate: true},
			{Owner: 1, Number: 2, Pos: engine.Point{X: 1150, Y: 1100}, Mines: 900, Kind: Heavy},
		},
		Packets: []PacketSighting{
			{Owner: 0, Number: 0, Pos: engine.Point{X: 1010, Y: 1020}, Own: &OwnPacket{Warp: 9, Target: 7, Cargo: engine.Minerals{88, 0, 0}}},
			{Owner: 1, Number: 5, Pos: engine.Point{X: 1030, Y: 1040}},
		},
		Traders: []TraderSighting{{Index: 0, Pos: engine.Point{X: 1300, Y: 1020}}},
		Salvage: []SalvageSighting{{Owner: 1, Number: 3, Pos: engine.Point{X: 1200, Y: 1200}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Report(0) =\n%+v\nwant\n%+v", got, want)
	}

	// Player 1's report of the same objects: its own field shows the
	// setting and its own packet the details; player 0's do not.
	got = s.Report(g, 1, seen)
	if !got.Minefields[1].Detonate || got.Minefields[0].Detonate {
		t.Errorf("player 1 detonate settings: %+v", got.Minefields)
	}
	if got.Packets[0].Own != nil || got.Packets[1].Own == nil || *got.Packets[1].Own != (OwnPacket{Warp: 10, Target: 2, Cargo: engine.Minerals{0, 50, 0}}) {
		t.Errorf("player 1 packets: %+v", got.Packets)
	}
}
