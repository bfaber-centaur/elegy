# Scanning status

Per-player knowledge in `engine/scanning.go` follows the public
specification stars-elegy `docs/SCANNING.md` and the measured data in
`docs/PARITY.md` "Scanning" (SC-001..SC-034), on stars-elegy `main` at
`9569af3`, which includes the implementer answers merged in PR #26 and
the SC-024..SC-034 batch merged in PR #50. Nothing else was used.

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
| Co-location | `coLocated` (switch `legacyColocation`) | LEGACY BUG, CONFIRMED | `TestConfirmedColocation` |
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

## Not modelled

Elegy's game has none of these yet, so their rules are not implemented:

- space-object visibility (SCANNING.md "Space objects", CONFIRMED by
  OB-011..OB-018): minefields, wormholes, mineral packets, the Mystery
  Trader, PP packet scanners, SD minefield detection, and owners made
  known through them. Elegy has none of these objects yet (stars-elegy
  `OBJECTS.md`);
- IT gate scans (no stargates), remote-mining reports (no remote mining);
- planet reports after minefield hits or a lost planet;
- design disclosure after SD hits and PP catches;
- chase retargeting and patrol target choice (orders that depend on
  sight);
- original environment, artifacts and terraformed flags in reports;
- public scores, messages that make a player known, and the client's
  report history.

## Choices where SCANNING.md is silent

- A player becomes known through a seen fleet or any report of an owned
  planet.
- The heading's warp is the warp of the waypoint the fleet moved toward
  in its last movement step.
- S1 and S2 were answered by stars-elegy #57 (open; BINARY-ONLY,
  predictions SC-035 and SC-036). The bombing check is taken at the
  bombing step and the report written from the end-of-year state
  (`bombChecks`, `TestPredictionBombingCheckAtBombingStep`). Every other
  battle participant, allies included, gets the designs in full and
  becomes a known player.
- Penetration needing `P > 0` and the estimate timing (end of the turn,
  after every battle draw) are now spec (#26).
