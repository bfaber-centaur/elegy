# Game loop status

`game/` runs a whole Elegy game around the turn engine. It is lane B of
the 2026-10-08 plan: generate a galaxy, accept orders, advance years with
`engine.GenerateTurn`, save and load deterministic state, and give each
player a report of only what that player knows. There is no UI.

## Per-player drivers

Each year the loop asks every living player's `game.Driver` for orders:

```go
type Driver interface {
	Orders(r Report) ([]engine.Order, error)
}
```

`Report` is that player's knowledge at the start of the year:

- the player's own record, planets, fleets and designs, copied in full
  (a player knows everything it owns);
- the engine's `PlayerView` for the year just generated, which holds only
  what the player's scanners, battles and messages revealed (stars-elegy
  `SCANNING.md`);
- the previous year's messages to the player and the outcome of each of
  its orders.

A driver never receives the authoritative game state. Computer opponents
(Robotoid, Rototill, Cybertron), scripted test players and front ends all
implement the same interface; planners stay outside the engine's turn
order. `game.Idle` submits nothing, so the player keeps its standing
orders (ORDERS.md "A player who submits nothing").

ELEGY CHOICE: a driver's own state between years is the driver's, not
the game's. A saved game holds the game and its random stream; a driver
that must survive a save and load rebuilds its state from its reports or
saves it itself.

## Not done yet

- the loop itself, save and load, the state hash and the scripted
  30–50-year smoke test;
- an order file format for a CLI;
- the starting knowledge of a new game: a test builds it with
  `engine.Views` and population estimates drawn from the game's stream;
  the loop will fix that choice.
