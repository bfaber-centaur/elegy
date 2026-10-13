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
    - RG-002: matched 8 / 44, random-driven 0 / 35, sample 1024 / 199, not compared 1 / 1 (main after #247 and #248).
    - RG-003: matched 8 / 100, random-driven 0 / 75, sample 5291 / 451, not compared 5 / 5.
    - RG-004: matched 4 / 47, random-driven 1 / 17, sample 2523 / 263, not compared 16 / 16.
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
  - lane D: the 0x05b fleet slot, whose owner bits are UNRESOLVED (the salvage slot and the 0x08f/0x090 bombing fleet are compared since #247);
  - lane A: the 0x004 slots and 0x007, with the invasion messages;
  - lane B: `design_gone`.
  - `alt_battle` is judged by the battle harness instead of the vectors comparator.

## What still matters

- The not-compared lines are unjudged, not matched. RG-004's 26 0x007 lines are the largest group.
  - stars-elegy MESSAGES.md "Invasion slots" now documents the 0x004 and 0x007 slots.
- `design_gone` (RG-004, 2 lines) is harness work for lane B.
- Random-driven lines stay open until the original's generator is in the harness (SPRINT.md P1-1, the seed-table publication pending Bobby).
- The parity case tally lists most RG cases as `fail`, although there are 0 deterministic misses. Restarts fail because no single seed matches all their random-driven expectations at once. Chained years after the first fail because they continue from a diverged year 1. PARITY-STATUS.md "Current status" has the per-seed check behind this.

## Best next move

Map 0x007 (and 0x004's slots) from Elegy's capture events, following MESSAGES.md "Invasion slots". That turns RG-004's largest not-compared group into compared lines. Then regenerate the ledgers and baseline and classify whatever it exposes.
