# Combat status

The battle phase in `engine/` follows the public combat specification,
stars-elegy `docs/COMBAT.md` as of stars-elegy `main` at `5a6621a` (PR #19
with the implementer answers merged in #24 and #31, and rounds 4 and 5
reconciled in #38: CB-023..CB-041 and CS-003), the owner cost rule
of `docs/COMPONENTS.md` (merged in #30), the turn order in
`docs/KERNEL.md`, and the measured data in `docs/PARITY.md` "Combat"
(CB-000..CB-041). Nothing else was used.

`go test -run Confirmed ./...` runs the CB vectors; `TestPrediction*`
tests pin BINARY-ONLY rules.

## Where it runs

`GenerateTurn` runs the research level-ups, then the battles, then the
second research level-up check of KERNEL.md step 6 (`LevelUpCheck`),
then repair. Research gained in a battle therefore levels up the same
turn (CB-018, CB-021). Battle draws come from the turn's `Rand`, in
the order COMBAT.md "Random draws in a battle" lists, including the 200
draws of a torpedo estimate for exactly 200 simulated torpedoes.

## Rules and tests

| Rule (COMBAT.md section) | Code | Status | Test |
|---|---|---|---|
| Locations, order of examination | `combat_who.go` `locations` | BINARY-ONLY | `TestPredictionBattleTurn` |
| Aggressors; only fleets start battles | `whoFights` | CONFIRMED | `TestConfirmedOnlyFleetsStartBattles` |
| Starbase joins with plan 0 | `whoFights` | CONFIRMED | `TestConfirmedStarbaseJoinsWithPlan0` |
| Procedure: P, Q, retaliation, friends | `whoFights` | BINARY-ONLY (firing back CONFIRMED) | `TestPredictionBattleTurn` |
| LEGACY BUG plan 0 "everyone"/named player, one-player battle; X at the first location (Elegy's chosen rule: contributes nothing) | `legacyPlan0Recipient`, `write` (ruleset switch `Legacy.Plan0Recipient`) | LEGACY BUG, CONFIRMED (CB-011..013, CB-022, CB-035); the first-location rule BINARY-ONLY | `TestConfirmedPlan0OnePlayerBattle`, `TestPredictionLegacyPlan0Recipient`, `TestPredictionPlan0AbsentPlayer` |
| Token cap: 255, quota 255/n, first pass from the location's first fleet then highest to lowest, re-add pass | `capTokens` | CONFIRMED (CB-039); a starbase and a skipped re-add BINARY-ONLY | `TestConfirmedTokenCap`, `TestPredictionTokenCapStarbase` |
| Start squares (flat table, rank in P, n = size of Q) | `startSquare` | CONFIRMED for n = 1 to 6 and a rank past row n (CB-022, CB-031..CB-036) | `TestConfirmedStartSquares`, `TestConfirmedPlan0OnePlayerBattle`, `TestConfirmedStartSquareFlatTable` |
| Setup: dump cargo, jitter, shuffle | `setup` | BINARY-ONLY | `TestPredictionBattleTurn` |
| Energy Dampener | `setup` | CONFIRMED | `TestConfirmedEnergyDampener` |
| Token values | `combat_token.go` `tokenValues` | CONFIRMED (starbase jammer BINARY-ONLY) | `TestConfirmedTokenValues`, `TestConfirmedRegeneratingShields`, `TestPredictionStarbaseJammer` |
| LEGACY BUG starbase is always "armed" | `starbaseClass` (ruleset switch `Legacy.StarbaseArmedClass`) | LEGACY BUG, CONFIRMED | `TestConfirmedStarbaseIsArmedTarget` |
| Speed code; cargo mass per ship `C·c/F` | `speedCode`, `battleMass` | CONFIRMED (designer CB-000; cargo and WM CB-038; dump CB-025) | `TestConfirmedEnergyDampener`, `TestConfirmedCargoMassPerShip` |
| Moves per round | `movesInRound` | CONFIRMED | `TestConfirmedMovesPerRound` |
| Rounds, out-of-battle check in player order (ends the battle only) | `combat_battle.go` `fight`, `checkIn` | CONFIRMED (CB-033) | `TestConfirmedPlan0OnePlayerBattle`, `TestPredictionOutPlayerStillFires` |
| Movement order, square choice, square score, damage estimate (no dropoff at r = 0) | `combat_move.go` | movement order CONFIRMED (CB-030), r = 0 CONFIRMED (CB-026); the rest BINARY-ONLY (one mover vs a station CONFIRMED, CB-012, CB-019..021) | `TestPredictionBattleTurn`, `TestPredictionEstimateDrawsAt200` |
| Disengaging | `step` | CONFIRMED; a move that stays put counts (CB-034) | `TestConfirmedDisengageLeavesOnEighthMove` |
| Regenerating Shields | `regenerate` | CONFIRMED | `TestConfirmedRegeneratingShields` |
| Firing order, target choice; live-player check before each token | `combat_fire.go` `fire`, `attractiveness` | CONFIRMED; the check BINARY-ONLY | `TestConfirmedTargetChoice`, `TestPredictionOutPlayerStillFires` |
| Design cost (COMPONENTS.md owner cost: miniaturization, PRT/LRT, Bleeding Edge) | `ships.go` `designCost` | CONFIRMED (CS-001) | `TestPredictionDesignCost` |
| Beams, dropoff (none at range 0, CB-026), carry | `beam` | CONFIRMED | `TestConfirmedBeamDropoff`, `TestConfirmedCarryRescaled` |
| Gatling, sappers | `gatling`, `damage` | CONFIRMED | `TestConfirmedGatlingHitsEveryTarget`, `TestConfirmedSapperShieldsOnly` |
| Torpedoes and missiles; `h = hits·d/2` once; a hit record for every reached target, even 0 hits | `hitChance`, `torpedoes` | CONFIRMED (CS-003-C2 for the rounding); the 0-hit record BINARY-ONLY | `TestConfirmedHitChance`, `TestConfirmedLargeSalvoHits`, `TestConfirmedMissileDoubleDamage`, `TestConfirmedOneKillPerMissile`, `TestConfirmedTorpedoMissesOnShields`, `TestConfirmedTorpedoHitRounding`, `TestPredictionTorpedoHitsPerTarget`, `TestPredictionTorpedoZeroHitRecord` |
| Damage, kills, spread | `damage` | CONFIRMED | `TestConfirmedMissileDoubleDamage` |
| Starbase damage | `starbaseDamage` | CONFIRMED in part | `TestConfirmedStarbaseDamageSteps` |
| Starbase loss: the queue's ship items removed, planetary items kept (starbase items kept: COMBAT.md names only ship items); AR planet uninhabited (emptied as TAKEOVER.md "Capture" lists) | `combat.go` `finish`, `emptyPlanet` | CONFIRMED (queue CB-047, AR CB-041) | parity CB-047, `TestPredictionAlternateRealityStarbaseLoss` |
| Salvage (10 kT steps, overflow objects) | `killEvent`, `cargoShare`, `addSalvage`, `finish` | CONFIRMED (one case each; overflow CB-040; fuel share CB-023); cargo share, empty object BINARY-ONLY | `TestConfirmedSalvage`, `TestConfirmedSalvageOverflow`, `TestPredictionCargoShare`, `TestPredictionFuelShare`, `TestPredictionSalvageLimit`, `TestPredictionEmptySalvageGetsTokenAmount` |
| Dump cargo (full amount; first salvage addition in deep space) | `setup` | CONFIRMED (CB-025) | `TestPredictionDumpCargo` |
| Repair | `repair` | CONFIRMED (moved, Inner Strength, starbase: CB-024) | `TestConfirmedRepair`, `TestPredictionRepairOthers` |
| Tech from battle, same-turn level; the "gained" mark is shared with capture (TAKEOVER.md) | `techAttempts`, `techAttempt`, `LevelUpCheck` | CONFIRMED in part (CB-018, CB-021) | `TestConfirmedTechFromBattleSameTurn`, `TestConfirmedTechAttemptLocation`, `TestPredictionTechAttemptRules` |
| Who attempts: participants (location, `n = 2` survivors, AR starbase), players outside the battle | `techAttempts` | CONFIRMED (CB-012, CB-021, CB-029, CB-031-n3, CB-041) | `TestConfirmedTechAttemptLocation`, `TestPredictionTechAttemptRules` |
| LEGACY BUG observer tech attempt (player number AND observer mask; a planet owner without a starbase is in the mask even as a participant) | `techAttempts` (ruleset switch `Legacy.ObserverTechMask`) | LEGACY BUG, CONFIRMED (CB-031-obs, CB-037) | `TestPredictionTechAttemptRules` |

## Not tested against the oracle

Implemented as written, with no oracle data yet (COMBAT.md "Open
experiments"):

- the token cap with more than two involved players, with a starbase,
  and with multi-design fleets;
- the exact plan-0 value X at the first location of a turn (Elegy
  follows COMBAT.md's chosen rule: plan 0 contributes nothing);
- the 0-hit torpedo record and the cargo share lost with destroyed
  ships;
- salvage at more than one point.

The battle record goes to every player in P and no one else (CONFIRMED,
CB-031): Elegy emits `EventBattle` for each player in P.

The exact CB-019/CB-020 squares and the CB-021 per-stream gains depend on
the original's random stream and part table, so they are not replayed
here; the stream-independent parts of CB-021 and CB-022 are tests.

## Implementation assumptions and placeholders

These are Elegy's own choices where COMBAT.md is silent, ambiguous or not
settled. They are **not** established Stars! behavior. Each one is marked
in the code with `ASSUMPTION An` or `PLACEHOLDER An`. An ASSUMPTION is a
reading of unclear text. A PLACEHOLDER stands in for a value or rule the
spec says is unknown.

None open. A6 (plan-0 X at the first location of a turn) is now
COMBAT.md's chosen rule for Elegy (#38), so it is cited above rather than
listed here.

Resolved by stars-elegy #31 and now cited rules: A1 (step 5 only ends
the battle; the live-player check before each token), A2 (every move
counts), A3 (dump cargo), A4 (fuel share by fuel capacity), A8 (Inner
Strength for both repair lines), A9 (who attempts tech, with the
observer LEGACY BUG), A10 (a starbase's plain owner cost in target
choice), and A7 (no dropoff for a range-0 beam, in real fire and in
the estimate). A5 (the Bleeding Edge suppression flag)
is gone: COMPONENTS.md's CONFIRMED rule has no such flag.

### Settled by current COMBAT.md

Earlier versions of this file listed these as open choices. The current
spec now determines them:

- **Off-board squares.** Only squares on the board are scored for the
  pick. A step toward the pick compares neighbours, and "a step that would
  leave the board leaves the token where it is" ("Choosing a square").
- **Beam estimate at distance 0.** The dropoff term applies only "if
  `x > 0`" ("Damage estimate").
- **Range-3 exception.** It uses the "longest non-sapper beam range"
  ("Choosing a square").
- **Capacitors.** The product runs over every item, capped at 2550;
  132% is CONFIRMED for one Flux and one Energy Capacitor (CB-002 C4)
  ("Token values").
- **Mystery Trader items.** Each kill event adds each Trader part slot's
  count to that part's chance, at most 25; the hull never counts. Each of
  a tech attempt's 13 `rand(13)` tries for k draws `rand(100)` only for
  an item with a chance the player lacks, and below the chance gives
  Trader part bit k (OBJECTS.md "Encounters" order; CONFIRMED CB-048;
  `traderChances`, `techAttempt`, `TestConfirmedTraderChances`,
  `TestConfirmedTechAttemptTraderItem`). Without space objects no part
  has a chance ("Tech from battle", "Mystery Trader chances"). Of the 13
  indices only k 8 (Mini Morph) is a hull; k 10 (Genesis Device) is on no
  ship and k 12 is the ship-gift bit, so neither ever has a chance
  (COMBAT.md's index table as corrected by stars-elegy #127).
- **Salvage order.** Deep-space salvage additions are made after the
  battle's tech attempts ("Random draws in a battle", step 5).

### Not implemented (no Elegy state yet)

- **Part and hull values** come from the component table
  (`engine/catalog.go`, `docs/COMPONENTS-STATUS.md`): `NewDesign` builds a
  `Design` from catalogue names. Designs built by hand still work.
- **Mystery Trader items** and **queued ships and packets** lost with a
  starbase: the production queue has no Trader items or ship items yet.
