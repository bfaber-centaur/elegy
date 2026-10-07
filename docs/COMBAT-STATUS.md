# Combat status

The battle phase in `engine/` follows the public combat specification,
stars-elegy `docs/COMBAT.md` as of stars-elegy `main` at `a6ac76e` (PR #19
plus the implementer answers merged in PR #24, `113a4d2`) with the
follow-up answers in open PR #31 (branch at `56f10c4`), the owner cost
rule of `docs/COMPONENTS.md` in open PR #30 (branch at `f851a4b`), the
turn order in `docs/KERNEL.md`, and the measured data in `docs/PARITY.md`
"Combat" (CB-000..CB-022). Nothing else was used.

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
| Rounds, out-of-battle check in player order (ends the battle only) | `combat_battle.go` `fight`, `checkIn` | BINARY-ONLY ordering | `TestConfirmedPlan0OnePlayerBattle`, `TestPredictionOutPlayerStillFires` |
| Movement order, square choice, square score, damage estimate | `combat_move.go` | BINARY-ONLY (one mover vs a station CONFIRMED, CB-012, CB-019..021) | `TestPredictionBattleTurn`, `TestPredictionEstimateDrawsAt200` |
| Disengaging | `step` | CONFIRMED; a move that stays put counts, BINARY-ONLY | `TestConfirmedDisengageLeavesOnEighthMove` |
| Regenerating Shields | `regenerate` | CONFIRMED | `TestConfirmedRegeneratingShields` |
| Firing order, target choice; live-player check before each token | `combat_fire.go` `fire`, `attractiveness` | CONFIRMED; the check BINARY-ONLY | `TestConfirmedTargetChoice`, `TestPredictionOutPlayerStillFires` |
| Design cost (COMPONENTS.md owner cost: miniaturization, PRT/LRT, Bleeding Edge) | `ships.go` `designCost` | CONFIRMED (CS-001) | `TestPredictionDesignCost` |
| Beams, dropoff, carry | `beam` | CONFIRMED | `TestConfirmedBeamDropoff`, `TestConfirmedCarryRescaled` |
| Gatling, sappers | `gatling`, `damage` | CONFIRMED | `TestConfirmedGatlingHitsEveryTarget`, `TestConfirmedSapperShieldsOnly` |
| Torpedoes and missiles | `hitChance`, `torpedoes` | CONFIRMED | `TestConfirmedHitChance`, `TestConfirmedLargeSalvoHits`, `TestConfirmedMissileDoubleDamage`, `TestConfirmedOneKillPerMissile`, `TestConfirmedTorpedoMissesOnShields`, `TestPredictionTorpedoHitsPerTarget` |
| Damage, kills, spread | `damage` | CONFIRMED | `TestConfirmedMissileDoubleDamage` |
| Starbase damage | `starbaseDamage` | CONFIRMED in part | `TestConfirmedStarbaseDamageSteps` |
| Starbase loss, AR planet uninhabited | `combat.go` `finish` | BINARY-ONLY | `TestPredictionAlternateRealityStarbaseLoss` |
| Salvage (10 kT steps, overflow objects) | `killEvent`, `cargoShare`, `addSalvage`, `finish` | CONFIRMED (one case each); cargo and fuel share, empty object BINARY-ONLY | `TestConfirmedSalvage`, `TestPredictionCargoShare`, `TestPredictionFuelShare`, `TestPredictionSalvageLimit`, `TestPredictionEmptySalvageGetsTokenAmount` |
| Dump cargo (full amount; first salvage addition in deep space) | `setup` | BINARY-ONLY | `TestPredictionDumpCargo` |
| Repair | `repair` | CONFIRMED; moved, Inner Strength, starbase BINARY-ONLY | `TestConfirmedRepair`, `TestPredictionRepairOthers` |
| Tech from battle, same-turn level | `techAttempts`, `techAttempt`, `LevelUpCheck` | CONFIRMED in part (CB-018, CB-021) | `TestConfirmedTechFromBattleSameTurn`, `TestConfirmedTechAttemptLocation` |

## Not tested against the oracle

Implemented as written, with no oracle data yet:

- battles with several moving tokens on both sides (movement order by
  jittered weight, square ties among many movers);
- three or more players, start squares for other counts, friends joining,
  a starbase owner in P but not in Q;
- the token cap, War Monger and cargo in the speed code;
- dump cargo, the fuel share, an out player still firing, and a
  disengaging token that stays on its square;
- the tech-attempt condition in larger battles and for outside players;
- starbase loss side effects, the "moved", Inner Strength and starbase repair rates.

The exact CB-019/CB-020 squares and the CB-021 per-stream gains depend on
the original's random stream and part table, so they are not replayed
here; the stream-independent parts of CB-021 and CB-022 are tests.

## Implementation assumptions and placeholders

These are Elegy's own choices where COMBAT.md is silent, ambiguous or not
settled. They are **not** established Stars! behavior. Each one is marked
in the code with `ASSUMPTION An` or `PLACEHOLDER An`. An ASSUMPTION is a
reading of unclear text. A PLACEHOLDER stands in for a value or rule the
spec says is unknown.

| Id | What Elegy does | What COMBAT.md says | When it matters | Code |
|---|---|---|---|---|
| A6 | **Placeholder.** On the first location of a turn, plan-0 LEGACY BUG X is treated as a non-player, so plan 0 adds nothing. | X is "not determined" there (leftover; neither player in two-player CB-022). An X that is not a player has no effect. | Three or more players, where the leftover X might be a real player. | `legacyPlan0Recipient` |
| A7 | A range-0 beam skips the dropoff term in the damage estimate. | The formula divides by the range `r`; range-0 beams are not covered. | Only for a design with a range-0 beam, if one exists. | `estimate` |
| A9 | In battles other than two players with two tokens, a player attempts tech only when another player's ships were destroyed. | "Probably only when ships other than their own were destroyed; this condition is not fully settled." | Larger battles. | `techAttempts` |
| A10 | A starbase token's cost for target choice is the plain owner cost. | "Design cost" points to COMPONENTS.md "Cost for an owner"; that section also gives a starbase design cost (ISB/AR `c − c/5`, then halved) without saying whether target choice uses it. | Ships choosing between a starbase and other targets. Raised with the spec author. | `designCost` |

Resolved by stars-elegy #31 and now cited rules: A1 (step 5 only ends
the battle; the live-player check before each token), A2 (every move
counts), A3 (dump cargo), A4 (fuel share by fuel capacity), A8 (Inner
Strength for both repair lines). A5 (the Bleeding Edge suppression flag)
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
- **Mystery Trader draws.** Each tech attempt makes its 13 `rand(13)`
  tries. Elegy has no trader items, so none is ever given ("Tech from
  battle").
- **Salvage order.** Deep-space salvage additions are made after the
  battle's tech attempts ("Random draws in a battle", step 5).

### Not implemented (no Elegy state yet)

- **Part and hull values.** Elegy has no part catalogue yet; designs carry
  their hull's and parts' values and base costs. The public component
  table (stars-elegy `docs/COMPONENTS.md`) can fill `Design` when it lands.
- **Mystery Trader items** and **queued ships and packets** lost with a
  starbase: Elegy has no trader items, and its production queue has no
  ship items yet.
