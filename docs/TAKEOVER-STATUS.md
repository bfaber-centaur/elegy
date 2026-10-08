# Takeover status

Bombing, transport unloads, colonist drops, ground combat, colonization
and capture in `engine/takeover.go` follow the public specification
stars-elegy `docs/TAKEOVER.md` on `main` at `8ddc64a`, which includes the
implementer answers merged in PR #34 (order inside a phase, unload
amounts and foreign minerals, capture tech, the mines' loss clamp) and
the corrections merged in PR #37 (deep-space unloads; colonize is tried
once, TK-113; the Claim Adjuster revert, TK-116), and the TK cases in
`experiments/tk/README.md`. Bomb, colonizer and defense values come from
the component table (`docs/COMPONENTS-STATUS.md`). Nothing else was used.

## Where it runs

`GenerateTurn` follows TAKEOVER.md "Where each task happens":

1. Before movement: each fleet's waypoint-0 task in fleet order (unloads,
   colonize), then the queued drops.
2. Movement, production and growth. A planet whose population dies out is
   emptied.
3. The after-movement phase records ownership, then battles, then bombing
   in fleet order, then the waypoint-0 tasks of the fleets' new locations,
   then the drops, then the second research level-up check.

A fleet's current task is `Fleet.Task`. A fleet that moves leaves it and
takes up `Waypoint.Task` of the waypoint it arrives at.

## Rules and tests

