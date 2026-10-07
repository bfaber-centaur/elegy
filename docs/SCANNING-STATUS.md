# Scanning status

Per-player knowledge in `engine/scanning.go` follows the public
specification stars-elegy `docs/SCANNING.md` and the measured data in
`docs/PARITY.md` "Scanning" (SC-001..SC-023), on stars-elegy `main` at
`df57443`, which includes the implementer answers merged in PR #26
(`d12dd50`). Nothing else was used.

`Views(game, estimates)` is a pure function from the post-turn game to
one `PlayerView` per player. `PopulationEstimates(game, rng)` makes the
year's shared population estimates. `GenerateTurn` runs both last and
returns the views in `TurnResult.Views`. Views are plain data; no file
encoding is implied.

Scanner ranges, cloak points and the planetary scanner catalogue are
caller-supplied (`Part`, `Hull.JOATScanner`, `Game.PlanetScanners`,
`Game.Defenses`) until
the public component table lands.

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
| Tachyon detectors (0–2 on one design) | `tachyonFactor`, `scanners` | CONFIRMED (more, or across designs, BINARY-ONLY) | `TestConfirmedTachyonDetectors` |
| Co-location | `coLocated` (switch `legacyColocation`) | LEGACY BUG, CONFIRMED | `TestConfirmedColocation` |
| Enemy fleets at the viewer's planets | `view` | BINARY-ONLY | `TestPredictionFleetsAtOwnPlanets` |
| Orbit report levels (none / Bat / Robber Baron) | `view` | CONFIRMED | `TestConfirmedOrbitReports` |
| Starbase cloak | `starbaseCloak`, `view` | CONFIRMED (ISB, SS, 25,000 cap BINARY-ONLY) | `TestConfirmedStarbaseCloak`, `TestPredictionSuperStealthAndStarbaseBonus` |
| SS fleet cloak | `fleetCloak` | BINARY-ONLY | `TestPredictionSuperStealthAndStarbaseBonus` |
| Population estimate (one draw per populated planet, rand(0) included) | `PopulationEstimates` | BINARY-ONLY | `TestPredictionPopulationEstimate` |
| Defense coverage estimate | `defenseEstimate` | BINARY-ONLY | `TestPredictionDefenseEstimate` |
| Every report shows owner, homeworld and starbase; orbit makes the owner known | `report`, `view` | BINARY-ONLY | `TestPredictionPositionReportStarbase` |
| Fleet heading (this year's move, halved to ±127) | `scanHeading`, movement `place` | BINARY-ONLY | `TestPredictionHeading` |
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
- planet reports after battle, bombing, minefield hits or a lost planet;
- design disclosure after battle, SD hits and PP catches;
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
- Penetration needing `P > 0` and the estimate timing (end of the turn,
  after every battle draw) are now spec (#26).
