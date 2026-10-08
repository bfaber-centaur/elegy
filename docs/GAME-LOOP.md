# Game loop status

`game/` runs a whole Elegy game around the turn engine, with no original
program: generate a galaxy, ask each player for orders, advance years with
`engine.GenerateTurn`, save and load the game, and give each player a
report of only what that player knows. There is no UI.

```go
g, err := game.New(rules, settings, seed) // engine.Ruleset, newgame.Settings, uint64
err = g.Run(40, drivers)                // or g.Advance(drivers) per year
r, err := g.Report(player)              // what that player knows now
err = g.Save(w); g2, err := game.Load(rd)
h, err := g.Hash()                      // SHA-256 of the save document
diffs, err := game.Diff(g, g2, 20)      // where two games differ
```

## The year

`Advance` asks every living player's `Driver` for orders, in player
order, then calls `engine.GenerateTurn` once with all the order files.
`engine/` owns the turn itself; the loop adds nothing between orders and
the turn.

- Drivers draw random numbers only from `Report.Rand`, which is the
  game's stream, before the year is generated. Every draw shifts the
  turn's later draws (stars-elegy AI.md §1 "Random numbers").
- On any error the game is unchanged and the stream does not advance:
  - `*DriverError` names the player whose driver failed;
  - `*TurnError` means the engine refused the year (for example
    `engine.ZeroMaxPopulationError`);
  - `*InvariantError` means the new state failed `Check`.
- `Check` tests Elegy's data model, not game rules: every owner is a
  player, every fleet has ships of existing designs, quantities are not
  negative, ids and design slots are unique.

`New` takes the game's ruleset (`engine.ElegyRules()`,
`engine.FaithfulRules()` or another valid `engine.Ruleset`); the game
carries it for its whole life, so games with different rulesets run side
by side in one process (`TestRulesetsCoexist`).

`New` gives the game one generator, `newgame.NewRand(seed)`, for
everything: creation, then the starting knowledge, then every year
(ELEGY CHOICE; the original's stream is not in the spec). It also sets
the engine's race check (`races.GameRaces`), the terraforming rules
(`terraform.Rules`) and the space objects, so every subsystem the engine
has runs.

ELEGY CHOICE: the first year's reports are `engine.Views` of the new
game. Their population estimates are drawn from the stream right after
creation.

## Player reports and drivers

```go
type Driver interface {
	Orders(r Report) ([]engine.Order, error)
}
```

Computer opponents, scripted players and front ends all implement
`Driver`. A driver gets only its `Report`, never the authoritative game.
`game.Idle` submits nothing, so the player keeps its standing orders
(ORDERS.md "A player who submits nothing").

| Report field | What it holds | Source |
|---|---|---|
| `Rules` | the game's ruleset | a game's rules are known to every player |
| `Self`, `Planets`, `Fleets`, `Designs` | the player's own record, planets, fleets and designs, in full | SCANNING.md: a player knows everything about its own objects |
| `View` | the engine's `PlayerView` from the year just generated | SCANNING.md |
| `Events`, `Orders` | the previous year's messages to the player, and the outcome of each of its orders | engine `TurnResult` |
| `History` | for every planet ever reported to the player, the latest report and the year it describes; lost colonies stay recorded as the player's own | AI.md §1 "What it sees" (CONFIRMED AI-12); ASSUMPTION G1 below |
| `Universe` | every planet's id, position and name index, seen or not | AI.md §1 (CONFIRMED AI-12: colonizers flown to never-scanned planets); ai/rototill.md §3 (MEASURED AI-17: scouts to the nearest never-seen planet) |
| `Wormholes` | for a computer player, each wormhole end it saw this year; for a human player, also each end it still knows but did not see, as last seen. Each shows the year seen, position, the stability its report named and the years since its last jump, plus its destination when the player knows it and sees the other end this year | SCANNING.md "Space objects" (MEASURED SC-038: what a sighting shows; nothing carries over in computer players' history); OBJECTS.md "Stability" (display BINARY-ONLY; file holds class and years, MEASURED SC-038), "Destination knowledge" (CONFIRMED WT-001, WT-004; file holds the partner id and every player's entry, MEASURED SC-038); ASSUMPTION G2 and the display ELEGY CHOICE below |
| `KnownDesigns` | other players' designs the player knows: hull and mass for every design seen; the whole design, parts included, once shown in full (a War Monger viewer, or a battle the player took part in); the last year seen | SCANNING.md "Designs" (CONFIRMED SC-001..SC-023, SC-015, SC-031, SC-036; MEASURED SC-037, computer players' files, for what later years keep); ASSUMPTION G3 below |
| `Objects` | the minefields, packets, Mystery Traders and salvage the player saw this year, each shown whole (`objects.ObjectReport`) | SCANNING.md "Space objects" (CONFIRMED, who sees what; MEASURED SC-038, what a sighting shows); ASSUMPTION V4 in OBJECTS-STATUS.md for what SC-038 leaves open |
| `Rand` | the game's stream, during `Advance` only | AI.md §1 "Random numbers" |

