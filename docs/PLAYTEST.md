# Playtest guide

This is the guide for sitting down and playing Elegy from the command
line: one human against one computer player. It is the release-candidate
playtest. [PLAYING.md](PLAYING.md) has the full reference for each
command; this file is the shortest path through a game, what to expect,
and how to report what breaks.

Playing is not the same as faithful. Elegy runs its own rules (ruleset
`elegy`), whose parity with the original game is measured separately
([COVERAGE.md](COVERAGE.md), [PARITY-STATUS.md](PARITY-STATUS.md)).
The human playtest is the next source of evidence: report anything that
feels wrong, not only crashes.

## Setup

You need Go (the version in `go.mod`) and a checkout of this repository.
From the repository root:

```sh
go build -o elegy ./cmd/elegy
./elegy help
```

Play in a directory of its own, so the save and order files stay
together. Every command reads and writes `game.json` in the current
directory unless `-game` names another file.

## The first turns, step by step

The commands below are literal: with `-seed 1` and this build, the game
is the same every time, so the ids in them (`#14`, fleet `2`, and so on)
are the ones you will see. `TestPlaytestScript` (cmd/elegy) runs every
`../elegy` line in the `sh` blocks of this section, in order, so they
stay correct. For your own games, find the ids with the discovery
commands in the next section instead.

### Year 2400: create the game and look around

```sh
mkdir -p playtest && cd playtest
../elegy new -seed 1 -players 1 -size tiny -ai robotoid
../elegy show
../elegy show planets fleets
../elegy show designs tech
../elegy show research plans
../elegy show galaxy
```

You are player 0; the computer player (Robotoid at expert level) is
player 1. Your homeworld is planet `#14`. `show planets` gives its
population, installations, minerals, resources a year and maximum
population; `show fleets` lists your six starting fleets by id (`#0` to
`#5`); `show designs` your ship and starbase designs; `show tech` every
hull and part you can build now, with each hull's slots; `show galaxy`
every other planet, nearest first.

### Year 2400: research, production, a new design, and orders

```sh
../elegy order research -budget 15 -field propulsion -next lowest
../elegy order design -name Gunboat -hull Destroyer 0:1:"Long Hump 6" 1:1:"X-Ray Laser" 2:1:"X-Ray Laser" 3:1:"Rhino Scanner" 4:2:Tritanium 6:1:"Battle Computer"
../elegy order queue -planet 14 factories=5 mines=3 ship:Gunboat=1
../elegy order move -fleet 1 -to planet:11 -to planet:18 -to planet:20
../elegy order move -fleet 0 -to planet:9 -to planet:8 -warp 5
../elegy order cargo -fleet 2 -planet 14 -colonists 25
../elegy check orders0-2400.json
../elegy turn orders0-2400.json
```

Each `order` line adds one order to `orders0-2400.json` after checking
it against a trial run of the year; the design order prints the design
the year will store. The queue builds factories, mines and one Gunboat.
The scout (fleet 1) and the armed scout (fleet 0) explore; the colony
ship (fleet 2) loads 2500 colonists. `turn` generates the year and
writes it back to `game.json`.

### Years 2401 to 2403: explore, then colonize

```sh
../elegy order move -fleet 4 -to planet:12 -to planet:16
../elegy turn orders0-2401.json
../elegy turn
../elegy show galaxy
../elegy order move -fleet 2 -to planet:18 -task colonize
../elegy order move -fleet 4 -to planet:21
../elegy turn orders0-2403.json
```

The destroyer (fleet 4) goes south. By 2403 the scouts have seen
planet `#18`, habitability 20%, the first planet seen that your race
can live on; the colony ship goes there to colonize, and the destroyer
heads for planet `#21`. `turn` with no file keeps your standing orders.

### Years 2404 to 2406: the new ship, the colony, and the opponent

```sh
../elegy turn
../elegy show messages
../elegy show fleets
../elegy order move -fleet 10 -to planet:21
../elegy turn orders0-2405.json
../elegy turn
../elegy show messages
../elegy show galaxy
```

The 2404 messages say the Gunboat was built (fleet `#10`) and planet
`#18` was colonized; the Gunboat follows the destroyer. In 2406 the
destroyer reaches planet `#21`, the computer player's homeworld: the
messages for 2406 report a battle there and your side's summary, and
`show galaxy` now lists planet `#21` as owned by player 1 and the
fleets of player 1 you saw.

