# Space objects: implementation status

`objects/` implements the space objects of stars-elegy `docs/OBJECTS.md`
(as of stars-elegy `main` at `5469029`), with the part statistics of
`COMPONENTS.md` (the component table the engine embeds). Nothing here
comes from the private archaeology repositories.

The package reads the engine's exported state and exposes step and query
functions. It does not change `engine/`, and `engine.GenerateTurn` does
not call it yet: the turn engine owns the year's order (OBJECTS.md "Turn
placement") and wires each step in. Object families land one at a time;
minefields are first.

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

### What the turn engine needs to call

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
