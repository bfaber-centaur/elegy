# Combat status

The battle phase in `engine/` follows the public combat specification,
stars-elegy `docs/COMBAT.md` as merged in PR #19 (`main` at `3968ec2`)
plus the implementer answers in stars-elegy PR #24 (branch at `c5dda13`:
per-target torpedo hits, the salvage limit, no leftover past a starbase,
capacitors, self-entries in attack sets, absent players in retaliation),
the turn order in `docs/KERNEL.md`, and the measured data in
`docs/PARITY.md` "Combat" (CB-000..CB-022). Nothing else was used.

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
| LEGACY BUG plan 0 "everyone"/named player, one-player battle | `legacyPlan0Recipient`, `write` (switch `legacyPlan0`) | LEGACY BUG, CONFIRMED (CB-011..013, CB-022) | `TestConfirmedPlan0OnePlayerBattle`, `TestPredictionLegacyPlan0Recipient`, `TestPredictionPlan0AbsentPlayer` |
| Token cap | `capTokens` | BINARY-ONLY | `TestPredictionTokenCap` |
| Start squares (flat table, rank in P, n = size of Q) | `startSquare` | CONFIRMED for n = 2 and n = 1 | `TestConfirmedStartSquares`, `TestConfirmedPlan0OnePlayerBattle`, `TestPredictionStartSquareFlatTable` |
| Setup: dump cargo, jitter, shuffle | `setup` | BINARY-ONLY | `TestPredictionBattleTurn` |
| Energy Dampener | `setup` | CONFIRMED | `TestConfirmedEnergyDampener` |
| Token values | `combat_token.go` `tokenValues` | CONFIRMED (starbase jammer BINARY-ONLY) | `TestConfirmedTokenValues`, `TestConfirmedRegeneratingShields`, `TestPredictionStarbaseJammer` |
| LEGACY BUG starbase is always "armed" | `starbaseClass` (switch `legacyStarbaseArmedClass`) | LEGACY BUG, CONFIRMED | `TestConfirmedStarbaseIsArmedTarget` |
| Speed code | `speedCode` | CONFIRMED by the designer; cargo, WM, dump BINARY-ONLY | `TestConfirmedEnergyDampener` |
| Moves per round | `movesInRound` | CONFIRMED | `TestConfirmedMovesPerRound` |
| Rounds, out-of-battle check in player order | `combat_battle.go` `fight`, `checkIn` | BINARY-ONLY ordering | `TestConfirmedPlan0OnePlayerBattle` |
| Movement order, square choice, square score, damage estimate | `combat_move.go` | BINARY-ONLY (one mover vs a station CONFIRMED, CB-012, CB-019..021) | `TestPredictionBattleTurn`, `TestPredictionEstimateDrawsAt200` |
| Disengaging | `step` | CONFIRMED | `TestConfirmedDisengageLeavesOnEighthMove` |
| Regenerating Shields | `regenerate` | CONFIRMED | `TestConfirmedRegeneratingShields` |
| Firing order, target choice | `combat_fire.go` `fire`, `attractiveness` | CONFIRMED | `TestConfirmedTargetChoice` |
| Design cost (miniaturization, PRT/LRT, Bleeding Edge) | `ships.go` `designCost` | BINARY-ONLY | `TestPredictionDesignCost` |
| Beams, dropoff, carry | `beam` | CONFIRMED | `TestConfirmedBeamDropoff`, `TestConfirmedCarryRescaled` |
| Gatling, sappers | `gatling`, `damage` | CONFIRMED | `TestConfirmedGatlingHitsEveryTarget`, `TestConfirmedSapperShieldsOnly` |
| Torpedoes and missiles | `hitChance`, `torpedoes` | CONFIRMED | `TestConfirmedHitChance`, `TestConfirmedLargeSalvoHits`, `TestConfirmedMissileDoubleDamage`, `TestConfirmedOneKillPerMissile`, `TestConfirmedTorpedoMissesOnShields`, `TestPredictionTorpedoHitsPerTarget` |
| Damage, kills, spread | `damage` | CONFIRMED | `TestConfirmedMissileDoubleDamage` |
| Starbase damage | `starbaseDamage` | CONFIRMED in part | `TestConfirmedStarbaseDamageSteps` |
| Starbase loss, AR planet uninhabited | `combat.go` `finish` | BINARY-ONLY | `TestPredictionAlternateRealityStarbaseLoss` |
| Salvage (10 kT steps, overflow objects) | `killEvent`, `cargoShare`, `addSalvage`, `finish` | CONFIRMED (one case each); cargo share and empty object BINARY-ONLY | `TestConfirmedSalvage`, `TestPredictionCargoShare`, `TestPredictionSalvageLimit`, `TestPredictionEmptySalvageGetsTokenAmount` |
| Repair | `repair` | CONFIRMED; moved, IS, starbase BINARY-ONLY | `TestConfirmedRepair`, `TestPredictionRepairOthers` |
| Tech from battle, same-turn level | `techAttempts`, `techAttempt`, `LevelUpCheck` | CONFIRMED in part (CB-018, CB-021) | `TestConfirmedTechFromBattleSameTurn`, `TestConfirmedTechAttemptLocation` |

## Not tested against the oracle

Implemented as written, with no oracle data yet:

- battles with several moving tokens on both sides (movement order by
  jittered weight, square ties among many movers);
- three or more players, start squares for other counts, friends joining,
  a starbase owner in P but not in Q;
- the token cap, dump cargo, War Monger and cargo in the speed code;
- design-cost race adjustments and Bleeding Edge doubling;
- the tech-attempt condition in larger battles and for outside players;
- starbase loss side effects, the "moved", IS and starbase repair rates.

The exact CB-019/CB-020 squares and the CB-021 per-stream gains depend on
the original's random stream and part table, so they are not replayed
here; the stream-independent parts of CB-021 and CB-022 are tests.

## Choices where COMBAT.md is silent or ambiguous

Each was raised with the spec author; the code keeps this default until
the spec says otherwise.

- **Part and hull values.** Elegy has no part catalogue yet; designs carry
  their hull's and parts' values and base costs. A public component table
  (stars-elegy `docs/COMPONENTS.md`) is planned and can fill `Design`.
- **Players still in the battle** while firing are those in after step 5
  that still have a live token; a player out at step 5 does not fire.
- **Beam estimate** at distance 0 or range 0 skips the dropoff term; the
  range-3 exception uses non-sapper beams.
- **Off-board steps** are scored, and choosing one leaves the token in
  place. The disengage counter counts each move action.
- **Dumped cargo** in deep space is one salvage addition, without the
  quarter loss. Salvage additions are made after the battle's tech
  attempts, in kill-event order (the draw order COMBAT.md lists).
- **Fuel** of destroyed ships is not removed from the fleet (COMBAT.md
  says it is destroyed but not how the share is computed).
- **Mystery Trader items** are not modelled; the 13 `rand(13)` item tries
  are still drawn.
- **Bleeding Edge suppression flag** is not identified; Elegy always
  doubles.
- **Queued ships and packets** lost with a starbase are not modelled
  (Elegy's queue has no ship items yet).
