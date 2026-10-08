# Space objects: implementation status

`objects/` implements the space objects of stars-elegy `docs/OBJECTS.md`
(as of stars-elegy `main` at `b022bf3`) and its Mystery Trader appearance
in `KERNEL.md`, with the part statistics of
`COMPONENTS.md` (the component table the engine embeds). Nothing here
comes from the private archaeology repositories.

The package reads the engine's exported state and exposes step and query
functions. It does not change `engine/`, and `engine.GenerateTurn` does
not call it yet: the turn engine owns the year's order (OBJECTS.md "Turn
placement") and wires each step in. Object families land one at a time:
minefields, wormholes, then the Mystery Trader.

## Minefields

| Rule (OBJECTS.md) | Function | Status |
|---|---|---|
| Size and inside test | `Minefield.Contains` | CONFIRMED (OB-002, OB-018-C) |
| Decay, incl. SD and detonation | `DecayLoss`, `Space.Decay` | CONFIRMED (OB-002, OB-014-A, OB-015, OB-016, MF-4) |
| Lay amounts, hull multiplier, MCM | `LayAmounts` | CONFIRMED (OB-002, OB-024) |
| Merging, merge cap, laying order | `Space.Lay` | CONFIRMED (OB-002-G, MF-10, MF-12) |
| SD half lay while moving | `Layer.Half` | CONFIRMED (OB-014-C) |
| 512-field limit | `Space.Lay` | MEASURED (MF-11); the 511 case is LEGACY BUG `LegacyFieldLimit511` (off, Elegy's chosen rule) |
| 4050-object limit | `Space.Lay` | BINARY-ONLY |
| Sweep rating and amount | `SweepRating`, `Space.Sweep` | CONFIRMED (OB-001, OB-007, OB-008, OB-010-S); ratings agree with the table's mines-swept column |
| Effective warp | `EffectiveWarp` | CONFIRMED (OB-010, MF-3) |
| Stop checks along a step | `CheckStep` | CONFIRMED as rates (MF-1, MF-5, MF-6); safe-warp bonuses for SS and SD BINARY-ONLY |
| Hit damage | `MineDamage` | CONFIRMED (MF-9, MF-8, OB-010-S, OB-002-M) |
| Mines lost, paying field | `MinesLost`, `PayingField`, `ApplyHit` | CONFIRMED (OB-010-S, OB-024, MF-4) |
| Salvage from destroyed ships; empty fleet `rand(10)` | `ApplyHit` | MEASURED (OB-024); the empty-fleet drop is a LEGACY BUG candidate, `LegacyEmptyFleetSalvage` (on) |
| SD disclosure on hits and detonations | `Hit.Disclosed`, `Detonation.Disclosed` | CONFIRMED for detonations (MF-7) |
| Detonation | `Space.Detonate` | CONFIRMED (OB-002-M, MF-7, MF-8) |

Tests follow the engine's naming: `TestConfirmed*` use the OB and MF
vectors OBJECTS.md and PARITY.md give; `TestPrediction*` cover the rest.
The MF runs' "Tank" is a Destroyer with two Superlatanium (3200 armor).

### Assumptions (spec gaps)

Marked `ASSUMPTION On` in the code and sent to the spec owner:

1. **O1** A Space Demolition half lay halves each kind's amount,
   truncating.
2. **O2** Ties for the nearest own field when merging go to the first
   field in object order.
3. **O3** A new field takes the owner's lowest unused number.
4. **O4** A step's stretch inside a field runs from the entry distance
   rounded up to the exit distance rounded down, one draw per whole ly from
   entry up to (not including) exit; a stop `k` ly into it is `entry + k`
   from the step's start.
5. **O5** A kind whose safe warp is at or above the effective warp makes
   no draws; stretches with equal entries go standard, heavy, speed bump.
6. **O6** Damage bookkeeping: groups are a fleet's stacks of one design;
   shields absorb ships × the design's shield; existing damage is
   `units·armor/500` per damaged ship, `damaged = ⌈pct·ships/100⌉`; new
   damage is stored as pct 100 and `units = ⌊average·500/armor⌋`; a design
   is destroyed when its average is greater than its armor; exempt layers
   are left out of the fleet minimum.
7. **O7** A destroyed group's share of the fleet's cargo is proportional
   to its cargo capacity (to ship count when the fleet has none).
8. **O8** A fleet inside several detonating fields is hit by the first in
   object order.

### What the turn engine needs to call (minefields)

In OBJECTS.md "Turn placement" order:

- movement: `CheckStep` for each movement step of a fleet moving at warp
  1–10, not through a stargate and not already at its waypoint, then
  `ApplyHit` at the stop point (the fleet stops, gets no ram-scoop fuel,
  and has spent its whole leg's fuel);
- after movement: `Space.Detonate`, then `Space.Decay`;
- waypoint tasks after battles: `Space.Lay` for the fleets laying this
  year (a "lay mines" task with its duration, and the SD half lay);
- then `Space.Sweep`;
- fleets left with no stacks are destroyed; salvage objects and messages
  come from the returned records.

The engine has no minefield list, "lay mines" task or minefield
visibility yet; those belong to the turn engine and scanning code.

## Wormholes

| Rule (OBJECTS.md) | Function | Status |
|---|---|---|
| Pair counts, classes, placement badness at creation | `WormholePairs`, `Surroundings.Badness`, `Surroundings.Place`; used by `newgame` | CONFIRMED (OB-006, UG01–UG21) |
| Jump chance | `JumpChance` | MEASURED (OB-025) |
| Stability name | `StabilityName` | BINARY-ONLY |
| Jump: years reset, knowledge cleared, uniform placement | `Space.MoveWormholes` | MEASURED (OB-005-B), CONFIRMED (OB-025) |
| Jiggle: ±12 per axis, never the old position, class kept | `Space.MoveWormholes` | CONFIRMED (OB-005, OB-017, OB-025) |
| Transit to the partner's start-of-year position, knowledge | `Space.Transit` | CONFIRMED (OB-005 C, D, WT-001) |
| Destination knowledge, shown while the partner is seen | `WormholeEnd.DestKnown`, `Space.Destination` | CONFIRMED (WT-001, WT-004); report BINARY-ONLY |

`newgame` now places wormholes through `objects`; its tests and the
creation results are unchanged.

OBJECTS.md now answers the two wormhole gaps (W1, W2), so they are no
longer assumptions; both answers are BINARY-ONLY:

- Outside the galaxy is a coordinate below 1000 or above 1000 + W; a
  coordinate equal to 1000 + W is inside (`Surroundings.Badness`).
- Jump and jiggle tries use the creation badness, with fleets, minefield
  centres and Traders as objects; each try draws x before y. A jiggle try
  on the old position is skipped without a score but uses up one of the
  100 tries. If every scored try is rejected, the end moves to the first
  of them; it does not stay put (`Surroundings.Place`).

### What the turn engine needs to call (wormholes)

- Fleet movement: a fleet whose reached waypoint targets a wormhole end
  (not a plain position on it, and not one it passes over) calls
  `Space.Transit`; fleets following it lose it.
- A waypoint aimed at an end keeps following it only while its owner knew
  the end (`WormholeEnd.KnownBy`) at the start of the year; otherwise,
  after the wormhole's first move, it becomes a plain position at the
  end's old position (CONFIRMED OB-025-F, OB-027).
- After fleets move (OBJECTS.md "Turn placement" step 7):
  `Space.MoveWormholes`.
- Scanning marks sightings with `WormholeEnd.MarkKnown` and shows
  destinations with `Space.Destination`.

## Mystery Trader

| Rule | Function | Status |
|---|---|---|
| Appearance: chance, warp, start, destination, item, in draw order | `Space.Appear` | CONFIRMED (KERNEL.md, KX-004 S6–S10) |
| Part reroll and late-year conversion at appearance | none yet | PLACEHOLDER T1 (spec does not name the parts or limits) |
| Movement: warp rise, new destination, packet-style step, arrival, leaving | `Space.MoveTraders`, `StepToward` | CONFIRMED in part (OB-023, OB-026, OB-031); the rest BINARY-ONLY |
| Trade threshold, one trade per player per Trader, fleet consumed | `Space.Encounters` | CONFIRMED (OB-004, OB-023, OB-030-T, WT-002..WT-004) |
| Part reward; parts owned per player | `TraderParts`, `TraderParts.Items` | CONFIRMED (WT-003 A) |
| Research reward: L from cargo and tech sum, field choice | `researchReward` | CONFIRMED (WT-002 A, B); field odds MEASURED |
| Every field at 26: nothing with 1/5, else a part or a ship | `Space.Encounters` | CONFIRMED (WT-004 C) |
| 25th redraw finds an unowned part but gives a ship | `LegacyTraderLastRedraw` (on) | LEGACY BUG, BINARY-ONLY |
| Ship gifts: designs, counts, design slot, new fleet | `shipGift`, `giftDesign` | CONFIRMED (WT-003 B, WT-004, OB-026); counts after year index 100 and computer players BINARY-ONLY |
| Computer players' planets | `Space.PlanetTrades` | CONFIRMED in part (TP-001, TP-002); Harder 3,500 kT CONFIRMED (O-53); 100 ly edge and the Standard/Easy exclusion BINARY-ONLY |

Traders sit in `Space.Traders` and count toward the object limit and the
wormhole placement objects.

### Assumptions (spec gaps)

1. **T1 (PLACEHOLDER)** No part reroll and no late-year conversion of a
   part to research at appearance until the spec names the parts and the
   year limits.
2. **T2** A Trader's step moves each coordinate by `dx·move/dist`, rounded
   half away from zero, with `dist` the exact distance.
3. **T3** The new-destination draw (1/3) is made only in a year the warp
   rose.
4. **T4** A new destination is on the edge opposite the Trader's current
   edge, on the same axis, with its free coordinate drawn as at
   appearance. A Trader not on an edge keeps its destination's edge.
5. **T5** Each research level draws `rand(4)`, then `rand(6)` only on the
   3/4 branch.
6. **T6** The ship gift designs fill the hull's slots in the order the
   spec lists the parts.
7. **T7** A matching gift design has the same name, hull and parts. The
   draws are design kind, then the Scout/Probe coin, then the counts. The
   new fleet takes the owner's lowest unused fleet number and battle plan
   0.
8. **T8** Planets trade in planet order; a ship item counts as research
   for planets; "within 100 ly" is `d² ≤ 10,000`.

### What the turn engine needs to call (Mystery Trader)

- End of production, after new minerals: `Space.Appear`, then the
  appearance message to every player.
- Before fleets move: `Space.MoveTraders`. Waypoints aimed at a Trader
  follow its new position.
- After battles: `Space.Encounters`, then `Space.PlanetTrades`; messages
  come from the returned records.
- Designs: pass `TraderParts.Items(player)` to `ReadDesign` so players can
  use the parts they own.
- Scanning shows Traders as objects; the engine has no Trader visibility
  yet.
