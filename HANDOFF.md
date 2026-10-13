# Handoff

## Current question

Do the three representative-game corpora (RG-002, RG-003, RG-004) give trustworthy, explicitly classified results, and what stands between Elegy and the original in them?

## State

- elegy main `215cd38` (after #251–#257). The RG vectors are stars-elegy main `104e8f5`'s, including #206's rebuilt year 6.
- The combined run on that SHA is recorded in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "Current status":
  - `go test ./...` passes.
  - The PR-subset long-game matrix passes: 24 games of 60 years, with replay and save/reload. The steward's full matrix is 64 cases at 150 years.
  - The three RG ledgers and `TestBattleRecords` match the run exactly. Regenerating them moves no line.
- Each corpus's ledger (`engine/testdata/vectors/rg/rg00N-ledger.txt`) is checked exactly. A PR that moves RG output regenerates it rather than editing it by hand:
  - `RG002_LEDGER=write go test ./internal/vectors -run 'TestRG00[234]Ledger'`
  - `RG002_LEDGER=write go test ./engine -run 'TestBattleRecords$'`
  - `PARITY_BASELINE=write` for the baseline.

## What we know

- **0 deterministic misses in all three corpora** on the reference seed: no incorrect-mechanic line and no unresolved-randomness line.
  - Every compared difference is a sample, or random-driven: it matches on at least one of the 8 seed variants.
  - The chained runs diverge in year 1 and are samples after it.
- **Ledger lines** (chained / restarts):

  | corpus | matched | random-driven | sample | not compared | total |
  |---|---|---|---|---|---:|
  | RG-002 | 8 / 44 | 0 / 35 | 1019 / 199 | 1 / 1 | 1307 |
  | RG-003 | 8 / 100 | 0 / 76 | 5286 / 450 | 2 / 2 | 5924 |
  | RG-004 | 4 / 47 | 1 / 17 | 2519 / 263 | 3 / 3 | 2857 |

- **Field expectations** (strict on year 1 and in the restarts): none is a strict miss.
  - The random-driven ones are RG-002 25, RG-003 59 and RG-004 13.
  - Everything that has left "not compared" since `5a735e1` matches or is a sample.
- **Orders:** every recorded order is loaded and accounted for, and no vector has a load error.
  - RG-002 has 1867 orders, and the chained run refuses 37.
  - RG-003 has 3731, and the chained run refuses 220.
  - RG-004 has 2049, and the chained run refuses 84.
  - All of these refusals are samples. No restart refuses an order.
- **Random-driven lines with a named draw.** RG-002 and RG-003 each have the same 3, in the Y6 restart:
  - fleet 6/12's waypoint (lane D: movement tie draws in the battle at (1296,1356));
  - planet 102's minerals and production queue (economy: the germanium mining +1 draw).
  - RG-004 has none.
- **The DA-1 rebuild of year 6 (stars-elegy #206) added no deterministic miss.** The per-expectation before/after table is [docs/DA1-Y6-RECLASS.tsv](docs/DA1-Y6-RECLASS.tsv).
- **Not compared** (RG-002 2, RG-003 4, RG-004 6 lines), by owner:
  - lane A: 0x02e's mineral and location slots;
  - lane B: `design_gone`;
  - judged by the battle harness instead of the vectors comparator: `alt_battle`;
  - nothing to compare: 2 sequences of sample entries only.

## What still matters

- Not-compared lines are unjudged, not matched. The ones left are 0x02e's slots (lane A) and `design_gone` (lane B, RG-004).
- Random-driven lines stay open until the original's generator is in the harness (SPRINT.md P1-1; the seed-table publication is pending Bobby).
- The parity case tally lists most RG cases as `fail`, although there are 0 deterministic misses.
  - Restarts fail because no single seed matches all their random-driven expectations at once.
  - Chained years after the first fail because they continue from a diverged year 1.
  - PARITY-STATUS.md "Current status" has the per-seed check behind this.

## Best next move

Compare `design_gone` in the harness (lane B), so that RG-004's two remaining lines are judged. Then regenerate the ledgers and baseline.