### Save, exit, reload and continue

The save file is the game: every `turn` writes it, and every command
reads it, so you can stop at any point and continue later with the same
build. To keep a copy and check that a reloaded game continues exactly
as the original:

```sh
cp game.json game-2407.json
../elegy turn
../elegy turn -game game-2407.json
../elegy hash
../elegy hash -game game-2407.json
```

The two `hash` lines print the same year and state hash. Copies of a
save played with the same orders stay identical.

## Finding ids in your own games

Ids depend on the seed and on what has happened. These find them:

| You want | Run |
|---|---|
| your planets and their ids | `../elegy show planets` (the `(#N)` after the name) |
| your homeworld | `../elegy show` (the `Homeworld:` line) |
| your fleets and their ids | `../elegy show fleets` (`#N` at the start of each fleet) |
| planets you could colonize | `../elegy show galaxy \| grep "habitability [1-9]"` |
| the opponent's planets | `../elegy show galaxy \| grep "owned by player"` |
| enemy fleets you have seen | `../elegy show galaxy` (the "Other players' fleets seen" list; `fleet #N` is their id for `-to fleet:N`) |
| what a hull's slots take | `../elegy show tech` |
| a design's slot fills, to copy | `../elegy show designs` (the `slots:` line, in the `SLOT:COUNT:PART` form `order design` takes) |
| battle plan numbers | `../elegy show plans` |
| last year's messages and rejected orders | `../elegy show messages` |

Messages are only last year's; read them each year before the next
`turn`.

## Commands

| Command | What it does | Example |
|---|---|---|
| `new` | creates a game (`-seed`, `-players` humans, `-ai` computer players `robotoid`, `rototill`, `cybertron`, each optionally `:easy`/`:standard`/`:harder`/`:expert`; `-size tiny..huge`; `-positions`; `-density`; `-rules elegy\|jrc3-faithful`; `-race FILE` for a custom race) | `elegy new -seed 7 -players 1 -size small -ai cybertron:standard` |
| `race` | writes a race file to edit and pass to `new -race` | `elegy race -h` |
| `show` | prints what one player knows; sections `summary` (default), `planets`, `galaxy`, `fleets`, `designs`, `tech`, `plans`, `research`, `messages`, `all` | `elegy show planets fleets` |
| `order` | adds one checked order to this year's file `orders<player>-<year>.json` | `elegy order queue -planet 14 mines=5` |
| `check` | checks order files against a trial run of the year | `elegy check orders0-2400.json` |
| `turn` | generates the next year with the human players' order files | `elegy turn orders0-2400.json` |
| `hash` | prints the save's year and state hash | `elegy hash` |
| `report`, `orders` | the raw JSON report and order files, for tools | `elegy report -h` |
| `play` | runs a computer-only game for some years | `elegy play -h` |
| `vector` | runs a parity vector (research use) | `elegy vector -h` |

`elegy <command> -h` lists a command's flags; `elegy order -h` lists the
order kinds and `elegy order <kind> -h` a kind's flags.

The order kinds:

| Kind | Example |
|---|---|
| `research` | `order research -budget 20 -field weapons -next lowest` |
| `move` | `order move -fleet 4 -to planet:21 -to 1200,1100 -warp 7`; `-task colonize\|remote-mine\|scrap` at the last waypoint; `-to none` clears |
| `cargo` | `order cargo -fleet 3 -planet 14 -ironium 50 -colonists 10` (negative unloads; `-with FLEET` for fleet to fleet; `-fuel` in mg) |
| `queue` | `order queue -planet 14 factories=10 mines=5 "ship:Gunboat=2"`; `-append` adds to the queue |
| `design` | `order design -name Hauler -from "Medium Freighter" 2:1:Tritanium` or `-hull HULL SLOT:COUNT:PART...`; `-starbase` for starbase designs |
| `delete-design` | `order delete-design -slot 6` |
| `plan` | `order plan -index 5 -name Hunt -tactic maximize-damage -primary armed -attack everyone` |
| `delete-plan` | `order delete-plan -index 5` |
| `fleet-plan` | `order fleet-plan -fleet 10 -plan 5` |

Queue item names are `mines`, `factories`, `defenses`, `alchemy`, the
auto items and packet items (`elegy order queue -h` lists them), and
`ship:DESIGN` or `starbase:DESIGN` by the design's name or slot. A design
added earlier in the same order file can be queued by name.

## When an order is rejected