Not in the report:

- Other players' primary racial traits. A known player is identified by
  name only (SCANNING.md "Players", CONFIRMED); a Claim Adjuster viewer
  also gets habitability ranges, which are in `View.Players`.
- A wormhole end's own stability class, its partner's id and other
  players' destination knowledge. The report gives the jump chance and
  the destination as the original displays them (ELEGY CHOICE below).

**ASSUMPTION G1:** a newer planet report replaces the whole history
record, whatever its level. stars-elegy does not say how the original
merges a lower-level report into its history.

A computer player's report lists only the wormhole ends it saw this
year. In computer players' history files, nothing about an object
carries over after it leaves view (SCANNING.md "Space objects", MEASURED
SC-038).

**ASSUMPTION G2:** a human player's report also lists a known wormhole
end the player did not see this year, with its last-seen values, until a
jump makes the player forget it. The original's client writes a human
player's history, and what it keeps is UNRESOLVED (SC-038).

**ELEGY CHOICE:** the wormhole report follows the original's display,
not its player file. The file carries each seen end's class and years
since its last jump, its partner's id and every player's
destination-known entry (OBJECTS.md "Destination knowledge" and
"Stability", MEASURED SC-038). The display shows a stability name
computed from class and years, and hides the pairing unless the viewer
transited (BINARY-ONLY). So `Wormholes` gives the jump chance, not the
class, and withholds the partner id and other players' transit entries.
The Report is the player's view, and withholding them keeps one
player's transits from leaking to another. The years since the last
jump are the exception: the file carries them, and they say nothing
about other players.

A known design is reported as it was shown. An owner can put a new
design in an unused slot under the same index (`engine.DesignOrder`),
and the report keeps the old design until a view shows the new one.

A known design stays in the report in later years when no view shows
it, and a design once shown in full stays known in full: the original's
history keeps every design it learned and merges each year's report,
keeping the higher level (SCANNING.md "Designs", MEASURED SC-037, in
computer players' files). A partial sighting of a different design in
the same slot replaced a partial record (MEASURED once, SC-037).

**ASSUMPTION G3** covers what SC-037 leaves UNRESOLVED or untested:

- A partial sighting whose hull and mass match a full record keeps the
  full design, which may be stale; one with another hull or mass
  replaces it. Whether a different design replaces a full record is
  UNRESOLVED.
- A design its owner deleted stays known. Whether the original ever
  drops it is UNRESOLVED.
- Human players' reports follow the same rule. The original's client
  writes a human player's history, and whether it behaves the same is
  UNRESOLVED.

**ELEGY CHOICE:** a driver's own state between years belongs to the
driver, not the game. A saved game holds the game, its stream and the
players' histories. A driver that must survive a save and load rebuilds
its state from its reports; `ai.Driver` keeps none. AI.md §1 says the original's computer players
keep no memory between years beyond the planet history (CONFIRMED
AI-10).

## Saved games

**ELEGY CHOICE:** a saved game is Elegy's own versioned JSON document,
`{"format": "elegy-save", "version": 7, ...}`, not any of the original's
file formats. It holds:

- the engine state, including the game's ruleset (`engine.Game.Rules`:
  identity, version and every legacy switch, docs/RULESET.md), which
  `Load` validates;
