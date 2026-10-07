# Combat status

The battle phase in `engine/` follows the public combat specification,
stars-elegy `docs/COMBAT.md` (PR #19, branch `claude/project-thread-9j7tt8`
at `9ae3586`), and the measured data it links in `docs/PARITY.md`
"Combat" (CB-000..CB-019). Nothing else was used.

`go test -run Confirmed ./...` runs the CB vectors; `TestPrediction*`
tests pin BINARY-ONLY rules.

## Where it runs

`GenerateTurn` fights battles after starbase refuelling and before
research level-ups, then repairs, so research gained in a battle levels
up the same turn (CB-018). Battle draws come from the turn's `Rand`, in
the order COMBAT.md "Random draws in a battle" lists.

## Rules and tests

| Rule (COMBAT.md section) | Code | Status | Test |
|---|---|---|---|
| Locations, order of examination | `combat_who.go` `locations` | BINARY-ONLY | `TestPredictionBattleTurn` |
| Aggressors; only fleets start battles | `whoFights` | CONFIRMED | `TestConfirmedOnlyFleetsStartBattles` |
| Starbase joins with plan 0 | `whoFights` | CONFIRMED | `TestConfirmedStarbaseJoinsWithPlan0` |
| Retaliation, friends, involvement | `whoFights` | BINARY-ONLY (firing back CONFIRMED) | `TestPredictionBattleTurn` |
| LEGACY BUG plan 0 "everyone"/named player | `legacyPlan0Recipient` (switch `legacyPlan0`) | LEGACY BUG, BINARY-ONLY | `TestPredictionLegacyPlan0Recipient` |
| Token cap | `capTokens` | BINARY-ONLY | `TestPredictionTokenCap` |
| Start squares | `startSquares` | CONFIRMED for 2 players | `TestConfirmedStartSquares` |
| Setup: dump cargo, jitter, shuffle | `setup` | BINARY-ONLY | `TestPredictionBattleTurn` |
| Energy Dampener | `setup` | CONFIRMED | `TestConfirmedEnergyDampener` |
| Token values | `combat_token.go` `tokenValues` | CONFIRMED (starbase jammer BINARY-ONLY) | `TestConfirmedTokenValues`, `TestConfirmedRegeneratingShields`, `TestPredictionStarbaseJammer` |
| LEGACY BUG starbase is always "armed" | `starbaseClass` (switch `legacyStarbaseArmedClass`) | LEGACY BUG, CONFIRMED | `TestConfirmedStarbaseIsArmedTarget` |
| Speed code | `speedCode` | CONFIRMED by the designer; cargo, WM, dump BINARY-ONLY | `TestConfirmedEnergyDampener` |
| Moves per round | `movesInRound` | CONFIRMED | `TestConfirmedMovesPerRound` |
| Rounds | `combat_battle.go` `fight` | BINARY-ONLY ordering | `TestPredictionBattleTurn` |
| Movement order, square choice, square score, damage estimate | `combat_move.go` | BINARY-ONLY | `TestPredictionBattleTurn` |
| Disengaging | `step` | CONFIRMED | `TestConfirmedDisengageLeavesOnEighthMove` |
| Regenerating Shields | `regenerate` | CONFIRMED | `TestConfirmedRegeneratingShields` |
| Firing order, target choice | `combat_fire.go` `fire`, `attractiveness` | CONFIRMED | `TestConfirmedTargetChoice` |
| Beams, dropoff, carry | `beam` | CONFIRMED | `TestConfirmedBeamDropoff`, `TestConfirmedCarryRescaled` |
| Gatling, sappers | `gatling`, `damage` | CONFIRMED | `TestConfirmedGatlingHitsEveryTarget`, `TestConfirmedSapperShieldsOnly` |
| Torpedoes and missiles | `hitChance`, `torpedoes` | CONFIRMED | `TestConfirmedHitChance`, `TestConfirmedLargeSalvoHits`, `TestConfirmedMissileDoubleDamage`, `TestConfirmedOneKillPerMissile`, `TestConfirmedTorpedoMissesOnShields` |
| Damage, kills, spread | `damage` | CONFIRMED | `TestConfirmedMissileDoubleDamage` |
| Starbase damage | `starbaseDamage` | CONFIRMED in part | `TestConfirmedStarbaseDamageSteps` |
| Starbase loss, AR planet uninhabited | `combat.go` `finish` | BINARY-ONLY | `TestPredictionAlternateRealityStarbaseLoss` |
| Salvage | `killEvent`, `finish` | CONFIRMED (one case each); empty object BINARY-ONLY | `TestConfirmedSalvage`, `TestPredictionEmptySalvageGetsTokenAmount` |
| Repair | `repair` | CONFIRMED; moved, IS, starbase BINARY-ONLY | `TestConfirmedRepair`, `TestPredictionRepairOthers` |
| Tech from battle, same-turn level | `techAttempts`, `techAttempt` | CONFIRMED in part | `TestConfirmedTechFromBattleSameTurn` |

## Not tested against the oracle

Implemented as written, with no oracle data yet:

- battles with several moving tokens on both sides (movement order by
  jittered weight, square ties among many movers);
- three or more players, start squares for other counts, friends joining;
- the token cap, dump cargo, War Monger and cargo in the speed code;
- the tech-attempt condition in larger battles and for outside players;
- starbase loss side effects, the "moved", IS and starbase repair rates.

## Choices where COMBAT.md is silent or ambiguous

Each was raised with the spec author; the code keeps this default until
the spec says otherwise.

- **Undamaged stacks.** `damaged = max(1, ships·pct/100)` is taken as 0
  when the stack has no damage (the literal rule gives 149 instead of the
  observed 148 in CB-001 B3).
- **Plan 0 at the first location.** The LEGACY BUG's X is undetermined for
  the first location examined; Elegy uses the starbase's owner. When X is
  the attacked player itself, no battle follows (rule 6).
- **Part and hull values.** Elegy has no part catalogue; designs carry
  their parts' values and the current per-ship cost (`Design.Cost`,
  including miniaturization and discounts).
- **Cargo share** for mass and salvage is by cargo capacity.
- **Square score** counts enemies that do not attack the token too; the
  out-of-battle check uses the attack set only; the torpedo estimate uses
  the expected hit count (no draws).
- **Beam estimate** at distance 0 or range 0 skips the dropoff term; the
  range-3 exception uses non-sapper beams.
- **Off-board steps** are scored, and choosing one leaves the token in
  place. The disengage counter counts each move action.
- **Players still in the battle** are recounted during firing (live
  tokens with something to attack).
- **Torpedo hit chance** depends on the target's jammer but the hits are
  drawn once per salvo; Elegy uses the first target chosen.
- **Starbase leftover** for beam carry is the damage past its remaining
  armor.
- **Deep-space salvage** over 30000 kT is scaled down in proportion per
  mineral. Dumped cargo goes into it without the quarter loss.
- **Mystery Trader items** are not modelled; the 13 `rand(13)` item tries
  are still drawn.
- **Capacitors.** COMBAT.md compounds per item, which gives 158% for two
  Flux Capacitors and an Energy Capacitor, not the 132% quoted for that
  design; the code follows the per-item rule, and the test uses one of
  each (132%).
- **Queued ships and packets** lost with a starbase are not modelled
  (Elegy's queue has no ship items yet).
