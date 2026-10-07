# Takeover status

Bombing, transport unloads, colonist drops, ground combat, colonization
and capture in `engine/takeover.go` follow the public specification
stars-elegy `docs/TAKEOVER.md` on `main` at `bd8f063`, which includes the
implementer answers merged in PR #34 (order inside a phase, unload
amounts and foreign minerals, capture tech, colonize retries, the mines'
loss clamp), and the deep-space unload rule of PR #37, and the TK-001..TK-007 cases in
`experiments/tk/README.md`. Bomb, colonizer and defense values come from
the component table (`docs/COMPONENTS-STATUS.md`). Nothing else was used.

## Where it runs

`GenerateTurn` follows TAKEOVER.md "Where each task happens":

1. Before movement: each fleet's waypoint-0 task in fleet order (unloads,
   colonize), then the queued drops, then colonize retries, whose drops
   wait for the after-movement resolution.
2. Movement, production and growth. A planet whose population dies out is
   emptied.
3. The after-movement phase records ownership, then battles, then bombing
   in fleet order, then the waypoint-0 tasks of the fleets' new locations,
   then the drops (the before-movement retries first), then the second
   research level-up check, then colonize retries (LEGACY BUG: their
   colonists are lost).

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
| Several players dropping | `dropWinner` (switch `legacyDropScan`) | LEGACY BUG, CONFIRMED (T-32) | `TestConfirmedSeveralPlayersDrop` |
| Colonization (requirements, whole fleet, ⌊3C/4⌋ minerals) | `colonize` | CONFIRMED (T-1, T-30, T-31); requirements BINARY-ONLY | `TestConfirmedColonyMinerals`, `TestConfirmedTakeoverTiming` |
| Timing (in orbit before growth, arriving after; bombing before arrivals) | `GenerateTurn` | CONFIRMED (TK-001..003) | `TestConfirmedTakeoverTiming` |
| Colonize retries | `loadPass` (switch `legacyRetryLost`) | BINARY-ONLY; after movement LEGACY BUG | `TestPredictionColonizeRetries` |
| Capture: what a planet keeps | `emptyPlanet` | CONFIRMED (T-21, T-26, T-27); CA environment BINARY-ONLY | `TestConfirmedGroundCombat`, `TestPredictionEmptiedPlanet` |
| Capture tech attempt (old owner's levels, shared "gained" mark) | `resolveDrops`, `techAttempt` | BINARY-ONLY (#34) | `TestPredictionCaptureTech` |
| New colony: default queue (AR, CA skips), default leftover setting | `newColony` | BINARY-ONLY | `TestPredictionEmptiedPlanet` |
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

Elegy has none of these yet:

- load actions ("load all", "fill to", "wait for", "set amount to", "set
  waypoint to"), merges, fleet transfers, scrap, remote mining, mine
  laying and cargo given to other players' fleets;
- colonists given by manual cargo transfers in the orders, which would
  start the before-movement drop queue;
- ancient artifacts (no random events);
- the starbase an Alternate Reality colony gets from its owner's first
  starbase design (Elegy's designs have no owner), so a new AR colony has
  no starbase and the next year is refused (`ZeroMaxPopulationError`);
- the design check at generation that drops parts above the owner's tech
  (Elegy's designs have no owner; Elegy keeps and uses every part, which
  is the original's behavior for race-restricted and Mystery Trader
  parts);
- the old owner's message beyond `EventPlanetEmptied`.
