# New-game generation: implementation status

`newgame/` builds a new game (year 2400) from game settings, following the
public specification in `bfaber-centaur/stars-elegy`: `docs/UNIVERSE.md`,
with `KERNEL.md` "Habitability", `COMPONENTS.md` (the component table the
engine already embeds) and `OBJECTS.md` "Wormholes", as of stars-elegy
`main` at `004b4dc` (PR #43). Nothing here comes from the private
archaeology repositories.

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
  (UG05, UG15), and the measured planet counts.
- `TestPrediction*`: BINARY-ONLY rules: the seed-dependent planet counts
  (Elegy's spacing pass lands in the spec's sampled 899–940 and 930–962
  ranges), the second-planet redraw limit and wormhole placement badness.
- `TestElegyDecision*`: Elegy's own choices: determinism, the generator,
  the shared-minerals switch, and a generated game running through
  `engine.GenerateTurn`.

| Rule (UNIVERSE.md) | Code | Status |
|---|---|---|
| Planet count, candidates, minimum spacing | `galaxy.go` | CONFIRMED; seed-dependent cells BINARY-ONLY |
| Clumping | `galaxy.go` | CONFIRMED |
| Names, environment, artifacts | `galaxy.go` | CONFIRMED |
| Concentrations, max minerals, BBS | `galaxy.go` | CONFIRMED |
| Homeworld placement | `homeworlds.go` | CONFIRMED |
| Starting tech | `players.go` | CONFIRMED |
| Homeworld setup, BBS, AR, computer players | `players.go` | CONFIRMED |
| Shared starting minerals | `players.go` | LEGACY BUG, `legacySharedHomeworldMinerals` |
| Leftover-point spends | `players.go` | CONFIRMED |
| Starbase designs | `designs.go` | CONFIRMED, loadout PLACEHOLDER |
| Starting ships and part upgrades | `designs.go` | CONFIRMED |
| Second planet (PP, IT) | `players.go` | CONFIRMED; redraw fallback LEGACY BUG (BINARY-ONLY), `legacySecondPlanetFallback` |
| Relations | `players.go` | MEASURED for one human |
| Wormholes | `wormholes.go` | creation CONFIRMED, placement badness BINARY-ONLY |

## LEGACY BUG switches

- `legacySharedHomeworldMinerals` (on): every homeworld gets one shared
  surface draw and planet 0's concentrations floored at 30. Off: each
  homeworld uses its own concentrations (floored at 30) and its own draw.
- `legacySecondPlanetFallback` (on): a second planet whose 100 redraws were
  all used takes the homeworld's environment. Off: it keeps the last
  redraw.

## Elegy choices (no observable rule to follow)

Marked `ELEGY CHOICE` in the code:

- the generator and every draw order;
- candidate coordinates read as inclusive 1010..1010+W−20; the spacing
  pass keeps candidates in order and drops any within 12 ly of one kept;
- ties for "nearest" go to the earliest planet or pick; name indexes wrap
  from 998 to 0; the homeworld list-order fallback scans from the last
  pick and wraps;
- players are assigned to homeworlds by a Fisher–Yates shuffle; second
  planets are picked in player order, after every homeworld is owned;
- BBS before the expert +10%, each truncating, then the second-planet
  split; the surface draw reads planet 0's concentration before the floor;
- which hull slot holds each starting part, and the design names;
- the ARM Midget Miners are two one-ship fleets; fleet ids run across the
  whole game in creation order;
- wormhole badness terms are scored by band and summed; "within 10 ly of
  an edge" is less than 10 ly; an end's class is drawn before its position.

## Placeholders (spec gaps, sent to the spec author)

Marked `PLACEHOLDER` in the code:

1. **Space Station design 0 loadout.** The spec says "armed with lasers
   and Mole-skin shields" without counts or slots. Elegy uses 16 Lasers in
   each of beam slots 1 and 3 and 16 Mole-skin Shields in slot 2. AR's
   design 1 uses the same loadout.
2. **Maximum minerals on homeworlds.** UNIVERSE.md says every
   concentration is 100; PARITY.md "Options" says "except the
   homeworlds". Elegy follows the shared-minerals rule, which gives 100.
3. **Installation spends on AR.** AR homeworlds have no installations;
   the spec does not say what a mines, factories or defenses spend does
   for AR. Elegy adds none.
4. **Relations** with zero or two or more humans: the default value is
   unstated; Elegy leaves the engine's neutral default.
5. **Research** for computer players (Elegy 15%, the human value), and the
   starting research field and next-field choice (Elegy: energy, same
   field).
6. **Production queue**: the starting queue is not specified; homeworlds
   and second planets start with an empty queue.
7. **Names**: planet name texts and the 24 computer-player names are
   original game data; `PlanetName` and "Computer N" are placeholders.
8. **Computer races** and **leftover points**: the original's computer
   races and the race-design point system are not specified; the caller
   supplies the race and the points (computers always use 50).

## Not modelled

Unseeded new-game wizard games (computer-player counts by size and
difficulty), the tutorial galaxy, random races, logos, and the original's
random stream.
