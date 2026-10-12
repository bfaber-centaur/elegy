# Engine completion sprint

The live coordination record for the sprint that started on 2026-10-12.
The coordinator updates it as work merges. It records claims with their evidence;
the status documents (`docs/*-STATUS.md`) stay the authority on what each
subsystem implements.

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

## Now (verified at main 4612da1, stars-elegy 9a355fe, 2026-10-12)

- `go test -skip '^TestLongGames$' ./...`: every package passes.
- `go test -run TestParityVectors -v ./engine`, over 31 corpora: 982
  pass, 173 measured pass, 25 sample-only, 8 fail (RG-001 7, CO-07b 1),
  2 differs (FO-03-E, FO-06-G), 569 skip, and 7 random cases (5 pass on
  the reference seed, 2 fail on it: RG-001-Y3-2432 and TK-120-A). MF-02
  is no longer random. Expectations skipped include 4,743 from other
  oracle streams and 360 samples of one stream's random outcome.
- `go test -run TestBattleRecords -v ./engine` (RANDOM-PARITY.md "Battle
  records"): every record with hits in cb, cb7, cs and rg replayed whole
  (13 of cb's 400 records have none). Compared
  hits (before any answered roll) match 12,561 of 12,561; conditioned hits
  (after one) match 952 of 952. Conditioned is not random parity.
- Nightly long games: not re-checked for this refresh.
- Open pull requests: #153 (lane B, K10) and this one.

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

State at 4612da1:

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
reference seed in 3 of 6 years; the chained run passes its first year
only (details below).

## Critical path

RG-001 diffs → fixes by the owning lane → longer RG recording (10 years)
→ the original generator in the harness (P1-1, waiting on Bobby's
seed-table decision) for exact random outcomes.

## Gaps

**P0 (blocks end-to-end execution):** none found.

**P1 (blocks representative multi-turn parity):**

1. The harness cannot draw the original's random stream. Skipped: 4,743
   expectations from other streams and 360 samples. `oraclerng` holds the
   seam (#133); RANDOM-PARITY.md counts exact random parity at 0. The
   generator's seed-table publication is pending Bobby.
2. RG-001's open diffs (below): waypoint fleet targets in 2434–2435 and the
   chained run's mineral drift.
3. RG-001 has 6 recorded years; milestone 2 needs 10.
4. Oracle computer players' orders must replay as vector orders; the
   capture-to-vector conversion is not checked for every order kind.

Done since the start: the representative vector (#145, #149), the
importable loader (#134), `elegy vector` (#140), following fleets (#130).

**P2 (blocks substantial gameplay coverage):**

- Messages are not compared (864 expectations): Elegy's events have no
  mapping to MESSAGES.md kinds.
- Score records (422) and default queue settings (58) are not compared.
  Battle records are compared by `TestBattleRecords`, not by the vector
  harness, which still lists 95 `battle` and 82 `battle_actions` as
  skipped.
- Super Stealth research stealing is in the engine (#146), but the harness
  still skips its expectations (37; `internal/vectors`, "Elegy does not
  steal research yet").
- Fleet-to-fleet transport and transport fuel (TAKEOVER-STATUS.md).
- Design and design-delete orders in vectors; queue items 10, 13 and
  18–27 in the harness.
- Views: `other_players`, `player`, `design`, and the estimate fields.
- Computer players: A7, A23, A41, A46 (AI-STATUS.md).

**P3 (rare or cosmetic):** FO-03-E and FO-06-G (ruleset choice), the
duplicate-serial penalty, the 32766 and 32765 caps. The −300 overcrowding
floor now has a test (#154).

**Stale documents:** KERNEL-STATUS.md "Not modelled yet" still lists Super
Stealth research stealing, merged in #146. ORDERS-STATUS.md "Not
modelled" still lists loads.

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

stars-elegy: fda4173 #138 (move draws), de2836a #141 (fleet numbers),
30272be #143 (concentration fractions; FN, WF, CB-053, GR-ORDER cases),
2f214f5 #142 (research level-up order, Super Stealth stealing conditions,
capture tech attempts), ee47fff #144 (CO-07b), ac792b0 #145 (homeworld
in start states), 8384191 #146 (CB-054), 397ebfc #147 (non-GR level-up
passes), 7c06f47 #148 (ancient artifacts on start-state planets),
9a355fe #149 (an emptied planet without mines or factories loses its
carry at the year boundary; BINARY-ONLY, MEASURED in RG-001 planets 50
and 87). Also merged: 77b282a #139, 52af513 #140.

Elegy's RG vectors are synced from stars-elegy 2f214f5 plus the homeworld
records of 8384191; 7c06f47 and 9a355fe are not synced yet.

## Known parity failures

- FO-03-E, FO-06-G: differ only because `Legacy.MergeOverflow` is off in
  `elegy`.
- CO-07b: fails on surface minerals and research (stars-elegy ee47fff).
  Lane A has the queue strip queued (coordinator triage).
- TK-120-A: random, 6 of 8 seeds, fails on the reference seed
  (environment). Not looked at in this sprint.

**RG-001 at 4612da1.** A first subset of the representative game, not
complete parity. Field counts from `elegy vector` on the reference seed
(variant 0); `sample` is a missed sample, which counts as skipped:

| Vector | Cases | Pass | Differ | Sample | Skip |
|---|---|---|---|---|---|
| `rg001` (chained 2430–2435) | 1 pass, 5 fail | 6,069 | 189 | 23 | 320 |
| `rg001-y1` (2430) | pass | 993 | 0 | 2 | 51 |
| `rg001-y2` (2431) | pass | 1,017 | 0 | 2 | 40 |
| `rg001-y3` (2432) | fail | 1,025 | 1 | 6 | 52 |
| `rg001-y4` (2433) | pass | 1,069 | 0 | 0 | 48 |
| `rg001-y5` (2434) | fail | 1,082 | 4 | 2 | 67 |
| `rg001-y6` (2435) | fail | 1,077 | 8 | 2 | 53 |

Over 8 seeds (`TestParityVectors`) Y1, Y2 and Y4 and chained 2430 are
random cases that pass on the reference seed; Y3 is random and fails on
it; Y5, Y6 and chained 2431–2435 fail. RG-001's battles: 16 records, all
replayed whole; compared hits 28 of 28, conditioned 50 of 50.

Restart mismatches (each year starts from the original's state):

- Y5, Y6: a waypoint aimed at another player's fleet loses its target
  (Elegy keeps a plain waypoint; the oracle keeps the fleet target, type 2,
  ids 100014 and 100048). 2 waypoints in Y5, 6 in Y6. stars-elegy PARITY
  "A waypoint aimed at another player's fleet (WF-01..05)", MEASURED,
  which contradicts ORDERS.md. Lane A.
- Y3 planet 92 and Y6 planet 95: `original_environment` off by one in one
  axis. Not looked at.
- Y5: player 6 research energy 1978, want 1977; planet 35's queue item is
  [8 1 8 7], want [8 1 11 7]. Not looked at.
- Y6: salvage at (1324, 1257) missing (want [3 3 3]). Not looked at.

Chained-only mismatches (Elegy's own state carried from year to year):

- Surface minerals drift on 149 planet-years, growing from 13 in 2431 to
  42 in 2435. Not looked at; random outcomes from earlier years feed it.
- Player 1 research and energy tech from 2431, while the restarts match
  them. Coordinator triage: artifact gifts, lane A; not checked here, and
  the synced vectors carry no artifact records yet.
- Planet 50's `excess` in 2434–2435: matches stars-elegy 9a355fe's
  emptied-planet carry rule (MEASURED in RG-001 planet 50). Economy.
- Other `excess` and population differences (planets 40, 119), planet 40
  environment and queue, planet 41 queue, planet 86 defenses, players 5
  and 6 research, and fleet 14 of player 2's position and fuel in 2435.
  Not looked at.

Comets: RG-001 has none; lane F was asked for an oracle comet case
(coordinator triage).

## Next actions

1. Lane A: keep a waypoint's fleet target while its owner sees the target
   (WF); CO-07b's queue strip; the artifact rule (stars-elegy 7c06f47).
2. Lane B: sync the vectors from stars-elegy 9a355fe (artifacts, the
   emptied-planet carry); have the harness compare Super Stealth stealing.
3. Economy: the emptied-planet carry rule; then the chained mineral drift.
4. Lane F: an oracle comet case; extend RG-001 toward 10 years.
5. Lane G, then D: publish the original generator as a public rule once
   Bobby decides on the seed table.
6. Lane A (for E): map events to MESSAGES.md kinds and compare messages.
