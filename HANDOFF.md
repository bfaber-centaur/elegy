# Handoff

## Current question

Is main `f09acd9` a parity release candidate? That means knowing what is compared and passes, what is random-dependent, sampled, absent or untested, and what stands between Elegy and a complete engine.

## State

- The release candidate is main `f09acd9`'s code, pinned as #260's merge commit (docs only on top). It is the base for lane F's out-of-sample game, RG-005. The RG vectors are stars-elegy main `104e8f5`'s, including #206's rebuilt year 6.
- The coverage report is [docs/COVERAGE.md](docs/COVERAGE.md). The per-corpus detail is in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "Current status".
- Tested at #260's head `b78c03c` (tree `59a494d`), a merge of main `f09acd9` with this PR's docs-only commits. Outside docs/ and HANDOFF.md its files are identical to `f09acd9`'s (`git diff f09acd9 b78c03c -- . ':!docs' ':!HANDOFF.md'` is empty), so the code tested is main `f09acd9`'s (tree `b7e0e77`). No test reads the files this PR changes. The candidate is #260's merge commit, whose tree differs from `f09acd9`'s only in docs/ and HANDOFF.md. On that head:
  - `go test ./...` passes.
  - The PR-subset long-game matrix passes: 24 games of 60 years, with replay and save/reload.
  - The nightly matrix: 96 of 96 games with no failure (12 seeds × 2 rulesets × 4 opponent sets, 150 years on a medium map, replayed and reloaded after 77 years; 444 s).
  - The RG ledgers and `TestBattleRecords` match the run exactly, and regenerating them moves no line.
- Each ledgered game registers from its files (`engine/testdata/vectors/rg/GAMES`, internal/vectors `RGGames`). A PR that moves RG output regenerates the ledgers rather than editing them by hand:
  - `RG002_LEDGER=write go test ./internal/vectors -run 'TestRG00.Ledger'`
  - `RG002_LEDGER=write go test ./engine -run 'TestBattleRecords$'`
  - `PARITY_BASELINE=write` for the baseline.

## What we know

- **0 known discrepancies in RG-002, RG-003 and RG-004** on the reference seed: no incorrect-mechanic line and no unresolved-randomness line.
  - Every compared difference is a sample, or random-driven: it matches on at least one of the 8 seed variants.
  - The chained runs diverge in year 1, and everything after year 1 is a sample.
- **Ledger lines** (chained / restarts):

  | corpus | matched | random-driven | sample | not compared | total |
  |---|---|---|---|---|---:|
  | RG-002 | 8 / 44 | 0 / 35 | 1019 / 199 | 0 / 0 | 1305 |
  | RG-003 | 8 / 100 | 0 / 76 | 5287 / 450 | 0 / 0 | 5921 |
  | RG-004 | 4 / 47 | 1 / 17 | 2519 / 263 | 2 / 2 | 2855 |

- **Orders:** every recorded order is loaded and accounted for, and no vector has a load error.
  - The chained runs refuse 37, 220 and 84 orders, all samples.
  - No restart refuses an order.
- **Random-driven lines with a named draw.** RG-002 and RG-003 each have the same 3, in the Y6 restart:
  - one is lane D's movement tie;
  - two are economy's germanium mining +1 draw.
- **Not compared:** RG-004's `design_gone` (lane B) and a sequence of sample entries only. `alt_battle` is judged by `TestBattleRecords`.
- **Zero known discrepancies is not complete equivalence.** COVERAGE.md "Biggest blind spots" covers:
  - random parity, which is 0 exact;
  - computer players, with no oracle comparison;
  - viewer reports and client estimates, 372 expectations;
  - 63 unmapped message ids;
  - fleet-to-fleet cargo.

## Engine completion

The full checklist, by subsystem and column, is docs/SPRINT.md "Engine-completion checklist".

| column | state | what is missing |
|---|---|---|
| playable | done for the vertical slice | Every step works from the CLI with readable output (`elegy show`, `order`, `check`, #266), and a saved game reloads identically. Deferred: design and battle-plan authoring by name, derived numbers and a front end. |
| behaviorally faithful | partial | 0 known discrepancies inside the compared set. Not compared: AI decisions, viewer reports, client estimates, 63 message ids and fleet-to-fleet cargo. |
| bit-exact RNG parity | missing | The original's generator is not in the harness (SPRINT.md P1-1, pending Bobby). |

## What still matters

- COVERAGE.md "Remaining comparator gaps, ranked" lists what to route next. The top three:
  - viewer reports (156 expectations, lane B);
  - minefield messages (35, lane D);
  - fleet-to-fleet transport (38, lane A, a mechanic).
- Random-driven lines stay open until the original's generator is in the harness.
- The parity case tally lists most RG cases as `fail`, although there are 0 known discrepancies.
  - Restarts fail because no single seed matches all their random-driven expectations at once.
  - Chained years after the first continue from a diverged year 1.
  - PARITY-STATUS.md "Current status" has the per-seed check behind this.

## Best next move

Run RG-005 (lane F) on `f09acd9` as the out-of-sample check of the zero. Then route the ranked gaps, starting with the viewer reports.
