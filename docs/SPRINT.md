# Engine completion sprint

The live coordination record for the sprint that started on 2026-10-12.
Lanes update it as work merges. It records claims with their evidence;
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

New in milestone 1: the representative vector, the command and the
importable loader. The turn, the order replay and the whole-year diff
machinery already exist.

This is not full parity. Random outcomes from Elegy's own generator are
not the original's, so they compare only as samples.

## Milestone 2: representative multi-turn parity

RG-001 compared for at least 10 consecutive years, two ways: reset to the
oracle's state each year, and run on from Elegy's own state. Exact random
outcomes need the harness to draw from the original's generator (P1-2).

## Critical path

RG-001 oracle run → RG-001 vector → loader and command → first diff →
fixes in lanes B–E. Lane A triages the diffs. The original generator for
the harness (P1-2) gates milestone 2, not milestone 1.

## Gaps

**P0 (blocks end-to-end execution):** none found.

**P1 (blocks representative multi-turn parity):**

1. No representative oracle game. Every corpus is a targeted 1–3 year
   experiment.
2. The harness cannot draw the original's random stream. Skipped: 4,298
   expectations from other streams and 327 samples. The generator is not
   in the public spec (KERNEL.md "Implementing").
3. The vector loader is test-only (`engine/parity_test.go`).
4. Following fleets, KERNEL.md step 1a.3 (#130, held on 3 review points).
5. Oracle computer players' orders must replay as vector orders; the
   capture-to-vector conversion is not checked for every order kind.

**P2 (blocks substantial gameplay coverage):**

- Messages are not compared (771 expectations): Elegy's events have no
  mapping to MESSAGES.md kinds.
- Score records (323), default queue settings (104) and battle records
  (61 `battle`, 48 `battle_actions`) are not compared.
- Fleet-to-fleet transport and transport fuel (TAKEOVER-STATUS.md).
- Design and design-delete orders in vectors; queue items 10, 13 and
  18–27 in the harness.
- Views: `other_players`, `player`, `design`, and the estimate fields.
- Computer players: A7, A23, A41, A46 (AI-STATUS.md).

**P3 (rare or cosmetic):** FO-03-E and FO-06-G (ruleset choice), MF-02
(random), Super Stealth research stealing, the −300 overcrowding floor
(not run), the duplicate-serial penalty, the 32766 and 32765 caps.

**Stale documents:** KERNEL-STATUS.md "Not modelled yet" lists IFE and
the Radiating Hydro-Ram Scoop, which its own table marks implemented.
ORDERS-STATUS.md "Not modelled" still lists loads.

## Lanes

Owners are filled in when the coordinator confirms them. A lane changes
only the files it owns. `engine/turn.go` and the shared state types
(`Game`, `Planet`, `Fleet`, `Player`) change only through lane A, and a
change to saved state bumps the save version.

| Lane | Scope | Files | Owner |
|---|---|---|---|
| A Integration | turn order, waypoints, orders replay, #130, diff triage | `engine/turn.go`, `waypoints.go`, `orders.go`, `fleetops.go`, `invariants.go`, `ruleset.go`, `catalog.go`, `game/`, `regress/` | TBD |
| B Economy | population, mining, production, research, terraforming | `engine/population.go`, `economy.go`, `mining.go`, `production*.go`, `research.go`, `habitability.go`, `randomevents.go`, `terraform*.go`, `terraform/`, `races/` | TBD |
| C Fleets | movement, cargo, launch, takeover, space objects | `engine/movement.go`, `fleetships.go`, `launch.go`, `orders_cargo.go`, `orders_ships.go`, `orders_design.go`, `takeover.go`, `scrap.go`, `objects/` | TBD |
| D Combat | battles, the harness random source | `engine/combat*.go`, `battleplan.go`, `ships.go`, new harness random source | TBD |
| E Output | views, messages, scores, reports | `engine/scanning.go`, `scores.go`, new `engine/messages.go`, `game/report.go`, `game/history.go` | TBD |
| F Oracle diff | RG corpus, loader, command, vector sync | `engine/*parity*_test.go`, `engine/testdata/vectors/`, new `parity/`, new `cmd/elegy/vector.go` | TBD |
| G Research | questions other lanes raise; public answers only | none in elegy | TBD |

`ai/` and `newgame/` are frozen this sprint except for fixes to a
long-game failure.

## Blockers

None yet.

## Merged work

None yet.

## Known parity failures

- FO-03-E, FO-06-G: differ only because `Legacy.MergeOverflow` is off in
  `elegy`.
- MF-02: random, 1 of 8 seeds, fails on the reference seed.

## Next actions

1. Lane F: run RG-001 in the oracle and build its vector.
2. Lane F: move the loader out of test code and add `elegy vector`.
3. Lane A: finish #130.
4. Lane G, then D: publish the original generator as a public rule so
   the harness can replay oracle streams.
5. Lane E: map events to MESSAGES.md kinds and compare messages.