| Rule (TAKEOVER.md section) | Code | Status | Test |
|---|---|---|---|
| Who bombs (owned, no starbase, attack-who covers the owner) | `bombsOwner`, `bombing` | CONFIRMED (T-3, T-19) | `TestConfirmedWhoBombs` |
| A player's fleets bomb as one | `bomberPass` | CONFIRMED (T-20, TK-005) | `TestConfirmedFleetsBombAsOne` |
| Bombing order: fleet order, one pass per player and planet | `bombing` | BINARY-ONLY (#34) | `TestPredictionBombingOrder` |
| Bomb totals (normal add, smart multiply, Retro, MCM, OCM) | `bombPass` | CONFIRMED (T-10..T-18) | `TestConfirmedBombingVectors`, `TestConfirmedSmartThenNormal` |
| Defenses: best type at current tech, `n = min(installed, operable)`, half for smart bombs and installations | `survival`, `bestDefense` | CONFIRMED (T-6..T-9) | `TestConfirmedBombingVectors` |
| Applying the pass (installations, population, draws) | `bombPlanet` | CONFIRMED; random roundings MEASURED; the mines' clamp BINARY-ONLY (#34) | `TestConfirmedBombingVectors`, `TestPredictionMineLossClamp` |
| Retro bombs | `bombPlanet` | CONFIRMED (T-16) | `TestConfirmedRetroBombs` |
| Unloading colonists on another player's planet (refusals, relation unchecked) | `dropColonists` | CONFIRMED (T-4, T-28, T-29); AR BINARY-ONLY | `TestConfirmedTakeoverTiming`, `TestConfirmedFriendInvaded`, `TestPredictionGroundStrengthTraits` |
| "At the start of this phase" ownership | `phaseStart` | BINARY-ONLY (#34) | `TestPredictionPhaseStartOwnership` |
| Unload amounts; own-planet unloads before or after growth; minerals on any planet | `unload` | BINARY-ONLY (#34) | `TestPredictionUnloadAmounts`, `TestPredictionForeignMinerals` |
| Deep space: minerals destroyed (no salvage), colonists refused | `unload`, `deepSpace` | BINARY-ONLY (#37) | `TestPredictionDeepSpaceUnload` |
| Ground combat | `resolveDrops` | CONFIRMED (T-21..T-25); WM, IS, AR BINARY-ONLY | `TestConfirmedGroundCombat`, `TestPredictionGroundStrengthTraits` |
| Drop resolution order | `resolveQueue` | BINARY-ONLY (#34) | none yet beyond one planet at a time |
| Several players dropping | `dropWinner` (ruleset switch `Legacy.DropScan`) | LEGACY BUG, CONFIRMED (T-32) | `TestConfirmedSeveralPlayersDrop` |
| Colonization (requirements, whole fleet, ⌊3C/4⌋ minerals) | `colonize` | CONFIRMED (T-1, T-30, T-31); requirements BINARY-ONLY | `TestConfirmedColonyMinerals`, `TestConfirmedTakeoverTiming` |
| Timing (in orbit before growth, arriving after; bombing before arrivals) | `GenerateTurn` | CONFIRMED (TK-001..003) | `TestConfirmedTakeoverTiming` |
| Colonize is tried once: any failure clears the task and keeps the cargo | `colonize` | CONFIRMED (TK-113, #37) | `TestConfirmedColonizeTriedOnce` |
| Load all and load exactly at the owner's own planet, in the load passes; load exactly keeps its shortfall, load all clears (ASSUMPTION T4) | `load`, `loadPass` | CONFIRMED (TK-301, TK-302, TK-201 G; FO-01-A..D, K and WP-1 vectors) | `TestConfirmedLoadAmounts` |
| Capture: what a planet keeps | `emptyPlanet` | CONFIRMED (T-21, T-26, T-27; CA environment TK-116) | `TestConfirmedGroundCombat`, `TestPredictionEmptiedPlanet` |
| Capture tech attempt (old owner's levels, shared "gained" mark) | `resolveDrops`, `techAttempt` | BINARY-ONLY (#34) | `TestPredictionCaptureTech` |
| New colony: default queue (AR, CA skips), default leftover setting | `newColony` | BINARY-ONLY | `TestPredictionEmptiedPlanet` |
| New colony with an empty default queue: no queue, all resources to research | `newColony` | MEASURED (WU-CAP) | `TestMeasuredEmptyDefaultQueueNoQueue` |
| Scrap: before movement only; 4C/5, C/3, 9C/10, 9C/20 by starbase and the planet owner's Ultimate Recycling, C/3 as fresh salvage in deep space; cargo minerals on top; colonists to the owner's own planet; the planet owner's tech attempt at a starbase; the Ultimate Recycling production bonus (ASSUMPTION K9: the planet owner's trait) | `scrap`, `recycledResources` | CONFIRMED (T-34, TK-203, KB-2A) | `TestConfirmedScrapMinerals`, `TestConfirmedScrapRecycledResources`, `TestScrapOnlyBeforeMovement`, TK-111-S1..S10, M1..M4, TK-203 vectors |
| New Alternate Reality colony: a starbase of the owner's first starbase design; "first" is the lowest occupied starbase slot (ASSUMPTION T3); with no starbase design, none (UNRESOLVED, the spec is silent) | `newColony`, `firstStarbaseDesign` | CONFIRMED (T-26, T-33) | `TestConfirmedAlternateRealityColonyStarbase`, `TestAlternateRealityColonyGeneratesYears` |
| Starvation and AR starbase loss empty the planet | `GenerateTurn`, combat `finish` | as Capture | `TestPredictionEmptiedPlanet` |

## Implementation assumptions

Elegy's own choices where TAKEOVER.md is silent. They are **not**
established Stars! behavior; each is marked `ASSUMPTION Tn` in the code.

| Id | What Elegy does | Why |
|---|---|---|
| T1 | Away from a planet, a fleet that shares its position with another fleet or a salvage object keeps the cargo it would unload (the action still clears). Anywhere else is deep space. | TAKEOVER.md defines deep space by the waypoint's target, which Elegy's task does not record, and unloads to fleets and salvage are not modelled. |

The scan for several drops starts at the first dropping player, so a lone
attacker of strength 0 (Alternate Reality colonists, `k = 0`) lands; the
spec does not give the scan's start value. The code says so at
`dropWinner`.

## Not modelled

- the load actions "fill to", "wait for", "set amount to" and "set
  waypoint to". "Load all", "load exactly", fleet transfers, remote
  mining, mine laying, cargo given to other players' fleets and colonists
  dropped by a manual cargo order are implemented (KERNEL-STATUS.md,
  ORDERS-LAYER-STATUS.md);
- ancient artifacts;
- the design check at generation that drops parts above the owner's tech:
  designs have owners through `Game.DesignSlots`, but `Catalog.ReadDesign`
  runs only when a design order (ORDERS-STATUS.md) or a computer player
  (`ai/designs.go`) makes a design, so a design made any other way keeps
  every part;
- the old owner's message beyond `EventPlanetEmptied`.
