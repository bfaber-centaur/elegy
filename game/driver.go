// Package game runs a whole Elegy game around the turn engine: it asks
// each player's driver for orders, advances years with
// engine.GenerateTurn, and gives each player a report of only what that
// player knows. See docs/GAME-LOOP.md.
package game

import "github.com/bfaber-centaur/elegy/engine"

// Driver decides one player's orders for one year. The game loop calls
// it once per living player per year with that player's Report and
// submits what it returns as the player's order file for the year
// (engine.PlayerOrders; the loop stamps the game id, year and player).
//
// Computer opponents and scripted or human players implement the same
// interface. A driver sees only its Report: it is never handed the
// authoritative game state, so a planner cannot read another player's
// secrets by accident.
//
// Returning no orders and no error is a valid turn: the player keeps
// their standing orders (stars-elegy ORDERS.md "A player who submits
// nothing"). Returning an error stops the loop with a diagnostic naming
// the player and year; the year is not generated.
//
// A driver may keep state of its own between years. ELEGY CHOICE: such
// state is the driver's, not the game's; a saved game holds the game and
// its random stream only, and a driver that must survive a save and load
// has to rebuild its state from its Reports or save it itself.
type Driver interface {
	Orders(r Report) ([]engine.Order, error)
}

// DriverFunc adapts a function to Driver.
type DriverFunc func(r Report) ([]engine.Order, error)

// Orders calls f.
func (f DriverFunc) Orders(r Report) ([]engine.Order, error) { return f(r) }

// Idle is a driver that never submits orders: the player keeps their
// standing orders every year.
var Idle Driver = DriverFunc(func(Report) ([]engine.Order, error) { return nil, nil })