`order` checks every order with a trial run of the year before writing
it. A rejected order is not written, and the error says why:

```console
$ ../elegy order move -fleet 99 -to planet:1
elegy: order rejected, not added: fleet 99: orders: no such object
```

Then:

1. Read the reason. Most rejections are a wrong id (check `show fleets`
   or `show planets`), an object that is not yours, or a value out of
   range.
2. Fix the command and run it again. Orders already in the file are
   kept; `order` warns if an earlier order in the file would now be
   rejected.
3. To start the year's orders over, delete `orders<player>-<year>.json`.
4. If you think the order should have been accepted, report it (below)
   with the save and the order file before you change anything.

`turn` never fails on a rejected order: it skips it, and next year's
`show messages` lists it as `order N rejected: reason`. The check is
against the game as it stands; another player's orders for the same
year can still make an order fail.

## Keeping a failing game

Before you change anything after something looks wrong:

```sh
mkdir -p ../bug-report
cp game.json orders*.json ../bug-report/
../elegy hash > ../bug-report/hash.txt
```

The save holds the whole game, including the computer players' state
and the random generator, so with the same build and the same order
files the next year comes out the same. Keep the save from before the
year that went wrong; if you only have the later one, keep that too.
Saves are tied to the build's save version: a different build may
refuse to load an old save.

## Bug report format

Copy this for each problem:

```text
Title: one line

Build: elegy commit <git rev-parse HEAD>
Command: the exact command line
Expected: what you expected to happen
Actual: what happened (paste the output)
Saved game: path to the save before the problem, the order files, and the hash
Steps to reproduce:
  1. ../elegy new ... (or: start from the attached save)
  2. ...
Notes: anything else (how it differs from the original game, if you know)
```

## Running the regression tests

From the repository root:

```sh
go test ./...                                            # the full suite, including the CLI walkthrough
go test ./cmd/elegy -run 'TestCLIPlayableSlice|TestPlaytestScript' -v
go test ./ai -run TestDriversSaveReload -v               # computer players across save and reload
go test ./regress -run '^TestLongGames$' -count=1 -v     # PR subset: 24 games of 60 years
go test ./regress -run '^TestLongGames$' -count=1 -v -timeout 60m -args -seeds=1,2,3,4,5,6,7,8 -years=150 -save-at=77 -size=2
```

The last line is the release matrix: 64 games of 150 years on a medium
map, each replayed and reloaded ([REGRESSION.md](REGRESSION.md)).

## Known limitations and missing actions

What the command line cannot do yet:

- No `order` kind for merging, splitting or renaming fleets, moving ships
  between fleets, following another fleet, repeating orders, planet
  settings (routes, packets, leftover-only research), relations with
  other players, the default queue or minefield detonation. The engine
  has all of them; they can be written into the order file by hand
  ([GAME-LOOP.md](GAME-LOOP.md) "Order files") and checked with `check`.
- `show` does not compute fleet range or fuel use, the cost or time of
  queue items, or how much research a year buys. Watch `fuel` in
  `show fleets`: a long trip can leave a fleet stranded.
- Messages name the event and the objects it concerns, not the
  original's message text, and some print a raw count (for example
  `battle: ..., count 3` is the number of battle tokens). There is no
  battle viewer.
- Planet names are placeholders (`Planet 482`), and the number in the
  name is not the id: orders use the `#` id.
- Mystery Trader parts are not listed by `show tech`.
- `show -player 1` prints a computer player's report. That is for
  debugging; it shows what you would not know.
- No interactive front end, by design for this release.

What is likely to differ from the original game (details in
[COVERAGE.md](COVERAGE.md) "Biggest blind spots"):

- Random outcomes (battles, mining remainders, random events) use
  Elegy's own generator, so a game never replays the original's dice.
- The computer players are Elegy's own, measured against no original
  game; expect different and sometimes odd strategies.
- Rules the original's client computes for display (estimates) are
  partly modelled.
- A small set of messages the original sends are not sent yet
  (COVERAGE.md lists the unmapped message ids).

What a human should test first:

1. The walkthrough above, then a longer game of 30 or more years on
   `-size small`.
2. Colonizing several planets, moving colonists and minerals with the
   freighter, and building a starbase on a colony.
3. A war: design warships with `show tech`, set battle plans, attack the
   computer player's planets, and bomb them.
4. Research across several fields and designs that use new parts.
5. Save, copy and continue at several points; compare `hash`.