- the space objects and race designs (the engine's `Objects` and `Races`)
  and whether terraforming is on;
- every design slot's creation year, picture and ships built
  (`engine.DesignSlot.Created`, `Picture`, `Built`; ASSUMPTION B2 for the
  starting designs: the first year, picture 0);
- the generator state;
- the planet name indexes;
- each computer player's level, in the engine state
  (`engine.Player.Level`, which `Game.ComputerLevel` reads), so a loaded
  game runs its computer players and the Mystery Trader's planet trades
  at the same level;
- what the last year told each player (views, events, order outcomes,
  planet histories, wormhole sightings, known designs).

An order outcome keeps its message and which engine error it wraps, so
`errors.Is` gives the same answer after a load.

The encoding is canonical: a game encodes to the same bytes every time,
and a loaded game saves back byte for byte. `Hash` is the SHA-256 of
those bytes. `Load` refuses an unknown format, another version, unknown
fields, a state that fails `Check` and a level out of range (0..3 for a
computer player, 0 for a human). Version 7 put the level in the engine
state; a version 6 save is refused because it would load every computer
player as Easy, and because its design slots' `Built` counts may predate
the field.

## The smoke test

`game/smoke_test.go` plays a small three-player galaxy for 40 years with
a scripted `expander` driver. The driver researches, builds factories and
mines on every planet, builds colony ships, flies loaded colonizers to
the nearest habitable unowned planet in its history, and sends scouts to
never-seen planets picked with `Report.Rand`. The tests check that:

- every scripted order is accepted, and players colonize;
- two runs with the same seed give the same hash every year, and another
  seed gives a different game;
- a game saved after year 17 and loaded again continues to the same
  hash every year until year 40;
- a failed year leaves the game unchanged;
- drivers run in player order on the game's stream;
- reports, errors included, survive a reload.

When a determinism check fails, the test prints the first JSON paths
where the two games differ (`Diff`), which name the objects that
diverged.

This is not a parity test. Elegy's stream and draw order are its own, so
nothing here is compared with the original.

## Order files

**ELEGY CHOICE:** an order file is Elegy's own versioned JSON document.
ORDERS.md "Scope and vocabulary" says Elegy defines its own order format.
Each order names its kind (the engine order type without `Order`) and
holds that type's fields:

```json
{"format": "elegy-orders", "version": 1, "game": 970338138316348672, "year": 2400, "player": 0,
 "orders": [{"kind": "Research", "order": {"Budget": 15, "Field": 0, "Next": -2}}]}
```

`game.EncodeOrders` and `game.DecodeOrders` read and write the files.
An unknown kind or field refuses the whole file. A test parses the
engine's source, so every engine order kind has a name in the format.

**ELEGY CHOICE:** the game id order files carry (ORDERS.md "Wrong game
or wrong year") is the first eight bytes of the SHA-256 of the new-game
settings and seed, and never 0. The spec does not say how the original
picks one.

## Command line

`cmd/elegy` plays games from save files and order files:

```sh
go run ./cmd/elegy new -seed 1 -players 3 -size small -rules elegy -o game.json   # or -rules jrc3-faithful
go run ./cmd/elegy new -race default -race me.json -race random -ai rototill:expert,cybertron -o game.json
go run ./cmd/elegy race -name Testers > me.json                     # a race file to edit
go run ./cmd/elegy orders -game game.json -player 0 > orders0.json   # an empty file to fill
go run ./cmd/elegy turn -game game.json orders0.json                # players without a file keep standing orders
go run ./cmd/elegy report -game game.json -player 0                 # the player's Report as JSON
go run ./cmd/elegy hash -game game.json
go run ./cmd/elegy play -seed 1 -players 1 -ai robotoid,rototill,cybertron -years 40
```

Players:

- `-players N` makes N human players with `races.Default()`.
- `-race` makes one human player per use. Its value is `default`,
  `random` (the wizard's Random race, generated at creation, RACES.md
  "Random race") or a race file.
- `-ai` adds computer players after the humans, in the order given:
  `robotoid`, `rototill` or `cybertron`, each with an optional level,
  `easy`, `standard`, `harder` or `expert`. Each gets its type's built-in
  race (AI.md "Built-in races"). These are the three personalities the
  project implements (AI.md "Project policy"). **ELEGY CHOICE:** the level
  defaults to expert, the level the `ai` package's own long games play.

`turn` and `play` run every computer player with `ai.Driver`. The
personality comes from the player's PRT (AI.md table: HE Robotoid, CA
Rototill, PP Cybertron). `turn` refuses an order file for a computer
player.

`turn` loads the game for every year, so its computer players start
each year with fresh drivers. `ai.Driver` keeps no state between years:
it reads its designs' creation years and pictures from `OwnDesign.Slot`,
which the save keeps. `ai`'s `TestDriversSaveReload` shows a game with
all three computer players, saved and reloaded into fresh drivers after
year 25 or after every year, continuing to the same hash as the
uninterrupted game for 50 years, under `elegy` and `jrc3-faithful`.

**ELEGY CHOICE:** a race file is Elegy's own versioned JSON document,
`{"format": "elegy-race", "version": 1, "race": {...}}`, holding a
`races.Design`.

`turn` refuses an order file for another game, another year or a
missing player before anything runs. It prints each rejected order and
writes the next year back to the save file, through a temporary file.
`play` prints each year's hash, each rejected order and, at the end,
every player's planets and ships, with the number of computer-player
steps Elegy cannot order yet (`ai.Result.Unsupported`).

`cmd/elegy`'s tests play 40 years against each computer player alone
and against all three together. Every computer player's order must be
accepted, every computer player must keep a planet, and the same seed
must give the same hashes.
