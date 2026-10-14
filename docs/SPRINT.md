# Engine completion sprint

The live coordination record for the sprint that started on 2026-10-12.
The coordinator updates it as work merges. It records claims with their evidence;
the status documents (`docs/*-STATUS.md`) stay the authority on what each
subsystem implements.

## Engine-completion checklist (release candidate, main `f09acd9`)

Three columns:
- **Playable:** a person can do every step of a game from the command line with readable output, and a saved game reloads and continues identically. This column says nothing about fidelity.
- **Behaviorally faithful:** the mechanics match the original's, by the behavioral record and the parity corpora.
- **Bit-exact RNG parity:** the same draws in the same order as the original's generator.

Each cell is done, partial or missing, with its evidence. The coverage numbers behind the faithful column are in [COVERAGE.md](COVERAGE.md). The playable rows at the end come from the "Playable vertical slice" thread's capability audit on main `7e6be58`, updated for its #266.

| subsystem | playable | behaviorally faithful | bit-exact RNG parity |
|---|---|---|---|
| turn order, kernel | done: `GenerateTurn`; `TestSmokeDeterministic`; long games with replay and reload | partial: every modelled step is in order (KERNEL-STATUS.md "Turn order implemented"); 33 ASSUMPTION and 20 BINARY-ONLY lines | partial: draw order checked (`TestConfirmedYearDrawOrderKX004`); player shuffle BINARY-ONLY; exact parity 0 |
| production, research, economy | done: queues, default queue, research orders (ORDERS-LAYER-STATUS.md) | done for RG: 0 deterministic misses in RG-002/003/004. Partial elsewhere: Trader items and queued items lost with a starbase are not modelled (COMBAT-STATUS.md "Not implemented") | missing: the mining +1 draw is Elegy's (economy's named draw, planet 102) |
| population, growth | done | partial: overcrowding floor and IS breeding tested. RG restarts match strictly or random-driven | missing |
| mining, terraform | done (TERRAFORM-STATUS.md "Turn wiring") | partial: kx, ob, cs corpora pass; ASSUMPTION T1–T3, R1; remote-mining reports not modelled | missing: PP packet terraform samples match 3 of 24 |
| movement, fuel, waypoints | done: order file v1, follow, chase | partial: fm corpus passes. Patrol, transfer-fleet and repeat are not accepted in waypoint orders; a waypoint can't target a wormhole end (AI-STATUS.md A46); transport fuel actions are not modelled | partial: movement tie draws checked only as allowed (`TestBattleRecords`) |
| fleets: merge, split, cargo | done between own fleets | partial: merge rules CONFIRMED. Fleet-to-fleet transport, loads from another player's fleet and Robber Baron stealing are missing (38 expectations, COVERAGE.md) | n/a |
| combat | done | partial: 12,561 of 12,561 hits compared and recorded moves all match; 9 NOT RUN (COMBAT-STATUS.md "Not tested against the oracle") | missing: torpedo rolls only conditioned |
| takeover, invasion, bombing | done | partial: bombing and ground combat CONFIRMED; GC-1 with several attackers NOT RUN; 12 ASSUMPTION | missing: bombing roundings not replayed |
| scanning, reports | done: `game.Report`, history, known designs | partial: sc corpus passes, including its 156 viewer-report expectations (compared since 2026-10-14); 216 client-estimate expectations not compared (milestone 3) | missing: population estimates use Elegy's stream (ELEGY CHOICE) |
| minefields, packets, wormholes, salvage, Mystery Trader | done (OBJECTS-STATUS.md "Turn wiring") | partial: 44 BINARY-ONLY lines; minefield messages unmapped (35 expectations); battles never award Trader items | missing: mf never matches its random outcomes; ob and wt are samples |
| races | done: `elegy race`, race file v1 | partial: RD-P1..RD-P21 pass; ASSUMPTION B1 | missing: random race uses Elegy's stream |
| components, designs | done: catalog, design orders | partial: verbatim component table; design orders not in the vectors | n/a |
| computer players | partial: Robotoid, Rototill and Cybertron play long games; unsupported steps are reported (`ai.Result.Unsupported`) | missing: no oracle comparison of any AI decision (AI-STATUS.md "Implemented"); 17 BINARY-ONLY lines | missing: draw counts follow the spec, no parity claimed |
| messages | partial: events in the turn result and report | partial: compared ids match; 63 ids unmapped (148 expectations with sequence entries, COVERAGE.md) | n/a |
| universe generation | done: `elegy new`, sizes, densities, built-in races | partial: ug vectors; planet and computer names are PLACEHOLDERs (UNIVERSE-STATUS.md "Not modelled") | missing: `newgame.NewRand` is an ELEGY CHOICE |
| save and load | done: save v12, `Hash`, `Diff`, reload in long games | n/a: Elegy's own format, not the original's files (ELEGY CHOICE) | n/a: the generator state is saved |
| create and load a game | done: `elegy new`/`race`/`-ai`, both rulesets (`TestCLIGame`) | n/a | n/a |
| player-scoped report | done: `game.Report` (`TestReportSurvivesSaveLoad`) | see scanning | n/a |
| turn with humans and computer players | done: order files, `elegy turn`, library clients through `game.Driver` | see above | see above |
| save and reload continues identically | done: save v12 (`TestSmokeSaveLoadContinues`, `TestDriversSaveReload`) | n/a | n/a |
| readable inspection (planets, fleets, designs, research, messages) | done: `elegy show` (#266, `TestShow`, `TestShowCommand`) | n/a | n/a |
| order authoring by name with dry-run validation | done: `elegy order` and `elegy check` (#266, `TestOrderKinds`, `TestOrderErrors`) | n/a | n/a |
| readable event names, end-to-end demo and docs/PLAYING.md | done: #266 (`TestCLIPlayableSlice`, docs/PLAYING.md) | n/a | n/a |
| ship design and battle plan authoring by name | done: `elegy order design`, `delete-design`, `plan`, `delete-plan`, `fleet-plan` and `show plans` (#268, `TestOrderDesigns`, `TestOrderPlans`) | n/a | n/a |
| derived numbers (resources a year, maximum population, research cost, range) | missing, deferred: not needed to play | n/a | n/a |
| graphical or terminal front end | missing, by choice for now (milestone 3) | n/a | n/a |

Overall: **playable** is done for the vertical slice: every step works from the command line with readable output (#266), including ship design and battle plan authoring by name (#268). Derived numbers and a front end remain. **Behaviorally faithful** is partial: the corpora show 0 known discrepancies, inside the coverage limits COVERAGE.md lists. **Bit-exact RNG parity** is missing, because the original's generator is not in the harness (P1-1).

## Starting point (verified at main ba7a03e, 2026-10-12)

- `go test -skip '^TestLongGames$' ./...`: every package passes.
- `go test -run TestParityVectors -v ./engine`: 979 pass, 166 measured
  pass, 25 sample-only, 0 fail, 2 differs, 2 random (1 failing on the
  reference seed), 579 skip, over 29 corpora. This is unchanged from the
  pause snapshot ([PAUSE-2026-10-08.md](PAUSE-2026-10-08.md)).
- Nightly long games (96 games of 150 years, saved and reloaded at 77):
  passed on ba7a03e on 2026-10-09, 10 and 11. The only failure is the
  first run, on 57aed2c (2026-10-08).
- Open pull requests: elegy #130 (following fleets, held) only.

The seven engine requirements, on main today:

| Requirement | State | Evidence |
|---|---|---|
| Phases in order | Met for every modelled step | `engine/turn.go` `GenerateTurn`; KERNEL-STATUS.md "Turn order implemented"; `TestConfirmedPGHomeworldTurns` |
| Deterministic transitions | Met | `TestSmokeDeterministic`; nightly replay of each game |
| Planets, fleets, populations, economy, research, combat interact | Met for what is modelled | parity corpora pg, pq, kx, fm2, cb, tk, ob, wu |
| Orders | Met, with gaps | `YearOrders`; gaps below (follow, fleet-to-fleet loads) |
| State persists across turns | Met | `TestSmokeSaveLoadContinues`, `TestDriversSaveReload`, nightly reload |
| Reproducible | Met | as above; `TestRulesetsCoexist` |
| Compared with the oracle | Partly | 1,754 vector cases, but only small targeted states of 1–3 years (pg: 1 homeworld, 29 years) |

Elegy already runs complete turns without the original executable. What
it cannot yet show is that a **representative** game, with several
players and many objects, comes out of a year the way the original's does.

## Now (verified at main 71f70f9, stars-elegy 56ebfcf, 2026-10-12)

- `go test -skip '^TestLongGames$' ./...`: every package passes.
- `go test -run TestParityVectors -v ./engine`, over 31 corpora: 981
  pass, 175 measured pass, 25 sample-only, 5 fail (chained RG-001
  2431–2435), 2 differs (FO-03-E, FO-06-G), 569 skip, and 10 random
  cases. 6 of those pass on the reference seed (CB-009, OB-031-A,
  RG-001-Y1, Y2, Y4 and chained 2430) and 4 fail on it (RG-001-Y3, Y5,
  Y6 and TK-120-A). Expectations skipped include 4,836 from other oracle
  streams and 358 samples of one stream's random outcome.
- `go test -run TestBattleRecords -v ./engine` (RANDOM-PARITY.md "Battle
  records"): every record with hits in cb, cb7, cs and rg replayed whole
  (13 of cb's 400 records have none).
  - Compared hits (before any answered roll): 12,561 of 12,561 match.
  - Conditioned hits (after one): 952 of 952 match. Conditioned is not
    random parity.
  - Recorded moves that Elegy's rule allows with some outcome of its tie
    draws: 9,111 of 9,111 checked; 6,462 unchecked (mostly CB-042 and
    CB-044).
- Nightly long games: not re-checked for this refresh.
- Open pull requests: this one.

## Milestone 1: one representative year against the oracle

Done when all of these hold:

1. A representative oracle game exists as a parity vector (RG-001): at
   least 3 players, a few dozen fleets and planets, production, research,
   movement, colonizing and, if the game reaches one, a battle. Its
   orders for the year are carried in the vector.
2. `go run ./cmd/elegy vector <file>` (new) loads a vector's start,
   generates its years and prints every compared field with its result
   and every skip with its reason. The loader leaves test code for a
   package other code can import.
3. RG-001's year is compared field by field. Each mismatch is fixed or
   listed under "Known parity failures" with a cause or "not looked at".

State at 71f70f9:

1. Met. RG-001 (stars-elegy `vectors/rg/rg001.json`, MEASURED): seven
   players, 97 fleets, 93 of 128 planets owned, six chained years
   2429→2435 with 8 battles; every player's orders are in the vector. In
   CI since #145 (`engine/testdata/vectors/rg/`), with one-year restarts
   `rg001-y1` to `rg001-y6` since #149.
2. Met. The loader is `internal/vectors` (#134); `elegy vector` (#140)
   prints every field as pass, differ, sample or skip, with skip reasons.
3. Met as listing: every mismatch on the reference seed is under "Known
   parity failures" below, with a cause or "not looked at". Not every
   mismatch is fixed.

This is not full parity. Random outcomes from Elegy's own generator are
not the original's, so they compare only as samples; battle hits after a
torpedo roll are conditioned on the record, not compared.

## Milestone 2: representative multi-turn parity

RG-001 compared for at least 10 consecutive years, two ways: reset to the
oracle's state each year, and run on from Elegy's own state. Exact random
outcomes need the harness to draw from the original's generator (P1-1).

State: both ways run, for 6 years, not 10. The restarts pass on the
reference seed in 3 of 6 years, and every restart is now a random case
(it passes on some seeds); the chained run passes its first year only
(details below). RG-002, a late-game start, is in with every order loaded
and a committed discrepancy ledger checked exactly in CI. On main 8426a6a plus #194
(which compares packet settings) its reference seed has one deterministic
discrepancy, Y4 planet 88's packet warp (missing mechanic, economy), 30 field
lines are random-driven, and one order is refused only on seed variants 1 and
6 (incorrect mechanic, lane A); see
[PARITY-STATUS.md](PARITY-STATUS.md) and [RG002-REPORT.md](RG002-REPORT.md).

## Critical path

RG-001's remaining diffs, all random-driven or not looked at → RG-002 and
longer recordings
→ the original generator in the harness (P1-1, waiting on Bobby's
seed-table decision) for exact random outcomes.

## Gaps

**P0 (blocks end-to-end execution):** none found.

**P1 (blocks representative multi-turn parity):**

1. The harness cannot draw the original's random stream. Skipped: 4,836
   expectations from other streams and 362 samples. `oraclerng` holds the
   seam (#133); RANDOM-PARITY.md counts exact random parity at 0. The
   generator's seed-table publication is pending Bobby. Every chained
   surface-mineral mismatch in RG-001 is random-driven (below).
2. RG-001 has 6 recorded years; milestone 2 needs 10. RG-002 adds 6 late-game
   years ([RG002-REPORT.md](RG002-REPORT.md)).
3. Oracle computer players' orders must replay as vector orders. RG-001
   rejects none of its orders, chained or restarted (#157); the
   capture-to-vector conversion is not checked for every order kind.

Done since the start: the representative vector (#145, #149), the
importable loader (#134), `elegy vector` (#140), following fleets (#130),
waypoints aimed at another player's fleet (W5, #167).

**P2 (blocks substantial gameplay coverage):**

- Messages: the harness compares the ids it can map from Elegy's events,
  and the ids Elegy sends no event for (PARITY-STATUS.md "Messages
  compared"). Other ids are skipped until they are mapped.
- Fields the harness still skips: default queue settings (58), then
  `score_record` (422), are lane B's next. The null fields (`defenses`
  103, `planetary_scanner` 2, `population` 1) are skipped too.
  Concentrations, `research_field` (#166) and `starbase` (#169) are
  compared now.
  Battle records are compared by `TestBattleRecords`, not by the vector
  harness, which still lists 95 `battle` and 82 `battle_actions` as
  skipped.
- Super Stealth research stealing is in the engine (#146). The harness
  now skips its expectations only when another player's spending is
  itself unsure (3 left; `internal/vectors` `unsureSpender`).
- Fleet-to-fleet transport and transport fuel (TAKEOVER-STATUS.md).
- Design and design-delete orders in vectors; queue items 10, 13 and
  18–27 in the harness.
- Views: `other_players`, `player`, `design`, and the estimate fields.
- Computer players: A7, A23, A41, A46 (AI-STATUS.md).

**P3 (rare or cosmetic):** FO-03-E and FO-06-G (ruleset choice), the
duplicate-serial penalty, the 32766 and 32765 caps. The −300 overcrowding
floor now has a test (#154).

**Stale documents:** ORDERS-STATUS.md "Not modelled" still lists loads.

## Lanes

A lane changes only the files it owns. `engine/turn.go` and the shared
state types (`Game`, `Planet`, `Fleet`, `Player`) change only through
lane A, and a change to saved state bumps the save version. Engine files
of lanes without a thread go through lane A.

| Lane | Scope | Files | Owner (thread) |
|---|---|---|---|
| A Integration, fleets and orders | turn order, waypoints, orders replay, movement, cargo, takeover, artifacts, diff triage | `engine/turn.go`, `waypoints.go`, `orders*.go`, `fleetops.go`, `fleetships.go`, `movement.go`, `launch.go`, `takeover.go`, `scrap.go`, `invariants.go`, `ruleset.go`, `catalog.go`, `objects/` | Engine integration and parity |
| B Game loop and save/load | vector loader, `elegy vector`, vector sync, game loop | `internal/vectors/`, `engine/*parity*_test.go`, `cmd/elegy/`, `game/`, `newgame/`, `regress/` | Game loop and save/load |
| Economy | population, growth, mining, production, research, terraforming, breeding | `engine/population.go`, `economy.go`, `mining.go`, `production*.go`, `research.go`, `breeding.go`, `habitability.go`, `randomevents.go`, `terraform*.go`, `terraform/`, `races/` | Economy |
| D Combat and random stream | battles, battle records, the original's generator in the harness | `engine/combat*.go`, `battleplan.go`, `ships.go`, `oraclerng/`, docs/RANDOM-PARITY.md | Original random numbers in the parity harness |
| E Output | views, messages, scores, reports | `engine/scanning.go`, `scores.go`, new `engine/messages.go`, `game/report.go`, `game/history.go` | none yet; through lane A |
| F Oracle recording | RG corpus, differential cases, vector sync | stars-elegy `vectors/`, `tools/vectors/`; never pushes to elegy | Record a representative game from the original |
| G Research | private-repo questions; public answers only | none in elegy | AI spec answers and errata |

`ai/` is frozen this sprint ("Computer opponents" is idle) except for
fixes to a long-game failure. The coordinator keeps this file current.

## Blockers

- Bobby: publishing the original generator's seed table (P1-1).

## Merged work

elegy (merge commit, PR):

- e232a23 #132: this coordination record.
- 56394e4 #130: following fleets, step 1a.3, messages 0x137 and 0x138.
- eb2192c #135: a destroyed starbase leaves no salvage (CB-047).
- ad91e6d #133: `oraclerng` seam to replay the oracle's stream; per-corpus
  random parity in RANDOM-PARITY.md.
- 6a779d4 #134: the vector loader moves to `internal/vectors`.
- 2a2e7ec #137: an owned planet with no colonists is lost when growth runs.
- 7042896 #136: research, the level-26 switch under "same field".
- 6199ee4 #141: the harness reports every mismatch; no case is skipped
  whole.
- 8821ff3 #139, a677b68 #148: battles compared with the original's battle
  records, at each year's battle phase.
- 68fddb7 #140: `elegy vector`.
- a6818dc #143: Inner Strength colonists breed in transit.
- 1254057 #144: Cheap Engines stall and warp-10 ship losses.
- dfcda8f #142: battles cost designs at the start-of-year tech.
- e409f79 #145: RG-001 in CI.
- 8f402d7 #147: a fleet that goes away keeps its number until the files
  are written.
- edc02c1 #146: Super Stealth research stealing.
- 1777670 #149: vectors synced from stars-elegy 2f214f5 (RG-001 restarts,
  homeworld flags, concentration fractions).
- 043ee9e #151: research checks fields in field-order passes.
- 1747dba #150: homeworld re-mark each year from the players' records
  (save version 8).
- 4a52f0e #152: torpedo rolls answered from the battle record (conditioned
  tally); a separate torpedo miss record.
- 4612da1 #154: a test of the overcrowding floor.
- 5b5e2ea #153: a part in a slot that does not take it is dropped at the
  end of the year (K10).
- da5b3f6 #138: this record refreshed to 4612da1.
- 9347d9e #155: an emptied planet without installations loses its carry
  at the year boundary.
- a5a5367 #156: every reader of a design's parts uses `Design.Parts`.
- 2de74b8 #157: RG-001 rejects no orders, chained and restarted.
- c8565c4 #159: RG-001's packet terraforming, conditioned on the draws.
- 5c63346 #160: prediction for target choice at a dropped part's
  next-year cost, pushed before CB-055 ran.
- 210b896 #158: deleting the starbase design in use strips the planet's
  queue, as a battle loss does (CO-07b).
- 2f05a84 #161: KX-004 S11's comet strikes, conditioned on the draws.
- 93820bb #162: a colony ship's minerals priced at the tech from before
  the year's research.
- 8856326 #163: salvage's skip-decay mark (`fresh`) loaded; vectors
  synced from stars-elegy d86c4f8.
- 395bcdc #164: dropped-part target choice re-cited to CB-055
  (CONFIRMED).
- 23a18e7 #168: this record refreshed to 395bcdc; K10 relabelled.
- 6d98b45 #165: remote mining refuses the orders the spec refuses.
- 589a12f #166: planet concentrations and the players' research fields
  compared.
- 5ea1c74 #169: a planet's starbase compared.
- 79f3cdc #170: a fleeing (tactic 0) token counts itself on its own
  square; recorded battle moves checked; late-game combat coverage.
- 641c9d5 #171: no lead flag without a second player in the game.
- 01a2906 #167: a waypoint aimed at another player's fleet keeps a seen
  target and retargets an unseen one (W5).
- f2b347a #172: a vector's victory conditions and public scores loaded.
- 71f70f9 #173: CB-056 predictions (late-game combat and mine damage).

stars-elegy: fda4173 #138 (move draws), de2836a #141 (fleet numbers),
30272be #143 (concentration fractions; FN, WF, CB-053, GR-ORDER cases),
2f214f5 #142 (research level-up order, Super Stealth stealing conditions,
capture tech attempts), ee47fff #144 (CO-07b), ac792b0 #145 (homeworld
in start states), 8384191 #146 (CB-054), 397ebfc #147 (non-GR level-up
passes), 7c06f47 #148 (ancient artifacts on start-state planets),
9a355fe #149 (an emptied planet without mines or factories loses its
carry at the year boundary; BINARY-ONLY, MEASURED in RG-001 planets 50
and 87), 4d3a3d9 #150 (KX-004 S11, comets on terraformed planets),
63a250d #151 (salvage's skip-decay mark is stored in the game file),
ec1b38a #153 (RG-001's mining +1 calibration; which tech prices a colony
ship), d86c4f8 #152 (salvage records carry `fresh`), b1e38ef #154 (CB-055,
target choice uses a dropped part's cost), e2fb8c1 #155 (remote-mining
check order and refusals, BINARY-ONLY), 56ebfcf #156 (packed ship counts
and end-of-year counting in the score record). Also merged: 77b282a #139,
52af513 #140.

apparatus: #69, the CB-055 oracle evidence (predictions pushed before
each round, run records and checksums), published as stars-elegy #154.

Elegy's vectors are synced from stars-elegy d86c4f8 (#163), which carries
7c06f47's artifact flags; Elegy ignores the flag until lane A's artifacts
PR.

## Known parity failures

- FO-03-E, FO-06-G: differ only because `Legacy.MergeOverflow` is off in
  `elegy`.
- TK-120-A: random, 6 of 8 seeds, fails on the reference seed
  (environment). Not looked at in this sprint.
- CB-009: random, 7 of 8 seeds, passes on the reference seed. Not looked
  at.

**RG-001 at 71f70f9.** A first subset of the representative game, not
complete parity. Field counts from `elegy vector` on the reference seed
(variant 0); `sample` is a missed sample, which counts as skipped:

| Vector | Cases | Pass | Differ | Sample | Skip |
|---|---|---|---|---|---|
| `rg001` (chained 2430–2435) | 1 pass, 5 fail | 6,199 | 179 | 27 | 196 |
| `rg001-y1` (2430) | pass | 1,014 | 0 | 2 | 30 |
| `rg001-y2` (2431) | pass | 1,032 | 0 | 2 | 25 |
| `rg001-y3` (2432) | fail | 1,047 | 1 | 6 | 30 |
| `rg001-y4` (2433) | pass | 1,089 | 0 | 0 | 28 |
| `rg001-y5` (2434) | fail | 1,111 | 2 | 2 | 40 |
| `rg001-y6` (2435) | fail | 1,102 | 1 | 3 | 34 |

Over 8 seeds (`TestParityVectors`) every restart is a random case: Y1 3,
Y2 6, Y3 5, Y4 7, Y5 4 and Y6 5 of 8 seeds pass, and Y3, Y5 and Y6 fail
on the reference seed. Chained 2430 passes on 3 of 8; chained 2431–2435
fail. RG-001's battles: 16 records, all replayed whole; compared hits 28
of 28, conditioned 50 of 50, recorded moves allowed 254 of 254.

Restart mismatches on the reference seed (each year starts from the
original's state):

- Y3 planet 92 and Y6 planet 95: `original_environment` off by one in one
  axis. Not looked at.
- Y5: player 6 research energy 1978, want 1977, and planet 35's queue item
  [8 1 8 7], want [8 1 11 7]. Random-driven, coordinator triage from the
  economy lane's analysis: one planet's mining +1 draw decides whether a
  queue item completes (planets 86, 41 and 35 across Y1 and Y5), and
  research matches on every seed where that planet's minerals match.
- Y5 over the 8 variants: the player 6 research and planet 35 queue pair
  above fails on variants 0 and 6, a minefield of player 2 on variant 4,
  and planet 39's concentrations (compared since #166) on variant 7.
  Random-driven by the coordinator's triage; the minefield is not looked
  at.
- Fixed since 395bcdc: the waypoint fleet targets in Y5 and Y6 (W5, #167).
- Y6 starbase damage on planets 54 and 55 (samples). #170 fixed a
  deterministic movement difference that kept the attacker away: a fleeing
  token did not count itself on its own square, so out of reach it stayed
  put (found by `TestBattleRecords`' move check; RANDOM-PARITY.md
  "Moves"). Since then, planet 55's Y6 starbase damage matches (sample
  pass). Planet 54's is still a sample miss, 380 vs 360, because of
  Elegy's own random walk. The replayed battles keep both values
  (`TestRG001StarbaseDamageKept`). In the chained run both are sample
  misses (54: 180 vs 360, 55: 0 vs 180).

Chained-only mismatches (Elegy's own state carried from year to year), all
in the random-driven group unless listed as not looked at:

- Surface minerals on 149 planet-years (13 in 2431, rising to 42 in 2435).
  - 140 are sums of mining's random +1 kT steps. MEASURED: stars-elegy
    ec1b38a, PARITY "Representative game (RG-001)"; over the restarts,
    the original's +1 came 584 times against 582 expected at `r/100`.
  - The other 9 (planets 86, 41, 40 and 71) follow from random outcomes
    too, planet 71 from mining ±1 as well. Coordinator triage, from the
    economy lane's analysis; not in a committed record.
- Player 1 research and energy tech from 2431, while the restarts match
  them: the ancient-artifact gift, whose field and amount are random
  draws. Coordinator triage; lane A's artifacts PR (pending) has a test
  conditioned on the oracle's gift draws, which clears 2433 and 2434.
- Players 0 and 2 research, off by 1–2 in 2434–2435. Coordinator triage,
  from the economy lane's analysis: a harness-only change in #158
  (333b4ac) unmasked them, with no change in Elegy's output; Super Stealth
  stealing matches the original on every restart, and the chained
  differences vary by seed (4, 4, 0, 8, 8, 8, 8, 2) because the thieves
  steal from spending that has drifted.
- Not looked at:
  - `excess` and population on planets 40 and 119;
  - planet 40's environment and queue, planet 41's queue and planet 86's
    defenses;
  - research of players 5 and 6;
  - planet 92's `original_environment` in 2432.

Comets: RG-001 has none. KX-004 S11's strikes are checked conditioned on
the draws (#161), which is not parity.

## Next actions

Per lane, as the coordinator set them on 2026-10-12:

1. Lane A: artifacts (the v5 switch, SaveVersion 9 and the conditioned
   planet-71 test), pending.
2. Lane B: load default queue settings, then compare `score_record`
   (stars-elegy 56ebfcf).
3. Lane F: RG-002 (`/mnt/project-files/rg002/plan.md`), the RM cases and
   CB-056's oracle run (predictions in #173).
4. Lane G: idle. The original generator as a public rule waits on Bobby's
   seed-table decision.
5. Lane A (for E): map events to MESSAGES.md kinds and compare messages.
