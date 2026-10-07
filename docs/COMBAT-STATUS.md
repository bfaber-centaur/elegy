# Combat status

The battle phase in `engine/` follows the public combat specification,
stars-elegy `docs/COMBAT.md` as of stars-elegy `main` at `a6ac76e` (PR #19
plus the implementer answers merged in PR #24, `113a4d2`), the turn order
in `docs/KERNEL.md`, and the measured data in `docs/PARITY.md` "Combat"
(CB-000..CB-022). Nothing else was used.

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

## Implementation assumptions and placeholders

These are Elegy's own choices where current COMBAT.md is silent,
ambiguous or not settled. They are **not** established Stars! behavior.
Each one is marked in the code with `ASSUMPTION An` or `PLACEHOLDER An`
and was sent to the spec author for a decision. An ASSUMPTION is a
reading of unclear text. A PLACEHOLDER stands in for a value or rule the
spec says is unknown.

| Id | What Elegy does | What COMBAT.md says | When it matters | Code |
|---|---|---|---|---|
| A1 | While firing, a player counts as "still in" only if it was in after step 5 **and** still has a live token. A player that is out at step 5 does not fire. Each round's step 5 starts again from every player with a live token. | Firing goes on "while at least two players are still in the battle"; "in" is defined only at round step 5. | Only with three or more players (with two, the battle ends at the first "out", and a side with no live tokens leaves nothing to target). Untested. | `fire`, `playersIn`, `checkIn` |
| A2 | Every move a disengaging token is given lowers the counter, including a move that leaves it on its square. | "Each move it makes lowers the counter by 1"; it leaves on its 8th move (CONFIRMED, CB-003/004). Whether a move that stays on the same square counts is not stated. | A disengaging token that scores its own square best, or whose step is blocked by the board edge. | `step` |
| A3 | Dumped cargo in deep space is one addition to the battle's salvage object, with no quarter lost, added before the kill events' additions. At a planet it all goes to the surface (spec). | "Its minerals go to the planet's surface, or to deep-space salvage." The quarter loss is stated for kill events only. Dump cargo is in "Open experiments". | Deep-space battles where a fleet's plan dumps cargo. | `setup` |
| A4 | **Placeholder.** The fuel of destroyed ships stays in the fleet. | "The lost ships' share of fuel and colonists is destroyed", with a formula for cargo but none for fuel. | Partly destroyed fleets: survivors may keep more fuel than their tanks hold. **Needs a spec rule.** | `cargoShare` |
| A5 | **Placeholder.** The Bleeding Edge doubling always applies. | "One game-wide flag, not identified, suppresses this." | Bleeding Edge races with parts at their tech requirement. | `bleedingEdgeSuppressed` |
| A6 | **Placeholder.** On the first location of a turn, plan-0 LEGACY BUG X is treated as a non-player, so plan 0 adds nothing. | X is "not determined" there (leftover; neither player in two-player CB-022). An X that is not a player has no effect. | Three or more players, where the leftover X might be a real player. | `legacyPlan0Recipient` |
| A7 | A range-0 beam skips the dropoff term in the damage estimate. | The formula divides by the range `r`; range-0 beams are not covered. | Only for a design with a range-0 beam, if one exists (no part table yet). | `estimate` |
| A8 | Fleets: Interstellar Traveler doubles `r`. Starbases: Inner Strength repairs 75. | "Interstellar Traveler doubles `r`" and "repairs 50 units (IS 75)"; the stars-elegy docs use IS for Inner Strength, and "Open experiments" lists "IS repair". The two lines may mean one race. **Needs a spec decision.** | IT and IS races' repair. | `repair` |
| A9 | In battles other than two players with two tokens, a player attempts tech only when another player's ships were destroyed. | "Probably only when ships other than their own were destroyed; this condition is not fully settled." | Larger battles. | `techAttempts` |

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
