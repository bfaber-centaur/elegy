package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/bfaber-centaur/elegy/engine"
)

// The order file format. ELEGY CHOICE: an order file is Elegy's own
// versioned JSON document (stars-elegy ORDERS.md "Scope and vocabulary":
// Elegy defines its own order format). Each order is an object naming its
// kind and holding the engine order's fields:
//
//	{"format": "elegy-orders", "version": 1, "game": 1, "year": 2400, "player": 0,
//	 "orders": [{"kind": "Research", "order": {"Budget": 15, "Field": 0, "Next": -2}}]}
const (
	OrdersFormat  = "elegy-orders"
	OrdersVersion = 1
)

// orderKinds names every engine order kind in the file format.
var orderKinds = map[string]engine.Order{
	"Research":       engine.ResearchOrder{},
	"BattlePlan":     engine.BattlePlanOrder{},
	"DeletePlan":     engine.DeletePlanOrder{},
	"FleetPlan":      engine.FleetPlanOrder{},
	"Rename":         engine.RenameOrder{},
	"Repeat":         engine.RepeatOrder{},
	"Merge":          engine.MergeOrder{},
	"Waypoint":       engine.WaypointOrder{},
	"Detonate":       engine.DetonateOrder{},
	"Queue":          engine.QueueOrder{},
	"PlanetSettings": engine.PlanetSettingsOrder{},
	"Relations":      engine.RelationsOrder{},
	"Cargo":          engine.CargoOrder{},
	"Design":         engine.DesignOrder{},
	"DeleteDesign":   engine.DeleteDesignOrder{},
	"Split":          engine.SplitOrder{},
	"MoveShips":      engine.MoveShipsOrder{},
}

// kindOf is the file name of an order's kind.
func kindOf(o engine.Order) (string, bool) {
	t := reflect.TypeOf(o)
	for k, v := range orderKinds {
		if reflect.TypeOf(v) == t {
			return k, true
		}
	}
	return "", false
}

// OrderFile is one player's orders for one year as a file holds them.
type OrderFile struct {
	GameID uint64
	Year   int
	Player int
	Orders []engine.Order
}

type orderFileJSON struct {
	Format  string      `json:"format"`
	Version int         `json:"version"`
	GameID  uint64      `json:"game"`
	Year    int         `json:"year"`
	Player  int         `json:"player"`
	Orders  []orderJSON `json:"orders"`
}

type orderJSON struct {
	Kind  string          `json:"kind"`
	Order json.RawMessage `json:"order"`
}

// ErrOrderFile is wrapped by every error reading an order file.
var ErrOrderFile = errors.New("game: bad order file")

// EncodeOrders writes an order file as indented JSON.
func EncodeOrders(w io.Writer, f OrderFile) error {
	doc := orderFileJSON{Format: OrdersFormat, Version: OrdersVersion, GameID: f.GameID, Year: f.Year, Player: f.Player, Orders: []orderJSON{}}
	for i, o := range f.Orders {
		k, ok := kindOf(o)
		if !ok {
			return fmt.Errorf("game: order %d: %T has no kind in the order file format", i, o)
		}
		b, err := json.Marshal(o)
		if err != nil {
			return fmt.Errorf("game: order %d: %v", i, err)
		}
		doc.Orders = append(doc.Orders, orderJSON{Kind: k, Order: b})
	}
	b, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// DecodeOrders reads an order file. An unknown kind or field refuses the
// whole file, naming the order.
func DecodeOrders(r io.Reader) (OrderFile, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var doc orderFileJSON
	if err := dec.Decode(&doc); err != nil {
		return OrderFile{}, fmt.Errorf("%w: %v", ErrOrderFile, err)
	}
	if doc.Format != OrdersFormat {
		return OrderFile{}, fmt.Errorf("%w: format %q, want %q", ErrOrderFile, doc.Format, OrdersFormat)
	}
	if doc.Version != OrdersVersion {
		return OrderFile{}, fmt.Errorf("%w: version %d; this build reads version %d", ErrOrderFile, doc.Version, OrdersVersion)
	}
	f := OrderFile{GameID: doc.GameID, Year: doc.Year, Player: doc.Player}
	for i, oj := range doc.Orders {
		proto, ok := orderKinds[oj.Kind]
		if !ok {
			return OrderFile{}, fmt.Errorf("%w: order %d: unknown kind %q", ErrOrderFile, i, oj.Kind)
		}
		ptr := reflect.New(reflect.TypeOf(proto))
		d := json.NewDecoder(bytes.NewReader(oj.Order))
		d.DisallowUnknownFields()
		if err := d.Decode(ptr.Interface()); err != nil {
			return OrderFile{}, fmt.Errorf("%w: order %d (%s): %v", ErrOrderFile, i, oj.Kind, err)
		}
		f.Orders = append(f.Orders, ptr.Elem().Interface().(engine.Order))
	}
	return f, nil
}
