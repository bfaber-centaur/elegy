# Handoff

## Current question

Do the three representative-game corpora (RG-002, RG-003, RG-004) give trustworthy, explicitly classified results, and what stands between Elegy and the original in them?

## State

- elegy main `5a735e1` (after #242). The RG vectors are stars-elegy main `104e8f5`'s, including #206's rebuilt year 6.
- The combined run on that SHA is recorded in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "Current status".
  - `go test ./...` passes.
  - The PR-subset long-game matrix passes: 24 games of 60 years, with replay and save/reload. The steward's full matrix is 64 cases at 150 years.
  - The three RG ledgers and `TestBattleRecords` match the run exactly.
- Each corpus's ledger (`engine/testdata/vectors/rg/rg00N-ledger.txt`) is checked exactly. A PR that moves RG output regenerates it rather than editing it by hand:
  - `RG002_LEDGER=write go test ./internal/vectors -run 'TestRG00[234]Ledger'`
  - `RG002_LEDGER=write go test ./engine -run 'TestBattleRecords$'`
  - and `PARITY_BASELINE=write` for the baseline.

## What we know

- **0 deterministic misses in all three corpora** on the reference seed: no incorrect-mechanic line and no unresolved-randomness line.
  - Every compared difference is a sample, or random-driven: it matches on at least one of the 8 seed variants.
  - The chained runs diverge in year 1 and are samples after it.
  - Ledger lines (chained / restarts):
    - RG-002: matched 8 / 44, random-driven 0 / 35, sample 1019 / 199, not compared 0 / 0 (main after #247 and #248).
    - RG-003: matched 8 / 100, random-driven 0 / 76, sample 5287 / 450, not compared 0 / 0.
    - RG-004: matched 4 / 47, random-driven 1 / 17, sample 2519 / 263, not compared 2 / 2.
    - Updated 2026-10-13 (lane A, invasion messages): these were RG-003 not compared 7 / 7, RG-004 sample 2523 / 263 and not compared 16 / 16.
    - Updated 2026-10-13 (0x0cd PR): RG-003 not compared was 4 / 4.
    - Updated 2026-10-13 (alt_battle PR): RG-003 was sample 5286 / 450 and not compared 2 / 2, RG-004 not compared 3 / 3; the `alt_battle` expectations are battle-ledger samples now.
    - Updated 2026-10-13 (lane A, chase, Z-MD): RG-003's random-driven was 0 / 75 (the Y11 battle at (1323,1300) is now random-driven); samples were RG-002 1024, RG-003 5291 / 451, RG-004 2531.
    - Updated 2026-10-14 (lane A, 0x02e slots, stars-elegy CM-1): not compared was RG-002 1 / 1, RG-003 1 / 1.
- **Orders:** every recorded order is loaded and accounted for, and no vector has a load error.
  - RG-002 has 1867 orders, of which 37 are refused in the chained run.
  - RG-003 has 3731, with 220 refused in the chained run.
  - RG-004 has 2049, with 84 refused in the chained run.
  - All chained refusals are samples. No restart refuses an order.
- **Random-driven lines with a named draw.** Each corpus has 3 in the Y6 restart, the same 3 in RG-002 and RG-003:
  - fleet 6/12's waypoint (lane D, movement tie draws in the battle at (1296,1356));
  - planet 102's minerals and production queue (economy, the germanium mining +1 draw).
  - RG-004's random-driven lines have no named draw.
- **The DA-1 rebuild of year 6 (stars-elegy #206) added no deterministic miss.** Its per-expectation before/after table is [docs/DA1-Y6-RECLASS.tsv](docs/DA1-Y6-RECLASS.tsv).
- **Not compared, by owner:**
  - lane B: `design_gone`.

## What still matters

- The not-compared lines are unjudged, not matched. Updated 2026-10-13: RG-004's 26 0x007 lines, the largest group, are compared since the invasion messages; what is left is 2 | 2 | 4 lines (updated 2026-10-14: 0 | 0 | 4 after the 0x02e slots) (PARITY-STATUS.md "Not compared, by owner"; updated 2026-10-13 after the 0x0cd and alt_battle PRs, it was 2 | 8 | 6).
- `design_gone` (RG-004, 2 lines) is harness work for lane B.
- Random-driven lines stay open until the original's generator is in the harness (SPRINT.md P1-1, the seed-table publication pending Bobby).
- The parity case tally lists most RG cases as `fail`, although there are 0 deterministic misses. Restarts fail because no single seed matches all their random-driven expectations at once. Chained years after the first fail because they continue from a diverged year 1. PARITY-STATUS.md "Current status" has the per-seed check behind this.

## Best next move

Updated 2026-10-13: the invasion messages are mapped (0x004's slots, 0x007, 0x00c); the restarts' messages match and the chained ones are samples. The not-compared lines left are small groups with named owners (PARITY-STATUS.md "Not compared, by owner").
