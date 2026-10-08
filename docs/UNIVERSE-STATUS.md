# New-game generation: implementation status

`newgame/` builds a new game (year 2400) from game settings, following the
public specification in `bfaber-centaur/stars-elegy`: `docs/UNIVERSE.md`,
with `KERNEL.md` "Habitability", `COMPONENTS.md` (the component table the
engine already embeds), `OBJECTS.md` "Wormholes" and `AI.md` "Built-in
races", as of stars-elegy `main` at `eac9a78`. Nothing here comes from the
private archaeology repositories.

The package reads the engine's state types and changes nothing in
`engine/`. Its entry point is `newgame.Generate(settings, rng)`, which
returns an `engine.Game` plus what the engine has no field for yet: planet
name indexes, artifacts, wormholes and where each player's starting objects
are.

## Randomness

UNIVERSE.md deliberately leaves out the original's random stream and draw
order. Elegy uses its own seeded generator (`NewRand`, SplitMix64 with an
unbiased bounded draw) and its own draw order. The same settings and seed
always give the same game. Tests therefore check rules through the rule
functions with scripted draws, and check games through invariants and
distributions over Elegy seeds, never against original seeds.

## Tests: ground truth versus predictions

Test names follow the engine's convention:

- `TestConfirmed*`: rules UNIVERSE.md marks CONFIRMED (or MEASURED), with
  the UG vectors it cites: starting tech per PRT, the JOAT minerals spend
  (423/253/234 → 548/378/484, UG16), the JOAT concentrations spend
  (53/30/82 → 66/68/95, UG20), the WM/SD/IT installation spends (UG16),
  BBS population 750 and IT 600/300 (UG03, UG21), the planet-0
  concentration floor (15/70/90 → 30/70/90, UG16), the WM scout's Yakimora
  Light Phaser and the SS scout's Possum Scanner, JOAT's six designs and
  fleets (UG21), the tiny-map PP/IT start (UG19), huge dense = huge packed
  (UG05, UG15), the measured planet counts, the second-planet redraw
  fallback (UG29, UG30) and the 24 built-in computer races (AI-0).
- `TestPrediction*`: BINARY-ONLY rules: the seed-dependent planet count
  distributions (Elegy's spacing pass lands in the spec's sampled 899–940
  and 930–962 ranges) and wormhole spacing.
- `TestElegyDecision*`: Elegy's own choices: determinism, the generator,
  the shared-minerals switch, and a generated game running through
  `engine.GenerateTurn`.

| Rule (UNIVERSE.md) | Code | Status |
|---|---|---|
| Planet count, candidates, minimum spacing | `galaxy.go` | CONFIRMED, including counts below N (UG22–UG24); seed-dependent ranges BINARY-ONLY |
| Clumping | `galaxy.go` | CONFIRMED |
| Names, environment, artifacts | `galaxy.go` | CONFIRMED |
| Concentrations, max minerals, BBS | `galaxy.go` | CONFIRMED |
| Homeworld placement | `homeworlds.go` | CONFIRMED |
| Starting tech | `players.go` | CONFIRMED |
| Homeworld setup, BBS, AR (spends RD-7), computer players | `players.go` | CONFIRMED |
| Built-in computer races | `races/builtin.go`, `ComputerPlayer` | CONFIRMED (AI.md, 23 of 24; SS harder BINARY-ONLY) |
| Shared starting minerals | `players.go` | LEGACY BUG, `Legacy.SharedHomeworldMinerals` |
| Leftover-point spends | `players.go` | CONFIRMED |
| Starbase designs and loadouts | `designs.go` | CONFIRMED |
| Starting ships and part upgrades | `designs.go` | CONFIRMED |
| Starting designs recorded in `Game.DesignSlots`: starbase slots 0/1, each new ship design the next ship slot | `designs.go` | CONFIRMED (UNIVERSE.md "Starbases", "Starting ships"; UG01..UG21); wired for the engine 2026-10-08, `TestConfirmedStartingDesignSlots` |
| Second planet (PP, IT) | `players.go` | CONFIRMED; redraw fallback LEGACY BUG CONFIRMED (UG29, UG30), `Legacy.SecondPlanetFallback` |
| Relations, research, queues | `players.go` | MEASURED |
| Wormholes | `wormholes.go` | creation and placement badness CONFIRMED |
| Expert +10% before the BBS factor, each truncating, then the second-planet split | `players.go` | MEASURED (UG03 player 2: 736/368; UG21 player 9: 768); not in UNIVERSE.md yet |
| Game options carried into `engine.Game`: random events, size, public scores | `newgame.go` | used by the turn (KERNEL.md "Game options during a turn") |
| Stored victory conditions: a disabled condition's value stored as 0 | `newgame.go` `storedVictory` | MEASURED (UG01-E..UG30-E); not in UNIVERSE.md or KERNEL.md yet |

## UG vectors

