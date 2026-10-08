# Scanning status

Per-player knowledge in `engine/scanning.go` follows the public
specification stars-elegy `docs/SCANNING.md` and the measured data in
`docs/PARITY.md` "Scanning" (SC-001..SC-036), on stars-elegy `main` at
`63635f0`. Nothing else was used.

`Views(game, estimates)` is a pure function from the post-turn game to
one `PlayerView` per player. `GenerateTurn` also passes the year's
battle records (planet, players, designs on the board, fleets left out by
the size limit), which add battle reports and design disclosure. `PopulationEstimates(game, rng)` makes the
year's shared population estimates. `GenerateTurn` runs both last and
returns the views in `TurnResult.Views`. Views are plain data; no file
encoding is implied.

Scanner ranges, cloak points, JOAT hulls and the planetary scanner and
defense catalogues come from the component table: `Components().NewDesign`,
`Components().PlanetScanners()` and `Components().Defenses()` fill
`Design`, `Game.PlanetScanners` and `Game.Defenses`
(`docs/COMPONENTS-STATUS.md`). The game still takes them as data, so
tests can supply their own.

## Rules and tests

| Rule (SCANNING.md section) | Code | Status | Test |
|---|---|---|---|
| Integer range test `d² ≤ R²` | `seesFleet` | CONFIRMED | `TestConfirmedScanRangeEdges` |
| Scanners on one design combine `⌊⁴√Σr⁴⌋`; fleet uses its best design | `designScan`, `scanners` | CONFIRMED | `TestConfirmedCombinedScanners` |
| JOAT hull scanner 20·E / 10·E | `designScan` | CONFIRMED | `TestConfirmedJOATHullScanner` |
| NAS (ships and planets) | `designScan`, `planetScan` | CONFIRMED | `TestConfirmedNoAdvancedScanners` |
| Planet scanner = best allowed by current tech; P = S/2 | `planetScan` | CONFIRMED | `TestConfirmedPlanetScanners` |
| AR planet scanners from population | `planetScan` | BINARY-ONLY | `TestPredictionAlternateRealityPlanetScanner` |
| Penetration for planets and orbiting fleets | `seesFleet`, `view` | CONFIRMED | `TestConfirmedPenetrationAndOrbit` |
| Fleet cloak (mass-weighted points, cargo, table, bound) | `fleetCloak`, `cloakPercent`, `cloakBound` | CONFIRMED | `TestConfirmedFleetCloak`, `TestConfirmedPenetrationAndOrbit` |
| Tachyon detectors (0–3 on one design) | `tachyonFactor`, `scanners` | CONFIRMED (more, or across designs, BINARY-ONLY) | `TestConfirmedTachyonDetectors` |
| Co-location | `seesFleet` (ruleset switch `Legacy.Colocation`) | LEGACY BUG, CONFIRMED | `TestConfirmedColocation` |
| Co-location off: a fleet with no scanner sees no fleet, also at distance 0; the cloak bound applies there too | `seesFleet` | Elegy choice (the setting's stated meaning, RULESET.md) | `TestElegyDecisionColocationOff` |
| Enemy fleets at the viewer's planets | `view` | CONFIRMED (SC-024, SC-026) | `TestPredictionFleetsAtOwnPlanets` |
| Orbit report levels (none / Bat / Robber Baron) | `view` | CONFIRMED | `TestConfirmedOrbitReports` |
| Starbase cloak | `starbaseCloak`, `view` | CONFIRMED, SS +300 too (SC-030; ISB and 25,000 cap BINARY-ONLY) | `TestConfirmedStarbaseCloak`, `TestPredictionSuperStealthAndStarbaseBonus` |
| SS fleet cloak (+300, cargo left out) | `fleetCloak` | CONFIRMED (SC-030) | `TestPredictionSuperStealthAndStarbaseBonus` |
| Population estimate (one draw per populated planet, rand(0) included) | `PopulationEstimates` | range CONFIRMED (SC-024..SC-033); draw order BINARY-ONLY | `TestPredictionPopulationEstimate` |
| Defense coverage estimate (operable cap included) | `defenseEstimate` | CONFIRMED (SC-024..SC-033) | `TestConfirmedDefenseEstimateVectors`, `TestPredictionDefenseEstimate` |
| Every report shows owner, homeworld and starbase; orbit makes the owner known | `report`, `view` | CONFIRMED (SC-024..SC-026) | `TestPredictionPositionReportStarbase` |
| Fleet heading (this year's move, halved toward zero to ±127; kept on arrival) | `scanHeading`, movement `place` | CONFIRMED for straight moves (SC-027); chasers BINARY-ONLY | `TestConfirmedHeadingVectors`, `TestPredictionHeading` |
| Bombing check: normal report where the viewer would bomb | `bombCheck` | CONFIRMED for blind fleets without bombs (SC-024, SC-031, SC-032); mechanism BINARY-ONLY | `TestConfirmedBombingCheck` |
| Battle planet: at least position only for each participant | `battles`, `view` | CONFIRMED (SC-032) at a planet the fleet still orbits; otherwise BINARY-ONLY | `TestConfirmedBattleDisclosure`, `TestPredictionBattleRecords` |
| Left out of a battle by its size limit: normal report | `battles`, `view` | BINARY-ONLY | `TestPredictionLeftOutOfBattle` |
| Designs that fought the viewer disclosed in full, destroyed ones too | `battles`, `view` | CONFIRMED (SC-031) | `TestConfirmedBattleDisclosure` |
| Disclosure: partial designs, WM full, CA habitability, cargo scan | `view` | CONFIRMED | `TestConfirmedDisclosure` |
| Allies do not share | `Views` | CONFIRMED (one run) | `TestConfirmedAlliesDoNotShare` |
| Views at the end of the turn | `GenerateTurn` | BINARY-ONLY | `TestPredictionTurnViews` |

## Space objects, history and scores

These are implemented outside `scanning.go`:

- Space-object visibility (SCANNING.md "Space objects", CONFIRMED by
  OB-011..OB-018): `objects.Space` sees minefields, wormhole ends, packets
  and Traders before the views, with PP packet scanners, SD minefield
  detection and the owners made known through them
  (`objects/engine_adapter.go` `SeeObjects`, OBJECTS-STATUS.md).
  `PlayerView.Objects` lists what was seen. Tests: the OB-011..OB-018
  parity cases pass through the whole turn; the rules are in
  `objects/visibility_test.go` (`TestConfirmedMinefieldSight`,
  `TestConfirmedWormholeSight`, `TestConfirmedPacketAndTraderSight`,
  `TestConfirmedPacketScanners`, `TestConfirmedDemolitionSight`).
- IT gate reports (CONFIRMED OB-013, parity OB-013-A..C) and a PP
  player's view of a catching starbase's design (`EventPacketDesignSeen`,
  `TestPredictionPacketDesignSeen`), KERNEL-STATUS.md "Space objects".
- Patrol target choice at the end of the turn (KERNEL-STATUS.md, ORDERS.md
  Q3, BINARY-ONLY; `TestPredictionPatrolNeverRepeats`).
- Public scores: `PlayerView.Scores` holds the records the viewer may see
  (`scores.go`, `TestConfirmedPublicScores`).
- The report history: `game.Report.History` keeps every planet's latest
  report (GAME-LOOP.md, `TestHistoryKeepsLatestAndLostColonies`).

## Not modelled

- remote-mining reports;
- planet reports after minefield hits or a lost planet;
- design disclosure after SD hits;
- chase retargeting (an order that depends on sight);
- original environment, artifacts and terraformed flags in reports;
- messages that make a player known.

## Choices where SCANNING.md is silent

- A player becomes known through a seen fleet or any report of an owned
  planet.
- The heading's warp is the warp of the waypoint the fleet moved toward
  in its last movement step.
- S1 and S2 are now spec, CONFIRMED by SC-035 and SC-036. The bombing
  check is taken at the bombing step and the report written from the
  end-of-year state (`bombChecks`,
  `TestPredictionBombingCheckAtBombingStep`). Every other battle
  participant, allies included, gets the designs in full and becomes a
  known player.
- Penetration needing `P > 0` and the estimate timing (end of the turn,
  after every battle draw) are now spec (#26).
