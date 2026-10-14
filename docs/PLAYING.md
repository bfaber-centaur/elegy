# Playing a game from the command line

This is the playable vertical slice: a person creates a game, looks at
what their player knows, gives orders, generates years, and saves and
continues, all with `cmd/elegy` and no hand-written JSON. There is no
other user interface yet.

Playable is not the same as faithful. A game played this way runs Elegy's
rules, whose parity with the original game is measured separately
([PARITY-STATUS.md](PARITY-STATUS.md)).

## A game against a computer player

Build the command once:

```sh
go build -o elegy ./cmd/elegy
```

Create a small galaxy with one human player (player 0) and Robotoid
(player 1). The game is the save file `game.json`:

```console
$ ./elegy new -seed 1 -players 1 -size small -ai robotoid -o game.json
game 2817366194855738197 year 2400 hash 4bc2612c...
```

### Looking at the game

`show` prints what one player knows, as text. It reads only that
player's report, so it shows nothing the player has not seen. The
sections are `summary` (the default), `planets`, `galaxy`, `fleets`,
`designs`, `research` and `messages`; `all` prints every one.

```console
$ ./elegy show -game game.json -player 0
Game 2817366194855738197, year 2400, player 0, ruleset elegy v6
Planets: 1, population 25000
Homeworld: Planet 937 (#37)
Fleets: 6, ships 6
Research: 15% of resources to energy (level 3), then same field
Messages: 0; last year's orders: 0, 0 rejected

$ ./elegy show -game game.json fleets
Fleets (6)
#0 Armed Scout #1, at Planet 937 (#37)
  ships: 1 Armed Scout
  fuel 50 mg; cargo ironium 0, boranium 0, germanium 0 kT, colonists 0
#1 Scout #2, at Planet 937 (#37)
...
#2 Colony Ship #3, at Planet 937 (#37)
  ships: 1 Colony Ship
  fuel 200 mg; cargo ironium 0, boranium 0, germanium 0 kT, colonists 0
...

$ ./elegy show -game game.json galaxy
Other planets (127)
Planet 325 (#45) at (1264, 1251), 44.0 ly, never seen
Planet 936 (#32) at (1196, 1261), 46.6 ly, never seen
...
```

Planets and fleets are named by their ids (`#37`, `#2`); orders use
those ids. `galaxy` lists every other planet nearest the homeworld
first, with what the player's latest report of it showed (habitability
for the player's race, mineral concentrations, owner) or "never seen".
Every player knows where every planet is.

### Giving orders

`order` adds one order to the player's order file for this year,
`orders<player>-<year>.json` unless `-o` names another. Before it writes
the order, it generates the year on a copy of the game with the whole
file, and an order the engine would reject is not added:

```console
$ ./elegy order -game game.json cargo -fleet 2 -planet 37 -colonists 25
orders0-2400.json: order 0 added (cargo)
$ ./elegy order -game game.json move -fleet 2 -to planet:45 -task colonize
orders0-2400.json: order 1 added (move)
$ ./elegy order -game game.json move -fleet 1 -to planet:32 -to planet:30 -warp 6
orders0-2400.json: order 2 added (move)
$ ./elegy order -game game.json research -budget 20 -field propulsion
orders0-2400.json: order 3 added (research)
$ ./elegy order -game game.json queue -planet 37 factories=10 mines=5 "ship:Colony Ship=1"
orders0-2400.json: order 4 added (queue)
$ ./elegy order -game game.json move -fleet 99 -to planet:1
elegy: order rejected, not added: fleet 99: orders: no such object
```

The order kinds:

| Kind | What it does | Flags |
|---|---|---|
| `research` | sets the research budget, field and next field | `-budget 0..100`, `-field NAME`, `-next same\|lowest\|NAME`; an unset flag keeps the current setting |
| `move` | replaces a fleet's waypoints | `-fleet ID`, `-to planet:ID\|fleet:ID\|X,Y` once per waypoint (`-to none` clears them), `-warp W`, `-task none\|colonize\|remote-mine\|scrap` at the last waypoint |
| `cargo` | loads (positive) or unloads (negative) cargo at a planet or another fleet where the fleet is | `-fleet ID`, `-planet ID` or `-with ID`, `-ironium`, `-boranium`, `-germanium` in kT, `-colonists` in kT (100 colonists each), `-fuel` in mg |
| `queue` | replaces a planet's production queue, or adds to it with `-append` | `-planet ID`, then items `NAME=COUNT`, or `ship:DESIGN=COUNT` and `starbase:DESIGN=COUNT` with the design's name or slot |

Later orders in the file build on earlier ones: `-append` adds to the
queue the file already sets, and an unset research flag keeps what the
file's earlier research order set.

`elegy order <kind> -h` lists a kind's flags. Every other order the
engine has (battle plans, designs, merges, splits, renames and the rest)
can still be written into the file by hand ([GAME-LOOP.md](GAME-LOOP.md)
"Order files"), and `check` checks a file written that way:

```console
$ ./elegy check -game game.json orders0-2400.json
5 orders checked, 0 would be rejected
```

The check is against the game as it stands: another player's orders for
the same year can still make an order fail.

### Generating years

`turn` generates the next year with every human player's order file;
computer players write their own orders, and a human player with no file
keeps their standing orders. The year is written back to the save file:

```console
$ ./elegy turn -game game.json orders0-2400.json
game 2817366194855738197 year 2401 hash 349c282a...
$ ./elegy turn -game game.json
game 2817366194855738197 year 2402 hash 29662a4c...
```

After a few years the colony ship has settled its planet:

```console
$ ./elegy show -game game.json planets messages
Planets (2)
Planet 937 (#37) at (1236, 1285), homeworld
...
Planet 325 (#45) at (1264, 1251)
  population 3000, mines 0, factories 0, defenses 0
...
Messages for 2403 (3)
built factories: Planet 937 (#37), count 1
built mines: Planet 937 (#37), count 5
no queue: Planet 325 (#45)
```

### Saving and continuing

Every command reads the save file and `turn` writes it, so a game is
saved after every year and can be continued at any time, on any machine
with the same build. A game continued from its save reaches the same
state as one played without stopping: `TestCLIPlayableSlice` plays the
walk-through above with a save and reload every year and checks that the
state hash and everything `show` prints match the same game played in
one process. `hash` prints a save's state hash to compare two copies.

Several human players take turns the same way: each writes their own
file with `-player N`, and one `turn` call takes every file.

## What the slice does not cover yet

- Ship design and battle plans have no `order` kind; they are written by
  hand.
- `show` does not compute what a fuller client would: resources per
  year, maximum population, fleet range, or the cost of queue items.
- Messages name the event and the objects it concerns, not the
  original's message text.
- Planet names are placeholders (`Planet 937`), not the original's name
  list.
- There is no interactive front end (sprint rule: no TUI unless it serves
  validation).
