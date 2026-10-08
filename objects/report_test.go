package objects

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// SC-038 (MEASURED): a sighting shows the whole object, whoever owns it,
// except the "known" marker, which shows only the viewer's own entry.
// Only the objects in the seen set appear.
func TestMeasuredObjectReport(t *testing.T) {
	l := newLab(t)
	g := l.g
	g.Salvage = []engine.Salvage{{Pos: engine.Point{X: 1200, Y: 1200}, Minerals: engine.Minerals{100, 0, 0}, Owner: 1, Number: 3}}
	s := &Space{
		Minefields: []Minefield{
			{Owner: 0, Number: 0, Kind: Standard, Pos: engine.Point{X: 1100, Y: 1100}, Count: 1600, Detonate: true, Known: []bool{false, true}},
			{Owner: 1, Number: 2, Kind: Heavy, Pos: engine.Point{X: 1150, Y: 1100}, Count: 900, Detonate: true, Known: []bool{true, false}},
			{Owner: 1, Number: 4, Kind: SpeedBump, Pos: engine.Point{X: 1390, Y: 1390}, Count: 400},
		},
		Packets: []Packet{
			{Owner: 0, Number: 0, Pos: engine.Point{X: 1010, Y: 1020}, Target: 7, Warp: 9, Class: 2, Cargo: engine.Minerals{88, 0, 0}, New: true},
			{Owner: 1, Number: 5, Pos: engine.Point{X: 1030, Y: 1040}, Target: 2, Warp: 10, Class: 1, Cargo: engine.Minerals{0, 50, 0}},
		},
		Traders: []Trader{{Pos: engine.Point{X: 1300, Y: 1020}, Dest: engine.Point{X: 1300, Y: 1380}, Warp: 9, Served: []bool{false, true}, Item: TraderItem{Kind: ItemPart, Bit: 3}}},
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
			{Owner: 1, Number: 2, Pos: engine.Point{X: 1150, Y: 1100}, Mines: 900, Kind: Heavy, Detonate: true, Known: true},
		},
		Packets: []PacketSighting{
			{Owner: 0, Number: 0, Pos: engine.Point{X: 1010, Y: 1020}, Target: 7, Warp: 9, Cargo: engine.Minerals{88, 0, 0}, Class: 2, New: true},
			{Owner: 1, Number: 5, Pos: engine.Point{X: 1030, Y: 1040}, Target: 2, Warp: 10, Cargo: engine.Minerals{0, 50, 0}, Class: 1},
		},
		Traders: []TraderSighting{{Index: 0, Pos: engine.Point{X: 1300, Y: 1020}, Dest: engine.Point{X: 1300, Y: 1380}, Warp: 9, Served: []bool{false, true}, Item: TraderItem{Kind: ItemPart, Bit: 3}}},
		Salvage: []SalvageSighting{{Owner: 1, Number: 3, Pos: engine.Point{X: 1200, Y: 1200}, Minerals: engine.Minerals{100, 0, 0}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Report(0) =\n%+v\nwant\n%+v", got, want)
	}

	// Player 1 sees the same contents; only its own known entries differ.
	got = s.Report(g, 1, seen)
	if !got.Minefields[0].Known || got.Minefields[1].Known {
		t.Errorf("player 1 known entries: %+v", got.Minefields)
	}
	for i := range want.Minefields {
		want.Minefields[i].Known = got.Minefields[i].Known
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Report(1) =\n%+v\nwant\n%+v", got, want)
	}

	// The report is a copy: changing it leaves the Trader alone.
	got.Traders[0].Served[0] = true
	if s.Traders[0].Served[0] {
		t.Error("the report shares the Trader's served list")
	}
}
