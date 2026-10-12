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
| Token values | `combat_token.go` `tokenValues` | CONFIRMED (CB-056, stars-elegy b0ca924: the starbase jammer's quarter after the 95 cap, the 255% capacitor cap, initiative 63 and computer 99 with sixteen Battle Nexus, RS counting Croby Sharmor's armor whole); RS halving each armor slot's armor, truncated once per slot: BINARY-ONLY (stars-elegy #164, 41e7a1c, COMBAT.md "Armor"), MEASURED for one slot (RG-002-Y6, one stream: 625, not 624 per part); the Langston Shell under RS not observed; the other late parts BINARY-ONLY | `TestConfirmedTokenValues`, `TestConfirmedRegeneratingShields`, `TestMeasuredRG002RegeneratingShieldsArmor`, `TestPredictionStarbaseJammer`, `TestPredictionLateShipTokenValues`, `TestPredictionDeathStarTokenValues` |
| LEGACY BUG starbase is always "armed" | `starbaseClass` (ruleset switch `Legacy.StarbaseArmedClass`) | LEGACY BUG, CONFIRMED | `TestConfirmedStarbaseIsArmedTarget` |
| Speed code; cargo mass per ship `C·c/F` | `speedCode`, `battleMass` | CONFIRMED (designer CB-000; cargo and WM CB-038; dump CB-025) | `TestConfirmedEnergyDampener`, `TestConfirmedCargoMassPerShip` |
| Moves per round | `movesInRound` | CONFIRMED | `TestConfirmedMovesPerRound` |
| Rounds, out-of-battle check in player order (ends the battle only) | `combat_battle.go` `fight`, `checkIn` | CONFIRMED (CB-033) | `TestConfirmedPlan0OnePlayerBattle`, `TestPredictionOutPlayerStillFires` |
| Movement order, square choice, square score, damage estimate (no dropoff at r = 0) | `combat_move.go` | movement order CONFIRMED (CB-030), r = 0 CONFIRMED (CB-026); one mover vs a station CONFIRMED (CB-012, CB-019..021); MEASURED: every recorded move the battle-record check can check is a square Elegy's rule allows (see "Battle records" in RANDOM-PARITY.md) | `TestPredictionBattleTurn`, `TestPredictionEstimateDrawsAt200`, `TestBattleRecords` |
| Disengaging; tactic 0's own square scores +2 for the token itself and −1 for being current | `step`, `score` | CONFIRMED; a move that stays put counts (CB-034); the own-square term CB-032 (a lone Runner never stayed) and the battle records' moves | `TestConfirmedDisengageLeavesOnEighthMove`, `TestConfirmedDisengageOwnSquare`, `TestBattleRecords` |
| Regenerating Shields | `regenerate` | CONFIRMED | `TestConfirmedRegeneratingShields` |
| Firing order, target choice; live-player check before each token | `combat_fire.go` `fire`, `attractiveness` | CONFIRMED; the check BINARY-ONLY | `TestConfirmedTargetChoice`, `TestPredictionOutPlayerStillFires` |
| Design cost (COMPONENTS.md owner cost: miniaturization, PRT/LRT, Bleeding Edge) | `ships.go` `designCost` | CONFIRMED (CS-001) | `TestPredictionDesignCost` |
| Beams, dropoff (none at range 0, CB-026), carry | `beam` | CONFIRMED | `TestConfirmedBeamDropoff`, `TestConfirmedCarryRescaled` |
| Gatling (each target's own deflector), sappers | `gatling`, `damage` | CONFIRMED; a gatling shot goes on past a starbase (CONFIRMED in one stream only, CB-056, stars-elegy b0ca924) | `TestConfirmedGatlingHitsEveryTarget`, `TestGatlingDeflectorPerTarget`, `TestConfirmedSapperShieldsOnly` |
| Torpedoes and missiles; `h = hits·d/2` once; misses' shield damage in its own record before the hit record; a hit record for every reached target, even 0 hits | `hitChance`, `torpedoes` | CONFIRMED (CS-003-C2 for the rounding, CB-049 for the miss record); the 0-hit record BINARY-ONLY | `TestConfirmedHitChance`, `TestConfirmedLargeSalvoHits`, `TestConfirmedMissileDoubleDamage`, `TestConfirmedOneKillPerMissile`, `TestConfirmedTorpedoMissesOnShields`, `TestConfirmedTorpedoHitRounding`, `TestPredictionTorpedoHitsPerTarget`, `TestPredictionTorpedoZeroHitRecord` |
| Damage, kills, spread | `damage` | CONFIRMED | `TestConfirmedMissileDoubleDamage` |
| Starbase damage; no damage left over after a hit on a starbase; kept through the year | `starbaseDamage`, `finish`, `repair` | CONFIRMED in part (Space Station steps CB-011..015; the stop CB-052); the Death Star's 1500 armor BINARY-ONLY; a 500th step, destroyed only at total ≥ armor, MEASURED (CB-056, 2 streams, stars-elegy b0ca924) | `TestConfirmedStarbaseDamageSteps`, `TestPredictionDeathStarDamageSteps`, `TestConfirmedBeamStopsAtStarbase`, `TestRG001StarbaseDamageKept` |
| Starbase loss: the queue's ship items and non-automatic packet items removed, a starbase item restarts at 0%, planetary items kept (shared with the starbase design delete); AR planet uninhabited (emptied as TAKEOVER.md "Capture" lists) | `combat.go` `finish`, `starbase_loss.go` `loseStarbase`, `emptyPlanet` | CONFIRMED (ship and planetary items CB-047, AR CB-041); packet items and the starbase restart BINARY-ONLY (PARITY.md "Queued ships after deleting the starbase design", stars-elegy ee47fff) | parity CB-047, `TestConfirmedBattleLossStripsQueue`, `TestPredictionAlternateRealityStarbaseLoss` |
| Salvage (10 kT steps, overflow objects) | `killEvent`, `cargoShare`, `addSalvage`, `finish` | CONFIRMED (one case each; overflow CB-040; fuel share CB-023); cargo share, empty object BINARY-ONLY | `TestConfirmedSalvage`, `TestConfirmedSalvageOverflow`, `TestPredictionCargoShare`, `TestPredictionFuelShare`, `TestPredictionSalvageLimit`, `TestPredictionEmptySalvageGetsTokenAmount` |
| Dump cargo (full amount; first salvage addition in deep space) | `setup` | CONFIRMED (CB-025) | `TestPredictionDumpCargo` |
| Repair | `repair` | CONFIRMED (moved, Inner Strength, starbase: CB-024; the Ultra Station's and Death Star's docks, fleets −100 and the starbase −50: CB-056, stars-elegy b0ca924) | `TestConfirmedRepair`, `TestPredictionRepairOthers`, `TestPredictionRepairLargeDocks`, `TestRG001StarbaseDamageKept` |
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

CB-058 asks these, with the Langston Shell under RS and the Death Star's
reach and armor (predictions below, "CB-058 predictions").

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

Resolved by CB-056 (stars-elegy b0ca924; see "CB-056 results" below):
A11 was refuted (a starbase at 499 goes to 500 and only a total of at
least its armor destroys it, MEASURED, now the rule in `starbaseDamage`);
A12 held in one stream only and A13 held for Croby Sharmor (the Langston
Shell is not observed), both CONFIRMED.

A6 (plan-0 X at the first location of a turn) is now
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

## Late-game coverage (before RG-002)

RG-001's battles (the `rg` vectors, years 2429 to 2435) never reach most
late-game paths. Run over RG-001 alone (statement coverage of `engine`
and `objects`), these had none: starbase damage (`starbaseDamage`),
`gatling`, a starbase loss (`loseStarbase`), and the minefield hit and
its damage (`objects` `MineDamage`, `MinesLost`, `PayingField`,
`ApplyHit`); the token cap (`capTokens`) had 31%. RG-001's battles use
Lasers, X-Ray Lasers, Alpha and Beta Torpedoes, Battle Computers, and the
Orbital Fort, Space Dock and Space Station. No battle record in the
vectors has an Ultra Station or a Death Star, a Battle Nexus, the
Blasters, Disruptors, Pulverizers, Gatling Neutrino Cannon, Big Mutha
Cannon or Syncro Sapper, a shield above the Langston Shell, or
Superlatanium, Valanium or Fielded Kelarium. The unit tests cover every
combat function; the tests added for this pass (rows above, and
`TestPredictionMineDamageLateFleets` in `objects`) pin the late values of
each rule from COMBAT.md and OBJECTS.md.

Oracle questions for a late-game run (RG-002 or a CB case); CB-056
answered all seven (below):

1. A starbase at 499/500 takes a hit of less than one unit of its armor
   and stays below its armor in total: is it destroyed, or left at 499
   (A11)?
2. A gatling shot whose targets include a starbase before other ships:
   are the ships after it hit (A12)?
3. With Regenerating Shields, does a Croby Sharmor's or Langston Shell's
   armor count whole or half (A13)?
4. A Death Star's battle values: initiative capped at 63 with many
   Battle Nexus, weapon reach +1, jammer with five Jammer 50s (72 if
   the cap comes before the starbase's quarter, 73 if after), capacitor
   capped at 255% with six Flux Capacitors.
5. Repair at an Ultra Station or Death Star planet (dock: 100 a year).
6. Mine damage per engine on a multi-engine ship (D = (ships·per +
   shortfall)·engines), and shields worth less than half the damage
   (absorbed = shield·ships).
7. Mine damage on a Regenerating Shields race: are shields ×7/5 and
   armor parts halved, as in battle? Elegy uses the plain sums;
   OBJECTS.md does not say.

### CB-056 results

Predictions pushed before the run (elegy 6bc9046); CB-056 ran on two
pinned streams per start. Results and labels from the stars-elegy
record (PARITY.md "Round 10", COMBAT.md, OBJECTS.md at b0ca924).

| Q | Setup | Elegy predicted | Original | Status now | Test |
|---|---|---|---|---|---|
| 1 | A starbase at 499/500 takes a hit worth less than one unit of its armor | destroyed (A11) | 500/500, not destroyed; the next hit destroyed it (2 streams) | prediction REFUTED; fixed, MEASURED: a 500th step, destroyed only at total ≥ armor | `TestPredictionDeathStarDamageSteps` |
| 2 | A Gatling Neutrino Cannon stack (3 ships, 2 per slot, one Flux Capacitor) whose targets include an enemy starbase and ships with 0, 1 and 2 Beam Deflectors | every target hit, the ships 576 / 518 / 466 by deflector (A12) | the starbase (token 0), then three ships: 466, 576, 518 for deflectors 81, 100, 90 | CONFIRMED in one stream only (in the other the ships were never in range) | `TestGatlingDeflectorPerTarget` |
| 3 | A Regenerating Shields Battleship with 8 Croby Sharmor and 2 Tritanium | armor 2570, shields 672 (A13) | shields 672; armor within 2546–2581 (2 streams) | CONFIRMED; the Langston Shell not observed | `TestConfirmedCB056RegeneratingShieldsCroby` |
| 4a | A Death Star with five Jammer 50s | jammer 72 | 72 (2 streams) | CONFIRMED | `TestPredictionDeathStarTokenValues` |
| 4b | A Death Star with six Flux Capacitors | capacitor 255% | 255% (2 streams) | CONFIRMED | `TestPredictionDeathStarTokenValues` |
| 5 | A damaged fleet at its owner's Ultra Station or Death Star planet | fleets −100, the starbase −50 | 400 → 300, 300 → 250 (2 streams) | CONFIRMED | `TestPredictionRepairLargeDocks` |
| 6 | Five Battleships (4 Trans-Galactic Drives, 1 Mole-skin, 2 Superlatanium) in a heavy field; the scoop, BB+Tank and Tank+BB variants | 197, 237, 157/15, 62/37 | as predicted (66 fleets, 2 streams) | CONFIRMED | `TestConfirmedMineDamageLateFleets` |
| 7 | The same fleet, owner with Regenerating Shields | 197 (plain sums) | 280 (23 fleets) | prediction REFUTED; fixed, MEASURED: shields × 7/5 and armor parts halved, as in battle | `TestMeasuredMineDamageRegeneratingShields`; `TestPredictionMineDamageRegeneratingShieldsCroby` (a shield part's armor whole in mines, 258) |

Also observed: the Nexus Star token had initiative 63 and computer 99
(CONFIRMED), as `TestPredictionDeathStarTokenValues` gives for sixteen
Battle Nexus. The weapon initiative, the Death Star's reach and its 1500
armor were not measured and stay predictions.

### RS armor per slot (RG-002, stars-elegy #164, 41e7a1c)

The halving under Regenerating Shields truncates once per armor slot:
`(count × part armor) / 2` per slot (stars-elegy COMBAT.md "Armor",
BINARY-ONLY). RG-002-Y6's battle at (1241,1313) measured it for one slot
(one stream): an RS Rogue with 2 Organic Armor in one slot took damage
whose recorded units (21 32 43 64 85 96 117) fit armor 625 only, where
Elegy's per-part halving (87 + 87) gave 624 and 42 for 43. Elegy #181
had labelled per-part truncation BINARY-ONLY in a test comment; that
label misread the public COMBAT.md sentence "each armor part counts
half" and had no private-side source. Per slot against the parts' total
halved once is BINARY-ONLY until an oracle design with an armor part in
each of two slots runs (174 against 175).

The halving is in the design's armor, so mine damage and stargate damage
use it too: the stargate's plain armor under RS (ASSUMPTION G5) is
replaced (BINARY-ONLY, OBJECTS-STATUS.md).

### CB-058 predictions

Pushed before lane F's run. Each is a PREDICTION from COMBAT.md and
OBJECTS.md as Elegy reads them; the tests are in
`engine/combat_cb058_test.go`. Values are Elegy's for a race with no
cost LRT.

| Q | Setup for the oracle | Elegy predicts | Current label | Test |
|---|---|---|---|---|
| 1 | A Destroyer (Long Hump 6, one Alpha Torpedo, no computer) fires at a Battleship with six Jammer 50 and no shields: hit chance 35·(100−95)/100 = 1% | when the torpedo misses, one hit record with 0 hits and 1 miss, and no damage | BINARY-ONLY | `TestPredictionCB058ZeroHitTorpedoRecord` |
| 2 | Three players, 100 one-ship fleets each (ids 0..99) at one deep-space point | quota 85; player 0's fleets 0 and 16..99 and players 1 and 2's 15..99 fight (255); all three get the left-out message | not observed for three players (CONFIRMED for two, CB-039) | `TestPredictionCB058TokenCapThreePlayers` |
| 3 | A fleet of 4 Large Freighters (2 Long Hump 6) with 1000 Fe, 700 Bo, 333 Ge, 500 kT colonists loses one freighter per kill event | cargo left 750/525/249/376, then 500/350/166/251, then 250/175/83/126, then none: each share truncated, the remainder 1 kT at a time to Fe, Bo, Ge, colonists | BINARY-ONLY (the fuel share CONFIRMED, CB-023) | `TestPredictionCB058CargoShare` |
| 4 | Player 0 destroys all of player 1's ships at two deep-space points in one turn, the second point's fleets earlier in the fleet table | the salvage at the point of player 0's lower fleet is player 1's number 0, the other number 1 (battles by each location's first fleet; lowest free number) | BINARY-ONLY | `TestPredictionCB058SalvageTwoPoints` |
| 5 | An RS player's Rogue with 3 Langston Shell in slot 1 (a shield-or-armor slot); a Battleship with 3 Langston Shell in slot 2 (shield only); a Rogue with 3 Organic Armor in slot 1. Each has Trans-Galactic Drives in slot 0 | 645, 2195, 712; shields 525. Elegy halves by part kind, so the Shell counts whole in either slot. Halving by slot kind would give the first Rogue 547 | BINARY-ONLY (Croby Sharmor in a shield-only slot CONFIRMED, CB-056) | `TestPredictionCB058LangstonShellRegeneratingShields` |
| 6 | A Death Star with 8 Heavy Blasters (range 3) | armor 1500; reach 4: a ship 4 squares away is a target, one 5 away is not | BINARY-ONLY (not measured in CB-056) | `TestPredictionCB058DeathStarReach` |

Q5 separates the two readings of "armor slot": the Rogue's slot 1 takes
shields or armor, the Battleship's slot 2 only shields. If the Rogue's
armor is 547, the halving goes by slot kind; if 645, by part kind.

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
