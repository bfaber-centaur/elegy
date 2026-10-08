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
| Salvage: a transport task loads minerals from it, capped by what it holds, and unloads minerals into it up to its stored size (ASSUMPTION T5–T7) | `loadSalvage`, `unloadSalvage`, `SpaceObjects.SalvageLoad`, `SalvageRoom` | BINARY-ONLY (OBJECTS.md "Salvage", "Loading") | `TestLoadFromSalvage`, `TestUnloadIntoSalvage`; through the adapter `TestSalvageLoadInTurn`, `TestSalvageUnloadInTurn`, `TestSalvageFirstAtPosition` |
| Ground combat | `resolveDrops` | CONFIRMED (T-21..T-25); WM, IS, AR BINARY-ONLY | `TestConfirmedGroundCombat`, `TestPredictionGroundStrengthTraits` |
| Drop resolution order | `resolveQueue` | BINARY-ONLY (#34) | none yet beyond one planet at a time |
| Several players dropping | `dropWinner` (ruleset switch `Legacy.DropScan`) | LEGACY BUG, CONFIRMED (T-32) | `TestConfirmedSeveralPlayersDrop` |
| Colonization (requirements, whole fleet, ⌊3C/4⌋ minerals) | `colonize` | CONFIRMED (T-1, T-30, T-31); requirements BINARY-ONLY | `TestConfirmedColonyMinerals`, `TestConfirmedTakeoverTiming` |
| Timing (in orbit before growth, arriving after; bombing before arrivals) | `GenerateTurn` | CONFIRMED (TK-001..003) | `TestConfirmedTakeoverTiming` |
| Colonize is tried once: any failure clears the task and keeps the cargo | `colonize` | CONFIRMED (TK-113, #37) | `TestConfirmedColonizeTriedOnce` |
| Load all and load exactly at the owner's own planet, in the load passes; load exactly keeps its shortfall, load all clears (ASSUMPTION T4) | `load`, `loadPass` | CONFIRMED (TK-301, TK-302, TK-201 G; FO-01-A..D and K vectors); MEASURED (WP-1 vector) | `TestConfirmedLoadAmounts` |
| Fill to and wait for v%: up to v% of the fleet's cargo capacity; fill clears after its load, wait keeps the task until met (ASSUMPTION T8, T9); also from salvage (T5) | `Transport.loadWant`, `Transport.loaded`, `load`, `loadSalvage` | CONFIRMED (TAKEOVER.md "Unload and load amounts"; FO-01-F, I, J vectors) | `TestFillAndWaitFor`, `TestFillAndWaitForFromSalvage`, `TestValidPercentTransport` |
| Set amount to v (v − C) and set waypoint to v (A − v): unload in the unload phase, load in the load pass; a short load keeps the action (ASSUMPTION T11); A away from a planet is the salvage's, else 0 (T5, T12) | `Transport.unloadWant`, `targetHolds`, `unload`, `Transport.loadWant`, `Transport.loaded` | CONFIRMED (TAKEOVER.md "Unload and load amounts"; FO-01-G, H and TK-114-U4, U5 vectors) | `TestSetAmountAndWaypointLoad`, `TestSetAmountAndWaypointUnload`, `TestSetWaypointAwayFromPlanet`, `TestSetAmountAndWaypointFromSalvage` |
| A fleet whose transport task is still current after the load pass (an unmet load) does not move; this includes a load exactly shortfall, under KERNEL.md's general rule (KB-4A T1 is a held "wait for" and T2 a satisfied unload that moved; no vector holds a load exactly shortfall directly), and the waiting loads of T4 and T6 (ASSUMPTION T10) | `moving` | CONFIRMED (KERNEL.md "Other movement rules", KB-4A T1, T2; FO-01-I vector) | `TestTransportTaskHoldsFleet` |
| Capture: what a planet keeps | `emptyPlanet` | CONFIRMED (T-21, T-26, T-27; CA environment TK-116) | `TestConfirmedGroundCombat`, `TestPredictionEmptiedPlanet` |
| Capture tech attempt (old owner's levels, shared "gained" mark) | `resolveDrops`, `techAttempt` | BINARY-ONLY (#34) | `TestPredictionCaptureTech` |
| New colony: default queue (AR, CA skips), default leftover setting | `newColony` | BINARY-ONLY | `TestPredictionEmptiedPlanet` |
| New colony with an empty default queue: no queue, all resources to research | `newColony` | MEASURED (WU-CAP) | `TestMeasuredEmptyDefaultQueueNoQueue` |
| Scrap: before movement only; 4C/5, C/3, 9C/10, 9C/20 by starbase and the planet owner's Ultimate Recycling, C/3 as fresh salvage in deep space; cargo minerals on top; colonists to the owner's own planet; the planet owner's tech attempt at a starbase, with the scrapped ships' Trader part counts as chances (MEASURED TK-305; ASSUMPTION K11: once per ship, capped at 25); the Ultimate Recycling production bonus (ASSUMPTION K9: the planet owner's trait) | `scrap`, `recycledResources` | CONFIRMED (T-34, TK-203, KB-2A) | `TestConfirmedScrapMinerals`, `TestConfirmedScrapRecycledResources`, `TestScrapOnlyBeforeMovement`, `TestScrapTraderChance`, TK-111-S1..S10, M1..M4, TK-203 vectors |
| New Alternate Reality colony: a starbase of the owner's first starbase design; "first" is the lowest occupied starbase slot (ASSUMPTION T3); with no starbase design, none (UNRESOLVED, the spec is silent) | `newColony`, `firstStarbaseDesign` | CONFIRMED (T-26, T-33) | `TestConfirmedAlternateRealityColonyStarbase`, `TestAlternateRealityColonyGeneratesYears` |
| Starvation and AR starbase loss empty the planet | `GenerateTurn`, combat `finish` | as Capture | `TestPredictionEmptiedPlanet` |

## Implementation assumptions

Elegy's own choices where TAKEOVER.md is silent. They are **not**
established Stars! behavior; each is marked `ASSUMPTION Tn` in the code.

| Id | What Elegy does | Why |
|---|---|---|
| T1 | Away from a planet and any salvage, a fleet that shares its position with another fleet keeps the cargo it would unload (the action still clears). Anywhere else is deep space. | TAKEOVER.md defines deep space by the waypoint's target, which Elegy's task does not record, and unloads to fleets are not modelled. |
| T5 | Away from a planet, a transport task loads from and unloads into the first salvage object (object order) at the fleet's position; with no space objects there is none. | OBJECTS.md "Salvage", "Loading" names a fleet at the salvage's position; the original acts on the waypoint's target, which Elegy's task does not record. |
| T6 | A colonist load action at salvage waits, as a load away from the owner's planet does (T4). | OBJECTS.md says colonists cannot be loaded from salvage, not what becomes of the action. |
| T7 | Minerals unloaded into salvage beyond its room stay aboard (the action clears); colonists unloaded there are refused, as in deep space. | OBJECTS.md says an unload is accepted only up to the stored size, not where the rest goes, and says nothing of colonists. |
| T8 | A "fill to" or "wait for" v% target is ⌊v·capacity/100⌋ of that cargo type, counting only that type's cargo aboard, not the whole hold. | TAKEOVER.md says "up to v% of capacity"; FO-01 loaded into empty holds, where per-type and whole-hold targets agree, and its 50% of 210 is exact. |
| T9 | "Wait for" is met when the fleet holds at least its target of that type; then it clears like any satisfied load. | TAKEOVER.md says the fleet "waits until met" without defining met; FO-01-I (100% with 100 of 210 available) stayed unmet. |
| T10 | A load that waits under T4 (at a planet the fleet's owner does not own) or T6 (colonists at salvage) keeps the task current, so the fleet holds its place until it gets new orders. | A consequence of T4 and T6 under KERNEL.md's rule that a current transport task does not move; no measurement covers these waits. |
| T11 | "Set amount to" and "set waypoint to" that load keep the action until satisfied (cargo ≥ v, or the source ≤ v), so a short load holds the fleet, as "load exactly" does; once they unload they clear. | TAKEOVER.md says load actions persist until satisfied and unload actions clear, but not which these are; every vector case was satisfied in one pass. |
| T12 | Away from a planet, "set waypoint to" takes A from the salvage object at the fleet's position (T5), and in deep space A is 0, so the shortfall is unloaded and destroyed. | TAKEOVER.md defines A only for a planet. |

The scan for several drops starts at the first dropping player, so a lone
attacker of strength 0 (Alternate Reality colonists, `k = 0`) lands; the
spec does not give the scan's start value. The code says so at
`dropWinner`.

## Not modelled

- every fuel action of a transport task. "Load all", "load exactly",
  "fill to", "wait for", "set amount to", "set waypoint to", fleet
  transfers, remote
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