`newgame/vectors_test.go` runs the 30 UG vectors from
`engine/testdata/vectors/ug` through `Generate` over 8 seeds (the
engine's parity harness has no new-game path):

- player expectations (starting tech, ship design count) are exact;
- the samples are checked for what the rules decide:
  - planet counts at or below N, reaching N when the original did;
  - wormhole counts in OBJECTS.md's range per size, none with random
    events off (the vectors count wormhole ends, two per pair);
  - homeworld and second-planet population and installations, and fleet
    counts;
  - the stored victory conditions.
- A computer player with level 0 uses the level in `computer.drawn`
  (UG05, UG10..UG15 record it). A vector without it would get a
  fallback: the test runs that player at easy and, only if its start
  differs, tries each level. Levels below expert give the same start, so
  the fallback cannot tell easy, standard and harder apart. No current
  vector needs the fallback.

The same test reads the RD/RW race-creation vectors
(`engine/testdata/vectors/rw`, stars-elegy #111) and fails if either
directory is empty. Each player's race as created (case `-R`) is
compared exactly with `Generate`'s. A raw 255 is read as the stored
immune marker in each field, so a file holding it in the low alone is
repaired at creation (RACES.md "Repairs", RW08). A Random race takes the
race the vector records.

All 291 player expectations (124 UG, 167 RD/RW) and all sample checks
(planet counts 38, starting planets 211, victory 38, wormholes 38) pass.

## LEGACY BUG switches

- `Legacy.SharedHomeworldMinerals` (on in the `elegy` ruleset): every homeworld gets one shared
  surface draw and planet 0's concentrations floored at 30. Off: each
  homeworld uses its own concentrations (floored at 30) and its own draw.
- `Legacy.SecondPlanetFallback` (on in the `elegy` ruleset): a second planet whose 100 redraws were
  all used takes the homeworld's environment. Off: it keeps the last
  redraw.

## Elegy choices (no observable rule to follow)

Marked `ELEGY CHOICE` in the code:

- the generator and every draw order;
- candidate coordinates read as inclusive 1010..1010+W−20; the sort by `x`
  before the spacing pass is stable (the original's is not);
- ties for the clumping neighbour go to the earliest planet; name indexes
  wrap from 998 to 0;
- players are assigned to homeworlds by a Fisher–Yates shuffle; second
  planets are picked in player order, after every homeworld is owned;
- which hull slot holds each starting ship part, and the design names;
- the ARM Midget Miners are two one-ship fleets; fleet ids run across the
  whole game in creation order;
- a wormhole end's class is drawn before its position.

## Placeholders (spec gaps)

Marked `PLACEHOLDER` in the code:

1. **Relations with no human player**: not run in the oracle; Elegy leaves
   every player neutral.
2. **Names**: planet name texts and the 24 computer-player names are
   original game data; `PlanetName` and "Computer N" are placeholders.

Computer players get their built-in race from `ComputerPlayer(type,
level)` (AI.md "Built-in races"). One point is open:

- **ASSUMPTION B1**: AI.md does not list a built-in race's leftover spend
  (UNIVERSE.md shows computer players with the minerals and the
  concentrations spend). `races.BuiltIn` leaves it at surface minerals for
  the caller to set; computer players always use L = 50.

Human races are scored, repaired or replaced and Random races generated
by `races/` (see [RACES-STATUS.md](RACES-STATUS.md)).

## Open items (lane handoff, 2026-10-08)

The new-game generation lane (`newgame/`, `races/`, `objects/` except
`engine_adapter.go`, `terraform/`) is closed. Each item below says where
it lives and what would settle it.

Vector corpus:

1. **RD/RW vectors** are in the corpus and run in `TestUGVectors`.
2. **Drawn computer levels**: `Generate` never draws a random computer
   type or level: the caller passes a type 1–6 and a level
   (`ComputerPlayer`). The test uses the vector's recorded race and
   `computer.drawn` level.

Spec gaps to send to stars-elegy (Elegy follows the vectors):

3. Expert +10% before the BBS factor (MEASURED UG03, UG21): not in
   UNIVERSE.md.
4. Stored victory conditions, a disabled condition stored as 0 (MEASURED
   UG01-E..UG30-E): not in UNIVERSE.md or KERNEL.md.
5. **ASSUMPTION B1**: built-in race leftover spend (here and
   RACES-STATUS.md).
6. **PLACEHOLDER** relations with no human player, planet names and
   computer-player names (above).

Labelled Elegy choices and assumptions in the lane's other packages
(each documented where it lives):

7. Generation: every `ELEGY CHOICE` above.
8. Objects ([OBJECTS-STATUS.md](OBJECTS-STATUS.md)): minefields O6, O9,
   O10; packets P6 (a destination naming no planet is no destination);
   stargates G4 (`D > s` destroys every ship of the design); salvage S1;
   visibility V1–V3. P5 is answered.
9. Terraforming ([TERRAFORM-STATUS.md](TERRAFORM-STATUS.md)): T1–T3, R1.
   P1–P3 are answered.
10. Spec question still open: OBJECTS.md's hostile-adjuster example
    "137 against 67" fits radiation (67), not temperature, for a 50
    (15–85) habitat. The `terraform/` tests assert all three scores.

Unimplemented behaviour:

11. Salvage as a space object: it stays in `engine.Game.Salvage`.
    Salvage numbering, decay and loading exist in `objects/`; the spec
    gaps on owner, merging and pickup are with the objects research lane.
12. Not wired by the kernel yet: the terraforming production items, the
    remote-mining task, Orbital Adjusters and `Impact.DiscloseDesign`
    (TERRAFORM-STATUS.md "Turn wiring"); the yearly race check
    (RACES-STATUS.md "Not wired yet").
13. Generation itself, under "Not modelled" below.

## Not modelled

Unseeded new-game wizard games (computer-player counts by size and
difficulty), a random computer type or level in a definition file, the
tutorial galaxy, logos, and the original's
random stream.
